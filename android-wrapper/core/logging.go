//go:build android && cgo

package core

import (
	"sync"

	"mihomo_android_wrapper/contract"

	"github.com/metacubex/mihomo/log"
)

var logMu sync.Mutex

// handleStartLog subscribes to mihomo log events and forwards them to the host.
func handleStartLog() {
	logMu.Lock()
	if logSubscriber != nil {
		log.UnSubscribe(logSubscriber)
		logSubscriber = nil
	}

	logSubscriber = log.Subscribe()
	sub := logSubscriber
	logMu.Unlock()

	go func() {
		for logData := range sub {
			if logData.LogLevel < log.Level() {
				continue
			}
			emitMessage(contract.Message{
				Type: contract.LogMessage,
				Data: map[string]string{
					"level":   logData.LogLevel.String(),
					"payload": logData.Payload,
				},
			})
		}
	}()
}

// handleStopLog stops forwarding log events to the host.
func handleStopLog() {
	logMu.Lock()
	sub := logSubscriber
	logSubscriber = nil
	logMu.Unlock()
	if sub != nil {
		log.UnSubscribe(sub)
	}
}
