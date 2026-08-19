package cmd

import (
	"fmt"
	"log/slog"

	"huawei.com/devbridge/internal/auth"
	"huawei.com/devbridge/internal/i18n"

	"github.com/spf13/cobra"
)

var (
	hcLoginAPIKey      string
	hcLoginHuaweiCloud bool
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: i18n.T(i18n.Msg.Common.AuthCommands),
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: i18n.T(i18n.Msg.Auth.LoginShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		cred, userInfo, err := auth.HCAuth(hcLoginAPIKey, hcLoginHuaweiCloud)
		if err != nil {
			return err
		}
		// 校验 API Key 有效性，无效则登录失败
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

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: i18n.T(i18n.Msg.Auth.LogoutShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		if err := auth.DeleteCredential(auth.CredentialName); err != nil {
			slog.Warn("failed to delete credential", "err", err)
		}
		if err := auth.DeleteDefaultTunnel(); err != nil {
			slog.Debug("no default tunnel to delete", "err", err)
		}
		fmt.Println(i18n.T(i18n.Msg.Auth.LogoutSuccess))
		return nil
	}),
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: i18n.T(i18n.Msg.Auth.StatusShort),
	Args:  cobra.NoArgs,
	RunE: runError(func(cmd *cobra.Command, args []string) error {
		cred := auth.ReadValidAPIKey()
		if cred == nil {
			fmt.Println(i18n.T(i18n.Msg.Auth.NotLoggedIn))
			return nil
		}
		// 远程校验 API Key 有效性
		if _, err := auth.VerifyAPIKey(cred.APIKey); err != nil {
			fmt.Printf("%s: %v\n", i18n.T(i18n.Msg.Auth.NotLoggedIn), err)
			return nil
		}
		if cred.LoginType == "huaweicloud" {
			fmt.Println(i18n.T(i18n.Msg.Auth.LoggedInHuaweiCloud))
		} else {
			fmt.Println(i18n.T(i18n.Msg.Auth.LoggedInUnknown))
		}
		_, userInfo, err := auth.LoadCredential(auth.CredentialName)
		if err == nil && userInfo != nil && userInfo.UserName != "" {
			fmt.Printf("%s:  %s\n", i18n.T(i18n.Msg.Auth.UserName), userInfo.UserName)
		}
		return nil
	}),
}

func init() {
	loginCmd.Flags().StringVar(&hcLoginAPIKey, "api-key", "", i18n.T(i18n.Msg.Common.FlagAPIKey))
	loginCmd.Flags().BoolVar(&hcLoginHuaweiCloud, "huaweicloud", true, i18n.T(i18n.Msg.Common.FlagHuaweiCloud))

	authCmd.AddCommand(loginCmd, logoutCmd, statusCmd)
	RootCmd.AddCommand(authCmd)
}
