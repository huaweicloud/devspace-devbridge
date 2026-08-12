---
title: 托管与远程连接
description: 在本地托管服务后，从远端通过 Connect 建立连接并访问。
---

# 托管与远程连接

<p class="lead">在设备 A 上启动服务并托管，在远端设备 B 通过 Connect 建立本地端口映射，以 localhost 访问远程服务。</p>

## 场景说明

目标：在设备 A 上启动一个 HTTP 服务，通过 DevBridge 隧道托管，再从设备 B 连接隧道，在设备 B 上以
`http://localhost:8080` 访问设备 A 的服务。

| 设备   | 角色    | 职责                                 |
| ------ | ------- | ------------------------------------ |
| 设备 A | Host    | 运行本地服务，托管端口到 DevBridge。 |
| 设备 B | Connect | 连接隧道，在本地建立端口映射并访问。 |

两台设备都需要已安装并登录 DevBridge CLI。若尚未安装，参阅[安装 DevBridge CLI](../install.md)。

## 第一步：在设备 A 启动本地服务

在设备 A 的终端 1 启动一个监听 `8080` 端口的 HTTP 服务：

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

## 第二步：在设备 A 启动 Host

Host 有两种方式，根据是否复用已有隧道选择。

### 方式一：使用临时隧道

没有可复用隧道时，让 Host 自动创建一条临时隧道并托管 `8080` 端口：

```bash
devbridge host -p 8080
```

如需设置描述和有效期：

```bash
devbridge host -p 8080 -d "隧道描述信息" -e 8
```

- `-p 8080`：托管本地 `8080` 端口。
- `-d`：隧道描述，可选。
- `-e`：有效期小时数，可选，默认 72 小时。

Host 成功后会输出隧道 ID 和访问地址，形如：

```text
https://<tunnelId>.<clusterId>.myhuaweicloud.com
```

记下这里的 `<tunnelId>`，设备 B 连接时需要使用。

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

## 第三步：在设备 B 启动 Connect

在设备 B 上运行以下命令连接隧道：

```bash
devbridge connect <tunnelId>
```

`<tunnelId>` 替换为第二步中 Host 输出的隧道 ID。

Connect 会读取隧道端口配置、申请 Connect 令牌并建立本地端口映射。连接成功后，设备 B 的
`8080` 端口会映射到设备 A 上的服务。

Connect 同样在前台持续运行，保持终端不要关闭，按 `Ctrl+C` 可停止连接。

![Connect 连接隧道](/images/host-connect/connect-tunnel.png)

上图展示的是 `devbridge connect ghirszae` 的实际输出：连接成功后显示 `Forwarding localhost:8080 -> tunnel port: 8080`，即设备 B 的本地 `8080` 端口已映射到隧道端口。

## 第四步：在设备 B 访问远程服务

连接建立后，在设备 B 上通过本机端口访问设备 A 的服务：

```bash
curl http://127.0.0.1:8080
```

![通过 curl 访问远程服务](/images/host-connect/curl-access.png)

返回的 HTML 目录列表即设备 A 上 `python3 -m http.server 8080` 服务的响应，说明设备 B 已通过本地端口映射成功访问到设备 A 的服务。

也可以在浏览器中打开：

```text
http://localhost:8080
```

或使用 `127.0.0.1`：

```text
http://127.0.0.1:8080
```

两者等价，都指向设备 B 本地建立的端口映射，再由 DevBridge 转发到设备 A 的 `8080` 服务。

## 完整流程速查

| 步骤 | 设备   | 终端   | 命令                           | 说明                     |
| ---- | ------ | ------ | ------------------------------ | ------------------------ |
| 1    | 设备 A | 终端 1 | `python3 -m http.server 8080`  | 启动本地 HTTP 服务。     |
| 2    | 设备 A | 终端 2 | `devbridge host -p 8080`       | 临时隧道托管（方式一）。 |
| 2    | 设备 A | 终端 2 | `devbridge host <tunnelId>`    | 已有隧道托管（方式二）。 |
| 3    | 设备 B | 终端   | `devbridge connect <tunnelId>` | 连接隧道建立本地映射。   |
| 4    | 设备 B | 浏览器 | `http://localhost:8080`        | 访问远程服务。           |

## 常见问题

### Host 启动后立即退出

确认终端 1 的本地服务仍在运行。Host 检测到本地端口无服务监听时会报错退出。

### Connect 成功但访问返回空或超时

依次检查：

1. 设备 A 的 Host 进程仍在运行；
2. 设备 A 的本地服务仍可访问：`curl http://127.0.0.1:8080`；
3. 设备 B 的 `8080` 端口没有被其他进程占用；
4. 两端使用的是同一个隧道 ID。

### 端口被占用

设备 B 上若 `8080` 已被其他进程占用，Connect 无法建立映射。先释放该端口，或修改本地服务监听端口后重新走一遍流程。

### 隧道 ID 遗忘

在设备 A 上查看：

```bash
devbridge list
```

临时隧道和持久隧道都会出现在有效隧道列表中。

## 清理

完成验证后，按顺序停止：

1. 设备 B：在 Connect 终端按 `Ctrl+C`；
2. 设备 A：在 Host 终端按 `Ctrl+C`；
3. 设备 A：在服务终端按 `Ctrl+C` 停止 `python3 -m http.server`。

临时隧道停止 Host 后不会自动删除。确认不再使用后清理：

```bash
devbridge delete <tunnelId>
```

## 相关内容

- [托管与公网访问](./host-public-access.md)
- [Host：托管本地服务](../host.md)
- [Connect：连接远程服务](../connect.md)
- [什么是开发隧道](../overview.md)
- [排查 Host 与 Connect 问题](../../reference/troubleshooting.md)
