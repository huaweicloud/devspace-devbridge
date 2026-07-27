# DevBridge tunnel documentation

面向 DevBridge 开发隧道使用者的文档站点。正文使用 Markdown 编写，由
[VitePress](https://vitepress.dev/) 生成并发布到 GitHub Pages。

在线文档：<https://huaweicloud.github.io/devspace-devbridge/>

## Local development

需要 Node.js 24。

```bash
npm ci
npm run docs:dev
```

本地开发地址是 `http://127.0.0.1:5173/devspace-devbridge/`。

提交前执行完整检查：

```bash
npm run check
```

生产版本可以这样预览：

```bash
npm run docs:build
npm run docs:preview
```

## Project structure

```text
docs/
├── index.md                 # 快速入门
├── guide/                   # 按用户任务组织的指南
├── reference/               # CLI、配置和 API 参考
├── public/                  # 原样发布的静态文件
└── .vitepress/
    ├── config.mjs           # 导航、侧栏、搜索和站点信息
    └── theme/               # 小范围主题定制
```

每个主题独立成页。新增页面时：

1. 在 `docs/guide` 或 `docs/reference` 中新增 Markdown 文件。
2. 在 `docs/.vitepress/config.mjs` 的 `sidebar` 中加入入口。
3. 使用相对链接连接相关主题。
4. 运行 `npm run check`。

不要直接编辑 `docs/.vitepress/dist`，它是构建产物且不会提交。

## Dependency policy

依赖版本和安装脚本许可都固定在 `package.json` 与 `package-lock.json` 中。
VitePress 1.6.4 默认依赖的 Vite 5 已存在安全告警，因此项目将 Vite 覆盖为兼容的
`6.4.3`。调整或删除该覆盖前，必须同时通过 `npm run check` 和 `npm run audit`。

## Content ownership

- CLI 命令以 DevBridge CLI 的实际行为为准。
- 隧道、端口、Host 和 Connect 的说明以 DevBridge 已发布能力为准。
- 修改相关业务后，应在同一个变更中更新对应 Markdown 页面。

## Publishing

推送到 `main` 后，GitHub Actions 会校验 Markdown、构建 VitePress 并发布
`docs/.vitepress/dist`。GitHub Pages 的 Source 必须设置为 **GitHub Actions**。
