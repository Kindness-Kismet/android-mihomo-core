//go:build android && cgo

package core

import (
	"runtime"
	"sync"
	"time"

	"mihomo_android_wrapper/contract"
)

var (
	memoryMu       sync.Mutex
	memoryTicker   *time.Ticker
	memoryStopChan chan struct{}
)

// handleStartMemory starts periodic memory usage reporting to the host.
func handleStartMemory() {
	memoryMu.Lock()
	defer memoryMu.Unlock()

	if memoryTicker != nil {
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	stopChan := make(chan struct{})
	memoryTicker = ticker
	memoryStopChan = stopChan

	go func() {
		emitMemoryData()
		for {
			select {
			case <-ticker.C:
				emitMemoryData()
			case <-stopChan:
				return
			}
		}
	}()
}

// handleStopMemory stops periodic memory usage reporting.
func handleStopMemory() {
	memoryMu.Lock()
	defer memoryMu.Unlock()

	if memoryTicker == nil {
		return
	}

	memoryTicker.Stop()
	close(memoryStopChan)
	memoryTicker = nil
	memoryStopChan = nil
}

func emitMemoryData() {
	// On Android, use Go runtime memory stats instead of process RSS,
	// because the Go core is loaded as a .so into the Flutter process,
	// and process RSS includes the entire app memory.
	// Sys is the total bytes of memory obtained from the OS, which includes
	// heap, stack, and other runtime structures.
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	emitMessage(contract.Message{
		Type: contract.MemoryMessage,
		Data: map[string]uint64{
			"inuse": m.Sys,
		},
	})
}
