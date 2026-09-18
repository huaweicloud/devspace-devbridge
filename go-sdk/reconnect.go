/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Reconnect tuning shared by Host and Connect. Both session loops use the
// same exponential backoff so a service behaves consistently whether it is
// hosting a tunnel or connecting to one.
const (
	maxReconnectAttempts = 5
	baseReconnectDelay   = 3 * time.Second
	maxReconnectDelay    = 30 * time.Second
)

// reconnectSession is a single attempt to establish and run one session.
// connected reports whether the session reached a usable state before ending
// (host: relay channels accepted; connect: forwarding established).
type reconnectSession func() (connected bool, err error)

// reconnectDecision lets callers customize per-iteration behavior:
//   - shouldStop reports a non-recoverable error that must abort immediately
//     (e.g. quota exceeded, tunnel not found, duplicate host before first connect).
//     It runs before state updates, so callbacks reading pre-iteration state
//     (like everConnected) see the previous session's value.
//   - onConnected runs once a session connects (before the failure counter resets).
//   - onReconnect runs before waiting for the backoff delay (e.g. status output).
type reconnectDecision struct {
	shouldStop  func(err error) bool
	onConnected func()
	onReconnect func(err error)
}

// reconnectLoop runs session attempts with exponential backoff (3s base,
// doubling, capped at 30s) until ctx is canceled, a stop error occurs, or
// maxReconnectAttempts consecutive failures are exhausted.
//
// The first attempt runs immediately; failures are counted consecutively and
// reset once a session connects. On ctx cancellation it returns nil.
func (d *Devbridge) reconnectLoop(ctx context.Context, run reconnectSession, decision reconnectDecision) error {
	consecutiveFailures := 0
	for {
		connected, err := run()
		if ctx.Err() != nil {
			return nil
		}
		if decision.shouldStop != nil && decision.shouldStop(err) {
			return err
		}
		if connected {
			consecutiveFailures = 0
			if decision.onConnected != nil {
				decision.onConnected()
			}
		} else {
			consecutiveFailures++
		}
		if consecutiveFailures >= maxReconnectAttempts {
			d.logger.Debug("reconnect exhausted", "maxAttempts", maxReconnectAttempts, "err", err)
			return fmt.Errorf("reconnect failed after %d attempts: %w", maxReconnectAttempts, err)
		}

		if decision.onReconnect != nil {
			decision.onReconnect(err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(reconnectDelay(consecutiveFailures)):
		}
	}
}

// reconnectDelay returns the exponential backoff for the nth consecutive
// failure: base 3s doubling each time, capped at 30s.
func reconnectDelay(failureCount int) time.Duration {
	shift := failureCount - 1
	if shift > 4 {
		shift = 4
	}
	delay := baseReconnectDelay << uint(shift)
	if delay > maxReconnectDelay {
		delay = maxReconnectDelay
	}
	return delay
}

// gatewayRejectedError reports whether err is a terminal gateway rejection
// (quota exceeded or tunnel not found) that must not be retried.
// Duplicate host is deliberately excluded: Host treats it as fatal only
// before the first successful connect, and retriable afterwards.
func gatewayRejectedError(err error) bool {
	return errors.Is(err, ErrQuotaExceeded) || errors.Is(err, ErrTunnelNotFound)
}