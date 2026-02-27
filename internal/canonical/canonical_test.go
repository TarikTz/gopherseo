package canonical

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func docFromHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("build document: %v", err)
	}
	return doc
}

func TestExtract_MissingCanonical(t *testing.T) {
	doc := docFromHTML(t, `<html><head></head><body></body></html>`)

	info := Extract("https://example.com/page", doc)
	if !info.Missing {
		t.Fatal("expected Missing=true")
	}
	if info.TagCount != 0 {
		t.Fatalf("TagCount=%d, want 0", info.TagCount)
	}
}

func TestExtract_AbsoluteCanonical(t *testing.T) {
	doc := docFromHTML(t, `<html><head><link rel="canonical" href="https://example.com/about/"/></head></html>`)

	info := Extract("https://example.com/page", doc)
	if info.CanonicalURL != "https://example.com/about" {
		t.Fatalf("CanonicalURL=%q, want %q", info.CanonicalURL, "https://example.com/about")
	}
	if info.Missing {
		t.Fatal("expected Missing=false")
	}
}

func TestExtract_RelativeCanonical(t *testing.T) {
	doc := docFromHTML(t, `<html><head><link rel="canonical" href="/services/seo/"/></head></html>`)

	info := Extract("https://example.com/page", doc)
	if info.CanonicalURL != "https://example.com/services/seo" {
		t.Fatalf("CanonicalURL=%q, want %q", info.CanonicalURL, "https://example.com/services/seo")
	}
}

func TestExtract_MultipleCanonical(t *testing.T) {
	doc := docFromHTML(t, `<html><head>
		<link rel="canonical" href="https://example.com/a"/>
		<link rel="canonical" href="https://example.com/b"/>
	</head></html>`)

	info := Extract("https://example.com/page", doc)
	if !info.Multiple {
		t.Fatal("expected Multiple=true")
	}
	if info.TagCount != 2 {
		t.Fatalf("TagCount=%d, want 2", info.TagCount)
	}
	if info.CanonicalURL != "https://example.com/a" {
		t.Fatalf("CanonicalURL=%q, want first canonical href", info.CanonicalURL)
	}
}

func TestExtract_StripsFragment(t *testing.T) {
	doc := docFromHTML(t, `<html><head><link rel="canonical" href="https://example.com/about#section"/></head></html>`)

	info := Extract("https://example.com/page", doc)
	if info.CanonicalURL != "https://example.com/about" {
		t.Fatalf("CanonicalURL=%q, want %q", info.CanonicalURL, "https://example.com/about")
	}
}

func TestExtract_EmptyHrefCanonical(t *testing.T) {
	doc := docFromHTML(t, `<html><head><link rel="canonical" href=""/></head></html>`)

	info := Extract("https://example.com/page", doc)
	if !info.Missing {
		t.Fatal("expected Missing=true for empty href")
	}
	if info.TagCount != 1 {
		t.Fatalf("TagCount=%d, want 1", info.TagCount)
	}
}

func TestValidate_NonHTTPScheme(t *testing.T) {
	issues := Validate(
		map[string]string{"https://example.com/page": "mailto:seo@example.com"},
		map[string]int{},
	)

	if len(issues) != 1 {
		t.Fatalf("issues len=%d, want 1", len(issues))
	}
	if issues[0].Type != IssueNonHTTPScheme {
		t.Fatalf("issue type=%s, want %s", issues[0].Type, IssueNonHTTPScheme)
	}
}

func TestValidate_CrossDomain(t *testing.T) {
	issues := Validate(
		map[string]string{"https://example.com/page": "https://other.com/target"},
		map[string]int{"https://other.com/target": 200},
	)

	if len(issues) != 1 {
		t.Fatalf("issues len=%d, want 1", len(issues))
	}
	if issues[0].Type != IssueCrossDomain {
		t.Fatalf("issue type=%s, want %s", issues[0].Type, IssueCrossDomain)
	}
}

func TestValidate_TargetRedirect(t *testing.T) {
	issues := Validate(
		map[string]string{"https://example.com/page": "https://example.com/canonical"},
		map[string]int{"https://example.com/canonical": 301},
	)

	if len(issues) != 1 {
		t.Fatalf("issues len=%d, want 1", len(issues))
	}
	if issues[0].Type != IssueTargetRedirect {
		t.Fatalf("issue type=%s, want %s", issues[0].Type, IssueTargetRedirect)
	}
}

func TestValidate_TargetBroken(t *testing.T) {
	issues := Validate(
		map[string]string{"https://example.com/page": "https://example.com/canonical"},
		map[string]int{"https://example.com/canonical": 404},
	)

	if len(issues) != 1 {
		t.Fatalf("issues len=%d, want 1", len(issues))
	}
	if issues[0].Type != IssueTargetBroken {
		t.Fatalf("issue type=%s, want %s", issues[0].Type, IssueTargetBroken)
	}
}

func TestValidate_CanonicalChain(t *testing.T) {
	issues := Validate(
		map[string]string{
			"https://example.com/a": "https://example.com/b",
			"https://example.com/b": "https://example.com/c",
		},
		map[string]int{
			"https://example.com/b": 200,
			"https://example.com/c": 200,
		},
	)

	found := false
	for _, issue := range issues {
		if issue.Type == IssueLoopOrChain && issue.PageURL == "https://example.com/a" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected loop_or_chain issue for canonical chain")
	}
}

func TestValidate_CanonicalLoop(t *testing.T) {
	issues := Validate(
		map[string]string{
			"https://example.com/a": "https://example.com/b",
			"https://example.com/b": "https://example.com/a",
		},
		map[string]int{
			"https://example.com/a": 200,
			"https://example.com/b": 200,
		},
	)

	found := false
	for _, issue := range issues {
		if issue.Type == IssueLoopOrChain {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected loop_or_chain issue for canonical loop")
	}
}

func TestValidate_SelfCanonicalIsOK(t *testing.T) {
	issues := Validate(
		map[string]string{"https://example.com/page": "https://example.com/page"},
		map[string]int{"https://example.com/page": 200},
	)

	if len(issues) != 0 {
		t.Fatalf("issues len=%d, want 0", len(issues))
	}
}
