# DevBridge Tunnel Documentation

面向 DevBridge 开发隧道使用者的 GitHub Pages 文档。主文档按 CLI 用户任务组织，覆盖安装、登录、隧道与端口管理、Host 和 Connect 远程端口转发。

## Local preview

页面不需要构建步骤，直接打开 `index.html` 即可。需要模拟站点路径时，可运行：

```bash
python3 -m http.server 4173
```

然后访问 `http://localhost:4173/`。

## Publishing

推送到 `main` 后，`.github/workflows/pages.yml` 会发布仓库根目录。仓库首次发布前，需要在 GitHub 的 **Settings > Pages > Build and deployment** 中将 Source 设为 **GitHub Actions**。

## Source of truth

控制面字段和路径以 Relay Controller 仓库的 `src/main/resources/static/openapi.yaml` 为准。CLI 命令以 Guacamole CLI 实现为准。修改 CLI、隧道业务或 Gateway 访问协议后，应同步本仓库的快速入门；修改 API 后还应同步 `openapi.yaml`。
