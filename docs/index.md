---
title: DevBridge 开发隧道
description: DevBridge 开发隧道的入口：五分钟上手、核心概念与命令参考。
---

# DevBridge 开发隧道

<p class="lead">使用 DevBridge 将本地开发服务安全地开放给远程设备，无需公网入站端口，并控制端口的访问方式。</p>

::: warning 开发者隧道服务升级通知
DevBridge 隧道域名将于 2026年9月20日（周日）晚间10点后切换：原 `*.cn-north-4-bridge.myhuaweicloud.com` 变更为 `*.devbridge-s2.hwtunnel.com`。切换后，旧版本端侧将无法连接，原域名下创建的隧道也会失效。为避免影响正常使用，请在变更后及时升级到最新版客户端，并重新创建隧道。新隧道访问地址格式为：`https://{隧道ID}-{端口}.devbridge-s2.hwtunnel.com`。如有疑问，请及时在 [DevBridge 团队 Issue](https://github.com/huaweicloud/devspace-devbridge/issues) 反馈。
:::

## 从这里开始

- [五分钟创建并托管第一条隧道](./guide/quickstart.md) —— 安装、登录、托管、连接的完整流程
- [了解开发隧道的工作方式](./guide/overview.md) —— 隧道、端口、Host 与 Connect 的关系，含视频介绍
- [CLI 命令参考](./reference/cli.md) —— 全部命令与参数说明

## 常用页面

- [管理隧道](./guide/tunnels.md)
- [管理端口](./guide/ports.md)
- [Host：托管本地服务](./guide/host.md)
- [Connect：连接远程服务](./guide/connect.md)
- [最佳实践](./guide/best-practices/host-public-access.md)
- [问题排查](./reference/troubleshooting.md)
