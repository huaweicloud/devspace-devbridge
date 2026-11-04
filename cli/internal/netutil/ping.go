package netutil

import (
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
func PingURI(rawURI string, timeout time.Duration) *PingResult {
	result := &PingResult{}
	start := time.Now()

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	req, err := http.NewRequest(http.MethodGet, rawURI, nil)
	if err != nil {
		result.Err = err
		result.StatusText = i18n.T(i18n.Msg.Ping.Disconnected)
		return result
	}

	resp, err := client.Do(req)
	result.Latency = time.Since(start)
	if err != nil {
		result.Err = err
		// 传输层错误（DNS 失败 / TLS 失败 / reset / EOF）统一归为
		// 「隧道连接断开」，不再伪装成 Bad Gateway / Gateway Timeout。
		result.StatusText = i18n.T(i18n.Msg.Ping.Disconnected)
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		// 有 HTTP 状态但缺少 X-Served-By：响应不是经隧道基础设施产生，
		// 请求可能被防火墙 / LB / CDN 拦截，根本没到达隧道服务。
		if resp.Header.Get("X-Served-By") == "" {
			result.Err = errors.New(resp.Status)
			result.StatusText = i18n.T(i18n.Msg.Ping.NotServedByTun)
			return result
		}
		// X-DevBridge-Error-Origin 区分错误产生方（自定义头，WAF/ELB 不改写）：
		// service → 隧道内服务返回；gateway/空 → 网关自产。
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
