---
title: 管理隧道
description: 创建、查询、更新、选择和删除 DevBridge 隧道。
---

# 管理隧道

<p class="lead">持久隧道适合重复托管、保留稳定地址或预先配置多个端口的场景。</p>

## 创建隧道

```bash
devbridge create frontend -d "隧道描述信息" -e 24
```

参数含义：

| 参数       | 说明                     |
| ---------- | ------------------------ |
| `frontend` | 隧道名称。               |
| `-d`       | 可选描述。               |
| `-e`       | 有效期规格，单位为小时。 |

未指定 `-e` 时使用 72 小时，允许范围为 1 到 720 小时。

创建成功后会返回隧道 ID、名称、描述和过期时间。

一个工作空间默认最多拥有 10 条有效隧道。达到配额后，应复用或删除已有隧道。

## 列出隧道

```bash
devbridge list
```

列表只包含当前工作空间中未删除且未过期的隧道，主要信息包括：

- 隧道 ID 和名称；
- 描述；
- 过期时间；
- 已配置的端口数量。

脚本需要结构化结果时使用 JSON：

```bash
devbridge list -j
```

## 查看详情

```bash
devbridge show <tunnelId>
```

详情用于查看指定隧道的隧道 ID、名称、描述和过期时间。过期或不属于当前工作空间的隧道不会作为有效资源返回。

## 更新隧道

```bash
devbridge update <tunnelId> -n "frontend-v2" -d "隧道描述信息" -e 48
```

只传需要修改的选项。`-e` 仍以小时为单位，最大为 720 小时。成功修改隧道或端口会刷新当前到期时间。

## 设置默认隧道

频繁操作同一条隧道时，可以保存本机默认上下文：

```bash
devbridge set <tunnelId>
```

后续部分命令可以省略隧道 ID：

```bash
devbridge port list
devbridge host -p 8080
```

清除默认上下文：

```bash
devbridge unset
```

默认隧道只保存在本机，不会改变隧道归属，也不会影响其他用户。

## 签发连接令牌

Host 和 Connect 通常会自动获取对应令牌。集成其他客户端时，可以手动签发：

```bash
devbridge token <tunnelId> -s host
devbridge token <tunnelId> -s connect
```

每次执行都会生成一个新令牌。令牌具有固定的短期有效期，不会随隧道有效期改变。

::: warning 保护令牌
令牌只应交给对应的 Host 或 Connect 使用。不要写入日志、URL、代码仓库或长期配置文件。
:::

## 删除隧道

删除一条隧道：

```bash
devbridge delete <tunnelId>
```

删除当前工作空间的全部隧道：

```bash
devbridge delete-all
```

`delete-all` 会影响当前工作空间的所有隧道。删除隧道会同时移除其端口配置，现有 Host 和 Connect 连接也应停止使用该隧道。

## 下一步

- [为隧道配置端口](./ports.md)
- [使用 Host 托管端口](./host.md)
- [查看完整 CLI 参考](../reference/cli.md)
