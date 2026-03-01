package canonical

import (
	"net/url"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Info contains canonical tag extraction details for a crawled page.
type Info struct {
	PageURL      string
	CanonicalURL string
	TagCount     int
	Missing      bool
	Multiple     bool
}

// IssueType describes a canonical validation problem category.
type IssueType string

const (
	IssueNonHTTPScheme  IssueType = "non_http_scheme"
	IssueCrossDomain    IssueType = "cross_domain"
	IssueTargetBroken   IssueType = "target_broken"
	IssueTargetRedirect IssueType = "target_redirect"
	IssueLoopOrChain    IssueType = "loop_or_chain"
)

// Issue represents a canonical validation finding for a page.
type Issue struct {
	PageURL      string
	CanonicalURL string
	Type         IssueType
	Detail       string
}

// Extract inspects a page document and extracts canonical link information.
// It resolves relative canonical href values against pageURL and applies URL
// normalization (strip fragments and trailing slash for non-root paths).
func Extract(pageURL string, doc *goquery.Document) Info {
	info := Info{PageURL: pageURL}
	if doc == nil {
		info.Missing = true
		return info
	}

	canonicalLinks := doc.Find(`link[rel="canonical"]`)
	info.TagCount = canonicalLinks.Length()
	if info.TagCount == 0 {
		info.Missing = true
		return info
	}
	if info.TagCount > 1 {
		info.Multiple = true
	}

	found := ""
	canonicalLinks.EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href := strings.TrimSpace(s.AttrOr("href", ""))
		if href == "" {
			return true
		}
		found = href
		return false
	})

	if found == "" {
		info.Missing = true
		return info
	}

	resolved, ok := resolveAgainstPage(pageURL, found)
	if !ok {
		info.CanonicalURL = found
		return info
	}

	info.CanonicalURL = resolved
	return info
}

func resolveAgainstPage(pageURL, href string) (string, bool) {
	base, err := url.Parse(pageURL)
	if err != nil {
		return "", false
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	resolved := base.ResolveReference(parsed)
	return normalizeURL(resolved.String())
}

func normalizeURL(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}
	return parsed.String(), true
}

// Validate applies canonical validation rules across extracted canonical data.
// statusByURL should contain HTTP status codes gathered during crawl.
func Validate(canonicalByPage map[string]string, statusByURL map[string]int) []Issue {
	issues := make([]Issue, 0)
	seen := make(map[string]struct{})

	for page, target := range canonicalByPage {
		if issue, ok := validatePair(page, target, statusByURL); ok {
			key := string(issue.Type) + "|" + issue.PageURL + "|" + issue.CanonicalURL
			if _, exists := seen[key]; !exists {
				issues = append(issues, issue)
				seen[key] = struct{}{}
			}
		}

		if issue, ok := detectLoopOrChain(page, canonicalByPage); ok {
			key := string(issue.Type) + "|" + issue.PageURL + "|" + issue.CanonicalURL + "|" + issue.Detail
			if _, exists := seen[key]; !exists {
				issues = append(issues, issue)
				seen[key] = struct{}{}
			}
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].PageURL != issues[j].PageURL {
			return issues[i].PageURL < issues[j].PageURL
		}
		if issues[i].Type != issues[j].Type {
			return issues[i].Type < issues[j].Type
		}
		if issues[i].CanonicalURL != issues[j].CanonicalURL {
			return issues[i].CanonicalURL < issues[j].CanonicalURL
		}
		return issues[i].Detail < issues[j].Detail
	})

	return issues
}

func validatePair(page, target string, statusByURL map[string]int) (Issue, bool) {
	parsedTarget, err := url.Parse(target)
	if err != nil {
		return Issue{}, false
	}

	if parsedTarget.Scheme != "http" && parsedTarget.Scheme != "https" {
		return Issue{PageURL: page, CanonicalURL: target, Type: IssueNonHTTPScheme, Detail: "canonical target is not HTTP(S)"}, true
	}

	parsedPage, err := url.Parse(page)
	if err == nil {
		if !strings.EqualFold(parsedPage.Hostname(), parsedTarget.Hostname()) {
			return Issue{PageURL: page, CanonicalURL: target, Type: IssueCrossDomain, Detail: "canonical target is on a different host"}, true
		}
	}

	if status, ok := statusByURL[target]; ok {
		if status >= 300 && status < 400 {
			return Issue{PageURL: page, CanonicalURL: target, Type: IssueTargetRedirect, Detail: "canonical target responds with redirect"}, true
		}
		if status == 0 || status >= 400 {
			return Issue{PageURL: page, CanonicalURL: target, Type: IssueTargetBroken, Detail: "canonical target is broken/unreachable"}, true
		}
	}

	return Issue{}, false
}

func detectLoopOrChain(start string, canonicalByPage map[string]string) (Issue, bool) {
	target, ok := canonicalByPage[start]
	if !ok || target == "" || target == start {
		return Issue{}, false
	}

	visited := map[string]int{start: 0}
	path := []string{start, target}
	steps := 1
	current := target

	for {
		next, exists := canonicalByPage[current]
		if !exists || next == "" {
			if steps >= 2 {
				return Issue{PageURL: start, CanonicalURL: target, Type: IssueLoopOrChain, Detail: "canonical chain detected"}, true
			}
			return Issue{}, false
		}

		if idx, seen := visited[current]; seen {
			_ = idx
			return Issue{PageURL: start, CanonicalURL: target, Type: IssueLoopOrChain, Detail: "canonical loop detected"}, true
		}
		visited[current] = len(path) - 1

		if next == current {
			if steps >= 2 {
				return Issue{PageURL: start, CanonicalURL: target, Type: IssueLoopOrChain, Detail: "canonical chain detected"}, true
			}
			return Issue{}, false
		}

		path = append(path, next)
		current = next
		steps++

		if steps > len(canonicalByPage)+1 {
			return Issue{PageURL: start, CanonicalURL: target, Type: IssueLoopOrChain, Detail: "canonical loop detected"}, true
		}
	}
}
