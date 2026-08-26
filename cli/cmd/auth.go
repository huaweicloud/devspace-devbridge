package cmd

import (
	"fmt"
	"log/slog"

	"huawei.com/devbridge/internal/auth"
	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var hcLoginAPIKey string //nolint:gochecknoglobals

var authCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "auth",
	Short: i18n.T(i18n.Msg.Common.AuthCommands),
}

var loginCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "login",
	Short: i18n.T(i18n.Msg.Auth.LoginShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {

		if hcLoginAPIKey == "" {
			if cred := auth.ReadValidAPIKey(); cred != nil {
				if _, err := auth.VerifyAPIKey(cred.APIKey); err == nil {
					fmt.Println(i18n.T(i18n.Msg.Auth.LoginSuccess))
					return nil
				}
			}
		}
		cred, userInfo, err := auth.HCAuth(hcLoginAPIKey)
		if err != nil {
			return err
		}

		if _, err := auth.VerifyAPIKey(cred.APIKey); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		if err := auth.StoreCredential(auth.CredentialName, &cred, userInfo); err != nil {
			return err
		}
		fmt.Println(i18n.T(i18n.Msg.Auth.LoginSuccess))
		return nil
	}),
}

var logoutCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "logout",
	Short: i18n.T(i18n.Msg.Auth.LogoutShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		if err := auth.DeleteCredential(auth.CredentialName); err != nil {
			slog.Warn("failed to delete credential", "err", err)
		}
		if err := config.DeleteDefaultTunnel(); err != nil {
			slog.Warn("failed to delete default tunnel", "err", err)
		}
		fmt.Println(i18n.T(i18n.Msg.Auth.LogoutSuccess))
		return nil
	}),
}

var statusCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "status",
	Short: i18n.T(i18n.Msg.Auth.StatusShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		cred := auth.ReadValidAPIKey()
		if cred == nil {
			fmt.Println(i18n.T(i18n.Msg.Auth.NotLoggedIn))
			return nil
		}

		if _, err := auth.VerifyAPIKey(cred.APIKey); err != nil {
			fmt.Printf("%s: %v\n", i18n.T(i18n.Msg.Auth.NotLoggedIn), err)
			return nil
		}
		fmt.Println(i18n.T(i18n.Msg.Auth.LoggedInHuaweiCloud))
		_, userInfo, err := auth.LoadCredential(auth.CredentialName)
		if err == nil && userInfo != nil && userInfo.UserName != "" {
			fmt.Printf("%s:  %s\n", i18n.T(i18n.Msg.Auth.UserName), userInfo.UserName)
		}
		return nil
	}),
}

func init() { //nolint:gochecknoinits
	loginCmd.Flags().StringVar(&hcLoginAPIKey, "api-key", "", i18n.T(i18n.Msg.Common.FlagAPIKey))

	authCmd.AddCommand(loginCmd, logoutCmd, statusCmd)
	RootCmd.AddCommand(authCmd)
}
