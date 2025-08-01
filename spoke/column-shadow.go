package spoke

import "unsafe"

type ShadowColumn struct {
	ItemSize uintptr
	buf      []byte
}

func (c *ShadowColumn) Append(ptrToValue unsafe.Pointer) {
	idx := Row(uintptr(len(c.buf)) / c.ItemSize)

	// this is supposed to not allocate the temporary slice
	c.buf = append(c.buf, make([]byte, c.ItemSize)...)
	rawCopy(c.ptrTo(idx), ptrToValue, c.ItemSize)
}

func (c *ShadowColumn) Copy(to, from Row) {
	// this is supposed to not allocate the temporary slice
	c.buf = append(c.buf, make([]byte, c.ItemSize)...)
	rawCopy(c.ptrTo(to), c.ptrTo(from), c.ItemSize)
}

func (c *ShadowColumn) Update(row Row, ptrToValue unsafe.Pointer, count uintptr) {
	rawCopy(c.ptrTo(row), ptrToValue, c.ItemSize*count)
}

func (c *ShadowColumn) Truncate(n Row) {
	c.buf = c.buf[:uintptr(n)*c.ItemSize]
}

func (c *ShadowColumn) CompareWith(bufCurrent []byte, startAt Row) (Row, bool) {
	item := uintptr(startAt)

	// start comparing from the current item
	offset := sliceCompare(c.buf, bufCurrent, item*c.ItemSize)
	if offset == noOffset {
		return 0, false
	}

	// calculate item of the first change
	return Row(offset / c.ItemSize), true
}

func (c *ShadowColumn) ptrTo(idx Row) unsafe.Pointer {
	return unsafe.Pointer(&c.buf[uintptr(idx)*c.ItemSize])
}

func rawCopy(to, from unsafe.Pointer, size uintptr) {
	dst := (*buf(to))[:size]
	src := (*buf(from))[:size]
	copy(dst, src)
}
