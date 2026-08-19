package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"huawei.com/devbridge/internal/api"
	"huawei.com/devbridge/internal/i18n"
	"huawei.com/devbridge/internal/logging"

	"github.com/spf13/cobra"
)

var verbose bool

var RootCmd = &cobra.Command{
	Use:   "devbridge",
	Short: i18n.T(i18n.Msg.Common.VersionInfo),
	Long:  i18n.T(i18n.Msg.Common.VersionInfo),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		api.InitClient("")
		if verbose {
			logging.SetLevel(slog.LevelDebug)
			slog.SetDefault(slog.New(logging.NewPlainHandler(os.Stderr)))
		}
	},
}

// runError wraps a RunE function to print errors without "Error:" prefix and suppress usage.
func runError(fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := fn(cmd, args)
		if err != nil {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			fmt.Fprintln(os.Stderr, err)
		}
		return err
	}
}
func init() {
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, i18n.T(i18n.Msg.Common.FlagVerbose))
}
