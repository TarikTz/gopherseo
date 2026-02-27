package crawler

import (
	"net/url"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "adds root slash", input: "https://example.com", want: "https://example.com/"},
		{name: "strips fragment", input: "https://example.com/page#section", want: "https://example.com/page"},
		{name: "strips trailing slash on non-root", input: "https://example.com/about/", want: "https://example.com/about"},
		{name: "keeps root slash", input: "https://example.com/", want: "https://example.com/"},
		{name: "preserves query string", input: "https://example.com/search?q=test", want: "https://example.com/search?q=test"},
		{name: "strips trailing slash with query", input: "https://example.com/page/?q=1", want: "https://example.com/page?q=1"},
		{name: "strips fragment but keeps query", input: "https://example.com/page?q=1#frag", want: "https://example.com/page?q=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := normalizeURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeRoot(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "https URL", input: "https://example.com", want: "https://example.com/"},
		{name: "http URL", input: "http://example.com", want: "http://example.com/"},
		{name: "bare domain adds https", input: "example.com", want: "https://example.com/"},
		{name: "with trailing slash", input: "https://example.com/", want: "https://example.com/"},
		{name: "empty string", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := normalizeRoot(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("normalizeRoot(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func mustParse(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func TestIsHTTP(t *testing.T) {
	tests := []struct {
		name string
		url  *url.URL
		want bool
	}{
		{"https", mustParse("https://example.com"), true},
		{"http", mustParse("http://example.com"), true},
		{"ftp", mustParse("ftp://example.com"), false},
		{"mailto", mustParse("mailto:user@example.com"), false},
		{"javascript", mustParse("javascript:void(0)"), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHTTP(tt.url); got != tt.want {
				t.Errorf("isHTTP(%v) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestIsInternal(t *testing.T) {
	root := mustParse("https://example.com")

	tests := []struct {
		name      string
		candidate *url.URL
		want      bool
	}{
		{"same domain", mustParse("https://example.com/page"), true},
		{"same domain different case", mustParse("https://Example.COM/page"), true},
		{"subdomain", mustParse("https://sub.example.com/page"), false},
		{"different domain", mustParse("https://other.com/page"), false},
		{"nil candidate", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInternal(root, tt.candidate); got != tt.want {
				t.Errorf("isInternal(root, %v) = %v, want %v", tt.candidate, got, tt.want)
			}
		})
	}

	t.Run("nil root", func(t *testing.T) {
		if got := isInternal(nil, mustParse("https://example.com")); got != false {
			t.Error("expected false for nil root")
		}
	})
}

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		patterns []string
		want     bool
	}{
		{name: "no patterns", link: "https://example.com/page", patterns: nil, want: false},
		{name: "empty patterns", link: "https://example.com/page", patterns: []string{"", "  "}, want: false},
		{name: "match path glob", link: "https://example.com/print/page", patterns: []string{"*/print/*"}, want: true},
		{name: "match file extension", link: "https://example.com/file.pdf", patterns: []string{"*.pdf"}, want: true},
		{name: "no match", link: "https://example.com/about", patterns: []string{"*.pdf", "*/print/*"}, want: false},
		{name: "match query pattern", link: "https://example.com/page?lang=rs", patterns: []string{"*?lang=rs"}, want: true},
		{name: "match path only", link: "https://example.com/admin/settings", patterns: []string{"/admin/*"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldExclude(tt.link, tt.patterns); got != tt.want {
				t.Errorf("shouldExclude(%q, %v) = %v, want %v", tt.link, tt.patterns, got, tt.want)
			}
		})
	}
}
