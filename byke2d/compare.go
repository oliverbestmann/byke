package byke2d

import (
	"cmp"
	"reflect"
	"unsafe"
)

type CompareTo interface {
	CompareTo(other any) int
}

func compareByAddress[T any](lhs, rhs *T) int {
	lhsAddr := uintptr(unsafe.Pointer(lhs))
	rhsAddr := uintptr(unsafe.Pointer(rhs))

	return cmp.Compare(lhsAddr, rhsAddr)
}

func compareType(lhs, rhs reflect.Type) int {
	lhsAddr := reflect.ValueOf(lhs).Pointer()
	rhsAddr := reflect.ValueOf(rhs).Pointer()
	return cmp.Compare(lhsAddr, rhsAddr)
}

func compareByType(lhs, rhs any) int {
	lhsType := reflect.TypeOf(lhs)
	rhsType := reflect.TypeOf(rhs)
	return compareType(lhsType, rhsType)
}
