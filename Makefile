BINARY     = clawfleet
BUILD_DIR  = ./bin
MODULE     = github.com/clawfleet/clawfleet
IMAGE      = ghcr.io/clawfleet/clawfleet:latest
GO_BOOTSTRAP = ./scripts/ensure-go.sh --print-path

VERSION    ?= $(shell git describe --tags --always --dirty="-pre" 2>/dev/null || echo dev)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS    = -s -w \
  -X '$(MODULE)/internal/version.Version=$(VERSION)' \
  -X '$(MODULE)/internal/version.GitCommit=$(GIT_COMMIT)' \
  -X '$(MODULE)/internal/version.BuildDate=$(BUILD_DATE)'

.PHONY: build build-all docker-build install clean reset tidy test vet

define run-go
	@GO_BIN="$$( $(GO_BOOTSTRAP) )" && \
	$(1)
endef

build:
	@mkdir -p $(BUILD_DIR)
	$(call run-go,env CGO_ENABLED=0 "$$GO_BIN" build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/clawfleet)

build-all:
	@mkdir -p $(BUILD_DIR)
	$(call run-go,env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 "$$GO_BIN" build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/clawfleet)
	$(call run-go,env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 "$$GO_BIN" build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/clawfleet)
	$(call run-go,env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "$$GO_BIN" build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/clawfleet)
	$(call run-go,env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 "$$GO_BIN" build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/clawfleet)

docker-build:
	docker build -t $(IMAGE) -f internal/assets/docker/Dockerfile internal/assets/docker/

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

test:
	$(call run-go,"$$GO_BIN" test ./...)

vet:
	$(call run-go,"$$GO_BIN" vet ./...)

tidy:
	$(call run-go,"$$GO_BIN" mod tidy)

clean:
	rm -rf $(BUILD_DIR)

reset:
	@echo "==> Destroying all instances..."
	@if [ -f $(BUILD_DIR)/$(BINARY) ]; then \
		echo y | $(BUILD_DIR)/$(BINARY) destroy --purge all 2>/dev/null || true; \
		$(BUILD_DIR)/$(BINARY) dashboard stop 2>/dev/null || true; \
	fi
	@echo "==> Removing state directory (~/.clawfleet)..."
	rm -rf $(HOME)/.clawfleet
	@echo "==> Removing build artifacts..."
	rm -rf $(BUILD_DIR)
	@echo "==> Done. Run 'make build' to start fresh."
