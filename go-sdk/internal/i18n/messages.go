/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package i18n

var (
	MsgConnectedToTunnel        = Message{ZH: "已连接到隧道: %s\n", EN: "Connected to tunnel: %s\n"}
	MsgReconnectingWithErr      = Message{ZH: "连接断开，正在重连... (%v)\n", EN: "Connection lost, reconnecting... (%v)\n"}
	MsgReconnecting             = Message{ZH: "连接断开，正在重连...", EN: "Connection lost, reconnecting..."}
	MsgModeActiveForwarding     = Message{ZH: "模式：主动转发（端口来自 API）", EN: "Mode: active forwarding (ports from API)"}
	MsgAllPortsURLHint          = Message{ZH: "全端口模式：请通过 URL 访问，例如 https://%s-<port>.%s\n", EN: "All ports mode: access via URL instead, e.g. https://%s-<port>.%s\n"}
	MsgModePassiveForwarding    = Message{ZH: "模式：被动转发（端口由 Host 通过 SSH 协商）", EN: "Mode: passive forwarding (ports from host via SSH)"}
	MsgAutoReconnectEnabled     = Message{ZH: "自动重连：已启用", EN: "Auto reconnect: enabled"}
	MsgAllPortsAnyPort          = Message{ZH: "全端口模式：该隧道通过 URL 接受任意端口访问", EN: "All ports mode: this tunnel accepts any port via URL"}
	MsgAccessServiceAt          = Message{ZH: "服务访问地址: https://%s-<port>.%s\n", EN: "Access your service at: https://%s-<port>.%s\n"}
	MsgHostingPort              = Message{ZH: "托管端口: %s%d%s\n", EN: "Hosting port: %s%d%s\n"}
	MsgTunnelURL                = Message{ZH: "隧道 URL: https://%s-%d.%s\n", EN: "Tunnel URL: https://%s-%d.%s\n"}
	MsgReadyToAccept            = Message{ZH: "已就绪，等待连接", EN: "Ready to accept connections"}
	MsgGatewayRejected          = Message{ZH: "连接被网关拒绝: %s，正在重试...\n", EN: "Connection rejected by gateway: %s, retrying...\n"}
	MsgConnectionFailedRetrying = Message{ZH: "连接失败，正在重试...", EN: "Connection failed, retrying..."}

	MsgPortForwardingUnavailable = Message{ZH: "端口转发服务不可用", EN: "port forwarding service unavailable"}
	MsgSessionDisconnected       = Message{ZH: "连接已断开", EN: "disconnected"}
	MsgSessionClosed             = Message{ZH: "会话已关闭", EN: "session closed"}
	MsgPortInUse                 = Message{ZH: "端口 %d 已被占用: %w", EN: "port %d is already in use: %w"}
	MsgGatewayRejected429        = Message{ZH: "连接被网关拒绝 (429): %s", EN: "connection rejected by gateway (429): %s"}
	MsgDialCancelled             = Message{ZH: "WebSocket 连接已取消: %w", EN: "websocket dial cancelled: %w"}
	MsgDialFailedRetries         = Message{ZH: "WebSocket 连接失败（已重试 %d 次）: %w", EN: "websocket dial failed after %d retries: %w"}
	MsgInvalidExpiration         = Message{ZH: "有效期必须是 1-720 小时，当前为 %d", EN: "expiration must be 1-720 hours, got %d"}
	MsgReconnectExhausted        = Message{ZH: "重连 %d 次后仍失败: %w", EN: "reconnect failed after %d attempts: %w"}
	MsgHostKeyFailed             = Message{ZH: "生成 Host 密钥失败: %w", EN: "generate host key: %w"}
	MsgOuterSSHFailed            = Message{ZH: "SSH 外层连接失败: %w", EN: "outer SSH connect failed: %w"}
	MsgMarshalRequestFailed      = Message{ZH: "序列化请求体失败: %w", EN: "marshal request body: %w"}
	MsgCreateRequestFailed       = Message{ZH: "创建请求失败: %w", EN: "create request: %w"}
	MsgHTTPRequestFailed         = Message{ZH: "HTTP 请求失败: %w", EN: "http request failed: %w"}
	MsgReadResponseFailed        = Message{ZH: "读取响应失败: %w", EN: "read response: %w"}
	MsgServerHTTPError           = Message{ZH: "服务端错误: HTTP %d: %s", EN: "server error: HTTP %d: %s"}
	MsgUnmarshalResponseFailed   = Message{ZH: "解析响应失败: %w", EN: "unmarshal response: %w"}
)
