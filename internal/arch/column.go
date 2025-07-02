package arch

type Row uint32

type Column interface {
	Append(component ComponentValue)
	Copy(from, to Row)
	Truncate(n Row)
	Get(row Row) ComponentValue
	Update(row Row, cv ComponentValue)
}

type ComponentValue struct {
	Added   uint64
	Changed uint64
	Hash    HashValue
	Value   ErasedComponent
}

func (c ComponentValue) ComponentType() *ComponentType {
	return c.Value.ComponentType()
}

type TypedComponentValue[C IsComponent[C]] struct {
	Added   uint64
	Changed uint64
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

type TypedColumn[C IsComponent[C]] struct {
	ComponentType *ComponentType
	Values        []TypedComponentValue[C]
}

func (c *TypedColumn[C]) Append(cv ComponentValue) {
	value := any(cv.Value).(*C)

	c.Values = append(c.Values, TypedComponentValue[C]{
		Value:   *value,
		Added:   cv.Added,
		Changed: cv.Changed,
		Hash:    cv.Hash,
	})
}

func (c *TypedColumn[C]) Copy(from, to Row) {
	c.Values[to] = c.Values[from]
}

func (c *TypedColumn[C]) Truncate(n Row) {
	clear(c.Values[n:])
	c.Values = c.Values[:n]
}

func (c *TypedColumn[C]) Get(row Row) ComponentValue {
	return c.Values[row].ToComponentValue()
}

func (c *TypedColumn[C]) Update(row Row, cv ComponentValue) {
	target := &c.Values[row]

	// copy the value
	value := any(cv.Value).(*C)

	target.Changed = cv.Changed
	target.Hash = cv.Hash
	target.Value = *value
}

func columnConstructorOf[C IsComponent[C]](componentType *ComponentType) MakeColumn {
	return func() Column {
		return &TypedColumn[C]{
			ComponentType: componentType,
		}
	}
}
