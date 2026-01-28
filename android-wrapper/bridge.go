//go:build android && cgo

package main

//#include "bridge.h"
import "C"
import "unsafe"

// protectSocket calls host-side protect_socket to prevent the socket from being captured by Android VPN.
// Equivalent to Android VpnService#protect.
func protectSocket(tunCtx unsafe.Pointer, fd int) {
	C.protect_socket(tunCtx, C.int(fd))
}

// releaseObject releases a host-owned handle (for example, a callback pointer).
func releaseObject(obj unsafe.Pointer) {
	C.release_object(obj)
}

// invokeResult sends a JSON result (or an error message) to the host callback.
// The host must not keep the C string pointer after the callback returns.
func invokeResult(callback unsafe.Pointer, data string) {
	s := C.CString(data)
	defer C.free(unsafe.Pointer(s))
	C.invoke_result(callback, s)
}

// takeCString converts a host C string to Go string and frees it via free_string.
func takeCString(s *C.char) string {
	if s == nil {
		return ""
	}
	defer C.free_string(s)
	return C.GoString(s)
}
