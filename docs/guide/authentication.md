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

## Headless and service environments

The CLI keeps credentials in one of two places, and chooses between them at startup:

- the system keyring (Secret Service over the D-Bus session bus), or
- the config file `~/.huawei/devbridge/config.yaml`.

Desktop sessions, and processes started under `dbus-run-session` or with a persistent `DBUS_SESSION_BUS_ADDRESS`, get the keyring. To decide, the CLI checks the session bus once per process, in this order:

1. `DBUS_SESSION_BUS_ADDRESS` is set to a real address (an `autolaunch:` value does not count);
2. `$XDG_RUNTIME_DIR/bus` exists and is a socket;
3. a `DBUS_SESSION_BUS_ADDRESS=` entry exists in a file under `$HOME/.dbus/session-bus`.

When one of the last two is found, the CLI exports the address it validated so the D-Bus client reuses that bus instead of starting one.

If none applies — a container, a sandbox, a systemd service, any host without a session bus — the CLI **does not call the keyring at all**. This matters even more than it sounds: without a session bus, a keyring call makes the D-Bus client run `dbus-launch`, which forks a `dbus-daemon --session` that nobody reaps. A long-running `devbridge host` would then leak one daemon per credential access. Skipping the keyring keeps the process count flat.

In that mode credentials are read from and written to the config file, and the CLI prints a warning once per process:

```text
no usable D-Bus session bus was detected; credentials are stored in the local config file instead:
~/.huawei/devbridge/config.yaml (directory mode 0700, file mode 0600). The file holds them
base64-encoded, which is not encryption. Run inside a desktop session or under dbus-run-session
to use the system keyring (Secret Service).
```

Treat that as a deliberate trade-off, not a failure:

- the credential is Base64-encoded, **not encrypted** — its confidentiality rests on the `0700` directory and `0600` file permissions, and on the host being single-user;
- on a shared host, prefer `HW_API_KEY` injected through a protected environment variable (or a secrets service) so that no credential is written to disk;
- if you want the keyring back in a headless host, give the process a real bus instead of a per-call one:

```bash
dbus-run-session -- devbridge host --tunnel <tunnel-id> --host-token <token> --ports 8080
```

To check which backend a process picked up, run with `-v` and look for the session bus probe line:

```bash
devbridge auth status -v
```

```text
level=DEBUG msg="d-bus session bus probe" available=false source=none
```

`source` is `env`, `xdg-runtime-dir`, or `session-bus-file` when a bus was found, and `none` when the config file is in use. Windows and macOS always use their native credential store (Credential Manager, Keychain), so this section does not apply to them.

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
