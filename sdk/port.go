package sdk

import (
	"context"
	"fmt"
)

// CreatePort 在隧道上创建端口
func (d *Devbridge) CreatePort(ctx context.Context, tunnelID string, port int, protocol string, allowAnonymous *bool) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	if err := validateProtocol(protocol); err != nil {
		return err
	}

	req := createPortRequest{
		Port:           port,
		Protocol:       protocol,
		AllowAnonymous: allowAnonymous,
	}
	return d.api.Post(ctx, fmt.Sprintf("/tunnels/%s/ports", tunnelID), req, nil)
}

// ListPorts 查询隧道的端口列表
func (d *Devbridge) ListPorts(ctx context.Context, tunnelID string) ([]Port, error) {
	if err := validateTunnelID(tunnelID); err != nil {
		return nil, err
	}
	var result []Port
	if err := d.api.Get(ctx, fmt.Sprintf("/tunnels/%s/ports", tunnelID), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ShowPort 查询端口详情
func (d *Devbridge) ShowPort(ctx context.Context, tunnelID string, port int) (*Port, error) {
	if err := validateTunnelID(tunnelID); err != nil {
		return nil, err
	}
	if err := validatePortNumber(port); err != nil {
		return nil, err
	}
	var result Port
	if err := d.api.Get(ctx, fmt.Sprintf("/tunnels/%s/ports/%d", tunnelID, port), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePort 更新端口的匿名访问策略，allowAnonymous 为 nil 时不修改
func (d *Devbridge) UpdatePort(ctx context.Context, tunnelID string, port int, allowAnonymous *bool) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	req := updatePortRequest{AllowAnonymous: allowAnonymous}
	return d.api.Put(ctx, fmt.Sprintf("/tunnels/%s/ports/%d", tunnelID, port), req, nil)
}

// DeletePort 删除端口
func (d *Devbridge) DeletePort(ctx context.Context, tunnelID string, port int) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	return d.api.Delete(ctx, fmt.Sprintf("/tunnels/%s/ports/%d", tunnelID, port), nil)
}
