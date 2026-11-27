//go:build linux

package auth

import (
	"sync"

	dbus "github.com/godbus/dbus/v5"
)

var (
	keyringBusOnce sync.Once
	keyringBusOK   bool
)

// keyringBackendUsable reports whether a D-Bus session bus is reachable
// WITHOUT autolaunching one. On Linux the secret-service keyring rides the
// session bus; in a headless/service environment (no DBUS_SESSION_BUS_ADDRESS,
// no /run/user/<uid>/bus, no /run/user/<uid>/dbus-session) this probe returns
// false so credentials fall back to the config file instead of fork-bombing
// dbus-launch daemons.
//
// It deliberately reuses godbus's SessionBusPrivateNoAutoStartup, which walks
// the exact same address-discovery order the library uses for real connections
// (env → /run/user/<uid>/bus → /run/user/<uid>/dbus-session) and returns an
// error instead of exec'ing dbus-launch when none exist. Re-implementing that
// discovery here would diverge and risk misclassifying a healthy desktop
// session as headless (downgrading its keyring to a plaintext file).
func keyringBackendUsable() bool {
	keyringBusOnce.Do(func() {
		conn, err := dbus.SessionBusPrivateNoAutoStartup()
		if err != nil {
			return
		}
		_ = conn.Close()
		keyringBusOK = true
	})
	return keyringBusOK
}
