# ClawFleet

[![GitHub release](https://img.shields.io/github/v/release/clawfleet/ClawFleet)](https://github.com/clawfleet/ClawFleet/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/clawfleet/ClawFleet/blob/main/LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-required-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey)](https://github.com/clawfleet/ClawFleet)
[![Wiki](https://img.shields.io/badge/Docs-Wiki-blue)](https://github.com/clawfleet/ClawFleet/wiki)

🌐 **官网:** [clawfleet.io](https://clawfleet.io) · 💬 **社区:** [Discord](https://discord.gg/b5ZSRyrqbt) · 📝 **博客:** [Dev.to](https://dev.to/weiyong1024/i-built-an-open-source-tool-to-run-ai-agents-on-my-laptop-they-collaborate-in-discord-managed-1c42)

> 在单台机器上部署和管理多个隔离的 [OpenClaw](https://github.com/openclaw/openclaw) 实例 — 每个实例运行在独立 Docker 沙箱中，通过浏览器管理。

[English](./README.md)

![Dashboard](docs/images/fleet.png)

## 开始使用

```bash
curl -fsSL https://clawfleet.io/install.sh | sh
```

10 分钟：Docker 自动安装、镜像拉取完毕、Dashboard 在 `http://localhost:8080` 运行。用 ChatGPT 账号登录——已有的 Plus 订阅即可驱动推理，无需 API Key。

[![安装演示](https://img.shields.io/badge/▶_安装演示-30秒-red?style=for-the-badge&logo=youtube)](https://youtu.be/FSxC2vUQ-6k)

---

**想象你买了 N 台专用 Mac Mini**，每台跑一个 OpenClaw 实例，互相隔离，在 Discord 群里自动协作。一家属于你的 AI 公司——数据在你手里，不付订阅费。

**ClawFleet 让这件事免费。** 每个实例跑在独立的 Docker 容器中，隔离的文件系统和网络。在你现有的 Mac 或 Linux 上运行。每个实例约 500 MB 内存。

---

## ClawFleet 能做什么

- **沙箱隔离** — 每个 OpenClaw 跑在独立 Docker 容器中，与宿主机和其他实例完全隔离。恶意技能无法读取你的文件
- **浏览器管理** — 创建、配置、监控、销毁实例，全程无需触碰终端
- **ChatGPT 登录** — 用已有的 ChatGPT 账号认证，或使用 OpenAI、Anthropic、Google AI Studio、DeepSeek 的 API Key
- **版本锁定** — 锁定已测试的 OpenClaw 版本，上游 breaking changes 与你无关
- **军团管理** — 按内存允许的数量创建实例，每个可配置不同模型、人设和频道
- **人设系统** — 定义可复用的角色人设（简介、背景、风格、特征），赋予每个实例
- **技能管理** — 浏览 52 个内置技能，从 ClawHub 13,000+ 社区技能中搜索安装
- **独立桌面** — 每个实例内含 XFCE 桌面，通过 noVNC 在浏览器中访问
- **灵魂存档** — 保存已配置实例的灵魂，随时克隆到新实例
- **自动恢复** — 实例在容器重启后自动恢复运行

## 前置要求

- macOS 或 Linux

## 安装详情

上面的安装命令会自动完成：
1. 自动安装 Docker（macOS 用 Colima，Linux 用 Docker Engine）
2. 下载并安装 `clawfleet` 命令行工具
3. 拉取预构建沙箱镜像（约 1.4 GB）
4. 以后台守护进程启动仪表盘
5. 在浏览器中打开 http://localhost:8080

<details>
<summary><strong>Linux 服务器部署说明</strong></summary>

Linux 环境下 Dashboard 默认监听所有网络接口（`0.0.0.0:8080`），可通过 `http://<服务器IP>:8080` 直接远程访问。如需限制只允许本机访问：

```bash
clawfleet dashboard stop
clawfleet dashboard start --host 127.0.0.1
```

通过 SSH 隧道从本地访问远程服务器上的 Dashboard：

```bash
ssh -fNL 8081:127.0.0.1:8080 user@your-server
# 然后浏览器访问 http://localhost:8081
# 关闭隧道: kill $(lsof -ti:8081)
```

`-fN` 参数使隧道在后台运行，关闭终端不会中断连接。这里使用 8081 端口是因为本地 8080 通常已被本地 ClawFleet 占用。

**控制面板**（OpenClaw 内置 Web UI）的 WebSocket 连接需要浏览器[安全上下文](https://developer.mozilla.org/zh-CN/docs/Web/Security/Secure_Contexts)——SSH 隧道可满足此要求。其他 Dashboard 功能（实例管理、配置、重启龙虾等）无需隧道，可通过 HTTP 直接访问。
</details>

> **手动安装？** 参阅[快速入门](https://github.com/clawfleet/ClawFleet/wiki/Getting-Started)。

### 经营你的公司

把 ClawFleet 想象成**你的 AI 公司**。资产管理是公司的工具仓库，Fleet 是你的 AI 员工团队。给不同员工分配不同的工具，让你的 AI 团队投入生产。

#### 备好工具库

**资产管理 → Model 配置** — 注册 LLM API Key，这是员工用来思考的「大脑」。保存前自动验证。

![Model 配置](docs/images/assets-models.png)

**资产管理 → Character 配置** — 定义可复用的人设。把它想象成「岗位说明书」—— Tony Stark 任 CTO、Steve Jobs 任 CPO、Ray Kroc 任 CMO。为每个角色设定简介、背景故事、沟通风格和性格特征。

![Character 配置](docs/images/assets-characters.png)

**资产管理 → Channel 配置** — 接入消息平台（Telegram、Discord、Slack 等），这是员工服务客户的「工位」。可选；保存前自动验证。

![Channel 配置](docs/images/assets-channels.png)

#### 招聘与装备团队

**实例管理 → 创建实例** — 创建 OpenClaw 实例，每一个都是加入公司的新员工。

**实例管理 → 配置** — 为每个实例分配 Model、Character 和 Channel。给 CTO 装上 Claude 大脑和 Discord 工位，给 CMO 装上 GPT 大脑和 Slack 工位。不同员工、不同工具、不同灵魂。

![实例管理](docs/images/fleet.png)

#### 教会新技能

**实例管理 → Skills** — 每个实例自带 52 个内置技能（天气、GitHub、编程等）。想要更多？在 [ClawHub](https://clawhub.com) 搜索 13,000+ 社区技能，一键安装。不同员工可以学习不同技能。

![技能管理](docs/images/skills.png)

#### 保存与克隆员工的灵魂

当一个员工被训练得足够好时，可以保存它的灵魂——人格、记忆、模型配置和对话历史——以便随时克隆。

**实例管理 → 灵魂保存** — 点击任意已配置实例，将其灵魂保存到存档。

![灵魂保存](docs/images/soul-save-dialog.png)

**实例管理 → 灵魂存档** — 浏览所有已保存的灵魂，随时可加载到新实例。

![灵魂存档](docs/images/soul-archive.png)

**实例管理 → 创建实例 → 灵魂加载** — 创建新实例时，从存档中选择一个灵魂。新员工将继承原始员工的全部知识和人格，无需重新培训。

![灵魂加载](docs/images/soul-create.png)

#### 监督你的团队

点击实例卡片上的 **「桌面」**，进入详情页——内嵌 noVNC 桌面、实时日志和资源图表。

![实例桌面](docs/images/instance-desktop.jpeg)

#### 观摩团队协作

将你的 AI 军团接入消息平台，观摩员工们自主协作。下图中，工程师、产品经理和市场专员在 Discord 群聊中欢迎新同事入职——全程自动运行。

![Bot 协作](docs/images/welcome-on-board-for-bot.jpeg)

## 文档

完整文档请参阅 **[Wiki](https://github.com/clawfleet/ClawFleet/wiki)**，包括：
- [快速开始](https://github.com/clawfleet/ClawFleet/wiki/Getting-Started) — 前置要求、安装、第一个实例
- [仪表盘指南](https://github.com/clawfleet/ClawFleet/wiki/Dashboard-Guide) — 侧边栏导航、资产管理、实例管理
- LLM 提供商指南 — [Anthropic](https://github.com/clawfleet/ClawFleet/wiki/Provider-Anthropic) | [OpenAI](https://github.com/clawfleet/ClawFleet/wiki/Provider-OpenAI) | [Google AI Studio](https://github.com/clawfleet/ClawFleet/wiki/Provider-Google) | [DeepSeek](https://github.com/clawfleet/ClawFleet/wiki/Provider-DeepSeek)
- 频道指南 — [Telegram](https://github.com/clawfleet/ClawFleet/wiki/Channel-Telegram) | [Discord](https://github.com/clawfleet/ClawFleet/wiki/Channel-Discord) | [Slack](https://github.com/clawfleet/ClawFleet/wiki/Channel-Slack) | [Lark](https://github.com/clawfleet/ClawFleet/wiki/Channel-Lark)
- [CLI 参考](https://github.com/clawfleet/ClawFleet/wiki/CLI-Reference) | [常见问题](https://github.com/clawfleet/ClawFleet/wiki/FAQ)

## CLI 命令

任何命令都可以加 `--help` 查看详细用法和示例：

```bash
clawfleet --help              # 查看所有可用命令
clawfleet dashboard --help    # 查看 dashboard 子命令组
```

常用命令速查：

```bash
clawfleet create <N>                  # 创建 N 个龙虾实例（需先构建镜像）
clawfleet create <N> --pull           # 创建 N 个实例，若镜像不存在则从 Registry 拉取
clawfleet configure <name>            # 为实例配置模型以及可选的 Channel 凭据
clawfleet list                        # 列出所有实例及状态
clawfleet desktop <name>              # 在浏览器中打开实例桌面
clawfleet start <name|all>            # 启动已停止的实例
clawfleet stop <name|all>             # 停止运行中的实例
clawfleet restart <name|all>          # 重启实例（先停止再启动）
clawfleet logs <name> [-f]            # 查看实例日志
clawfleet destroy <name|all>          # 销毁实例（默认保留数据）
clawfleet destroy --purge <name|all>  # 销毁实例并删除数据
clawfleet snapshot save <name>        # 保存实例的灵魂到存档
clawfleet snapshot list               # 列出所有已保存的灵魂
clawfleet snapshot delete <name>      # 删除已保存的灵魂
clawfleet create 1 --from-snapshot <soul>  # 从灵魂存档创建实例
clawfleet dashboard serve              # 启动 Web 仪表盘
clawfleet dashboard stop               # 停止 Web 仪表盘
clawfleet dashboard restart            # 重启 Web 仪表盘
clawfleet dashboard open               # 在浏览器中打开仪表盘
clawfleet build                        # 本地构建镜像（离线或自定义场景）
clawfleet config                       # 显示当前配置
clawfleet version                      # 查看版本信息
```

## 重置

销毁所有实例（含数据）、停止 Dashboard、清除构建产物，恢复到全新状态：

```bash
make reset
```

重置后从[开始使用](#开始使用)第 1 步重新开始。

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

使用中如果遇到任何问题，欢迎联系我咨询：weiyong1024@gmail.com

## License

MIT
