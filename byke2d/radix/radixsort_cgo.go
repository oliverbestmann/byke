//go:build cgo && !radixsort_go

package radix

import (
	"unsafe"

	"github.com/oliverbestmann/byke/byke2d/radix/radix_c"
)

func doSort(values, scratch []Value) {
	ptrValues := unsafe.SliceData(values)
	ptrScratch := unsafe.SliceData(scratch)

	cValues := unsafe.Slice((*radix_c.Value)(ptrValues), len(values))
	cScratch := unsafe.Slice((*radix_c.Value)(ptrScratch), len(values))

	radix_c.Sort(cValues, cScratch)
}
