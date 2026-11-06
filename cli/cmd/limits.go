package cmd

import (
	"context"
	"fmt"

	devbridge "github.com/huaweicloud/devspace-devbridge/go-sdk"
	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var limitsCmd = &cobra.Command{
	Use:   "limits",
	Short: i18n.T(i18n.Msg.Limits.LimitsShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		client := newSDKClient()
		result, err := client.GetLimits(context.Background())
		if err != nil {
			return err
		}
		printLimitsDetail(result)
		return nil
	}),
}

func printLimitsDetail(l *devbridge.Limits) {
	printKV(limitsDetailKV(l))
}

func limitsDetailKV(l *devbridge.Limits) [][2]string {
	current := l.QuotaBytes - l.RemainingBytes
	currentStr := formatBytes(current)
	if l.QuotaBytes > 0 {
		percent := float64(current) * 100 / float64(l.QuotaBytes)
		currentStr = fmt.Sprintf("%s (%.0f%%)", currentStr, percent)
	}
	return [][2]string{
		{i18n.T(i18n.Msg.Limits.QuotaResetAt), formatTime(l.ResetAt)},
		{i18n.T(i18n.Msg.Limits.QuotaBytes), formatBytes(l.QuotaBytes)},
		{i18n.T(i18n.Msg.Limits.Current), currentStr},
		{i18n.T(i18n.Msg.Limits.ActiveTunnels), fmt.Sprintf("%d", l.ActiveTunnels)},
		{i18n.T(i18n.Msg.Limits.MaxTunnels), fmt.Sprintf("%d", l.MaxTunnels)},
		{i18n.T(i18n.Msg.Limits.MaxPortsPerTunnel), fmt.Sprintf("%d", l.MaxPortsPerTunnel)},
		{i18n.T(i18n.Msg.Limits.MaxHostsPerTunnel), fmt.Sprintf("%d", l.MaxHostsPerTunnel)},
		{i18n.T(i18n.Msg.Limits.MaxTunnelBandwidth), formatBandwidthLimit(l.HasMaxTunnelBandwidth(), l.MaxTunnelBandwidthBytesPerSecond)},
		{i18n.T(i18n.Msg.Limits.MaxHTTPRequestsPerMinutePerPort), formatCountLimit(l.HasMaxHTTPRequestsPerMinutePerPort(), l.MaxHTTPRequestsPerMinutePerPort)},
		{i18n.T(i18n.Msg.Limits.MaxConnectionsPerPort), formatCountLimit(l.HasMaxConnectionsPerPort(), l.MaxConnectionsPerPort)},
	}
}

// formatCountLimit renders a per-port limit. A field the server did not send is
// reported as such instead of being shown as an indistinguishable zero; a
// negative value means the server does not cap the resource.
func formatCountLimit(delivered bool, value int32) string {
	switch {
	case !delivered:
		return i18n.T(i18n.Msg.Limits.NotProvided)
	case value < 0:
		return i18n.T(i18n.Msg.Limits.Unlimited)
	default:
		return fmt.Sprintf("%d", value)
	}
}

// formatBandwidthLimit renders the tunnel bandwidth limit as bytes per second.
func formatBandwidthLimit(delivered bool, bytesPerSecond int64) string {
	switch {
	case !delivered:
		return i18n.T(i18n.Msg.Limits.NotProvided)
	case bytesPerSecond < 0:
		return i18n.T(i18n.Msg.Limits.Unlimited)
	default:
		return formatBytes(bytesPerSecond) + "/s"
	}
}

func init() {
	RootCmd.AddCommand(limitsCmd)
}
