//go:build android && cgo

package core

import "mihomo_android_wrapper/contract"

type Options struct {
	Emitter contract.Emitter
	StopTun func()
}

var (
	emitter     contract.Emitter
	stopTunHook func()
)

// emitMessage emits an event to the host if an Emitter is set.
func emitMessage(message contract.Message) {
	if emitter == nil {
		return
	}
	emitter.Emit(message)
}

type Service struct{}

// New creates a Service and installs global hooks required by core logic.
func New(opts Options) *Service {
	emitter = opts.Emitter
	stopTunHook = opts.StopTun
	return &Service{}
}

// InitClash delegates to handleInitClash.
func (s *Service) InitClash(params contract.InitParams) bool {
	return handleInitClash(params)
}

// GetVersion delegates to handleGetVersion.
func (s *Service) GetVersion() string {
	return handleGetVersion()
}

// GetIsInit delegates to handleGetIsInit.
func (s *Service) GetIsInit() bool {
	return handleGetIsInit()
}

// ForceGC delegates to handleForceGC.
func (s *Service) ForceGC() {
	handleForceGC()
}

// Shutdown delegates to handleShutdown.
func (s *Service) Shutdown() bool {
	return handleShutdown()
}

// ValidateConfig delegates to handleValidateConfig.
func (s *Service) ValidateConfig(path string) string {
	return handleValidateConfig(path)
}

// GetConfig delegates to handleGetConfig.
func (s *Service) GetConfig(path string) (any, error) {
	return handleGetConfig(path)
}

// UpdateConfig delegates to handleUpdateConfig.
func (s *Service) UpdateConfig(payload string) string {
	return handleUpdateConfig([]byte(payload))
}

// SetupConfig delegates to handleSetupConfig.
func (s *Service) SetupConfig(payload string) string {
	return handleSetupConfig([]byte(payload))
}

// GetProxies delegates to handleGetProxies.
func (s *Service) GetProxies() any {
	return handleGetProxies()
}

// ChangeProxy delegates to handleChangeProxy.
func (s *Service) ChangeProxy(params contract.ChangeProxyParams) string {
	return handleChangeProxy(params)
}

// GetTraffic delegates to handleGetTraffic.
func (s *Service) GetTraffic(onlyProxy bool) string {
	return handleGetTraffic(onlyProxy)
}

// GetTotalTraffic delegates to handleGetTotalTraffic.
func (s *Service) GetTotalTraffic(onlyProxy bool) string {
	return handleGetTotalTraffic(onlyProxy)
}

// ResetTraffic delegates to handleResetTraffic.
func (s *Service) ResetTraffic() {
	handleResetTraffic()
}

// AsyncTestDelay delegates to handleAsyncTestDelay.
func (s *Service) AsyncTestDelay(payload string) string {
	return handleAsyncTestDelay(payload)
}

// GetConnections delegates to handleGetConnections.
func (s *Service) GetConnections() string {
	return handleGetConnections()
}

// CloseConnections delegates to handleCloseConnections.
func (s *Service) CloseConnections() bool {
	return handleCloseConnections()
}

// ResetConnections delegates to handleResetConnections.
func (s *Service) ResetConnections() bool {
	return handleResetConnections()
}

// CloseConnection delegates to handleCloseConnection.
func (s *Service) CloseConnection(id string) bool {
	return handleCloseConnection(id)
}

// GetExternalProviders delegates to handleGetExternalProviders.
func (s *Service) GetExternalProviders() string {
	return handleGetExternalProviders()
}

// GetExternalProvider delegates to handleGetExternalProvider.
func (s *Service) GetExternalProvider(name string) string {
	return handleGetExternalProvider(name)
}

// UpdateGeoData delegates to handleUpdateGeoData.
func (s *Service) UpdateGeoData(payload string) string {
	return handleUpdateGeoData(payload)
}

// SideLoadExternalProvider delegates to handleSideLoadExternalProvider.
func (s *Service) SideLoadExternalProvider(payload string) string {
	return handleSideLoadExternalProvider(payload)
}

// UpdateExternalProvider delegates to handleUpdateExternalProvider.
func (s *Service) UpdateExternalProvider(providerName string) string {
	return handleUpdateExternalProvider(providerName)
}

// GetCountryCode delegates to handleGetCountryCode.
func (s *Service) GetCountryCode(ip string) string {
	return handleGetCountryCode(ip)
}

// GetMemory delegates to handleGetMemory.
func (s *Service) GetMemory() string {
	return handleGetMemory()
}

// StartLog delegates to handleStartLog.
func (s *Service) StartLog() {
	handleStartLog()
}

// StopLog delegates to handleStopLog.
func (s *Service) StopLog() {
	handleStopLog()
}

// StartListener delegates to handleStartListener.
func (s *Service) StartListener() bool {
	return handleStartListener()
}

// StopListener delegates to handleStopListener.
func (s *Service) StopListener() bool {
	return handleStopListener()
}

// UpdateDns delegates to handleUpdateDns.
func (s *Service) UpdateDns(value string) {
	handleUpdateDns(value)
}

// Suspend delegates to handleSuspend.
func (s *Service) Suspend(suspended bool) bool {
	return handleSuspend(suspended)
}

// Crash delegates to handleCrash.
func (s *Service) Crash() {
	handleCrash()
}

// DeleteFile delegates to handleDeleteFile.
func (s *Service) DeleteFile(path string) string {
	return handleDeleteFile(path)
}
