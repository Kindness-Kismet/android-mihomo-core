//go:build android && cgo

package core

import (
	"encoding/json"
	"sync"
	"time"

	"mihomo_android_wrapper/contract"

	"github.com/metacubex/mihomo/tunnel/statistic"
)

var (
	connectionsMu       sync.Mutex
	connectionsTicker   *time.Ticker
	connectionsStopChan chan struct{}
)

// handleStartConnections starts periodic connections reporting to the host.
func handleStartConnections() {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	if connectionsTicker != nil {
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	stopChan := make(chan struct{})
	connectionsTicker = ticker
	connectionsStopChan = stopChan

	go func() {
		emitConnectionsData()
		for {
			select {
			case <-ticker.C:
				emitConnectionsData()
			case <-stopChan:
				return
			}
		}
	}()
}

// handleStopConnections stops periodic connections reporting.
func handleStopConnections() {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	if connectionsTicker == nil {
		return
	}

	connectionsTicker.Stop()
	close(connectionsStopChan)
	connectionsTicker = nil
	connectionsStopChan = nil
}

func emitConnectionsData() {
	snapshot := statistic.DefaultManager.Snapshot()
	data, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	emitMessage(contract.Message{
		Type: contract.ConnectionsMessage,
		Data: json.RawMessage(data),
	})
}
