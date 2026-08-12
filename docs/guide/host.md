---
title: Host：托管本地服务
description: 使用 Host 模式把一个或多个本地端口连接到 DevBridge。
---

# Host：托管本地服务

<p class="lead">Host 运行在服务所在设备，通过出站连接把本地端口注册到 DevBridge。</p>

## 托管已有隧道

先确认本地服务正在监听：

```bash
python3 -m http.server 8080
```

确认对应端口已经配置在同一条隧道上，通过该命令查询：

```bash
devbridge port list <tunnelId>
```

然后托管已创建隧道的端口：

```bash
devbridge host <tunnelId>
```

Host 会自动申请新的 Host 令牌，加载隧道中的全部端口配置，连接中继服务并开始转发。命令在前台持续运行，按 `Ctrl+C` 停止。

## 创建并立即托管

没有可复用隧道时，可以让 Host 创建一条临时使用的隧道：

```bash
devbridge host -p 8080 -d "隧道描述信息" -e 8
```

这里的 `-d` 和 `-e` 属于新建隧道：分别设置隧道描述和有效期小时数。它们不是端口参数。

需要稳定地址、多个端口或明确访问策略时，建议先使用 `devbridge create` 和
`devbridge port create` 完成配置，再启动 Host。

## 访问隧道地址

Host 成功后会输出隧道 ID 和访问地址，形如：

```text
https://<tunnelId>.<clusterId>.myhuaweicloud.com
```

除了通过 Connect 建立本地映射访问外，也可以直接在浏览器中打开该地址访问托管的服务。访问行为取决于端口的匿名访问策略：

| 端口策略             | 浏览器访问行为                                      |
| -------------------- | --------------------------------------------------- |
| 允许匿名访问（`-a`） | 直接打开地址即可访问，不需要 DevBridge 身份或凭证。 |
| 禁止匿名访问（默认） | 跳转到登录页，完成认证获取凭证后即可访问。          |

::: tip 两种访问方式

- **直接访问隧道地址**：在任意设备浏览器中打开 `https://<tunnelId>.<clusterId>.myhuaweicloud.com`，适合快速分享和验证。
- **通过 Connect 访问**：在另一台设备运行 `devbridge connect <tunnelId>`，再访问 `http://localhost:<port>`，适合需要本地端口映射的场景。

两种方式等价，都能访问 Host 托管的服务。详见[Connect：连接远程服务](./connect.md)。
:::

端口的匿名访问策略在创建端口时通过 `-a` 或 `--deny-anonymous` 指定，详见[管理端口](./ports.md)。

## 自动重连

Host 在网络短暂中断后自动尝试重连。以下状态需要人工处理：

- 登录凭证已失效；
- 隧道已经过期或删除；
- Host 令牌无法重新签发；
- 本地端口没有服务监听；
- 当前网络无法连接 DevBridge。

自动重连不会重新创建已删除的隧道，也不会绕过端口访问策略。

## 停止 Host

在 Host 终端按 `Ctrl+C`。停止 Host 只结束当前转发：

- 持久隧道和端口配置仍然保留；
- Connect 端无法继续访问没有 Host 的端口；
- 再次执行 Host 命令即可恢复托管。

不再使用的隧道应通过 `devbridge delete <tunnelId>` 删除。

## 安全建议

- 只托管本次联调需要的端口；
- 托管前确认端口的匿名访问策略；
- 不在 Host 输出中打印完整令牌；
- 不使用管理员或 root 身份运行普通开发服务；
- 完成联调后停止 Host。

## 下一步

- [从另一台设备连接隧道](./connect.md)
- [排查 Host 连接问题](../reference/troubleshooting.md)
