---
title: 管理端口
description: 配置隧道端口、协议和匿名访问策略。
---

# 管理端口

<p class="lead">隧道只转发已经配置的端口。每个端口拥有独立的协议和匿名访问策略。</p>

## 创建端口

创建一个禁止匿名访问的 HTTP 端口（`--deny-anonymous` 可选，默认禁止匿名访问）：

```bash
devbridge port create <tunnelId> -p 8080 --protocol http --deny-anonymous
```

创建一个允许匿名访问的 HTTP 端口：

```bash
devbridge port create <tunnelId> -p 8080 --protocol http -a
```

创建一个使用自动协议识别的端口：

```bash
devbridge port create <tunnelId> -p 3000 --protocol auto
```

创建时必须提供端口和协议。端口范围为 `1` 到 `65535`，同一条隧道内不能重复。

## 选择协议

| 协议    | 使用场景                 |
| ------- | ------------------------ |
| `http`  | 本地服务接收普通 HTTP。  |
| `https` | 本地服务自身提供 HTTPS。 |
| `auto`  | 由连接端自动识别协议。   |

协议描述的是 Host 侧本地服务。选择错误可能导致握手失败、连接中断或返回不可识别的数据。

## 控制匿名访问

`-a`、`--allow-anonymous` 表示该端口可匿名访问：

```bash
devbridge port create <tunnelId> -p 8080 --protocol http -a
```

`--deny-anonymous` 表示该端口必须经过身份认证：

```bash
devbridge port create <tunnelId> -p 8080 --protocol http --deny-anonymous
```

以下服务应始终禁止匿名访问：

- 管理后台；
- 调试和诊断端点；
- 包含用户数据的服务；
- 具备数据修改能力的接口。

::: warning
匿名访问意味着持有隧道地址的访问者不需要 DevBridge 身份即可访问端口。只对明确需要公开的开发内容启用。
:::

## 列出端口

```bash
devbridge port list <tunnelId>
```

如果已经设置默认隧道，可以省略隧道 ID：

```bash
devbridge port list
```

## 查看端口

```bash
devbridge port show <tunnelId> -p 8080
```

结果包含端口、协议和匿名访问状态。

## 更新端口

只更新协议：

```bash
devbridge port update <tunnelId> -p 8080 --protocol https
```

将端口改为禁止匿名访问：

```bash
devbridge port update <tunnelId> -p 8080 --deny-anonymous
```

将端口改为允许匿名访问：

```bash
devbridge port update <tunnelId> -p 8080 -a
```

更新时，`--protocol` 和匿名访问选项不是必填；未指定的字段保持不变。端口不支持描述字段，因此端口命令中没有 `-d`。

## 删除端口

```bash
devbridge port delete <tunnelId> -p 8080
```

端口没有公开的批量删除命令。需要移除整条隧道及其端口时，删除隧道。

## 下一步

- [使用 Host 托管本地端口](./host.md)
- [使用 Connect 建立本地映射](./connect.md)
