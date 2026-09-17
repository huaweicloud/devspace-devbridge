/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/huaweicloud/devspace-devbridge/go-sdk"
)

// ──────────────────────────────────────────────────────────────
// Example 1: full workflow - create tunnel, add port, host, then connect
// ──────────────────────────────────────────────────────────────

func ExampleDevbridge_fullWorkflow() {
	ctx := context.Background()

	// Create the client (the API key can also be set via the HW_API_KEY env var)
	client := sdk.New(sdk.Config{APIKey: "your-api-key"})

	// 1. Create a tunnel
	tunnel, err := client.CreateTunnel(ctx, "my-dev-tunnel", "开发联调环境", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("隧道已创建: %s\n", tunnel.ID)

	// 2. Add a port
	allowAnon := true
	if err := client.CreatePort(ctx, tunnel.ID, 8080, "http", &allowAnon); err != nil {
		log.Fatal(err)
	}

	// 3. Host: run on the device where the service lives
	//
	// This blocks, so it usually runs in a separate goroutine:
	hostCtx, hostCancel := context.WithCancel(context.Background())
	go func() {
		if err := client.Host(hostCtx, sdk.HostConfig{
			TunnelID: tunnel.ID,
			Ports:    []int{8080},
		}); err != nil {
			log.Printf("Host 退出: %v", err)
		}
	}()

	time.Sleep(2 * time.Second) // wait for Host to be ready

	// 4. Connect: run on the accessing device
	//
	// After connecting, http://localhost:8080 on the accessing device reaches the remote service
	connectCtx, connectCancel := context.WithCancel(context.Background())
	go func() {
		if err := client.Connect(connectCtx, sdk.ConnectConfig{
			TunnelID: tunnel.ID,
			Ports:    []int{8080},
		}); err != nil {
			log.Printf("Connect 退出: %v", err)
		}
	}()

	time.Sleep(5 * time.Second)

	// 5. Cleanup
	connectCancel()
	hostCancel()
	client.DeleteTunnel(ctx, tunnel.ID)
}

// ──────────────────────────────────────────────────────────────
// Example 2: host an existing tunnel
// ──────────────────────────────────────────────────────────────

func ExampleDevbridge_host() {
	client := sdk.New(sdk.Config{APIKey: "your-api-key"})

	// List the tunnel's ports
	ports, err := client.ListPorts(context.Background(), "aaaadysa")
	if err != nil {
		log.Fatal(err)
	}

	portList := make([]int, len(ports))
	for i, p := range ports {
		portList[i] = p.Port
	}

	// Start the host (blocks)
	err = client.Host(context.Background(), sdk.HostConfig{
		TunnelID: "aaaadysa",
		Ports:    portList,
	})
	if err != nil {
		log.Fatal(err)
	}
}

// ──────────────────────────────────────────────────────────────
// Example 3: use a JWT token (skip API calls)
// ──────────────────────────────────────────────────────────────

func ExampleDevbridge_hostWithToken() {
	client := sdk.New(sdk.Config{})

	// Issue a host token first
	token, err := client.IssueToken(context.Background(), "aaaadysa", "host")
	if err != nil {
		log.Fatal(err)
	}

	// Start the host with the token, without calling the REST API
	err = client.Host(context.Background(), sdk.HostConfig{
		TunnelID: "aaaadysa",
		JWTToken: token.Token,
	})
	if err != nil {
		log.Fatal(err)
	}
}

// ──────────────────────────────────────────────────────────────
// Example 4: connect and reach a remote service
// ──────────────────────────────────────────────────────────────

func ExampleDevbridge_connect() {
	client := sdk.New(sdk.Config{APIKey: "your-api-key"})

	// Connect to the tunnel and set up local port mappings
	// After connecting, http://localhost:8080 -> port 8080 of the remote host
	err := client.Connect(context.Background(), sdk.ConnectConfig{
		TunnelID: "aaaadysa",
		Ports:    []int{8080},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// ──────────────────────────────────────────────────────────────
// Example 5: tunnel management
// ──────────────────────────────────────────────────────────────

func ExampleDevbridge_tunnelManagement() {
	ctx := context.Background()
	client := sdk.New(sdk.Config{APIKey: "your-api-key"})

	// Create a tunnel with 24-hour validity
	exp := 24
	tunnel, _ := client.CreateTunnel(ctx, "my-tunnel", "描述", &exp)

	// List tunnels
	tunnels, _ := client.ListTunnels(ctx)
	for _, t := range tunnels {
		fmt.Printf("%s: %s\n", t.ID, t.Name)
	}

	// Get tunnel details
	detail, _ := client.ShowTunnel(ctx, tunnel.ID)
	if detail.Status != nil {
		fmt.Printf("端口数: %d\n", detail.Status.HostConnectionCount)
	}

	// Update the tunnel
	newName := "renamed-tunnel"
	client.UpdateTunnel(ctx, tunnel.ID, &newName, nil, nil)

	// Delete the tunnel
	client.DeleteTunnel(ctx, tunnel.ID)
}
