---
title: DevBridge 隧道接收 GitCode Webhook 回调
description: 在华为云 AI 开发者空间开发 Webhook 服务端，通过 DevBridge 隧道暴露公网地址并校验 GitCode 密码，接收仓库事件回调。
---

# DevBridge 隧道接收 GitCode Webhook 回调

<p class="lead">在 AI 开发者空间中用码道代码智能体或 AI Shell 开发一个 Webhook 接收服务端，经 DevBridge 隧道获得公网 HTTPS 地址，再在 GitCode 仓库配置回调，把推送、议题等事件安全地送到云端。</p>

## 场景与目的

实践在华为云 AI 开发者空间体验版中完成，全程对话驱动、零手动编码，大约 40 分钟：

- 开发并启动监听 8080 端口的 Webhook 服务端，记录 HTTP POST 的完整载荷。
- 校验请求头 `X-GitCode-Token`，只有携带正确密码的回调才会被处理，其余请求返回 401。
- 通过 DevBridge 开发隧道把服务端暴露为公网 HTTPS 地址，无需购买公网 IP。
- 在 GitCode 仓库配置 Webhook 的回调 URL 与密码，把推送、议题等仓库事件回调到隧道地址。
- 触发事件后查看服务端日志，确认通知已送达；体验结束后清理隧道、服务进程与 Webhook 配置。

这条路径适合需要调试 Webhook 回调、又不想为回调地址准备公网 IP 或开放入站端口的开发者，同时演示了用密码请求头实现非匿名访问控制的常见做法。

## 开始实操

完整步骤、提示词与验证方式见华为云 AI 云原生开发实践中心：

[DevBridge 隧道接收 GitCode Webhook 回调](https://aidevbp.developer.huaweicloud.com/practices/p_f92a1ddd)
