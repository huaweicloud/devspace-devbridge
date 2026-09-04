package devbridge_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"huawei.com/devbridge/sdk"
)

// ──────────────────────────────────────────────────────────────
// 示例 1：完整流程 — 创建隧道 → 添加端口 → Host 托管 → Connect 连接
// ──────────────────────────────────────────────────────────────

func ExampleClient_fullWorkflow() {
	ctx := context.Background()

	// 创建客户端（API Key 也可通过 HW_API_KEY 环境变量设置）
	client, err := devbridge.NewClient(devbridge.Config{APIKey: "your-api-key"})
	if err != nil {
		log.Fatal(err)
	}

	// 1. 创建隧道
	tunnel, err := client.CreateTunnel(ctx, "my-dev-tunnel", "开发联调环境", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("隧道已创建: %s\n", tunnel.ID)

	// 2. 添加端口
	allowAnon := true
	if err := client.CreatePort(ctx, tunnel.ID, 8080, "http", &allowAnon); err != nil {
		log.Fatal(err)
	}

	// 3. Host 托管（在服务所在设备运行）
	//
	// 这会阻塞，通常在单独的 goroutine 中运行：
	hostCtx, hostCancel := context.WithCancel(context.Background())
	go func() {
		if err := client.Host(hostCtx, devbridge.HostConfig{
			TunnelID: tunnel.ID,
			Ports:    []int{8080},
		}); err != nil {
			log.Printf("Host 退出: %v", err)
		}
	}()

	time.Sleep(2 * time.Second) // 等待 Host 就绪

	// 4. Connect 连接（在访问设备运行）
	//
	// 连接成功后，在访问设备上 http://localhost:8080 即可访问远端服务
	connectCtx, connectCancel := context.WithCancel(context.Background())
	go func() {
		if err := client.Connect(connectCtx, devbridge.ConnectConfig{
			TunnelID: tunnel.ID,
			Ports:    []int{8080},
		}); err != nil {
			log.Printf("Connect 退出: %v", err)
		}
	}()

	time.Sleep(5 * time.Second)

	// 5. 清理
	connectCancel()
	hostCancel()
	client.DeleteTunnel(ctx, tunnel.ID)
}

// ──────────────────────────────────────────────────────────────
// 示例 2：使用已有隧道 Host 托管
// ──────────────────────────────────────────────────────────────

func ExampleClient_host() {
	client, err := devbridge.NewClient(devbridge.Config{APIKey: "your-api-key"})
	if err != nil {
		log.Fatal(err)
	}

	// 查询隧道端口
	ports, err := client.ListPorts(context.Background(), "aaaadysa")
	if err != nil {
		log.Fatal(err)
	}

	portList := make([]int, len(ports))
	for i, p := range ports {
		portList[i] = int(p.Port)
	}

	// 启动 Host（阻塞）
	err = client.Host(context.Background(), devbridge.HostConfig{
		TunnelID: "aaaadysa",
		Ports:    portList,
	})
	if err != nil {
		log.Fatal(err)
	}
}

// ──────────────────────────────────────────────────────────────
// 示例 3：使用 JWT 令牌（跳过 API 调用）
// ──────────────────────────────────────────────────────────────

func ExampleClient_hostWithToken() {
	client, err := devbridge.NewClient(devbridge.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 先签发 Host 令牌
	token, err := client.IssueToken(context.Background(), "aaaadysa", "host")
	if err != nil {
		log.Fatal(err)
	}

	// 用令牌启动 Host，不再调用 REST API
	err = client.Host(context.Background(), devbridge.HostConfig{
		TunnelID: "aaaadysa",
		JWTToken: token.Token,
	})
	if err != nil {
		log.Fatal(err)
	}
}

// ──────────────────────────────────────────────────────────────
// 示例 4：Connect 连接并访问远端服务
// ──────────────────────────────────────────────────────────────

func ExampleClient_connect() {
	client, err := devbridge.NewClient(devbridge.Config{APIKey: "your-api-key"})
	if err != nil {
		log.Fatal(err)
	}

	// 连接隧道，在本地建立端口映射
	// 连接成功后，http://localhost:8080 → 远端 Host 的 8080 端口
	err = client.Connect(context.Background(), devbridge.ConnectConfig{
		TunnelID: "aaaadysa",
		Ports:    []int{8080},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// ──────────────────────────────────────────────────────────────
// 示例 5：隧道管理
// ──────────────────────────────────────────────────────────────

func ExampleClient_tunnelManagement() {
	ctx := context.Background()
	client, err := devbridge.NewClient(devbridge.Config{APIKey: "your-api-key"})
	if err != nil {
		log.Fatal(err)
	}

	// 创建隧道，有效期 24 小时
	exp := 24
	tunnel, _ := client.CreateTunnel(ctx, "my-tunnel", "描述", &exp)

	// 查询隧道列表
	tunnels, _ := client.ListTunnels(ctx)
	for _, t := range tunnels {
		fmt.Printf("%s: %s\n", t.ID, t.Name)
	}

	// 查询隧道详情
	detail, _ := client.ShowTunnel(ctx, tunnel.ID)
	fmt.Printf("端口数: %d\n", detail.Status.HostConnectionCount)

	// 更新隧道
	newName := "renamed-tunnel"
	client.UpdateTunnel(ctx, tunnel.ID, &newName, nil, nil)

	// 删除隧道
	client.DeleteTunnel(ctx, tunnel.ID)
}
