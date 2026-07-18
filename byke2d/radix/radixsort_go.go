//go:build !cgo || radixsort_go

package radix

func doSort(values, scratch []Value) {
	radixsortGo(values, scratch)
}
