BINARY     = clawsandbox
BUILD_DIR  = ./bin
MODULE     = github.com/weiyong1024/clawsandbox
IMAGE      = clawsandbox/openclaw:latest

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS    = -s -w \
  -X '$(MODULE)/internal/version.Version=$(VERSION)' \
  -X '$(MODULE)/internal/version.GitCommit=$(GIT_COMMIT)' \
  -X '$(MODULE)/internal/version.BuildDate=$(BUILD_DATE)'

.PHONY: build build-all docker-build install clean reset tidy test vet

build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/clawsandbox

build-all:
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64  ./cmd/clawsandbox
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64  ./cmd/clawsandbox
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64   ./cmd/clawsandbox
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-arm64   ./cmd/clawsandbox

docker-build:
	docker build -t $(IMAGE) -f internal/assets/docker/Dockerfile internal/assets/docker/

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BUILD_DIR)

reset:
	@echo "==> Destroying all claw instances..."
	@if [ -f $(BUILD_DIR)/$(BINARY) ]; then \
		echo y | $(BUILD_DIR)/$(BINARY) destroy --purge all 2>/dev/null || true; \
		$(BUILD_DIR)/$(BINARY) dashboard stop 2>/dev/null || true; \
	fi
	@echo "==> Removing state directory (~/.clawsandbox)..."
	rm -rf $(HOME)/.clawsandbox
	@echo "==> Removing build artifacts..."
	rm -rf $(BUILD_DIR)
	@echo "==> Done. Run 'make build' to start fresh."
