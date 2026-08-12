---
title: 托管与公网访问
description: 启动本地服务并通过 Host 托管，在浏览器中直接访问隧道地址。
---

# 托管与公网访问

<p class="lead">在本地启动服务，使用 Host 托管端口生成公网访问地址，直接在浏览器中打开地址即可访问。</p>

## 场景说明

目标：在本地启动一个 HTTP 服务，通过 DevBridge 隧道托管，获得公网访问地址后在浏览器中直接访问。

无需另一台设备安装 CLI 或建立 Connect 连接，适合快速分享本地开发内容。

## 第一步：启动本地服务

在终端 1 启动一个监听 `8080` 端口的 HTTP 服务：

```bash
python3 -m http.server 8080
```

启动后该终端会持续运行，服务默认监听 `0.0.0.0:8080`。保持此终端不要关闭。

![启动本地 HTTP 服务](/images/host-connect/python-http-server.png)

验证服务可用：

```bash
curl http://127.0.0.1:8080
```

能返回当前目录的文件列表即表示服务正常。

## 第二步：启动 Host 托管端口

Host 有两种方式，根据是否复用已有隧道选择。

### 方式一：使用临时隧道

没有可复用隧道时，让 Host 自动创建一条临时隧道并托管 `8080` 端口：

```bash
devbridge host -p 8080
```

如需设置描述和有效期：

```bash
devbridge host -p 8080 -d "Temporary preview" -e 8
```

- `-p 8080`：托管本地 `8080` 端口。
- `-d`：隧道描述，可选。
- `-e`：有效期小时数，可选，默认 72 小时。

Host 成功后会输出隧道 ID 和访问地址，形如：

```text
https://<tunnelId>.<clusterId>.myhuaweicloud.com
```

![使用临时隧道启动 Host](/images/host-connect/host-temporary-tunnel.png)

上图展示的是 `devbridge host -p 8080` 的实际输出：自动创建隧道 `yf7oqwea` 并托管 `8080` 端口，输出对应的 Tunnel URL。

::: tip 临时隧道生命周期
`devbridge host -p 8080` 创建的隧道在停止 Host 后仍然保留，不会自动删除。不再使用时应通过
`devbridge delete <tunnelId>` 清理。
:::

### 方式二：使用已有隧道

复用已有隧道前，必须确认该隧道存在，且已配置 `8080` 端口。

1. 列出当前工作空间的有效隧道：

   ```bash
   devbridge list
   ```

   确认目标隧道 ID 在列表中。

2. 查看隧道的端口配置：

   ```bash
   devbridge port list <tunnelId>
   ```

   确认 `8080` 端口已经存在。如果不存在，先创建：

   ```bash
   devbridge port create <tunnelId> -p 8080 --protocol http
   ```

3. 启动 Host 托管该隧道的全部端口：

   ```bash
   devbridge host <tunnelId>
   ```

   Host 会加载隧道中配置的全部端口并开始转发，不需要再指定 `-p`。

   ![使用已有隧道启动 Host](/images/host-connect/host-existing-tunnel.png)

   上图展示的是 `devbridge host ghirszae` 的实际输出：Host 成功托管 `8080` 端口并输出隧道地址，运行过程中触发了一次自动重连，恢复后继续转发。

无论哪种方式，Host 都在前台持续运行。保持终端 2 不要关闭，按 `Ctrl+C` 可停止托管。

## 第三步：在浏览器中访问隧道地址

Host 输出的隧道地址形如：

```text
https://<tunnelId>.<clusterId>.myhuaweicloud.com
```

在任意设备的浏览器中直接打开该地址即可访问本地服务。访问行为取决于端口的匿名访问策略：

| 端口策略             | 浏览器访问行为                                      |
| -------------------- | --------------------------------------------------- |
| 允许匿名访问（`-a`） | 直接打开地址即可访问，不需要 DevBridge 身份或凭证。 |
| 禁止匿名访问（默认） | 跳转到登录页，完成认证获取凭证后即可访问。          |

端口的匿名访问策略在创建端口时通过 `-a` 或 `--deny-anonymous` 指定，详见[管理端口](../ports.md)。

## 完整流程速查

| 步骤 | 终端   | 命令                                               | 说明                     |
| ---- | ------ | -------------------------------------------------- | ------------------------ |
| 1    | 终端 1 | `python3 -m http.server 8080`                      | 启动本地 HTTP 服务。     |
| 2    | 终端 2 | `devbridge host -p 8080`                           | 临时隧道托管（方式一）。 |
| 2    | 终端 2 | `devbridge host <tunnelId>`                        | 已有隧道托管（方式二）。 |
| 3    | 浏览器 | `https://<tunnelId>.<clusterId>.myhuaweicloud.com` | 直接访问隧道地址。       |

## 常见问题

### Host 启动后立即退出

确认终端 1 的本地服务仍在运行。Host 检测到本地端口无服务监听时会报错退出。

### 浏览器访问返回登录页

端口配置为禁止匿名访问时，浏览器会跳转到登录页。完成认证后即可访问，或创建端口时使用 `-a` 允许匿名访问。

### 隧道 ID 遗忘

查看：

```bash
devbridge list
```

临时隧道和持久隧道都会出现在有效隧道列表中。

## 清理

完成验证后，按顺序停止：

1. 在 Host 终端按 `Ctrl+C`；
2. 在服务终端按 `Ctrl+C` 停止 `python3 -m http.server`。

临时隧道停止 Host 后不会自动删除。确认不再使用后清理：

```bash
devbridge delete <tunnelId>
```

## 相关内容

- [托管与远程连接](./host-remote-connect.md)
- [Host：托管本地服务](../host.md)
- [管理端口](../ports.md)
- [什么是开发隧道](../overview.md)
- [排查 Host 连接问题](../../reference/troubleshooting.md)
