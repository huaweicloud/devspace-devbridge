package auth

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

// isolateBusEnv puts the process into a state where no session bus can be
// discovered, and clears the cached probe result so each test observes a fresh
// decision. HOME points at a per-test temporary directory.
func isolateBusEnv(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(envSessionBusAddress, "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	resetBusProbe()
	t.Cleanup(func() {
		// The probe may inject DBUS_SESSION_BUS_ADDRESS itself; undo that before
		// t.Setenv restores the values captured at test start.
		_ = os.Unsetenv(envSessionBusAddress)
		resetBusProbe()
	})
	return home
}

func TestSessionBusAvailableFromEnv(t *testing.T) {
	isolateBusEnv(t)
	t.Setenv(envSessionBusAddress, "unix:path=/run/user/1000/bus")

	available, source := sessionBusAvailable()
	if !available {
		t.Fatal("expected a session bus to be reported available when the address is set")
	}
	if source != busSourceEnv {
		t.Fatalf("unexpected probe source: got %q, want %q", source, busSourceEnv)
	}
}

func TestSessionBusUnavailableWhenAddressIsAutolaunch(t *testing.T) {
	isolateBusEnv(t)
	t.Setenv(envSessionBusAddress, autolaunchValue)

	available, source := sessionBusAvailable()
	if available {
		t.Fatalf("an %q address must not be treated as a usable bus", autolaunchValue)
	}
	if source != busSourceNone {
		t.Fatalf("unexpected probe source: got %q, want %q", source, busSourceNone)
	}
}

func TestSessionBusUnavailableWhenNothingIsFound(t *testing.T) {
	isolateBusEnv(t)

	available, source := sessionBusAvailable()
	if available {
		t.Fatal("expected no session bus in a clean headless environment")
	}
	if source != busSourceNone {
		t.Fatalf("unexpected probe source: got %q, want %q", source, busSourceNone)
	}
}

func TestSessionBusAvailableFromRuntimeDirSocket(t *testing.T) {
	isolateBusEnv(t)
	dir := t.TempDir()
	socket := filepath.Join(dir, "bus")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("failed to create a unix socket fixture: %v", err)
	}
	defer func() { _ = listener.Close() }()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	available, source := sessionBusAvailable()
	if !available {
		t.Fatal("expected $XDG_RUNTIME_DIR/bus socket to be reported available")
	}
	if source != busSourceRuntimeDir {
		t.Fatalf("unexpected probe source: got %q, want %q", source, busSourceRuntimeDir)
	}
	if got, want := os.Getenv(envSessionBusAddress), "unix:path="+socket; got != want {
		t.Fatalf("probe must inject the discovered address: got %q, want %q", got, want)
	}
}

func TestSessionBusUnavailableWhenRuntimeDirEntryIsNotSocket(t *testing.T) {
	isolateBusEnv(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bus"), []byte("not a socket"), 0o600); err != nil {
		t.Fatalf("failed to create the fixture: %v", err)
	}
	t.Setenv("XDG_RUNTIME_DIR", dir)

	available, _ := sessionBusAvailable()
	if available {
		t.Fatal("a regular file at $XDG_RUNTIME_DIR/bus is not a session bus socket")
	}
}

func TestSessionBusAvailableFromSessionBusFile(t *testing.T) {
	home := isolateBusEnv(t)
	dir := filepath.Join(home, ".dbus", "session-bus")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("failed to create the fixture directory: %v", err)
	}
	content := "# This file allows processes on the machine to find the session bus\n" +
		"DBUS_SESSION_BUS_ADDRESS=unix:abstract=/tmp/dbus-abc123,guid=deadbeef\n" +
		"DBUS_SESSION_BUS_PID=4242\n"
	if err := os.WriteFile(filepath.Join(dir, "machine-id-0"), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to create the session bus file: %v", err)
	}

	available, source := sessionBusAvailable()
	if !available {
		t.Fatal("expected a session bus entry in $HOME/.dbus/session-bus to be discovered")
	}
	if source != busSourceSessionFile {
		t.Fatalf("unexpected probe source: got %q, want %q", source, busSourceSessionFile)
	}
	want := "unix:abstract=/tmp/dbus-abc123,guid=deadbeef"
	if got := os.Getenv(envSessionBusAddress); got != want {
		t.Fatalf("probe must inject the discovered address: got %q, want %q", got, want)
	}
}

func TestSessionBusFileWithAutolaunchAddressIsIgnored(t *testing.T) {
	home := isolateBusEnv(t)
	dir := filepath.Join(home, ".dbus", "session-bus")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("failed to create the fixture directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "machine-id-0"), []byte("DBUS_SESSION_BUS_ADDRESS=autolaunch:\n"), 0o600); err != nil {
		t.Fatalf("failed to create the session bus file: %v", err)
	}

	available, _ := sessionBusAvailable()
	if available {
		t.Fatal("an autolaunch address recorded in a session bus file is not a usable bus")
	}
}

func TestSessionBusProbeIsCachedForProcessLifetime(t *testing.T) {
	isolateBusEnv(t)

	if available, _ := sessionBusAvailable(); available {
		t.Fatal("expected no session bus in a clean headless environment")
	}

	// A later command in the same process must reuse the cached decision instead
	// of probing again: no bus was available when the process started.
	t.Setenv(envSessionBusAddress, "unix:path=/run/user/1000/bus")
	available, source := sessionBusAvailable()
	if available || source != busSourceNone {
		t.Fatalf("probe result must be cached per process: got available=%v source=%q", available, source)
	}
}
