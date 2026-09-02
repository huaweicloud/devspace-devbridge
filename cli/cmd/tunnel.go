package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"
	"huawei.com/devbridge/internal/sdk"
	devbridge "huawei.com/devbridge/sdk"

	"github.com/spf13/cobra"
)

// TunnelNotFoundCode 与服务端约定的"隧道不存在"错误码。
const TunnelNotFoundCode = "10002"

var tunnelIDPattern = regexp.MustCompile(`^[a-z2-7]{8}$`)

var (
	tunnelDescription string
	tunnelExpiration  int
	tunnelName        string
	tunnelScope       string
)

func validateTunnelIDLocal(id string) error {
	if !tunnelIDPattern.MatchString(id) {
		return fmt.Errorf("invalid tunnel id: %q (only lowercase letters and digits 2-7 allowed, length must be 8)", id)
	}
	return nil
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T(i18n.Msg.Tunnel.ListShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		client := sdk.NewClient()
		tunnels, err := client.ListTunnels(context.Background())
		if err != nil {
			return err
		}
		if len(tunnels) == 0 {
			fmt.Println(i18n.T(i18n.Msg.Tunnel.TunnelListEmpty))
			return nil
		}
		headers := []string{
			i18n.T(i18n.Msg.Tunnel.TunnelID),
			i18n.T(i18n.Msg.Tunnel.Name),
			i18n.T(i18n.Msg.Tunnel.Description),
			i18n.T(i18n.Msg.Tunnel.TunnelExpiration),
			i18n.T(i18n.Msg.Tunnel.PortCount),
		}
		var rows [][]string
		for _, t := range tunnels {
			rows = append(rows, []string{
				t.ID,
				t.Name,
				t.Description,
				formatTunnelRemaining(int64(t.TunnelExpiration)),
				strconv.Itoa(t.PortCount),
			})
		}
		printTable(headers, rows)
		return nil
	}),
}

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: i18n.T(i18n.Msg.Tunnel.CreateShort),
	Args:  cobra.ExactArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		var exp *int
		if cmd.Flags().Changed("expiration") {
			exp = &tunnelExpiration
		}
		client := sdk.NewClient()
		result, err := client.CreateTunnel(context.Background(), args[0], tunnelDescription, exp)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Tunnel.CreateFailed), err)
		}
		printKV([][2]string{
			{i18n.T(i18n.Msg.Tunnel.TunnelID), result.ID},
			{i18n.T(i18n.Msg.Tunnel.Name), result.Name},
			{i18n.T(i18n.Msg.Tunnel.Description), result.Description},
			{i18n.T(i18n.Msg.Tunnel.TunnelExpiration), formatTunnelExpiration(int64(result.ExpirationHours))},
		})
		return nil
	}),
}

var showCmd = &cobra.Command{
	Use:   "show [tunnel-id]",
	Short: i18n.T(i18n.Msg.Tunnel.ShowShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		client := sdk.NewClient()
		result, err := client.ShowTunnel(context.Background(), tunnelID)
		if err != nil {
			return err
		}
		var status devbridge.TunnelStatus
		if result.Status != nil {
			status = *result.Status
		}
		printKV([][2]string{
			{i18n.T(i18n.Msg.Tunnel.TunnelID), result.ID},
			{i18n.T(i18n.Msg.Tunnel.Name), result.Name},
			{i18n.T(i18n.Msg.Tunnel.TunnelExpiration), formatTunnelRemaining(int64(result.TunnelExpiration))},
			{i18n.T(i18n.Msg.Tunnel.Description), result.Description},
			{i18n.T(i18n.Msg.Tunnel.ClientConnectionCount), fmt.Sprintf("%d", status.ClientConnectionCount)},
			{i18n.T(i18n.Msg.Tunnel.HostConnectionCount), fmt.Sprintf("%d", status.HostConnectionCount)},
			{i18n.T(i18n.Msg.Tunnel.TotalUploadBytes), formatBytes(status.TotalUploadBytes)},
			{i18n.T(i18n.Msg.Tunnel.TotalDownloadBytes), formatBytes(status.TotalDownloadBytes)},
		})
		return nil
	}),
}

var updateCmd = &cobra.Command{
	Use:   "update [tunnel-id]",
	Short: i18n.T(i18n.Msg.Tunnel.UpdateShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		var exp *int
		if cmd.Flags().Changed("expiration") {
			exp = &tunnelExpiration
		}
		var name *string
		if cmd.Flags().Changed("name") {
			name = &tunnelName
		}
		var desc *string
		if cmd.Flags().Changed("description") {
			desc = &tunnelDescription
		}
		client := sdk.NewClient()
		if err := client.UpdateTunnel(context.Background(), tunnelID, name, desc, exp); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Tunnel.UpdateFailed), err)
		}
		fmt.Println(i18n.T(i18n.Msg.Tunnel.TunnelUpdated))
		return nil
	}),
}

var deleteCmd = &cobra.Command{
	Use:   "delete [tunnel-id]",
	Short: i18n.T(i18n.Msg.Tunnel.DeleteShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		client := sdk.NewClient()
		if err := client.DeleteTunnel(context.Background(), tunnelID); err != nil {
			return err
		}
		if def, e := config.LoadDefaultTunnel(); e == nil && def == tunnelID {
			_ = config.DeleteDefaultTunnel()
			fmt.Println(i18n.T(i18n.Msg.Tunnel.DefaultTunnelCleared))
		}
		fmt.Println(i18n.T(i18n.Msg.Tunnel.TunnelDeleted))
		return nil
	}),
}

var deleteAllCmd = &cobra.Command{
	Use:   "delete-all",
	Short: i18n.T(i18n.Msg.Tunnel.DeleteAllShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		client := sdk.NewClient()
		if err := client.DeleteAllTunnels(context.Background()); err != nil {
			return err
		}
		fmt.Println(i18n.T(i18n.Msg.Tunnel.TunnelDeletedAll))
		return nil
	}),
}

var tokenCmd = &cobra.Command{
	Use:   "token [tunnel-id]",
	Short: i18n.T(i18n.Msg.Tunnel.TokenIssueShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		client := sdk.NewClient()
		result, err := client.IssueToken(context.Background(), tunnelID, tunnelScope)
		if err != nil {
			return err
		}
		printKV([][2]string{
			{i18n.T(i18n.Msg.Tunnel.TunnelID), result.TunnelID},
			{i18n.T(i18n.Msg.Tunnel.Scope), result.Scope},
			{i18n.T(i18n.Msg.Tunnel.Token), result.Token},
		})
		return nil
	}),
}

var setCmd = &cobra.Command{
	Use:   "set [tunnel-id]",
	Short: i18n.T(i18n.Msg.Tunnel.SetShort),
	Args:  cobra.ExactArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		if err := validateTunnelIDLocal(args[0]); err != nil {
			return err
		}
		client := sdk.NewClient()
		if _, err := client.ShowTunnel(context.Background(), args[0]); err != nil {
			if code, ok := devbridge.IsAPIError(err); ok && code == TunnelNotFoundCode {
				return fmt.Errorf("%s: %s", i18n.T(i18n.Msg.Tunnel.TunnelNotFound), args[0])
			}
			return err
		}
		if err := config.StoreDefaultTunnel(args[0]); err != nil {
			return err
		}
		fmt.Printf("%s: %s\n", i18n.T(i18n.Msg.Tunnel.DefaultTunnelSet), args[0])
		return nil
	}),
}

var unsetCmd = &cobra.Command{
	Use:   "unset",
	Short: i18n.T(i18n.Msg.Tunnel.UnsetShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		_ = config.DeleteDefaultTunnel()
		fmt.Println(i18n.T(i18n.Msg.Tunnel.DefaultTunnelUnset))
		return nil
	}),
}

func getTunnelIDArg(args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}
	return config.LoadDefaultTunnel()
}

func init() {
	createCmd.Flags().StringVarP(&tunnelDescription, "description", "d", "", i18n.T(i18n.Msg.Tunnel.FlagDescription))
	createCmd.Flags().IntVarP(&tunnelExpiration, "expiration", "e", 0, i18n.T(i18n.Msg.Tunnel.FlagExpiration))

	updateCmd.Flags().StringVarP(&tunnelName, "name", "n", "", i18n.T(i18n.Msg.Tunnel.FlagName))
	updateCmd.Flags().StringVarP(&tunnelDescription, "description", "d", "", i18n.T(i18n.Msg.Tunnel.FlagDescription))
	updateCmd.Flags().IntVarP(&tunnelExpiration, "expiration", "e", 0, i18n.T(i18n.Msg.Tunnel.FlagExpiration))

	tokenCmd.Flags().StringVarP(&tunnelScope, "scope", "s", "", i18n.T(i18n.Msg.Tunnel.FlagScope))
	_ = tokenCmd.MarkFlagRequired("scope")

	RootCmd.AddCommand(listCmd, createCmd, showCmd, updateCmd, deleteCmd, deleteAllCmd, tokenCmd, setCmd, unsetCmd)
}
