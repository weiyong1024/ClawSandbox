FROM node:22-bookworm

# ── Layer 1: System packages ──────────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Lightweight desktop
    xfce4 \
    xfce4-terminal \
    dbus-x11 \
    # VNC server
    tigervnc-standalone-server \
    # noVNC web client
    novnc \
    websockify \
    # Desktop browser (for accessing OpenClaw Control UI at localhost:18789)
    chromium \
    # Fonts (CJK support)
    fonts-noto-cjk \
    # Process manager
    supervisor \
    # Utilities
    curl \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Serve noVNC at /
RUN ln -sf /usr/share/novnc/vnc.html /usr/share/novnc/index.html

# Chromium requires --no-sandbox inside containers.
# Wrap the real binary so every launch path (terminal, .desktop, x-www-browser) gets the flag.
RUN mv /usr/bin/chromium /usr/bin/chromium-real \
    && printf '#!/bin/sh\nexec /usr/bin/chromium-real --no-sandbox "$@"\n' > /usr/bin/chromium \
    && chmod +x /usr/bin/chromium \
    && ln -sf /usr/bin/chromium /usr/bin/chromium-browser \
    && update-alternatives --install /usr/bin/x-www-browser x-www-browser /usr/bin/chromium 200 \
    && update-alternatives --set x-www-browser /usr/bin/chromium

# ── Layer 2: OpenClaw ─────────────────────────────────────────────────────────
RUN npm install -g openclaw@latest \
    && cd /usr/local/lib/node_modules/openclaw/extensions/feishu \
    && npm install --omit=dev

# ── Layer 3: Playwright Chromium (build-time, shared path) ───────────────────
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

# Install system deps required by Playwright's Chromium
RUN node /usr/local/lib/node_modules/openclaw/node_modules/playwright-core/cli.js install-deps chromium

# Download Chromium into /ms-playwright and make world-readable
RUN node /usr/local/lib/node_modules/openclaw/node_modules/playwright-core/cli.js install chromium \
    && chmod -R 755 /ms-playwright

# ── Layer 4: Startup configuration ───────────────────────────────────────────
COPY supervisord.conf /etc/supervisor/supervisord.conf
COPY entrypoint.sh    /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 6901 18789

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s \
    CMD curl -sf http://localhost:6901/ > /dev/null || exit 1

ENTRYPOINT ["/entrypoint.sh"]
