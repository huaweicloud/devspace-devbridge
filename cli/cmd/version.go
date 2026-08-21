package cmd

import (
	"fmt"

	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: i18n.T(i18n.Msg.Common.VersionInfo),
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
