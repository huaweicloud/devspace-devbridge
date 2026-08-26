package cmd

import (
	"fmt"
	"strconv"

	"huawei.com/devbridge/internal/api"
	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var (
	tunnelDescription string //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	tunnelExpiration  int    //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	tunnelName        string //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	tunnelScope       string //nolint:gochecknoglobals // cobra CLI 惯用全局变量
)

var listCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "list",
	Short: i18n.T(i18n.Msg.Tunnel.ListShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnels, err := api.ListTunnels()
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
				t.TunnelID,
				t.Name,
				t.Description,
				FormatTunnelRemaining(int64(t.TunnelExpiration)),
				strconv.Itoa(t.PortCount),
			})
		}
		printTable(headers, rows)
		return nil
	}),
}

var createCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "create [name]",
	Short: i18n.T(i18n.Msg.Tunnel.CreateShort),
	Args:  cobra.ExactArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		var exp *int
		if cmd.Flags().Changed("expiration") {
			exp = &tunnelExpiration
		}
		result, err := api.CreateTunnel(args[0], tunnelDescription, exp)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Tunnel.CreateFailed), err)
		}
		printKV([][2]string{
			{i18n.T(i18n.Msg.Tunnel.TunnelID), result.TunnelID},
			{i18n.T(i18n.Msg.Tunnel.Name), result.Name},
			{i18n.T(i18n.Msg.Tunnel.Description), result.Description},
			{i18n.T(i18n.Msg.Tunnel.TunnelExpiration), FormatTunnelExpiration(int64(result.ExpirationHours))},
		})
		return nil
	}),
}

var showCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "show [tunnel-id]",
	Short: i18n.T(i18n.Msg.Tunnel.ShowShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		result, err := api.ShowTunnel(tunnelID)
		if err != nil {
			return err
		}
		var status api.TunnelStatus
		if result.Status != nil {
			status = *result.Status
		}
		printKV([][2]string{
			{i18n.T(i18n.Msg.Tunnel.TunnelID), result.TunnelID},
			{i18n.T(i18n.Msg.Tunnel.Name), result.Name},
			{i18n.T(i18n.Msg.Tunnel.TunnelExpiration), FormatTunnelRemaining(int64(result.TunnelExpiration))},
			{i18n.T(i18n.Msg.Tunnel.Description), result.Description},
			{i18n.T(i18n.Msg.Tunnel.ClientConnectionCount), fmt.Sprintf("%d", status.ClientConnectionCount)},
			{i18n.T(i18n.Msg.Tunnel.HostConnectionCount), fmt.Sprintf("%d", status.HostConnectionCount)},
			{i18n.T(i18n.Msg.Tunnel.TotalUploadBytes), formatBytes(status.TotalUploadBytes)},
			{i18n.T(i18n.Msg.Tunnel.TotalDownloadBytes), formatBytes(status.TotalDownloadBytes)},
		})
		return nil
	}),
}

var updateCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
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
		if err := api.UpdateTunnel(tunnelID, name, desc, exp); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Tunnel.UpdateFailed), err)
		}
		fmt.Println(i18n.T(i18n.Msg.Tunnel.TunnelUpdated))
		return nil
	}),
}

var deleteCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "delete [tunnel-id]",
	Short: i18n.T(i18n.Msg.Tunnel.DeleteShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		if err := api.DeleteTunnel(tunnelID); err != nil {
			return err
		}
		if def, e := config.LoadDefaultTunnel(); e == nil && def == tunnelID {
			_ = config.DeleteDefaultTunnel() //nolint:errcheck // keyring 中不存在时删除失败属正常
			fmt.Println(i18n.T(i18n.Msg.Tunnel.DefaultTunnelCleared))
		}
		fmt.Println(i18n.T(i18n.Msg.Tunnel.TunnelDeleted))
		return nil
	}),
}

var deleteAllCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "delete-all",
	Short: i18n.T(i18n.Msg.Tunnel.DeleteAllShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		if err := api.DeleteAllTunnels(); err != nil {
			return err
		}
		fmt.Println(i18n.T(i18n.Msg.Tunnel.TunnelDeletedAll))
		return nil
	}),
}

var tokenCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "token [tunnel-id]",
	Short: i18n.T(i18n.Msg.Tunnel.TokenIssueShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		result, err := api.TunnelToken(tunnelID, tunnelScope)
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

var setCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "set [tunnel-id]",
	Short: i18n.T(i18n.Msg.Tunnel.SetShort),
	Args:  cobra.ExactArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		if err := api.ValidateTunnelID(args[0]); err != nil {
			return err
		}
		if _, err := api.ShowTunnel(args[0]); err != nil {
			if api.GetAPIErrorCode(err) == api.TunnelNotFoundCode {
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

var unsetCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "unset",
	Short: i18n.T(i18n.Msg.Tunnel.UnsetShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		_ = config.DeleteDefaultTunnel() //nolint:errcheck // keyring 中不存在时删除失败属正常
		fmt.Println(i18n.T(i18n.Msg.Tunnel.DefaultTunnelUnset))
		return nil
	}),
}

func getTunnelIDArg(args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}
	return api.ResolveTunnelID("")
}

func init() { //nolint:gochecknoinits // cobra CLI 惯用 init 函数
	createCmd.Flags().StringVarP(&tunnelDescription, "description", "d", "", i18n.T(i18n.Msg.Tunnel.FlagDescription))
	createCmd.Flags().IntVarP(&tunnelExpiration, "expiration", "e", 0, i18n.T(i18n.Msg.Tunnel.FlagExpiration))

	updateCmd.Flags().StringVarP(&tunnelName, "name", "n", "", i18n.T(i18n.Msg.Tunnel.FlagName))
	updateCmd.Flags().StringVarP(&tunnelDescription, "description", "d", "", i18n.T(i18n.Msg.Tunnel.FlagDescription))
	updateCmd.Flags().IntVarP(&tunnelExpiration, "expiration", "e", 0, i18n.T(i18n.Msg.Tunnel.FlagExpiration))

	tokenCmd.Flags().StringVarP(&tunnelScope, "scope", "s", "", i18n.T(i18n.Msg.Tunnel.FlagScope))
	_ = tokenCmd.MarkFlagRequired("scope") //nolint:errcheck // flag 名固定，不会失败

	RootCmd.AddCommand(listCmd, createCmd, showCmd, updateCmd, deleteCmd, deleteAllCmd, tokenCmd, setCmd, unsetCmd)
}
