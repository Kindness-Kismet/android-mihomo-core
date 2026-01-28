//go:build android && cgo && cmfa

package core

import (
	"strings"

	"github.com/metacubex/mihomo/dns"
	"github.com/metacubex/mihomo/log"
)

// handleUpdateDns updates system DNS (Android cmfa builds only).
func handleUpdateDns(value string) {
	go func() {
		log.Infoln("[DNS] update system DNS: %s", value)

		if strings.TrimSpace(value) == "" {
			dns.UpdateSystemDNS(nil)
		} else {
			parts := strings.Split(value, ",")
			addrs := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				addrs = append(addrs, p)
			}
			dns.UpdateSystemDNS(addrs)
		}

		dns.FlushCacheWithDefaultResolver()
	}()
}
