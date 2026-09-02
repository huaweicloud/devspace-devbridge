package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"huawei.com/devbridge/internal/auth"
	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/sdk"
	devbridge "huawei.com/devbridge/sdk"
)

var hostPorts []uint
var hostDescription string
var hostExpiration int
var connectToken string
var hostToken string
var hostAPIKey string
var connectAPIKey string

var tunnelIDPatternConnect = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func validateTunnelIDConnect(id string) error {
	if id == "" {
		return fmt.Errorf("tunnel ID cannot be empty")
	}
	if !tunnelIDPatternConnect.MatchString(id) {
		return fmt.Errorf("invalid tunnel ID: %q (only letters, digits, hyphens, underscores allowed, length 1-64)", id)
	}
	return nil
}

func portsToInt(ports []uint) []int {
	result := make([]int, len(ports))
	for i, p := range ports {
		result[i] = int(p)
	}
	return result
}

func portResultsToInt(results []devbridge.Port) []int {
	ports := make([]int, len(results))
	for i, p := range results {
		ports[i] = int(p.Port)
	}
	return ports
}

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

func resolveHostConfig(cmd *cobra.Command, args []string) (tunnelID string, ports []int, jwtToken string, err error) {
	if hostAPIKey != "" {
		auth.SetOverrideAPIKey(hostAPIKey)
	}
	if hostToken != "" {

		if len(args) == 0 || args[0] == "" {
			return "", nil, "", fmt.Errorf("tunnelID is required when using --token")
		}
		tunnelID = args[0]
		if err := validateTunnelIDConnect(tunnelID); err != nil {
			return "", nil, "", err
		}
		if cmd.Flags().Changed("ports") {
			fmt.Println("Note: --ports is ignored in --token mode, ports will be fetched from gateway")
		}
		jwtToken = hostToken
		return
	}

	tunnelID, ports, err = resolveHostTunnelPorts(cmd, args)
	if err != nil {
		return "", nil, "", err
	}

	if hostAPIKey == "" {
		client := sdk.NewClient()
		tokenResult, err := client.IssueToken(context.Background(), tunnelID, "host")
		if err != nil {
			return "", nil, "", fmt.Errorf("Failed to get host token: %w", err)
		}
		jwtToken = tokenResult.Token
	}
	return
}

func resolveHostTunnelPorts(cmd *cobra.Command, args []string) (tunnelID string, ports []int, err error) {
	client := sdk.NewClient()

	if len(args) > 0 && args[0] != "" {

		tunnelID = args[0]
		if err := validateTunnelIDConnect(tunnelID); err != nil {
			return "", nil, err
		}
		portsResult, err := client.ListPorts(context.Background(), tunnelID)
		if err != nil {
			return "", nil, fmt.Errorf("Failed to list ports: %w", err)
		}
		ports = portResultsToInt(portsResult)
		if len(ports) == 0 {
			return "", nil, fmt.Errorf("No ports configured for tunnel %s", tunnelID)
		}
		return tunnelID, ports, nil
	}

	if !cmd.Flags().Changed("ports") {
		defaultID, err := config.LoadDefaultTunnel()
		if err != nil {
			return "", nil, fmt.Errorf("no tunnelID and no -p specified; either set a default tunnel "+
				"(tunnel set) or pass -p to create a temporary tunnel: %w", err)
		}
		tunnelID = defaultID
		portsResult, err := client.ListPorts(context.Background(), tunnelID)
		if err != nil {
			return "", nil, fmt.Errorf("Failed to list ports: %w", err)
		}
		ports = portResultsToInt(portsResult)
		if len(ports) == 0 {
			return "", nil, fmt.Errorf("No ports configured for tunnel %s", tunnelID)
		}
		return tunnelID, ports, nil
	}

	if err := validatePorts(hostPorts); err != nil {
		return "", nil, err
	}
	ports = portsToInt(hostPorts)

	slog.Debug("Creating new tunnel", "ports", ports)
	var exp *int
	if cmd.Flags().Changed("expiration") {
		exp = &hostExpiration
	}
	result, err := client.CreateTunnel(
		context.Background(),
		fmt.Sprintf("tunnel-%d-%d", ports[0], time.Now().UnixMilli()%10000),
		hostDescription, exp)
	if err != nil {
		return "", nil, fmt.Errorf("Failed to create tunnel: %w", err)
	}
	tunnelID = result.ID
	fmt.Printf("Created tunnel: %s\n", tunnelID)

	allowAnon := true
	for _, p := range ports {
		if err := client.CreatePort(context.Background(), tunnelID, p, "auto", &allowAnon); err != nil {
			return "", nil, fmt.Errorf("Failed to create port %d for tunnel %s: %w", p, tunnelID, err)
		}
	}
	return
}

var hostCmd = &cobra.Command{
	Use:   "host [tunnel-id]",
	Short: "Host local service through DevBridge tunnel",
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, ports, jwtToken, err := resolveHostConfig(cmd, args)
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		client := sdk.NewClient()
		return client.Host(ctx, devbridge.HostConfig{
			TunnelID: tunnelID,
			Ports:    ports,
			JWTToken: jwtToken,
			APIKey:   hostAPIKey,
		})
	}),
}

func resolveConnectConfig(args []string) (tunnelID string, ports []int, jwtToken string, err error) {
	if connectAPIKey != "" {
		auth.SetOverrideAPIKey(connectAPIKey)
	}

	tunnelID, err = resolveConnectTunnelID(args)
	if err != nil {
		return "", nil, "", err
	}
	if err := validateTunnelIDConnect(tunnelID); err != nil {
		return "", nil, "", err
	}

	if connectToken != "" {

		jwtToken = connectToken
		return
	}

	client := sdk.NewClient()
	portsResult, err := client.ListPorts(context.Background(), tunnelID)
	if err != nil {
		return "", nil, "", fmt.Errorf("Failed to list ports: %w", err)
	}
	if len(portsResult) == 0 {
		return "", nil, "", fmt.Errorf("No ports configured for tunnel %s", tunnelID)
	}
	ports = portResultsToInt(portsResult)

	if connectAPIKey == "" {
		tokenResult, err := client.IssueToken(context.Background(), tunnelID, "connect")
		if err != nil {
			return "", nil, "", fmt.Errorf("Failed to get connect token: %w", err)
		}
		jwtToken = tokenResult.Token
	}
	return
}

func resolveConnectTunnelID(args []string) (string, error) {
	if connectToken != "" {

		if len(args) == 0 || args[0] == "" {
			return "", fmt.Errorf("tunnelID is required when using --token")
		}
		return args[0], nil
	}
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}
	id, err := config.LoadDefaultTunnel()
	if err != nil {
		return "", fmt.Errorf("tunnelID is required (no default tunnel set): %w", err)
	}
	return id, nil
}

var connectCmd = &cobra.Command{
	Use:   "connect [tunnel-id]",
	Short: "Start sender, connect to gateway and wait for port forwarding requests",
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, ports, jwtToken, err := resolveConnectConfig(args)
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		client := sdk.NewClient()
		return client.Connect(ctx, devbridge.ConnectConfig{
			TunnelID: tunnelID,
			Ports:    ports,
			JWTToken: jwtToken,
			APIKey:   connectAPIKey,
		})
	}),
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
		"API key for host (skip TunnelToken, use X-API-Key for WebSocket auth)")
	connectCmd.Flags().StringVarP(&connectToken, "token", "t", "",
		"JWT token for connect (skip API token and port lookup)")
	connectCmd.Flags().StringVarP(&connectAPIKey, "api-key", "k", "",
		"API key for connect (skip TunnelToken, use X-API-Key for WebSocket auth)")
}
