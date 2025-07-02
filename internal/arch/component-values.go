package arch

type ComponentValues []ComponentValue

func (values ComponentValues) ByType(ty *ComponentType) (value *ComponentValue, ok bool) {
	for idx := range values {
		if values[idx].ComponentType() == ty {
			return &values[idx], true
		}
	}

	return nil, false
}
