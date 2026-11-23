package cmd

import "testing"

func TestFormatRemainingPluralization(t *testing.T) {
	cases := []struct {
		name             string
		v                float64
		singular, plural string
		want             string
	}{
		{"one hour singular", 1, "hour", "hours", "1 hour"},
		{"one day singular", 1, "day", "days", "1 day"},
		{"two hours plural", 2, "hour", "hours", "2 hours"},
		{"half day plural", 0.5, "day", "days", "0.5 days"},
		{"1.1 hours plural", 1.1, "hour", "hours", "1.1 hours"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatRemaining(tc.v, tc.singular, tc.plural)
			if got != tc.want {
				t.Fatalf("formatRemaining(%v, %q, %q) = %q, want %q", tc.v, tc.singular, tc.plural, got, tc.want)
			}
		})
	}
}

func TestFormatDurationUnitConsistency(t *testing.T) {
	cases := []struct {
		name  string
		hours float64
		want  string
	}{
		// create (static integer -e) and show/list (remaining ~24h) must agree.
		{"exactly 24h is 1 day", 24, "1 day"},
		{"23.99h remaining rounds to 1 day", 23.99, "1 day"},
		{"under 24h stays hours", 23.9, "23.9 hours"},
		{"1h singular", 1, "1 hour"},
		{"48h is 2 days", 48, "2 days"},
		{"36h is 1.5 days", 36, "1.5 days"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatDuration(tc.hours)
			if got != tc.want {
				t.Fatalf("formatDuration(%v) = %q, want %q", tc.hours, got, tc.want)
			}
		})
	}
}
