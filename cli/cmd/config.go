/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package cmd

import (
	"fmt"

	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var (
	cfgGatewayAddr string
	cfgGatewayHost string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: i18n.T(i18n.Msg.Config.ConfigCommands),
}

// configGetCmd shows the gateway configuration; when empty it displays the build-time defaults.
var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: i18n.T(i18n.Msg.Config.GetShort),
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		gatewayAddr := config.LoadGatewayAddr()
		gatewayHost := config.LoadGatewayHost()

		printKV([][2]string{
			{i18n.T(i18n.Msg.Config.GatewayAddr), gatewayAddr},
			{i18n.T(i18n.Msg.Config.GatewayHost), gatewayHost},
		})
	},
}

// configSetCmd sets the gateway configuration.
var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: i18n.T(i18n.Msg.Config.SetShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		changed := false
		if cfgGatewayAddr != "" {
			if err := config.StoreGatewayAddr(cfgGatewayAddr); err != nil {
				return err
			}
			changed = true
		}
		if cfgGatewayHost != "" {
			if err := config.StoreGatewayHost(cfgGatewayHost); err != nil {
				return err
			}
			changed = true
		}
		if !changed {
			return fmt.Errorf("%s", i18n.T(i18n.Msg.Config.NothingToSet))
		}
		fmt.Println(i18n.T(i18n.Msg.Config.SetSuccess))
		return nil
	}),
}

// configUnsetCmd clears the gateway configuration, restoring the build-time defaults.
var configUnsetCmd = &cobra.Command{
	Use:   "unset",
	Short: i18n.T(i18n.Msg.Config.UnsetShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		if err := config.DeleteGatewayAddr(); err != nil {
			return err
		}
		if err := config.DeleteGatewayHost(); err != nil {
			return err
		}
		fmt.Println(i18n.T(i18n.Msg.Config.SetSuccess))
		return nil
	}),
}

func init() {
	configSetCmd.Flags().StringVar(&cfgGatewayAddr, "gateway-addr", "", "gateway address (host:port)")
	configSetCmd.Flags().StringVar(&cfgGatewayHost, "gateway-host", "", "gateway SNI host")
	configCmd.AddCommand(configGetCmd, configSetCmd, configUnsetCmd)
	RootCmd.AddCommand(configCmd)
}
