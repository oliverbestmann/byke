package spoke

import "unsafe"

type ZeroSizedColumn[T IsComponent[T]] struct {
	dummyValue T
	erased     ErasedComponent
	lastAdded  Tick
	added      []Tick
}

func NewZeroSizedColumn[T IsComponent[T]]() Column {
	return &ZeroSizedColumn[T]{
		erased:    any(new(T)).(ErasedComponent),
		lastAdded: NoTick,
	}
}

func (c *ZeroSizedColumn[T]) Append(tick Tick, component ErasedComponent) {
	c.added = append(c.added, tick)
	c.lastAdded = tick
}

func (c *ZeroSizedColumn[T]) Update(tick Tick, row Row, component ErasedComponent) {
	// zero values do not change
}

func (c *ZeroSizedColumn[T]) Get(row Row) ErasedComponent {
	return c.erased
}

func (c *ZeroSizedColumn[T]) Copy(from, to Row) {
	c.added[to] = c.added[from]
}

func (c *ZeroSizedColumn[T]) Truncate(n Row) {
	c.added = c.added[:n]
}

func (c *ZeroSizedColumn[T]) Access() ColumnAccess {
	return ColumnAccess{
		base:   unsafe.Pointer(&c.dummyValue),
		stride: 0,
	}
}

func (c *ZeroSizedColumn[T]) Import(column Column, source Row) {
	added := column.Added(source)
	c.lastAdded = max(c.lastAdded, added)
	c.added = append(c.added, added)
}

func (c *ZeroSizedColumn[T]) Added(row Row) Tick {
	return c.added[row]
}

func (c *ZeroSizedColumn[T]) LastAdded() Tick {
	return c.lastAdded
}

func (c *ZeroSizedColumn[T]) Changed(row Row) Tick {
	return c.added[row]
}

func (c *ZeroSizedColumn[T]) LastChanged() Tick {
	return c.lastAdded
}

func (c *ZeroSizedColumn[T]) Len() int {
	return len(c.added)
}

func (c *ZeroSizedColumn[T]) CheckChanged(tick Tick) {
	// no zero sized component will ever change
}

func (c *ZeroSizedColumn[T]) OnGrow(onGrow func()) {
	// memory will never change, so we dont need to trigger onGrow callbacks
}
