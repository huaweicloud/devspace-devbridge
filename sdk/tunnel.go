package sdk

import (
	"context"
	"fmt"
)

// CreateTunnel 创建隧道，expiration 为 nil 时使用默认有效期 72 小时
func (d *Devbridge) CreateTunnel(ctx context.Context, name, description string, expiration *int) (*Tunnel, error) {
	if !tunnelNameRegexp.MatchString(name) {
		return nil, fmt.Errorf("invalid tunnel name: %q", name)
	}
	if err := validateTunnelDescription(description); err != nil {
		return nil, err
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
	if err := d.api.Post(ctx, "/tunnels", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListTunnels 查询当前工作空间的有效隧道列表
func (d *Devbridge) ListTunnels(ctx context.Context) ([]Tunnel, error) {
	var result []Tunnel
	if err := d.api.Get(ctx, "/tunnels", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ShowTunnel 查询隧道详情
func (d *Devbridge) ShowTunnel(ctx context.Context, tunnelID string) (*TunnelDetail, error) {
	if err := validateTunnelID(tunnelID); err != nil {
		return nil, err
	}
	var result TunnelDetail
	if err := d.api.Get(ctx, fmt.Sprintf("/tunnels/%s", tunnelID), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateTunnel 更新隧道。只传需要修改的字段，nil 表示不修改
func (d *Devbridge) UpdateTunnel(ctx context.Context, tunnelID string, name, description *string, expiration *int) error {
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
		if err := validateTunnelDescription(*description); err != nil {
			return err
		}
		req.Description = description
	}
	if expiration != nil {
		if *expiration < 1 || *expiration > 720 {
			return fmt.Errorf("expiration must be 1-720 hours, got %d", *expiration)
		}
		req.Expiration = expiration
	}

	return d.api.Put(ctx, fmt.Sprintf("/tunnels/%s", tunnelID), req, nil)
}

// DeleteTunnel 删除指定隧道
func (d *Devbridge) DeleteTunnel(ctx context.Context, tunnelID string) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}
	return d.api.Delete(ctx, fmt.Sprintf("/tunnels/%s", tunnelID), nil)
}

// DeleteAllTunnels 删除当前工作空间的全部隧道
//
// ⚠️ 谨慎调用，会删除所有隧道
func (d *Devbridge) DeleteAllTunnels(ctx context.Context) error {
	return d.api.Delete(ctx, "/tunnels", nil)
}

// IssueToken 签发隧道令牌，scope 必须是 "host" 或 "connect"
func (d *Devbridge) IssueToken(ctx context.Context, tunnelID, scope string) (*TunnelToken, error) {
	if err := validateTunnelID(tunnelID); err != nil {
		return nil, err
	}
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	var result TunnelToken
	if err := d.api.Post(ctx, fmt.Sprintf("/tunnels/%s/token?scope=%s", tunnelID, scope), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetLimits 查询当前配额
func (d *Devbridge) GetLimits(ctx context.Context) (*Limits, error) {
	var result Limits
	if err := d.api.Get(ctx, "/limits", &result); err != nil {
		return nil, err
	}
	return &result, nil
}
