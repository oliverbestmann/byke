package arch

type IsComponent[T any] interface {
	ErasedComponentValue
	IsComponent(T)
}

type IsComparableComponent[T comparable] interface {
	IsComponent[T]
	comparable
}

type Component[C IsComponent[C]] struct{}

func (Component[C]) IsComponent(C) {}

func (Component[C]) isComponent(isComponentMarker) {}

func (Component[C]) ComponentType() *ComponentType {
	return ComponentTypeOf[C]()
}

type ComparableComponent[T IsComparableComponent[T]] struct{}

func (ComparableComponent[T]) IsComponent(t T) {}

func (ComparableComponent[T]) ComponentType() *ComponentType {
	return ComparableComponentTypeOf[T]()
}

func (ComparableComponent[T]) isComponent(isComponentMarker) {}
