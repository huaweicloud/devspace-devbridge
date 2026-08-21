package api

import (
	"errors"
	"fmt"
	"regexp"

	"huawei.com/devbridge/internal/auth"
	"huawei.com/devbridge/internal/i18n"
)

var (
	tunnelIdRegexp   = regexp.MustCompile(`^[a-z2-7]{8}$`)
	tunnelNameRegexp = regexp.MustCompile(`^[\x{4e00}-\x{9fa5}A-Za-z0-9]([\x{4e00}-\x{9fa5}A-Za-z0-9-]{0,62}[\x{4e00}-\x{9fa5}A-Za-z0-9])?$`)
	tunnelDescRegexp = regexp.MustCompile(`^[\x{4e00}-\x{9fa5}A-Za-z0-9]{0,64}$`)
)

const (
	listTunnelsPath      = "/tunnels"
	createTunnelPath     = "/tunnels"
	showTunnelPath       = "/tunnels/%s"
	updateTunnelPath     = "/tunnels/%s"
	deleteTunnelPath     = "/tunnels/%s"
	deleteAllTunnelsPath = "/tunnels"
	tunnelTokenPath      = "/tunnels/%s/token"
)

type createTunnelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Expiration  int    `json:"expiration,omitempty"`
	ClusterId   string `json:"ClusterId"`
}

type CreateTunnelResult struct {
	TunnelId        string `json:"tunnelId"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	ExpirationHours int    `json:"expirationHours"`
}

type ListTunnelResult struct {
	Name             string `json:"name"`
	TunnelId         string `json:"tunnelId"`
	TunnelExpiration uint32 `json:"tunnelExpiration"`
	Description      string `json:"description"`
	PortCount        int    `json:"portCount"`
}

type TunnelStatus struct {
	ClientConnectionCount int   `json:"clientConnectionCount"`
	HostConnectionCount   int   `json:"hostConnectionCount"`
	TotalUploadBytes      int64 `json:"totalUploadBytes"`
	TotalDownloadBytes    int64 `json:"totalDownloadBytes"`
}

type ShowTunnelResult struct {
	Name             string        `json:"name"`
	TunnelId         string        `json:"tunnelId"`
	TunnelExpiration uint32        `json:"tunnelExpiration"`
	Description      string        `json:"description"`
	Status           *TunnelStatus `json:"status,omitempty"`
}

type updateTunnelRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Expiration  *int    `json:"expiration,omitempty"`
}

type TunnelTokenResult struct {
	TunnelId string `json:"tunnelId"`
	Scope    string `json:"scope"`
	Token    string `json:"token"`
}

func ValidateTunnelId(tunnelId string) error {
	if !tunnelIdRegexp.MatchString(tunnelId) {
		return fmt.Errorf("invalid tunnel id: %q (only lowercase letters and digits 2-7 allowed, length must be 8)", tunnelId)
	}
	return nil
}

func ResolveTunnelId(tunnelId string) (string, error) {
	if tunnelId == "" {
		id, err := auth.LoadDefaultTunnel()
		if err != nil {
			return "", err
		}
		return id, nil
	}
	return tunnelId, nil
}

func ListTunnels() ([]ListTunnelResult, error) {
	var result []ListTunnelResult
	if err := get(listTunnelsPath, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func CreateTunnel(name, description string, expiration *int) (*CreateTunnelResult, error) {
	if !tunnelNameRegexp.MatchString(name) {
		return nil, errors.New(i18n.T(i18n.Msg.Tunnel.TunnelNameInvalid))
	}
	if !tunnelDescRegexp.MatchString(description) {
		return nil, errors.New(i18n.T(i18n.Msg.Tunnel.TunnelDescInvalid))
	}
	req := createTunnelRequest{Name: name, Description: description, ClusterId: "cn-north-4-bridge"}
	if expiration != nil {
		if *expiration < 1 || *expiration > 720 {
			return nil, errors.New(i18n.T(i18n.Msg.Tunnel.TunnelExpInvalid))
		}
		req.Expiration = *expiration
	}
	var result CreateTunnelResult
	if err := post(createTunnelPath, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func ShowTunnel(tunnelId string) (*ShowTunnelResult, error) {
	if err := ValidateTunnelId(tunnelId); err != nil {
		return nil, err
	}
	var result ShowTunnelResult
	if err := get(fmt.Sprintf(showTunnelPath, tunnelId), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func UpdateTunnel(tunnelId string, name, description *string, expiration *int) error {
	if err := ValidateTunnelId(tunnelId); err != nil {
		return err
	}
	req := updateTunnelRequest{}
	if name != nil {
		if !tunnelNameRegexp.MatchString(*name) {
			return errors.New(i18n.T(i18n.Msg.Tunnel.TunnelNameInvalid))
		}
		req.Name = name
	}
	if description != nil {
		if !tunnelDescRegexp.MatchString(*description) {
			return errors.New(i18n.T(i18n.Msg.Tunnel.TunnelDescInvalid))
		}
		req.Description = description
	}
	if expiration != nil {
		if *expiration < 1 || *expiration > 720 {
			return errors.New(i18n.T(i18n.Msg.Tunnel.TunnelExpInvalid))
		}
		req.Expiration = expiration
	}
	return put(fmt.Sprintf(updateTunnelPath, tunnelId), req, nil)
}

func DeleteTunnel(tunnelId string) error {
	if err := ValidateTunnelId(tunnelId); err != nil {
		return err
	}
	return deleteReq(fmt.Sprintf(deleteTunnelPath, tunnelId), nil)
}

func DeleteAllTunnels() error {
	return deleteReq(deleteAllTunnelsPath, nil)
}

func TunnelToken(tunnelId, scope string) (*TunnelTokenResult, error) {
	if err := ValidateTunnelId(tunnelId); err != nil {
		return nil, err
	}
	if scope != "host" && scope != "connect" {
		return nil, fmt.Errorf("invalid token scope: %q (scope must be one of host, connect)", scope)
	}
	var result TunnelTokenResult
	if err := post(fmt.Sprintf(tunnelTokenPath, tunnelId)+"?scope="+scope, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
