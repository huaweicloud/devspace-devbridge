/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"context"
	"fmt"
)

// CreatePort creates a port on a tunnel.
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

// ListPorts lists the ports of a tunnel.
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

// ShowPort returns the details of a port.
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

// UpdatePort updates the anonymous access policy of a port; a nil allowAnonymous leaves it unchanged.
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

// DeletePort deletes a port.
func (d *Devbridge) DeletePort(ctx context.Context, tunnelID string, port int) error {
	if err := validateTunnelID(tunnelID); err != nil {
		return err
	}
	if err := validatePortNumber(port); err != nil {
		return err
	}
	return d.api.Delete(ctx, fmt.Sprintf("/tunnels/%s/ports/%d", tunnelID, port), nil)
}
