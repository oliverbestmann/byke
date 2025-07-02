package arch

type ComponentValue struct {
	Added   Tick
	Changed Tick
	Hash    HashValue
	Value   ErasedComponent
}

func (c ComponentValue) ComponentType() *ComponentType {
	return c.Value.ComponentType()
}

type TypedComponentValue[C IsComponent[C]] struct {
	Added   Tick
	Changed Tick
	Hash    HashValue
	Value   C
}

func (t *TypedComponentValue[C]) ToComponentValue() ComponentValue {
	return ComponentValue{
		Added:   t.Added,
		Changed: t.Changed,
		Hash:    t.Hash,
		Value:   any(&t.Value).(ErasedComponent),
	}
}

type ComponentValues []ComponentValue

func (values ComponentValues) ByType(ty *ComponentType) (value *ComponentValue, ok bool) {
	for idx := range values {
		if values[idx].ComponentType() == ty {
			return &values[idx], true
		}
	}

	return nil, false
}
