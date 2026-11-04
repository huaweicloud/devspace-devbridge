package netutil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"huawei.com/devbridge/internal/i18n"
)

type PingResult struct {
	StatusText string
	Latency    time.Duration
	Err        error
}

// PingURI probes a URI and classifies the outcome into one of three layers:
// tunneled-service/gateway answered (an HTTP status), the transport broke
// (tunnel disconnected), or the request never reached the tunnel infrastructure
// (no X-Served-By on a 4xx/5xx response). It no longer fakes "Bad Gateway" /
// "Gateway Timeout" for transport failures, which were indistinguishable from a
// real gateway 502/504.
// pingClient is shared across ping probes. DisableKeepAlives closes each
// connection after use so a long-running `devbridge ping` does not leak idle
// TCP connections or transports.
var pingClient = &http.Client{
	Transport: &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
	},
}

func PingURI(ctx context.Context, rawURI string, timeout time.Duration) *PingResult {
	result := &PingResult{}
	start := time.Now()

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURI, nil)
	if err != nil {
		result.Err = err
		result.StatusText = i18n.T(i18n.Msg.Ping.Disconnected)
		return result
	}

	resp, err := pingClient.Do(req)
	result.Latency = time.Since(start)
	if err != nil {
		result.Err = err
		// Transport errors (DNS/TLS failure, reset, EOF) mean the tunnel is gone; don't fake Bad Gateway/Timeout.
		result.StatusText = i18n.T(i18n.Msg.Ping.Disconnected)
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		// No X-Served-By on an HTTP error: the request never reached the tunnel (likely blocked by firewall/LB/CDN).
		if resp.Header.Get("X-Served-By") == "" {
			result.Err = errors.New(resp.Status)
			result.StatusText = i18n.T(i18n.Msg.Ping.NotServedByTun)
			return result
		}
		// X-DevBridge-Error-Origin (custom, WAF/ELB won't rewrite): "service" = tunneled service, else gateway.
		if resp.Header.Get("X-DevBridge-Error-Origin") == "service" {
			// 业务服务自己返回的 4xx/5xx：请求已成功穿过隧道到达业务，网络是通的，
			// 不算连接失败。Err 保持 nil，仅展示业务返回的状态，ping 循环继续。
			result.StatusText = fmt.Sprintf(i18n.T(i18n.Msg.Ping.ServiceError), resp.Status)
			return result
		}
		result.Err = errors.New(resp.Status)
		result.StatusText = fmt.Sprintf(i18n.T(i18n.Msg.Ping.GatewayError), resp.Status)
		return result
	}

	result.StatusText = resp.Status
	return result
}
