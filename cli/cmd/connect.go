package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	devbridge "github.com/huaweicloud/devspace-devbridge/go-sdk"
	"github.com/spf13/cobra"
	"huawei.com/devbridge/internal/auth"
	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"
)

var hostPorts []int
var hostDescription string
var hostExpiration int
var connectToken string
var hostToken string
var hostAPIKey string
var connectAPIKey string

func portResultsToInt(results []devbridge.Port) []int {
	ports := make([]int, len(results))
	for i, p := range results {
		ports[i] = p.Port
	}
	return ports
}

func validatePorts(ports []int) error {
	if len(ports) == 0 {
		return fmt.Errorf("%s", i18n.T(i18n.Msg.Connect.PortsRequired))
	}
	for _, p := range ports {
		if p == -1 {
			continue
		}
		if p < 1 || p > 65535 {
			return fmt.Errorf(i18n.T(i18n.Msg.Connect.InvalidPortNumber), p)
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
			return "", nil, "", fmt.Errorf("%s", i18n.T(i18n.Msg.Connect.TokenRequiresTunnelID))
		}
		tunnelID = args[0]
		if cmd.Flags().Changed("ports") {
			fmt.Println(i18n.T(i18n.Msg.Connect.TokenModePortsNote))
		}
		jwtToken = hostToken
		return
	}

	tunnelID, ports, err = resolveHostTunnelPorts(cmd, args)
	if err != nil {
		return "", nil, "", err
	}

	if hostAPIKey == "" {
		client := newSDKClient()
		tokenResult, err := client.IssueToken(context.Background(), tunnelID, "host")
		if err != nil {
			return "", nil, "", fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Connect.HostTokenFailed), err)
		}
		jwtToken = tokenResult.Token
	}
	return
}

func resolveHostTunnelPorts(cmd *cobra.Command, args []string) (tunnelID string, ports []int, err error) {
	client := newSDKClient()

	if len(args) > 0 && args[0] != "" {

		tunnelID = args[0]
		portsResult, err := client.ListPorts(context.Background(), tunnelID)
		if err != nil {
			return "", nil, fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Connect.ListPortsFailed), err)
		}
		ports = portResultsToInt(portsResult)
		if len(ports) == 0 {
			return "", nil, fmt.Errorf(i18n.T(i18n.Msg.Connect.NoPortsConfigured), tunnelID)
		}
		return tunnelID, ports, nil
	}

	if !cmd.Flags().Changed("ports") {
		defaultID, err := config.LoadDefaultTunnel()
		if err != nil {
			return "", nil, fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Connect.NoTunnelIDNoPorts), err)
		}
		tunnelID = defaultID
		portsResult, err := client.ListPorts(context.Background(), tunnelID)
		if err != nil {
			return "", nil, fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Connect.ListPortsFailed), err)
		}
		ports = portResultsToInt(portsResult)
		if len(ports) == 0 {
			return "", nil, fmt.Errorf(i18n.T(i18n.Msg.Connect.NoPortsConfigured), tunnelID)
		}
		return tunnelID, ports, nil
	}

	if err := validatePorts(hostPorts); err != nil {
		return "", nil, err
	}
	ports = hostPorts

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
		return "", nil, fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Tunnel.CreateFailed), err)
	}
	tunnelID = result.ID
	fmt.Printf(i18n.T(i18n.Msg.Connect.TunnelCreated), tunnelID)

	allowAnon := true
	for _, p := range ports {
		if err := client.CreatePort(context.Background(), tunnelID, p, "auto", &allowAnon); err != nil {
			return "", nil, fmt.Errorf(i18n.T(i18n.Msg.Connect.CreatePortFailed), p, tunnelID, err)
		}
	}
	return
}

var hostCmd = &cobra.Command{
	Use:   "host [tunnel-id]",
	Short: i18n.T(i18n.Msg.Connect.HostShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, ports, jwtToken, err := resolveHostConfig(cmd, args)
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		client := newSDKClient()
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
	if connectToken != "" {

		jwtToken = connectToken
		return
	}

	client := newSDKClient()
	portsResult, err := client.ListPorts(context.Background(), tunnelID)
	if err != nil {
		return "", nil, "", fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Connect.ListPortsFailed), err)
	}
	if len(portsResult) == 0 {
		return "", nil, "", fmt.Errorf(i18n.T(i18n.Msg.Connect.NoPortsConfigured), tunnelID)
	}
	ports = portResultsToInt(portsResult)

	if connectAPIKey == "" {
		tokenResult, err := client.IssueToken(context.Background(), tunnelID, "connect")
		if err != nil {
			return "", nil, "", fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Connect.ConnectTokenFailed), err)
		}
		jwtToken = tokenResult.Token
	}
	return
}

func resolveConnectTunnelID(args []string) (string, error) {
	if connectToken != "" {

		if len(args) == 0 || args[0] == "" {
			return "", fmt.Errorf("%s", i18n.T(i18n.Msg.Connect.TokenRequiresTunnelID))
		}
		return args[0], nil
	}
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}
	id, err := config.LoadDefaultTunnel()
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Connect.TunnelIDRequired), err)
	}
	return id, nil
}

var connectCmd = &cobra.Command{
	Use:   "connect [tunnel-id]",
	Short: i18n.T(i18n.Msg.Connect.ConnectShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, ports, jwtToken, err := resolveConnectConfig(args)
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		client := newSDKClient()
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
	hostCmd.Flags().IntSliceVarP(&hostPorts, "ports", "p", nil, i18n.T(i18n.Msg.Connect.FlagPorts))
	hostCmd.Flags().StringVarP(&hostDescription, "description", "d", "", i18n.T(i18n.Msg.Connect.FlagDescription))
	hostCmd.Flags().IntVarP(&hostExpiration, "expiration", "e", 0,
		i18n.T(i18n.Msg.Connect.FlagExpiration))
	hostCmd.Flags().StringVarP(&hostToken, "token", "t", "",
		i18n.T(i18n.Msg.Connect.FlagHostToken))
	hostCmd.Flags().StringVarP(&hostAPIKey, "api-key", "k", "",
		i18n.T(i18n.Msg.Connect.FlagHostAPIKey))
	connectCmd.Flags().StringVarP(&connectToken, "token", "t", "",
		i18n.T(i18n.Msg.Connect.FlagConnectToken))
	connectCmd.Flags().StringVarP(&connectAPIKey, "api-key", "k", "",
		i18n.T(i18n.Msg.Connect.FlagConnectAPIKey))
}
