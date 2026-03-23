# ClawSandbox System Design

> Version: v0.1 (Draft)
> Date: 2026-03-07

[中文文档](./SYSTEM_DESIGN.zh-CN.md)

---

## 1. Overview

ClawSandbox is a CLI tool that creates and manages multiple isolated OpenClaw instances on a single host machine. Each instance runs in its own Docker container with a full Linux desktop environment, accessible from any browser via noVNC. Users interact with each instance independently through Telegram.

## 2. System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Host Machine (macOS / Linux)              │
│                                                             │
│  ┌───────────────────────────────────┐                      │
│  │        ClawSandbox CLI (Go)       │                      │
│  │  ┌─────────┬──────────┬────────┐  │                      │
│  │  │ create  │  list    │ desktop│  │                      │
│  │  │ start   │  stop    │destroy │  │                      │
│  │  └─────────┴──────────┴────────┘  │                      │
│  │  ┌─────────────────────────────┐  │                      │
│  │  │   Docker Engine SDK (Go)    │  │                      │
│  │  └──────────────┬──────────────┘  │                      │
│  │                 │                 │                      │
│  │  ┌──────────────┴──────────────┐  │                      │
│  │  │     Port Allocator          │  │                      │
│  │  │  noVNC: 6901, 6902, ...     │  │                      │
│  │  │  Gateway: 18789, 18790, ... │  │                      │
│  │  └─────────────────────────────┘  │                      │
│  │  ┌─────────────────────────────┐  │                      │
│  │  │     Instance State Store    │  │                      │
│  │  │  ~/.clawfleet/state.json  │  │                      │
│  │  └─────────────────────────────┘  │                      │
│  └───────────────────────────────────┘                      │
│                     │ Docker API                            │
│    ┌────────────────┼────────────────────────┐              │
│    │                │                        │              │
│    ▼                ▼                        ▼              │
│  ┌──────────┐  ┌──────────┐           ┌──────────┐         │
│  │claw-1 │  │claw-2 │    ...    │claw-N │         │
│  │          │  │          │           │          │         │
│  │ XFCE     │  │ XFCE     │           │ XFCE     │         │
│  │ noVNC    │  │ noVNC    │           │ noVNC    │         │
│  │ Node≥22  │  │ Node≥22  │           │ Node≥22  │         │
│  │ Chromium │  │ Chromium │           │ Chromium │         │
│  │ OpenClaw │  │ OpenClaw │           │ OpenClaw │         │
│  │ Gateway  │  │ Gateway  │           │ Gateway  │         │
│  └──────────┘  └──────────┘           └──────────┘         │
│   :6901/:18789  :6902/:18790           :690N/:1878(8+N)    │
└─────────────────────────────────────────────────────────────┘
```

## 3. Component Design

### 3.1 ClawSandbox CLI

**Stack:**
- Language: Go 1.25+
- CLI framework: [cobra](https://github.com/spf13/cobra)
- Docker integration: [go-dockerclient](https://github.com/fsouza/go-dockerclient)
- Artifact: single statically-linked binary (darwin/arm64, darwin/amd64, linux/amd64, linux/arm64)

**Commands:**

```
clawfleet build                       # Build the Docker image
clawfleet create <N>                  # Create N claw instances
clawfleet list                        # List all instances and their status
clawfleet start <name|all>            # Start a stopped instance
clawfleet stop <name|all>             # Stop a running instance
clawfleet destroy <name|all>          # Destroy instance (data kept by default)
clawfleet destroy --purge <name|all>  # Destroy instance and delete its data
clawfleet desktop <name>              # Open an instance's desktop in the browser
clawfleet logs <name> [-f]            # View instance logs
```

**Config file:** `~/.clawfleet/config.yaml`

```yaml
image:
  name: "ghcr.io/clawfleet/clawfleet"
  tag: "latest"

ports:
  novnc_start: 6901      # noVNC base port, allocated sequentially
  gateway_start: 18789   # Gateway base port, allocated sequentially

resources:
  memory_limit: "4g"     # Per-container memory cap
  cpu_limit: "2.0"       # Per-container CPU cap (cores)

naming:
  prefix: "claw"      # Instance names: claw-1, claw-2, ...
```

### 3.2 Docker Image (ghcr.io/clawfleet/clawfleet)

**Base image:** `node:22-bookworm` — consistent with OpenClaw's official Docker setup.

**Layer design:**

```dockerfile
# Layer 1: System packages + desktop environment
FROM node:22-bookworm

RUN apt-get update && apt-get install -y \
    xfce4 xfce4-terminal \
    tigervnc-standalone-server \
    novnc websockify \
    dbus-x11 \
    chromium \
    fonts-noto-cjk \
    supervisor curl wget \
    && rm -rf /var/lib/apt/lists/*

# Wrap Chromium with --no-sandbox (required inside containers)
RUN mv /usr/bin/chromium /usr/bin/chromium-real \
    && printf '#!/bin/sh\nexec /usr/bin/chromium-real --no-sandbox "$@"\n' > /usr/bin/chromium \
    && chmod +x /usr/bin/chromium

# Layer 2: OpenClaw
RUN npm install -g openclaw@latest

# Layer 3: Playwright Chromium (installed at build time to avoid runtime downloads)
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
RUN node /usr/local/lib/node_modules/openclaw/node_modules/playwright-core/cli.js install chromium

# Layer 4: Startup configuration
COPY supervisord.conf /etc/supervisor/supervisord.conf
COPY entrypoint.sh /entrypoint.sh

EXPOSE 6901 18789
ENTRYPOINT ["/entrypoint.sh"]
```

**Process management (supervisord):**

| Process | Role | Port |
|---------|------|------|
| Xvnc | Virtual framebuffer + VNC server | DISPLAY :1 / 5901 (container-internal) |
| XFCE4 | Desktop environment | — |
| noVNC + websockify | VNC → WebSocket proxy | 6901 (host-mapped) |
| OpenClaw Gateway | Core AI service (manual start) | 18789 (host-mapped) |

**entrypoint.sh responsibilities:**
1. Initialize VNC password (from env var or use default)
2. Create `.vnc` and `.openclaw` directories, set ownership to `node`
3. Start supervisord

**Estimated image size:**
- Base system + XFCE + VNC/noVNC: ~800 MB
- Node.js 22 + OpenClaw: ~300 MB
- Playwright Chromium: ~300 MB
- **Total: ~1.4 GB**

### 3.3 Port Allocation

The CLI maintains a port allocator to avoid conflicts:

```
Instance   noVNC port   Gateway port
claw-1  6901         18789
claw-2  6902         18790
claw-3  6903         18791
...
claw-N  6900+N       18788+N
```

Allocation logic:
- Start from the configured base port, increment until a free port is found
- Probe availability via `net.Listen` before allocation
- Allocated ports are recorded in the state file

### 3.4 State Management

**State file:** `~/.clawfleet/state.json`

```json
{
  "instances": [
    {
      "name": "claw-1",
      "container_id": "abc123...",
      "status": "running",
      "ports": {
        "novnc": 6901,
        "gateway": 18789
      },
      "created_at": "2026-03-07T10:00:00Z"
    }
  ]
}
```

The state file is a metadata cache. Docker is the source of truth — the CLI reconciles against the Docker API on every operation.

### 3.5 Data Volumes

Each container gets its own bind-mounted host directory for persistence and isolation:

```
~/.clawfleet/data/
├── claw-1/
│   └── openclaw/     → /home/node/.openclaw inside container
├── claw-2/
│   └── openclaw/     → /home/node/.openclaw inside container
└── claw-3/
    └── openclaw/     → /home/node/.openclaw inside container
```

- Data is preserved by default when a container is destroyed
- `clawfleet destroy --purge` removes the data directory as well
- Directories are owned by uid 1000 (`node` user), matching the container's user

### 3.6 Network Design

**Docker network:**
- A dedicated bridge network `clawfleet-net` is created on first use
- Containers can reach each other by name (reserved for future inter-claw communication)
- Only the noVNC and Gateway ports are bound to the host

```
clawfleet-net (bridge)
├── claw-1 (172.20.0.2)
├── claw-2 (172.20.0.3)
└── claw-3 (172.20.0.4)
```

**Host access:**
- noVNC: `http://localhost:6901` → claw-1 desktop
- Gateway: accessible at `http://127.0.0.1:18789` from inside the container's Chromium (loopback — not directly exposed to the host user)

## 4. User Workflows

### 4.1 First-time setup

```bash
# 1. Build the Docker image (once, ~1.4 GB)
clawfleet build

# 2. Create 3 claws
clawfleet create 3
# Output:
#   Creating claw-1 ... ✓  desktop: http://localhost:6901
#   Creating claw-2 ... ✓  desktop: http://localhost:6902
#   Creating claw-3 ... ✓  desktop: http://localhost:6903

# 3. Open a claw's desktop and complete one-time onboarding
clawfleet desktop claw-1
```

### 4.2 Daily use

```bash
clawfleet list
# NAME        STATUS    DESKTOP              UPTIME
# claw-1   running   http://localhost:6901 2d 3h
# claw-2   running   http://localhost:6902 2d 3h
# claw-3   stopped   -                    -

clawfleet start claw-3
clawfleet stop claw-1
clawfleet logs claw-1
```

### 4.3 OpenClaw onboarding (inside the container)

After opening the noVNC desktop, open the terminal and run:

```bash
# Step 1: Run the setup wizard (configure LLM API key, Telegram bot, etc.)
openclaw onboard --flow quickstart

# Step 2: Start the Gateway
openclaw gateway --port 18789
```

Once the Gateway is running, open Chromium on the desktop and navigate to the URL printed in the terminal (e.g. `http://127.0.0.1:18789/#token=...`) to access the OpenClaw Control UI.

**Future automation (not in v1):** ClawSandbox CLI will automate onboarding — users will only need to supply an API key and Telegram bot token, with no manual desktop interaction required.

## 5. Resource Budget

Tested on M4 MacBook Air (16 GB RAM, 512 GB SSD):

| Resource | Per instance | 3 instances | 5 instances |
|----------|-------------|-------------|-------------|
| RAM (idle) | ~1.5 GB | ~4.5 GB | ~7.5 GB |
| RAM (Chromium active) | ~3 GB | ~9 GB | not recommended |
| Disk (image, shared) | 1.4 GB | 1.4 GB | 1.4 GB |
| Disk (data volume) | ~200 MB | ~600 MB | ~1 GB |
| CPU (idle) | <0.5 core | <1.5 cores | <2.5 cores |

**Recommendations:**
- 16 GB host: up to 3 active instances (with Chromium), or 5 light-load instances
- Default `memory_limit=4g` per container prevents a single runaway instance from affecting the host
- Adjust limits via `~/.clawfleet/config.yaml`

## 6. Repository Structure

```
ClawSandbox/
├── cmd/
│   └── clawsandbox/
│       └── main.go
├── internal/
│   ├── cli/
│   │   ├── root.go
│   │   ├── build.go
│   │   ├── create.go
│   │   ├── list.go
│   │   ├── start.go
│   │   ├── stop.go
│   │   ├── destroy.go
│   │   ├── desktop.go
│   │   ├── logs.go
│   │   └── helpers.go
│   ├── container/
│   │   ├── client.go
│   │   ├── manager.go
│   │   ├── image.go
│   │   └── network.go
│   ├── port/
│   │   └── allocator.go
│   ├── state/
│   │   └── store.go
│   ├── config/
│   │   └── config.go
│   └── assets/
│       ├── embed.go
│       └── docker/
│           ├── Dockerfile
│           ├── supervisord.conf
│           └── entrypoint.sh
├── docs/
│   ├── SYSTEM_DESIGN.md         # English (primary)
│   └── SYSTEM_DESIGN.zh-CN.md   # Chinese
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── README.zh-CN.md
└── CLAUDE.md
```

## 7. Dependencies

### Host (ClawSandbox CLI)
| Dependency | Purpose |
|------------|---------|
| Go 1.25+ | Compile the CLI |
| Docker Engine | Container runtime (Docker Desktop for Mac or colima) |

### Inside each container
| Dependency | Version | Purpose |
|------------|---------|---------|
| Debian Bookworm | 12 | Base OS |
| Node.js | 22 | OpenClaw runtime |
| OpenClaw | latest | AI assistant core |
| Chromium (Playwright) | — | Browser automation |
| XFCE4 | 4.18 | Lightweight desktop |
| TigerVNC | — | VNC server |
| noVNC + websockify | — | Browser-accessible VNC client |
| supervisord | — | Multi-process management inside container |

### Go modules
| Module | Purpose |
|--------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/fsouza/go-dockerclient` | Docker Engine API |
| `gopkg.in/yaml.v3` | Config file parsing |

## 8. UI Evolution Strategy

The project uses a phased delivery approach — validate the core before polishing the experience.

### Phase 1: CLI (current)
All operations via the command line. Goal: a fully functional, production-usable CLI tool.

### Phase 2: Web UI (before public release)
After the CLI is stable, add a Web UI for non-technical users:

- **Fleet management panel**: a web server on the host provides a unified overview page (create/start/stop/destroy instances, view status)
- **Instance configuration**: each claw gets its own settings page (LLM provider, Telegram bot, model selection) — no desktop terminal needed for onboarding
- **Desktop access**: embedded or linked noVNC view per instance

The CLI is always kept as the primary interface for advanced users and automation.

## 9. Scope

### Phase 1 MVP (CLI)

- [x] Docker image build (Dockerfile + supervisord + entrypoint)
- [x] `build` — build the Docker image
- [x] `create N` — create N container instances
- [x] `list` — list all instances with live status from Docker
- [x] `start` / `stop` — start and stop instances
- [x] `destroy [--purge]` — destroy instances, optionally purging data
- [x] `desktop` — open noVNC desktop in browser
- [x] `logs [-f]` — tail instance logs
- [x] Automatic port allocation with conflict detection
- [x] Per-instance data volume persistence

### Phase 2 (Web UI)
- [ ] Fleet management panel (web server)
- [ ] Per-instance configuration pages (LLM / Telegram / etc.)
- [ ] Automated OpenClaw onboarding

**Out of scope (future):**
- Inter-claw group chat and task delegation
- Instance templates / preset configurations
- Remote host support

## 10. Validation

### Build validation
```bash
make build
clawfleet build
```

### End-to-end validation
```bash
# 1. Create 2 instances
clawfleet create 2

# 2. Confirm containers are running
clawfleet list

# 3. Open desktop
clawfleet desktop claw-1
# → Browser opens http://localhost:6901 showing XFCE desktop

# 4. Complete onboarding inside the container
openclaw onboard --flow quickstart
openclaw gateway --port 18789
# → Open Chromium, navigate to printed URL, confirm Control UI loads

# 5. Stop / start (verify data persistence)
clawfleet stop claw-1
clawfleet start claw-1
# → ~/.openclaw data survives container restart

# 6. Destroy
clawfleet destroy claw-2
clawfleet list
# → claw-2 no longer listed
```

### Resource validation
```bash
docker stats claw-1 claw-2
# → Confirm memory stays within memory_limit
```

## 11. Version Management

A single `git tag` locks both the CLI binary and Docker image to the same version, with automated publishing to GitHub Release and GHCR.

### Architecture

```
git tag v0.1.0 && git push origin v0.1.0
        │
        ▼
   GitHub Actions
   ┌──────────────────┬──────────────────────┐
   │  GoReleaser       │  Docker Build+Push    │
   │  CLI binaries x4  │  ghcr.io image        │
   │  (darwin/linux    │  :0.1.0 + :latest     │
   │   x amd64/arm64) │                        │
   └────────┬─────────┴──────────┬─────────────┘
            ▼                    ▼
     GitHub Release         ghcr.io/clawfleet/clawfleet
```

### Shared version package (`internal/version/`)

Version metadata (`Version`, `GitCommit`, `BuildDate`) is injected at build time via ldflags into `internal/version`. Both `internal/cli/version.go` and `internal/config/config.go` import from this shared package.

The `ImageTag()` helper derives the Docker image tag from the CLI version:
- Release build (e.g. `v0.1.0`): returns `0.1.0`
- Dev build (no tag): returns `latest`

### Image naming and tagging

- **Registry**: `ghcr.io/clawfleet/clawfleet`
- **Default tag**: determined at runtime via `version.ImageTag()`
- `clawfleet build` tags the image with both `:<version>` and `:latest`
- `clawfleet create` uses `:<version>` to ensure CLI-image version consistency

### Auto-pull on create

When `clawfleet create` (or the Dashboard's create action) finds the image missing locally, it automatically attempts `docker pull` from GHCR before failing. Users with internet access never need to run `clawfleet build`.

```
image not found locally
  → docker pull ghcr.io/clawfleet/clawfleet:0.1.0
  → success? proceed with create
  → fail? suggest `clawfleet build`
```

### CI pipeline (`.github/workflows/release.yml`)

Triggered on `v*` tag push. Two parallel jobs:

| Job | Tool | Output |
|-----|------|--------|
| `release` | GoReleaser | 4 CLI binaries → GitHub Release |
| `docker` | docker/build-push-action | Image → `ghcr.io/clawfleet/clawfleet:<version>` + `:latest` |

### Release workflow (for maintainers)

```bash
git checkout main
git pull origin main
git tag v0.1.0
git push origin v0.1.0
# CI handles the rest
```
