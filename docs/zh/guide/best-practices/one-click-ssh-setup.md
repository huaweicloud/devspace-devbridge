---
title: 远端服务器一键配置 DevBridge SSH 隧道
description: 在华为云 ECS 上用一条命令完成 DevBridge CLI 安装、SSH 隧道创建与开机自启，无需开放任何入站端口。
---

# 远端服务器一键配置 DevBridge SSH 隧道

<p class="lead">在远端 ECS 上执行一条命令，自动安装 DevBridge CLI、暴露 SSH 端口 22 并配置开机自启，之后可从其他设备通过隧道登录，无需开放公网入站端口。</p>

## 场景与目的

手工配置 SSH 隧道要在服务器上安装 CLI、登录 DevBridge、创建隧道、托管端口、管理凭证与开机自启，步骤多且容易遗漏。本实践提供的一键脚本可在 Linux 或 Windows ECS 上自动完成：

- 安装 DevBridge CLI，并准备 SSH 服务（Windows 自动安装 OpenSSH Server，Linux 使用系统自带）。
- 使用 API Key 登录、创建隧道并暴露 SSH 端口 22。
- 默认加密存储 API Key（AES-256-GCM + PBKDF2-SHA256），也支持明文模式。
- 配置开机自启（Linux 用 systemd，Windows 用计划任务），并在到期前自动续期。
- 启动隧道并验证，输出隧道 ID 与从其他设备连接的方式。

服务器只需出方向访问 TCP 443，安全组入方向不需要任何规则。适合新购 ECS 的快速初始化、临时运维服务器，以及不希望向公网暴露 22 端口的场景。

## 开始实操

安装步骤、参数说明与验证方法见仓库中的一键安装指南：

[DevBridge SSH 隧道一键安装指南](https://github.com/huaweicloud/devspace-devbridge/blob/master/setup/%E4%B8%80%E9%94%AE%E5%AE%89%E8%A3%85%E9%A2%84%E5%90%AF%E5%8A%A8%E6%8C%87%E5%AF%BC.md)
