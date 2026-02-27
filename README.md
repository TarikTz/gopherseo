<p align="center">
  <img src="gopherseo.png" alt="GopherSEO logo" width="200"/>
</p>

<h1 align="center">GopherSEO</h1>

<p align="center">
  <strong>A fast, concurrent CLI tool for crawling websites, generating sitemaps, and finding broken links.</strong>
</p>

<p align="center">
  <a href="https://github.com/tariktz/gopherseo/actions"><img src="https://github.com/tariktz/gopherseo/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://goreportcard.com/report/github.com/tariktz/gopherseo"><img src="https://goreportcard.com/badge/github.com/tariktz/gopherseo" alt="Go Report Card"></a>
  <a href="https://github.com/tariktz/gopherseo/releases"><img src="https://img.shields.io/github/v/release/tariktz/gopherseo" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
</p>

---

## What it does

GopherSEO crawls a given root URL, recursively discovers all internal pages, validates their HTTP status codes, and produces:

- A **sitemap.xml** (Sitemap 0.9 schema) ready for search engine submission
- A **broken-link report** (Markdown) with actionable fix tasks including source pages

## Features

- Recursive internal link discovery from `<a href>` tags
- Adjustable crawl depth (`--depth`, `0` = unlimited)
- Adjustable concurrency (`--threads`)
- Broken-link detection with source page tracking
- Canonical URL validation (missing/multiple tags, cross-domain, redirect/broken targets, chains/loops)
- Markdown task report for broken links (`broken-link-tasks.md`)
- Markdown task report for canonical issues (`canonical-issues.md`)
- Custom User-Agent (`--user-agent`)
- URL exclusion rules via glob patterns (`--exclude`)
- `robots.txt` compliance via [Colly](https://github.com/gocolly/colly)
- Live terminal crawl indicator while pages are being processed

## Installation

### From source

Requires [Go 1.22+](https://go.dev/dl/):

```bash
go install github.com/tariktz/gopherseo@latest
```

### Pre-built binaries

Download the latest release from the [Releases page](https://github.com/tariktz/gopherseo/releases).

### Build from source

```bash
git clone https://github.com/tariktz/gopherseo.git
cd gopherseo
make build
```

## Quick start

```bash
# Crawl a site, generate sitemap and broken-link report
gopherseo crawl https://example.com

# With custom options
gopherseo crawl https://example.com \
  --output ./sitemap.xml \
  --issues-output ./broken-link-tasks.md \
  --canonical-report-output ./canonical-issues.md \
  --threads 10 \
  --depth 5
```

## Usage

```
gopherseo crawl <url> [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `./sitemap.xml` | Output path for the generated sitemap |
| `--issues-output` | | `./broken-link-tasks.md` | Output path for broken-link fix tasks |
| `--canonical-report-output` | | `./canonical-issues.md` | Output path for canonical URL issue tasks |
| `--threads` | | `5` | Maximum concurrent crawler workers |
| `--depth` | | `0` | Max crawl depth (`0` = unlimited) |
| `--user-agent` | | `GopherSEO-Bot/1.0` | Crawler User-Agent string |
| `--exclude` | | | Glob pattern to skip (repeatable) |

### Global commands

```bash
gopherseo version    # Print version
gopherseo --help     # Show help
```

### Exclusion examples

```bash
gopherseo crawl https://example.com \
  --exclude '*/print/*' \
  --exclude '*.pdf' \
  --exclude '*?lang=rs'
```

## Output

### sitemap.xml

A standard [Sitemap 0.9](https://www.sitemaps.org/protocol.html) XML file containing all discovered valid URLs, ready to submit to Google Search Console or other search engines.

### broken-link-tasks.md

A Markdown checklist of broken links found during the crawl. Its purpose is to provide an actionable cleanup queue you can use in issues, PRs, or maintenance sprints. Each entry includes the broken URL, its HTTP status code, and every page where the broken link appears:

```markdown
- [ ] Fix `https://example.com/missing-page` (status: 404)
  - Found on: `https://example.com/about`
  - Found on: `https://example.com/contact`
```

### canonical-issues.md

A Markdown checklist of canonical URL problems found during the crawl. Its purpose is to provide an actionable queue for SEO canonical cleanup and duplicate-content prevention.

```markdown
- [ ] Resolve canonical issue on `https://example.com/page-a`
  - Type: `cross_domain`
  - Canonical target: `https://other-domain.com/page-a`
  - Detail: canonical target is on a different host
```

## Roadmap

Planned features for upcoming releases:

- [ ] Canonical URL validation
- [ ] Meta tag analysis (title, description, OG tags)
- [ ] `robots.txt` parsing and analysis
- [ ] Core Web Vitals integration
- [ ] Schema.org / structured data validation
- [ ] HTML report output
- [ ] JSON export format

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

## Acknowledgements

Built with these excellent Go libraries:

- [Colly](https://github.com/gocolly/colly) — Elegant scraper and crawler framework
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [goquery](https://github.com/PuerkitoBio/goquery) — HTML parsing for metadata extraction
