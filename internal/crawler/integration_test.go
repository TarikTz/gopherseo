package crawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

// newTestServer creates an httptest.Server with a small site structure:
//
//	/           -> 200, links to /about and /contact
//	/about      -> 200, links to / and /broken
//	/contact    -> 200, no outgoing links
//	/broken     -> 404
//	/excluded   -> 200, but should be filtered by exclusion patterns
func newTestServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="/about">About</a>
			<a href="/contact">Contact</a>
			<a href="/excluded">Excluded</a>
		</body></html>`)
	})

	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="/">Home</a>
			<a href="/broken">Broken</a>
		</body></html>`)
	})

	mux.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><p>Contact us</p></body></html>`)
	})

	mux.HandleFunc("/broken", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	mux.HandleFunc("/excluded", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><p>Excluded page</p></body></html>`)
	})

	return httptest.NewServer(mux)
}

func TestCrawl_Integration(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	result, err := Crawl(Options{
		RootURL:         ts.URL,
		Threads:         2,
		MaxDepth:        0,
		ExcludePatterns: []string{"/excluded"},
		RequestTimeout:  10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Crawl() error: %v", err)
	}

	// Check valid URLs: should include /, /about, /contact but NOT /broken or /excluded
	sort.Strings(result.ValidURLs)

	wantValid := []string{
		ts.URL + "/",
		ts.URL + "/about",
		ts.URL + "/contact",
	}
	sort.Strings(wantValid)

	if len(result.ValidURLs) != len(wantValid) {
		t.Errorf("ValidURLs count = %d, want %d\n  got:  %v\n  want: %v",
			len(result.ValidURLs), len(wantValid), result.ValidURLs, wantValid)
	} else {
		for i, u := range result.ValidURLs {
			if u != wantValid[i] {
				t.Errorf("ValidURLs[%d] = %q, want %q", i, u, wantValid[i])
			}
		}
	}

	// Check broken links: should contain /broken
	brokenURL := ts.URL + "/broken"
	if status, ok := result.BrokenLinks[brokenURL]; !ok {
		t.Errorf("BrokenLinks missing %q", brokenURL)
	} else if status != 404 {
		t.Errorf("BrokenLinks[%q] = %d, want 404", brokenURL, status)
	}

	// Check broken link tasks have source tracking
	if len(result.BrokenLinkTasks) < 1 {
		t.Fatal("expected at least 1 BrokenLinkTask")
	}

	var brokenTask *BrokenLinkTask
	for i := range result.BrokenLinkTasks {
		if result.BrokenLinkTasks[i].URL == brokenURL {
			brokenTask = &result.BrokenLinkTasks[i]
			break
		}
	}
	if brokenTask == nil {
		t.Fatalf("BrokenLinkTasks missing entry for %q", brokenURL)
	}
	if len(brokenTask.Sources) == 0 {
		t.Error("broken link task should have at least one source page")
	}

	// The source should be /about (which links to /broken)
	foundAboutSource := false
	aboutURL := ts.URL + "/about"
	for _, src := range brokenTask.Sources {
		if src == aboutURL {
			foundAboutSource = true
			break
		}
	}
	if !foundAboutSource {
		t.Errorf("broken link sources %v should contain %q", brokenTask.Sources, aboutURL)
	}

	// Excluded URL should not appear in valid or broken
	excludedURL := ts.URL + "/excluded"
	for _, u := range result.ValidURLs {
		if u == excludedURL {
			t.Error("excluded URL should not appear in ValidURLs")
		}
	}
	if _, ok := result.BrokenLinks[excludedURL]; ok {
		t.Error("excluded URL should not appear in BrokenLinks")
	}

	// Discovered count should be reasonable
	if result.Discovered < 3 {
		t.Errorf("Discovered = %d, expected at least 3", result.Discovered)
	}
}

func TestCrawl_EmptyURL(t *testing.T) {
	_, err := Crawl(Options{RootURL: ""})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestCrawl_InvalidURL(t *testing.T) {
	_, err := Crawl(Options{RootURL: "not-a-real-server.invalid.test"})
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}

func TestCrawl_MaxDepth(t *testing.T) {
	// Create a deep chain: / -> /level1 -> /level2 -> /level3
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><a href="/level1">L1</a></body></html>`)
	})
	mux.HandleFunc("/level1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><a href="/level2">L2</a></body></html>`)
	})
	mux.HandleFunc("/level2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><a href="/level3">L3</a></body></html>`)
	})
	mux.HandleFunc("/level3", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><p>Deep page</p></body></html>`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	result, err := Crawl(Options{
		RootURL:        ts.URL,
		MaxDepth:       2,
		Threads:        1,
		RequestTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Crawl() error: %v", err)
	}

	// With depth=2, we should reach / (depth 0), /level1 (depth 1), /level2 (depth 2)
	// but NOT /level3 (depth 3)
	level3 := ts.URL + "/level3"
	for _, u := range result.ValidURLs {
		if u == level3 {
			t.Error("/level3 should not be reached with MaxDepth=2")
		}
	}

	// Root and level1 should be valid
	found := map[string]bool{}
	for _, u := range result.ValidURLs {
		found[u] = true
	}
	if !found[ts.URL+"/"] {
		t.Error("root URL should be in ValidURLs")
	}
	if !found[ts.URL+"/level1"] {
		t.Error("/level1 should be in ValidURLs")
	}
}

func TestCrawl_ExternalLinksIgnored(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="https://external-site.example.com/page">External</a>
			<a href="mailto:test@example.com">Email</a>
			<a href="javascript:void(0)">JS</a>
		</body></html>`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	result, err := Crawl(Options{
		RootURL:        ts.URL,
		Threads:        1,
		RequestTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Crawl() error: %v", err)
	}

	// Should only have the root page
	if len(result.ValidURLs) != 1 {
		t.Errorf("ValidURLs = %v, expected only root", result.ValidURLs)
	}
	if len(result.BrokenLinks) != 0 {
		t.Errorf("BrokenLinks = %v, expected none (external links should be ignored)", result.BrokenLinks)
	}
}

func TestCrawl_DefaultsApplied(t *testing.T) {
	// Test that zero-value Options get sensible defaults
	ts := newTestServer()
	defer ts.Close()

	result, err := Crawl(Options{
		RootURL: ts.URL,
		// Threads, UserAgent left as zero/empty to test defaults
	})
	if err != nil {
		t.Fatalf("Crawl() error: %v", err)
	}

	if len(result.ValidURLs) == 0 {
		t.Error("expected some valid URLs with default options")
	}
}

func TestCrawl_TrailingSlashNormalization(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		// Both links point to the same page, one with trailing slash
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="/about">About</a>
			<a href="/about/">About slash</a>
		</body></html>`)
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><p>About</p></body></html>`)
	})
	mux.HandleFunc("/about/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><p>About</p></body></html>`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	result, err := Crawl(Options{
		RootURL:        ts.URL,
		Threads:        1,
		RequestTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Crawl() error: %v", err)
	}

	// /about and /about/ should be normalised to the same URL
	aboutCount := 0
	for _, u := range result.ValidURLs {
		if strings.HasSuffix(u, "/about") {
			aboutCount++
		}
	}
	if aboutCount != 1 {
		t.Errorf("expected exactly 1 /about entry, got %d (ValidURLs: %v)", aboutCount, result.ValidURLs)
	}
}
