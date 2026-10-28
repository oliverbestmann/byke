//go:build tinygo

package byke

import (
	"sync"
	"unsafe"
)

// TinyGo represents a func value as {context, function pointer} and boxes a
// fresh copy each time it is converted to an interface. Intern the pair to get
// a stable id.
var funcvals sync.Map // [2]unsafe.Pointer -> *[2]unsafe.Pointer

// funcvalOf returns a stable pointer identifying the func value stored
// in an interface, given the interface's data pointer.
func funcvalOf(data unsafe.Pointer) unsafe.Pointer {
	key := *(*[2]unsafe.Pointer)(data)
	id, _ := funcvals.LoadOrStore(key, &key)
	return unsafe.Pointer(id.(*[2]unsafe.Pointer))
}
