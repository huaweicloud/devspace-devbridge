package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"huawei.com/devbridge/internal/api"
	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var verbose bool

var version = "dev"

var RootCmd = &cobra.Command{
	Use:   "devbridge",
	Short: i18n.T(i18n.Msg.Common.VersionInfo),
	Long:  i18n.T(i18n.Msg.Common.VersionInfo),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		api.InitClient("")
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: i18n.T(i18n.Msg.Common.VersionInfo),
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func runError(fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := fn(cmd, args)
		if err != nil {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.PrintErrln(err)
		}
		return err
	}
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, i18n.T(i18n.Msg.Common.FlagVerbose))
	RootCmd.AddCommand(versionCmd)
}
