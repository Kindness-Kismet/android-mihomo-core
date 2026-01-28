//go:build android && cgo

package core

import (
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/metacubex/mihomo/adapter/provider"
	"github.com/metacubex/mihomo/component/profile/cachefile"
	cp "github.com/metacubex/mihomo/constant/provider"
	rp "github.com/metacubex/mihomo/rules/provider"
	"github.com/metacubex/mihomo/tunnel"
)

type ExternalProvider struct {
	Name             string                     `json:"name"`
	Type             string                     `json:"type"`
	VehicleType      string                     `json:"vehicle-type"`
	Count            int                        `json:"count"`
	Path             string                     `json:"path"`
	UpdateAt         time.Time                  `json:"update-at"`
	SubscriptionInfo *provider.SubscriptionInfo `json:"subscription-info,omitempty"`
}

// getExternalProvidersRaw returns all non-Compatible providers (proxy/rule).
func getExternalProvidersRaw() map[string]cp.Provider {
	eps := make(map[string]cp.Provider)
	for name, p := range tunnel.Providers() {
		if p == nil || p.VehicleType() == cp.Compatible {
			continue
		}
		eps[name] = p
	}
	for name, p := range tunnel.RuleProviders() {
		if p == nil || p.VehicleType() == cp.Compatible {
			continue
		}
		eps[name] = p
	}
	return eps
}

// subscriptionInfoForProvider loads cached subscription info.
func subscriptionInfoForProvider(name string) *provider.SubscriptionInfo {
	userInfo := cachefile.Cache().GetSubscriptionInfo(name)
	if userInfo == "" {
		return nil
	}
	return provider.NewSubscriptionInfo(userInfo)
}

// toExternalProvider converts a mihomo Provider into the JSON shape expected by the host.
func toExternalProvider(p cp.Provider) (*ExternalProvider, error) {
	switch pv := p.(type) {
	case *provider.ProxySetProvider:
		return &ExternalProvider{
			Name:             pv.Name(),
			Type:             pv.Type().String(),
			VehicleType:      pv.VehicleType().String(),
			Count:            pv.Count(),
			Path:             pv.Vehicle().Path(),
			UpdateAt:         pv.UpdatedAt(),
			SubscriptionInfo: subscriptionInfoForProvider(pv.Name()),
		}, nil
	case *rp.RuleSetProvider:
		return &ExternalProvider{
			Name:        pv.Name(),
			Type:        pv.Type().String(),
			VehicleType: pv.VehicleType().String(),
			Count:       pv.Count(),
			Path:        pv.Vehicle().Path(),
			UpdateAt:    pv.UpdatedAt(),
		}, nil
	default:
		return nil, errors.New("not an external provider")
	}
}

// handleGetExternalProviders returns a JSON list of all external providers.
func handleGetExternalProviders() string {
	coreMu.Lock()
	defer coreMu.Unlock()

	raw := getExternalProvidersRaw()
	list := make([]ExternalProvider, 0, len(raw))
	for _, p := range raw {
		ep, err := toExternalProvider(p)
		if err != nil {
			continue
		}
		list = append(list, *ep)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })

	data, err := json.Marshal(list)
	if err != nil {
		return ""
	}
	return string(data)
}

// handleUpdateExternalProvider triggers an update on the specified external provider.
func handleUpdateExternalProvider(providerName string) string {
	p := getExternalProvidersRaw()[providerName]
	if p == nil {
		return "external provider not found"
	}
	if err := p.Update(); err != nil {
		return err.Error()
	}
	return ""
}
