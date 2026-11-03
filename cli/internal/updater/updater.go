/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package updater

import (
	"errors"
	"fmt"
	"runtime"

	devbridge "github.com/huaweicloud/devspace-devbridge/go-sdk"
)

const (
	obsInstallSh     = "https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.sh"
	obsInstallPs     = "https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.ps1"
	gitCodeInstallSh = "https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.sh"
	gitCodeInstallPs = "https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.ps1"
	githubInstallSh  = "https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.sh"
	githubInstallPs  = "https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.ps1"
)

// installCommand returns the one-liner install/upgrade commands.
func installCommand() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("  # OBS\n  irm %s | iex\n\n  # GitCode\n  irm %s | iex\n\n  # GitHub\n  irm %s | iex",
			obsInstallPs, gitCodeInstallPs, githubInstallPs)
	}
	return fmt.Sprintf("  # OBS\n  curl -fsSL %s | bash\n\n  # GitCode\n  curl -fsSL %s | bash\n\n  # GitHub\n  curl -fsSL %s | bash",
		obsInstallSh, gitCodeInstallSh, githubInstallSh)
}

// FormatError renders command errors for display. Upgrade-required errors are
// turned into an upgrade prompt matching the previous version-check notice.
func FormatError(err error) string {
	var ue *devbridge.UpgradeRequiredError
	if errors.As(err, &ue) && ue.Message != "" {
		return fmt.Sprintf("\n%s\nUpdate:\n%s\n", ue.Message, installCommand())
	}
	return err.Error()
}