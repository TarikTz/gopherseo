package lastmod

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// helper to build a *goquery.Document from raw HTML.
func docFromHTML(html string) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		panic(err)
	}
	return doc
}

var fixedNow = time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)

// --- JSON-LD tests ---

func TestJSONLD_DateModified(t *testing.T) {
	html := `<html><head>
	<script type="application/ld+json">
	{"@type":"Article","dateModified":"2025-06-15T10:30:00Z"}
	</script>
	</head><body></body></html>`

	got := GetLastModified(nil, docFromHTML(html), fixedNow)
	want := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("JSON-LD dateModified: got %v, want %v", got, want)
	}
}

func TestJSONLD_GraphArray(t *testing.T) {
	html := `<html><head>
	<script type="application/ld+json">
	{"@graph":[
		{"@type":"WebSite","name":"Test"},
		{"@type":"Article","dateModified":"2024-12-01"}
	]}
	</script>
	</head><body></body></html>`

	got := GetLastModified(nil, docFromHTML(html), fixedNow)
	want := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("JSON-LD @graph: got %v, want %v", got, want)
	}
}

func TestJSONLD_ArrayOfObjects(t *testing.T) {
	html := `<html><head>
	<script type="application/ld+json">
	[{"@type":"BreadcrumbList"},{"@type":"Article","dateModified":"2025-01-20T08:00:00+01:00"}]
	</script>
	</head><body></body></html>`

	got := GetLastModified(nil, docFromHTML(html), fixedNow)
	want := time.Date(2025, 1, 20, 7, 0, 0, 0, time.UTC) // +01:00 -> UTC
	if !got.Equal(want) {
		t.Errorf("JSON-LD array: got %v, want %v", got, want)
	}
}

func TestJSONLD_InvalidJSON(t *testing.T) {
	html := `<html><head>
	<script type="application/ld+json">{not valid json</script>
	<meta property="article:modified_time" content="2025-03-10">
	</head><body></body></html>`

	got := GetLastModified(nil, docFromHTML(html), fixedNow)
	// Should fall through to meta tag.
	want := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("invalid JSON-LD fallback: got %v, want %v", got, want)
	}
}

func TestJSONLD_EmptyScript(t *testing.T) {
	html := `<html><head>
	<script type="application/ld+json">   </script>
	<meta property="article:modified_time" content="2025-09-01">
	</head><body></body></html>`

	got := GetLastModified(nil, docFromHTML(html), fixedNow)
	want := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("empty JSON-LD fallback: got %v, want %v", got, want)
	}
}

// --- Meta tag tests ---

func TestMeta_ArticleModifiedTime(t *testing.T) {
	html := `<html><head>
	<meta property="article:modified_time" content="2025-08-20T14:00:00Z">
	</head><body></body></html>`

	got := GetLastModified(nil, docFromHTML(html), fixedNow)
	want := time.Date(2025, 8, 20, 14, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("article:modified_time: got %v, want %v", got, want)
	}
}

func TestMeta_OGUpdatedTime(t *testing.T) {
	html := `<html><head>
	<meta property="og:updated_time" content="2025-07-10">
	</head><body></body></html>`

	got := GetLastModified(nil, docFromHTML(html), fixedNow)
	want := time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("og:updated_time: got %v, want %v", got, want)
	}
}

func TestMeta_PriorityOrder(t *testing.T) {
	// article:modified_time should win over og:updated_time.
	html := `<html><head>
	<meta property="article:modified_time" content="2025-11-01">
	<meta property="og:updated_time" content="2025-10-01">
	</head><body></body></html>`

	got := GetLastModified(nil, docFromHTML(html), fixedNow)
	want := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("meta priority: got %v, want %v", got, want)
	}
}

// --- HTTP header tests ---

func TestHeader_LastModified(t *testing.T) {
	html := `<html><head></head><body></body></html>`
	h := http.Header{}
	h.Set("Last-Modified", "Wed, 15 Jan 2025 10:00:00 GMT")

	got := GetLastModified(h, docFromHTML(html), fixedNow)
	want := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Last-Modified header: got %v, want %v", got, want)
	}
}

func TestHeader_RFC850(t *testing.T) {
	html := `<html><head></head><body></body></html>`
	h := http.Header{}
	h.Set("Last-Modified", "Wednesday, 15-Jan-25 10:00:00 GMT")

	got := GetLastModified(h, docFromHTML(html), fixedNow)
	want := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("RFC850 header: got %v, want %v", got, want)
	}
}

// --- Fallback tests ---

func TestFallback_NilDoc(t *testing.T) {
	got := GetLastModified(nil, nil, fixedNow)
	if !got.Equal(fixedNow) {
		t.Errorf("nil doc fallback: got %v, want %v", got, fixedNow)
	}
}

func TestFallback_EmptyPage(t *testing.T) {
	html := `<html><head></head><body></body></html>`
	got := GetLastModified(nil, docFromHTML(html), fixedNow)
	if !got.Equal(fixedNow) {
		t.Errorf("empty page fallback: got %v, want %v", got, fixedNow)
	}
}

func TestFallback_UnparsableHeader(t *testing.T) {
	html := `<html><head></head><body></body></html>`
	h := http.Header{}
	h.Set("Last-Modified", "not-a-date")

	got := GetLastModified(h, docFromHTML(html), fixedNow)
	if !got.Equal(fixedNow) {
		t.Errorf("bad header fallback: got %v, want %v", got, fixedNow)
	}
}

// --- Priority hierarchy tests ---

func TestPriority_JSONLDWinsOverMeta(t *testing.T) {
	html := `<html><head>
	<script type="application/ld+json">
	{"@type":"Article","dateModified":"2025-01-01"}
	</script>
	<meta property="article:modified_time" content="2024-06-01">
	</head><body></body></html>`

	h := http.Header{}
	h.Set("Last-Modified", "Mon, 01 Jan 2024 00:00:00 GMT")

	got := GetLastModified(h, docFromHTML(html), fixedNow)
	want := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("JSON-LD priority: got %v, want %v", got, want)
	}
}

func TestPriority_MetaWinsOverHeader(t *testing.T) {
	html := `<html><head>
	<meta property="article:modified_time" content="2025-05-05">
	</head><body></body></html>`

	h := http.Header{}
	h.Set("Last-Modified", "Mon, 01 Jan 2024 00:00:00 GMT")

	got := GetLastModified(h, docFromHTML(html), fixedNow)
	want := time.Date(2025, 5, 5, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("meta over header: got %v, want %v", got, want)
	}
}

// --- FormatW3C tests ---

func TestFormatW3C(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"UTC date", time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC), "2025-06-15"},
		{"with time", time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC), "2025-01-02"},
		{"non-UTC tz", time.Date(2025, 3, 10, 22, 0, 0, 0, time.FixedZone("CET", 3600)), "2025-03-10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatW3C(tt.t); got != tt.want {
				t.Errorf("FormatW3C() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- parseTime tests ---

func TestParseTime_Formats(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{"RFC3339", "2025-06-15T10:30:00Z", time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)},
		{"RFC3339 offset", "2025-06-15T10:30:00+02:00", time.Date(2025, 6, 15, 10, 30, 0, 0, time.FixedZone("", 7200))},
		{"date only", "2025-06-15", time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)},
		{"no tz", "2025-06-15T10:30:00", time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)},
		{"RFC1123", "Mon, 02 Jan 2006 15:04:05 MST", time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("MST", 0))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseTime(tt.input)
			if !ok {
				t.Fatalf("parseTime(%q) returned false", tt.input)
			}
			if !got.Equal(tt.want) {
				t.Errorf("parseTime(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTime_Invalid(t *testing.T) {
	invalids := []string{"", "   ", "not-a-date", "yesterday", "2025/06/15"}
	for _, s := range invalids {
		if _, ok := parseTime(s); ok {
			t.Errorf("parseTime(%q) should return false", s)
		}
	}
}
