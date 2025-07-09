package arch

type ComponentValue struct {
	Type    *ComponentType
	Value   ErasedComponent
	Hash    HashValue
	Added   Tick
	Changed Tick
}

type TypedComponentValue[C IsComponent[C]] struct {
	Value   C
	Hash    HashValue
	Added   Tick
	Changed Tick
}

type ComponentValues []ComponentValue

func (values ComponentValues) ByType(ty *ComponentType) (value *ComponentValue, ok bool) {
	for idx := range values {
		if values[idx].Type == ty {
			return &values[idx], true
		}
	}

	return nil, false
}
