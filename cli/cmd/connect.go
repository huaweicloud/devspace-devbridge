package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"huawei.com/devbridge/internal/api"
	"huawei.com/devbridge/internal/auth"
	client "huawei.com/devbridge/internal/connect"
)

// hostCmd 和 connectCmd 相关的命令行参数.
var hostPorts []uint       // host 模式下需要转发的本地端口列表 //nolint:gochecknoglobals // cobra CLI 惯用全局变量
var hostDescription string // host 模式下新隧道的描述 //nolint:gochecknoglobals // cobra CLI 惯用全局变量
var hostExpiration int     // host 模式下隧道过期时间（小时） //nolint:gochecknoglobals // cobra CLI 惯用全局变量
var connectToken string    // connect 模式下用户直接提供的 JWT token //nolint:gochecknoglobals // cobra CLI 惯用全局变量
var hostToken string       // host 模式下用户直接提供的 JWT token //nolint:gochecknoglobals // cobra CLI 惯用全局变量
var hostAPIKey string      // host 模式下用户提供的 API Key //nolint:gochecknoglobals // cobra CLI 惯用全局变量
var connectAPIKey string   // connect 模式下用户提供的 API Key //nolint:gochecknoglobals // cobra CLI 惯用全局变量

// tunnelIDPattern 隧道ID格式：仅允许字母、数字、连字符、下划线，长度 1-64.
var tunnelIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// validateTunnelID 校验隧道ID格式.
func validateTunnelID(id string) error {
	if id == "" {
		return fmt.Errorf("tunnel ID cannot be empty")
	}
	if !tunnelIDPattern.MatchString(id) {
		return fmt.Errorf("invalid tunnel ID: %q (only letters, digits, hyphens, underscores allowed, length 1-64)", id)
	}
	return nil
}

// portsToInt 将 uint 列表转换为 int 列表.
func portsToInt(ports []uint) []int {
	result := make([]int, len(ports))
	for i, p := range ports {
		result[i] = int(p) //nolint:gosec // p 是端口号 0~65535，远在 int 范围内
	}
	return result
}

// portResultsToInt 将 API 返回的端口列表转换为 int 列表.
func portResultsToInt(results []api.ListPortsResult) []int {
	ports := make([]int, len(results))
	for i, p := range results {
		ports[i] = int(p.Port)
	}
	return ports
}

// validatePorts 校验端口列表.
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

// resolveHostConfig 解析 host 模式下的隧道配置（tunnelID、端口、JWT token）.
func resolveHostConfig(cmd *cobra.Command, args []string) (tunnelID string, ports []int, jwtToken string) {
	if hostAPIKey != "" {
		auth.SetOverrideAPIKey(hostAPIKey)
	}
	if hostToken != "" {
		// --token 模式：用户直接提供 JWT token，跳过 API 调用.
		// tunnelID 必须指定，端口由 gateway 通过 relay channel 下发
		if len(args) == 0 || args[0] == "" {
			log.Fatalf("tunnelID is required when using --token")
		}
		tunnelID = args[0]
		if err := validateTunnelID(tunnelID); err != nil {
			log.Fatalf("%v", err)
		}
		if cmd.Flags().Changed("ports") {
			fmt.Println("Note: --ports is ignored in --token mode, ports will be fetched from gateway")
		}
		jwtToken = hostToken
		return
	}

	// 默认模式或 --api-key 模式：正常走 API 调用（API Key 认证）.
	tunnelID, ports = resolveHostTunnelPorts(cmd, args)

	// --api-key 模式跳过 TunnelToken，直接用 API Key 鉴权.
	if hostAPIKey == "" {
		tokenResult, err := api.TunnelToken(tunnelID, "host")
		if err != nil {
			log.Fatalf("Failed to get host token: %v", err)
		}
		jwtToken = tokenResult.Token
	}
	return
}

// resolveHostTunnelPorts 在非 --token 模式下解析隧道 ID 和端口列表.
func resolveHostTunnelPorts(cmd *cobra.Command, args []string) (tunnelID string, ports []int) {
	if len(args) > 0 && args[0] != "" {
		// 指定了 tunnelID：从 API 查询该隧道绑定的所有端口，-p 参数忽略.
		tunnelID = args[0]
		if err := validateTunnelID(tunnelID); err != nil {
			log.Fatalf("%v", err)
		}
		portsResult, err := api.ListPorts(tunnelID)
		if err != nil {
			log.Fatalf("Failed to list ports: %v", err)
		}
		ports = portResultsToInt(portsResult)
		if len(ports) == 0 {
			log.Fatalf("No ports configured for tunnel %s", tunnelID)
		}
		return
	}

	// 未指定 tunnelID：
	//   - 带 -p：创建临时隧道
	//   - 不带 -p：使用 tunnel set 设置的默认隧道
	if !cmd.Flags().Changed("ports") {
		defaultID, err := api.ResolveTunnelID("")
		if err != nil {
			log.Fatalf("no tunnelID and no -p specified; either set a default tunnel "+
				"(tunnel set) or pass -p to create a temporary tunnel: %v", err)
		}
		tunnelID = defaultID
		portsResult, err := api.ListPorts(tunnelID)
		if err != nil {
			log.Fatalf("Failed to list ports: %v", err)
		}
		ports = portResultsToInt(portsResult)
		if len(ports) == 0 {
			log.Fatalf("No ports configured for tunnel %s", tunnelID)
		}
		return
	}

	// 带 -p：创建临时隧道.
	if err := validatePorts(hostPorts); err != nil {
		log.Fatalf("%v", err)
	}
	ports = portsToInt(hostPorts)

	slog.Debug("Creating new tunnel", "ports", ports)
	var exp *int
	if cmd.Flags().Changed("expiration") {
		exp = &hostExpiration
	}
	result, err := api.CreateTunnel(
		fmt.Sprintf("tunnel-%d-%d", ports[0], time.Now().UnixMilli()%10000),
		hostDescription, exp)
	if err != nil {
		log.Fatalf("Failed to create tunnel: %v", err)
	}
	tunnelID = result.TunnelID
	fmt.Printf("Created tunnel: %s\n", tunnelID)

	// 自动为新建隧道绑定端口.
	allowAnon := true
	for _, p := range ports {
		if err := api.CreatePort(tunnelID, p, "auto", &allowAnon); err != nil {
			log.Fatalf("Failed to create port %d for tunnel %s: %v", p, tunnelID, err)
		}
	}
	return
}

// hostCmd 以 host 模式启动隧道，将本地端口暴露到远程隧道服务.
var hostCmd = &cobra.Command{
	Use:   "host [tunnelID]",
	Short: "Start listener, register to gateway and wait for connections",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		tunnelID, ports, jwtToken := resolveHostConfig(cmd, args)
		client.Listen(tunnelID, ports, jwtToken, hostAPIKey)
	},
} //nolint:gochecknoglobals // cobra CLI 惯用全局变量

// connectCmd 以 sender 模式连接到远程隧道，等待 host 端的端口转发请求.
var connectCmd = &cobra.Command{
	Use:   "connect [tunnelID]",
	Short: "Start sender, connect to gateway and wait for port forwarding requests",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if connectAPIKey != "" {
			auth.SetOverrideAPIKey(connectAPIKey)
		}

		var tunnelID string
		if connectToken != "" {
			// --token 模式：必须显式指定 tunnelID，不走默认隧道.
			if len(args) == 0 || args[0] == "" {
				log.Fatalf("tunnelID is required when using --token")
			}
			tunnelID = args[0]
		} else if len(args) > 0 && args[0] != "" {
			tunnelID = args[0]
		} else {
			id, err := api.ResolveTunnelID("")
			if err != nil {
				log.Fatalf("tunnelID is required (no default tunnel set): %v", err)
			}
			tunnelID = id
		}
		if err := validateTunnelID(tunnelID); err != nil {
			log.Fatalf("%v", err)
		}

		var jwtToken string
		var ports []int

		if connectToken != "" {
			// --token 模式：用户直接提供 JWT token，跳过 TunnelToken 和 ListPorts.
			// 端口列表由 host 端通过 SSH ForwardFromRemotePort 协商下发
			jwtToken = connectToken
		} else {
			// 默认模式或 --api-key 模式：通过 API 获取端口列表.
			portsResult, err := api.ListPorts(tunnelID)
			if err != nil {
				log.Fatalf("Failed to list ports: %v", err)
			}
			if len(portsResult) == 0 {
				log.Fatalf("No ports configured for tunnel %s", tunnelID)
			}
			ports = portResultsToInt(portsResult)

			// --api-key 模式跳过 TunnelToken，直接用 API Key 鉴权.
			if connectAPIKey == "" {
				tokenResult, err := api.TunnelToken(tunnelID, "connect")
				if err != nil {
					log.Fatalf("Failed to get connect token: %v", err)
				}
				jwtToken = tokenResult.Token
			}
		}

		client.Connect(tunnelID, jwtToken, ports, connectAPIKey)
	}, //nolint:gochecknoglobals // cobra CLI 惯用全局变量
}

func init() {
	RootCmd.AddCommand(hostCmd)
	RootCmd.AddCommand(connectCmd)
	hostCmd.Flags().UintSliceVarP(&hostPorts, "ports", "p", nil, "Local server port numbers")
	hostCmd.Flags().StringVarP(&hostDescription, "description", "d", "", "Description for new tunnel")
	hostCmd.Flags().IntVarP(&hostExpiration, "expiration", "e", 0,
		"Tunnel expiration (hours, 1-720)")
	hostCmd.Flags().StringVarP(&hostToken, "token", "t", "",
		"JWT token for host (skip API token and port lookup)")
	hostCmd.Flags().StringVarP(&hostAPIKey, "api-key", "k", "",
		"API key for host (skip TunnelToken, use X-API-Key for WebSocket auth)") //nolint:lll // flag 描述不可拆行
	connectCmd.Flags().StringVarP(&connectToken, "token", "t", "",
		"JWT token for connect (skip API token and port lookup)")
	connectCmd.Flags().StringVarP(&connectAPIKey, "api-key", "k", "",
		"API key for connect (skip TunnelToken, use X-API-Key for WebSocket auth)") //nolint:lll // flag 描述不可拆行
}
