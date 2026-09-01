# DevBridge

DevBridge 使用独立子目录组织文档和其他项目，避免不同工具链相互影响。

在线文档：<https://huaweicloud.github.io/devspace-devbridge/>

## Repository structure

```text
.
├── .github/              # 仓库级 CI、发布和同步工作流
├── docs/                 # VitePress 文档站点及其 Node.js 工具链
└── README.md             # 仓库总览
```

后续项目应各自使用独立的首层目录，并将构建配置和依赖文件保存在对应目录中。

## Documentation

文档正文使用 Markdown 编写，由 [VitePress](https://vitepress.dev/) 生成并发布到
GitHub Pages。开发文档需要 Node.js 24：

```bash
cd docs
npm ci
npm run docs:dev
```

本地开发地址是 `http://127.0.0.1:5173/devspace-devbridge/`。

提交文档前执行：

```bash
cd docs
npm run check
npm run audit
```

文档目录结构：

```text
docs/
├── index.md                 # 快速入门
├── guide/                   # 按用户任务组织的指南
├── reference/               # CLI、配置和 API 参考
├── public/                  # 原样发布的静态文件
├── package.json             # 文档构建与校验命令
└── .vitepress/
    ├── config.mjs           # 导航、侧栏、搜索和站点信息
    └── theme/               # 小范围主题定制
```

新增页面时：

1. 在 `docs/guide` 或 `docs/reference` 中新增 Markdown 文件。
2. 在 `docs/.vitepress/config.mjs` 的 `sidebar` 中加入入口。
3. 使用相对链接连接相关主题。
4. 在 `docs/` 中运行 `npm run check`。

不要直接编辑 `docs/.vitepress/dist`，它是构建产物且不会提交。

## Dependency policy

文档依赖固定在 `docs/package.json` 与 `docs/package-lock.json` 中。调整依赖和 Vite
覆盖版本时，必须同时通过 `npm run check` 和 `npm run audit`。

## Content ownership

- CLI 命令以 DevBridge CLI 的实际行为为准。
- 隧道、端口、Host 和 Connect 的说明以 DevBridge 已发布能力为准。
- 修改相关业务后，应在同一个变更中更新对应 Markdown 页面。

## Publishing

推送到 `master` 后，GitHub Actions 会在 `docs/` 中校验 Markdown、构建 VitePress，
并发布 `docs/.vitepress/dist`。GitHub Pages 的 Source 必须设置为 **GitHub Actions**。
