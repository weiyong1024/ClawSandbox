# ClawSandbox

> Deploy and manage a fleet of isolated [OpenClaw](https://github.com/openclaw/openclaw) instances on a single machine.

[中文文档](./README.zh-CN.md)

---

**You don't need a dedicated server.** If you have a Mac with Apple Silicon, ClawSandbox lets you:

- **Deploy OpenClaw in minutes** — fully sandboxed in Docker, completely isolated from everything else on your machine
- **Run as many as you want** — spin up an entire fleet of OpenClaw instances and experience a one-person company powered by AI

No cloud bills. No new hardware. Everything runs on the machine you already have.

---

## Background

LLM AI applications are evolving through three stages:

1. **ChatBot** — helps everyone access knowledge
2. **Agent** — makes everyone a professional
3. **OpenClaw** — makes everyone a manager

OpenClaw is a self-hosted personal AI assistant that connects to 20+ messaging platforms including WhatsApp, Telegram, and Slack. ClawSandbox removes the deployment bottleneck — instead of struggling to run a single instance, you can spin up an entire fleet with one command.

## What ClawSandbox Does

- **One-command fleet deployment** — give it a number, get that many isolated OpenClaw instances
- **Web Dashboard** — manage your entire fleet from a browser with real-time stats, one-click actions, and embedded noVNC desktops
- **Full desktop per instance** — each claw runs in its own Docker container with an XFCE desktop, accessible via noVNC
- **Lifecycle management** — create, start, stop, restart, and destroy instances via CLI or Dashboard
- **Data persistence** — each instance's data survives container restarts
- **Resource isolation** — instances are isolated from your host system and from each other

## Requirements

- macOS or Linux
- A Docker environment (e.g. [Docker Desktop](https://www.docker.com/products/docker-desktop/))

## Quick Start

### 1. Install

```bash
git clone https://github.com/weiyong1024/ClawSandbox.git
cd ClawSandbox
make build
# Optionally install to PATH (otherwise use ./bin/clawsandbox in place of clawsandbox below):
sudo make install
```

### 2. Build the Docker image

The first run requires building the sandbox image (~1.4 GB, takes several minutes):

```bash
clawsandbox build
```

### 3. Deploy your fleet

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

### 4. Set up each claw

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
clawsandbox create <N>                  # Create N claw instances (auto-pulls image)
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
clawsandbox build                        # Build image locally (offline/custom use)
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

## License

MIT
