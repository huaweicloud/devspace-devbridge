package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"huawei.com/devbridge/internal/api"
	client "huawei.com/devbridge/internal/connect"
)

// hostCmd 和 connectCmd 相关的命令行参数
var hostPorts []uint       // host 模式下需要转发的本地端口列表
var hostDescription string // host 模式下新隧道的描述
var hostExpiration int     // host 模式下隧道过期时间（小时）
var connectToken string    // connect 模式下用户直接提供的 JWT token
var hostToken string       // host 模式下用户直接提供的 JWT token
var hostAPIKey string      // host 模式下用户提供的 API Key
var connectAPIKey string   // connect 模式下用户提供的 API Key

// tunnelIDPattern 隧道ID格式：仅允许字母、数字、连字符、下划线，长度 1-64
var tunnelIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// validateTunnelID 校验隧道ID格式
func validateTunnelID(id string) error {
	if id == "" {
		return fmt.Errorf("tunnel ID cannot be empty")
	}
	if !tunnelIDPattern.MatchString(id) {
		return fmt.Errorf("invalid tunnel ID: %q (only letters, digits, hyphens, underscores allowed, length 1-64)", id)
	}
	return nil
}

// portsToInt 将 uint 列表转换为 int 列表
func portsToInt(ports []uint) []int {
	result := make([]int, len(ports))
	for i, p := range ports {
		result[i] = int(p)
	}
	return result
}

// portResultsToInt 将 API 返回的端口列表转换为 int 列表
func portResultsToInt(results []api.ListPortsResult) []int {
	ports := make([]int, len(results))
	for i, p := range results {
		ports[i] = int(p.Port)
	}
	return ports
}

// validatePorts 校验端口列表
func validatePorts(ports []uint) error {
	if len(ports) == 0 {
		return fmt.Errorf("at least one port must be specified via -p/--ports")
	}
	for _, p := range ports {
		if p == 0 || p > 65535 {
			return fmt.Errorf("invalid port number: %d (valid range: 1-65535)", p)
		}
	}
	return nil
}

// hostCmd 以 host 模式启动隧道，将本地端口暴露到远程隧道服务
var hostCmd = &cobra.Command{
	Use:   "host [tunnelId]",
	Short: "Start listener, register to gateway and wait for connections",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		var tunnelId string
		var ports []int
		var jwtToken string

		if hostToken != "" {
			// --token 模式：用户直接提供 JWT token，跳过 API 调用
			// tunnelId 必须指定，端口由 gateway 通过 relay channel 下发
			if len(args) == 0 || args[0] == "" {
				log.Fatalf("tunnelId is required when using --token")
			}
			tunnelId = args[0]
			if err := validateTunnelID(tunnelId); err != nil {
				log.Fatalf("%v", err)
			}
			if cmd.Flags().Changed("ports") {
				fmt.Println("Note: --ports is ignored in --token mode, ports will be fetched from gateway")
			}
			jwtToken = hostToken
		} else if hostAPIKey != "" {
			// --api-key 模式：跳过 AKSK 和 TunnelToken，端口由网关下发
			if len(args) == 0 || args[0] == "" {
				log.Fatalf("tunnelId is required when using --api-key")
			}
			tunnelId = args[0]
			if err := validateTunnelID(tunnelId); err != nil {
				log.Fatalf("%v", err)
			}
			if cmd.Flags().Changed("ports") {
				fmt.Println("Note: --ports is ignored in --api-key mode, ports will be fetched from gateway")
			}
			jwtToken = "" // 不使用 JWT
		} else {

			if len(args) > 0 && args[0] != "" {
				// 指定了 tunnelId：从 API 查询该隧道绑定的所有端口，-p 参数忽略
				tunnelId = args[0]
				if err := validateTunnelID(tunnelId); err != nil {
					log.Fatalf("%v", err)
				}

				portsResult, err := api.ListPorts(tunnelId)
				if err != nil {
					log.Fatalf("Failed to list ports: %v", err)
				}
				ports = portResultsToInt(portsResult)

				if len(ports) == 0 {
					log.Fatalf("No ports configured for tunnel %s", tunnelId)
				}
			} else {
				// 未指定 tunnelId：必须带 -p，自动创建新隧道
				if err := validatePorts(hostPorts); err != nil {
					log.Fatalf("%v", err)
				}
				ports = portsToInt(hostPorts)

				slog.Debug("Creating new tunnel", "ports", ports)
				var exp *int
				if cmd.Flags().Changed("expiration") {
					exp = &hostExpiration
				}
				result, err := api.CreateTunnel(fmt.Sprintf("tunnel-%d-%d", ports[0], time.Now().UnixMilli()%10000), hostDescription, exp)
				if err != nil {
					log.Fatalf("Failed to create tunnel: %v", err)
				}
				tunnelId = result.TunnelId
				fmt.Printf("Created tunnel: %s\n", tunnelId)

				// 自动为新建隧道绑定端口
				allowAnon := true
				for _, p := range ports {
					if err := api.CreatePort(tunnelId, p, "auto", &allowAnon); err != nil {
						log.Fatalf("Failed to create port %d for tunnel %s: %v", p, tunnelId, err)
					}
				}
			}

			// 通过 TunnelToken 接口获取 host JWT
			tokenResult, err := api.TunnelToken(tunnelId, "host")
			if err != nil {
				log.Fatalf("Failed to get host token: %v", err)
			}
			jwtToken = tokenResult.Token
		}

		client.Listen(tunnelId, ports, jwtToken, hostAPIKey)
	},
}

// connectCmd 以 server 模式连接到远程隧道，等待监听端的端口转发请求
var connectCmd = &cobra.Command{
	Use:   "connect [tunnelId]",
	Short: "Start sender, connect to gateway and wait for port forwarding requests",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tunnelId := args[0]
		if err := validateTunnelID(tunnelId); err != nil {
			log.Fatalf("%v", err)
		}

		var jwtToken string
		var ports []int

		if connectToken != "" {
			// -token 模式：用户直接提供 JWT token，跳过 TunnelToken 和 ListPorts
			// 端口列表由 host 端通过 SSH ForwardFromRemotePort 协商下发
			jwtToken = connectToken
		} else if connectAPIKey != "" {
			// --api-key 模式：跳过 TunnelToken 和 ListPorts
			// 端口由 host 端通过 SSH ForwardFromRemotePort 下发
			jwtToken = ""
		} else {
			// 默认模式：通过 API 获取 token 和端口列表

			tokenResult, err := api.TunnelToken(tunnelId, "connect")
			if err != nil {
				log.Fatalf("Failed to get connect token: %v", err)
			}
			jwtToken = tokenResult.Token

			portsResult, err := api.ListPorts(tunnelId)
			if err != nil {
				log.Fatalf("Failed to list ports: %v", err)
			}
			if len(portsResult) == 0 {
				log.Fatalf("No ports configured for tunnel %s", tunnelId)
			}
			ports = portResultsToInt(portsResult)
		}

		client.Send(tunnelId, jwtToken, ports, connectAPIKey)
	},
}

func init() {
	RootCmd.AddCommand(hostCmd)
	RootCmd.AddCommand(connectCmd)
	hostCmd.Flags().UintSliceVarP(&hostPorts, "ports", "p", nil, "Local server port numbers")
	hostCmd.Flags().StringVarP(&hostDescription, "description", "d", "", "Description for new tunnel")
	hostCmd.Flags().IntVarP(&hostExpiration, "expiration", "e", 0, "Tunnel expiration (hours, 1-720)")
	hostCmd.Flags().StringVarP(&hostToken, "token", "t", "", "JWT token for host (skip API token and port lookup)")
	hostCmd.Flags().StringVarP(&hostAPIKey, "api-key", "k", "", "API key for host (skip AKSK auth and token API)")
	connectCmd.Flags().StringVarP(&connectToken, "token", "t", "", "JWT token for connect (skip API token and port lookup)")
	connectCmd.Flags().StringVarP(&connectAPIKey, "api-key", "k", "", "API key for connect (skip AKSK auth and token API)")
}
