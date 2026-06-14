SHELL := /bin/bash
GO    ?= go

.PHONY: all build test test-race cover lint vet tidy clean

all: vet test

build:
	$(GO) build ./...

test:
	$(GO) test ./...

test-race:
	$(GO) test -race -count=1 ./...

cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -n 1

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

lint:
	@which golangci-lint >/dev/null 2>&1 || { echo "install golangci-lint first"; exit 1; }
	golangci-lint run ./...

clean:
	rm -f coverage.out
