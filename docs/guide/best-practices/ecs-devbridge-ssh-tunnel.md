---
title: Zero-inbound-port SSH tunnel on ECS
description: Buy a Huawei Cloud ECS and use an outbound DevBridge tunnel for SSH login and file transfer, with zero inbound security group rules.
---

# Zero-inbound-port SSH tunnel on ECS

<p class="lead">Use natural language in AI Shell to buy a Huawei Cloud ECS whose security group has no inbound rules, then log in over SSH and transfer files through an outbound DevBridge tunnel.</p>

## Scenario and goal

Traditional SSH access requires opening port 22 in the security group; once the ECS has a public address, it is continuously exposed to port scans and brute-force attempts. This practice closes inbound traffic entirely:

- The ECS security group has zero inbound rules, so nothing is reachable from the internet.
- The EIP is used for outbound traffic only: the ECS initiates WebSocket connections to the DevBridge relay.
- SSH and scp traffic is forwarded through the DevBridge tunnel, so your local machine never connects to the ECS directly.

AI Shell generates and deploys the Terraform configuration from a natural-language request. The practice fits remote ECS administration where you don't want inbound ports, a bastion host, or a VPN.

## Start the walkthrough

The full steps, Terraform code, and verification commands are on the Huawei Cloud AI for IaC case page:

[Buy an ECS with AI Shell and build a zero-inbound-port SSH tunnel with DevBridge](https://aiiac.developer.huaweicloud.com/cases.html#iac-practice/ecs-devbridge-ssh-tunnel)
