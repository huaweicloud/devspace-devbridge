---
title: AI Agent Skill
description: 通过 huawei-cloud-devbridge-tunnel skill 让 AI 代理自动管理开发隧道。
---

# AI Agent Skill

<p class="lead"><code>huawei-cloud-devbridge-tunnel</code> 是一个 AI 代理技能，封装了 DevBridge CLI 的全部隧道操作，让代理自动完成安装、认证、创建隧道和托管服务。</p>

## 适用场景

- 让 AI 代理帮你一键暴露本地服务，无需手动逐条执行 CLI 命令；
- 在对话中自然语言创建、查询、更新和删除隧道；
- 自动处理 CLI 版本差异，无需关心 flag 重命名等兼容问题。

## 安装 Skill

### 通过 Skill 管理器安装

```bash
npx skills add https://gitcode.com/huaweicloud/huaweicloud-skills.git#master \
  --skill huawei-cloud-devbridge-tunnel -y
```

安装后 skill 位于 `~/.agents/skills/huawei-cloud-devbridge-tunnel`。

### 从 Skill 库搜索安装

已安装 `huawei-cloud-find-skills` 的代理可以通过关键词搜索并安装。由于该 skill 目前未被技能索引收录，搜索时需直接使用 skill 名称。

## Skill 能做什么

Skill 封装了以下操作，均通过适配层函数执行，自动适配不同 CLI 版本：

| 适配层函数       | 对应 CLI 命令           | 用途          |
| ---------------- | ----------------------- | ------------- |
| `db_init`        | —                       | 验证 CLI 可用 |
| `db_create`      | `devbridge create`      | 创建隧道      |
| `db_port_create` | `devbridge port create` | 创建端口      |
| `db_host`        | `devbridge host`        | 托管本地服务  |
| `db_connect`     | `devbridge connect`     | 连接远程隧道  |
| `db_list`        | `devbridge list`        | 列出隧道      |
| `db_show`        | `devbridge show`        | 查看隧道详情  |
| `db_update`      | `devbridge update`      | 更新隧道      |
| `db_delete`      | `devbridge delete`      | 删除隧道      |
| `db_port_list`   | `devbridge port list`   | 列出端口      |
| `db_port_delete` | `devbridge port delete` | 删除端口      |
| `db_version`     | `devbridge version`     | 查看 CLI 版本 |
| `db_auth_status` | `devbridge auth status` | 查看登录状态  |
| `db_auth_login`  | `devbridge auth login`  | 登录          |

## 适配层

DevBridge CLI 会在每次使用时自动更新，可能导致 flag 重命名或参数变化。Skill 通过 `scripts/devbridge_cmd.sh` 适配层解决这一问题：

1. 执行前通过 `devbridge <command> --help` 动态发现当前版本支持的 flag；
2. 执行后解析错误信息，自动恢复因 flag 重命名、新增必选参数或命令重命名导致的失败；
3. 自动恢复失败时，将错误和建议命令返回给代理。

::: tip 为什么不直接调用 devbridge
所有隧道、端口、Host 和 Connect 操作必须通过 `db_*` 适配层函数执行，而不是直接调用 `devbridge` 命令。只有 `devbridge auth` 和 `devbridge version` 不受此限制。
:::

## 前置条件

Skill 会在首次使用时自动完成以下准备工作：

1. 安装或更新 DevBridge CLI 到最新版本；
2. 加载适配层脚本；
3. 检查登录状态，未登录时自动引导登录。

自动化环境建议使用 API Key 认证，避免交互登录：

```bash
devbridge auth login --api-key "$HW_API_KEY"
```

## 使用示例

### 快速创建隧道并托管

告诉代理：

> 帮我把本地 8080 端口暴露出去

代理会自动执行以下流程：

1. 安装 CLI 并检查认证状态；
2. 创建隧道（名称 `dev-tunnel-<random>`，有效期 8 小时）；
3. 添加端口 8080，协议 `http`，允许匿名访问；
4. 后台启动 Host 托管服务。

完成后返回隧道 ID 和访问地址。

### 指定参数

> 创建一个叫 frontend 的隧道，暴露 3000 端口，有效期 24 小时，禁止匿名访问

代理会使用你指定的参数，未指定的使用默认值，不会逐个确认。

### 默认参数

| 参数     | 默认值                |
| -------- | --------------------- |
| 隧道名称 | `dev-tunnel-<random>` |
| 描述     | `开发隧道`            |
| 有效期   | 8 小时                |
| 端口     | 8080                  |
| 协议     | `http`                |
| 匿名访问 | 允许                  |

## 清理

使用完成后，停止 Host 进程并删除隧道：

> 删掉刚才创建的隧道

代理会执行 `db_delete <tunnelId>` 删除隧道及其端口配置。

## 相关内容

- [CLI 命令参考](./cli.md)
- [安装 DevBridge CLI](../guide/install.md)
- [登录与凭证](../guide/authentication.md)
- [问题排查](./troubleshooting.md)
