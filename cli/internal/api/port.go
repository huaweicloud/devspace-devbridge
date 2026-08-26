package api

import (
	"fmt"

	"huawei.com/devbridge/internal/i18n"
)

const (
	listPortsPath  = "/tunnels/%s/ports"
	createPortPath = "/tunnels/%s/ports"
	updatePortPath = "/tunnels/%s/ports/%d"
	deletePortPath = "/tunnels/%s/ports/%d"
	showPortPath   = "/tunnels/%s/ports/%d"
)

type portRequest struct {
	Protocol       string `json:"protocol,omitempty"`
	AllowAnonymous *bool  `json:"allowAnonymous,omitempty"`
	Port           int    `json:"port"`
}

type updatePortRequest struct {
	AllowAnonymous *bool `json:"allowAnonymous,omitempty"`
}

type ListPortsResult struct {
	Port           uint16 `json:"port"`
	Protocol       string `json:"protocol"`
	AllowAnonymous bool   `json:"allowAnonymous"`
	TunnelID       string `json:"tunnelId"`
}

type ShowPortResult struct {
	TunnelID       string `json:"tunnelId"`
	Port           uint16 `json:"port"`
	Protocol       string `json:"protocol"`
	AllowAnonymous bool   `json:"allowAnonymous"`
}

func validatePortNumber(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s", i18n.T(i18n.Msg.Port.PortInvalid))
	}
	return nil
}

func validateProtocol(protocol string) error {
	switch protocol {
	case "http", "https", "auto", "":
		return nil
	default:
		return fmt.Errorf("%s: %s", i18n.T(i18n.Msg.Port.ProtocolInvalid), protocol)
	}
}

func CreatePort(tunnelID string, port int, protocol string, allowAnonymous *bool) error {
	if err := ValidateTunnelID(tunnelID); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	if err := validateProtocol(protocol); err != nil {
		return err
	}
	req := portRequest{Port: port, Protocol: protocol, AllowAnonymous: allowAnonymous}
	return post(fmt.Sprintf(createPortPath, tunnelID), req, nil)
}

func ListPorts(tunnelID string) ([]ListPortsResult, error) {
	if err := ValidateTunnelID(tunnelID); err != nil {
		return nil, err
	}
	var result []ListPortsResult
	if err := get(fmt.Sprintf(listPortsPath, tunnelID), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func UpdatePort(tunnelID string, port int, allowAnonymous *bool) error {
	if err := ValidateTunnelID(tunnelID); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	req := updatePortRequest{AllowAnonymous: allowAnonymous}
	return put(fmt.Sprintf(updatePortPath, tunnelID, port), req, nil)
}

func DeletePort(tunnelID string, port int) error {
	if err := ValidateTunnelID(tunnelID); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	return deleteReq(fmt.Sprintf(deletePortPath, tunnelID, port), nil)
}

func ShowPort(tunnelID string, port int) (*ShowPortResult, error) {
	if err := ValidateTunnelID(tunnelID); err != nil {
		return nil, err
	}
	if err := validatePortNumber(port); err != nil {
		return nil, err
	}
	var result ShowPortResult
	if err := get(fmt.Sprintf(showPortPath, tunnelID, port), &result); err != nil {
		return nil, err
	}
	return &result, nil
}
