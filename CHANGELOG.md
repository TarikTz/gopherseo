# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- Canonical URL extraction from `<link rel="canonical">` including relative URL resolution and URL normalization.
- Canonical validation rules for non-HTTP schemes, cross-domain targets, redirect/broken canonical targets, and loop/chain detection.
- Canonical issue summary in CLI crawl output.
- Canonical issue report generation to `canonical-issues.md` via `--canonical-report-output`.

### Changed
- README updated with canonical report flag, output documentation, and sample report block.
