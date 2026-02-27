// Package lastmod extracts the most trustworthy "last modified" timestamp
// from an HTTP response using a priority-based extraction hierarchy:
//
//  1. JSON-LD structured data (dateModified)
//  2. HTML meta tags (article:modified_time, og:updated_time)
//  3. HTTP Last-Modified header
//  4. Fallback: current crawl time
package lastmod

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// knownFormats lists the date/time layouts we try when parsing timestamps
// found in HTML or HTTP headers. They are attempted in order.
var knownFormats = []string{
	time.RFC3339,           // 2006-01-02T15:04:05Z07:00
	"2006-01-02T15:04:05Z", // UTC explicit
	"2006-01-02T15:04:05",  // no tz
	"2006-01-02",           // date only (W3C short)
	time.RFC1123,           // Mon, 02 Jan 2006 15:04:05 MST
	time.RFC1123Z,          // Mon, 02 Jan 2006 15:04:05 -0700
	time.RFC850,            // Monday, 02-Jan-06 15:04:05 MST
	"Mon, 2 Jan 2006 15:04:05 MST",
}

// GetLastModified returns the best available "last modified" time for a page.
// It inspects (in priority order): JSON-LD dateModified, HTML meta tags,
// the HTTP Last-Modified header, and finally falls back to time.Now().
//
// The returned time is always in UTC.
func GetLastModified(header http.Header, doc *goquery.Document, now time.Time) time.Time {
	if doc != nil {
		// 1. JSON-LD structured data.
		if t, ok := fromJSONLD(doc); ok {
			return t.UTC()
		}

		// 2. HTML meta tags.
		if t, ok := fromMetaTags(doc); ok {
			return t.UTC()
		}
	}

	// 3. HTTP Last-Modified header.
	if header != nil {
		if t, ok := fromHeader(header); ok {
			return t.UTC()
		}
	}

	// 4. Fallback.
	return now.UTC()
}

// fromJSONLD scans all <script type="application/ld+json"> blocks for a
// "dateModified" key. If the JSON is an array of objects, each element is
// checked.
func fromJSONLD(doc *goquery.Document) (time.Time, bool) {
	var result time.Time
	var found bool

	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		raw := strings.TrimSpace(s.Text())
		if raw == "" {
			return true // continue
		}

		// Try single object first.
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &obj); err == nil {
			if t, ok := extractDateModified(obj); ok {
				result = t
				found = true
				return false // break
			}
			return true
		}

		// Try array of objects.
		var arr []map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &arr); err == nil {
			for _, item := range arr {
				if t, ok := extractDateModified(item); ok {
					result = t
					found = true
					return false
				}
			}
		}

		return true
	})

	return result, found
}

// extractDateModified looks for "dateModified" in a JSON-LD object,
// including inside a nested "@graph" array.
func extractDateModified(obj map[string]interface{}) (time.Time, bool) {
	if val, ok := obj["dateModified"]; ok {
		if s, ok := val.(string); ok {
			if t, ok := parseTime(s); ok {
				return t, true
			}
		}
	}

	// Check @graph array (common in WordPress JSON-LD).
	if graph, ok := obj["@graph"]; ok {
		if items, ok := graph.([]interface{}); ok {
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					if t, ok := extractDateModified(m); ok {
						return t, true
					}
				}
			}
		}
	}

	return time.Time{}, false
}

// fromMetaTags checks <meta> tags for article:modified_time and
// og:updated_time (in that order).
func fromMetaTags(doc *goquery.Document) (time.Time, bool) {
	selectors := []string{
		`meta[property="article:modified_time"]`,
		`meta[property="og:updated_time"]`,
	}

	for _, sel := range selectors {
		if val, exists := doc.Find(sel).First().Attr("content"); exists {
			val = strings.TrimSpace(val)
			if t, ok := parseTime(val); ok {
				return t, true
			}
		}
	}

	return time.Time{}, false
}

// fromHeader parses the HTTP Last-Modified header.
func fromHeader(header http.Header) (time.Time, bool) {
	raw := strings.TrimSpace(header.Get("Last-Modified"))
	if raw == "" {
		return time.Time{}, false
	}
	return parseTime(raw)
}

// parseTime attempts to parse a date string against all known formats.
func parseTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range knownFormats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// FormatW3C formats a time as a W3C Datetime / ISO 8601 date (YYYY-MM-DD).
func FormatW3C(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}
