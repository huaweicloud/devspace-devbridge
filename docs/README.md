# DevBridge Docs Site

> 中文文档：[README.zh-CN.md](README.zh-CN.md)

This directory holds the source of the DevBridge documentation site, built with
[VitePress](https://vitepress.dev/) and published to GitHub Pages:
<https://huaweicloud.github.io/devspace-devbridge/>

## Local preview

Development requires Node.js 24 (run inside `docs/`):

```bash
npm ci
npm run docs:dev
```

The local dev address is `http://127.0.0.1:5173/devspace-devbridge/`.

Before submitting, run:

```bash
npm run check
npm run audit
```

## Adding pages

1. Add a Markdown file under `guide/` or `reference/`.
2. Register an entry in the `sidebar` of `.vitepress/config.mjs`.
3. Use relative links to connect related topics.
4. Run `npm run check`.

Do not edit `.vitepress/dist` directly — it is a build artifact and is not committed.

## Dependency policy

Docs dependencies are pinned in `package.json` and `package-lock.json`. When adjusting
dependencies or Vite override versions, both `npm run check` and `npm run audit` must pass.

## Content ownership

- CLI commands are authoritative against the actual behavior of the DevBridge CLI.
- Tunnel, port, Host and Connect descriptions are authoritative against DevBridge's published
  capabilities.
- After changing related business behavior, update the corresponding Markdown pages in the same
  change.

## Publishing

On push to `master` or `develop`, GitHub Actions lints Markdown in `docs/`, builds VitePress, and
publishes `.vitepress/dist`. GitHub Pages Source must be set to **GitHub Actions**.
