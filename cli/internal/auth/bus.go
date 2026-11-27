package auth

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	// envSessionBusAddress locates the D-Bus session bus. Every D-Bus client,
	// including godbus (used by go-keyring), reads it first.
	envSessionBusAddress = "DBUS_SESSION_BUS_ADDRESS"

	// envRuntimeDir holds the per-user runtime directory whose "bus" entry, when
	// it is a socket, is itself the session bus.
	envRuntimeDir = "XDG_RUNTIME_DIR"

	// autolaunchValue asks D-Bus clients to launch a new bus on demand. It never
	// refers to a persistent bus, so honouring it is exactly what spawns the
	// orphan dbus-daemon processes this probe exists to prevent.
	autolaunchValue = "autolaunch:"

	// sessionBusFilePattern matches the files dbus-launch writes per display:
	// $HOME/.dbus/session-bus/<machine-id>-<display>.
	sessionBusFilePattern = "*-*"
)

// Sources of a positive probe result, reported through the debug log.
const (
	busSourceEnv         = "env"
	busSourceRuntimeDir  = "xdg-runtime-dir"
	busSourceSessionFile = "session-bus-file"
	busSourceNone        = "none"
	busSourceNonLinux    = "unsupported-platform"
)

// busProbe is the cached outcome of the session bus probe.
type busProbe struct {
	available bool
	address   string
	source    string
}

var (
	busProbeOnce  sync.Once
	busProbeState busProbe
)

// sessionBusAvailable reports whether a usable D-Bus session bus is known to
// this process, together with the probe source. The result is cached for the
// lifetime of the process: a long-running command such as "devbridge host"
// must decide once, not once per credential access.
//
// When the address is discovered rather than inherited from the environment, it
// is exported as DBUS_SESSION_BUS_ADDRESS so godbus dials exactly the bus that
// was validated instead of falling back to dbus-launch autolaunch.
func sessionBusAvailable() (bool, string) {
	busProbeOnce.Do(func() {
		busProbeState = probeSessionBus()
		slog.Debug("d-bus session bus probe",
			"available", busProbeState.available,
			"source", busProbeState.source,
		)
	})
	return busProbeState.available, busProbeState.source
}

// resetBusProbe clears the cached probe result. It exists for tests only.
func resetBusProbe() {
	busProbeOnce = sync.Once{}
	busProbeState = busProbe{}
}

// probeSessionBus decides whether a session bus is usable without connecting to
// it. Only read-only checks are performed: dialling would itself reach the
// autolaunch path on a bus-less host.
func probeSessionBus() busProbe {
	// Non-Linux platforms use a native credential store (Keychain, Credential
	// Manager) instead of the Secret Service, so nothing changes for them.
	if runtime.GOOS != "linux" {
		return busProbe{available: true, source: busSourceNonLinux}
	}

	if address := usableBusAddress(os.Getenv(envSessionBusAddress)); address != "" {
		return busProbe{available: true, address: address, source: busSourceEnv}
	}
	if address := busAddressFromRuntimeDir(); address != "" {
		exportBusAddress(address)
		return busProbe{available: true, address: address, source: busSourceRuntimeDir}
	}
	if address := busAddressFromSessionFiles(); address != "" {
		exportBusAddress(address)
		return busProbe{available: true, address: address, source: busSourceSessionFile}
	}
	return busProbe{source: busSourceNone}
}

// usableBusAddress returns the first address in a (possibly semicolon-separated)
// D-Bus address list that refers to an existing bus, or "" when there is none.
func usableBusAddress(raw string) string {
	for _, candidate := range strings.Split(raw, ";") {
		address := strings.TrimSpace(candidate)
		if address == "" || strings.EqualFold(address, autolaunchValue) {
			continue
		}
		return address
	}
	return ""
}

// busAddressFromRuntimeDir returns the address of $XDG_RUNTIME_DIR/bus when that
// entry is a socket.
func busAddressFromRuntimeDir() string {
	dir := strings.TrimSpace(os.Getenv(envRuntimeDir))
	if dir == "" {
		return ""
	}
	socket := filepath.Join(dir, "bus")
	info, err := os.Stat(socket)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return ""
	}
	return "unix:path=" + socket
}

// busAddressFromSessionFiles reads the addresses dbus-launch persists under
// $HOME/.dbus/session-bus, one file per display.
func busAddressFromSessionFiles() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	matches, err := filepath.Glob(filepath.Join(home, ".dbus", "session-bus", sessionBusFilePattern))
	if err != nil {
		return ""
	}
	for _, path := range matches {
		if address := busAddressFromSessionFile(path); address != "" {
			return address
		}
	}
	return ""
}

// busAddressFromSessionFile extracts DBUS_SESSION_BUS_ADDRESS from a single
// dbus-launch session file.
func busAddressFromSessionFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		value, found := strings.CutPrefix(line, envSessionBusAddress+"=")
		if !found {
			continue
		}
		if address := usableBusAddress(strings.Trim(value, `"'`)); address != "" {
			return address
		}
	}
	return ""
}

// exportBusAddress publishes the discovered address to the environment so the
// D-Bus client dials it instead of starting a bus of its own.
func exportBusAddress(address string) {
	if err := os.Setenv(envSessionBusAddress, address); err != nil {
		slog.Debug("failed to export the discovered session bus address", "err", err)
	}
}
