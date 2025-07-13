package arch

type ComponentValue struct {
	Value   ErasedComponent
	Added   Tick
	Changed Tick
}

type TypedComponentValue[C IsComponent[C]] struct {
	Value   C
	Hash    HashValue
	Added   Tick
	Changed Tick
}
