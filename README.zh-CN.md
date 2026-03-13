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
- **灵魂存档** — 保存已配置实例的灵魂（人格、记忆、模型配置），随时克隆到新实例
- **自动恢复** — 已配置实例在容器重启后自动恢复 gateway 运行
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
# 如果本机没有 Go，ClawSandbox 会自动把 go.mod 需要的版本
# 下载到当前用户的本地 toolchain 目录，再继续构建。
# 可选：安装到系统 PATH（若不执行，后续命令请用 ./bin/clawsandbox 替代 clawsandbox）
sudo make install
```

### 2. 启动仪表盘

```bash
clawsandbox dashboard serve
# 打开 http://localhost:8080
```

### 3. 构建镜像

在仪表盘中点击 **「系统 → 镜像管理」**，构建沙箱镜像（约 1.4 GB，首次构建需要几分钟）。

### 4. 经营你的公司

把 ClawSandbox 想象成**你的 AI 公司**。资产管理是公司的工具仓库，Fleet 是你的 AI 员工团队。给不同员工分配不同的工具，让你的 AI 团队投入生产。

#### 备好工具库

**资产管理 → Model 配置** — 注册 LLM API Key，这是员工用来思考的「大脑」。保存前自动验证。

![Model 配置](docs/images/assets-models.png)

**资产管理 → Channel 配置** — 接入消息平台（Telegram、Discord、Slack 等），这是员工服务客户的「工位」。可选；保存前自动验证。

![Channel 配置](docs/images/assets-channels.png)

#### 招聘与装备团队

**实例管理 → 创建实例** — 创建 OpenClaw 实例，每一个都是加入公司的新员工。

**实例管理 → 配置** — 从资产池为每个实例分配 Model 和 Channel。不同员工可以使用不同的工具，承担不同的任务。

![实例管理](docs/images/fleet.png)

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

## 文档

完整文档请参阅 **[Wiki](https://github.com/weiyong1024/ClawSandbox/wiki)**，包括：
- [快速开始](https://github.com/weiyong1024/ClawSandbox/wiki/Getting-Started) — 前置要求、安装、第一个实例
- [仪表盘指南](https://github.com/weiyong1024/ClawSandbox/wiki/Dashboard-Guide) — 侧边栏导航、资产管理、实例管理
- LLM 提供商指南 — [Anthropic](https://github.com/weiyong1024/ClawSandbox/wiki/Provider-Anthropic) | [OpenAI](https://github.com/weiyong1024/ClawSandbox/wiki/Provider-OpenAI) | [Google](https://github.com/weiyong1024/ClawSandbox/wiki/Provider-Google) | [DeepSeek](https://github.com/weiyong1024/ClawSandbox/wiki/Provider-DeepSeek)
- 频道指南 — [Telegram](https://github.com/weiyong1024/ClawSandbox/wiki/Channel-Telegram) | [Discord](https://github.com/weiyong1024/ClawSandbox/wiki/Channel-Discord) | [Slack](https://github.com/weiyong1024/ClawSandbox/wiki/Channel-Slack) | [Lark](https://github.com/weiyong1024/ClawSandbox/wiki/Channel-Lark)
- [CLI 参考](https://github.com/weiyong1024/ClawSandbox/wiki/CLI-Reference) | [常见问题](https://github.com/weiyong1024/ClawSandbox/wiki/FAQ)

## CLI 命令

任何命令都可以加 `--help` 查看详细用法和示例：

```bash
clawsandbox --help              # 查看所有可用命令
clawsandbox dashboard --help    # 查看 dashboard 子命令组
```

常用命令速查：

```bash
clawsandbox create <N>                  # 创建 N 个龙虾实例（需先构建镜像）
clawsandbox create <N> --pull           # 创建 N 个实例，若镜像不存在则从 Registry 拉取
clawsandbox list                        # 列出所有实例及状态
clawsandbox desktop <name>              # 在浏览器中打开实例桌面
clawsandbox start <name|all>            # 启动已停止的实例
clawsandbox stop <name|all>             # 停止运行中的实例
clawsandbox restart <name|all>          # 重启实例（先停止再启动）
clawsandbox logs <name> [-f]            # 查看实例日志
clawsandbox destroy <name|all>          # 销毁实例（默认保留数据）
clawsandbox destroy --purge <name|all>  # 销毁实例并删除数据
clawsandbox snapshot save <name>        # 保存实例的灵魂到存档
clawsandbox snapshot list               # 列出所有已保存的灵魂
clawsandbox snapshot delete <name>      # 删除已保存的灵魂
clawsandbox create 1 --from-snapshot <soul>  # 从灵魂存档创建实例
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
