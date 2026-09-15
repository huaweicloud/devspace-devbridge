package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"huawei.com/devbridge/internal/i18n"
	"huawei.com/devbridge/internal/updater"

	"github.com/spf13/cobra"
)

var verbose bool

var version = "dev"

var RootCmd = &cobra.Command{
	Use:   "devbridge",
	Short: i18n.T(i18n.Msg.Common.VersionInfo),
	Long:  i18n.T(i18n.Msg.Common.VersionInfo),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

		// Synchronous version check: print update notice to stderr if a newer
		// version is available. Uses a 24h cache so most invocations are instant.
		// Skip for the version command itself (it does its own sync check).
		if cmd.Name() != "version" {
			if result := updater.CheckSync(version); result != nil {
				if updater.IsNewer(version, result.LatestVersion) {
					fmt.Fprintf(os.Stderr, "\nA new version is available: %s (current: %s)\nUpdate:\n%s\n\n",
						result.LatestVersion, version, updater.InstallCommand())
				}
			}
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: i18n.T(i18n.Msg.Common.VersionInfo),
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
		if result := updater.CheckSync(version); result != nil {
			if updater.IsNewer(version, result.LatestVersion) {
				// Update notice goes to stderr so stdout contains only the
				// version number, keeping CI version checks reliable.
				fmt.Fprintf(os.Stderr, "\nA new version is available: %s (current: %s)\nUpdate:\n%s\n",
					result.LatestVersion, version, updater.InstallCommand())
			}
		}
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
