package cmd

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	devbridge "github.com/huaweicloud/devspace-devbridge/go-sdk"
	"github.com/spf13/cobra"

	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"
)

// testHome isolates HOME (and the Windows equivalent used by os.UserHomeDir)
// so the CLI config file (~/.huawei/devbridge/config.yaml) never touches the
// real user config. Returns the temp home dir.
func testHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

// storeDefaultTunnel writes default-tunnel-id into the isolated HOME config.
func storeDefaultTunnel(t *testing.T, id string) {
	t.Helper()
	if err := config.StoreDefaultTunnel(id); err != nil {
		t.Fatalf("store default tunnel: %v", err)
	}
}

// loadDefaultTunnelOpt returns the default tunnel ID, or "" when unset.
func loadDefaultTunnelOpt(t *testing.T) string {
	t.Helper()
	id, err := config.LoadDefaultTunnel()
	if err != nil {
		return ""
	}
	return id
}

// newDeleteAllTestServer spins up an HTTP test server that counts DELETE
// requests to the tunnels collection and answers each call with the given
// status (default 200).
func newDeleteAllTestServer(t *testing.T, statuses ...int) (*httptest.Server, *[]int) {
	t.Helper()
	calls := []int{}
	mux := http.NewServeMux()
	mux.HandleFunc("/open-api-inner/v1/relay-controller/tunnels", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, 1)
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method %q, want DELETE", r.Method)
		}
		status := http.StatusOK
		if len(statuses) > 0 {
			status = statuses[len(calls)-1]
		}
		w.WriteHeader(status)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &calls
}

// redirectConfigServer points the CLI API server domain at the test server.
func redirectConfigServer(t *testing.T, srv *httptest.Server) {
	t.Helper()
	old := config.DefaultServerDomain
	config.DefaultServerDomain = srv.URL
	t.Cleanup(func() { config.DefaultServerDomain = old })
}

// stubSDKClient replaces the SDK client factory with one returning a client
// bound to the redirected test server (no credentials needed).
func stubSDKClient(t *testing.T) {
	t.Helper()
	old := newSDKClient
	newSDKClient = func() *devbridge.Devbridge { return config.NewClient("") }
	t.Cleanup(func() { newSDKClient = old })
}

// runDeleteAll executes deleteAllCmd.RunE under controlled conditions:
// yes toggles the --yes flag, tty controls TTY detection, and answer (when
// non-empty) is the text served to the confirmation reader. Returns captured
// stdout and the RunE error.
func runDeleteAll(t *testing.T, yes, tty bool, answer string) (string, error) {
	t.Helper()

	if err := deleteAllCmd.Flags().Set("yes", boolStr(yes)); err != nil {
		t.Fatalf("set --yes flag: %v", err)
	}
	t.Cleanup(func() { _ = deleteAllCmd.Flags().Set("yes", "false") })

	oldIsTTY := confirmIsTTY
	confirmIsTTY = func(f *os.File) bool { return tty }
	t.Cleanup(func() { confirmIsTTY = oldIsTTY })

	oldReader := confirmReader
	confirmReader = nil
	if answer != "" {
		confirmReader = bufio.NewReader(strings.NewReader(answer))
	}
	t.Cleanup(func() { confirmReader = oldReader })

	var out bytes.Buffer
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	os.Stdout = w

	err = func() error {
		runErr := deleteAllCmd.RunE(deleteAllCmd, nil)
		os.Stdout = oldStdout
		w.Close()
		return runErr
	}()
	_, _ = out.ReadFrom(r)
	r.Close()
	return out.String(), err
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// --- deleteAllCmd regression tests (Issue #45) ---

func TestDeleteAllClearsDefaultTunnelOnSuccess(t *testing.T) {
	testHome(t)

	srv, calls := newDeleteAllTestServer(t)
	redirectConfigServer(t, srv)
	stubSDKClient(t)

	storeDefaultTunnel(t, "t-123")

	out, err := runDeleteAll(t, true, false, "")
	if err != nil {
		t.Fatalf("delete-all --yes: %v", err)
	}
	if got := len(*calls); got != 1 {
		t.Fatalf("DELETE /tunnels called %d times, want 1", got)
	}
	if id := loadDefaultTunnelOpt(t); id != "" {
		t.Fatalf("default tunnel = %q after delete-all, want cleared", id)
	}
	if !strings.Contains(out, i18n.T(i18n.Msg.Tunnel.DefaultTunnelCleared)) {
		t.Errorf("stdout missing cleared hint: %q", out)
	}
	if !strings.Contains(out, i18n.T(i18n.Msg.Tunnel.TunnelDeletedAll)) {
		t.Errorf("stdout missing deleted-all hint: %q", out)
	}
}

func TestDeleteAllNonTTYWithoutYesRefuses(t *testing.T) {
	testHome(t)

	srv, calls := newDeleteAllTestServer(t)
	redirectConfigServer(t, srv)
	stubSDKClient(t)

	storeDefaultTunnel(t, "t-123")

	out, err := runDeleteAll(t, false, false, "")
	if err == nil {
		t.Fatal("delete-all without --yes in non-TTY should fail")
	}
	if !strings.Contains(err.Error(), i18n.T(i18n.Msg.Tunnel.ConfirmNonInteractive)) {
		t.Errorf("error should mention --yes / non-interactive, got: %v", err)
	}
	if got := len(*calls); got != 0 {
		t.Fatalf("DELETE /tunnels called %d times, want 0 (refused before API)", got)
	}
	if id := loadDefaultTunnelOpt(t); id != "t-123" {
		t.Fatalf("default tunnel = %q, want preserved t-123", id)
	}
	if strings.Contains(out, i18n.T(i18n.Msg.Tunnel.TunnelDeletedAll)) {
		t.Errorf("stdout should not claim deletion: %q", out)
	}
}

func TestDeleteAllKeepsDefaultTunnelOnFailure(t *testing.T) {
	testHome(t)

	srv, calls := newDeleteAllTestServer(t, http.StatusForbidden)
	redirectConfigServer(t, srv)
	stubSDKClient(t)

	storeDefaultTunnel(t, "t-123")

	_, err := runDeleteAll(t, true, false, "")
	if err == nil {
		t.Fatal("delete-all --yes should propagate API failure")
	}
	if got := len(*calls); got != 1 {
		t.Fatalf("DELETE /tunnels called %d times, want 1", got)
	}
	if id := loadDefaultTunnelOpt(t); id != "t-123" {
		t.Fatalf("default tunnel = %q after failed delete-all, want kept t-123", id)
	}
}

func TestDeleteAllTTYConfirmAccepted(t *testing.T) {
	testHome(t)

	srv, calls := newDeleteAllTestServer(t)
	redirectConfigServer(t, srv)
	stubSDKClient(t)

	storeDefaultTunnel(t, "t-123")

	out, err := runDeleteAll(t, false, true, "y\n")
	if err != nil {
		t.Fatalf("delete-all with y answer: %v", err)
	}
	if got := len(*calls); got != 1 {
		t.Fatalf("DELETE /tunnels called %d times, want 1", got)
	}
	if id := loadDefaultTunnelOpt(t); id != "" {
		t.Fatalf("default tunnel = %q, want cleared", id)
	}
	if !strings.Contains(out, i18n.T(i18n.Msg.Tunnel.ConfirmDeleteAll)) {
		t.Errorf("stdout missing confirmation prompt: %q", out)
	}
}

func TestDeleteAllTTYConfirmDeclined(t *testing.T) {
	testHome(t)

	srv, calls := newDeleteAllTestServer(t)
	redirectConfigServer(t, srv)
	stubSDKClient(t)

	storeDefaultTunnel(t, "t-123")

	_, err := runDeleteAll(t, false, true, "n\n")
	if err == nil {
		t.Fatal("delete-all with n answer should abort")
	}
	if !strings.Contains(err.Error(), i18n.T(i18n.Msg.Tunnel.DeleteAborted)) {
		t.Errorf("error should say aborted, got: %v", err)
	}
	if got := len(*calls); got != 0 {
		t.Fatalf("DELETE /tunnels called %d times, want 0 (aborted before API)", got)
	}
	if id := loadDefaultTunnelOpt(t); id != "t-123" {
		t.Fatalf("default tunnel = %q, want preserved t-123", id)
	}
}

func TestDeleteAllClearsConfigFileOnDisk(t *testing.T) {
	home := testHome(t)

	srv, _ := newDeleteAllTestServer(t)
	redirectConfigServer(t, srv)
	stubSDKClient(t)

	storeDefaultTunnel(t, "t-abc")
	cfgPath := filepath.Join(home, ".huawei", "devbridge", "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config file should exist: %v", err)
	}

	if _, err := runDeleteAll(t, true, false, ""); err != nil {
		t.Fatalf("delete-all --yes: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(data), "default-tunnel-id") {
		t.Fatalf("config still contains default-tunnel-id: %s", data)
	}
}

// --- confirmOrAbort unit tests ---

// newConfirmTestCmd builds a bare command carrying the --yes flag.
func newConfirmTestCmd() *cobra.Command {
	c := &cobra.Command{Use: "confirm-test"}
	c.Flags().BoolP("yes", "y", false, "")
	return c
}

func withTTY(t *testing.T, tty bool) {
	t.Helper()
	old := confirmIsTTY
	confirmIsTTY = func(f *os.File) bool { return tty }
	t.Cleanup(func() { confirmIsTTY = old })
}

func withAnswer(t *testing.T, answer string) {
	t.Helper()
	old := confirmReader
	confirmReader = bufio.NewReader(strings.NewReader(answer))
	t.Cleanup(func() { confirmReader = old })
}

func TestConfirmOrAbortYesFlagSkipsPrompt(t *testing.T) {
	withTTY(t, false)

	cmd := newConfirmTestCmd()
	_ = cmd.Flags().Set("yes", "true")
	if err := confirmOrAbort(cmd, "yes", "prompt"); err != nil {
		t.Fatalf("--yes should skip confirmation: %v", err)
	}
}

func TestConfirmOrAbortNonTTYErrors(t *testing.T) {
	withTTY(t, false)

	err := confirmOrAbort(newConfirmTestCmd(), "yes", "prompt")
	if err == nil {
		t.Fatal("non-TTY without --yes should error")
	}
	if !strings.Contains(err.Error(), i18n.T(i18n.Msg.Tunnel.ConfirmNonInteractive)) {
		t.Errorf("error should mention --yes, got: %v", err)
	}
}

func TestConfirmOrAbortTTYAcceptsY(t *testing.T) {
	withTTY(t, true)
	withAnswer(t, "y\n")

	if err := confirmOrAbort(newConfirmTestCmd(), "yes", "prompt"); err != nil {
		t.Fatalf("y should be accepted: %v", err)
	}
}

func TestConfirmOrAbortTTYDeclines(t *testing.T) {
	withTTY(t, true)

	for _, answer := range []string{"n\n", "\n", "no\n", "what\n", "YES extra\n"} {
		withAnswer(t, answer)
		err := confirmOrAbort(newConfirmTestCmd(), "yes", "prompt")
		if err == nil {
			t.Fatalf("answer %q should abort", answer)
		}
		if !strings.Contains(err.Error(), i18n.T(i18n.Msg.Tunnel.DeleteAborted)) {
			t.Errorf("answer %q: error should say aborted, got %v", answer, err)
		}
	}
}
