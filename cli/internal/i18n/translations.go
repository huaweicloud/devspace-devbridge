package i18n

func init() {
	Msg.Common.VersionInfo = Message{ZH: "DevBridge 远程隧道端口转发工具", EN: "DevBridge remote tunnel port forwarding tool"}
	Msg.Common.Days = Message{ZH: "天", EN: "days"}
	Msg.Common.Hours = Message{ZH: "小时", EN: "hours"}
	Msg.Common.Expired = Message{ZH: "已过期", EN: "expired"}
	Msg.Common.AuthCommands = Message{ZH: "认证命令", EN: "Authentication commands"}
	Msg.Common.FlagAPIKey = Message{ZH: "API Key", EN: "API Key"}
	Msg.Common.FlagVerbose = Message{ZH: "启用调试日志", EN: "Enable debug logging"}
	Msg.Common.FlagEchoPort = Message{ZH: "本地服务器端口号 (未指定则自动分配随机端口)", EN: "Local server port number (auto-assign random port if not specified)"} //nolint:lll
	Msg.Common.FlagInterface = Message{ZH: "本地接口地址", EN: "Local interface address"}
	Msg.Common.FlagPingInterval = Message{ZH: "消息间隔(毫秒)", EN: "Interval between messages in ms"}

	Msg.Auth.LoginSuccess = Message{ZH: "登录成功", EN: "Login successful"}
	Msg.Auth.LogoutSuccess = Message{ZH: "注销成功", EN: "Logout successful"}
	Msg.Auth.LogoutShort = Message{ZH: "注销并清除凭证", EN: "Logout and clear credentials"}
	Msg.Auth.NotLoggedIn = Message{ZH: "未登录", EN: "Not logged in"}
	Msg.Auth.StatusShort = Message{ZH: "显示登录状态", EN: "Show login status"}
	Msg.Auth.UserName = Message{ZH: "用户名", EN: "User Name"}
	Msg.Auth.OpenBrowser = Message{ZH: "正在打开浏览器...", EN: "Opening browser..."}
	Msg.Auth.BrowserOpened = Message{ZH: "浏览器已打开，请在浏览器中完成登录", EN: "Browser opened, please complete login in browser"}
	Msg.Auth.NoBrowserHint = Message{ZH: "无可用浏览器，请手动登录。API Key 管理页面: %s\n例如: ./devbridge auth login --api-key=YOUR_API_KEY", EN: "No browser available, please login manually. API Key management page: %s\nFor example: ./devbridge auth login --api-key=YOUR_API_KEY"} //nolint:lll
	Msg.Auth.LoginShort = Message{ZH: "登录华为云", EN: "Login to Huawei Cloud"}
	Msg.Auth.LoginErrorHint = Message{ZH: "请前往管理页面删除 API Key 后重试。API Key 管理页面: %s", EN: "Please go to the management page to delete API Keys and try again. API Key management page: %s"} //nolint:lll
	Msg.Auth.LoggedInHuaweiCloud = Message{ZH: "已登录 (华为云 IAM)", EN: "Logged in (Huawei Cloud IAM)"}

	Msg.Tunnel.TunnelID = Message{ZH: "隧道ID", EN: "Tunnel ID"}
	Msg.Tunnel.TunnelName = Message{ZH: "隧道名称", EN: "Tunnel Name"}
	Msg.Tunnel.TunnelExpiration = Message{ZH: "隧道过期时间", EN: "Tunnel Expiration"}
	Msg.Tunnel.PortCount = Message{ZH: "端口数", EN: "Port Count"}
	Msg.Tunnel.TunnelUpdated = Message{ZH: "隧道更新成功", EN: "Tunnel updated successfully"}
	Msg.Tunnel.TunnelDeleted = Message{ZH: "隧道删除成功", EN: "Tunnel deleted successfully"}
	Msg.Tunnel.TunnelDeletedAll = Message{ZH: "所有隧道已删除", EN: "All tunnels deleted"}
	Msg.Tunnel.TunnelNotFound = Message{ZH: "隧道不存在", EN: "Tunnel not found"}
	Msg.Tunnel.TunnelNameInvalid = Message{ZH: "隧道名称格式无效: 仅中文、字母、数字、连字符(连字符不能在首尾)，长度1-64", EN: "Invalid tunnel name: only Chinese characters, digits, letters, hyphens allowed (hyphens cannot be at the beginning or end), length 1-64"} //nolint:lll
	Msg.Tunnel.TunnelDescInvalid = Message{ZH: "隧道描述无效: 仅中文、字母、数字，长度0-64", EN: "Invalid tunnel description: only Chinese characters, digits, letters, length 0-64"}                                                                       //nolint:lll
	Msg.Tunnel.TunnelExpInvalid = Message{ZH: "过期时间无效 (1-720小时)", EN: "Invalid expiration (1-720 hours)"}
	Msg.Tunnel.TunnelListEmpty = Message{ZH: "没有隧道", EN: "No tunnels found."}
	Msg.Tunnel.ListShort = Message{ZH: "列出所有隧道", EN: "List all tunnels"}
	Msg.Tunnel.CreateShort = Message{ZH: "创建新隧道", EN: "Create a new tunnel"}
	Msg.Tunnel.ShowShort = Message{ZH: "显示隧道详情", EN: "Show tunnel details"}
	Msg.Tunnel.UpdateShort = Message{ZH: "更新隧道设置", EN: "Update tunnel settings"}
	Msg.Tunnel.DeleteShort = Message{ZH: "删除隧道", EN: "Delete a tunnel"}
	Msg.Tunnel.DeleteAllShort = Message{ZH: "删除所有隧道", EN: "Delete all tunnels"}
	Msg.Tunnel.SetShort = Message{ZH: "设置默认隧道", EN: "Set default tunnel"}
	Msg.Tunnel.UnsetShort = Message{ZH: "取消默认隧道", EN: "Unset default tunnel"}
	Msg.Tunnel.DefaultTunnelSet = Message{ZH: "默认隧道已设置为", EN: "Default tunnel set to"}
	Msg.Tunnel.DefaultTunnelUnset = Message{ZH: "默认隧道已清除", EN: "Default tunnel unset"}
	Msg.Tunnel.DefaultTunnelCleared = Message{ZH: "默认隧道已随隧道删除而清除", EN: "Default tunnel cleared as the tunnel was deleted"}
	Msg.Tunnel.TokenIssueShort = Message{ZH: "颁发隧道访问令牌", EN: "Issue tunnel access token"}
	Msg.Tunnel.Scope = Message{ZH: "范围", EN: "Scope"}
	Msg.Tunnel.Token = Message{ZH: "令牌", EN: "Token"}
	Msg.Tunnel.HostConnectionCount = Message{ZH: "Host 连接数", EN: "Host Connection Count"}
	Msg.Tunnel.ClientConnectionCount = Message{ZH: "Client 连接数", EN: "Client Connection Count"}
	Msg.Tunnel.TotalUploadBytes = Message{ZH: "总上传字节", EN: "Total Upload Bytes"}
	Msg.Tunnel.TotalDownloadBytes = Message{ZH: "总下载字节", EN: "Total Download Bytes"}
	Msg.Tunnel.Name = Message{ZH: "名称", EN: "Name"}
	Msg.Tunnel.Description = Message{ZH: "描述", EN: "Description"}
	Msg.Tunnel.CreateFailed = Message{ZH: "创建隧道失败", EN: "Failed to create tunnel"}
	Msg.Tunnel.UpdateFailed = Message{ZH: "更新隧道失败", EN: "Failed to update tunnel"}
	Msg.Tunnel.FlagDescription = Message{ZH: "隧道描述", EN: "Tunnel description"}
	Msg.Tunnel.FlagExpiration = Message{ZH: "过期时间(小时, 1-720)", EN: "Expiration hours (1-720)"}
	Msg.Tunnel.FlagName = Message{ZH: "隧道名称", EN: "Tunnel name"}
	Msg.Tunnel.FlagScope = Message{ZH: "令牌范围, 选项: host, connect (必填)", EN: "Token scope, options: host, connect (required)"} //nolint:lll

	Msg.Port.Protocol = Message{ZH: "协议", EN: "Protocol"}
	Msg.Port.AllowAnonymous = Message{ZH: "允许匿名", EN: "Allow Anonymous"}
	Msg.Port.PortCreated = Message{ZH: "端口创建成功", EN: "Port created successfully"}
	Msg.Port.PortUpdated = Message{ZH: "端口更新成功", EN: "Port updated successfully"}
	Msg.Port.PortInvalid = Message{ZH: "端口必须为 1 到 65535 之间", EN: "port must be between 1 and 65535"}
	Msg.Port.ProtocolInvalid = Message{ZH: "协议无效 (http, https, auto)", EN: "Invalid protocol (http, https, auto)"}
	Msg.Port.PortListEmpty = Message{ZH: "该隧道没有绑定端口", EN: "No ports bound to this tunnel."}
	Msg.Port.Port = Message{ZH: "端口", EN: "Port"}
	Msg.Port.TunnelID = Message{ZH: "隧道ID", EN: "Tunnel ID"}
	Msg.Port.PortCreateShort = Message{ZH: "为持久隧道添加本地端口", EN: "Add a local port to a persistent tunnel"}
	Msg.Port.PortListShort = Message{ZH: "列出隧道端口", EN: "List ports of a tunnel"}
	Msg.Port.PortShowShort = Message{ZH: "显示端口详情", EN: "Show port details"}
	Msg.Port.PortUpdateShort = Message{ZH: "更新端口设置", EN: "Update port settings"}
	Msg.Port.PortDeleteShort = Message{ZH: "从隧道删除端口", EN: "Delete a port from tunnel"}
	Msg.Port.PortShort = Message{ZH: "管理隧道端口", EN: "Manage tunnel ports"}
	Msg.Port.FlagPortNumber = Message{ZH: "端口号", EN: "Port number"}
	Msg.Port.FlagPortRequired = Message{ZH: "端口号 (必填)", EN: "Port number (required)"}
	Msg.Port.FlagProtocol = Message{ZH: "端口协议 (选项: http/https/auto)", EN: "Port protocol (options: http/https/auto)"}
	Msg.Port.FlagAllowAnon = Message{ZH: "允许匿名客户端访问", EN: "Allow anonymous client access"}
	Msg.Port.FlagDenyAnon = Message{ZH: "禁止匿名客户端访问", EN: "Deny anonymous client access"}

	Msg.API.ServerError = Message{ZH: "服务器错误", EN: "Server error"}
	Msg.API.Unauthorized = Message{ZH: "未授权", EN: "Unauthorized"}
	Msg.API.InvalidResponse = Message{ZH: "无效响应", EN: "Invalid response"}
	Msg.API.APIKeyExpired = Message{ZH: "API Key 已过期，请重新登录", EN: "API key expired, please login again"}

	Msg.Limits.LimitsShort = Message{ZH: "查看限制和余额", EN: "View limits and balance"}
	Msg.Limits.ActiveTunnels = Message{ZH: "活跃隧道数", EN: "Active Tunnels"}
	Msg.Limits.MaxTunnels = Message{ZH: "隧道数上限", EN: "Max Tunnels"}
	Msg.Limits.MaxPortsPerTunnel = Message{ZH: "单隧道端口数上限", EN: "Max Ports Per Tunnel"}
	Msg.Limits.MaxHostsPerTunnel = Message{ZH: "单隧道 Host 数上限", EN: "Max Hosts Per Tunnel"}
	Msg.Limits.MaxHTTPRequestsPerMinutePerPort = Message{ZH: "单端口 HTTP 请求频率上限", EN: "Max HTTP Requests Per Minute Per Port"} //nolint:lll
	Msg.Limits.MaxConnectionsPerPort = Message{ZH: "单端口连接数上限", EN: "Max Connections Per Port"}
	Msg.Limits.QuotaResetAt = Message{ZH: "配额重置时间", EN: "Quota Reset At"}
	Msg.Limits.QuotaBytes = Message{ZH: "流量配额", EN: "Quota Bytes"}
	Msg.Limits.Current = Message{ZH: "当前用量", EN: "Current"}
	Msg.Limits.MaxTunnelBandwidth = Message{ZH: "隧道带宽上限", EN: "Max Tunnel Bandwidth"}

	Msg.Echo.EchoStarted = Message{ZH: "Echo 服务已启动", EN: "Echo service started"}
	Msg.Echo.EchoShort = Message{ZH: "启动 Echo 诊断服务", EN: "Start Echo diagnostic service"}
	Msg.Echo.Method = Message{ZH: "方法", EN: "Method"}
	Msg.Echo.URL = Message{ZH: "URL", EN: "URL"}
	Msg.Echo.Host = Message{ZH: "Host", EN: "Host"}
	Msg.Echo.RemoteAddr = Message{ZH: "远程地址", EN: "Remote Addr"}
	Msg.Echo.Proto = Message{ZH: "协议", EN: "Proto"}
	Msg.Echo.Headers = Message{ZH: "请求头", EN: "Headers"}

	Msg.Ping.PingShort = Message{ZH: "测试到服务器的连通性", EN: "Test connectivity to the server"}
	Msg.Ping.URIInvalid = Message{ZH: "URI 无效", EN: "Invalid URI"}
}
