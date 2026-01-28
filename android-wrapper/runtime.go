//go:build android && cgo

package main

import (
	"sync"

	"mihomo_android_wrapper/api"
	"mihomo_android_wrapper/contract"
	"mihomo_android_wrapper/core"
)

var (
	runtimeOnce   sync.Once
	runtimeSvc    contract.Service
	runtimeRouter *api.Dispatcher
)

type runtimeEmitter struct{}

// Emit forwards core events (logs, delay, etc) to the host listener callback.
func (runtimeEmitter) Emit(message contract.Message) {
	sendMessage(message)
}

// ensureRuntime initializes the singleton Service and Dispatcher used by exported symbols.
func ensureRuntime() {
	runtimeOnce.Do(func() {
		runtimeSvc = core.New(core.Options{
			Emitter: runtimeEmitter{},
			StopTun: stopTun,
		})
		runtimeRouter = api.New(runtimeSvc)
	})
}

// getService returns the singleton contract.Service.
func getService() contract.Service {
	ensureRuntime()
	return runtimeSvc
}

// getDispatcher returns the singleton API dispatcher (method routing).
func getDispatcher() *api.Dispatcher {
	ensureRuntime()
	return runtimeRouter
}
