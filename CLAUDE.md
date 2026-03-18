# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

ClawFleet — deploy and manage a fleet of isolated OpenClaw instances on a single machine, from a browser dashboard. Built on ClawSandbox, a purpose-built infrastructure layer for container orchestration and instance lifecycle management. Open-sourced on GitHub.

## Architecture Layers

ClawFleet is built on top of ClawSandbox, a purpose-built infrastructure layer
for container orchestration and instance lifecycle management.

### ClawSandbox (Infrastructure)
Packages: container/, state/, port/, config/, assets/, snapshot/, version/
Responsible for: Docker orchestration, instance state persistence, port allocation,
container networking, image management, snapshot archival.
Standard: production-grade reliability, defensive coding, thorough test coverage.

### ClawFleet (Product)
Packages: web/, cli/
Responsible for: REST API, WebSocket real-time updates, Dashboard UI, CLI commands,
asset management (models/channels/characters), skill management, i18n.
Standard: user experience, feature velocity, accessibility.

Dependency rule: ClawFleet → ClawSandbox (never reverse).

## Workflow

This is a multi-contributor project with rapid iteration. Before planning or starting any task in a session:

1. **Sync to latest main** — always pull the latest remote main branch first:
   ```bash
   git fetch origin
   git checkout main
   git pull origin main
   ```
2. **Build full context** — read the codebase, documentation (CLAUDE.md, README, SYSTEM_DESIGN, etc.), and project memory thoroughly. Understand recent changes by reviewing git log and any files touched by recent commits.
3. **Design from current state** — ensure all design decisions and implementation plans are based on a comprehensive understanding of the project's latest state, not assumptions from prior sessions.

Then create a feature branch from the up-to-date main. Never work directly on a stale branch.

## Build/Test/Lint Commands

```bash
# Download dependencies
go mod tidy

# Build CLI binary → bin/clawfleet
make build

# Run tests
make test

# Build Docker image (first time, ~1.4GB, takes several minutes)
make docker-build
# or via the CLI itself (uses embedded Dockerfile):
./bin/clawfleet build

# Cross-compile for all platforms
make build-all
```

## Architecture

Go CLI tool (cobra) that manages Docker containers with an embedded Web Dashboard. Key packages:

- `cmd/clawfleet/` — binary entry point
- `internal/cli/` — cobra commands (build, create, list, start, stop, restart, destroy, desktop, logs, dashboard, config, version)
- `internal/container/` — Docker SDK wrappers (client, image build/check/pull, network, container lifecycle, stats)
- `internal/port/` — TCP port availability checker and allocator
- `internal/state/` — instance metadata store (`~/.clawfleet/state.json`), mutex-protected
- `internal/config/` — config file loader (`~/.clawfleet/config.yaml`)
- `internal/assets/` — embedded Docker build context (Dockerfile, supervisord.conf, entrypoint.sh)
- `internal/web/` — Web Dashboard: HTTP server, REST API handlers, WebSocket endpoints (stats/logs/events), embedded frontend
- `internal/version/` — build version info (injected via ldflags)

Each claw instance is a Docker container running: XFCE4 desktop + TigerVNC + noVNC (browser access on port 690N) + OpenClaw Gateway (port 1878N).

Container data is persisted at `~/.clawfleet/data/<name>/openclaw/` → `/home/node/.openclaw` inside the container.

## OpenClaw Integration

ClawFleet manages OpenClaw instances via `docker exec` CLI commands. Key integration points:

### Character / SOUL.md
- OpenClaw uses `SOUL.md` (Markdown) at `~/.openclaw/SOUL.md` for character/persona definition
- Gateway watches this file — hot-reloads on change, no restart needed
- ClawFleet renders `CharacterAsset` fields into SOUL.md and writes via `docker exec`

### Skills
- **Bundled Skills** (52): Ship with OpenClaw, status depends on binary/env requirements
- **Managed Skills**: Installed via `clawhub` CLI to `~/.openclaw/skills/`
- `openclaw skills list --json` returns structured skill data
- `npx clawhub --workdir ~/.openclaw --dir skills install/uninstall <slug>` manages community skills
- ClawHub has rate limits (~20 requests/minute) — handle errors gracefully

### Useful CLI Commands (run as `node` user inside container)
- `openclaw skills list --json` — list all skills with status
- `openclaw plugins list` — list all plugins (41 stock plugins)
- `openclaw config set <path> <value>` — set any config value
- `npx clawhub search "<query>"` — search community skills
- `npx clawhub --workdir /home/node/.openclaw --dir skills install <slug> --no-input` — install skill

## Engineering Principles

All design decisions, project structure, and code implementation must follow best engineering practices. Specifically:

### Security & Least Privilege
- Never use overly permissive settings (e.g. `chmod 0777`) as shortcuts. Solve permission problems by understanding the ownership model and applying minimal necessary access.
- Container processes must run with the correct user for the operation: system management commands (e.g. `supervisorctl`) run as `root`, application commands (e.g. `openclaw`) run as the unprivileged `node` user.
- Never embed secrets or credentials in code, images, or config files.

### Correctness Over Convenience
- Understand the tools you're automating. Read `--help`, check actual behavior, and verify assumptions (e.g. model name formats, plugin enablement, API readiness) before writing integration code.
- Prefer explicit configuration over implicit defaults. If a third-party tool has a default that doesn't suit our use case (e.g. `dmPolicy: "pairing"`), set the desired value explicitly rather than hoping users will figure it out.
- When orchestrating multi-step processes, respect ordering dependencies and readiness checks (e.g. wait for a service to be healthy before issuing commands against it).

### Simplicity & Minimal Surface
- Don't add abstractions, flags, or config options for hypothetical future needs. Solve the current problem directly.
- Prefer calling existing CLI tools (`docker exec` + `openclaw` CLI) over writing config files directly — this keeps the integration resilient to upstream format changes.

### Verify Before Handoff
- After fixing a bug or implementing a feature that affects API/server behavior, smoke-test the change yourself (e.g. `curl` requests to the local server, `docker exec` commands) before asking the user to verify on the UI.

## Wiki Documentation

The project wiki lives in a separate repo (`git@github.com:weiyong1024/ClawFleet.wiki.git`) and is browsable at `https://github.com/weiyong1024/ClawFleet/wiki`. It is the primary documentation hub beyond the README.

### Wiki structure

| File | Purpose |
|------|---------|
| `_Sidebar.md` | Navigation sidebar shown on every page |
| `_Footer.md` | Footer with repo link |
| `Home.md` | Landing page — value prop, quickstart roadmap, page index |
| `Getting-Started.md` | Prerequisites, install, first instance end-to-end |
| `Dashboard-Guide.md` | Sidebar navigation, asset management, fleet management, image management |
| `CLI-Reference.md` | All CLI commands with descriptions and examples |
| `FAQ.md` | Common issues grouped by install / config / resource / operations |
| `Provider-{Anthropic,OpenAI,Google,DeepSeek}.md` | LLM provider guides (one per provider) |
| `Channel-{Telegram,Discord,Slack,Lark}.md` | Messaging channel guides (one per channel) |

### Content conventions

- **Provider pages** follow a consistent template: Overview → Get API Key (step-by-step) → Add in Dashboard → Assign to Instance → Pricing Notes → Troubleshooting.
- **Channel pages** follow: Overview → Create Bot (step-by-step) → Add in Dashboard → Assign to Instance → Test → Notes → Troubleshooting.
- Key facts to keep consistent across pages:
  - **Models are shared** — multiple instances can use the same model config simultaneously.
  - **Channels are exclusive** — each channel can only be assigned to one instance at a time.
  - **Validation required** — must click "Test" and see validation pass before saving.
  - **Lark is different** — uses App ID + App Secret, not a single token.
  - **Auto-configuration** — ClawFleet sets DM/group policies to "open" and allows all senders automatically.
- Preset models are defined in `internal/web/static/js/components/model-asset-dialog.js` (`MODEL_PRESETS`). When models change there, update the wiki provider pages and `Dashboard-Guide.md`.
- Validation logic is in `internal/web/validate.go`. If validation endpoints change, update the corresponding channel/provider troubleshooting sections.

### When to update the wiki

**You must update the wiki whenever a code change affects user-facing behavior.** Specifically:

- **New or removed CLI command** → update `CLI-Reference.md`, and `Getting-Started.md` / `FAQ.md` if relevant.
- **New LLM provider** → create `Provider-<Name>.md` using the existing template, add to `_Sidebar.md`, update `Home.md` page index and `Dashboard-Guide.md` preset table.
- **New messaging channel** → create `Channel-<Name>.md` using the existing template, add to `_Sidebar.md`, update `Home.md` page index.
- **Dashboard UI changes** (new sections, renamed labels, changed workflows) → update `Dashboard-Guide.md` and `Getting-Started.md`.
- **Model preset changes** → update `Dashboard-Guide.md` preset table and the relevant `Provider-*.md` page.
- **README wiki links** — both `README.md` and `README.zh-CN.md` have a "Documentation" section linking to wiki pages. Update these links if wiki page names change.

### How to edit the wiki

```bash
git clone git@github.com:weiyong1024/ClawFleet.wiki.git /tmp/ClawFleet.wiki
cd /tmp/ClawFleet.wiki
# Edit files...
git add . && git commit -m "describe the change" && git push
```

The wiki repo uses `master` as its default branch.
