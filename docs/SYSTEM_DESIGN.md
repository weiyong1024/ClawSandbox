# ClawFleet System Design

> Version: v1.2.0 | Date: 2026-04-21

[中文文档](./SYSTEM_DESIGN.zh-CN.md)

---

## 1. Overview

ClawFleet deploys and manages a fleet of isolated OpenClaw instances on a single machine, from a browser dashboard. Each instance runs in its own Docker container with a full Linux desktop (XFCE4 + TigerVNC + noVNC), accessible from any browser. Users manage their fleet — create instances, configure LLM providers, assign messaging channels, define character personas, and monitor resources — entirely through the web dashboard or CLI.

## 2. Architecture Layers

ClawFleet has two layers: the product layer (web dashboard, CLI, asset management) and the infrastructure layer (container orchestration, state persistence, port allocation).

```
┌─────────────────────────────────────────────────────────────┐
│                   Browser (Dashboard UI)                     │
│              Preact SPA @ http://localhost:8080              │
└──────────────────────────┬──────────────────────────────────┘
                           │ REST API + WebSocket
┌──────────────────────────▼──────────────────────────────────┐
│                  ClawFleet (Product Layer)                    │
│  internal/web/ + internal/cli/                               │
│  REST API, WebSocket streams, asset management, skills,      │
│  i18n, roster, snapshots, daemon management                  │
├─────────────────────────────────────────────────────────────┤
│                Infrastructure Layer                           │
│  internal/container/, /state/, /port/, /config/,             │
│  /assets/, /snapshot/, /version/                             │
│  Docker orchestration, state persistence, port allocation    │
└──────────────────────────┬──────────────────────────────────┘
                           │ Docker API (go-dockerclient)
┌──────────────────────────▼──────────────────────────────────┐
│                      Docker Engine                           │
│  ┌──────────┐  ┌──────────┐           ┌──────────┐          │
│  │ claw-1   │  │ claw-2   │    ...    │ claw-N   │          │
│  │ XFCE4    │  │ XFCE4    │           │ XFCE4    │          │
│  │ noVNC    │  │ noVNC    │           │ noVNC    │          │
│  │ OpenClaw │  │ OpenClaw │           │ OpenClaw │          │
│  │ Gateway  │  │ Gateway  │           │ Gateway  │          │
│  └──────────┘  └──────────┘           └──────────┘          │
│   :6901/:18789  :6902/:18790           :690N/:1878(8+N)     │
└─────────────────────────────────────────────────────────────┘
```

**Dependency rule:** Product layer → Infrastructure layer (never reverse).

## 3. Component Design

### 3.1 CLI

**Stack:** Go 1.25+, Cobra, go-dockerclient. Single statically-linked binary for darwin/linux × amd64/arm64.

**Commands:**

| Group | Command | Description |
|-------|---------|-------------|
| Fleet | `create <N>` | Create N instances |
| | `list` | List all instances with status |
| | `start <name\|all>` | Start stopped instance(s) |
| | `stop <name\|all>` | Stop running instance(s) |
| | `restart <name\|all>` | Restart instance(s) |
| | `destroy <name\|all> [--purge]` | Destroy instance(s), optionally delete data |
| | `desktop <name>` | Open noVNC desktop in browser |
| | `logs <name> [-f]` | View/tail container logs |
| | `configure <name>` | Interactive configuration wizard |
| Dashboard | `dashboard serve` | Start web server (foreground) |
| | `dashboard start [--host --port]` | Start as background daemon |
| | `dashboard stop` | Stop daemon |
| | `dashboard restart` | Restart daemon |
| | `dashboard status` | Check daemon status |
| Image | `build` | Build Docker image locally |
| Snapshot | `snapshot save <name>` | Archive instance soul |
| | `snapshot list` | List saved souls |
| | `snapshot delete <name>` | Delete saved soul |
| System | `config` | Show config file |
| | `version` | Print version info |

### 3.2 Web Dashboard

An embedded Preact SPA served by the Go HTTP server at port 8080.

**REST API (25+ endpoints):**

| Category | Endpoints | Purpose |
|----------|-----------|---------|
| Instances | `GET/POST /instances`, `POST /{name}/start\|stop`, `DELETE /{name}`, `POST /batch-destroy`, `POST /{name}/configure`, `GET /{name}/configure/status`, `POST /{name}/restart-bot`, `POST /{name}/reset` | Full instance lifecycle |
| Assets | `GET/POST/PUT/DELETE /assets/models`, `/assets/channels`, `/assets/characters`, `POST /assets/models/test`, `POST /assets/channels/test` | Model, channel, character CRUD with validation |
| OAuth | `POST /oauth/codex/start`, `POST /oauth/codex/callback`, `GET /oauth/codex/poll` | Codex OAuth flow (start → callback relay → poll for result) |
| Skills | `GET /{name}/skills`, `POST /{name}/skills/install`, `DELETE /{name}/skills/{slug}`, `GET /skills/search` | Skill management via ClawHub |
| Snapshots | `GET/POST/DELETE /snapshots` | Soul archival |
| Image | `GET /image/status`, `POST /image/build`, `POST /image/pull`, `GET /image/openclaw-versions` | Image lifecycle + OpenClaw version selector |

**WebSocket streams:**

| Endpoint | Purpose |
|----------|---------|
| `/ws/stats` | Real-time CPU/memory per instance |
| `/ws/logs/{name}` | Live container log stream |
| `/ws/events` | Lifecycle events (create, start, stop, etc.) |

**Control Panel:** For local access, clicking "Control Panel" opens the Gateway's native port directly (`http://localhost:{gateway_port}/`). Each instance runs on its own port → its own browser origin → localStorage is naturally isolated between tabs. This ensures multiple Control Panel tabs never share session state.

**Console proxy (remote fallback):** `/console/{name}/` reverse-proxies to the instance's Gateway UI via the LAN bridge. Used when the Dashboard is accessed remotely (e.g. via SSH tunnel) and Gateway ports aren't directly reachable. Requests without trailing slash are 301-redirected (except WebSocket upgrades, which bypass the redirect).

**Auto-pull on create:** When `POST /instances` or `clawfleet create` finds the expected image tag missing locally, it automatically pulls from GHCR before creating the instance. This removes friction when the binary is upgraded but the image hasn't been re-pulled. The CLI also supports `--pull` to force re-pull even when the image exists.

**Frontend components (21):** toolbar, sidebar, dashboard, instance-card, instance-desktop, create-dialog, configure-dialog, image-page, logs-viewer, model/channel/character asset pages and dialogs, skills, skill-manager-dialog, snapshots, snapshot-dialog, stats-chart, connection-status, toast.

**i18n:** English and Chinese, switchable from the toolbar.

### 3.3 Docker Image

**Registry:** `ghcr.io/clawfleet/clawfleet`

**Base image:** `node:22-bookworm`

**Layer design:**

| Layer | Content | Size |
|-------|---------|------|
| 1 | System packages: XFCE4, TigerVNC, noVNC, Chromium, CJK fonts, supervisord | ~800 MB |
| 2 | OpenClaw: `npm install -g openclaw@${OPENCLAW_VERSION}` + feishu extension | ~300 MB |
| 3 | Playwright Chromium: pre-installed at `/ms-playwright` | ~300 MB |
| 4 | Startup config: supervisord.conf + entrypoint.sh | <1 MB |
| **Total** | | **~1.4 GB** |

**Process management (supervisord):**

| Process | Role | User | Port | Autostart |
|---------|------|------|------|-----------|
| xvnc | VNC server + X11 framebuffer | node | 5901 (internal) | yes |
| xfce4 | Desktop environment | node | — | yes |
| novnc | VNC → WebSocket proxy | node | 6901 (host-mapped) | yes |
| openclaw | Gateway (started after configuration) | node | 18789 (internal) | conditional |
| gateway-bridge | TCP proxy 18789 → 18790 on 0.0.0.0 | node | 18790 (host-mapped) | conditional |

**entrypoint.sh:** Creates `.vnc` and `.openclaw` directories, sets VNC password if `$VNC_PASSWORD` is provided, auto-starts OpenClaw if `.configured` marker exists, then launches supervisord.

### 3.4 Multi-Runtime Support (OpenClaw + Hermes)

ClawFleet supports two AI agent runtimes: **OpenClaw** and **Hermes Agent**. They use fundamentally different image strategies.

**Image strategy comparison:**

| | OpenClaw | Hermes Agent |
|---|---|---|
| **Image source** | Custom-built by ClawFleet (`ghcr.io/clawfleet/clawfleet`) | Official from Nous Research (`nousresearch/hermes-agent`) |
| **Base** | `node:22-bookworm` + XFCE/noVNC/supervisord/Chromium | Debian 13 + Python 3.13 + uv |
| **Version control** | ClawFleet pins `RecommendedOpenClawVersion` per release | Follows official `latest` tag |
| **CI build** | Every ClawFleet release triggers image build | Not built by ClawFleet |
| **Why this model** | OpenClaw is an npm package with no official Docker image — we add desktop, process management, gateway bridge, and auto-recovery | Hermes ships a complete Docker image with dashboard + gateway + CLI |

**Why OpenClaw needs a custom image:**

OpenClaw distributes as `npm install -g openclaw` only. Our image adds components that OpenClaw itself doesn't provide:

| Component | Purpose | Without it |
|-----------|---------|-----------|
| XFCE4 + TigerVNC + noVNC | Browser-accessible desktop (Desktop button) | No desktop access |
| supervisord | Multi-process management (VNC + Gateway + Bridge) | Can only run one process per container |
| gateway-bridge | Exposes Gateway on 0.0.0.0:18790 for Docker port mapping | Control Panel unreachable from host |
| entrypoint.sh + `.configured` marker | Auto-recovery after container restart | Manual reconfigure on every restart |
| Chromium + Playwright | Browser-based skills (web_search, screenshots) | Skills that need a browser fail |
| CJK fonts | Chinese/Japanese/Korean text rendering | Garbled text in desktop/screenshots |

**Why Hermes uses the official image:**

Hermes's official Docker image already includes everything needed: dashboard server, gateway, CLI, skill sync, and config bootstrap. No additional components needed.

**Runtime detection:**

Each `Instance` has a `RuntimeType` field (`"openclaw"` or `"hermes"`). This determines:
- Which image to pull/use at creation time
- Port mapping (OpenClaw: noVNC 6901 + Gateway 18790; Hermes: Dashboard 9119 + Gateway 3000)
- Volume mount (OpenClaw: `/home/node/.openclaw`; Hermes: `/opt/data`)
- Container startup command (OpenClaw: supervisord; Hermes: custom entrypoint running dashboard + gateway)
- Which Dashboard buttons are shown (OpenClaw: Desktop/Control Panel/Configure/Skills; Hermes: Dashboard only)
- Which API operations are allowed (Configure, Skills, Restart Bot are OpenClaw-only)

**Hermes container startup:**

The official Hermes entrypoint does UID/GID remapping + setup, then `exec hermes "$@"`. ClawFleet overrides the entrypoint to run both dashboard and gateway simultaneously:

```bash
hermes dashboard --host 0.0.0.0 --port 9119 --no-open --insecure &
exec hermes gateway run
```

**CLI shell access:**

`clawfleet shell <name>` provides interactive terminal access:
- Hermes instances: launches the Hermes TUI (interactive chat)
- OpenClaw instances: opens a bash shell as the node user

### 3.5 Asset Management

Assets are shared resources that can be assigned to instances.

**Model assets:** LLM provider configuration. Supports ChatGPT (Codex) via OAuth, plus Anthropic, OpenAI, Google AI Studio, DeepSeek via API keys. API keys are validated before saving via provider-specific test endpoints. **Models are shared** — multiple instances can use the same model simultaneously.

**Channel assets:** Messaging platform configuration (Telegram bot token, Discord bot token, Slack webhook, Lark App ID + Secret). Credentials are validated before saving. **Channels are exclusive** — each channel can only be assigned to one instance at a time.

**Character assets:** Persona definition (name, role, personality, backstory, quirks, constraints). Rendered into `SOUL.md` Markdown and written to the instance's `~/.openclaw/SOUL.md`. The Gateway hot-reloads this file on change.

### 3.6 Codex OAuth (ChatGPT Subscription Login)

Users with a ChatGPT Plus/Pro subscription can authenticate via OAuth instead of API keys. This is the recommended and default provider.

**Protocol:** OAuth 2.0 Authorization Code + PKCE, using OpenClaw's registered client ID with OpenAI.

**Architecture: Stateless :1455 Relay**

The OAuth callback URI is hardcoded to `http://localhost:1455/auth/callback` (registered by OpenClaw with OpenAI, immutable). ClawFleet runs a stateless relay server on port 1455 to handle callbacks.

```
Dashboard (:8080 or :8081 via tunnel)       :1455 Relay (stateless)
┌─────────────────────────┐                 ┌──────────────────────┐
│ POST /oauth/codex/start │                 │ GET /auth/callback   │
│  → generate PKCE        │                 │  → serve static HTML │
│  → store verifier       │                 │  → JS reads code +   │
│  → return auth_url with │                 │    state from URL    │
│    state=<nonce>.<origin>│                 │  → JS forwards to   │
│                         │                 │    <origin>/callback │
│ POST /oauth/codex/callback               │                      │
│  → exchange code+verifier│◄───── fetch ───│                      │
│  → store tokens as asset │                │                      │
│                         │                 │                      │
│ GET /oauth/codex/poll   │                 │                      │
│  → return result to UI  │                 │                      │
└─────────────────────────┘                 └──────────────────────┘
```

**Key design: the relay is stateless, the Dashboard is stateful.** The :1455 relay serves a single HTML page that reads `code` and `state` from the URL, parses the Dashboard origin from state (`<nonce>.<origin>`), and forwards the code via `fetch()` to that Dashboard's `/api/v1/oauth/codex/callback`. The relay holds no state.

The Dashboard API maintains an in-memory map of pending OAuth flows (keyed by nonce), each storing the PKCE verifier, selected model, and a 5-minute TTL. When the callback arrives, the Dashboard looks up the verifier, exchanges the code for tokens, and creates the model asset. Flows are cleaned up after poll or timeout.

**Token security:** When returning model assets to the frontend, the Dashboard strips `OAuthRefresh` (secret) but preserves `OAuthAccountID` (opaque hint for UI display). Refresh tokens never leave the backend.

**Multi-Dashboard coexistence:** Because the relay is stateless and the Dashboard origin is encoded in the state parameter, a single :1455 listener correctly routes callbacks to any Dashboard — local (:8080) or remote (:8081 via SSH tunnel).

```
Scenario: local Dashboard (:8080) + remote Dashboard (:8081 via SSH tunnel)

Local Dashboard starts :1455 relay on boot.
SSH tunnel tries to bind :1455 → fails (already in use) → harmless warning.

User logs in from :8080 → state="abc.http://localhost:8080"
  → callback hits local :1455 → relay forwards to :8080 ✓

User logs in from :8081 → state="def.http://localhost:8081"
  → callback hits local :1455 → relay forwards to :8081 (→ tunnel → remote) ✓

If no local Dashboard is running:
  SSH tunnel binds :1455 → remote Dashboard's relay handles callbacks ✓
```

**Container configuration for Codex:** Unlike API key providers that use `openclaw onboard --<provider>-api-key`, Codex uses:
1. `openclaw onboard --auth-choice skip` (creates workspace without auth)
2. Direct write to `auth-profiles.json` with OAuth tokens (access, refresh, expires, accountId)
3. `openclaw models set openai-codex/<model>`

OpenClaw handles token refresh internally at runtime using the stored refresh token.

### 3.7 Instance Configuration

When a user clicks "Configure" on an instance in the Dashboard, the system applies a multi-step configuration sequence via `docker exec`:

1. Set model provider and API key (`openclaw config set`)
2. Set model name
3. Set DM and group policies to "open" and allow all senders
4. Write channel configuration
5. Render and write `SOUL.md` (character + roster)
6. Write `.configured` marker
7. Start/restart the OpenClaw Gateway process

Configuration status is tracked and reported to the frontend in real-time.

**Provider name mapping:** ClawFleet presents unified provider names in the UI but maps them for OpenClaw's onboard CLI:
- `google` → `--gemini-api-key` (OpenClaw uses "gemini" not "google")
- All others (`anthropic`, `openai`, `deepseek`) map directly

**Per-channel policy configuration:** OpenClaw channel plugins have different config schemas. ClawFleet normalizes them:
- All channels: `allowFrom=["*"]`, `dmPolicy/groupPolicy="open"`
- Discord/Lark: `allowBots="mentions"` (text-based mention detection)
- Slack: `allowBots=true` (boolean, not string — schema difference)
- Telegram: additional `groupAllowFrom=["*"]`

**Bot name resolution:** For Discord and Slack, ClawFleet resolves the bot's display name from the platform API at configuration time and injects it into the agent identity config. This enables text @mention detection. Lark/Feishu uses native platform mentions, so resolution is skipped.

**Gateway health synchronization:** After each `supervisorctl start` during configuration, ClawFleet polls the Gateway's `/health` endpoint every 1 second until it responds or 30 seconds elapse. This ensures subsequent steps operate on a ready Gateway.

**Instance reset:** `POST /instances/{name}/reset` purges the instance's OpenClaw configuration (`openclaw.json`, `agents/`, `sessions/`, `channels/`, `.configured` marker) while preserving the Docker container. The container is restarted to clear Node.js V8 caches. Reset releases any assigned channel asset (allowing reassignment) and triggers roster refresh for other running instances.

### 3.8 Roster System

The Roster enables bot-to-bot collaboration by injecting team metadata into each instance's `SOUL.md`. Each bot knows who is on the team, what their role is, and when to @mention them.

**Rendering:** When configuring an instance, ClawFleet collects all configured instances' character data, builds a `## Your Team` section with each teammate's name, role, channel, and a one-line description, then appends it to SOUL.md.

**Design principles (prompt-as-code):**
- Explicit judgment criteria: when to @mention a teammate
- Negative constraints: when NOT to mention (e.g., don't mention yourself)
- Dense, scannable format: one line per teammate, not full lore dumps

**Roster synchronization:** SOUL.md is refreshed in all other *running* instances when any of these events occur:
1. An instance is configured (new teammate joins the fleet)
2. An instance is destroyed or reset (teammate leaves)
3. An instance is started (catches up with fleet changes during downtime)

Refresh is best-effort — errors are logged but do not fail the primary operation. Stopped instances do not receive roster updates until restarted.

**SOUL.md path:** Character data is written to `/home/node/.openclaw/workspace/SOUL.md` (the workspace directory). The Gateway watches this file and hot-reloads character data without restart.

**Batch destroy:** `POST /instances/batch-destroy` accepts a list of instance names and destroys them in a single state load/save cycle. Individual failures do not block others (partial success). Roster refresh is triggered once after all deletions complete.

### 3.9 Skill Management

- **Bundled skills (52):** Ship with OpenClaw. Status depends on binary/environment requirements.
- **Managed skills:** Installed via `npx clawhub` to `~/.openclaw/skills/`.
- The Dashboard provides search (via ClawHub API), install, and uninstall operations.
- ClawHub has rate limits (~20 requests/minute) — errors are handled gracefully.

### 3.10 Snapshot System (Soul Archival)

Snapshots capture an instance's OpenClaw data directory for later reuse:

- **Save:** Copies `~/.clawfleet/data/<name>/openclaw/` to `~/.clawfleet/snapshots/<id>/`, stripping `channels/` and `sessions/` (sensitive/ephemeral data).
- **Load:** A snapshot can be restored into a new instance.
- **Metadata:** Name, source instance, creation timestamp stored in `state.json`.

### 3.11 Port Allocation

Sequential allocation from configured base ports:

```
Instance   noVNC    Gateway (internal)   Gateway LAN bridge
claw-1     6901     18789                18789+1=18790 (→ 0.0.0.0)
claw-2     6902     18790                18791
claw-N     6900+N   18788+N              18789+N
```

Ports are probed via `net.Listen` before allocation to avoid conflicts.

### 3.12 State Management

**State file:** `~/.clawfleet/state.json` — metadata cache for instances, assets, and snapshots. Docker is the source of truth for container status; the CLI reconciles on every operation.

```json
{
  "instances": [{
    "name": "claw-1",
    "container_id": "abc123...",
    "status": "running",
    "ports": { "novnc": 6901, "gateway": 18789 },
    "created_at": "2026-03-30T10:00:00Z",
    "model_asset_id": "anthropic-1",
    "channel_asset_id": "telegram-1",
    "character_asset_id": "alice-1"
  }],
  "model_assets": [...],
  "channel_assets": [...],
  "character_assets": [...],
  "snapshots": [...]
}
```

### 3.13 Data Volumes

```
~/.clawfleet/
├── config.yaml              # User configuration
├── state.json               # Instance + asset metadata
├── serve.pid                # Dashboard daemon PID
├── logs/                    # Dashboard logs
├── data/                    # Per-instance data
│   ├── claw-1/
│   │   └── openclaw/        → /home/node/.openclaw in container
│   │       ├── SOUL.md      # Character prompt
│   │       ├── openclaw.json
│   │       ├── skills/
│   │       ├── knowledge/
│   │       └── sessions/
│   └── claw-N/
└── snapshots/               # Saved souls
    └── <id>/
        └── openclaw/
```

Data persists across container restarts. `clawfleet destroy --purge` removes it.

### 3.14 Network Design

- Bridge network `clawfleet-net` created on first use
- Containers can reach each other by name (used for inter-instance communication)
- noVNC port bound to host for desktop access
- Gateway LAN bridge port (`18790`) bound to `0.0.0.0` for console proxy access

## 4. Installation & Deployment

### 4.1 One-Line Install

```bash
curl -fsSL https://clawfleet.io/install.sh | sh
```

**What it does:**
1. Detects OS (macOS/Linux) and architecture (amd64/arm64)
2. Ensures Docker is installed (Colima on macOS, Docker Engine on Linux)
3. Downloads the latest CLI binary from GitHub Releases (with checksum verification)
4. Pulls the pre-built Docker image from GHCR (~1.4 GB)
5. Starts the Dashboard as a background daemon
6. Opens the browser to `http://localhost:8080`

**Options:** `--version <tag>`, `--skip-pull`, `--no-daemon`

### 4.2 Daemon Management

The Dashboard runs as a background daemon, managed per platform:

| Platform | Manager | Mechanism |
|----------|---------|-----------|
| macOS | launchd | `~/Library/LaunchAgents/com.clawfleet.dashboard.plist` |
| Linux (non-root) | systemd user service | `~/.config/systemd/user/clawfleet-dashboard.service` |
| Linux (root) | systemd system service | `/etc/systemd/system/clawfleet-dashboard.service` |
| Fallback | PID file | `~/.clawfleet/serve.pid` |

**Default bind address:** `127.0.0.1` on macOS (local only), `0.0.0.0` on Linux (remote access).

## 5. Version Management

### 5.1 ClawFleet Version

A single `git tag` locks both the CLI binary and Docker image to the same version.

```
git tag v0.4.0 && git push origin v0.4.0
        │
        ▼
   GitHub Actions (release.yml)
   ┌──────────────────────┬────────────────────────────────┐
   │  release job          │  docker job                     │
   │  GoReleaser           │  docker/build-push-action       │
   │  CLI binaries × 4     │  ghcr.io image (multi-arch)     │
   │  (darwin/linux         │  :v0.4.0 + :latest             │
   │   × amd64/arm64)      │                                 │
   └──────────┬────────────┴───────────────┬────────────────┘
              ▼                            ▼
       GitHub Release              ghcr.io/clawfleet/clawfleet
```

**Version package (`internal/version/`):** `Version`, `GitCommit`, `BuildDate` injected via ldflags. `ImageTag()` derives the Docker image tag — release builds (e.g., `v0.4.0`) use the version tag, dev builds fall back to `latest`.

### 5.2 OpenClaw Version Locking

The OpenClaw version inside the Docker image is controlled, not left to npm `@latest` at build time.

**Single source of truth:** `internal/version/version.go`

```go
const RecommendedOpenClawVersion = "2026.4.1"
```

**How it flows through the system:**

```
version.go: RecommendedOpenClawVersion = "2026.3.23-2"
        │
        ├──→ CI (release.yml)
        │    Extracted via: grep 'RecommendedOpenClawVersion =' version.go
        │    Passed as: OPENCLAW_VERSION build-arg to Docker build
        │    Result: Pre-built GHCR image contains openclaw@2026.4.1
        │
        ├──→ Dashboard → Build (local)
        │    Version selector defaults to RecommendedOpenClawVersion
        │    User can override to any version from npm registry
        │
        └──→ Dashboard → Pull
             Pulls the pre-built GHCR image (version already baked in by CI)
```

**User experience by path:**

| Path | OpenClaw Version | Determined By |
|------|-----------------|---------------|
| `install.sh` (one-line install) | `RecommendedOpenClawVersion` | CI build-arg ← `version.go` |
| Dashboard → Pull | Same as above | Same pre-built image |
| Dashboard → Build (local) | User's choice (default: recommended) | Version selector in UI |

**Upgrade workflow:** When a new OpenClaw version is tested and validated, update `RecommendedOpenClawVersion` in `version.go`, cut a new ClawFleet release. The next `install.sh` run or Dashboard Pull will deliver the new version.

### 5.3 Image Naming and Tagging

- **Registry:** `ghcr.io/clawfleet/clawfleet`
- **Tags:** `:<version>` (e.g., `:v0.4.0`) + `:latest`
- **Default tag at runtime:** Determined by `version.ImageTag()` — release builds use the version tag, dev builds use `latest`

### 5.4 Auto-Pull on Create

When `clawfleet create` or the Dashboard's create API finds the image missing locally, it automatically attempts `docker pull` from GHCR before failing.

## 6. Resource Budget

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
- Adjust via `~/.clawfleet/config.yaml`

## 7. Repository Structure

```
ClawFleet/
├── cmd/clawfleet/              # Binary entry point
│   └── main.go
├── internal/
│   ├── cli/                    # Cobra commands (24 files)
│   │   ├── root.go             # Root command, registers subcommands
│   │   ├── create.go           # Instance creation
│   │   ├── list.go             # Fleet listing
│   │   ├── start/stop/restart/destroy.go
│   │   ├── desktop.go          # Open noVNC in browser
│   │   ├── logs.go             # Container log viewer
│   │   ├── configure.go        # Interactive configuration wizard
│   │   ├── dashboard*.go       # Dashboard serve/start/stop/restart/status
│   │   ├── daemon*.go          # Platform-specific daemon management
│   │   ├── snapshot*.go        # Snapshot save/list/delete
│   │   ├── build.go            # Image build command
│   │   ├── config_show.go      # Show config file
│   │   └── version.go          # Version display
│   ├── container/              # Docker orchestration (8 files)
│   │   ├── client.go           # Docker client init
│   │   ├── manager.go          # Container lifecycle
│   │   ├── image.go            # Image build/pull/check/tag
│   │   ├── configure.go        # Multi-step OpenClaw configuration
│   │   ├── network.go          # Docker network management
│   │   ├── skills.go           # Skill install/uninstall
│   │   └── stats.go            # Resource stats collection
│   ├── port/                   # Port allocator
│   │   └── allocator.go
│   ├── state/                  # JSON state persistence
│   │   ├── store.go            # Instance metadata
│   │   ├── assets.go           # Model/channel/character assets
│   │   └── snapshots.go        # Snapshot metadata
│   ├── config/                 # YAML config loader
│   │   └── config.go
│   ├── assets/                 # Embedded Docker build context
│   │   ├── embed.go
│   │   └── docker/
│   │       ├── Dockerfile
│   │       ├── supervisord.conf
│   │       └── entrypoint.sh
│   ├── snapshot/               # Soul archival logic
│   │   └── snapshot.go
│   ├── version/                # Build version info
│   │   └── version.go          # Version + RecommendedOpenClawVersion
│   └── web/                    # Web Dashboard (15+ files)
│       ├── server.go           # HTTP server + PID management
│       ├── routes.go           # Route registration
│       ├── embed.go            # Frontend asset embedding
│       ├── handlers.go         # Instance lifecycle handlers
│       ├── handlers_assets.go  # Asset CRUD
│       ├── handlers_configure.go  # Configuration endpoint
│       ├── handlers_image.go   # Image build/pull/versions
│       ├── handlers_skills.go  # Skill management
│       ├── handlers_snapshots.go  # Snapshot CRUD
│       ├── handlers_console.go # Console proxy (reverse proxy to noVNC)
│       ├── events.go           # Event bus for real-time updates
│       ├── ws_stats.go         # WebSocket: resource stats
│       ├── ws_logs.go          # WebSocket: container logs
│       ├── ws_events.go        # WebSocket: lifecycle events
│       ├── oauth_codex.go      # Codex OAuth flow + :1455 relay server
│       ├── validate.go         # LLM/channel credential validation
│       └── static/             # Embedded frontend
│           ├── index.html
│           ├── css/style.css
│           └── js/
│               ├── app.js      # Main Preact app
│               ├── api.js      # REST client
│               ├── ws.js       # WebSocket manager
│               ├── i18n.js     # Internationalization
│               └── components/ # 21 Preact components
├── scripts/
│   ├── install.sh              # One-liner install script
│   └── ensure-go.sh            # Go version bootstrap
├── docs/
│   ├── SYSTEM_DESIGN.md
│   ├── SYSTEM_DESIGN.zh-CN.md
│   └── images/                 # Screenshots
├── growth/                     # Marketing materials
├── .github/workflows/
│   └── release.yml             # CI/CD pipeline
├── .goreleaser.yml             # Multi-platform release config
├── Makefile                    # Build targets
├── CLAUDE.md                   # AI assistant guide
├── ROADMAP.md                  # Product roadmap
├── README.md / README.zh-CN.md
└── LICENSE                     # MIT
```

## 8. Dependencies

### Host
| Dependency | Purpose |
|------------|---------|
| Go 1.25+ | Compile the CLI |
| Docker Engine | Container runtime (Colima on macOS, Docker Engine on Linux) |

### Inside each container
| Dependency | Version | Purpose |
|------------|---------|---------|
| Debian Bookworm | 12 | Base OS |
| Node.js | 22 | OpenClaw runtime |
| OpenClaw | Locked per release | AI assistant core |
| Chromium (Playwright) | — | Browser automation |
| XFCE4 | 4.18 | Lightweight desktop |
| TigerVNC | — | VNC server |
| noVNC + websockify | — | Browser-accessible VNC client |
| supervisord | — | Multi-process management |

### Go modules
| Module | Purpose |
|--------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/fsouza/go-dockerclient` | Docker Engine API |
| `github.com/gorilla/websocket` | WebSocket support |
| `gopkg.in/yaml.v3` | Config file parsing |

## 9. CI/CD Pipeline

**Trigger:** Push tag matching `v*` (e.g., `v0.4.0`)

**Jobs (parallel):**

| Job | Tool | Output |
|-----|------|--------|
| `release` | GoReleaser | 4 CLI binaries (darwin/linux × amd64/arm64) → GitHub Release with checksums |
| `docker` | docker/build-push-action | Multi-arch image (linux/amd64 + linux/arm64) → GHCR with version tag + `:latest` |

The `docker` job extracts `RecommendedOpenClawVersion` from `internal/version/version.go` and passes it as the `OPENCLAW_VERSION` build-arg, ensuring the pre-built image contains the tested OpenClaw version.

**Release workflow:**

```bash
# 1. Update RecommendedOpenClawVersion if needed
# 2. Tag and push
git tag v0.5.0
git push origin v0.5.0
# CI handles: binary builds, image build+push, GitHub Release creation
```

## 10. Configuration

**File:** `~/.clawfleet/config.yaml`

```yaml
image:
  name: "ghcr.io/clawfleet/clawfleet"
  tag: "v0.4.0"             # Determined by version.ImageTag()

ports:
  novnc_start: 6901         # Sequential: 6901, 6902, ...
  gateway_start: 18789      # Sequential: 18789, 18790, ...

resources:
  memory_limit: "4g"        # Per-container
  cpu_limit: 2.0            # Per-container (cores)

naming:
  prefix: "claw"            # Instance names: claw-1, claw-2, ...
```

## 11. Validation

### End-to-end (one-line install)
```bash
# Fresh machine
curl -fsSL https://clawfleet.io/install.sh | sh
# → Docker installed, CLI downloaded, image pulled, Dashboard running at :8080

# Verify OpenClaw version inside the image
docker exec claw-1 npm list -g openclaw
# → Should show RecommendedOpenClawVersion
```

### Build validation
```bash
make build && make test
```

### Manual lifecycle
```bash
clawfleet create 2
clawfleet list
clawfleet stop claw-1
clawfleet start claw-1
# → Data persists across restarts
clawfleet destroy claw-2
```

### Resource validation
```bash
docker stats claw-1 claw-2
# → Memory within memory_limit
```
