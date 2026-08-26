package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"huawei.com/devbridge/internal/i18n"
	"huawei.com/devbridge/internal/netutil"

	"github.com/spf13/cobra"
)

var (
	echoPort      int    //nolint:gochecknoglobals
	echoInterface string //nolint:gochecknoglobals
)

var pingInterval int //nolint:gochecknoglobals

var echoCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "echo",
	Short: i18n.T(i18n.Msg.Echo.EchoShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {

		if cmd.Flags().Changed("port") && (echoPort < 1 || echoPort > 65535) {
			return fmt.Errorf("Invalid port number %d (Port must be between 1 and 65535)", echoPort)
		}
		addr := echoInterface
		if addr == "" {
			addr = "127.0.0.1"
		}
		listenAddr := fmt.Sprintf("%s:%d", addr, echoPort)
		return runHTTPEcho(listenAddr)
	}),
}

var pingCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "ping <uri>",
	Short: i18n.T(i18n.Msg.Ping.PingShort),
	Args:  cobra.ExactArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		uri := args[0]
		parsedURL, err := url.Parse(uri)
		if err != nil {
			return fmt.Errorf("%s: %s (%w)", i18n.T(i18n.Msg.Ping.URIInvalid), uri, err)
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

func runHTTPEcho(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("%s at: http://%s\n", i18n.T(i18n.Msg.Echo.EchoStarted), ln.Addr().String())
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHTTPEcho)
	server := &http.Server{Handler: mux}
	return server.Serve(ln)
}

func handleHTTPEcho(w http.ResponseWriter, r *http.Request) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", i18n.T(i18n.Msg.Echo.Method), r.Method))
	sb.WriteString(fmt.Sprintf("%s: %s\n", i18n.T(i18n.Msg.Echo.URL), r.URL.String()))
	sb.WriteString(fmt.Sprintf("%s: %s\n", i18n.T(i18n.Msg.Echo.Host), r.Host))
	sb.WriteString(fmt.Sprintf("%s: %s\n", i18n.T(i18n.Msg.Echo.RemoteAddr), r.RemoteAddr))
	sb.WriteString(fmt.Sprintf("%s: %s\n", i18n.T(i18n.Msg.Echo.Proto), r.Proto))
	sb.WriteString(fmt.Sprintf("%s:\n", i18n.T(i18n.Msg.Echo.Headers)))
	for k, v := range r.Header {
		sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sb.String())) //nolint:errcheck
}

func init() { //nolint:gochecknoinits
	echoCmd.Flags().IntVarP(&echoPort, "port", "p", 0, i18n.T(i18n.Msg.Common.FlagEchoPort))
	echoCmd.Flags().StringVarP(&echoInterface, "interface", "i", "127.0.0.1", i18n.T(i18n.Msg.Common.FlagInterface))
	RootCmd.AddCommand(echoCmd)

	pingCmd.Flags().IntVarP(&pingInterval, "interval", "i", 1000, i18n.T(i18n.Msg.Common.FlagPingInterval))
	RootCmd.AddCommand(pingCmd)
}
