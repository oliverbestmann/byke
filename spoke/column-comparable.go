package spoke

import (
	"bytes"
	"unsafe"
)

type ShadowComparableColumn[C IsComparableComponent[C]] struct {
	TypedColumn[C]
	shadow ShadowColumn
}

func NewShadowComparableColumn[C IsComparableComponent[C]]() Column {
	var cZero C
	return &ShadowComparableColumn[C]{
		shadow: ShadowColumn{
			ItemSize: unsafe.Sizeof(cZero),
		},
	}
}

func (c *ShadowComparableColumn[C]) Append(tick Tick, component ErasedComponent) {
	c.TypedColumn.Append(tick, component)
	c.shadow.Append(c.unsafeLastValue())
}

func (c *ShadowComparableColumn[C]) Import(column Column, source Row) {
	c.TypedColumn.Import(column, source)
	c.shadow.Append(c.unsafeValue(Row(c.Len() - 1)))
}

func (c *ShadowComparableColumn[C]) Update(tick Tick, row Row, component ErasedComponent) {
	c.TypedColumn.Update(tick, row, component)
	c.shadow.Update(row, c.unsafeValue(row), 1)
}

func (c *ShadowComparableColumn[C]) Copy(from, to Row) {
	c.TypedColumn.Copy(from, to)
	c.shadow.Update(to, c.unsafeValue(from), 1)
}

func (c *ShadowComparableColumn[C]) Truncate(n Row) {
	c.TypedColumn.Truncate(n)
	c.shadow.Truncate(n)
}

func (c *ShadowComparableColumn[C]) CheckChanged(tick Tick) {
	if c.Len() == 0 {
		// no need to check an empty column
		return
	}

	// no need to check if we do not have any items
	itemSize := unsafe.Sizeof(c.values[0])
	if itemSize == 0 {
		panic("itemSize must not be zero")
	}

	// view of the current data as a byte slice
	slice := unsafe.Slice((*byte)(c.unsafeValue(0)), uintptr(len(c.values))*itemSize)

	// keep track of the range of changes
	var firstChange Row
	var copyCount uintptr

	rowCount := Row(len(c.values))

	for row := Row(0); row < rowCount; row++ {
		// skip to the next changed row
		next, more := c.shadow.CompareWith(slice, row)
		if !more {
			break
		}

		// continue next iteration at the next row
		row = next

		c.markChanged(row, tick)

		if copyCount == 0 {
			firstChange = row
		}

		// number of elements to copy
		copyCount = uintptr(row - firstChange + 1)
	}

	if copyCount > 0 {
		c.shadow.Update(firstChange, c.unsafeValue(firstChange), copyCount)
	}

	if debug {
		if !bytes.Equal(slice, c.shadow.buf) {
			panic("more changes detected")
		}
	}
}

func (c *ShadowComparableColumn[C]) unsafeValue(row Row) unsafe.Pointer {
	return unsafe.Pointer(&c.values[row])
}

func (c *ShadowComparableColumn[C]) unsafeLastValue() unsafe.Pointer {
	return c.unsafeValue(Row(c.Len() - 1))
}
