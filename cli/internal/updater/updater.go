/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

const (
	gitCodeAPI       = "https://gitcode.com/api/v5/repos/CloudDeveloperDepartment/devbrige/releases"
	gitCodeInstallSh = "https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.sh"
	gitCodeInstallPs = "https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.ps1"
	cacheTTL         = 24 * time.Hour
	httpTimeout      = 10 * time.Second
)

var (
	// Loose version matching: supports v0.1.9 / 0.1.9 / 0.1.13.333 / 0.1.3-release / 0.1.3.release etc.
	// Only parses the leading numeric segments (major.minor.patch[.build]), ignoring any suffix.
	versionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:\.(\d+))?`)
)

// CheckResult holds the version check result.
type CheckResult struct {
	LatestVersion string `json:"latestVersion"`
	LatestTag     string `json:"latestTag"`
	CheckedAt     int64  `json:"checkedAt"`
}

// gitcodeRelease represents a single release from the GitCode API.
type gitcodeRelease struct {
	TagName string `json:"tag_name"`
}

// IsNewer compares current version with latest. Returns true if latest > current.
// Version strings can be "0.1.9", "0.1.13.333" (four-segment build number),
// "0.1.3-release", "0.1.3.release" — only the leading major.minor.patch[.build]
// segments are compared; suffixes are ignored.
func IsNewer(current, latest string) bool {
	return isNewerRaw(current, latest)
}

func parseVersion(v string) (major, minor, patch, build int, ok bool) {
	m := versionRegex.FindStringSubmatch(v)
	if m == nil {
		return 0, 0, 0, 0, false
	}
	major, _ = strconv.Atoi(m[1])
	minor, _ = strconv.Atoi(m[2])
	patch, _ = strconv.Atoi(m[3])
	if m[4] != "" {
		build, _ = strconv.Atoi(m[4])
	}
	return major, minor, patch, build, true
}

// versionFromTag extracts a normalized numeric version from a tag (e.g.
// "0.2.0-release" → "0.2.0", "0.1.13.333" → "0.1.13.333"), ignoring suffixes.
// Non-version tags (e.g. "latest", "test-*") return !ok.
func versionFromTag(s string) (string, bool) {
	m := versionRegex.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	v := m[1] + "." + m[2] + "." + m[3]
	if m[4] != "" {
		v += "." + m[4]
	}
	return v, true
}

// cachePath returns the path to the version cache file.
func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".huawei", "devbridge", "version_cache.json"), nil
}

// loadCache reads the cached check result if still valid.
func loadCache() *CheckResult {
	path, err := cachePath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var r CheckResult
	if err := json.Unmarshal(data, &r); err != nil {
		return nil
	}
	if time.Since(time.Unix(r.CheckedAt, 0)) > cacheTTL {
		return nil
	}
	return &r
}

// saveCache writes the check result to the cache file.
func saveCache(r *CheckResult) {
	path, err := cachePath()
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	data, _ := json.Marshal(r)
	_ = os.WriteFile(path, data, 0o600)
}

// fetchLatestRelease queries the GitCode API and returns the latest release tag.
func fetchLatestRelease() (string, error) {
	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequest("GET", gitCodeAPI, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gitcode API returned %d", resp.StatusCode)
	}
	var releases []gitcodeRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}
	var bestTag string
	var bestVer string
	for _, r := range releases {
		if !versionRegex.MatchString(r.TagName) {
			continue
		}
		ver, ok := versionFromTag(r.TagName)
		if !ok {
			continue
		}
		if bestVer == "" || isNewerRaw(bestVer, ver) {
			bestVer = ver
			bestTag = r.TagName
		}
	}
	if bestTag == "" {
		return "", fmt.Errorf("no valid release found")
	}
	return bestTag, nil
}

func isNewerRaw(oldV, newV string) bool {
	omaj, omin, opat, obuild, ok1 := parseVersion(oldV)
	nmaj, nmin, npat, nbuild, ok2 := parseVersion(newV)
	if !ok1 || !ok2 {
		return false
	}
	if nmaj != omaj {
		return nmaj > omaj
	}
	if nmin != omin {
		return nmin > omin
	}
	if npat != opat {
		return npat > opat
	}
	return nbuild > obuild
}

// Check performs a version check, using cache when available.
func Check() *CheckResult {
	// Try cache first
	if cached := loadCache(); cached != nil {
		return cached
	}

	// Fetch latest release
	latestTag, err := fetchLatestRelease()
	if err != nil {
		return nil
	}

	// Extract version number from tag
	latestVersion := latestTag
	if v, ok := versionFromTag(latestTag); ok {
		latestVersion = v
	}

	result := &CheckResult{
		LatestVersion: latestVersion,
		LatestTag:     latestTag,
		CheckedAt:     time.Now().Unix(),
	}
	saveCache(result)
	return result
}

// InstallCommand returns the one-liner install commands for the current platform.
func InstallCommand() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("  # GitCode\n  irm %s | iex\n\n  # GitHub\n  irm https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.ps1 | iex",
			gitCodeInstallPs)
	}
	return fmt.Sprintf("  # GitCode\n  curl -fsSL %s | bash\n\n  # GitHub\n  curl -fsSL https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.sh | bash",
		gitCodeInstallSh)
}
