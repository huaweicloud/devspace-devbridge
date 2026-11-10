/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
)

// sessionCloseGrace bounds how long a caller waits for the SSH OnClosed callback
// after Session.Done() is signalled. The ssh package closes Done() just before
// it fires OnClosed, so reading the captured close reason right after Done()
// can miss it; a short grace period makes the classification deterministic.
const sessionCloseGrace = 500 * time.Millisecond

// closeReasonCapture records the SSH session close event so Host and Connect can
// tell a terminal authentication failure from a recoverable network drop.
// Both session loops install it via Session.OnClosed.
type closeReasonCapture struct {
	mu    sync.Mutex
	args  *ssh.SessionClosedEventArgs
	fired chan struct{}
}

func newCloseReasonCapture() *closeReasonCapture {
	return &closeReasonCapture{fired: make(chan struct{}, 1)}
}

// handler records the close event and signals waiters. Suitable for
// ssh.Session.OnClosed.
func (c *closeReasonCapture) handler(args *ssh.SessionClosedEventArgs) {
	if args == nil {
		return
	}
	c.mu.Lock()
	c.args = args
	c.mu.Unlock()

	select {
	case c.fired <- struct{}{}:
	default:
	}
}

// wait returns the captured close event, waiting up to grace for it to arrive.
// A zero grace means "return what has been recorded so far".
func (c *closeReasonCapture) wait(grace time.Duration) *ssh.SessionClosedEventArgs {
	if grace > 0 {
		select {
		case <-c.fired:
		case <-time.After(grace):
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.args
}

// authError returns a terminal authentication error when the close reason or
// message describes expired/invalid credentials, otherwise nil. This turns a
// post-connect auth close into a terminal error instead of a recoverable
// "connection lost" retry, for both Host and Connect.
func (c *closeReasonCapture) authError() error {
	args := c.wait(sessionCloseGrace)
	if args == nil {
		return nil
	}
	return authFailedFromClose(fmt.Sprint(args.Reason), args.Message)
}

// authFailedFromClose classifies a session close reason/message pair. Matching
// is delegated to isAuthCloseReason so the gateway wording list stays in one
// place (see websocket.go).
func authFailedFromClose(reason, message string) error {
	reasonText := strings.TrimSpace(reason)
	detail := strings.TrimSpace(message)
	if detail == "" {
		detail = reasonText
	}
	if isAuthCloseReason(reasonText + " " + message) {
		return &AuthFailedError{Message: detail}
	}
	return nil
}
