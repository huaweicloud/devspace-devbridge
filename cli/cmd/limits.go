package cmd

import (
	"fmt"

	"huawei.com/devbridge/internal/api"
	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var limitsCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	Use:   "limits",
	Short: i18n.T(i18n.Msg.Limits.LimitsShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		result, err := api.GetLimits()
		if err != nil {
			return err
		}
		printLimitsDetail(result)
		return nil
	}),
}

func printLimitsDetail(l *api.LimitsResult) {
	current := l.QuotaBytes - l.RemainingBytes
	currentStr := formatBytes(current)
	if l.QuotaBytes > 0 {
		percent := float64(current) * 100 / float64(l.QuotaBytes)
		currentStr = fmt.Sprintf("%s (%.0f%%)", currentStr, percent)
	}
	printKV([][2]string{
		{i18n.T(i18n.Msg.Limits.QuotaResetAt), formatTime(l.ResetAt)},
		{i18n.T(i18n.Msg.Limits.QuotaBytes), formatBytes(l.QuotaBytes)},
		{i18n.T(i18n.Msg.Limits.Current), currentStr},
		{i18n.T(i18n.Msg.Limits.ActiveTunnels), fmt.Sprintf("%d", l.ActiveTunnels)},
		{i18n.T(i18n.Msg.Limits.MaxTunnels), fmt.Sprintf("%d", l.MaxTunnels)},
		{i18n.T(i18n.Msg.Limits.MaxPortsPerTunnel), fmt.Sprintf("%d", l.MaxPortsPerTunnel)},
		{i18n.T(i18n.Msg.Limits.MaxHostsPerTunnel), fmt.Sprintf("%d", l.MaxHostsPerTunnel)},
		{i18n.T(i18n.Msg.Limits.MaxTunnelBandwidth), formatBytes(l.MaxTunnelBandwidthBytesPerSecond) + "/s"},
		{i18n.T(i18n.Msg.Limits.MaxHTTPRequestsPerMinutePerPort), fmt.Sprintf("%d", l.MaxHTTPRequestsPerMinutePerPort)},
		{i18n.T(i18n.Msg.Limits.MaxConnectionsPerPort), fmt.Sprintf("%d", l.MaxConnectionsPerPort)},
	})
}

func init() { //nolint:gochecknoinits // cobra CLI 惯用 init 函数
	RootCmd.AddCommand(limitsCmd)
}
