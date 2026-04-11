#!/usr/bin/env sh

set -eu

REPO="clawfleet/ClawFleet"
BINARY="clawfleet"
IMAGE_REPO="ghcr.io/clawfleet/clawfleet"
DOCKER_CMD="docker"

# ---------------------------------------------------------------------------
# Color helpers (disabled when stdout is not a terminal)
# ---------------------------------------------------------------------------
setup_colors() {
  if [ -t 1 ]; then
    BOLD='\033[1m'
    GREEN='\033[32m'
    BLUE='\033[34m'
    CYAN='\033[36m'
    YELLOW='\033[33m'
    RED='\033[31m'
    RESET='\033[0m'
  else
    BOLD='' GREEN='' BLUE='' CYAN='' YELLOW='' RED='' RESET=''
  fi
}

step() { printf "${BOLD}${BLUE}==>${RESET} ${BOLD}%s${RESET}\n" "$*"; }
ok()   { printf "  ${GREEN}[OK]${RESET} %s\n" "$*"; }
warn() { printf "  ${YELLOW}[WARN]${RESET} %s\n" "$*" >&2; }

die() {
  printf "${RED}Error: %s${RESET}\n" "$*" >&2
  exit 1
}

print_banner() {
  printf "${BOLD}${BLUE}"
  cat <<'BANNER'

    ____ _               _____ _           _
   / ___| | __ ___      _|  ___| | ___  ___| |_
  | |   | |/ _` \ \ /\ / / |_  | |/ _ \/ _ \ __|
  | |___| | (_| |\ V  V /|  _| | |  __/  __/ |_
   \____|_|\__,_| \_/\_/ |_|   |_|\___|\___|\__|

BANNER
  printf "${RESET}"
  printf "  ${CYAN}Deploy and manage your AI workforce${RESET}\n\n"
}

# ---------------------------------------------------------------------------
# OS / Architecture detection
# ---------------------------------------------------------------------------
detect_os() {
  case "$(uname -s)" in
    Darwin) printf 'darwin\n' ;;
    Linux)  printf 'linux\n' ;;
    *)      die "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    arm64|aarch64) printf 'arm64\n' ;;
    x86_64|amd64)  printf 'amd64\n' ;;
    *)             die "unsupported architecture: $(uname -m)" ;;
  esac
}

# ---------------------------------------------------------------------------
# Download / checksum helpers
# ---------------------------------------------------------------------------
download_file() {
  url=$1
  destination=$2

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$destination"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO "$destination" "$url"
    return
  fi

  die "curl or wget is required"
}

sha256_verify() {
  file=$1
  expected=$2

  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$file" | awk '{ print $1 }')
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$file" | awk '{ print $1 }')
  else
    warn "no sha256sum or shasum found, skipping checksum verification"
    return 0
  fi

  if [ "$actual" != "$expected" ]; then
    die "checksum mismatch: expected $expected, got $actual"
  fi
}

latest_version() {
  url="https://api.github.com/repos/${REPO}/releases/latest"

  if command -v curl >/dev/null 2>&1; then
    tag=$(curl -fsSL "$url" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')
  elif command -v wget >/dev/null 2>&1; then
    tag=$(wget -qO- "$url" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')
  else
    die "curl or wget is required"
  fi

  [ -n "$tag" ] || die "failed to determine latest release"
  printf '%s\n' "$tag"
}

# ---------------------------------------------------------------------------
# Docker: ensure installed and running
# ---------------------------------------------------------------------------
ensure_docker() {
  step "Checking Docker..."

  # Phase 1: is docker CLI available?
  if command -v docker >/dev/null 2>&1; then
    # Phase 2: is the daemon running?
    if docker info >/dev/null 2>&1; then
      ok "Docker is ready"
      return
    fi
    # Installed but daemon not running — try to start it
    start_docker_daemon
    return
  fi

  # Not installed — install automatically
  case "$(uname -s)" in
    Darwin) install_docker_macos ;;
    Linux)  install_docker_linux ;;
    *)      die "Cannot auto-install Docker on $(uname -s). Please install Docker manually." ;;
  esac
}

install_docker_macos() {
  # Use Colima (open source, no EULA popup) + Docker CLI via Homebrew.
  # Docker Desktop is recommended for best experience but not required.
  warn "Docker not found. ClawFleet will install Colima (open-source Docker runtime)."
  warn "For the best experience, install Docker Desktop first: https://docker.com/products/docker-desktop"
  warn "Continuing with Colima — this may take a few minutes on first run..."
  printf "\n"

  # 1. Ensure Homebrew
  if ! command -v brew >/dev/null 2>&1; then
    step "Installing Homebrew (prerequisite for Colima)..."
    NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

    # Set up PATH for newly installed Homebrew
    if [ -f /opt/homebrew/bin/brew ]; then
      eval "$(/opt/homebrew/bin/brew shellenv)"
    elif [ -f /usr/local/bin/brew ]; then
      eval "$(/usr/local/bin/brew shellenv)"
    fi

    command -v brew >/dev/null 2>&1 || die "Homebrew installation failed"
    ok "Homebrew installed"
  fi

  # 2. Install Colima + Docker CLI
  step "Installing Colima and Docker CLI via Homebrew..."
  brew install colima docker 2>&1 | tail -1
  ok "Colima and Docker CLI installed"

  # 3. Start Colima (first run downloads a ~300 MB VM image — be patient)
  step "Starting Docker runtime (Colima) — first launch may take 3-5 minutes..."
  colima start --cpu 2 --memory 4 --disk 60

  wait_for_docker
}

install_docker_linux() {
  step "Installing Docker Engine..."
  curl -fsSL https://get.docker.com | sudo sh 2>&1 | tail -5

  # Start Docker daemon
  if command -v systemctl >/dev/null 2>&1; then
    sudo systemctl enable --now docker 2>/dev/null || true
  else
    sudo service docker start 2>/dev/null || true
  fi

  # If official Docker install failed, fallback to distro package
  sleep 3
  if ! docker info >/dev/null 2>&1 && ! sudo docker info >/dev/null 2>&1; then
    warn "Official Docker install incomplete, trying distro package..."
    sudo apt-get update -qq 2>/dev/null
    sudo apt-get install -y docker.io 2>&1 | tail -3
    if command -v systemctl >/dev/null 2>&1; then
      sudo systemctl enable --now docker 2>/dev/null || true
    else
      sudo service docker start 2>/dev/null || true
    fi
  fi

  # Add user to docker group
  if ! groups | grep -q docker; then
    sudo usermod -aG docker "$USER" 2>/dev/null || true
    # In the current shell, docker still needs sudo
    DOCKER_CMD="sudo docker"
    warn "Added $USER to docker group. Using sudo for docker commands in this session."
    warn "Log out and back in to use docker without sudo."
  fi

  wait_for_docker
}

start_docker_daemon() {
  step "Starting Docker daemon..."
  case "$(uname -s)" in
    Darwin)
      # Try Colima first
      if command -v colima >/dev/null 2>&1; then
        colima start 2>/dev/null || true
      fi
      # Try Docker Desktop as fallback
      if ! docker info >/dev/null 2>&1; then
        open -a Docker 2>/dev/null || true
      fi
      ;;
    Linux)
      if command -v systemctl >/dev/null 2>&1; then
        sudo systemctl start docker 2>/dev/null || true
      else
        sudo service docker start 2>/dev/null || true
      fi
      ;;
  esac

  wait_for_docker
}

wait_for_docker() {
  attempts=0
  max_attempts=150
  while ! $DOCKER_CMD info >/dev/null 2>&1; do
    attempts=$((attempts + 1))
    if [ "$attempts" -ge "$max_attempts" ]; then
      die "Docker did not start within 5 minutes. Please check your Docker installation."
    fi
    printf "."
    sleep 2
  done
  printf "\n"
  ok "Docker is ready"
}

# ---------------------------------------------------------------------------
# Binary installation
# ---------------------------------------------------------------------------
install_binary() {
  src=$1

  # Try /usr/local/bin first
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    install_dir="/usr/local/bin"
  elif [ -w "$HOME/.local/bin" ] || mkdir -p "$HOME/.local/bin" 2>/dev/null; then
    install_dir="$HOME/.local/bin"
  else
    die "cannot find a writable install directory. Run with sudo or create ~/.local/bin"
  fi

  cp "$src" "$install_dir/$BINARY"
  chmod +x "$install_dir/$BINARY"

  ok "Installed $BINARY to $install_dir/$BINARY"

  # Check if install_dir is in PATH
  case ":$PATH:" in
    *":$install_dir:"*) ;;
    *)
      warn "Add \"$install_dir\" to your PATH to use $BINARY from anywhere."
      warn "  export PATH=\"$install_dir:\$PATH\""
      # Try to add to current session so subsequent steps work
      export PATH="$install_dir:$PATH"
      ;;
  esac
}

download_and_install() {
  step "Fetching latest release..."
  if [ -z "$version" ]; then
    version=$(latest_version)
  fi
  ok "Version: $version"

  # Strip leading 'v' for archive name (GoReleaser uses version without 'v')
  ver_num="${version#v}"
  archive_name="${BINARY}_${ver_num}_${os}_${arch}.tar.gz"
  base_url="https://github.com/${REPO}/releases/download/${version}"

  tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/clawfleet-install.XXXXXX")
  trap 'rm -rf "$tmp_dir"' EXIT INT TERM HUP

  step "Downloading ${archive_name}..."
  download_file "${base_url}/${archive_name}" "$tmp_dir/${archive_name}"

  step "Downloading checksums..."
  download_file "${base_url}/checksums.txt" "$tmp_dir/checksums.txt"

  expected=$(grep "${archive_name}" "$tmp_dir/checksums.txt" | awk '{ print $1 }')
  [ -n "$expected" ] || die "checksum for ${archive_name} not found in checksums.txt"

  step "Verifying checksum..."
  sha256_verify "$tmp_dir/${archive_name}" "$expected"
  ok "Checksum verified"

  step "Extracting..."
  tar -C "$tmp_dir" -xzf "$tmp_dir/${archive_name}"

  step "Installing binary..."
  install_binary "$tmp_dir/$BINARY"
}

# ---------------------------------------------------------------------------
# Docker image pull
# ---------------------------------------------------------------------------
pull_docker_image() {
  # Determine image tag: release versions use version tag, others use latest
  case "$version" in
    v[0-9]*.[0-9]*.[0-9]*) image_tag="$version" ;;
    *)                     image_tag="latest" ;;
  esac

  step "Pulling sandbox image ${IMAGE_REPO}:${image_tag} (~1.4 GB)..."
  if $DOCKER_CMD pull "${IMAGE_REPO}:${image_tag}"; then
    ok "Image pulled successfully"
  else
    warn "Image pull failed. You can pull later from the Dashboard."
  fi
}

# ---------------------------------------------------------------------------
# Daemon setup
# ---------------------------------------------------------------------------
setup_daemon() {
  # macOS: local dev machine, bind localhost only
  # Linux: typically a remote server, bind all interfaces
  case "$(uname -s)" in
    Darwin) dashboard_host="127.0.0.1" ;;
    *)      dashboard_host="0.0.0.0" ;;
  esac

  step "Starting ClawFleet Dashboard..."
  if "$install_dir/$BINARY" dashboard start --port 8080 --host "$dashboard_host" 2>&1; then
    ok "Dashboard started"
  else
    warn "Could not start daemon automatically."
    warn "Run 'clawfleet dashboard start' manually."
  fi
}

# ---------------------------------------------------------------------------
# Browser
# ---------------------------------------------------------------------------
open_browser() {
  url="http://localhost:8080"
  case "$(uname -s)" in
    Darwin) open "$url" 2>/dev/null || true ;;
    Linux)  xdg-open "$url" 2>/dev/null || true ;;
  esac
}

# ---------------------------------------------------------------------------
# Success message
# ---------------------------------------------------------------------------
print_success() {
  printf "\n${BOLD}${GREEN}"
  printf "  ClawFleet installed successfully!\n"
  printf "${RESET}\n"
  if [ "$(uname -s)" = "Darwin" ]; then
    printf "  Dashboard:  ${CYAN}http://localhost:8080${RESET}\n"
  else
    printf "  Dashboard:  ${CYAN}http://0.0.0.0:8080${RESET} (accessible from your network)\n"
  fi
  printf "  Version:    %s\n" "$version"
  printf "\n"
  printf "  ${BOLD}Next steps:${RESET}\n"
  printf "    1. Add an LLM API key in Assets > Models\n"
  printf "    2. Create instances in Fleet\n"
  printf "    3. Configure and deploy your AI workforce\n"
  printf "\n"
  printf "  ${BOLD}Useful commands:${RESET}\n"
  printf "    clawfleet dashboard status   Check daemon status\n"
  printf "    clawfleet dashboard stop     Stop the daemon\n"
  printf "    clawfleet create 3           Create 3 instances\n"
  printf "    clawfleet list               List all instances\n"
  printf "\n"
  printf "  Docs:    ${CYAN}https://github.com/clawfleet/ClawFleet/wiki${RESET}\n"
  printf "\n"
}

# ---------------------------------------------------------------------------
# Usage
# ---------------------------------------------------------------------------
usage() {
  cat >&2 <<'EOF'
Usage: install.sh [OPTIONS]

Install and deploy ClawFleet with a single command.

Options:
  --version <tag>   Install a specific version (e.g. v0.1.0). Default: latest.
  --skip-pull       Skip Docker image pull (pull later via Dashboard).
  --no-daemon       Don't start the background daemon after install.
  --help, -h        Show this help message.
EOF
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  version=""
  skip_pull=false
  no_daemon=false

  while [ $# -gt 0 ]; do
    case "$1" in
      --version)
        [ $# -ge 2 ] || die "--version requires a value"
        version="$2"
        shift 2
        ;;
      --skip-pull)
        skip_pull=true
        shift
        ;;
      --no-daemon)
        no_daemon=true
        shift
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        die "unknown option: $1"
        ;;
    esac
  done

  setup_colors
  print_banner

  os=$(detect_os)
  arch=$(detect_arch)

  # 1. Ensure Docker is available
  ensure_docker

  # 2. Download and install the binary
  download_and_install

  # 3. Pull pre-built Docker image
  if [ "$skip_pull" = false ]; then
    pull_docker_image
  fi

  # 4. Start the Dashboard daemon
  if [ "$no_daemon" = false ]; then
    setup_daemon
    open_browser
  fi

  print_success
}

main "$@"
