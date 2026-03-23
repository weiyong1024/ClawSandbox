# ClawSandbox 系统设计文档

> 版本: v0.1 (Draft)
> 日期: 2026-03-07

[English](./SYSTEM_DESIGN.md)

---

## 1. 概述

ClawSandbox 是一个 CLI 工具，在单台宿主机上批量创建和管理多个隔离的 OpenClaw 实例。每个实例运行在独立的 Docker 容器中，拥有完整的 Linux 桌面环境，用户可通过浏览器访问桌面、通过 Telegram 与各实例独立对话。

## 2. 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                      宿主机 (macOS / Linux)                  │
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

## 3. 组件详细设计

### 3.1 ClawSandbox CLI

**技术栈：**
- 语言：Go 1.25+
- CLI 框架：[cobra](https://github.com/spf13/cobra)
- Docker 交互：[go-dockerclient](https://github.com/fsouza/go-dockerclient)
- 编译产物：单个静态链接二进制文件（darwin/arm64, darwin/amd64, linux/amd64, linux/arm64）

**命令设计：**

```
clawfleet build                       # 构建 Docker 镜像
clawfleet create <N>                  # 创建 N 个龙虾实例
clawfleet list                        # 列出所有实例及状态
clawfleet start <name|all>            # 启动实例
clawfleet stop <name|all>             # 停止实例
clawfleet destroy <name|all>          # 销毁实例（默认保留数据）
clawfleet destroy --purge <name|all>  # 销毁实例并删除数据
clawfleet desktop <name>              # 在浏览器中打开该实例的 noVNC 桌面
clawfleet logs <name> [-f]            # 查看实例日志
```

**配置文件：** `~/.clawfleet/config.yaml`

```yaml
# 基础镜像配置
image:
  name: "ghcr.io/clawfleet/clawfleet"
  tag: "latest"

# 端口范围
ports:
  novnc_start: 6901      # noVNC 起始端口，递增分配
  gateway_start: 18789   # Gateway 起始端口，递增分配

# 容器资源限制
resources:
  memory_limit: "4g"     # 每个容器内存上限
  cpu_limit: "2.0"       # 每个容器 CPU 上限（核数）

# 实例命名前缀
naming:
  prefix: "claw"      # 实例名: claw-1, claw-2, ...
```

### 3.2 Docker 镜像 (ghcr.io/clawfleet/clawfleet)

**基础镜像选择：** `node:22-bookworm`（与 OpenClaw 官方 Docker 方案一致）

**镜像分层设计：**

```dockerfile
# Layer 1: 基础系统 + 桌面环境
FROM node:22-bookworm

# 系统依赖 + XFCE 桌面 + VNC/noVNC
RUN apt-get update && apt-get install -y \
    xfce4 xfce4-terminal \
    tigervnc-standalone-server \
    novnc websockify \
    dbus-x11 \
    chromium \
    fonts-noto-cjk \
    supervisor curl wget \
    && rm -rf /var/lib/apt/lists/*

# Chromium --no-sandbox 封装（容器内必需）
RUN mv /usr/bin/chromium /usr/bin/chromium-real \
    && printf '#!/bin/sh\nexec /usr/bin/chromium-real --no-sandbox "$@"\n' > /usr/bin/chromium \
    && chmod +x /usr/bin/chromium

# Layer 2: OpenClaw 安装
RUN npm install -g openclaw@latest

# Layer 3: Playwright Chromium（构建时安装，避免运行时下载）
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
RUN node /usr/local/lib/node_modules/openclaw/node_modules/playwright-core/cli.js install chromium

# Layer 4: 启动配置
COPY supervisord.conf /etc/supervisor/supervisord.conf
COPY entrypoint.sh /entrypoint.sh

EXPOSE 6901 18789
ENTRYPOINT ["/entrypoint.sh"]
```

**进程管理 (supervisord)：**

容器内需要同时运行多个进程，使用 supervisord 统一管理：

| 进程 | 作用 | 端口 |
|------|------|------|
| Xvnc | 虚拟帧缓冲 + VNC 服务端 | DISPLAY :1 / 5901 (容器内) |
| XFCE4 | 桌面环境 | — |
| noVNC + websockify | VNC → WebSocket 代理 | 6901 (映射到宿主) |
| OpenClaw Gateway | OpenClaw 核心服务（手动启动） | 18789 (映射到宿主) |

**entrypoint.sh 职责：**
1. 初始化 VNC 密码（可由环境变量注入或使用默认值）
2. 创建 `.vnc` 和 `.openclaw` 目录，设置权限
3. 启动 supervisord

**预估镜像大小：**
- 基础系统 + XFCE + VNC/noVNC: ~800MB
- Node.js 22 + OpenClaw: ~300MB
- Chromium (Playwright): ~300MB
- **总计: ~1.4GB**

### 3.3 端口分配策略

CLI 维护一个简单的端口分配器，避免冲突：

```
实例       noVNC 端口     Gateway 端口
claw-1  6901          18789
claw-2  6902          18790
claw-3  6903          18791
...
claw-N  6900+N        18788+N
```

分配逻辑：
- 从配置的起始端口开始递增
- 创建前检测端口是否被占用（`net.Listen` 探测）
- 分配结果记录在状态文件中

### 3.4 状态管理

**状态文件：** `~/.clawfleet/state.json`

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

状态文件是 CLI 的元数据缓存，真实状态以 Docker 容器状态为准。CLI 每次操作时与 Docker API 对账，修正不一致。

### 3.5 数据卷设计

每个容器挂载独立的宿主目录，实现数据持久化和隔离：

```
~/.clawfleet/data/
├── claw-1/
│   └── openclaw/     → 容器内 /home/node/.openclaw
├── claw-2/
│   └── openclaw/     → 容器内 /home/node/.openclaw
└── claw-3/
    └── openclaw/     → 容器内 /home/node/.openclaw
```

- 容器销毁后数据默认保留在宿主机上
- `clawfleet destroy --purge` 可同时删除数据
- 目录权限设为 uid 1000（node 用户），与容器内用户一致

### 3.6 网络设计

**Docker 网络：**
- 创建专用 bridge 网络 `clawfleet-net`
- 各容器通过容器名互相可达（为后续龙虾间通信预留）
- 每个容器仅暴露 noVNC 和 Gateway 端口到宿主机

```
clawfleet-net (bridge)
├── claw-1 (172.20.0.2)
├── claw-2 (172.20.0.3)
└── claw-3 (172.20.0.4)
```

**宿主机访问：**
- noVNC: `http://localhost:6901` → claw-1 桌面
- Gateway: `ws://localhost:18789` → claw-1 Gateway（内部用，用户一般不直接访问）

## 4. 用户操作流程

### 4.1 首次使用

```bash
# 1. 构建镜像（首次，约 1.4GB）
clawfleet build

# 2. 创建 3 个龙虾
clawfleet create 3
# → 输出:
#   Creating claw-1 ... ✓  desktop: http://localhost:6901
#   Creating claw-2 ... ✓  desktop: http://localhost:6902
#   Creating claw-3 ... ✓  desktop: http://localhost:6903

# 3. 打开龙虾桌面，进入 OpenClaw onboard 配置 Telegram 等
clawfleet desktop claw-1
```

### 4.2 日常使用

```bash
# 查看军团状态
clawfleet list
# NAME        STATUS    DESKTOP              UPTIME
# claw-1   running   http://localhost:6901 2d 3h
# claw-2   running   http://localhost:6902 2d 3h
# claw-3   stopped   -                    -

# 启动/停止
clawfleet start claw-3
clawfleet stop claw-1

# 查看日志
clawfleet logs claw-1
```

### 4.3 OpenClaw Onboard 流程（容器内）

用户通过 noVNC 进入桌面后，在终端中执行：

```bash
# 第一步：运行初始化向导（配置 LLM API Key、Telegram Bot 等）
openclaw onboard --flow quickstart

# 第二步：启动 Gateway
openclaw gateway --port 18789
```

Gateway 启动后，在桌面的 Chromium 浏览器中访问终端输出的地址（形如 `http://127.0.0.1:18789/#token=...`），即可打开 OpenClaw 控制台。

**后续优化方向（非第一版）：** 将 onboard 流程通过 ClawSandbox CLI 自动化，用户只需提供 API key 和 Telegram bot token，无需手动进入桌面操作。

## 5. 资源预算

基于 M4 MacBook Air (16GB RAM, 512GB SSD) 的测试环境：

| 资源 | 单个实例 | 3 个实例 | 5 个实例 |
|------|---------|---------|---------|
| 内存（idle） | ~1.5GB | ~4.5GB | ~7.5GB |
| 内存（Chromium 活跃） | ~3GB | ~9GB | 不建议 |
| 磁盘（镜像） | 1.4GB（共享） | 1.4GB | 1.4GB |
| 磁盘（数据卷/实例） | ~200MB | ~600MB | ~1GB |
| CPU（idle） | <0.5 核 | <1.5 核 | <2.5 核 |

**建议：**
- 16GB 宿主机：最多同时运行 3 个活跃实例（含 Chromium），或 5 个轻载实例
- 默认每容器 memory_limit=4GB，防止单实例内存泄漏影响宿主
- 用户可通过配置调整资源限制

## 6. 项目目录结构

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
│   ├── SYSTEM_DESIGN.md        # English (primary)
│   └── SYSTEM_DESIGN.zh-CN.md  # 中文
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── README.zh-CN.md
└── CLAUDE.md
```

## 7. 技术依赖总览

### 宿主机（ClawSandbox CLI）
| 依赖 | 用途 |
|------|------|
| Go 1.25+ | 编译 CLI |
| Docker Engine | 容器运行时（Docker Desktop for Mac 或 colima） |

### 容器内
| 依赖 | 版本 | 用途 |
|------|------|------|
| Debian Bookworm | 12 | 基础 OS |
| Node.js | 22 | OpenClaw 运行时 |
| OpenClaw | latest | AI 助手核心 |
| Chromium (Playwright) | — | 浏览器自动化 |
| XFCE4 | 4.18 | 轻量桌面环境 |
| TigerVNC | — | VNC 服务端 |
| noVNC + websockify | — | Web VNC 客户端 |
| supervisord | — | 容器内多进程管理 |

### Go 模块依赖
| 模块 | 用途 |
|------|------|
| `github.com/spf13/cobra` | CLI 框架 |
| `github.com/fsouza/go-dockerclient` | Docker Engine API |
| `gopkg.in/yaml.v3` | 配置文件解析 |

## 8. 交互界面演进策略

项目采用分阶段交付策略，先跑通核心再打磨体验：

### 阶段一：CLI（开发调试期）
所有操作通过命令行完成，快速验证核心功能闭环。这一阶段的产出是一个功能完整、可流畅使用的 CLI 工具。

### 阶段二：Web 配置页面（开源发布前）
在 CLI 跑通后，为非技术用户实现 Web UI：

- **军团管理面板**：ClawSandbox 在宿主机启动一个 Web 服务，提供统一的军团总览页面（创建/启停/销毁实例、查看状态）
- **实例管理后台**：每个龙虾实例有独立的配置页面（LLM Provider、Telegram Bot、模型选择等），用户无需进入桌面终端即可完成 OpenClaw 配置
- **桌面入口**：页面内嵌或链接到各实例的 noVNC 桌面

CLI 始终保留作为高级用户和自动化场景的入口。

## 9. 实现范围

### 阶段一 MVP（CLI）

**包含：**
- [x] Docker 镜像构建（Dockerfile + supervisord + entrypoint）
- [x] `build` — 构建 Docker 镜像
- [x] `create N` — 创建 N 个容器实例
- [x] `list` — 列出所有实例及状态（实时从 Docker 获取）
- [x] `start/stop` — 启停实例
- [x] `destroy [--purge]` — 销毁实例
- [x] `desktop` — 浏览器打开 noVNC
- [x] `logs [-f]` — 查看实例日志
- [x] 端口自动分配和冲突检测
- [x] 实例数据卷持久化

### 阶段二（Web UI）
- [ ] 军团管理面板（Web 服务）
- [ ] 实例配置页面（LLM/Telegram 等）
- [ ] OpenClaw onboard 自动化

**不包含（更远期）：**
- 龙虾间群聊和任务分派
- 实例模板/预设配置
- 远程宿主机支持

## 10. 验证方案

### 构建验证
```bash
make build
clawfleet build
```

### 端到端验证
```bash
# 1. 创建 2 个实例
clawfleet create 2

# 2. 确认容器运行中
clawfleet list

# 3. 访问桌面
clawfleet desktop claw-1
# → 浏览器打开 http://localhost:6901，看到 XFCE 桌面

# 4. 在桌面终端中完成 OpenClaw 配置
openclaw onboard --flow quickstart
openclaw gateway --port 18789
# → 在 Chromium 中打开终端输出的地址，确认控制台可用

# 5. 停止/启动
clawfleet stop claw-1
clawfleet start claw-1
# → 确认数据（~/.openclaw）在容器重启后保留

# 6. 销毁
clawfleet destroy claw-2
clawfleet list
# → 确认 claw-2 已移除
```

### 资源验证
```bash
docker stats claw-1 claw-2
# → 确认内存在 memory_limit 之内
```

## 11. 版本管理

一次 `git tag` 同时锁定 CLI 和 Docker 镜像的版本，自动发布到 GitHub Release 和 GHCR。

### 架构

```
git tag v0.1.0 && git push origin v0.1.0
        │
        ▼
   GitHub Actions
   ┌──────────────────┬──────────────────────┐
   │  GoReleaser       │  Docker Build+Push    │
   │  CLI 二进制 x4    │  ghcr.io 镜像         │
   │  (darwin/linux    │  :0.1.0 + :latest     │
   │   x amd64/arm64) │                        │
   └────────┬─────────┴──────────┬─────────────┘
            ▼                    ▼
     GitHub Release         ghcr.io/clawfleet/clawfleet
```

### 共享版本包 (`internal/version/`)

版本元数据（`Version`、`GitCommit`、`BuildDate`）在构建时通过 ldflags 注入到 `internal/version`。`internal/cli/version.go` 和 `internal/config/config.go` 均从此包导入。

`ImageTag()` 函数根据 CLI 版本推导 Docker 镜像标签：
- release 版本（如 `v0.1.0`）：返回 `0.1.0`
- dev 版本（无标签）：返回 `latest`

### 镜像命名与标签

- **仓库地址**：`ghcr.io/clawfleet/clawfleet`
- **默认标签**：运行时通过 `version.ImageTag()` 动态确定
- `clawfleet build` 同时打 `:<version>` 和 `:latest` 两个标签
- `clawfleet create` 使用 `:<version>` 确保 CLI 与镜像版本一致

### 自动拉取镜像

当 `clawfleet create`（或仪表盘的创建接口）发现本地没有镜像时，自动尝试从 GHCR 拉取。联网用户无需执行 `clawfleet build`。

```
本地未找到镜像
  → docker pull ghcr.io/clawfleet/clawfleet:0.1.0
  → 成功？继续创建
  → 失败？提示用户执行 clawfleet build
```

### CI 流水线 (`.github/workflows/release.yml`)

触发条件：`v*` 标签推送。两个并行任务：

| 任务 | 工具 | 产出 |
|------|------|------|
| `release` | GoReleaser | 4 个平台的 CLI 二进制 → GitHub Release |
| `docker` | docker/build-push-action | 镜像 → `ghcr.io/clawfleet/clawfleet:<version>` + `:latest` |

### 发版流程（维护者操作）

```bash
git checkout main
git pull origin main
git tag v0.1.0
git push origin v0.1.0
# CI 自动完成剩余工作
```
