package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveAllowAnon(t *testing.T) {
	newCmd := func(args ...string) *cobra.Command {
		c := &cobra.Command{Use: "test"}
		c.Flags().Bool("allow-anonymous", false, "")
		c.Flags().Bool("deny-anonymous", false, "")
		_ = c.ParseFlags(args)
		return c
	}

	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name    string
		args    []string
		want    *bool
		wantErr bool
	}{
		{"unset", nil, nil, false},
		{"allow", []string{"--allow-anonymous"}, boolPtr(true), false},
		{"allow-false", []string{"--allow-anonymous=false"}, boolPtr(false), false},
		{"deny", []string{"--deny-anonymous"}, boolPtr(false), false},
		{"deny-false", []string{"--deny-anonymous=false"}, boolPtr(true), false},
		{"both", []string{"--allow-anonymous", "--deny-anonymous"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveAllowAnon(newCmd(tt.args...))
			if tt.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if (got == nil) != (tt.want == nil) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			if got != nil && tt.want != nil && *got != *tt.want {
				t.Fatalf("got %v, want %v", *got, *tt.want)
			}
		})
	}
}
