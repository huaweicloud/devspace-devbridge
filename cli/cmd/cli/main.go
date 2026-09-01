package main

import (
	"os"
	"path/filepath"

	"huawei.com/devbridge/cmd"
)

func main() {
	cmd.RootCmd.Use = filepath.Base(os.Args[0])
	if err := cmd.RootCmd.Execute(); err != nil {
		// Error already printed by runError wrapper in root.go;
		// only need to set a non-zero exit code here.
		os.Exit(1)
	}
}
