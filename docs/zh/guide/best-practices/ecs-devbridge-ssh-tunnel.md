---
title: ECS 零入站端口 SSH 隧道
description: 在华为云购买 ECS 并通过 DevBridge 出站隧道完成 SSH 登录与文件传输，安全组入方向无需开放任何端口。
---

# ECS 零入站端口 SSH 隧道

<p class="lead">在 AI DevSpace 的 AI Shell 中用自然语言购买一台华为云 ECS，安全组入方向保持 0 条规则，再通过 DevBridge 出站隧道登录 SSH 并传输文件。</p>

## 场景与目的

传统 SSH 需要在安全组放行 22 端口；ECS 绑定公网 IP 后，22 端口会持续承受端口扫描和暴力破解。本实践把入方向完全关闭：

- ECS 安全组入方向 0 条规则，公网无法直连任何端口。
- EIP 仅用于出站，ECS 主动以 WebSocket 连接 DevBridge 中继。
- SSH、scp 流量经 DevBridge 隧道转发，开发者本机也不直连 ECS。

环境由 AI Shell 根据自然语言指令生成 Terraform 配置并完成部署，适合需要远程运维 ECS、但不想开放入站端口，也不想维护跳板机或 VPN 的场景。

## 开始实操

完整步骤、Terraform 代码和验证命令见华为云 AI for IaC 实操案例：

[AI Shell 购买 ECS + DevBridge 零入站端口 SSH 隧道](https://aiiac.developer.huaweicloud.com/cases.html#iac-practice/ecs-devbridge-ssh-tunnel)
