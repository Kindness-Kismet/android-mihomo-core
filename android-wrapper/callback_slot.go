//go:build android && cgo

package main

import (
	"sync"
	"unsafe"
)

type callbackRef struct {
	ptr     unsafe.Pointer
	refs    int
	retired bool
}

// callbackSlot stores a host pointer and uses ref-counting to avoid early release under concurrency.
// The host must provide release_object_func.
type callbackSlot struct {
	mu  sync.Mutex
	ref *callbackRef
}

// Store replaces the current pointer; the old pointer is released when its ref count drops to zero.
func (s *callbackSlot) Store(ptr unsafe.Pointer) {
	s.mu.Lock()
	old := s.ref
	if ptr == nil {
		s.ref = nil
	} else {
		s.ref = &callbackRef{ptr: ptr}
	}

	var release unsafe.Pointer
	if old != nil {
		old.retired = true
		if old.refs == 0 {
			release = old.ptr
		}
	}
	s.mu.Unlock()

	if release != nil {
		releaseObject(release)
	}
}

// Acquire returns the current pointer and increments its ref count; must be paired with Release.
func (s *callbackSlot) Acquire() *callbackRef {
	s.mu.Lock()
	ref := s.ref
	if ref != nil {
		ref.refs++
	}
	s.mu.Unlock()
	return ref
}

// Release decrements the ref count; if the pointer is retired and refs reaches 0, it is released.
func (s *callbackSlot) Release(ref *callbackRef) {
	if ref == nil {
		return
	}
	s.mu.Lock()
	ref.refs--

	var release unsafe.Pointer
	if ref.refs == 0 && ref.retired {
		release = ref.ptr
	}
	s.mu.Unlock()

	if release != nil {
		releaseObject(release)
	}
}
