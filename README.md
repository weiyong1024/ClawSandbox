# ClawSandbox

> Turn one OpenClaw into an isolated AI crew.

[中文文档](./README.zh-CN.md)

---

**Run multiple OpenClaw instances on one machine, each with its own identity, credentials, browser session, data, and blast radius.**

Built for the moment when one assistant becomes a team:

- **One claw, one identity** — separate Telegram / WhatsApp / Slack accounts
- **One claw, one boundary** — isolated container, filesystem, browser, ports, and runtime state
- **One claw, one budget** — different models, prompts, API keys, skills, or experiments
- **One stable, one experimental** — keep production running while you test the next claw

No extra Mac Mini. No default cloud bill. Just the machine you already own.

## Why Not Just One OpenClaw?

OpenClaw already supports multiple agents and even multiple Gateways on the same host. ClawSandbox exists for the next boundary: **runtime isolation**.

Start with one OpenClaw if you only need one assistant. Reach for ClawSandbox when you need:

- different bot or channel accounts
- different credentials or model providers
- different browser sessions or long-running automation state
- separate "prod", "staging", or "backup" claws

## What ClawSandbox Does

- **One-command fleet deployment** — give it a number, get that many isolated OpenClaw instances
- **Web Dashboard** — manage your entire fleet from a browser with real-time stats, one-click actions, and embedded noVNC desktops
- **Full desktop per instance** — each claw runs in its own Docker container with an XFCE desktop, accessible via noVNC
- **Lifecycle management** — create, start, stop, restart, and destroy instances via CLI or Dashboard
- **Data persistence** — each instance's data survives container restarts
- **Resource isolation** — instances are isolated from your host system and from each other

## What You Need

- **macOS or Linux** — Apple Silicon Macs are a good fit for local use
- **Docker installed and running** — for most users this means [Docker Desktop](https://www.docker.com/products/docker-desktop/) must already show the engine as running
- **A local Docker context that works from the terminal** — `docker version` should succeed before you continue
- **Go 1.25+ and `make`** — only needed if you are building ClawSandbox from source, as shown below
- **Enough local resources** — at least 8 GB RAM and 10+ GB free disk; 16 GB RAM is recommended if you want to run multiple claws comfortably
- **Internet on first run** — a fresh local image build downloads base layers, packages, and browser assets

## First-Run Expectation

The recommended first-run path today is:

- **Build the local image first** — run `clawsandbox build` before you create anything
- **Then start the Dashboard or use the CLI** — once the image is local, instance creation takes seconds
- **Expect the first local build to take time** — on a fresh machine, several minutes is normal

If Docker is not running, ClawSandbox will fail early. Start Docker first, then continue.

## Quick Start

### 1. Build the CLI

```bash
git clone https://github.com/weiyong1024/ClawSandbox.git
cd ClawSandbox
make build
# Optionally install to PATH (otherwise use ./bin/clawsandbox in place of clawsandbox below):
sudo make install
```

### 2. Make sure Docker is ready

Before going further, confirm Docker works from your terminal:

```bash
docker version
```

If that command fails, open Docker Desktop and wait until the engine is running.

### 3. Run a quick preflight

This gives you a plain-English answer before you wait on a build:

```bash
clawsandbox doctor
```

It tells you:

- whether Docker is reachable
- whether the local image already exists
- what startup path you are on
- what to do next

### 4. Build the local Docker image

Build the local image before your first create (~4 GB image, takes several minutes on a fresh machine):

```bash
clawsandbox build
```

### 5. Deploy your fleet

**Option A: Web Dashboard (recommended)**

```bash
# Start the Dashboard
clawsandbox dashboard serve
```

Open [http://localhost:8080](http://localhost:8080) in your browser. Click **"Create Instances"**, choose a count, and you're done.

![Dashboard](docs/images/dashboard.jpeg)

The Dashboard provides:
- Real-time CPU/memory stats for every instance
- One-click Start / Stop / Destroy actions
- Click **"Desktop"** on any running instance to open its detail page with an embedded noVNC desktop, live logs, and resource charts

![Instance Desktop](docs/images/instance-desktop.jpeg)

**Option B: CLI**

```bash
# Create 3 isolated OpenClaw instances
clawsandbox create 3

# Check status
clawsandbox list
```

### 6. Set up each claw

Each claw needs a one-time configuration via its desktop. Open it from the Dashboard (click **"Desktop"** on an instance card) or via CLI:

```bash
clawsandbox desktop claw-1
```

Inside the desktop terminal:

```bash
# Step 1: Run the setup wizard (configure LLM API key, Telegram bot, etc.)
openclaw onboard --flow quickstart

# Step 2: Start the Gateway
openclaw gateway --port 18789
```

Once the Gateway is running, open **Chromium** on the desktop and navigate to the URL shown in the terminal (e.g. `http://127.0.0.1:18789/#token=...`) to access the OpenClaw Control UI.

## CLI Reference

Every command supports `--help` for detailed usage and examples:

```bash
clawsandbox --help              # List all available commands
clawsandbox dashboard --help    # Show dashboard subcommands
```

Quick reference:

```bash
clawsandbox doctor                      # Run preflight checks and get the next step
clawsandbox create <N>                  # Create N claw instances (run `clawsandbox build` first on a new machine)
clawsandbox list                        # List all instances and their status
clawsandbox desktop <name>              # Open an instance's desktop in the browser
clawsandbox start <name|all>            # Start a stopped instance
clawsandbox stop <name|all>             # Stop a running instance
clawsandbox restart <name|all>          # Restart an instance (stop + start)
clawsandbox logs <name> [-f]            # View instance logs
clawsandbox destroy <name|all>          # Destroy instance (data kept by default)
clawsandbox destroy --purge <name|all>  # Destroy instance and delete its data
clawsandbox dashboard serve              # Start the Web Dashboard
clawsandbox dashboard stop               # Stop the Web Dashboard
clawsandbox dashboard restart            # Restart the Web Dashboard
clawsandbox dashboard open               # Open the Dashboard in your browser
clawsandbox build                        # Build the local image for first run, offline use, or customization
clawsandbox config                       # Show current configuration
clawsandbox version                      # Print version info
```

## Reset

To destroy all instances (including data), stop the Dashboard, and remove all build artifacts — effectively returning to a clean slate:

```bash
make reset
```

After resetting, start over from [Quick Start](#quick-start) step 1.

## Resource Usage

Tested on M4 MacBook Air (16 GB RAM):

| Instances | RAM (idle) | RAM (Chromium active) |
|-----------|------------|-----------------------|
| 1         | ~1.5 GB    | ~3 GB                 |
| 3         | ~4.5 GB    | ~9 GB                 |
| 5         | ~7.5 GB    | not recommended       |

## Project Status

Actively developed. Both CLI and Web Dashboard are functional.

Contributions and feedback welcome — please open an issue or PR.
Security policy: [SECURITY.md](./SECURITY.md). Contribution guide: [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

MIT
