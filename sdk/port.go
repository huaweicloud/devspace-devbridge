package devbridge

import (
	"context"
	"fmt"
)

// ──────────────────────────────────────────────────────────────
// 端口 API — Create / List / Show / Update / Delete
// ──────────────────────────────────────────────────────────────

// CreatePort 在隧道上创建端口
//
//	allowAnon := true
//	client.CreatePort(ctx, "tunnelId", 8080, "http", &allowAnon)
func (c *Client) CreatePort(ctx context.Context, tunnelID string, port int, protocol string, allowAnonymous *bool) error {
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
	return c.post(ctx, fmt.Sprintf("/tunnels/%s/ports", tunnelID), req, nil)
}

// ListPorts 查询隧道的端口列表
func (c *Client) ListPorts(ctx context.Context, tunnelID string) ([]Port, error) {
	if err := validateTunnelID(tunnelID); err != nil {
		return nil, err
	}
	var result []Port
	if err := c.get(ctx, fmt.Sprintf("/tunnels/%s/ports", tunnelID), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ShowPort 查询端口详情
func (c *Client) ShowPort(ctx context.Context, tunnelID string, port int) (*Port, error) {
	if err := validateTunnelID(tunnelID); err != nil {
		return nil, err
	}
	if err := validatePortNumber(port); err != nil {
		return nil, err
	}
	var result Port
	if err := c.get(ctx, fmt.Sprintf("/tunnels/%s/ports/%d", tunnelID, port), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePort 更新端口的匿名访问策略
//
//	allowAnon := false
//	client.UpdatePort(ctx, "tunnelId", 8080, &allowAnon)  // 禁止匿名访问
//	client.UpdatePort(ctx, "tunnelId", 8080, nil)          // 不修改
func (c *Client) UpdatePort(ctx context.Context, tunnelID string, port int, allowAnonymous *bool) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	req := updatePortRequest{AllowAnonymous: allowAnonymous}
	return c.put(ctx, fmt.Sprintf("/tunnels/%s/ports/%d", tunnelID, port), req, nil)
}

// DeletePort 删除端口
func (c *Client) DeletePort(ctx context.Context, tunnelID string, port int) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	return c.delete(ctx, fmt.Sprintf("/tunnels/%s/ports/%d", tunnelID, port), nil)
}
