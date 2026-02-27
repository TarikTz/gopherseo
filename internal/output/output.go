// Package output handles writing crawl results to disk in various formats
// (sitemap XML, Markdown broken-link reports).
package output

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/tariktz/gopherseo/internal/crawler"
)

// sitemapURLSet is the root element of a Sitemap 0.9 XML document.
type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// sitemapURL represents a single <url> entry.
type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// WriteSitemap creates a Sitemap 0.9 XML file at outputPath containing the
// given URLs. If lastModifiedMap is non-nil, each URL's <lastmod> element is
// populated with the corresponding W3C date (YYYY-MM-DD). Parent directories
// are created automatically.
func WriteSitemap(outputPath string, urls []string, lastModifiedMap map[string]time.Time) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}

	urlset := sitemapURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]sitemapURL, 0, len(urls)),
	}
	for _, link := range urls {
		u := sitemapURL{Loc: link}
		if lastModifiedMap != nil {
			if t, ok := lastModifiedMap[link]; ok {
				u.LastMod = t.UTC().Format("2006-01-02")
			}
		}
		urlset.URLs = append(urlset.URLs, u)
	}

	if _, err := f.Write([]byte(xml.Header)); err != nil {
		_ = f.Close()
		return fmt.Errorf("write xml header: %w", err)
	}

	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(urlset); err != nil {
		_ = f.Close()
		return fmt.Errorf("write sitemap xml: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close sitemap file: %w", err)
	}

	return nil
}

// WriteIssueTasks creates a Markdown checklist at outputPath documenting every
// broken link and the source pages that reference it.
func WriteIssueTasks(outputPath string, tasks []crawler.BrokenLinkTask) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create issues output directory: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create issues output file: %w", err)
	}

	w := bufio.NewWriter(f)

	flushAndClose := func() error {
		if fErr := w.Flush(); fErr != nil {
			_ = f.Close()
			return fmt.Errorf("flush issues file: %w", fErr)
		}
		if cErr := f.Close(); cErr != nil {
			return fmt.Errorf("close issues file: %w", cErr)
		}
		return nil
	}

	writeErr := func(msg string, err error) error {
		_ = f.Close()
		return fmt.Errorf("%s: %w", msg, err)
	}

	if _, err := w.WriteString("# Link Cleanup Tasks\n\n"); err != nil {
		return writeErr("write issues header", err)
	}

	if len(tasks) == 0 {
		if _, err := w.WriteString("No broken links were found in this crawl.\n"); err != nil {
			return writeErr("write no-issues message", err)
		}
		return flushAndClose()
	}

	for i, task := range tasks {
		statusLabel := strconv.Itoa(task.Status)
		if task.Status == 0 {
			statusLabel = "request_failed"
		}

		if _, err := fmt.Fprintf(w, "- [ ] Fix `%s` (status: %s)\n", task.URL, statusLabel); err != nil {
			return writeErr("write task item", err)
		}

		if len(task.Sources) == 0 {
			if _, err := w.WriteString("  - Found on: (source page not captured)\n"); err != nil {
				return writeErr("write task source fallback", err)
			}
		} else {
			for _, source := range task.Sources {
				if _, err := fmt.Fprintf(w, "  - Found on: `%s`\n", source); err != nil {
					return writeErr("write task source", err)
				}
			}
		}

		if i < len(tasks)-1 {
			if _, err := w.WriteString("\n"); err != nil {
				return writeErr("write task separator", err)
			}
		}
	}

	return flushAndClose()
}
