package spoke

type HashedComparableColumn[C IsComparableComponent[C]] struct {
	TypedColumn[C]
	hashes []HashValue
}

func NewHashedComparableColumn[C IsComparableComponent[C]]() Column {
	return &HashedComparableColumn[C]{}
}

func (c *HashedComparableColumn[C]) Append(tick Tick, component ErasedComponent) {
	c.TypedColumn.Append(tick, component)
	c.hashes = append(c.hashes, c.hashOf(Row(c.Len()-1)))
}

func (c *HashedComparableColumn[C]) Import(column Column, source Row) {
	c.TypedColumn.Import(column, source)
	c.hashes = append(c.hashes, c.hashOf(Row(c.Len()-1)))
}

func (c *HashedComparableColumn[C]) Update(tick Tick, row Row, component ErasedComponent) {
	c.TypedColumn.Update(tick, row, component)
	c.hashes = append(c.hashes, c.hashOf(row))
}

func (c *HashedComparableColumn[C]) Copy(from, to Row) {
	c.TypedColumn.Copy(from, to)
	c.hashes[to] = c.hashes[from]
}

func (c *HashedComparableColumn[C]) Truncate(n Row) {
	c.TypedColumn.Truncate(n)
	c.hashes = c.hashes[:n]
}

func (c *HashedComparableColumn[C]) CheckChanged(tick Tick) {
	if c.Len() == 0 {
		// no need to check an empty column
		return
	}

	hashes := c.hashes
	values := c.values

	// eliminate bound checks
	_ = hashes[len(c.values)-1]

	for row := range values {
		hash := maphashOf(&values[row])
		if hashes[row] != hash {
			hashes[row] = hash
			c.ChangeTracker.markChanged(Row(row), tick)
		}
	}
}

func (c *HashedComparableColumn[C]) hashOf(row Row) HashValue {
	return maphashOf(&c.values[row])
}
