package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

// confirmReader is the reader used by confirmOrAbort to read the y/N answer.
// It is a package-level variable so tests can inject a fake input source.
var confirmReader *bufio.Reader

// confirmIsTTY reports whether the given file is attached to an interactive
// terminal. It is a package-level variable so tests can override the TTY
// detection without needing a real terminal.
var confirmIsTTY = func(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// confirmOrAbort guards a destructive command behind a confirmation gate.
//
// The prompt is shown when the command runs in an interactive terminal
// (stdin is a TTY); the user must answer y/Y to proceed. Any other answer
// (including an empty line) aborts the operation with a "cancelled" error.
//
// The --yes/-y flag (looked up on cmd via flagNameYes) skips the prompt
// entirely, which is required in non-interactive environments: when stdin is
// not a TTY and --yes was not passed, the operation is refused with an error
// telling the user to re-run with --yes.
func confirmOrAbort(cmd *cobra.Command, flagNameYes, prompt string) error {
	yes, err := cmd.Flags().GetBool(flagNameYes)
	if err == nil && yes {
		return nil
	}
	if !confirmIsTTY(os.Stdin) {
		return fmt.Errorf("%s", i18n.T(i18n.Msg.Tunnel.ConfirmNonInteractive))
	}
	fmt.Print(prompt)
	if confirmReader == nil {
		confirmReader = bufio.NewReader(os.Stdin)
	}
	line, err := confirmReader.ReadString('\n')
	if err != nil && line == "" {
		return fmt.Errorf("%s", i18n.T(i18n.Msg.Tunnel.DeleteAborted))
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "y" || answer == "yes" {
		return nil
	}
	return fmt.Errorf("%s", i18n.T(i18n.Msg.Tunnel.DeleteAborted))
}
