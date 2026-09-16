/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package cmd

import (
	"fmt"
	"strconv"

	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var (
	cfgGatewayAddr   string
	cfgGatewayHost   string
	cfgTLSSkipVerify bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: i18n.T(i18n.Msg.Config.ConfigCommands),
}

// configGetCmd 查看网关配置。配置为空时显示编译时默认值。
var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: i18n.T(i18n.Msg.Config.GetShort),
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		gatewayAddr := config.LoadGatewayAddr()
		gatewayHost := config.LoadGatewayHost()
		tlsSkipVerify := config.LoadTLSSkipVerify()

		printKV([][2]string{
			{i18n.T(i18n.Msg.Config.GatewayAddr), gatewayAddr},
			{i18n.T(i18n.Msg.Config.GatewayHost), gatewayHost},
			{"Skip TLS verification", strconv.FormatBool(tlsSkipVerify)},
		})
	},
}

// configSetCmd 设置网关配置。
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
		if cmd.Flags().Changed("tls-skip-verify") {
			if err := config.StoreTLSSkipVerify(cfgTLSSkipVerify); err != nil {
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

// configUnsetCmd 清空网关配置，恢复编译时默认值。
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
		if err := config.DeleteTLSSkipVerify(); err != nil {
			return err
		}
		fmt.Println(i18n.T(i18n.Msg.Config.SetSuccess))
		return nil
	}),
}

func init() {
	configSetCmd.Flags().StringVar(&cfgGatewayAddr, "gateway-addr", "", "网关地址（host:port）")
	configSetCmd.Flags().StringVar(&cfgGatewayHost, "gateway-host", "", "网关 SNI 域名")
	configSetCmd.Flags().BoolVar(&cfgTLSSkipVerify, "tls-skip-verify", false, "跳过网关 TLS 证书校验（true/false）")
	configCmd.AddCommand(configGetCmd, configSetCmd, configUnsetCmd)
	RootCmd.AddCommand(configCmd)
}
