CGO_ENABLED ?= 0
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
MACOSX_DEPLOYMENT_TARGET ?= 10.14

VERSION ?= 0.1.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

.PHONY: build-amd64 build-arm64 build-all test lint clean run

build-amd64:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		go build -ldflags "$(LDFLAGS)" -o bin/inice-darwin-amd64 .

build-arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) \
		go build -ldflags "$(LDFLAGS)" -o bin/inice-darwin-arm64 .

build-all: build-amd64 build-arm64

build-universal: build-amd64 build-arm64
	lipo -create -output bin/inice bin/inice-darwin-amd64 bin/inice-darwin-arm64

run:
	go run . --router $(ROUTER) --user $(USER)

test:
	go test ./... -race -count=1 -short

test-v:
	go test ./... -race -count=1 -v

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
	go clean -cache

deps:
	go mod tidy
	go mod download
