---
title: Contributing
description: Contribute to DevBridge open-source development — environment setup, code conventions, testing, branching strategy, commit conventions, and the PR workflow.
---

# Contributing

Thank you for your interest in DevBridge! Feel free to open Issues, submit Pull Requests, or join discussions.

By participating in this project, you agree to abide by the Apache License 2.0 terms and that your contributions will be licensed under the same license.

---

## Project structure

DevBridge is a monorepo with two components, each in its own top-level directory with its own toolchain:

```text
.
├── .github/              # CI workflows, Dependabot config
├── cli/                  # DevBridge CLI (Go, for end users)
│   ├── cmd/              # cobra command definitions
│   ├── internal/         # internal logic (auth, config, i18n, netutil)
│   ├── scripts/          # release helper scripts
│   ├── .golangci.yml     # golangci-lint config
│   ├── .goreleaser.yaml  # GoReleaser production config
│   ├── .goreleaser.test.yaml  # GoReleaser test config
│   ├── Makefile          # build, test, cross-compile entry point
│   └── go.mod            # module huawei.com/devbridge
├── go-sdk/               # DevBridge Go SDK (for third-party integration)
│   ├── client.go         # API client
│   ├── tunnel.go         # tunnel management
│   ├── host.go           # local service hosting
│   ├── connect.go        # remote connection
│   ├── port.go           # port forwarding
│   └── go.mod            # module github.com/huaweicloud/devspace-devbridge/go-sdk
├── LICENSE               # Apache 2.0
└── README.md             # repository overview
```

> **Principle**: new components should each use their own top-level directory, keeping build configs and dependency files within that directory to avoid toolchain conflicts.

---

## Local environment setup

### Toolchain version requirements

| Component | Minimum version         | Purpose         |
| --------- | ----------------------- | --------------- |
| **Git**   | 2.20                    | Version control |
| **Go**    | 1.26 (CLI) / 1.22 (SDK) | Go build & test |

### Install Go (>= 1.26)

The CLI's `go.mod` declares `go 1.26.0`. Install Go 1.26 or later.

```bash
# Option 1: official binary (recommended)
# Download the 1.26 package for your platform from https://go.dev/dl/

# Linux example
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Option 2: package manager (version may lag; verify with go version)
# macOS:  brew install go
# Ubuntu: sudo apt install golang-go
```

Verify:

```bash
go version
# Expected: go version go1.26.x <platform>
```

> **Only working on the Go SDK?** The SDK requires Go >= 1.22. However, if you also need to build the CLI, install 1.26.

### Install golangci-lint (optional, recommended for CLI contributors)

```bash
# See the official guide: https://golangci-lint.run/usage/install/
# macOS: brew install golangci-lint
# Linux:
curl -sSfL https://raw.githubusercontent.com/golangci-lint/golangci-lint/master/install.sh | sh
```

Verify:

```bash
golangci-lint version
```

### One-time check

```bash
git --version      # >= 2.20
go version         # >= 1.26 (CLI) / >= 1.22 (SDK)
```

Once everything passes, proceed to [Quick start](#quick-start).

---

## Quick start

### 1. Fork and clone

```bash
# Fork this repo on GitHub, then:
git clone https://github.com/<your-username>/devspace-devbridge.git
cd devspace-devbridge
```

### 2. Add the upstream remote

```bash
git remote add upstream https://github.com/huaweicloud/devspace-devbridge.git
```

### 3. Install dependencies

```bash
# CLI (Go deps are fetched automatically)
cd cli && go mod tidy

# Go SDK
cd go-sdk && go mod tidy
```

### 4. Pick the component you want to modify and verify locally

```bash
# CLI
cd cli && make build-dev
./bin/devbridge version

# Go SDK
cd go-sdk && go build ./...
```

---

## Code conventions

### Go code (cli/ and go-sdk/)

| Rule            | Tool / notes                                                                                            |
| --------------- | ------------------------------------------------------------------------------------------------------- |
| Formatting      | `gofmt -s -w .` or `go fmt ./...` — run before every commit                                             |
| Static analysis | `go vet ./...`                                                                                          |
| Lint            | `golangci-lint run` (CLI uses `cli/.golangci.yml`)                                                      |
| Naming          | Exported identifiers use PascalCase, unexported use camelCase, package names are lowercase single words |
| Import order    | stdlib → third-party → internal packages, sorted with `goimports`                                       |
| Error handling  | Never ignore error return values; do not discard errors with `_`                                        |
| Comments        | Exported functions/types must have `// Name ...` doc comments                                           |

> **golangci-lint config**: the current `cli/.golangci.yml` disables strict linters such as `errcheck`, `gosec`, and `lll`. New code should follow the spirit of these rules even when the tool does not enforce them.

### General conventions

- **Trailing newline**: keep one empty line at the end of every file (POSIX convention).
- **Indentation**: Go uses tabs.
- **Encoding**: UTF-8, no BOM.

---

## Testing

### CLI tests

```bash
cd cli
make test         # equivalent to go test ./...
```

> **Current status**: the CLI has no unit test files yet. When adding features, please include tests in the same directory as the code under test, named `*_test.go`.

### Go SDK tests

```bash
cd go-sdk
go test ./...     # run all tests
go test -v ./...  # verbose output
go test -race ./...  # race detection
go vet ./...      # static analysis
```

Existing test files:

| File                | Coverage                         |
| ------------------- | -------------------------------- |
| `tunnel_test.go`    | Tunnel creation & management     |
| `websocket_test.go` | WebSocket connection & reconnect |
| `example_test.go`   | Public API usage examples        |

New public APIs must include a corresponding `_test.go` file.

---

## Branching strategy

| Branch    | Purpose            | Direct push?       |
| --------- | ------------------ | ------------------ |
| `master`  | Stable release     | ❌ Merge only      |
| `develop` | Integration branch | ❌ PR merge only   |
| `feat/*`  | Feature branch     | ✅ (personal fork) |
| `fix/*`   | Bug fix branch     | ✅ (personal fork) |

**PR target branch**: `develop` by default. Only maintainers merge to `master` during release windows.

---

## Commit conventions

We use [Conventional Commits](https://www.conventionalcommits.org/) format:

```text
<type>(<scope>): <subject>
```

### type values

| type       | Description                        |
| ---------- | ---------------------------------- |
| `feat`     | New feature                        |
| `fix`      | Bug fix                            |
| `docs`     | Documentation change               |
| `style`    | Code formatting (no logic change)  |
| `refactor` | Refactoring                        |
| `test`     | Test-related                       |
| `chore`    | Build, tooling, dependency changes |

### scope values

| scope | Component    |
| ----- | ------------ |
| `cli` | CLI tool     |
| `sdk` | Go SDK       |
| `ci`  | CI/CD config |

### Examples

```text
feat(cli): support automatic tunnel reconnection
fix(sdk): fix goroutine leak on WebSocket disconnect
chore(ci): upgrade actions/checkout to v7
```

---

## Pull Request workflow

### 1. Create a working branch

```bash
git checkout develop
git pull upstream develop
git checkout -b feat/your-feature
```

### 2. Write code and verify locally

Run the appropriate checks for the component you modified:

```bash
# Modified the CLI
cd cli && golangci-lint run && go vet ./... && make test && make build-dev

# Modified the Go SDK
cd go-sdk && go vet ./... && go test ./... && go build ./...
```

### 3. Commit and push

```bash
git commit -m "feat(cli): your feature description"
git push origin feat/your-feature
```

### 4. Open a PR on GitHub

- **Target branch**: `develop`
- **PR title**: follow the commit convention format
- **PR description** must include:
  - **Change summary**: what you did
  - **Related Issue**: e.g. `Closes #123`
  - **How to test**: how to verify
  - **Screenshots**: attach screenshots for CLI output changes

### 5. PR review checklist

- [ ] All CI passes
- [ ] New features include tests
- [ ] No new lint warnings introduced
- [ ] Dependency changes pass `go mod tidy`

### Automatic CI checks

Opening a PR automatically triggers the following CI:

| Workflow        | Trigger                  | Checks                                 |
| --------------- | ------------------------ | -------------------------------------- |
| `build-cli.yml` | PR to develop/master     | GoReleaser snapshot build verification |
| `build-sdk.yml` | PR modifying `go-sdk/**` | `go build` + `go vet` + `go test`      |

---

## CLI development guide

The CLI lives in `cli/` and is built with Go + cobra.

### Local build

```bash
cd cli
make build-dev    # dev build (output: bin/devbridge)
make build-test   # test environment build
make build-prod   # production build
```

### Run tests

```bash
cd cli
make test         # equivalent to go test ./...
```

### Code checks

```bash
cd cli
golangci-lint run    # uses .golangci.yml
go vet ./...         # standard static analysis
```

### Cross-compilation

```bash
cd cli
make build-all    # builds 6 platform binaries + SHA256 checksums
```

### Code organization conventions

- Command definitions go in `cmd/`, one file per subcommand
- Business logic goes in `internal/`, organized by domain (`auth`, `config`, `i18n`, `netutil`)
- Do not write complex business logic in `cmd/` — keep the command layer thin
- New dependencies must be justified in the PR, and `go mod tidy` must produce a consistent `go.sum`

### Environment variables

CLI configuration priority: config file > ldflags (build-time) > hardcoded defaults.

| Variable         | Scope | Purpose                                       | Required? |
| ---------------- | ----- | --------------------------------------------- | --------- |
| `HW_API_KEY`     | Both  | API key for REST API authentication           | Yes       |
| `DEVBRIDGE_LANG` | CLI   | Override interface language (e.g. `zh`, `en`) | No        |

Create a DevBridge key on the [API key management page](https://devstation.connect.huaweicloud.com/space/devbridge?tab=apikey), then set the environment variable:

```bash
export HW_API_KEY="devbridge_your_api_key"
export DEVBRIDGE_LANG=zh
```

### CLI configuration file

The CLI stores user-level settings in `~/.huawei/devbridge/config.yaml` (YAML, file mode `0600`). Use the `config` subcommand to manage gateway addresses:

```bash
# View current configuration
devbridge config get

# Override the WebSocket gateway address (host:port)
devbridge config set --gateway-addr gateway.example.com:443

# Override the WebSocket gateway SNI host
devbridge config set --gateway-host example.com

# Restore build-time defaults
devbridge config unset
```

Config file values take precedence over the ldflags-injected defaults.

### Build-time address injection (ldflags)

The REST API base URL, login page URL, and WebSocket gateway address are injected at build time via `-ldflags`. The Makefile exposes four variables:

| Makefile variable | Injected into                | Purpose                       |
| ----------------- | ---------------------------- | ----------------------------- |
| `SERVER_DOMAIN`   | `config.DefaultServerDomain` | REST API server domain        |
| `LOGIN_URL`       | `auth.LoginURL`              | Browser login page URL        |
| `GATEWAY_ADDR`    | `config.ServerAddr`          | WebSocket gateway (host:port) |
| `CLUSTER_DOMAIN`  | `config.ServerHost`          | WebSocket gateway SNI host    |

The full REST API base URL is `SERVER_DOMAIN + /open-api-inner/v1/relay-controller`.

The production build (`build-prod`) injects these addresses:

| Variable         | `build-prod`                                 |
| ---------------- | -------------------------------------------- |
| `SERVER_DOMAIN`  | `https://bridge.developer.myhuaweicloud.com` |
| `LOGIN_URL`      | `https://devstation.connect.huaweicloud.com` |
| `GATEWAY_ADDR`   | `gateway.devbridge-s2.hwtunnel.com:443`      |
| `CLUSTER_DOMAIN` | `devbridge-s2.hwtunnel.com`                  |

To target a custom environment, override any variable:

```bash
make build-dev SERVER_DOMAIN=https://my-test-server.com LOGIN_URL=https://my-login.com
```

When modifying these variables, update both `Makefile` and `.goreleaser.yaml` (or `.goreleaser.test.yaml`).

### Relationship to the API documentation

The REST API base URL documented in [REST API](/reference/api) corresponds to the prod build value of `SERVER_DOMAIN` + `RelayControllerPath`:

```text
https://bridge.developer.myhuaweicloud.com  +  /open-api-inner/v1/relay-controller
= https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller
```

---

## Go SDK development guide

The Go SDK lives in `go-sdk/` and enables third-party applications to integrate DevBridge capabilities.

### Local build and tests

```bash
cd go-sdk
go build ./...
go test ./...
go vet ./...
```

### SDK configuration

SDK configuration priority: `Config` struct > environment variable > hardcoded defaults.

Third-party applications configure the SDK via the `Config` struct. Empty fields fall back to environment variables and then to hardcoded defaults:

```go
import devbridge "github.com/huaweicloud/devspace-devbridge/go-sdk"

cfg := devbridge.Config{
    APIBaseURL:  devbridge.DefaultAPIBaseURL,  // REST API base URL
    GatewayAddr: devbridge.DefaultGatewayAddr, // WebSocket gateway (host:port)
    GatewayHost: devbridge.DefaultGatewayHost, // WebSocket gateway SNI host
    APIKey:      "",                            // falls back to HW_API_KEY env var
}
client := devbridge.New(cfg)
```

SDK default constants:

| Constant             | Value                                                                           |
| -------------------- | ------------------------------------------------------------------------------- |
| `DefaultAPIBaseURL`  | `https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller` |
| `DefaultGatewayAddr` | `gateway.devbridge-s2.hwtunnel.com:443`                                         |
| `DefaultGatewayHost` | `devbridge-s2.hwtunnel.com`                                                     |
| `DefaultClusterID`   | `devbridge-s2` (overridable via ldflags)                                        |

### Code organization conventions

- The SDK is an independent Go module (`github.com/huaweicloud/devspace-devbridge/go-sdk`)
- The CLI references the local SDK via a `replace` directive (see `cli/go.mod`)
- Public APIs live in the package root (`client.go`, `tunnel.go`, etc.); internal implementation goes in `internal/`
- Every public type/function should have a corresponding `_test.go` file
- **Backward compatibility**: released APIs must not introduce breaking changes; if a break is unavoidable, mark it with `BREAKING CHANGE` in the PR

---

## Issue guidelines

### Bug reports

Please include:

- **Environment**: OS, Go version (CLI/SDK)
- **DevBridge version**: output of `devbridge version`
- **Reproduction steps**: step-by-step and actionable
- **Expected behavior**: what should happen
- **Actual behavior**: what actually happened
- **Logs/screenshots**: if available

### Feature requests

- **Use case**: when would you need this feature
- **Proposed solution**: how you imagine it could work (optional)
- **Alternatives**: other approaches you considered (optional)

### Good issues

- One issue per problem
- Search existing issues first to avoid duplicates
- Use a clear, descriptive title

---

## Release process

Releases are handled by project maintainers. Contributors do not need to worry about this, but understanding the flow helps explain the branch cadence:

### CLI release

1. A maintainer manually triggers `build-cli.yml` via GitHub Actions `workflow_dispatch`
2. If the version number contains `release`, it is automatically published to GitHub Release and GitCode Release
3. Artifacts include 6 platform tar.gz packages + checksums.txt + install script

### Go SDK release

1. A maintainer manually triggers `build-sdk.yml` via GitHub Actions `workflow_dispatch`, passing a version (e.g. `0.2.3`)
2. The workflow runs build/vet/test quality gates, then creates a GitHub Release (source tar.gz + checksums.txt, `--latest=false` so it never steals the CLI's latest) and syncs the `go-sdk/v*` tag to the GitCode mirror repo
3. The Go module proxy (proxy.golang.org / goproxy.cn) indexes and permanently caches it — the tag is immutable once created, and re-running with an already-released version fails (422)

---

## FAQ

### `go build` fails with "go: go.mod requires go >= 1.26"

Your local Go version is below the CLI requirement. Upgrade to Go 1.26 or later — see [Local environment setup](#local-environment-setup).

If you are only modifying the Go SDK, you can build inside the `go-sdk/` directory, which requires Go >= 1.22.

### `make build-dev` cannot find go-sdk

The CLI references the local `../go-sdk` via a `replace` directive. Make sure you cloned the full monorepo at the root level, not just the `cli/` subdirectory.

### PR CI fails but passes locally

Confirm your local toolchain versions match CI: Go 1.26 (CLI) / 1.22 (SDK). Version mismatches can cause local passes but CI failures.

---

## Contact

- **Issues**: [GitHub Issues](https://github.com/huaweicloud/devspace-devbridge/issues)

---

Thank you again for contributing! Every PR and Issue makes DevBridge better.
