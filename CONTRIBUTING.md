# Contributing to GopherSEO

Thank you for your interest in contributing to GopherSEO! This document provides guidelines and instructions for contributing.

## Getting started

1. **Fork** the repository on GitHub
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/gopherseo.git
   cd gopherseo
   ```
3. **Create a branch** for your change:
   ```bash
   git checkout -b feat/my-new-feature
   ```
4. **Install dependencies**:
   ```bash
   go mod download
   ```

## Development

### Build

```bash
make build
```

### Run tests

```bash
make test
```

### Lint

We use `golangci-lint`. Install it via [their docs](https://golangci-lint.run/welcome/install/) and run:

```bash
make lint
```

## Code style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Add doc comments to all exported types, functions, and methods
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Keep functions focused and reasonably short

## Pull request process

1. Ensure your code **compiles** and **passes all tests**
2. Update documentation if your change affects user-facing behaviour
3. Write a clear PR description explaining **what** and **why**
4. Reference any related issues (e.g., `Closes #42`)
5. Keep commits atomic — one logical change per commit

## Commit messages

Use clear, descriptive commit messages:

```
feat: add canonical URL validation
fix: handle redirects in broken-link detection
docs: update installation instructions
refactor: extract URL normalization helpers
```

## Reporting bugs

Open an issue with:

- GopherSEO version (`gopherseo version`)
- OS and Go version
- Steps to reproduce
- Expected vs actual behaviour

## Suggesting features

Open an issue describing:

- The problem you want to solve
- Your proposed solution
- Any alternatives you considered

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
