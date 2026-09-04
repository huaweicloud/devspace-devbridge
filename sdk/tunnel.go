package devbridge

import (
	"context"
	"fmt"
)

// ──────────────────────────────────────────────────────────────
// 隧道 API — Create / List / Show / Update / Delete / DeleteAll
// ──────────────────────────────────────────────────────────────

// CreateTunnel 创建隧道
//
//	client.CreateTunnel(ctx, "my-tunnel", "描述", nil)  // 默认有效期 72h
//	exp := 24
//	client.CreateTunnel(ctx, "my-tunnel", "", &exp)     // 24h
func (c *Client) CreateTunnel(ctx context.Context, name, description string, expiration *int) (*Tunnel, error) {
	if !tunnelNameRegexp.MatchString(name) {
		return nil, fmt.Errorf("invalid tunnel name: %q", name)
	}

	req := createTunnelRequest{
		Name:        name,
		Description: description,
		ClusterID:   DefaultClusterID,
	}
	if expiration != nil {
		if *expiration < 1 || *expiration > 720 {
			return nil, fmt.Errorf("expiration must be 1-720 hours, got %d", *expiration)
		}
		req.Expiration = *expiration
	}

	var result Tunnel
	if err := c.post(ctx, "/tunnels", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListTunnels 查询当前工作空间的有效隧道列表
func (c *Client) ListTunnels(ctx context.Context) ([]Tunnel, error) {
	var result []Tunnel
	if err := c.get(ctx, "/tunnels", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ShowTunnel 查询隧道详情
func (c *Client) ShowTunnel(ctx context.Context, tunnelID string) (*TunnelDetail, error) {
	if err := validateTunnelID(tunnelID); err != nil {
		return nil, err
	}
	var result TunnelDetail
	if err := c.get(ctx, fmt.Sprintf("/tunnels/%s", tunnelID), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateTunnel 更新隧道
//
// 只传需要修改的字段，nil 表示不修改：
//
//	name := "new-name"
//	desc := "new-desc"
//	client.UpdateTunnel(ctx, "tunnelId", &name, &desc, nil)  // 改名称和描述，不改有效期
func (c *Client) UpdateTunnel(ctx context.Context, tunnelID string, name, description *string, expiration *int) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}

	req := updateTunnelRequest{}
	if name != nil {
		if !tunnelNameRegexp.MatchString(*name) {
			return fmt.Errorf("invalid tunnel name: %q", *name)
		}
		req.Name = name
	}
	if description != nil {
		req.Description = description
	}
	if expiration != nil {
		if *expiration < 1 || *expiration > 720 {
			return fmt.Errorf("expiration must be 1-720 hours, got %d", *expiration)
		}
		req.Expiration = expiration
	}

	return c.put(ctx, fmt.Sprintf("/tunnels/%s", tunnelID), req, nil)
}

// DeleteTunnel 删除指定隧道
func (c *Client) DeleteTunnel(ctx context.Context, tunnelID string) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}
	return c.delete(ctx, fmt.Sprintf("/tunnels/%s", tunnelID), nil)
}

// DeleteAllTunnels 删除当前工作空间的全部隧道
//
// ⚠️ 谨慎调用，会删除所有隧道
func (c *Client) DeleteAllTunnels(ctx context.Context) error {
	return c.delete(ctx, "/tunnels", nil)
}

// IssueToken 签发隧道令牌
//
//	scope 必须是 "host" 或 "connect"：
//
//	token, _ := client.IssueToken(ctx, "tunnelId", "host")    // Host 令牌
//	token, _ := client.IssueToken(ctx, "tunnelId", "connect") // Connect 令牌
func (c *Client) IssueToken(ctx context.Context, tunnelID, scope string) (*TunnelToken, error) {
	if err := validateTunnelID(tunnelID); err != nil {
		return nil, err
	}
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	var result TunnelToken
	if err := c.post(ctx, fmt.Sprintf("/tunnels/%s/token?scope=%s", tunnelID, scope), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetLimits 查询当前配额
func (c *Client) GetLimits(ctx context.Context) (*Limits, error) {
	var result Limits
	if err := c.get(ctx, "/limits", &result); err != nil {
		return nil, err
	}
	return &result, nil
}
