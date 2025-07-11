package arch

type ImmutableTypedColumn[C IsComponent[C]] struct {
	ComponentType *ComponentType
	Values        []ImmutableTypedComponentValue[C]
}

func MakeImmutableColumnOf[C IsComponent[C]](componentType *ComponentType) MakeColumn {
	return func() Column {
		return &ImmutableTypedColumn[C]{
			ComponentType: componentType,
		}
	}
}

type ImmutableTypedComponentValue[C IsComponent[C]] struct {
	Value   C
	Changed Tick
}

func (c *ImmutableTypedColumn[C]) Append(tick Tick, component ErasedComponent) {
	c.Values = append(c.Values, ImmutableTypedComponentValue[C]{
		Value:   *any(component).(*C),
		Changed: tick,
	})
}

func (c *ImmutableTypedColumn[C]) Copy(from, to Row) {
	c.Values[to] = c.Values[from]
}

func (c *ImmutableTypedColumn[C]) Truncate(n Row) {
	c.Values = c.Values[:n]
}

func (c *ImmutableTypedColumn[C]) Get(row Row) ComponentValue {
	value := &c.Values[row]

	return ComponentValue{
		Type:    c.ComponentType,
		Value:   any(&value.Value).(ErasedComponent),
		Added:   value.Changed,
		Changed: value.Changed,
	}
}

func (c *ImmutableTypedColumn[C]) Update(tick Tick, row Row, erasedValue ErasedComponent) {
	target := &c.Values[row]
	target.Value = erasedValue.(C)
	target.Changed = tick
}

func (c *ImmutableTypedColumn[C]) Import(other Column, row Row) {
	otherColumn := other.(*ImmutableTypedColumn[C])
	c.Values = append(c.Values, otherColumn.Values[row])
}

func (c *ImmutableTypedColumn[C]) CheckChanged(Tick) {
	// do nothing, all values are immutable
}

func (c *ImmutableTypedColumn[C]) Len() int {
	return len(c.Values)
}
