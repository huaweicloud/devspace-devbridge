---
title: One-click DevBridge SSH tunnel on a remote server
description: Set up the DevBridge CLI, an SSH tunnel, and autostart on a Huawei Cloud ECS with one command, without opening any inbound ports.
---

# One-click DevBridge SSH tunnel on a remote server

<p class="lead">Run one command on a remote ECS to install the DevBridge CLI, expose SSH port 22, and enable autostart, then log in from any device through the tunnel without opening public inbound ports.</p>

## Scenario and goal

Setting up an SSH tunnel by hand means installing the CLI, signing in, creating a tunnel, hosting the port, handling credentials, and configuring autostart. This practice provides a one-command script that automates the work on a Linux or Windows ECS:

- Install the DevBridge CLI and an SSH server (OpenSSH Server on Windows).
- Sign in with an API key, create a tunnel, and expose SSH port 22.
- Store the API key encrypted by default (AES-256-GCM + PBKDF2-SHA256), with a plaintext option.
- Configure autostart (systemd on Linux, Task Scheduler on Windows) and renew the tunnel before expiry.
- Start and verify the tunnel, then print the tunnel ID and how to connect from another device.

The server only needs outbound TCP 443; the security group needs no inbound rules. It fits quick ECS initialization, temporary admin hosts, and cases where you don't want port 22 exposed to the internet.

## Start the walkthrough

The installation steps, flags, and verification are in the one-click setup guide in the repository:

[DevBridge SSH tunnel one-click setup guide](https://github.com/huaweicloud/devspace-devbridge/blob/master/setup/%E4%B8%80%E9%94%AE%E5%AE%89%E8%A3%85%E9%A2%84%E5%90%AF%E5%8A%A8%E6%8C%87%E5%AF%BC.md)
