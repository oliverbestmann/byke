package spoke

import (
	"unsafe"
)

type ColumnAccess struct {
	base   unsafe.Pointer
	stride uintptr
}

func (c *ColumnAccess) At(row Row) unsafe.Pointer {
	return unsafe.Add(c.base, c.stride*uintptr(row))
}
