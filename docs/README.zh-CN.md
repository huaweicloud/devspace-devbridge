# DevBridge 文档站点

> English: [README.md](README.md)

本目录是 DevBridge 文档站点的源码，由 [VitePress](https://vitepress.dev/) 构建并发布到
GitHub Pages：<https://huaweicloud.github.io/devspace-devbridge/>

## 本地预览

开发需要 Node.js 24（在 `docs/` 目录下执行）：

```bash
npm ci
npm run docs:dev
```

本地开发地址是 `http://127.0.0.1:5173/devspace-devbridge/`。

提交前执行：

```bash
npm run check
npm run audit
```

## 新增页面

1. 在 `guide/` 或 `reference/` 中新增 Markdown 文件。
2. 在 `.vitepress/config.mjs` 的 `sidebar` 中加入入口。
3. 使用相对链接连接相关主题。
4. 运行 `npm run check`。

不要直接编辑 `.vitepress/dist`，它是构建产物且不会提交。

## Dependency policy

文档依赖固定在 `package.json` 与 `package-lock.json` 中。调整依赖和 Vite 覆盖版本时，必须同时通过
`npm run check` 和 `npm run audit`。

## Content ownership

- CLI 命令以 DevBridge CLI 的实际行为为准。
- 隧道、端口、Host 和 Connect 的说明以 DevBridge 已发布能力为准。
- 修改相关业务后，应在同一个变更中更新对应 Markdown 页面。

## Publishing

推送到 `master` 或 `develop` 后，GitHub Actions 会在 `docs/` 中校验 Markdown、构建 VitePress，并发布
`.vitepress/dist`。GitHub Pages 的 Source 必须设置为 **GitHub Actions**。
