// Command connect-service 是服务 B：建立远程连接并验证访问服务 A 的本地服务。
//
// 流程：
//  1. 用 SDK 连接服务 A 创建的隧道
//  2. 在本地建立端口映射
//  3. 自动发起 HTTP 请求验证能否访问远端服务
//  4. 持续保持连接，Ctrl+C 退出
//
// 用法：
//
//	export HW_API_KEY="your-api-key"
//	TUNNEL_ID="服务A打印的隧道ID" go run ./cmd/connect-service
//
// 可选环境变量：
//
//	PORT         本地映射端口/远端端口（默认 18080）
//	TUNNEL_ID    隧道 ID（必填）
//	LOCAL_IP     本地监听地址（默认 127.0.0.1）
//	API_BASE_URL 覆盖 SDK 默认 REST API 地址
//	GATEWAY_ADDR 覆盖 SDK 默认网关地址（host:port）
//	GATEWAY_HOST 覆盖 SDK 默认网关 SNI host
//	NO_VERIFY     设为 1 时跳过自动访问验证
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
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

	tunnelID := os.Getenv("TUNNEL_ID")
	if tunnelID == "" {
		fail("缺少隧道 ID：请设置 TUNNEL_ID 环境变量（由服务 A 打印）")
	}

	portNum, _ := strconv.Atoi(envOr("PORT", "18080"))
	if portNum <= 0 || portNum > 65535 {
		failf("PORT 无效: %d", portNum)
	}
	localIP := envOr("LOCAL_IP", "127.0.0.1")
	noVerify := os.Getenv("NO_VERIFY") == "1"

	// 构建 SDK 客户端
	cfg := sdk.Config{
		APIKey: apiKey,
		Logger: logger,
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

	// 1. 验证 API Key
	fmt.Println("== 步骤 1: 验证 API Key ==")
	limits, err := client.GetLimits(ctx)
	if err != nil {
		failf("GetLimits 失败: %v", err)
	}
	fmt.Printf("   ✓ 配额: maxTunnels=%d activeTunnels=%d\n", limits.MaxTunnels, limits.ActiveTunnels)

	// 2. 启动 Connect 远程连接
	fmt.Println("== 步骤 2: 启动 Connect 远程连接 ==")
	fmt.Printf("   隧道 ID:  %s\n", tunnelID)
	fmt.Printf("   远端端口: %d\n", portNum)
	fmt.Printf("   本地监听: %s:%d\n", localIP, portNum)

	connectCtx, connectCancel := context.WithCancel(ctx)
	connectDone := make(chan error, 1)
	go func() {
		connectDone <- client.Connect(connectCtx, sdk.ConnectConfig{
			TunnelID: tunnelID,
			Ports:    []int{portNum},
			LocalIP:  localIP,
		})
	}()

	// 等待连接建立
	time.Sleep(5 * time.Second)

	localURL := fmt.Sprintf("http://%s:%d/", localIP, portNum)

	// 3. 自动验证访问
	if !noVerify {
		fmt.Println("== 步骤 3: 验证访问远端服务 ==")
		fmt.Printf("   请求: GET %s\n", localURL)
		if ok := verifyAccess(localURL, 30*time.Second); ok {
			fmt.Println()
			fmt.Println("════════════════════════════════════════════════════════════")
			fmt.Println("  ✓ 验证成功！服务 B 已通过远程连接访问到服务 A 的本地服务")
			fmt.Println()
			fmt.Printf("    本地映射:  %s\n", localURL)
			fmt.Printf("    远端隧道:  %s (端口 %d)\n", tunnelID, portNum)
			fmt.Println()
			fmt.Println("  你也可以在浏览器打开上面地址继续访问")
			fmt.Println("  按 Ctrl+C 退出")
			fmt.Println("════════════════════════════════════════════════════════════")
			fmt.Println()
		} else {
			fmt.Println("   ⚠ 自动验证未成功，但连接仍在保持，可手动在浏览器尝试访问")
			fmt.Println()
			fmt.Println("════════════════════════════════════════════════════════════")
			fmt.Println("  Connect 连接已建立，请在浏览器打开下面地址手动验证：")
			fmt.Println()
			fmt.Printf("    %s\n", localURL)
			fmt.Println()
			fmt.Println("  按 Ctrl+C 退出")
			fmt.Println("════════════════════════════════════════════════════════════")
			fmt.Println()
		}
	} else {
		fmt.Println()
		fmt.Println("════════════════════════════════════════════════════════════")
		fmt.Println("  ✓ Connect 连接已建立，请在浏览器打开下面地址访问远端服务：")
		fmt.Println()
		fmt.Printf("    %s\n", localURL)
		fmt.Println()
		fmt.Println("  按 Ctrl+C 退出")
		fmt.Println("════════════════════════════════════════════════════════════")
		fmt.Println()
	}

	// 4. 阻塞等待
	select {
	case <-ctx.Done():
	case err := <-connectDone:
		fmt.Printf("Connect 退出: %v\n", err)
	}

	connectCancel()
	<-connectDone
	fmt.Println("已停止 Connect")
}

func handleSignal(cancel context.CancelFunc) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n收到退出信号，正在清理...")
	cancel()
}

// verifyAccess 反复尝试访问本地映射地址，验证能否拿到远端服务响应
func verifyAccess(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Printf("   ✓ 收到响应: HTTP %d\n", resp.StatusCode)
				// 打印前 500 字节内容预览
				preview := string(body)
				if len(preview) > 500 {
					preview = preview[:500] + "..."
				}
				fmt.Printf("   响应预览:\n%s\n", preview)
				return true
			}
		}
		time.Sleep(1 * time.Second)
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
