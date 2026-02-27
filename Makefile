.PHONY: build test lint clean install

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/tariktz/gopherseo/cmd.Version=$(VERSION)
BINARY  := gopherseo

## build: Compile the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## install: Install the binary to $GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" .

## test: Run all tests
test:
	go test -race -count=1 ./...

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format all Go source files
fmt:
	gofmt -s -w .
	goimports -w .

## vet: Run go vet
vet:
	go vet ./...

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
