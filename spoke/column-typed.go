package spoke

import (
	"fmt"
	"unsafe"
)

type TypedColumn[C IsComponent[C]] struct {
	ChangeTracker
	values []C
	onGrow func()
}

func NewTypedColumn[C IsComponent[C]]() Column {
	return &TypedColumn[C]{}
}

func (c *TypedColumn[C]) Append(tick Tick, component ErasedComponent) {
	grew := c.appendValue(c.toValue(component))
	c.ChangeTracker.append(tick, tick)

	if grew && c.onGrow != nil {
		c.onGrow()
	}
}

func (c *TypedColumn[C]) Update(tick Tick, row Row, component ErasedComponent) {
	c.values[row] = c.toValue(component)
	c.ChangeTracker.markChanged(row, tick)
}

func (c *TypedColumn[C]) Get(row Row) ErasedComponent {
	return any(&c.values[row]).(ErasedComponent)
}

func (c *TypedColumn[C]) Copy(from Row, to Row) {
	c.values[to] = c.values[from]
	c.ChangeTracker.copy(to, from)
}

func (c *TypedColumn[C]) Truncate(n Row) {
	clear(c.values[n:])
	c.values = c.values[:n]
	c.ChangeTracker.truncate(n)
}

func (c *TypedColumn[C]) Access() ColumnAccess {
	return ColumnAccess{
		base:   unsafe.Pointer(unsafe.SliceData(c.values)),
		stride: unsafe.Sizeof(c.values[0]),
	}
}

func (c *TypedColumn[C]) Import(other Column, row Row) {
	realloc := c.appendValue(c.toValue(other.Get(row)))
	c.ChangeTracker.append(other.Added(row), other.Changed(row))

	if realloc && c.onGrow != nil {
		c.onGrow()
	}
}

func (c *TypedColumn[C]) Len() int {
	return len(c.values)
}

func (c *TypedColumn[C]) CheckChanged(Tick) {
	// not a comparable component, not doing anything here
}

func (c *TypedColumn[C]) OnGrow(onGrow func()) {
	c.onGrow = onGrow
}

func (c *TypedColumn[C]) toValue(component ErasedComponent) C {
	switch value := any(component).(type) {
	case C:
		return value
	case *C:
		return *value
	default:
		var c C
		var ptrC C
		panic(fmt.Errorf("got type %T, expected either %T or %T", component, c, ptrC))
	}
}

func (c *TypedColumn[C]) appendValue(value C) (realloc bool) {
	before := unsafe.SliceData(c.values)
	c.values = append(c.values, value)

	// realloc occurred if the pointer data has changed
	after := unsafe.SliceData(c.values)
	return before != after
}

type changeTick struct {
	Added   Tick
	Changed Tick
}

type ChangeTracker struct {
	ticks       []changeTick
	lastAdded   Tick
	lastChanged Tick
}

func (c *ChangeTracker) markChanged(row Row, tick Tick) {
	c.ticks[row].Changed = tick
	c.lastChanged = max(c.lastChanged, tick)
}

func (c *ChangeTracker) copy(to, from Row) {
	c.ticks[to] = c.ticks[from]
}

func (c *ChangeTracker) append(added, changed Tick) {
	c.ticks = append(c.ticks, changeTick{
		Added:   added,
		Changed: changed,
	})

	c.lastAdded = max(c.lastAdded, added)
	c.lastChanged = max(c.lastChanged, changed)
}

func (c *ChangeTracker) truncate(n Row) {
	c.ticks = c.ticks[:n]
}

func (c *ChangeTracker) Added(row Row) Tick {
	return c.ticks[row].Added
}

func (c *ChangeTracker) Changed(row Row) Tick {
	return c.ticks[row].Changed
}

func (c *ChangeTracker) LastAdded() Tick {
	return c.lastAdded
}

func (c *ChangeTracker) LastChanged() Tick {
	return c.lastChanged
}
