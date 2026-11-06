package cmd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	devbridge "github.com/huaweicloud/devspace-devbridge/go-sdk"
	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

// TunnelNotFoundCode is the "tunnel not found" error code agreed with the server.
const TunnelNotFoundCode = "10002"

// tunnelStatusStaleAfter is how old a reportedAt timestamp may be before the
// statistics are considered stale. The server collects status every minute,
// so 120s tolerates one missed collection cycle.
const tunnelStatusStaleAfter = 120 * time.Second

// tunnelStatusWarning returns a non-empty user-facing warning when the status
// is missing, has no reportedAt, or has not been refreshed within the stale
// threshold. It returns "" for fresh status.
func tunnelStatusWarning(status *devbridge.TunnelStatus, now time.Time) string {
	if status == nil {
		return i18n.T(i18n.Msg.Tunnel.StatusNotReported)
	}
	if status.ReportedAt == 0 {
		return i18n.T(i18n.Msg.Tunnel.StatusReportedAtMiss)
	}
	age := now.Unix() - status.ReportedAt
	if age > int64(tunnelStatusStaleAfter/time.Second) {
		return fmt.Sprintf(i18n.T(i18n.Msg.Tunnel.StatusStale), (time.Duration(age) * time.Second).String())
	}
	return ""
}

var (
	tunnelDescription string
	tunnelExpiration  int
	tunnelName        string
	tunnelScope       string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T(i18n.Msg.Tunnel.ListShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		client := newSDKClient()
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
		client := newSDKClient()
		result, err := client.CreateTunnel(context.Background(), args[0], tunnelDescription, exp)
		if err != nil {
			if errors.Is(err, devbridge.ErrInvalidTunnelName) {
				return errors.New(i18n.T(i18n.Msg.Tunnel.TunnelNameInvalid))
			}
			if errors.Is(err, devbridge.ErrInvalidTunnelDescription) {
				return errors.New(i18n.T(i18n.Msg.Tunnel.TunnelDescInvalid))
			}
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
		client := newSDKClient()
		result, err := client.ShowTunnel(context.Background(), tunnelID)
		if err != nil {
			return err
		}
		status := result.Status
		clientCount, hostCount, upload, download := "-", "-", "-", "-"
		if status != nil {
			clientCount = strconv.Itoa(status.ClientConnectionCount)
			hostCount = strconv.Itoa(status.HostConnectionCount)
			upload = formatBytes(status.TotalUploadBytes)
			download = formatBytes(status.TotalDownloadBytes)
		}
		kv := [][2]string{
			{i18n.T(i18n.Msg.Tunnel.TunnelID), result.ID},
			{i18n.T(i18n.Msg.Tunnel.Name), result.Name},
			{i18n.T(i18n.Msg.Tunnel.TunnelExpiration), formatTunnelRemaining(int64(result.TunnelExpiration))},
			{i18n.T(i18n.Msg.Tunnel.Description), result.Description},
			{i18n.T(i18n.Msg.Tunnel.ClientConnectionCount), clientCount},
			{i18n.T(i18n.Msg.Tunnel.HostConnectionCount), hostCount},
			{i18n.T(i18n.Msg.Tunnel.TotalUploadBytes), upload},
			{i18n.T(i18n.Msg.Tunnel.TotalDownloadBytes), download},
		}
		if status != nil && status.ReportedAt > 0 {
			kv = append(kv, [2]string{i18n.T(i18n.Msg.Tunnel.ReportedAt), formatTime(status.ReportedAt)})
		}
		printKV(kv)
		if warning := tunnelStatusWarning(status, time.Now()); warning != "" {
			fmt.Println(warning)
		}
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
		client := newSDKClient()
		if err := client.UpdateTunnel(context.Background(), tunnelID, name, desc, exp); err != nil {
			if errors.Is(err, devbridge.ErrInvalidTunnelName) {
				return errors.New(i18n.T(i18n.Msg.Tunnel.TunnelNameInvalid))
			}
			if errors.Is(err, devbridge.ErrInvalidTunnelDescription) {
				return errors.New(i18n.T(i18n.Msg.Tunnel.TunnelDescInvalid))
			}
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
		client := newSDKClient()
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
		client := newSDKClient()
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
		client := newSDKClient()
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
		client := newSDKClient()
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
