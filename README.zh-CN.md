# ClawSandbox

> 在单台机器上部署和管理多个隔离的 [OpenClaw](https://github.com/openclaw/openclaw) 实例。

[English](./README.md)

---

**不需要额外买一台 Mac Mini。** 只要你有一台 Apple Silicon 的 Mac，ClawSandbox 就能让你：

- **几分钟部署 OpenClaw** — 完全运行在 Docker 沙箱中，与电脑上的现有资源彻底隔离
- **想部署多少就部署多少** — 一键拉起 OpenClaw 军团，体验 AI 驱动的一人公司

不用买云服务器，不用添置新硬件，一切在你的工作机上完成。

---

## 背景

LLM AI 的应用层发展分三个阶段：

1. **ChatBot** — 让人人获取知识
2. **Agent** — 让人人成为专业的人
3. **OpenClaw** — 让人人成为管理者

OpenClaw 是一个自托管的个人 AI 助手，能连接 WhatsApp、Telegram、Slack 等 20+ 消息平台。ClawSandbox 的目标是消除部署瓶颈——不应该让 OpenClaw 单体的获取成为卡点，而应该能轻松部署一个 OpenClaw 军团。

## ClawSandbox 能做什么

- **一条命令部署军团** — 给一个数字，就能得到对应数量的隔离 OpenClaw 实例
- **Web 仪表盘** — 在浏览器中管理整个军团，实时资源监控、一键操作、内嵌 noVNC 桌面
- **每个实例独立桌面** — 每只龙虾运行在独立的 Docker 容器中，内含 XFCE 桌面，通过 noVNC 在任意浏览器中访问
- **生命周期管理** — 通过 CLI 或仪表盘创建、启动、停止、重启、销毁实例
- **数据持久化** — 每个实例的数据在容器重启后保留
- **资源隔离** — 实例之间以及与宿主系统之间完全隔离

## 前置要求

- macOS 或 Linux
- 已安装 Docker 环境（如 [Docker Desktop](https://www.docker.com/products/docker-desktop/)）

## 快速开始

### 1. 安装

```bash
git clone https://github.com/weiyong1024/ClawSandbox.git
cd ClawSandbox
make build
# 可选：安装到系统 PATH（若不执行，后续命令请用 ./bin/clawsandbox 替代 clawsandbox）
sudo make install
```

### 2. 构建 Docker 镜像

首次使用需要构建沙箱镜像（约 1.4 GB，需要几分钟）：

```bash
clawsandbox build
```

### 3. 部署龙虾军团

**方式 A：Web 仪表盘（推荐）**

```bash
# 启动仪表盘
clawsandbox dashboard serve
```

在浏览器中打开 [http://localhost:8080](http://localhost:8080)，点击 **「创建实例」**，选择数量即可。

![仪表盘](docs/images/dashboard.jpeg)

仪表盘提供：
- 所有实例的实时 CPU / 内存监控
- 一键 启动 / 停止 / 销毁 操作
- 点击实例卡片上的 **「桌面」**，进入详情页，内嵌 noVNC 桌面、实时日志和资源图表

![实例桌面](docs/images/instance-desktop.jpeg)

**方式 B：CLI**

```bash
# 创建 3 个隔离的 OpenClaw 实例
clawsandbox create 3

# 查看状态
clawsandbox list
```

### 4. 配置每只龙虾

每只龙虾需要通过其桌面完成一次初始化。可以在仪表盘点击 **「桌面」** 打开，也可以用 CLI：

```bash
clawsandbox desktop claw-1
```

在桌面的终端中执行：

```bash
# 第一步：运行初始化向导（配置 LLM API Key、Telegram Bot 等）
openclaw onboard --flow quickstart

# 第二步：启动 Gateway
openclaw gateway --port 18789
```

Gateway 启动后，在桌面的 **Chromium 浏览器**中访问终端输出的地址（形如 `http://127.0.0.1:18789/#token=...`），即可打开 OpenClaw 控制台。

## CLI 命令

任何命令都可以加 `--help` 查看详细用法和示例：

```bash
clawsandbox --help              # 查看所有可用命令
clawsandbox dashboard --help    # 查看 dashboard 子命令组
```

常用命令速查：

```bash
clawsandbox create <N>                  # 创建 N 个龙虾实例（自动拉取镜像）
clawsandbox list                        # 列出所有实例及状态
clawsandbox desktop <name>              # 在浏览器中打开实例桌面
clawsandbox start <name|all>            # 启动已停止的实例
clawsandbox stop <name|all>             # 停止运行中的实例
clawsandbox restart <name|all>          # 重启实例（先停止再启动）
clawsandbox logs <name> [-f]            # 查看实例日志
clawsandbox destroy <name|all>          # 销毁实例（默认保留数据）
clawsandbox destroy --purge <name|all>  # 销毁实例并删除数据
clawsandbox dashboard serve              # 启动 Web 仪表盘
clawsandbox dashboard stop               # 停止 Web 仪表盘
clawsandbox dashboard restart            # 重启 Web 仪表盘
clawsandbox dashboard open               # 在浏览器中打开仪表盘
clawsandbox build                        # 本地构建镜像（离线或自定义场景）
clawsandbox config                       # 显示当前配置
clawsandbox version                      # 查看版本信息
```

## 重置

销毁所有实例（含数据）、停止 Dashboard、清除构建产物，恢复到全新状态：

```bash
make reset
```

重置后从[快速开始](#快速开始)第 1 步重新开始。

## 资源占用参考

测试环境：M4 MacBook Air（16 GB 内存）

| 实例数 | 内存（空闲） | 内存（Chromium 活跃） |
|--------|-------------|----------------------|
| 1      | ~1.5 GB     | ~3 GB                |
| 3      | ~4.5 GB     | ~9 GB                |
| 5      | ~7.5 GB     | 不建议               |

## 项目状态

正在积极开发中。CLI 和 Web 仪表盘均已可用。

欢迎提 Issue 或 PR 参与贡献。

## License

MIT
