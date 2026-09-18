/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"context"
	"fmt"
)

// CreateTunnel creates a tunnel; a nil expiration uses the default 72-hour validity.
func (d *Devbridge) CreateTunnel(ctx context.Context, name, description string, expiration *int) (*Tunnel, error) {
	if err := validateTunnelName(name); err != nil {
		return nil, err
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

// ListTunnels lists the active tunnels in the current workspace.
func (d *Devbridge) ListTunnels(ctx context.Context) ([]Tunnel, error) {
	var result []Tunnel
	if err := d.api.Get(ctx, "/tunnels", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ShowTunnel returns the details of a tunnel.
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

// UpdateTunnel updates a tunnel; only non-nil fields are modified.
func (d *Devbridge) UpdateTunnel(ctx context.Context, tunnelID string, name, description *string, expiration *int) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}

	req := updateTunnelRequest{}
	if name != nil {
		if err := validateTunnelName(*name); err != nil {
			return err
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

// DeleteTunnel deletes the specified tunnel.
func (d *Devbridge) DeleteTunnel(ctx context.Context, tunnelID string) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}
	return d.api.Delete(ctx, fmt.Sprintf("/tunnels/%s", tunnelID), nil)
}

// DeleteAllTunnels deletes all tunnels in the current workspace.
//
// Use with caution: this deletes every tunnel.
func (d *Devbridge) DeleteAllTunnels(ctx context.Context) error {
	return d.api.Delete(ctx, "/tunnels", nil)
}

// IssueToken issues a tunnel token; scope must be "host" or "connect".
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

// GetLimits returns the current quota.
func (d *Devbridge) GetLimits(ctx context.Context) (*Limits, error) {
	var result Limits
	if err := d.api.Get(ctx, "/limits", &result); err != nil {
		return nil, err
	}
	return &result, nil
}
