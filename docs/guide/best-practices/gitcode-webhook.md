---
title: Receive GitCode webhook callbacks through a DevBridge tunnel
description: Build a webhook receiver in Huawei Cloud AI DevSpace, expose it through a DevBridge tunnel, and validate the GitCode password header to accept repository events.
---

# Receive GitCode webhook callbacks through a DevBridge tunnel

<p class="lead">Build a webhook receiver with the CodeArts coding agent or AI Shell in AI DevSpace, expose it as a public HTTPS endpoint through a DevBridge tunnel, then configure GitCode to deliver push and issue events to it.</p>

## Scenario and goal

The practice runs in the Huawei Cloud AI DevSpace experience edition and takes about 40 minutes, with no manual coding:

- Build and start a webhook receiver on port 8080 that logs the full HTTP POST payload.
- Validate the `X-GitCode-Token` header so only callbacks with the correct password are processed; other requests get a 401.
- Expose the receiver as a public HTTPS URL with a DevBridge tunnel, without buying a public IP.
- Configure the callback URL and password in a GitCode repository webhook for push, issue, and other events.
- Check the receiver logs to confirm delivery, then clean up the tunnel, service process, and webhook configuration.

It fits developers who need to debug webhook callbacks without allocating a public IP or opening inbound ports, and shows how a password header can enforce non-anonymous access.

## Start the walkthrough

The full steps, prompts, and verification are on the Huawei Cloud AI cloud-native development practice center:

[Receive GitCode webhook callbacks through a DevBridge tunnel](https://aidevbp.developer.huaweicloud.com/practices/p_f92a1ddd)
