//go:build android && cgo

package core

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"strconv"

	"github.com/metacubex/mihomo/adapter/provider"
	"github.com/metacubex/mihomo/component/mmdb"
	"github.com/metacubex/mihomo/component/updater"
	"github.com/metacubex/mihomo/config"
	cp "github.com/metacubex/mihomo/constant/provider"
	rp "github.com/metacubex/mihomo/rules/provider"
	"github.com/metacubex/mihomo/tunnel/statistic"
)

// handleGetConfig reads a config file from disk and parses it as mihomo RawConfig.
func handleGetConfig(path string) (*config.RawConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return config.UnmarshalRawConfig(data)
}

// handleGetExternalProvider returns a JSON description of the given external provider.
func handleGetExternalProvider(providerName string) string {
	coreMu.Lock()
	defer coreMu.Unlock()

	p := getExternalProvidersRaw()[providerName]
	if p == nil {
		return ""
	}

	ep, err := toExternalProvider(p)
	if err != nil {
		return ""
	}

	data, err := json.Marshal(ep)
	if err != nil {
		return ""
	}
	return string(data)
}

// handleUpdateGeoData updates Geo databases (GeoIP/GeoSite/MMDB/ASN) based on payload.
func handleUpdateGeoData(payload string) string {
	var params map[string]string
	if err := json.Unmarshal([]byte(payload), &params); err != nil {
		return err.Error()
	}

	geoType := params["geo-type"]
	if geoType == "" {
		return "missing geo-type"
	}

	var err error
	switch geoType {
	case "MMDB":
		err = updater.UpdateMMDB()
	case "ASN":
		err = updater.UpdateASN()
	case "GEOIP":
		err = updater.UpdateGeoIp()
	case "GEOSITE":
		err = updater.UpdateGeoSite()
	default:
		err = errors.New("unknown geo-type")
	}

	if err != nil {
		return err.Error()
	}
	return ""
}

// sideUpdateExternalProvider performs SideUpdate on an external provider.
func sideUpdateExternalProvider(p cp.Provider, data []byte) error {
	switch pv := p.(type) {
	case *provider.ProxySetProvider:
		_, _, err := pv.SideUpdate(data)
		return err
	case *rp.RuleSetProvider:
		_, _, err := pv.SideUpdate(data)
		return err
	default:
		return errors.New("not an external provider")
	}
}

// handleSideLoadExternalProvider side-loads data into an external provider.
func handleSideLoadExternalProvider(payload string) string {
	var params map[string]string
	if err := json.Unmarshal([]byte(payload), &params); err != nil {
		return err.Error()
	}

	providerName := params["provider-name"]
	if providerName == "" {
		return "missing provider-name"
	}

	p := getExternalProvidersRaw()[providerName]
	if p == nil {
		return "external provider not found"
	}

	data := []byte(params["data"])
	if err := sideUpdateExternalProvider(p, data); err != nil {
		return err.Error()
	}

	return ""
}

// handleGetCountryCode looks up the country/region code for an IP using MMDB.
func handleGetCountryCode(ip string) string {
	codes := mmdb.IPInstance().LookupCode(net.ParseIP(ip))
	if len(codes) == 0 {
		return ""
	}
	return codes[0]
}

// handleGetMemory returns current memory usage as a decimal string.
func handleGetMemory() string {
	return strconv.FormatUint(statistic.DefaultManager.Memory(), 10)
}

// handleCrash is used for crash testing; it terminates the process.
func handleCrash() {
	os.Exit(2)
}

// handleDeleteFile deletes a file or directory (missing path is treated as success).
func handleDeleteFile(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		return err.Error()
	}
	if info.IsDir() {
		if err := os.RemoveAll(path); err != nil {
			return err.Error()
		}
		return ""
	}
	if err := os.Remove(path); err != nil {
		return err.Error()
	}
	return ""
}
