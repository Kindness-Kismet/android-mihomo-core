package contract

import "encoding/json"

type Method string

const (
	MessageMethod Method = "message"

	InitClashMethod                Method = "initClash"
	GetVersionMethod               Method = "getVersion"
	GetIsInitMethod                Method = "getIsInit"
	ForceGcMethod                  Method = "forceGc"
	ShutdownMethod                 Method = "shutdown"
	ValidateConfigMethod           Method = "validateConfig"
	GetConfigMethod                Method = "getConfig"
	UpdateConfigMethod             Method = "updateConfig"
	SetupConfigMethod              Method = "setupConfig"
	GetProxiesMethod               Method = "getProxies"
	ChangeProxyMethod              Method = "changeProxy"
	GetTrafficMethod               Method = "getTraffic"
	GetTotalTrafficMethod          Method = "getTotalTraffic"
	ResetTrafficMethod             Method = "resetTraffic"
	AsyncTestDelayMethod           Method = "asyncTestDelay"
	GetConnectionsMethod           Method = "getConnections"
	CloseConnectionsMethod         Method = "closeConnections"
	ResetConnectionsMethod         Method = "resetConnections"
	CloseConnectionMethod          Method = "closeConnection"
	GetExternalProvidersMethod     Method = "getExternalProviders"
	GetExternalProviderMethod      Method = "getExternalProvider"
	UpdateGeoDataMethod            Method = "updateGeoData"
	SideLoadExternalProviderMethod Method = "sideLoadExternalProvider"
	UpdateExternalProviderMethod   Method = "updateExternalProvider"
	GetCountryCodeMethod           Method = "getCountryCode"
	GetMemoryMethod                Method = "getMemory"
	StartLogMethod                 Method = "startLog"
	StopLogMethod                  Method = "stopLog"
	StartMemoryMethod              Method = "startMemory"
	StopMemoryMethod               Method = "stopMemory"
	StartConnectionsMethod         Method = "startConnections"
	StopConnectionsMethod          Method = "stopConnections"
	StartListenerMethod            Method = "startListener"
	StopListenerMethod             Method = "stopListener"
	UpdateDnsMethod                Method = "updateDns"
	CrashMethod                    Method = "crash"
	DeleteFileMethod               Method = "deleteFile"
)

type MessageType string

const (
	LogMessage         MessageType = "log"
	DelayMessage       MessageType = "delay"
	RequestMessage     MessageType = "request"
	MemoryMessage      MessageType = "memory"
	ConnectionsMessage MessageType = "connections"
)

type Action struct {
	ID     string          `json:"id"`
	Method Method          `json:"method"`
	Data   json.RawMessage `json:"data"`
}

type Response struct {
	ID     string `json:"id"`
	Method Method `json:"method"`
	Data   any    `json:"data"`
	Code   int    `json:"code"`
}

type Message struct {
	Type MessageType `json:"type"`
	Data any         `json:"data"`
}

type InitParams struct {
	HomeDir string `json:"home-dir"`
	Version int    `json:"version"`
}

type ChangeProxyParams struct {
	GroupName string `json:"group-name"`
	ProxyName string `json:"proxy-name"`
}

type Emitter interface {
	Emit(message Message)
}

type Service interface {
	InitClash(params InitParams) bool
	GetVersion() string
	GetIsInit() bool
	ForceGC()
	Shutdown() bool

	ValidateConfig(path string) string
	GetConfig(path string) (any, error)

	UpdateConfig(payload string) string
	SetupConfig(payload string) string

	GetProxies() any
	ChangeProxy(params ChangeProxyParams) string

	GetTraffic(onlyProxy bool) string
	GetTotalTraffic(onlyProxy bool) string
	ResetTraffic()

	AsyncTestDelay(payload string) string

	GetConnections() string
	CloseConnections() bool
	ResetConnections() bool
	CloseConnection(id string) bool

	GetExternalProviders() string
	GetExternalProvider(name string) string
	UpdateGeoData(payload string) string
	SideLoadExternalProvider(payload string) string
	UpdateExternalProvider(providerName string) string

	GetCountryCode(ip string) string
	GetMemory() string

	StartLog()
	StopLog()

	StartMemory()
	StopMemory()

	StartConnections()
	StopConnections()

	StartListener() bool
	StopListener() bool

	UpdateDns(value string)
	Suspend(suspended bool) bool

	Crash()
	DeleteFile(path string) string
}
