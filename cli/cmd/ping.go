package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"huawei.com/devbridge/internal/i18n"
	"huawei.com/devbridge/internal/netutil"

	"github.com/spf13/cobra"
)

var pingInterval int

var pingCmd = &cobra.Command{
	Use:   "ping <uri>",
	Short: i18n.T(i18n.Msg.Ping.PingShort),
	Args:  cobra.ExactArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		uri := args[0]
		parsedURL, err := url.Parse(uri)
		if err != nil {
			return fmt.Errorf("%s: %s (%v)", i18n.T(i18n.Msg.Ping.URIInvalid), uri, err)
		}
		scheme := strings.ToUpper(parsedURL.Scheme)
		if scheme == "" {
			scheme = "UNKNOWN"
		}
		interval := time.Duration(pingInterval) * time.Millisecond

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				result := netutil.PingURI(uri, 10*time.Second)
				statusText := result.StatusText
				if result.Err != nil {
					fmt.Printf("%s %s -- %d ms (err: %v)\n",
						scheme, statusText,
						result.Latency.Milliseconds(), result.Err)
					return nil
				} else {
					fmt.Printf("%s %s -- %d ms\n",
						scheme, statusText,
						result.Latency.Milliseconds())
				}
			}
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
		}
	}),
}

func init() {
	pingCmd.Flags().IntVarP(&pingInterval, "interval", "i", 1000, i18n.T(i18n.Msg.Common.FlagPingInterval))
	RootCmd.AddCommand(pingCmd)
}
