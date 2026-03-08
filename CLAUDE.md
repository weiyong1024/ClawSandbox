# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

ClawSandbox — a minimal, easy-to-use virtualized sandbox tool for OpenClaw. Solves the problem of rapidly deploying multiple isolated OpenClaw instances on a single machine, enabling users to run an entire OpenClaw fleet. Open-sourced on GitHub.

## Build/Test/Lint Commands

```bash
# Download dependencies
go mod tidy

# Build CLI binary → bin/clawsandbox
make build

# Run tests
make test

# Build Docker image (first time, ~1.4GB, takes several minutes)
make docker-build
# or via the CLI itself (uses embedded Dockerfile):
./bin/clawsandbox build

# Cross-compile for all platforms
make build-all
```

## Architecture

Go CLI tool (cobra) that manages Docker containers with an embedded Web Dashboard. Key packages:

- `cmd/clawsandbox/` — binary entry point
- `internal/cli/` — cobra commands (build, create, list, start, stop, restart, destroy, desktop, logs, dashboard, config, version)
- `internal/container/` — Docker SDK wrappers (client, image build/check/pull, network, container lifecycle, stats)
- `internal/port/` — TCP port availability checker and allocator
- `internal/state/` — instance metadata store (`~/.clawsandbox/state.json`), mutex-protected
- `internal/config/` — config file loader (`~/.clawsandbox/config.yaml`)
- `internal/assets/` — embedded Docker build context (Dockerfile, supervisord.conf, entrypoint.sh)
- `internal/web/` — Web Dashboard: HTTP server, REST API handlers, WebSocket endpoints (stats/logs/events), embedded frontend
- `internal/version/` — build version info (injected via ldflags)

Each claw instance is a Docker container running: XFCE4 desktop + TigerVNC + noVNC (browser access on port 690N) + OpenClaw Gateway (port 1878N).

Container data is persisted at `~/.clawsandbox/data/<name>/openclaw/` → `/home/node/.openclaw` inside the container.
