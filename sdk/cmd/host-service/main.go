// Command host-service 是服务 A：启动本地 HTTP 服务并创建托管。
//
// 流程：
//  1. 用 python3 -m http.server 启动本地 HTTP 服务
//  2. 用 SDK 创建隧道 + 添加端口
//  3. 用 SDK 启动 Host，把本地端口托管到 DevBridge 中继
//  4. 打印隧道 ID，供服务 B（connect-service）使用
//  5. Ctrl+C 退出时自动停止 python 服务并删除隧道
//
// 用法：
//
//	export HW_API_KEY="your-api-key"
//	go run ./cmd/host-service
//
// 可选环境变量：
//
//	PORT         本地服务端口/隧道端口（默认 18080）
//	TUNNEL_NAME  隧道名称（默认 host-service-verify）
//	KEEP_TUNNEL  设为 1 时退出不删隧道
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
	"os/exec"
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
	tunnelName := envOr("TUNNEL_NAME", "host-service-verify")
	keepTunnel := os.Getenv("KEEP_TUNNEL") == "1"

	// 构建 SDK 客户端
	opts := []sdk.Option{sdk.WithAPIKey(apiKey), sdk.WithLogger(logger)}
	if url := os.Getenv("API_BASE_URL"); url != "" {
		opts = append(opts, sdk.WithAPIBaseURL(url))
	}
	if addr := os.Getenv("GATEWAY_ADDR"); addr != "" {
		host := os.Getenv("GATEWAY_HOST")
		opts = append(opts, sdk.WithGateway(addr, host))
	}
	client := sdk.NewClient(opts...)

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

	// 2. 启动本地 HTTP 服务（python3 -m http.server）
	fmt.Printf("== 步骤 2: 启动本地 HTTP 服务 (python3 -m http.server %d) ==\n", portNum)
	pyCtx, pyCancel := context.WithCancel(ctx)
	pyCmd := exec.CommandContext(pyCtx, "python3", "-m", "http.server", strconv.Itoa(portNum))
	pyCmd.Stdout = os.Stdout
	pyCmd.Stderr = os.Stderr
	if err := pyCmd.Start(); err != nil {
		failf("启动 python3 失败: %v", err)
	}
	localURL := fmt.Sprintf("http://127.0.0.1:%d/", portNum)
	if !waitReady(localURL, 30*time.Second) {
		failf("等待超时：python3 服务未就绪")
	}
	fmt.Printf("   ✓ 本地服务已就绪: %s\n", localURL)

	// 3. 创建隧道
	fmt.Println("== 步骤 3: 创建隧道 ==")
	exp := 1
	tunnel, err := client.CreateTunnel(ctx, tunnelName, "服务A托管验证", &exp)
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

	// 5. 启动 Host 托管
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

	// 6. 打印隧道 ID 和公网访问地址
	tunnelURL := fmt.Sprintf("https://%s-%d.cn-north-4-bridge.myhuaweicloud.com", tunnel.ID, portNum)
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("  ✓ 服务 A 托管已启动！")
	fmt.Println()
	fmt.Printf("    隧道 ID:  %s\n", tunnel.ID)
	fmt.Printf("    本地服务:  %s\n", localURL)
	fmt.Printf("    公网 URL:  %s\n", tunnelURL)
	fmt.Println()
	fmt.Println("  请在另一个终端运行服务 B 建立远程连接：")
	fmt.Println()
	fmt.Printf("    HW_API_KEY=$HW_API_KEY TUNNEL_ID=%s PORT=%d go run ./cmd/connect-service\n", tunnel.ID, portNum)
	fmt.Println()
	fmt.Println("  按 Ctrl+C 退出（自动停止 python 服务并删除隧道）")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println()

	// 7. 阻塞等待
	select {
	case <-ctx.Done():
	case err := <-hostDone:
		fmt.Printf("Host 退出: %v\n", err)
	}

	hostCancel()
	<-hostDone
	fmt.Println("已停止 Host")

	pyCancel()
	_ = pyCmd.Wait()
	fmt.Println("已停止本地 HTTP 服务")
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
