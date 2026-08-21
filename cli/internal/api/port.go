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
	Port           int    `json:"port"`
	AllowAnonymous *bool  `json:"allowAnonymous,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
}

// updatePortRequest 仅含 allowAnonymous，更新端口时不需要 port 入参。
type updatePortRequest struct {
	AllowAnonymous *bool `json:"allowAnonymous,omitempty"`
}

type ListPortsResult struct {
	Port           uint16 `json:"port"`
	Protocol       string `json:"protocol"`
	AllowAnonymous bool   `json:"allowAnonymous"`
	TunnelId       string `json:"tunnelId"`
}

type ShowPortResult struct {
	TunnelId       string `json:"tunnelId"`
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

func CreatePort(tunnelId string, port int, protocol string, allowAnonymous *bool) error {
	if err := ValidateTunnelId(tunnelId); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	if err := validateProtocol(protocol); err != nil {
		return err
	}
	req := portRequest{Port: port, Protocol: protocol, AllowAnonymous: allowAnonymous}
	return post(fmt.Sprintf(createPortPath, tunnelId), req, nil)
}

func ListPorts(tunnelId string) ([]ListPortsResult, error) {
	if err := ValidateTunnelId(tunnelId); err != nil {
		return nil, err
	}
	var result []ListPortsResult
	if err := get(fmt.Sprintf(listPortsPath, tunnelId), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func UpdatePort(tunnelId string, port int, allowAnonymous *bool) error {
	if err := ValidateTunnelId(tunnelId); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	req := updatePortRequest{AllowAnonymous: allowAnonymous}
	return put(fmt.Sprintf(updatePortPath, tunnelId, port), req, nil)
}

func DeletePort(tunnelId string, port int) error {
	if err := ValidateTunnelId(tunnelId); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	return deleteReq(fmt.Sprintf(deletePortPath, tunnelId, port), nil)
}

func ShowPort(tunnelId string, port int) (*ShowPortResult, error) {
	if err := ValidateTunnelId(tunnelId); err != nil {
		return nil, err
	}
	if err := validatePortNumber(port); err != nil {
		return nil, err
	}
	var result ShowPortResult
	if err := get(fmt.Sprintf(showPortPath, tunnelId, port), &result); err != nil {
		return nil, err
	}
	return &result, nil
}
