package cmd

import (
	"context"
	"fmt"
	"strconv"

	"huawei.com/devbridge/internal/i18n"
	"huawei.com/devbridge/internal/sdk"

	"github.com/spf13/cobra"
)

var (
	portNumber    int
	portProtocol  string
	portAllowAnon bool
	portDenyAnon  bool
)

var validProtocols = map[string]bool{
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

var portCmd = &cobra.Command{
	Use:   "port",
	Short: i18n.T(i18n.Msg.Port.PortShort),
}

var portCreateCmd = &cobra.Command{
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
		client, err := sdk.NewClient()
		if err != nil {
			return err
		}
		if err := client.CreatePort(context.Background(), tunnelID, portNumber, portProtocol, resolveAllowAnon(cmd)); err != nil {
			return fmt.Errorf("Failed to add port %d: %w", portNumber, err)
		}
		fmt.Println(i18n.T(i18n.Msg.Port.PortCreated))
		return nil
	}),
}

var portListCmd = &cobra.Command{
	Use:   "list [tunnel-id]",
	Short: i18n.T(i18n.Msg.Port.PortListShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		client, err := sdk.NewClient()
		if err != nil {
			return err
		}
		ports, err := client.ListPorts(context.Background(), tunnelID)
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

var portShowCmd = &cobra.Command{
	Use:   "show [tunnel-id]",
	Short: i18n.T(i18n.Msg.Port.PortShowShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		client, err := sdk.NewClient()
		if err != nil {
			return err
		}
		result, err := client.ShowPort(context.Background(), tunnelID, portNumber)
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

var portUpdateCmd = &cobra.Command{
	Use:   "update [tunnel-id]",
	Short: i18n.T(i18n.Msg.Port.PortUpdateShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		client, err := sdk.NewClient()
		if err != nil {
			return err
		}
		if err := client.UpdatePort(context.Background(), tunnelID, portNumber, resolveAllowAnon(cmd)); err != nil {
			return err
		}
		fmt.Println(i18n.T(i18n.Msg.Port.PortUpdated))
		return nil
	}),
}

var portDeleteCmd = &cobra.Command{
	Use:   "delete [tunnel-id]",
	Short: i18n.T(i18n.Msg.Port.PortDeleteShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		tunnelID, err := getTunnelIDArg(args)
		if err != nil {
			return err
		}
		client, err := sdk.NewClient()
		if err != nil {
			return err
		}
		if err := client.DeletePort(context.Background(), tunnelID, portNumber); err != nil {
			return fmt.Errorf("Failed to delete port %d: %w", portNumber, err)
		}
		fmt.Printf("Port %d removed from tunnel %s.\n", portNumber, tunnelID)
		return nil
	}),
}

func init() {
	portCreateCmd.Flags().IntVarP(&portNumber, "port-number", "p", 0, i18n.T(i18n.Msg.Port.FlagPortRequired))
	portCreateCmd.Flags().StringVar(&portProtocol, "protocol", "auto", i18n.T(i18n.Msg.Port.FlagProtocol))
	portCreateCmd.Flags().BoolVarP(&portAllowAnon, "allow-anonymous", "a", false, i18n.T(i18n.Msg.Port.FlagAllowAnon))
	portCreateCmd.Flags().BoolVar(&portDenyAnon, "deny-anonymous", false, i18n.T(i18n.Msg.Port.FlagDenyAnon))
	_ = portCreateCmd.MarkFlagRequired("port-number")

	portShowCmd.Flags().IntVarP(&portNumber, "port-number", "p", 0, i18n.T(i18n.Msg.Port.FlagPortNumber))
	_ = portShowCmd.MarkFlagRequired("port-number")

	portUpdateCmd.Flags().IntVarP(&portNumber, "port-number", "p", 0, i18n.T(i18n.Msg.Port.FlagPortNumber))
	portUpdateCmd.Flags().BoolVarP(&portAllowAnon, "allow-anonymous", "a", false, i18n.T(i18n.Msg.Port.FlagAllowAnon))
	portUpdateCmd.Flags().BoolVar(&portDenyAnon, "deny-anonymous", false, i18n.T(i18n.Msg.Port.FlagDenyAnon))
	_ = portUpdateCmd.MarkFlagRequired("port-number")

	portDeleteCmd.Flags().IntVarP(&portNumber, "port-number", "p", 0, i18n.T(i18n.Msg.Port.FlagPortNumber))
	_ = portDeleteCmd.MarkFlagRequired("port-number")

	portCmd.AddCommand(portCreateCmd, portListCmd, portShowCmd, portUpdateCmd, portDeleteCmd)
	RootCmd.AddCommand(portCmd)
}
