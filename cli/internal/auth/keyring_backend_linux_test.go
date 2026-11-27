//go:build linux

package auth

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"huawei.com/devbridge/internal/config"
)

// keyringHelperEnv marks the re-exec'd child process that runs the behaviour
// assertions in a pristine memory state (fresh process = fresh sync.Once for
// the keyringBusOnce probe).
const keyringHelperEnv = "AUTH_KEYRING_HELPER_PROCESS"

// TestKeyringBackendHeadless proves the dbus-daemon leak fix end-to-end and,
// critically, does so WITHOUT ever touching a real session keyring.
//
// The child process is started with a HOME pointing at a throwaway directory
// and DBUS_SESSION_BUS_ADDRESS pointing at a nonexistent unix socket. That pair
// deterministically reproduces a headless environment on every host:
//
//   - The config file lives under the throwaway HOME, so StoreCredential's
//     file-fallback path never writes the caller's real ~/.huawei config.
//   - The nonexistent bus address makes godbus's Dial fail, so
//     keyringBackendUsable() returns false without consulting /run/user/<uid>/bus
//     or autolaunching dbus-launch, and StoreCredential never reaches keyring.Set
//     (which would otherwise write the host's real D-Bus secret service).
//
// Three invariants are asserted inside the child:
//
//  1. keyringBackendUsable() returns false.
//  2. StoreCredential/LoadCredential round-trip through the config file.
//  3. No dbus-daemon process is forked (counted via /proc before the probe and
//     after the credential round-trip).
//
// The first version of this test set DBUS_SESSION_BUS_ADDRESS to an empty
// string, which still let godbus fall through to tryDiscoverDbusSessionBusAddress
// and find /run/user/<uid>/bus on a desktop — the probe then returned true and
// StoreCredential(..., "ci-test-api-key") wrote a garbage entry into the host's
// real gnome-keyring. The nonexistent-address approach below closes that hole.
func TestKeyringBackendHeadless(t *testing.T) {
	if os.Getenv(keyringHelperEnv) == "1" {
		runHeadlessAssertions(t)
		return
	}

	home := t.TempDir()
	// A socket that cannot exist: the trailing directory is a file-free temp
	// path, so Dial fails fast and no autolaunch is possible.
	nonexistentBus := "unix:path=" + filepath.Join(home, "no-such-dir", "bus")

	cmd := exec.Command(os.Args[0], "-test.run=^TestKeyringBackendHeadless$")
	cmd.Env = []string{
		keyringHelperEnv + "=1",
		"HOME=" + home,
		"PATH=" + os.Getenv("PATH"),
		"DBUS_SESSION_BUS_ADDRESS=" + nonexistentBus,
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper 子进程退出异常 %v:\n%s", err, out)
	}
	s := string(out)
	if strings.Contains(s, "panic") || strings.Contains(s, "--- FAIL") {
		t.Fatalf("helper 断言失败:\n%s", s)
	}
	if !strings.Contains(s, "USABLE=false") {
		t.Logf("helper 未报告 USABLE=false，完整输出:\n%s", s)
	}
}

// runHeadlessAssertions executes the real behaviour checks in the re-exec'd
// child process and reports its observations on stdout for the parent to parse.
func runHeadlessAssertions(t *testing.T) {
	before := countDbusDaemons()

	usable := keyringBackendUsable()

	const apiKey = "ci-test-api-key"
	storeErr := StoreCredential(CredentialName, &Credential{APIKey: apiKey}, nil)

	cfg, _ := config.Load()
	_, fileStored := cfg[credentialsKey]

	got, _, loadErr := LoadCredential(CredentialName)

	after := countDbusDaemons()

	// Structural observations for the parent process.
	fmt.Printf("USABLE=%v\n", usable)
	fmt.Printf("DBUS_BEFORE=%d\n", before)
	fmt.Printf("DBUS_AFTER=%d\n", after)
	fmt.Printf("STORE_ERR=%v\n", storeErr)
	fmt.Printf("LOAD_ERR=%v\n", loadErr)
	fmt.Printf("FILE_STORED=%v\n", fileStored)
	fmt.Printf("LOAD_OK=%v\n", got != nil && got.APIKey == apiKey)

	// Invariant 1: the nonexistent-address probe must report "no bus".
	if usable {
		t.Errorf("无头环境 keyringBackendUsable 应为 false，得到 true（探活可能 autolaunch）")
	}

	// Invariant 2: the credential goes through the config file, untouched by D-Bus.
	if storeErr != nil {
		t.Errorf("无头 StoreCredential 应成功写文件，got %v", storeErr)
	}
	if !fileStored {
		t.Errorf("无头 StoreCredential 应将 credentials 写入 config，实际未写")
	}
	if loadErr != nil || got == nil || got.APIKey != apiKey {
		t.Errorf("无头 LoadCredential 应从文件读回 apiKey=%q，got %+v err=%v", apiKey, got, loadErr)
	}

	// Invariant 3 — the regression guard. Counted *before* the probe, so an
	// autolaunching probe (a regression) shows up as an increase.
	if after != before {
		t.Errorf("dbus-daemon 泄漏：%d -> %d（探活或凭据操作触发了 autolaunch）", before, after)
	}
}

// countDbusDaemons counts processes whose /proc/<pid>/comm is exactly
// "dbus-daemon". Returns -1 if /proc cannot be read (assertions then degrade
// harmlessly because both sides read -1).
func countDbusDaemons() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return -1
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "" || name[0] < '0' || name[0] > '9' {
			continue
		}
		b, err := os.ReadFile(filepath.Join("/proc", name, "comm"))
		if err == nil && string(bytes.TrimSpace(b)) == "dbus-daemon" {
			n++
		}
	}
	return n
}
