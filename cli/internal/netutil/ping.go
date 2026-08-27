package netutil

import (
	"crypto/tls"
	"net/http"
	"time"
)

type PingResult struct {
	StatusText string
	Latency    time.Duration
	Err        error
}

func PingURI(rawURI string, timeout time.Duration) *PingResult {
	result := &PingResult{}
	start := time.Now()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	req, err := http.NewRequest(http.MethodGet, rawURI, nil)
	if err != nil {
		result.Err = err
		result.StatusText = "Bad Gateway"
		return result
	}

	resp, err := client.Do(req)
	result.Latency = time.Since(start)
	if err != nil {
		result.Err = err
		if isTimeout(err) {
			result.StatusText = "Gateway Timeout"
		} else {
			result.StatusText = "Bad Gateway"
		}
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	result.StatusText = resp.Status
	return result
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	type timeout interface{ Timeout() bool }
	if t, ok := err.(timeout); ok {
		return t.Timeout()
	}
	return false
}
