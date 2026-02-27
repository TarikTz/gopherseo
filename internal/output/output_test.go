package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tariktz/gopherseo/internal/canonical"
	"github.com/tariktz/gopherseo/internal/crawler"
)

func TestWriteSitemap_BasicOutput(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sitemap.xml")

	urls := []string{
		"https://example.com/",
		"https://example.com/about",
	}

	if err := WriteSitemap(out, urls, nil); err != nil {
		t.Fatalf("WriteSitemap: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	body := string(data)
	for _, u := range urls {
		if !strings.Contains(body, u) {
			t.Errorf("sitemap missing URL %q", u)
		}
	}

	if !strings.Contains(body, "<urlset") {
		t.Error("sitemap missing <urlset> element")
	}
}

func TestWriteSitemap_EmptyURLs(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sitemap.xml")

	if err := WriteSitemap(out, nil, nil); err != nil {
		t.Fatalf("WriteSitemap: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if !strings.Contains(string(data), "<urlset") {
		t.Error("empty sitemap should still contain <urlset>")
	}
}

func TestWriteSitemap_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "nested", "deep", "sitemap.xml")

	if err := WriteSitemap(out, []string{"https://example.com/"}, nil); err != nil {
		t.Fatalf("WriteSitemap: %v", err)
	}

	if _, err := os.Stat(out); os.IsNotExist(err) {
		t.Error("expected file to be created in nested directory")
	}
}

func TestWriteIssueTasks_NoTasks(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "issues.md")

	if err := WriteIssueTasks(out, nil); err != nil {
		t.Fatalf("WriteIssueTasks: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, "No broken links") {
		t.Error("expected 'No broken links' message for empty task list")
	}
}

func TestWriteIssueTasks_WithTasks(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "issues.md")

	tasks := []crawler.BrokenLinkTask{
		{
			URL:     "https://example.com/dead",
			Status:  404,
			Sources: []string{"https://example.com/"},
		},
		{
			URL:     "https://example.com/timeout",
			Status:  0,
			Sources: nil,
		},
	}

	if err := WriteIssueTasks(out, tasks); err != nil {
		t.Fatalf("WriteIssueTasks: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, "# Link Cleanup Tasks") {
		t.Error("missing header")
	}
	if !strings.Contains(body, "https://example.com/dead") {
		t.Error("missing broken link URL")
	}
	if !strings.Contains(body, "404") {
		t.Error("missing status code")
	}
	if !strings.Contains(body, "request_failed") {
		t.Error("expected 'request_failed' for status 0")
	}
	if !strings.Contains(body, "Found on:") {
		t.Error("missing source page reference")
	}
	if !strings.Contains(body, "- [ ] Fix") {
		t.Error("missing checkbox markdown")
	}
}

func TestWriteIssueTasks_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "a", "b", "issues.md")

	if err := WriteIssueTasks(out, nil); err != nil {
		t.Fatalf("WriteIssueTasks: %v", err)
	}

	if _, err := os.Stat(out); os.IsNotExist(err) {
		t.Error("expected file to be created in nested directory")
	}
}

func TestWriteSitemap_InvalidPath(t *testing.T) {
	// Writing to a read-only directory should return an error.
	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(roDir, 0o555); err != nil {
		t.Fatal(err)
	}

	err := WriteSitemap(filepath.Join(roDir, "sub", "sitemap.xml"), nil, nil)
	if err == nil {
		t.Error("expected error writing inside a read-only directory")
	}
}

func TestWriteSitemap_WithLastModified(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sitemap.xml")

	urls := []string{
		"https://example.com/",
		"https://example.com/about",
	}

	lm := map[string]time.Time{
		"https://example.com/":      time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		"https://example.com/about": time.Date(2025, 8, 20, 14, 30, 0, 0, time.UTC),
	}

	if err := WriteSitemap(out, urls, lm); err != nil {
		t.Fatalf("WriteSitemap: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, "<lastmod>") {
		t.Error("sitemap should contain <lastmod> elements")
	}
	if !strings.Contains(body, "2025-06-15") {
		t.Error("sitemap missing date 2025-06-15")
	}
	if !strings.Contains(body, "2025-08-20") {
		t.Error("sitemap missing date 2025-08-20")
	}
}

func TestWriteIssueTasks_InvalidPath(t *testing.T) {
	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(roDir, 0o555); err != nil {
		t.Fatal(err)
	}

	err := WriteIssueTasks(filepath.Join(roDir, "sub", "issues.md"), nil)
	if err == nil {
		t.Error("expected error writing inside a read-only directory")
	}
}

func TestWriteIssueTasks_SourceFallback(t *testing.T) {
	// Task with empty sources should show the "(source page not captured)" fallback.
	dir := t.TempDir()
	out := filepath.Join(dir, "issues.md")

	tasks := []crawler.BrokenLinkTask{
		{URL: "https://example.com/gone", Status: 410, Sources: []string{}},
	}

	if err := WriteIssueTasks(out, tasks); err != nil {
		t.Fatalf("WriteIssueTasks: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if !strings.Contains(string(data), "source page not captured") {
		t.Error("expected source fallback message for task with no sources")
	}
}

func TestWriteCanonicalIssues_NoIssues(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "canonical-issues.md")

	if err := WriteCanonicalIssues(out, nil); err != nil {
		t.Fatalf("WriteCanonicalIssues: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, "No canonical URL issues") {
		t.Error("expected no-issues canonical message")
	}
}

func TestWriteCanonicalIssues_WithIssues(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "canonical-issues.md")

	issues := []canonical.Issue{
		{
			PageURL:      "https://example.com/page-a",
			CanonicalURL: "https://other.com/page-a",
			Type:         canonical.IssueCrossDomain,
			Detail:       "canonical target is on a different host",
		},
	}

	if err := WriteCanonicalIssues(out, issues); err != nil {
		t.Fatalf("WriteCanonicalIssues: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, "# Canonical URL Cleanup Tasks") {
		t.Error("missing canonical report header")
	}
	if !strings.Contains(body, "https://example.com/page-a") {
		t.Error("missing issue page URL")
	}
	if !strings.Contains(body, "cross_domain") {
		t.Error("missing issue type")
	}
}
