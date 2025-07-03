package arch

import "hash/maphash"

type Row uint32

type Column interface {
	Append(tick Tick, component ErasedComponent)
	Copy(from, to Row)
	Truncate(n Row)
	Get(row Row) ComponentValue
	Update(tick Tick, row Row, cv ErasedComponent)
	Import(other Column, row Row)
	CheckChanged(tick Tick)
	Len() int
}

type TypedColumn[C IsComponent[C]] struct {
	ComponentType *ComponentType
	Values        []TypedComponentValue[C]
}

type ComparableTypedColumn[C IsComparableComponent[C]] struct {
	TypedColumn[C]
}

func MakeColumnOf[C IsComponent[C]](componentType *ComponentType) MakeColumn {
	return func() Column {
		return &TypedColumn[C]{
			ComponentType: componentType,
		}
	}
}

func MakeComparableColumnOf[C IsComparableComponent[C]](componentType *ComponentType) MakeColumn {
	return func() Column {
		return &ComparableTypedColumn[C]{
			TypedColumn: TypedColumn[C]{
				ComponentType: componentType,
			},
		}
	}
}

func (c *TypedColumn[C]) Len() int {
	return len(c.Values)
}

func (c *TypedColumn[C]) Append(tick Tick, component ErasedComponent) {
	value := any(component).(*C)

	c.Values = append(c.Values, TypedComponentValue[C]{
		Value:   *value,
		Added:   tick,
		Changed: tick,
	})
}

func (c *ComparableTypedColumn[C]) Append(tick Tick, component ErasedComponent) {
	value := any(component).(*C)

	c.Values = append(c.Values, TypedComponentValue[C]{
		Value:   *value,
		Added:   tick,
		Changed: tick,
		Hash:    hashOf(value),
	})
}

func (c *TypedColumn[C]) Copy(from, to Row) {
	c.Values[to] = c.Values[from]
}

func (c *TypedColumn[C]) Import(other Column, row Row) {
	otherColumn := other.(*TypedColumn[C])
	c.Values = append(c.Values, otherColumn.Values[row])
}

func (c *ComparableTypedColumn[C]) Import(other Column, row Row) {
	otherColumn := other.(*ComparableTypedColumn[C])
	c.Values = append(c.Values, otherColumn.Values[row])
}

func (c *TypedColumn[C]) Truncate(n Row) {
	clear(c.Values[n:])
	c.Values = c.Values[:n]
}

func (c *TypedColumn[C]) Get(row Row) ComponentValue {
	return c.Values[row].ToComponentValue(c.ComponentType)
}

func (c *TypedColumn[C]) Update(tick Tick, row Row, erasedValue ErasedComponent) {
	target := &c.Values[row]
	target.Value = erasedValue.(C)
	target.Changed = tick
}

func (c *ComparableTypedColumn[C]) Update(tick Tick, row Row, erasedValue ErasedComponent) {
	target := &c.Values[row]
	target.Value = erasedValue.(C)

	if hash := hashOf(&target.Value); hash != target.Hash {
		target.Hash = hash
		target.Changed = tick
	}
}

func (c *TypedColumn[C]) CheckChanged(tick Tick) {
	panic("not supported")
}

func (c *ComparableTypedColumn[C]) CheckChanged(tick Tick) {
	for idx := range c.Values {
		cv := &c.Values[idx]

		if hash := hashOf(&cv.Value); hash != cv.Hash {
			cv.Hash = hash
			cv.Changed = tick
		}
	}
}

func hashOf[C IsComparableComponent[C]](value *C) HashValue {
	return HashValue(maphash.Comparable[C](seed, *value))
}
