//go:build cgo

package main

import "C"

// main is required by cgo builds, but this module is used as a shared library.
func main() {
}
