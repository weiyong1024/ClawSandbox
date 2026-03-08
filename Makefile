BINARY     = clawsandbox
BUILD_DIR  = ./bin
MODULE     = github.com/weiyong1024/clawsandbox
IMAGE      = clawsandbox/openclaw:latest

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS    = -s -w \
  -X '$(MODULE)/internal/cli.Version=$(VERSION)' \
  -X '$(MODULE)/internal/cli.GitCommit=$(GIT_COMMIT)' \
  -X '$(MODULE)/internal/cli.BuildDate=$(BUILD_DATE)'

.PHONY: build build-all docker-build install clean tidy

build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/clawsandbox

build-all:
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64  ./cmd/clawsandbox
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64  ./cmd/clawsandbox
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64   ./cmd/clawsandbox
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-arm64   ./cmd/clawsandbox

docker-build:
	docker build -t $(IMAGE) -f internal/assets/docker/Dockerfile internal/assets/docker/

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

tidy:
	go mod tidy

clean:
	rm -rf $(BUILD_DIR)
