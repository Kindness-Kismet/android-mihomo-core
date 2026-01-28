//go:build android && cgo

package core

import (
	"encoding/json"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/component/dialer"
	"github.com/metacubex/mihomo/component/process"
	"github.com/metacubex/mihomo/component/resolver"
	"github.com/metacubex/mihomo/hub"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel"
)

type UpdateParams struct {
	AllowLan           *bool                    `json:"allow-lan"`
	MixedPort          *int                     `json:"mixed-port"`
	FindProcessMode    *process.FindProcessMode `json:"find-process-mode"`
	Mode               *tunnel.TunnelMode       `json:"mode"`
	LogLevel           *log.LogLevel            `json:"log-level"`
	IPv6               *bool                    `json:"ipv6"`
	Sniffing           *bool                    `json:"sniffing"`
	TCPConcurrent      *bool                    `json:"tcp-concurrent"`
	ExternalController *string                  `json:"external-controller"`
	Interface          *string                  `json:"interface-name"`
	UnifiedDelay       *bool                    `json:"unified-delay"`
}

// handleUpdateConfig incrementally updates the loaded config without restarting the core.
func handleUpdateConfig(data []byte) string {
	coreMu.Lock()
	defer coreMu.Unlock()

	if !isInit {
		return "not initialized"
	}

	var params UpdateParams
	if err := json.Unmarshal(data, &params); err != nil {
		return err.Error()
	}

	cfg, err := executor.Parse()
	if err != nil {
		return err.Error()
	}

	// Android provides the VPN fd via startTUN(), so we disable mihomo's built-in TUN.
	// This avoids attempting to open /dev/net/tun (which usually requires root).
	if cfg.General != nil {
		cfg.General.Tun.Enable = false
	}

	if cfg.General != nil {
		if params.AllowLan != nil {
			cfg.General.AllowLan = *params.AllowLan
		}
		if params.MixedPort != nil {
			cfg.General.MixedPort = *params.MixedPort
		}
		if params.Sniffing != nil {
			cfg.General.Sniffing = *params.Sniffing
		}
		if params.FindProcessMode != nil {
			cfg.General.FindProcessMode = *params.FindProcessMode
		}
		if params.TCPConcurrent != nil {
			cfg.General.TCPConcurrent = *params.TCPConcurrent
		}
		if params.Interface != nil {
			cfg.General.Interface = *params.Interface
		}
		if params.UnifiedDelay != nil {
			cfg.General.UnifiedDelay = *params.UnifiedDelay
		}
		if params.Mode != nil {
			cfg.General.Mode = *params.Mode
		}
		if params.LogLevel != nil {
			cfg.General.LogLevel = *params.LogLevel
		}
		if params.IPv6 != nil {
			cfg.General.IPv6 = *params.IPv6
		}
	}

	if cfg.Controller != nil && params.ExternalController != nil {
		cfg.Controller.ExternalController = *params.ExternalController
	}

	hub.ApplyConfig(cfg)

	// Keep behavior consistent with the old implementation: sync global state for hosts relying on side effects.
	if cfg.General != nil {
		adapter.UnifiedDelay.Store(cfg.General.UnifiedDelay)
		dialer.SetTcpConcurrent(cfg.General.TCPConcurrent)
		dialer.DefaultInterface.Store(cfg.General.Interface)
		tunnel.SetSniffing(cfg.General.Sniffing)
		tunnel.SetFindProcessMode(cfg.General.FindProcessMode)
		tunnel.SetMode(cfg.General.Mode)
		log.SetLevel(cfg.General.LogLevel)
		resolver.DisableIPv6 = !cfg.General.IPv6
	}

	return ""
}
