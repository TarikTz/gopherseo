// Package crawler implements a concurrent web crawler that discovers internal
// links, validates HTTP status codes, and reports broken URLs.
package crawler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	pathpkg "path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/tariktz/gopherseo/internal/lastmod"
)

const defaultUserAgent = "GopherSEO-Bot/1.0"

// Options configures the behaviour of a crawl run.
type Options struct {
	// RootURL is the seed URL from which crawling starts.
	RootURL string
	// MaxDepth limits how many link-hops away from the root the crawler
	// will follow. A value of 0 means unlimited depth.
	MaxDepth int
	// Threads sets the maximum number of concurrent HTTP requests.
	Threads int
	// UserAgent is sent as the User-Agent header on every request.
	UserAgent string
	// ExcludePatterns is a list of glob patterns; URLs matching any
	// pattern are skipped during the crawl.
	ExcludePatterns []string
	// RequestTimeout is the maximum duration for a single HTTP request.
	// A zero value means no timeout.
	RequestTimeout time.Duration
}

// Result holds the output of a completed crawl.
type Result struct {
	// ValidURLs contains every discovered URL that returned a 2xx/3xx status.
	ValidURLs []string
	// BrokenLinks maps each broken URL to its HTTP status code (0 = request failed).
	BrokenLinks map[string]int
	// BrokenLinkTasks provides a structured list of broken links together with
	// the pages on which each broken link was found.
	BrokenLinkTasks []BrokenLinkTask
	// LastModified maps each valid URL to its best-available last-modified
	// timestamp, extracted using the lastmod extraction hierarchy.
	LastModified map[string]time.Time
	// Discovered is the total number of unique URLs seen during the crawl.
	Discovered int
	// ExcludedURLs is the number of URLs that were skipped due to exclusion rules.
	ExcludedURLs int
}

// BrokenLinkTask represents a single broken link and every source page that
// references it. This is used to generate actionable fix-task reports.
type BrokenLinkTask struct {
	URL     string
	Status  int
	Sources []string
}

// Crawl performs a recursive crawl starting from opts.RootURL. It returns a
// Result containing all discovered valid URLs, broken links, and associated
// metadata. The function blocks until the crawl is complete.
func Crawl(opts Options) (Result, error) {
	normalizedRoot, parsedRoot, err := normalizeRoot(opts.RootURL)
	if err != nil {
		return Result{}, err
	}

	if opts.Threads <= 0 {
		opts.Threads = 5
	}
	if opts.UserAgent == "" {
		opts.UserAgent = defaultUserAgent
	}

	collectorOptions := []colly.CollectorOption{
		colly.Async(true),
		colly.UserAgent(opts.UserAgent),
		colly.AllowedDomains(parsedRoot.Hostname()),
	}
	if opts.MaxDepth > 0 {
		collectorOptions = append(collectorOptions, colly.MaxDepth(opts.MaxDepth))
	}

	c := colly.NewCollector(collectorOptions...)
	c.IgnoreRobotsTxt = false

	if err := c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: opts.Threads}); err != nil {
		return Result{}, fmt.Errorf("configure crawler concurrency: %w", err)
	}

	if opts.RequestTimeout > 0 {
		c.SetRequestTimeout(opts.RequestTimeout)
	}

	var mu sync.Mutex
	valid := make(map[string]struct{})
	broken := make(map[string]int)
	discovered := make(map[string]struct{})
	sources := make(map[string]map[string]struct{})
	lastModified := make(map[string]time.Time)
	excluded := 0
	now := time.Now()

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		raw := strings.TrimSpace(e.Attr("href"))
		if raw == "" {
			return
		}

		absolute := e.Request.AbsoluteURL(raw)
		if absolute == "" {
			return
		}

		normalizedLink, parsedLink, err := normalizeURL(absolute)
		if err != nil {
			return
		}

		if !isHTTP(parsedLink) || !isInternal(parsedRoot, parsedLink) {
			return
		}

		if shouldExclude(normalizedLink, opts.ExcludePatterns) {
			mu.Lock()
			excluded++
			mu.Unlock()
			return
		}

		mu.Lock()
		discovered[normalizedLink] = struct{}{}
		sourceURL, _, sourceErr := normalizeURL(e.Request.URL.String())
		if sourceErr == nil {
			if _, ok := sources[normalizedLink]; !ok {
				sources[normalizedLink] = make(map[string]struct{})
			}
			sources[normalizedLink][sourceURL] = struct{}{}
		}
		mu.Unlock()

		_ = e.Request.Visit(normalizedLink)
	})

	c.OnResponse(func(r *colly.Response) {
		normalizedLink, _, err := normalizeURL(r.Request.URL.String())
		if err != nil {
			return
		}

		mu.Lock()
		defer mu.Unlock()

		discovered[normalizedLink] = struct{}{}
		if r.StatusCode >= 200 && r.StatusCode < 400 {
			valid[normalizedLink] = struct{}{}
			delete(broken, normalizedLink)

			// Extract last-modified timestamp using the priority hierarchy.
			var header http.Header
			if r.Headers != nil {
				header = *r.Headers
			}
			doc, _ := goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
			lastModified[normalizedLink] = lastmod.GetLastModified(header, doc, now)
			return
		}

		broken[normalizedLink] = r.StatusCode
		delete(valid, normalizedLink)
	})

	c.OnError(func(r *colly.Response, err error) {
		if r == nil || r.Request == nil || r.Request.URL == nil {
			return
		}

		normalizedLink, _, parseErr := normalizeURL(r.Request.URL.String())
		if parseErr != nil {
			return
		}

		status := r.StatusCode

		mu.Lock()
		broken[normalizedLink] = status
		delete(valid, normalizedLink)
		mu.Unlock()
	})

	if err := c.Visit(normalizedRoot); err != nil {
		return Result{}, fmt.Errorf("start crawling: %w", err)
	}
	c.Wait()

	validURLs := make([]string, 0, len(valid))
	for u := range valid {
		if shouldExclude(u, opts.ExcludePatterns) {
			continue
		}
		validURLs = append(validURLs, u)
	}
	sort.Strings(validURLs)

	brokenURLs := make(map[string]int, len(broken))
	brokenTasks := make([]BrokenLinkTask, 0, len(broken))
	for u, status := range broken {
		if shouldExclude(u, opts.ExcludePatterns) {
			continue
		}
		brokenURLs[u] = status

		sourceList := make([]string, 0)
		if sourceSet, ok := sources[u]; ok {
			sourceList = make([]string, 0, len(sourceSet))
			for source := range sourceSet {
				sourceList = append(sourceList, source)
			}
			sort.Strings(sourceList)
		}

		brokenTasks = append(brokenTasks, BrokenLinkTask{
			URL:     u,
			Status:  status,
			Sources: sourceList,
		})
	}
	sort.Slice(brokenTasks, func(i, j int) bool {
		return brokenTasks[i].URL < brokenTasks[j].URL
	})

	return Result{
		ValidURLs:       validURLs,
		BrokenLinks:     brokenURLs,
		BrokenLinkTasks: brokenTasks,
		LastModified:    lastModified,
		Discovered:      len(discovered),
		ExcludedURLs:    excluded,
	}, nil
}

func normalizeRoot(raw string) (string, *url.URL, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return "", nil, fmt.Errorf("root url is required")
	}
	if !strings.HasPrefix(clean, "http://") && !strings.HasPrefix(clean, "https://") {
		clean = "https://" + clean
	}

	normalized, parsed, err := normalizeURL(clean)
	if err != nil {
		return "", nil, fmt.Errorf("invalid root url: %w", err)
	}
	if parsed.Hostname() == "" {
		return "", nil, fmt.Errorf("invalid root url: missing host")
	}

	return normalized, parsed, nil
}

func normalizeURL(raw string) (string, *url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", nil, err
	}
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	// Remove trailing slash for non-root paths to prevent duplicate entries.
	// e.g. /about/ and /about become the same normalised URL.
	if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}

	return parsed.String(), parsed, nil
}

func isHTTP(u *url.URL) bool {
	if u == nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func isInternal(root *url.URL, candidate *url.URL) bool {
	if root == nil || candidate == nil {
		return false
	}

	return strings.EqualFold(root.Hostname(), candidate.Hostname())
}

func shouldExclude(link string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Use path.Match (not filepath.Match) so glob behaviour is consistent
		// across operating systems — URL paths always use forward slashes.
		if matched, _ := pathpkg.Match(pattern, link); matched {
			return true
		}

		parsed, err := url.Parse(link)
		if err != nil {
			continue
		}

		// Match against the full path (e.g. /admin/*).
		if matched, _ := pathpkg.Match(pattern, parsed.Path); matched {
			return true
		}

		// Match against just the filename so *.pdf matches /dir/file.pdf.
		if matched, _ := pathpkg.Match(pattern, pathpkg.Base(parsed.Path)); matched {
			return true
		}

		// Match path+query with and without the leading slash so that
		// patterns like *?lang=rs work against page?lang=rs.
		if parsed.RawQuery != "" {
			queryPath := parsed.Path + "?" + parsed.RawQuery
			if matched, _ := pathpkg.Match(pattern, queryPath); matched {
				return true
			}
			trimmed := strings.TrimPrefix(queryPath, "/")
			if matched, _ := pathpkg.Match(pattern, trimmed); matched {
				return true
			}
		}
	}

	return false
}
