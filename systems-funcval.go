//go:build !tinygo

package byke

import "unsafe"

// funcvalOf returns a stable pointer identifying the func value stored
// in an interface, given the interface's data pointer.
func funcvalOf(data unsafe.Pointer) unsafe.Pointer {
	// gc stores the pointer to the funcval directly in the interface.
	return data
}
