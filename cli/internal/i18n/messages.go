package i18n

type Messages struct {
	Auth   AuthMessages
	Tunnel TunnelMessages
	Port   PortMessages
	Common CommonMessages
	Limits LimitsMessages
	Echo   EchoMessages
	Ping   PingMessages
	Config ConfigMessages
}

type AuthMessages struct {
	LoginSuccess        Message
	LogoutSuccess       Message
	LogoutShort         Message
	NotLoggedIn         Message
	StatusShort         Message
	UserName            Message
	OpenBrowser         Message
	BrowserOpened       Message
	NoBrowserHint       Message
	LoginShort          Message
	LoggedInHuaweiCloud Message
	LoginErrorHint      Message
}

type TunnelMessages struct {
	TunnelID              Message
	TunnelExpiration      Message
	PortCount             Message
	TunnelUpdated         Message
	TunnelDeleted         Message
	TunnelDeletedAll      Message
	TunnelNotFound        Message
	TunnelDescInvalid     Message
	TunnelListEmpty       Message
	ListShort             Message
	CreateShort           Message
	ShowShort             Message
	UpdateShort           Message
	DeleteShort           Message
	DeleteAllShort        Message
	SetShort              Message
	UnsetShort            Message
	DefaultTunnelSet      Message
	DefaultTunnelUnset    Message
	DefaultTunnelCleared  Message
	TokenIssueShort       Message
	Scope                 Message
	Token                 Message
	HostConnectionCount   Message
	ClientConnectionCount Message
	TotalUploadBytes      Message
	TotalDownloadBytes    Message
	Name                  Message
	Description           Message
	CreateFailed          Message
	UpdateFailed          Message
	FlagDescription       Message
	FlagExpiration        Message
	FlagName              Message
	FlagScope             Message
}

type PortMessages struct {
	Protocol         Message
	AllowAnonymous   Message
	PortCreated      Message
	PortUpdated      Message
	PortListEmpty    Message
	Port             Message
	TunnelID         Message
	PortCreateShort  Message
	PortListShort    Message
	PortShowShort    Message
	PortUpdateShort  Message
	PortDeleteShort  Message
	PortShort        Message
	FlagPortNumber   Message
	FlagPortRequired Message
	FlagProtocol     Message
	FlagAllowAnon    Message
	FlagDenyAnon     Message
}

type CommonMessages struct {
	VersionInfo      Message
	Days             Message
	Hours            Message
	Expired          Message
	AuthCommands     Message
	FlagAPIKey       Message
	FlagVerbose      Message
	FlagEchoPort     Message
	FlagInterface    Message
	FlagPingInterval Message
}

type LimitsMessages struct {
	LimitsShort                     Message
	ActiveTunnels                   Message
	MaxTunnels                      Message
	MaxPortsPerTunnel               Message
	MaxHostsPerTunnel               Message
	MaxHTTPRequestsPerMinutePerPort Message
	MaxConnectionsPerPort           Message
	QuotaResetAt                    Message
	QuotaBytes                      Message
	Current                         Message
	MaxTunnelBandwidth              Message
}

type EchoMessages struct {
	EchoStarted Message
	EchoShort   Message
	Method      Message
	URL         Message
	Host        Message
	RemoteAddr  Message
	Proto       Message
	Headers     Message
}

type PingMessages struct {
	PingShort Message
}

type ConfigMessages struct {
	ConfigCommands Message
	GetShort       Message
	SetShort       Message
	UnsetShort     Message
	GatewayAddr    Message
	GatewayHost    Message
	SetSuccess     Message
	NothingToSet   Message
}

var Msg Messages
