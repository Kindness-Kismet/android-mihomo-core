//go:build android && cgo

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"mihomo_android_wrapper/contract"
)

var eventListenerSlot callbackSlot

// invokeAction is the host entrypoint. It runs asynchronously and returns a JSON {id, method, data, code}.
// It uses recover to prevent panics from crashing the process.
//
//export invokeAction
func invokeAction(callback unsafe.Pointer, paramsChar *C.char) {
	params := takeCString(paramsChar)

	var action contract.Action
	if err := json.Unmarshal([]byte(params), &action); err != nil {
		(&ActionResult{
			Code:     -1,
			Data:     err.Error(),
			callback: callback,
		}).send()
		return
	}

	go func(action contract.Action, callback unsafe.Pointer) {
		sent := false
		defer func() {
			if r := recover(); r != nil {
				// Recover panics to keep the host-side protocol stable.
				if !sent {
					(&ActionResult{
						ID:       action.ID,
						Method:   action.Method,
						Code:     -1,
						Data:     fmt.Sprintf("panic recovered: %v", r),
						callback: callback,
					}).send()
				}
			}
		}()

		dispatched := getDispatcher().Dispatch(action)
		resp := dispatched.Response
		result := ActionResult{
			ID:       resp.ID,
			Method:   resp.Method,
			Data:     resp.Data,
			Code:     resp.Code,
			callback: callback,
		}
		result.send()
		sent = true
		if dispatched.AfterSend != nil {
			dispatched.AfterSend()
		}
	}(action, callback)
}

// setEventListener sets the message event listener callback (logs, delay, etc).
// When replaced, the old callback is released after in-flight calls complete.
//
//export setEventListener
func setEventListener(listener unsafe.Pointer) {
	eventListenerSlot.Store(listener)
}

// sendMessage sends a contract.Message to the current listener callback.
func sendMessage(message contract.Message) {
	ref := eventListenerSlot.Acquire()
	if ref == nil {
		return
	}
	result := ActionResult{
		Method:   contract.MessageMethod,
		Data:     message,
		callback: ref.ptr,
	}
	result.send()
	eventListenerSlot.Release(ref)
}

// suspend notifies the core to enter suspended/resumed state.
//
//export suspend
func suspend(suspended bool) {
	_ = getService().Suspend(suspended)
}

// forceGC triggers a GC cycle and tries to return memory to the OS.
//
//export forceGC
func forceGC() {
	getService().ForceGC()
}

// updateDns requests updating system DNS (cmfa builds only).
//
//export updateDns
func updateDns(valueChar *C.char) {
	getService().UpdateDns(takeCString(valueChar))
}

// getTraffic returns a JSON string snapshot of current upload/download traffic.
//
//export getTraffic
func getTraffic(onlyStatisticsProxy bool) *C.char {
	return C.CString(getService().GetTraffic(onlyStatisticsProxy))
}

// getTotalTraffic returns a JSON string snapshot of total upload/download traffic.
//
//export getTotalTraffic
func getTotalTraffic(onlyStatisticsProxy bool) *C.char {
	return C.CString(getService().GetTotalTraffic(onlyStatisticsProxy))
}
