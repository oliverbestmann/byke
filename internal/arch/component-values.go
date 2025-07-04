package arch

type ComponentValue struct {
	Type    *ComponentType
	Added   Tick
	Changed Tick
	Hash    HashValue
	Value   ErasedComponent
}

type TypedComponentValue[C IsComponent[C]] struct {
	Added   Tick
	Changed Tick
	Hash    HashValue
	Value   C
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
