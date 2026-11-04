---
title: Changelog
description: Changes to the DevBridge CLI and related services.
---

# Changelog

## 2026-09-21

### Improvements

- Added the `config` command for viewing and customizing gateway connection settings.
- Switched the default gateway address and SNI domain to `devbridge-s2.hwtunnel.com`.
- Added full-port hosting (port `-1`), which requires an API key in the DevBox use case.
- A tunnel now allows only one Host connection at a time and returns a clearer error for duplicate connections.

## 2026-08-31

### Improvements

- Migrated authentication from AK/SK to API keys, and added the `--api-key` flag and `HW_API_KEY` environment variable.
- Login credentials are now stored primarily in the operating system keyring.
- `host` and `connect` now support API key authentication and the default tunnel.
- `connect --token` mode now requires an explicit tunnel ID.
- `devbridge show` now reports Host/Connect connection counts and upload/download traffic.
- Added an API key management page for creating, viewing, and deleting API keys online, organized by DevBridge and DevBox use cases.

### Removed

- Removed the AK/SK login flags (`--access-key`, `--secret-key`, `--security-token`).
- Removed the JSON output flag from the `list` and `port list` commands.
- Removed the `--huaweicloud` flag and the `loginType` parameter.

## 2026-08-11

### Improvements

- Added the `limits` command to view your account quota and current usage.
- Added the `echo` and `ping` debug tools for verifying the tunnel link.
- Added a global `-v` / `--verbose` flag to enable debug-level logging.

## 2026-07-31

### Improvements

- Added Bash and PowerShell installation, supporting x86-64 and ARM64.
- Added interactive, AK/SK, and temporary-credential login, along with login-status and logout commands.
- Added tunnel management (create, list, view, update, delete), token issuance, and default-tunnel support.
- Added port create, list, view, update, and delete operations, with protocol selection and anonymous access control.
- Added Host support for multiple local ports, temporary tunnel creation, and automatic reconnection.
- Added Connect support for local port mapping and automatic reconnection.
- Added REST APIs for tunnels and ports, with corresponding error codes.
