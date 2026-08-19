package cmd

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var (
	echoPort      int
	echoInterface string
)

var echoCmd = &cobra.Command{
	Use:   "echo [http]",
	Short: i18n.T(i18n.Msg.Echo.EchoShort),
	Args:  cobra.MaximumNArgs(1),
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && args[0] != "http" {
			return fmt.Errorf("%s: %s", i18n.T(i18n.Msg.Port.ProtocolInvalid), args[0])
		}
		// Only validate port when -p is explicitly provided.
		// When -p is omitted, echoPort stays 0 and net.Listen picks a random free port.
		if cmd.Flags().Changed("port") && (echoPort < 1 || echoPort > 65535) {
			return fmt.Errorf("Invalid port number %d (Port must be between 1 and 65535)", echoPort)
		}
		addr := echoInterface
		if addr == "" {
			addr = "127.0.0.1"
		}
		listenAddr := fmt.Sprintf("%s:%d", addr, echoPort)
		return runHttpEcho(listenAddr)
	}),
}

func runHttpEcho(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("%s at: http://%s\n", i18n.T(i18n.Msg.Echo.EchoStarted), ln.Addr().String())
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHttpEcho)
	server := &http.Server{Handler: mux}
	return server.Serve(ln)
}

func handleHttpEcho(w http.ResponseWriter, r *http.Request) {
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
	_, _ = w.Write([]byte(sb.String()))
}

func init() {
	echoCmd.Flags().IntVarP(&echoPort, "port", "p", 0, i18n.T(i18n.Msg.Common.FlagEchoPort))
	echoCmd.Flags().StringVarP(&echoInterface, "interface", "i", "127.0.0.1", i18n.T(i18n.Msg.Common.FlagInterface))
	RootCmd.AddCommand(echoCmd)
}
