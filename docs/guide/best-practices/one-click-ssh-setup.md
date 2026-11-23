---
title: Automate DevBridge setup and host an SSH tunnel
description: Automate DevBridge setup and host SSH port 22 through a tunnel on a Huawei Cloud ECS with a single command, without opening any inbound ports.
---

# Automate DevBridge setup and host an SSH tunnel

<p class="lead">A single command on a remote ECS installs and configures DevBridge, hosts SSH port 22 through a tunnel, and enables autostart; you can then log in from another device without opening public inbound ports.</p>

## Scenario and goal

Setting up an SSH tunnel by hand means installing the CLI, signing in, creating a tunnel, hosting the port, handling credentials, and configuring autostart. This practice provides a setup script that automates these steps on a Linux or Windows ECS:

- Install the DevBridge CLI and make sure an SSH server is available (OpenSSH Server is installed on Windows; Linux images ship with one).
- Sign in with an API key, create a tunnel, and host SSH port 22 through it.
- Store the API key encrypted by default (AES-256-GCM + PBKDF2-SHA256), with a plaintext option.
- Configure autostart (systemd on Linux, Task Scheduler on Windows) and renew the tunnel before expiry.
- Start and verify the tunnel, then print the tunnel ID and how to connect from another device.

The server only needs outbound TCP 443; the security group needs no inbound rules. It fits quick ECS initialization, temporary admin hosts, and cases where you don't want port 22 exposed to the internet.

## Start the walkthrough

The installation steps, flags, and verification are in the setup guide in the repository:

[DevBridge setup and SSH tunnel guide](https://github.com/huaweicloud/devspace-devbridge/blob/master/setup/%E4%B8%80%E9%94%AE%E5%AE%89%E8%A3%85%E9%A2%84%E5%90%AF%E5%8A%A8%E6%8C%87%E5%AF%BC.md)
