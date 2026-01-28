//go:build android && cgo

package main

import (
	"encoding/json"
	"unsafe"

	"mihomo_android_wrapper/contract"
)

type ActionResult struct {
	ID       string          `json:"id"`
	Method   contract.Method `json:"method"`
	Data     any             `json:"data"`
	Code     int             `json:"code"`
	callback unsafe.Pointer
}

// send marshals to JSON and calls back into the host. Non-message methods release the callback after sending.
func (r *ActionResult) send() {
	data, err := json.Marshal(r)
	if err != nil {
		// Even if r.Data fails to marshal, keep the response shape stable.
		fallback := struct {
			ID     string          `json:"id"`
			Method contract.Method `json:"method"`
			Data   any             `json:"data"`
			Code   int             `json:"code"`
		}{
			ID:     r.ID,
			Method: r.Method,
			Data:   err.Error(),
			Code:   -1,
		}
		if data2, err2 := json.Marshal(fallback); err2 == nil {
			data = data2
		} else {
			invokeResult(r.callback, err.Error())
			if r.Method != contract.MessageMethod {
				releaseObject(r.callback)
			}
			return
		}
	}

	invokeResult(r.callback, string(data))
	if r.Method != contract.MessageMethod {
		releaseObject(r.callback)
	}
}
