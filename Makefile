BINARY  = public-api-mcp
PREFIX  = /usr/local
BINDIR  = $(PREFIX)/bin
LDFLAGS = -s -w

.DEFAULT_GOAL := build

.PHONY: build clean install uninstall test test-cover deps fmt vet run-stdio run-http

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

clean:
	rm -f $(BINARY) coverage.out coverage.html
	go clean -testcache

install: build
	install -d $(BINDIR)
	install -m 755 $(BINARY) $(BINDIR)/$(BINARY)

uninstall:
	rm -f $(BINDIR)/$(BINARY)

test:
	go test -race ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "커버리지 리포트: coverage.html"

deps:
	go mod tidy
	go mod download

fmt:
	go fmt ./...

vet:
	go vet ./...

run-stdio: build
	./$(BINARY) serve stdio

run-http: build
	./$(BINARY) serve http --addr :8080
