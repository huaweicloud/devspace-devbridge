// Command verify 用 SDK 托管本地服务，打印隧道公网 URL 供浏览器访问。
//
// 前置：先在另一个终端用 python 起本地服务：
//
//	python3 -m http.server 18080
//
// 然后运行本程序：
//
//	export HW_API_KEY="your-api-key"
//	go run ./cmd/verify
//
// 流程：
//  1. 检测本地端口是否已被 python 服务占用（等待用户起好服务）
//  2. 用 SDK 创建隧道 + 添加端口
//  3. 用 SDK 启动 Host，把本地端口托管到 DevBridge 中继
//  4. 打印隧道公网 URL，在浏览器打开即可访问
//  5. Ctrl+C 退出时自动删除隧道
//
// 可选环境变量：
//
//	PORT         本地服务端口/隧道端口（默认 18080）
//	TUNNEL_NAME  隧道名称（默认 sdk-verify）
//	KEEP_TUNNEL  设为 1 时退出不删隧道，便于复查
//	API_BASE_URL 覆盖 SDK 默认 REST API 地址
//	GATEWAY_ADDR 覆盖 SDK 默认网关地址（host:port）
//	GATEWAY_HOST 覆盖 SDK 默认网关 SNI host
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	sdk "huawei.com/devbridge/sdk"
)

func main() {
	verbose := flag.Bool("v", false, "开启 SDK debug 日志")
	flag.Parse()

	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	apiKey := os.Getenv("HW_API_KEY")
	if apiKey == "" {
		fail("缺少 API Key：请设置 HW_API_KEY 环境变量")
	}

	portNum, _ := strconv.Atoi(envOr("PORT", "18080"))
	if portNum <= 0 || portNum > 65535 {
		failf("PORT 无效: %d", portNum)
	}
	tunnelName := envOr("TUNNEL_NAME", "sdk-verify")
	keepTunnel := os.Getenv("KEEP_TUNNEL") == "1"

	// 构建 SDK 客户端
	cfg := sdk.Config{
		APIKey: apiKey,
	}
	if url := os.Getenv("API_BASE_URL"); url != "" {
		cfg.APIBaseURL = url
	}
	if addr := os.Getenv("GATEWAY_ADDR"); addr != "" {
		cfg.GatewayAddr = addr
		cfg.GatewayHost = os.Getenv("GATEWAY_HOST")
	}
	client, err := sdk.NewClient(cfg)
	if err != nil {
		failf("创建 SDK 客户端失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleSignal(cancel)

	// 1. 确认凭证可用
	fmt.Println("== 步骤 1: 验证 API Key ==")
	limits, err := client.GetLimits(ctx)
	if err != nil {
		failf("GetLimits 失败: %v", err)
	}
	fmt.Printf("   ✓ 配额: maxTunnels=%d activeTunnels=%d\n", limits.MaxTunnels, limits.ActiveTunnels)

	// 2. 提示用户用 python 起本地服务，并等待端口就绪
	fmt.Printf("== 步骤 2: 等待本地服务 127.0.0.1:%d 就绪 ==\n", portNum)
	fmt.Printf("   请在另一个终端运行：python3 -m http.server %d\n", portNum)
	fmt.Println("   （或任意其他方式在 127.0.0.1 上监听该端口）")
	localURL := fmt.Sprintf("http://127.0.0.1:%d/", portNum)
	if !waitReady(localURL, 5*time.Minute) {
		failf("等待超时：5 分钟内未检测到 127.0.0.1:%d 上的服务，请确认 python 已启动", portNum)
	}
	fmt.Printf("   ✓ 检测到本地服务已就绪: %s\n", localURL)

	// 3. 创建隧道
	fmt.Println("== 步骤 3: 创建隧道 ==")
	exp := 1
	tunnel, err := client.CreateTunnel(ctx, tunnelName, "SDK 托管验证", &exp)
	if err != nil {
		failf("CreateTunnel 失败: %v", err)
	}
	fmt.Printf("   ✓ 隧道已创建: id=%s\n", tunnel.ID)
	if !keepTunnel {
		defer cleanup(context.Background(), client, tunnel.ID)
	}

	// 4. 添加端口
	fmt.Println("== 步骤 4: 添加端口 ==")
	allowAnon := true
	if err := client.CreatePort(ctx, tunnel.ID, portNum, "http", &allowAnon); err != nil {
		failf("CreatePort 失败: %v", err)
	}
	fmt.Printf("   ✓ 端口 %d 已添加（允许匿名访问）\n", portNum)

	// 5. 启动 Host
	fmt.Println("== 步骤 5: 启动 Host 托管 ==")
	hostCtx, hostCancel := context.WithCancel(ctx)
	hostDone := make(chan error, 1)
	go func() {
		hostDone <- client.Host(hostCtx, sdk.HostConfig{
			TunnelID: tunnel.ID,
			Ports:    []int{portNum},
		})
	}()
	time.Sleep(3 * time.Second)

	// 6. 打印公网访问地址
	tunnelURL := fmt.Sprintf("https://%s-%d.cn-north-4-bridge.myhuaweicloud.com", tunnel.ID, portNum)
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("  ✓ Host 托管已启动，在浏览器打开下面地址访问本地服务：")
	fmt.Println()
	fmt.Printf("    %s\n", tunnelURL)
	fmt.Println()
	fmt.Println("  按 Ctrl+C 退出并自动删除隧道")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println()

	// 7. 阻塞等待，直到收到信号
	select {
	case <-ctx.Done():
	case err := <-hostDone:
		fmt.Printf("Host 退出: %v\n", err)
	}

	hostCancel()
	<-hostDone
	fmt.Println("已停止 Host")
}

func handleSignal(cancel context.CancelFunc) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n收到退出信号，正在清理...")
	cancel()
}

func cleanup(ctx context.Context, client *sdk.Client, tunnelID string) {
	fmt.Printf("== 清理: 删除隧道 %s ==\n", tunnelID)
	if err := client.DeleteTunnel(ctx, tunnelID); err != nil {
		fmt.Printf("   ⚠ 删除隧道失败: %v\n", err)
		return
	}
	fmt.Println("   ✓ 隧道已删除")
}

func waitReady(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "✗", msg)
	os.Exit(1)
}

func failf(format string, args ...any) {
	fmt.Fprintln(os.Stderr, "✗", fmt.Sprintf(format, args...))
	os.Exit(1)
}
