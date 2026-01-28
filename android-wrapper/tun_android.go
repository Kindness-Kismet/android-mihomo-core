//go:build android && cgo

package main

/*
#include <stdbool.h>
*/
import "C"

import (
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/metacubex/mihomo/component/dialer"
	"github.com/metacubex/mihomo/constant"
	LC "github.com/metacubex/mihomo/listener/config"
	"github.com/metacubex/mihomo/listener/sing_tun"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel"
)

var (
	tunMu            sync.Mutex
	tunListener      *sing_tun.Listener
	tunCallbackSlot  callbackSlot
	previousSockHook dialer.SocketControl
)

// buildTunConfig builds a mihomo TUN config from the Android VPN file descriptor.
func buildTunConfig(fd int, stack, address, dns string) (LC.Tun, error) {
	if fd <= 0 {
		return LC.Tun{}, errors.New("invalid TUN file descriptor")
	}

	tunStack, ok := constant.StackTypeMapping[strings.ToLower(stack)]
	if !ok {
		tunStack = constant.TunSystem
	}

	var prefix4 []netip.Prefix
	var prefix6 []netip.Prefix
	for _, a := range strings.Split(address, ",") {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(a)
		if err != nil {
			return LC.Tun{}, err
		}
		if prefix.Addr().Is4() {
			prefix4 = append(prefix4, prefix)
		} else {
			prefix6 = append(prefix6, prefix)
		}
	}

	var dnsHijack []string
	for _, d := range strings.Split(dns, ",") {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		dnsHijack = append(dnsHijack, net.JoinHostPort(d, "53"))
	}

	return LC.Tun{
		Enable:              true,
		Device:              "Mihomo",
		Stack:               tunStack,
		DNSHijack:           dnsHijack,
		AutoRoute:           false,
		AutoDetectInterface: false,
		Inet4Address:        prefix4,
		Inet6Address:        prefix6,
		MTU:                 9000,
		FileDescriptor:      fd,
	}, nil
}

// stopTunLocked stops the TUN listener and restores the socket hook (requires tunMu).
func stopTunLocked() {
	if tunListener != nil {
		_ = tunListener.Close()
		tunListener = nil
	}

	dialer.DefaultSocketHook = previousSockHook
	previousSockHook = nil

	tunCallbackSlot.Store(nil)
}

// startTunLocked starts TUN and installs the socket hook (requires tunMu).
func startTunLocked(callback unsafe.Pointer, fd int, stack, address, dns string) error {
	stopTunLocked()

	tunConf, err := buildTunConfig(fd, stack, address, dns)
	if err != nil {
		if callback != nil {
			releaseObject(callback)
		}
		return err
	}

	previousSockHook = dialer.DefaultSocketHook
	tunCallbackSlot.Store(callback)

	dialer.DefaultSocketHook = func(network, address string, conn syscall.RawConn) error {
		ref := tunCallbackSlot.Acquire()
		if ref == nil {
			return nil
		}
		err := conn.Control(func(fd uintptr) {
			protectSocket(ref.ptr, int(fd))
		})
		tunCallbackSlot.Release(ref)
		return err
	}

	listener, err := sing_tun.New(tunConf, tunnel.Tunnel)
	if err != nil {
		stopTunLocked()
		return err
	}

	tunListener = listener
	log.Infoln("[TUN] started: %s", tunListener.Address())
	return nil
}

// startTUN starts TUN using the Android VPN fd; callback is used to call protect_socket.
//
//export startTUN
func startTUN(callback unsafe.Pointer, fd C.int, stackChar, addressChar, dnsChar *C.char) bool {
	stack := takeCString(stackChar)
	address := takeCString(addressChar)
	dns := takeCString(dnsChar)

	tunMu.Lock()
	defer tunMu.Unlock()

	if err := startTunLocked(callback, int(fd), stack, address, dns); err != nil {
		log.Errorln("[TUN] start failed: %s", err.Error())
		return false
	}
	return true
}

// stopTun stops the current TUN and releases the host callback.
//
//export stopTun
func stopTun() {
	tunMu.Lock()
	defer tunMu.Unlock()
	stopTunLocked()
}
