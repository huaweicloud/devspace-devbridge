package i18n

import (
	"strings"
	"testing"
)

// TestIssue50MessagesNonEmpty ensures the port-change hint and host-status
// labels shipped for issue #50 exist in both languages.
func TestIssue50MessagesNonEmpty(t *testing.T) {
	msgs := []Message{
		Msg.Port.PortChangeHint,
		Msg.Tunnel.HostOnlineStatus,
		Msg.Tunnel.StatusOnline,
		Msg.Tunnel.StatusOffline,
	}
	for i, m := range msgs {
		if m.ZH == "" || m.EN == "" {
			t.Fatalf("message %d must have both ZH and EN translations, got %+v", i, m)
		}
	}
}

// TestPortChangeHintShowsRestartCommand guards the fallback requirement: a
// successful port change must never be reported without telling the user how a
// static-mode Host picks it up (an explicit restart command).
func TestPortChangeHintShowsRestartCommand(t *testing.T) {
	cases := []struct {
		lang string
		body string
	}{
		{"zh", Msg.Port.PortChangeHint.ZH},
		{"en", Msg.Port.PortChangeHint.EN},
	}
	for _, tc := range cases {
		if !strings.Contains(tc.body, "devbridge host %s -p") {
			t.Fatalf("%s hint must include the restart command, got %q", tc.lang, tc.body)
		}
		if !strings.Contains(tc.body, "%s") {
			t.Fatalf("%s hint must interpolate the tunnel ID", tc.lang)
		}
	}
}
