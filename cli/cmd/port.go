package cmd

import (
	"fmt"
	"strconv"

	"huawei.com/devbridge/internal/api"
	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var (
	portNumber    int    //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	portProtocol  string //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	portAllowAnon bool   //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	portDenyAnon  bool   //nolint:gochecknoglobals // cobra CLI 惯用全局变量
)

var validProtocols = map[string]bool{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	"http":  true,
	"https": true,
	"auto":  true,
}

func validateProtocolLocal(protocol string) error {
	if !validProtocols[protocol] {
		return fmt.Errorf("Protocol must be one of http, https, auto, got: %s", protocol)
	}
	return nil
}

// resolveAllowAnon 根据 --allow-anonymous/--deny-anonymous 标志解析 allowAnonymous 值。
// 未指定任何标志时默认 false。
func resolveAllowAnon(cmd *cobra.Command) *bool {
	if cmd.Flags().Changed("allow-anonymous") {
		v := true
		return &v
	}
	if cmd.Flags().Changed("deny-anonymous") {
		v := false
		return &v
	}
	v := false
	return &v
}

var portCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "port",
	Short: i18n.T(i18n.Msg.Port.PortShort),
}

var portCreateCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "create [tunnel-id]",
	Short: i18n.T(i18n.Msg.Port.PortCreateShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		if err := validateProtocolLocal(portProtocol); err != nil {
			return err
		}
		if err := api.CreatePort(tunnelID, portNumber, portProtocol, resolveAllowAnon(cmd)); err != nil {
			return fmt.Errorf("Failed to add port %d: %w", portNumber, err)
		}
		fmt.Println(i18n.T(i18n.Msg.Port.PortCreated))
		return nil
	}),
}

var portListCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "list [tunnel-id]",
	Short: i18n.T(i18n.Msg.Port.PortListShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		ports, err := api.ListPorts(tunnelID)
		if err != nil {
			return err
		}
		if len(ports) == 0 {
			fmt.Println(i18n.T(i18n.Msg.Port.PortListEmpty))
			return nil
		}
		headers := []string{
			i18n.T(i18n.Msg.Port.Port),
			i18n.T(i18n.Msg.Port.Protocol),
			i18n.T(i18n.Msg.Port.AllowAnonymous),
			i18n.T(i18n.Msg.Port.TunnelID),
		}
		var rows [][]string
		for _, p := range ports {
			rows = append(rows, []string{
				strconv.Itoa(int(p.Port)),
				p.Protocol,
				strconv.FormatBool(p.AllowAnonymous),
				p.TunnelID,
			})
		}
		printTable(headers, rows)
		return nil
	}),
}

var portShowCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "show [tunnel-id]",
	Short: i18n.T(i18n.Msg.Port.PortShowShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		result, err := api.ShowPort(tunnelID, portNumber)
		if err != nil {
			return err
		}
		printKV([][2]string{
			{i18n.T(i18n.Msg.Tunnel.TunnelID), result.TunnelID},
			{i18n.T(i18n.Msg.Port.Port), strconv.Itoa(int(result.Port))},
			{i18n.T(i18n.Msg.Port.Protocol), result.Protocol},
			{i18n.T(i18n.Msg.Port.AllowAnonymous), strconv.FormatBool(result.AllowAnonymous)},
		})
		return nil
	}),
}

var portUpdateCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "update [tunnel-id]",
	Short: i18n.T(i18n.Msg.Port.PortUpdateShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		if err := api.UpdatePort(tunnelID, portNumber, resolveAllowAnon(cmd)); err != nil {
			return err
		}
		fmt.Println(i18n.T(i18n.Msg.Port.PortUpdated))
		return nil
	}),
}

var portDeleteCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "delete [tunnel-id]",
	Short: i18n.T(i18n.Msg.Port.PortDeleteShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		if err := api.DeletePort(tunnelID, portNumber); err != nil {
			return fmt.Errorf("Failed to delete port %d: %w", portNumber, err)
		}
		fmt.Printf("Port %d removed from tunnel %s.\n", portNumber, tunnelID)
		return nil
	}),
}

func init() { //nolint:gochecknoinits // cobra CLI 惯用 init 函数
	portCreateCmd.Flags().IntVarP(&portNumber, "port-number", "p", 0, i18n.T(i18n.Msg.Port.FlagPortRequired))
	portCreateCmd.Flags().StringVar(&portProtocol, "protocol", "auto", i18n.T(i18n.Msg.Port.FlagProtocol))
	portCreateCmd.Flags().BoolVarP(&portAllowAnon, "allow-anonymous", "a", false, i18n.T(i18n.Msg.Port.FlagAllowAnon))
	portCreateCmd.Flags().BoolVar(&portDenyAnon, "deny-anonymous", false, i18n.T(i18n.Msg.Port.FlagDenyAnon))
	_ = portCreateCmd.MarkFlagRequired("port-number") //nolint:errcheck // flag 名固定，不会失败

	portShowCmd.Flags().IntVarP(&portNumber, "port-number", "p", 0, i18n.T(i18n.Msg.Port.FlagPortNumber))
	_ = portShowCmd.MarkFlagRequired("port-number") //nolint:errcheck // flag 名固定，不会失败

	portUpdateCmd.Flags().IntVarP(&portNumber, "port-number", "p", 0, i18n.T(i18n.Msg.Port.FlagPortNumber))
	portUpdateCmd.Flags().BoolVarP(&portAllowAnon, "allow-anonymous", "a", false, i18n.T(i18n.Msg.Port.FlagAllowAnon))
	portUpdateCmd.Flags().BoolVar(&portDenyAnon, "deny-anonymous", false, i18n.T(i18n.Msg.Port.FlagDenyAnon))
	_ = portUpdateCmd.MarkFlagRequired("port-number") //nolint:errcheck // flag 名固定，不会失败

	portDeleteCmd.Flags().IntVarP(&portNumber, "port-number", "p", 0, i18n.T(i18n.Msg.Port.FlagPortNumber))
	_ = portDeleteCmd.MarkFlagRequired("port-number") //nolint:errcheck // flag 名固定，不会失败

	portCmd.AddCommand(portCreateCmd, portListCmd, portShowCmd, portUpdateCmd, portDeleteCmd)
	RootCmd.AddCommand(portCmd)
}
