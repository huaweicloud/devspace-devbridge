---
title: DevBridge 开发隧道
description: DevBridge 开发隧道的入口：五分钟上手、核心概念与命令参考。
---

# DevBridge 开发隧道

<p class="lead">使用 DevBridge 将本地开发服务安全地开放给远程设备，无需公网入站端口，并控制端口的访问方式。</p>

::: warning 📢 致用户
DevBridge 计划于 **11 月 19 日（周四）晚间 7 点后** 发布新版本。发布窗口期内，服务可能出现短暂连接中断，敬请提前知悉并做好相应安排。

**一、影响范围**

1. 隧道连接可能短暂断开，系统将自动重连
2. 正在执行的长耗时任务或交互会话可能被中断，需重新发起
3. 访客 URL 可能出现短暂不可达

**二、建议事项**

1. 请避免在发布窗口期内安排关键的长耗时任务
2. 有重要操作需求，请尽量于发布前完成
3. 发布后如遇连接异常，请等待片刻后重连，通常将自动恢复

如有疑问或需协调事项，请[反馈 issue](https://github.com/huaweicloud/devspace-devbridge/issues)。
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
