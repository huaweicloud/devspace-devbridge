package main

import (
	"log"
	"os"
	"path/filepath"

	"huawei.com/devbridge/cmd"
)

func main() {
	cmd.RootCmd.Use = filepath.Base(os.Args[0])
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
