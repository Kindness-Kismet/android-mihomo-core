//go:build android && cgo

package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"mihomo_android_wrapper/contract"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/adapter/outboundgroup"
	"github.com/metacubex/mihomo/common/observable"
	"github.com/metacubex/mihomo/component/resolver"
	"github.com/metacubex/mihomo/config"
	"github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel"
	"github.com/metacubex/mihomo/tunnel/statistic"
)

var (
	coreMu sync.Mutex
	isInit = false

	logSubscriber observable.Subscription[log.Event]
)

type SetupParams struct {
	ConfigPath  string            `json:"config-path"`
	Payload     string            `json:"payload"`
	SelectedMap map[string]string `json:"selected-map"`
	TestURL     string            `json:"test-url"`
}

// handleInitClash initializes the mihomo runtime and config directory.
func handleInitClash(params contract.InitParams) bool {
	coreMu.Lock()
	defer coreMu.Unlock()

	if params.HomeDir == "" {
		log.Errorln("[APP] invalid init params: home-dir is empty")
		return false
	}
	// params.Version is reserved for Android API-level compatibility handling.

	if !isInit {
		constant.SetHomeDir(params.HomeDir)
		constant.SetConfig(filepath.Join(params.HomeDir, "config.yaml"))
		if err := config.Init(params.HomeDir); err != nil {
			log.Errorln("[APP] failed to init config directory: %s", err.Error())
			return false
		}
		isInit = true
	}

	return true
}

// handleGetIsInit reports whether InitClash has been successfully called.
func handleGetIsInit() bool {
	coreMu.Lock()
	defer coreMu.Unlock()
	return isInit
}

// handleGetVersion returns the version string with a leading "v".
func handleGetVersion() string {
	version := strings.TrimSpace(constant.Version)
	if version == "" {
		return ""
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

// handleForceGC forces a GC cycle and tries to return memory to the OS.
func handleForceGC() {
	runtime.GC()
	debug.FreeOSMemory()
}

// handleShutdown stops the VPN TUN, log subscription, and shuts down the mihomo executor.
func handleShutdown() bool {
	coreMu.Lock()
	defer coreMu.Unlock()

	if stopTunHook != nil {
		stopTunHook()
	}
	handleStopLog()
	executor.Shutdown()
	isInit = false
	return true
}

// handleValidateConfig validates a config file; returns empty string on success.
func handleValidateConfig(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return err.Error()
	}
	_, err = config.Parse(data)
	if err != nil {
		return err.Error()
	}
	return ""
}

// handleSetupConfig loads config and applies the proxy selection mapping.
// Supports two modes:
// 1. File mode: params.ConfigPath specifies the config file path
// 2. Payload mode: params.Payload contains the config content directly
func handleSetupConfig(data []byte) string {
	coreMu.Lock()
	defer coreMu.Unlock()

	if !isInit {
		return "not initialized"
	}

	var params SetupParams
	if err := json.Unmarshal(data, &params); err != nil {
		return err.Error()
	}

	var cfg *config.Config
	var err error

	if params.Payload != "" {
		// Payload mode: parse config from memory
		cfg, err = executor.ParseWithBytes([]byte(params.Payload))
		if err != nil {
			return err.Error()
		}
	} else {
		// File mode: parse config from file
		if params.ConfigPath != "" {
			if _, err := os.Stat(params.ConfigPath); err != nil {
				return "config file not found: " + params.ConfigPath
			}
			constant.SetConfig(params.ConfigPath)
		}
		cfg, err = executor.Parse()
		if err != nil {
			return err.Error()
		}
	}

	// Android provides the VPN fd via startTUN(), so we disable mihomo's built-in TUN.
	if cfg.General != nil {
		cfg.General.Tun.Enable = false
	}

	hub.ApplyConfig(cfg)
	patchSelectGroup(params.SelectedMap)
	return ""
}

// handleGetProxies returns the current proxy list (including providers).
func handleGetProxies() map[string]constant.Proxy {
	return allProxies()
}

// allProxies merges core proxies with provider proxies.
// Reason: upstream mihomo removed tunnel.ProxiesWithProviders() in v1.19.20,
// so we keep one unified path that works across old/new versions.
func allProxies() map[string]constant.Proxy {
	all := make(map[string]constant.Proxy)
	for name, proxy := range tunnel.Proxies() {
		all[name] = proxy
	}
	for _, provider := range tunnel.Providers() {
		if provider == nil {
			continue
		}
		for _, proxy := range provider.Proxies() {
			if proxy == nil {
				continue
			}
			all[proxy.Name()] = proxy
		}
	}
	return all
}

// patchSelectGroup applies host-side selections to SelectAble groups.
func patchSelectGroup(mapping map[string]string) {
	if len(mapping) == 0 {
		return
	}

	for name, proxy := range allProxies() {
		outbound, ok := proxy.(*adapter.Proxy)
		if !ok {
			continue
		}

		selector, ok := outbound.ProxyAdapter.(outboundgroup.SelectAble)
		if !ok {
			continue
		}

		selected, exist := mapping[name]
		if !exist {
			continue
		}
		selector.ForceSet(selected)
	}
}

// handleChangeProxy updates the selected proxy for a selector group.
func handleChangeProxy(params contract.ChangeProxyParams) string {
	coreMu.Lock()
	defer coreMu.Unlock()

	if params.GroupName == "" {
		return "missing group-name"
	}

	proxies := allProxies()
	group, ok := proxies[params.GroupName]
	if !ok {
		return "group not found"
	}

	adapterProxy, ok := group.(*adapter.Proxy)
	if !ok {
		return "group is not selectable"
	}

	selector, ok := adapterProxy.ProxyAdapter.(outboundgroup.SelectAble)
	if !ok {
		return "group is not selectable"
	}

	if params.ProxyName == "" {
		selector.ForceSet("")
		return ""
	}

	if err := selector.Set(params.ProxyName); err != nil {
		return err.Error()
	}

	return ""
}

// handleGetTraffic returns a JSON traffic snapshot of current upload/download.
func handleGetTraffic(_ bool) string {
	up, down := statistic.DefaultManager.Now()
	traffic := map[string]int64{
		"up":   up,
		"down": down,
	}
	data, err := json.Marshal(traffic)
	if err != nil {
		return ""
	}
	return string(data)
}

// handleGetTotalTraffic returns a JSON traffic snapshot of total upload/download.
func handleGetTotalTraffic(_ bool) string {
	snapshot := statistic.DefaultManager.Snapshot()
	traffic := map[string]int64{
		"up":   snapshot.UploadTotal,
		"down": snapshot.DownloadTotal,
	}
	data, err := json.Marshal(traffic)
	if err != nil {
		return ""
	}
	return string(data)
}

// handleResetTraffic resets traffic statistics.
func handleResetTraffic() {
	statistic.DefaultManager.ResetStatistic()
}

// handleGetConnections returns a JSON snapshot of connections.
func handleGetConnections() string {
	snapshot := statistic.DefaultManager.Snapshot()
	data, err := json.Marshal(snapshot)
	if err != nil {
		return ""
	}
	return string(data)
}

// handleCloseConnections closes all active connections.
func handleCloseConnections() bool {
	var trackers []statistic.Tracker
	statistic.DefaultManager.Range(func(c statistic.Tracker) bool {
		trackers = append(trackers, c)
		return true
	})
	for _, t := range trackers {
		_ = t.Close()
	}
	return true
}

// handleResetConnections resets DNS/TCP connection pools without restarting the core.
func handleResetConnections() bool {
	// Reset DNS/TCP pools without using Shutdown, to avoid affecting global state.
	resolver.ResetConnection()
	return true
}

// handleCloseConnection closes a single connection by tracker id.
func handleCloseConnection(id string) bool {
	c := statistic.DefaultManager.Get(id)
	if c == nil {
		return false
	}
	return c.Close() == nil
}

// handleSuspend toggles mihomo tunnel between suspended and running states.
func handleSuspend(suspended bool) bool {
	if suspended {
		tunnel.OnSuspend()
	} else {
		tunnel.OnRunning()
	}
	return true
}
