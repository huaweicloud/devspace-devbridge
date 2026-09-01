# 版本号: 优先用环境变量，否则从 git tag 推导，最后回退到 dev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
SERVER_DOMAIN ?= http://relay-dev-local.tailb4159e.ts.net:8443
LOGIN_URL ?= https://devstation.ulanqab.huawei.com
GATEWAY_ADDR ?= 100.95.181.191:4433
CLUSTER_DOMAIN ?= cn-north-4-bridge.myhuaweicloud.com

LDFLAGS = -X huawei.com/devbridge/cmd.version=$(VERSION) \
          -X huawei.com/devbridge/internal/auth.LoginURL=$(LOGIN_URL) \
          -X huawei.com/devbridge/internal/config.DefaultServerDomain=$(SERVER_DOMAIN) \
          -X huawei.com/devbridge/internal/connect.ServerAddr=$(GATEWAY_ADDR) \
          -X huawei.com/devbridge/internal/connect.ServerHost=$(CLUSTER_DOMAIN)

GOFLAGS = -ldflags "-s -w $(LDFLAGS)" -buildmode=pie -trimpath
CGO_FLAG = CGO_ENABLED=0

.PHONY: build build-dev build-test build-prod build-all build-linux build-linux-arm64 build-windows build-windows-arm64 build-darwin build-darwin-arm64 checksum clean test help

build:
	$(CGO_FLAG) go build $(GOFLAGS) -o bin/devbridge ./cmd/cli

build-dev: VERSION := dev
build-dev: SERVER_DOMAIN := http://relay-dev-local.tailb4159e.ts.net:8443
build-dev: LOGIN_URL := https://devstation.ulanqab.huawei.com
build-dev: build

build-test: VERSION := test
build-test: SERVER_DOMAIN := http://relay-dev-local.tailb4159e.ts.net:8443
build-test: LOGIN_URL := https://devstation.ulanqab.huawei.com
build-test: build

build-prod: VERSION := prod
build-prod: SERVER_DOMAIN := https://devstation.myhuaweicloud.com
build-prod: LOGIN_URL := https://devstation.connect.huaweicloud.com
build-prod: GATEWAY_ADDR := gateway.cn-north-4-bridge.myhuaweicloud.com:443
build-prod: build

# 跨平台构建: 产物名含版本号（与 install.sh 对齐: devbridge_{OS}_{Arch}_{Version}）
build-all: build-linux build-linux-arm64 build-windows build-windows-arm64 build-darwin build-darwin-arm64 checksum

build-linux:
	GOOS=linux GOARCH=amd64 $(CGO_FLAG) go build $(GOFLAGS) -o bin/devbridge_linux_amd64_$(VERSION) ./cmd/cli

build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(CGO_FLAG) go build $(GOFLAGS) -o bin/devbridge_linux_arm64_$(VERSION) ./cmd/cli

build-windows:
	GOOS=windows GOARCH=amd64 $(CGO_FLAG) go build $(GOFLAGS) -o bin/devbridge_windows_amd64_$(VERSION).exe ./cmd/cli

build-windows-arm64:
	GOOS=windows GOARCH=arm64 $(CGO_FLAG) go build $(GOFLAGS) -o bin/devbridge_windows_arm64_$(VERSION).exe ./cmd/cli

build-darwin:
	GOOS=darwin GOARCH=amd64 $(CGO_FLAG) go build $(GOFLAGS) -o bin/devbridge_darwin_amd64_$(VERSION) ./cmd/cli

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(CGO_FLAG) go build $(GOFLAGS) -o bin/devbridge_darwin_arm64_$(VERSION) ./cmd/cli

# 为所有产物生成 SHA256 校验文件
checksum:
	@cd bin && for f in devbridge_*_*; do \
		[ -f "$$f" ] || continue; \
		[[ "$$f" == *.sha256 ]] && continue; \
		sha256sum "$$f" | awk '{print $$1}' > "$$f.sha256"; \
		echo "  ✓ $$f.sha256"; \
	done

clean:
	rm -rf bin/

test:
	go test ./...

help:
	@echo "DevBridge Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  build          - Build for current platform"
	@echo "  build-dev      - Build for dev environment"
	@echo "  build-test     - Build for test environment"
	@echo "  build-prod     - Build for prod environment"
	@echo "  build-all      - Build for all 6 platforms (with version in filename)"
	@echo "  checksum       - Generate SHA256 checksums for all binaries in bin/"
	@echo "  clean          - Remove bin/ directory"
	@echo "  test           - Run all tests"
	@echo "  help           - Show this help"
	@echo ""
	@echo "Variables (override with make VAR=value):"
	@echo "  VERSION           - Version string (default: git describe or 'dev')"
	@echo "  SERVER_DOMAIN     - API server domain"
	@echo "  LOGIN_URL         - Login page URL"
	@echo "  GATEWAY_ADDR      - WebSocket gateway address"
	@echo "  CLUSTER_DOMAIN    - Cluster domain for SNI"
