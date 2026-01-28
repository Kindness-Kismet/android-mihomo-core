//go:build android && cgo

package core

import (
	"github.com/metacubex/mihomo/component/resolver"
	"github.com/metacubex/mihomo/hub"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/hub/route"
	"github.com/metacubex/mihomo/listener"
	LC "github.com/metacubex/mihomo/listener/config"
	"github.com/metacubex/mihomo/tunnel"
)

// handleStartListener reapplies the current config and recreates inbound listeners.
func handleStartListener() bool {
	coreMu.Lock()
	defer coreMu.Unlock()

	if !isInit {
		return false
	}

	cfg, err := executor.Parse()
	if err != nil {
		return false
	}

	// See handleUpdateConfig: Android provides the VPN fd via startTUN().
	if cfg.General != nil {
		cfg.General.Tun.Enable = false
	}

	hub.ApplyConfig(cfg)
	resolver.ResetConnection()
	return true
}

// handleStopListener stops all inbound listeners without stopping the core process.
func handleStopListener() bool {
	coreMu.Lock()
	defer coreMu.Unlock()

	// Stop common inbound listeners.
	listener.ReCreateHTTP(0, tunnel.Tunnel)
	listener.ReCreateSocks(0, tunnel.Tunnel)
	listener.ReCreateRedir(0, tunnel.Tunnel)
	listener.ReCreateTProxy(0, tunnel.Tunnel)
	listener.ReCreateMixed(0, tunnel.Tunnel)
	listener.ReCreateShadowSocks("", tunnel.Tunnel)
	listener.ReCreateVmess("", tunnel.Tunnel)
	listener.ReCreateTuic(LC.TuicServer{}, tunnel.Tunnel)
	listener.PatchInboundListeners(nil, tunnel.Tunnel, true)
	listener.Cleanup()

	route.ReCreateServer(&route.Config{})
	resolver.ResetConnection()
	return true
}
