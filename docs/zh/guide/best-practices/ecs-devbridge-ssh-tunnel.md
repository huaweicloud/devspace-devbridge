---
title: ECS 零入站端口 SSH 隧道
description: 在华为云购买 ECS 并通过 DevBridge 出站隧道完成 SSH 登录与文件传输，安全组入方向无需开放任何端口。
---

# ECS 零入站端口 SSH 隧道

<p class="lead">在 AI DevSpace 的 AI Shell 中用自然语言购买一台华为云 ECS，安全组入方向保持 0 条规则，再通过 DevBridge 出站隧道登录 SSH 并传输文件。</p>

## 场景与目的

传统 SSH 需要在安全组放行 22 端口，ECS 一旦暴露公网地址，就会持续承受端口扫描和暴力破解。本实践把入方向完全关闭：

- ECS 安全组入方向 0 条规则，公网无法直连任何端口。
- EIP 仅用于出站，ECS 主动以 WebSocket 连接 DevBridge 中继。
- SSH、scp 流量经 DevBridge 隧道转发，开发者本机也不直连 ECS。

环境由 AI Shell 根据自然语言指令生成 Terraform 并部署，适合需要远程运维 ECS、但不想开放入站端口或维护跳板机、VPN 的场景。

## 实践内容

| 项   | 说明                                                                                      |
| ---- | ----------------------------------------------------------------------------------------- |
| 目标 | 安全组入方向全关的 ECS，通过 DevBridge 隧道完成 SSH 登录与 scp 文件传输                   |
| 产出 | 1 台 ECS（VPC、子网、EIP、FunctionGraph、IAM 委托）和 1 条 DevBridge 出站隧道             |
| 耗时 | 约 30–45 分钟                                                                             |
| 环境 | AI DevSpace 的 AI Shell（内置 Terraform、hcloud CLI、DevBridge CLI），默认区域 cn-north-4 |
| 版本 | 适用 DevBridge 0.3.0-release（x86-64 / ARM64）                                            |

实践中还会用到 FunctionGraph 与 IAM 委托：DevBridge API Key 存放在函数中，ECS 通过临时凭证取回，避免写入 user_data 或 Terraform state。

## 实操步骤概览

1. 检查华为云凭证与 DevBridge API Key 是否可用。
2. 在 AI Shell 中描述需求，由 Agent 生成 Terraform 配置。
3. 确认计划并部署 ECS、EIP、安全组等资源。
4. 等待 cloud-init 安装 DevBridge、创建隧道并托管 22 端口。
5. 通过隧道 SSH 登录、scp 传输文件，并验证入方向确实不可达。
6. 执行 destroy，自动清理资源与隧道。

## 开始实操

完整步骤、Terraform 代码和验证命令见华为云 AI for IaC 实操案例：

[AI Shell 购买 ECS + DevBridge 零入站端口 SSH 隧道](https://aiiac.developer.huaweicloud.com/cases.html#iac-practice/ecs-devbridge-ssh-tunnel)

::: tip
该实操页面为中文，包含可直接复制的自然语言指令与 Terraform 配置。
:::

## 相关内容

- [托管与公网访问](./host-public-access.md)
- [托管与远程连接](./host-remote-connect.md)
- [Host：托管本地服务](../host.md)
- [Connect：连接远程服务](../connect.md)
