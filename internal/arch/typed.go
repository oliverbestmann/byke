package arch

type IsComponent[T any] interface {
	ErasedComponent
	IsComponent(T)
}

type IsComparableComponent[T comparable] interface {
	IsComponent[T]
	isComparableComponent
	comparable
}

type Component[C IsComponent[C]] struct{}

func (Component[C]) IsComponent(C) {}

func (Component[C]) isComponent(isComponentMarker) {}

func (Component[C]) ComponentType() *ComponentType {
	return nonComparableComponentTypeOf[C]()
}

type ComparableComponent[T IsComparableComponent[T]] struct{}

func (ComparableComponent[T]) IsComponent(t T) {}

func (ComparableComponent[T]) ComponentType() *ComponentType {
	return comparableComponentTypeOf[T]()
}

func (ComparableComponent[T]) isComponent(isComponentMarker) {}

func (ComparableComponent[T]) isComparableComponent(isComparableComponentMarker) {}

type isComparableComponentMarker struct{}

type isComparableComponent interface {
	isComparableComponent(isComparableComponentMarker)
}
