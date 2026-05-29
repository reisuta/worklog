VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
GOBIN   ?= $(shell go env GOPATH)/bin

.PHONY: all build test vet fmt lint install clean

all: build

build: ## Build both binaries into ./bin
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/worklog ./cmd/worklog
	go build $(LDFLAGS) -o bin/worklog-statusbar ./cmd/worklog-statusbar

test: ## Run tests with the race detector
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint: ## Run golangci-lint (must be installed)
	golangci-lint run

install: build ## Install both binaries to GOBIN
	go install $(LDFLAGS) ./cmd/worklog
	go install $(LDFLAGS) ./cmd/worklog-statusbar

clean:
	rm -rf bin dist
