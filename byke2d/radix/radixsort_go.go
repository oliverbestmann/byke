//go:build !cgo

package radix

func doSort(values, scratch []Value) {
	radixsortGo(values, scratch)
}
