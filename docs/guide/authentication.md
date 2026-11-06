---
title: Login and credentials
description: Log in to DevBridge interactively or with an API key.
---

# Login and credentials

<p class="lead">Before you create, host, or connect to a tunnel, log in with the DevBridge CLI.</p>

## Interactive login

For personal development environments, prefer interactive login:

```bash
devbridge auth login
```

The CLI opens a browser to walk you through authentication and saves your login credentials to a protected local credential store. This store uses the system keyring; when the keyring is unavailable, it falls back to a configuration file.

## Using an API key

For dedicated service accounts or automated tasks, you can log in directly with an API key:

```bash
devbridge auth login --api-key "$HW_API_KEY"
```

You can also inject it through an environment variable, and the CLI reads it automatically:

```bash
export HW_API_KEY="your-api-key"
devbridge auth login
```

Don't write a real API key directly into commands, scripts, or pipeline definitions. Inject it through a protected environment variable or a secrets service instead.

### Get an API key

You can create and manage API keys on the DevBridge management page:

> **API key management page**: <https://devstation.connect.huaweicloud.com/space/devbridge?tab=apikey>

After creating a new API key on that page, copy it and inject it through the `HW_API_KEY` environment variable.

## Check login status

```bash
devbridge auth status
```

The status command confirms whether the current identity and credentials are valid; it does not print the full API key.

## Log out

```bash
devbridge auth logout
```

Logging out clears the login credentials saved by the CLI, but does not delete the tunnels you've created.

Don't delete the configuration directory as a substitute for logging out, unless the CLI no longer runs and you're sure you don't need to preserve local state.

## Expired or invalid tokens

`devbridge host` and `devbridge connect` accept a tunnel token via `-t/--token`.
If that token is expired or invalid, the command fails fast instead of retrying
forever:

- The gateway handshake returns HTTP `401`/`403`, or the session is closed with
  an authentication reason. Either is treated as terminal.
- The command exits with a non-zero status (normally `1`) within a few seconds
  and prints a message containing `API key expired` or `invalid token`
  semantics, for example:

  ```text
  authentication failed: API key expired or invalid token: API key expired
  ```

- Listeners and session resources are released before exit; no background
  reconnect goroutine is left running.

This is intentionally different from network failures (DNS errors, gateway
unreachable, `5xx`), which keep the existing reconnect behavior: retries start
at 3 seconds and back off up to 30 seconds. To recover from an expired token,
run `devbridge auth login` again or issue a fresh tunnel token.

## Credential usage principles

- Use interactive login on personal terminals.
- Use an API key for automated tasks, injected through an environment variable.
- Don't pass API keys through chat, logs, or support tickets.
- Don't commit `~/.huawei/devbridge`.
- If you suspect credentials have leaked, log out, revoke them, and reissue immediately.
- Host and Connect tokens are scoped to their corresponding tunnel and connection direction.

## Next steps

- [Create and manage tunnels](./tunnels.md)
- [Host local services](./host.md)
