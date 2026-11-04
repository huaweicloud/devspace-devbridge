---
title: AI Agent Skill
description: Use the huawei-cloud-devbridge-tunnel skill to let an AI agent manage DevBridge automatically.
---

# AI Agent Skill

<p class="lead"><code>huawei-cloud-devbridge-tunnel</code> is an AI agent skill that wraps all of the DevBridge CLI's tunnel operations, letting an agent automatically install, authenticate, create tunnels, and host services.</p>

## Use cases

- Have an AI agent expose a local service for you in one step, without running CLI commands one by one.
- Create, query, update, and delete tunnels in natural language during a conversation.
- Handle CLI version differences automatically, so you don't have to worry about compatibility issues such as renamed flags.

## Install the skill

### Install via the skill manager

```bash
npx skills add https://gitcode.com/huaweicloud/huaweicloud-skills.git#master \
  --skill huawei-cloud-devbridge-tunnel -y
```

After installation, the skill lives at `~/.agents/skills/huawei-cloud-devbridge-tunnel`.

### Install from the skill catalog

An agent that has `huawei-cloud-find-skills` installed can search by keyword and install. Because this skill isn't yet indexed, search by the skill name directly.

## What the skill can do

The skill wraps the following operations, all executed through adapter functions that automatically accommodate different CLI versions:

| Adapter function | CLI command             | Purpose                    |
| ---------------- | ----------------------- | -------------------------- |
| `db_init`        | —                       | Verify the CLI is usable   |
| `db_create`      | `devbridge create`      | Create a tunnel            |
| `db_port_create` | `devbridge port create` | Create a port              |
| `db_host`        | `devbridge host`        | Host local services        |
| `db_connect`     | `devbridge connect`     | Connect to a remote tunnel |
| `db_list`        | `devbridge list`        | List tunnels               |
| `db_show`        | `devbridge show`        | View tunnel details        |
| `db_update`      | `devbridge update`      | Update a tunnel            |
| `db_delete`      | `devbridge delete`      | Delete a tunnel            |
| `db_port_list`   | `devbridge port list`   | List ports                 |
| `db_port_delete` | `devbridge port delete` | Delete a port              |
| `db_version`     | `devbridge version`     | View the CLI version       |
| `db_auth_status` | `devbridge auth status` | View login status          |
| `db_auth_login`  | `devbridge auth login`  | Log in                     |

## Adapter layer

The DevBridge CLI updates itself automatically on each use, which may rename flags or change parameters. The skill solves this with the `scripts/devbridge_cmd.sh` adapter layer:

1. Before running, it dynamically discovers the flags supported by the current version via `devbridge <command> --help`.
2. After running, it parses the error and automatically recovers from failures caused by renamed flags, new required parameters, or renamed commands.
3. When automatic recovery fails, it returns the error and a suggested command to the agent.

::: tip Why not call devbridge directly
All tunnel, port, Host, and Connect operations must go through the `db_*` adapter functions, not call `devbridge` directly. Only `devbridge auth` and `devbridge version` are exempt.
:::

## Prerequisites

On first use, the skill completes the following preparation automatically:

1. Installs or updates the DevBridge CLI to the latest version.
2. Loads the adapter script.
3. Checks the login status and prompts for login when signed out.

For automated environments, use API key authentication to avoid interactive login:

```bash
devbridge auth login --api-key "$HW_API_KEY"
```

## Usage examples

### Quickly create and host a tunnel

Tell the agent:

> Expose my local port 8080

The agent automatically runs through the following flow:

1. Installs the CLI and checks the authentication status.
2. Creates a tunnel with the name `dev-tunnel-<random>` and an 8-hour expiration.
3. Adds port 8080 with the `http` protocol and allows anonymous access.
4. Starts Host in the background to host the service.

When done, it returns the tunnel ID and access URL.

### Specify parameters

> Create a tunnel named frontend, expose port 3000, expiration 24 hours, and deny anonymous access

The agent uses the parameters you specify and defaults for the rest, without confirming one by one.

### Default parameters

| Parameter        | Default               |
| ---------------- | --------------------- |
| Tunnel name      | `dev-tunnel-<random>` |
| Description      | DevBridge             |
| Expiration       | 8 hours               |
| Port             | 8080                  |
| Protocol         | `http`                |
| Anonymous access | Allowed               |

## Cleanup

When done, stop the Host process and delete the tunnel:

> Delete the tunnel I just created

The agent runs `db_delete <tunnelId>` to delete the tunnel and its port configuration.

## Related content

- [CLI command reference](../reference/cli.md)
- [Install the DevBridge CLI](../guide/install.md)
- [Login and credentials](../guide/authentication.md)
- [Troubleshooting](../reference/troubleshooting.md)
