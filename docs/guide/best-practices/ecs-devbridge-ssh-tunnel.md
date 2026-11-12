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

## What the practice covers

| Item        | Description                                                                                                |
| ----------- | ---------------------------------------------------------------------------------------------------------- |
| Goal        | SSH login and scp file transfer over a DevBridge tunnel to an ECS with all inbound rules closed            |
| Result      | One ECS (VPC, subnet, EIP, FunctionGraph, IAM agency) and one outbound DevBridge tunnel                    |
| Time        | About 30–45 minutes                                                                                        |
| Environment | AI Shell in AI DevSpace, with Terraform, hcloud CLI, and DevBridge CLI built in; default region cn-north-4 |
| Version     | DevBridge 0.3.0-release (x86-64 / ARM64)                                                                   |

The practice also uses FunctionGraph and an IAM agency: the DevBridge API key is stored in a function and fetched by the ECS with temporary credentials, so it never lands in user_data or Terraform state.

## Walkthrough at a glance

1. Verify that the Huawei Cloud credentials and the DevBridge API key are usable.
2. Describe the requirement in AI Shell and let the agent generate the Terraform configuration.
3. Review the plan and deploy the ECS, EIP, security group, and related resources.
4. Wait for cloud-init to install DevBridge, create the tunnel, and host port 22.
5. Log in over SSH and transfer files with scp through the tunnel, then confirm that the inbound path is unreachable.
6. Run destroy to clean up all resources and the tunnel.

## Start the walkthrough

The full steps, Terraform code, and verification commands are on the Huawei Cloud AI for IaC case page:

[Buy an ECS with AI Shell and build a zero-inbound-port SSH tunnel with DevBridge](https://aiiac.developer.huaweicloud.com/cases.html#iac-practice/ecs-devbridge-ssh-tunnel)

::: tip
The walkthrough is currently available in Chinese and includes copy-ready prompts and Terraform configuration.
:::

## Related content

- [Hosting and public access](./host-public-access.md)
- [Hosting and remote connection](./host-remote-connect.md)
- [Host: expose local services](../host.md)
- [Connect: connect to remote services](../connect.md)
