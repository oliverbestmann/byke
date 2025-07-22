package spoke

type HashValue uint32

type isComponentMarker struct{}

// ErasedComponent holds a pointer to a value
// that implements the IsComponent interface.
type ErasedComponent interface {
	ComponentType() *ComponentType
	isComponent(isComponentMarker)
}

// ErasedComponentValue directly holds a value that implements IsComponent.
// As such, the value will never be a pointer.
type ErasedComponentValue interface {
	ErasedComponent
}
