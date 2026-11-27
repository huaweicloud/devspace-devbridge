//go:build !linux

package auth

// keyringBackendUsable always reports true off Linux: those platforms back the
// keyring with the OS keychain / credential manager directly (no D-Bus), so
// there is nothing to probe and no reason to downgrade to the config file.
func keyringBackendUsable() bool { return true }
