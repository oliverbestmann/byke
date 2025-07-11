package arch

type IsComponent[T any] interface {
	ErasedComponent
	IsComponent(T)
}

type IsComparableComponent[T IsComponent[T]] interface {
	IsComponent[T]
	IsSupportsChangeDetectionComponent[T]
	isErasedComparableComponent
	comparable
}

type IsImmutableComponent[T IsComponent[T]] interface {
	IsComponent[T]
	IsSupportsChangeDetectionComponent[T]
	isErasedImmutableComponent
}

type Component[C IsComponent[C]] struct{}

func (Component[C]) IsComponent(C) {}

func (Component[C]) isComponent(isComponentMarker) {}

func (Component[C]) ComponentType() *ComponentType {
	return nonComparableComponentTypeOf[C]()
}

type ImmutableComponent[C IsComponent[C]] struct{}

func (ImmutableComponent[C]) IsComponent(C) {}

func (ImmutableComponent[C]) isComponent(isComponentMarker) {}

func (ImmutableComponent[C]) ComponentType() *ComponentType {
	return nonComparableComponentTypeOf[C]()
}

func (ImmutableComponent[T]) supportsChangeDetection(componentMarkerType) {}

type ComparableComponent[T IsComparableComponent[T]] struct{}

func (ComparableComponent[T]) IsComponent(t T) {}

func (ComparableComponent[T]) ComponentType() *ComponentType {
	return comparableComponentTypeOf[T]()
}

func (ComparableComponent[T]) isComponent(isComponentMarker) {}

func (ComparableComponent[T]) isComparableComponent(componentMarkerType) {}

func (ComparableComponent[T]) supportsChangeDetection(componentMarkerType) {}

type componentMarkerType struct{}

type isErasedComparableComponent interface {
	isComparableComponent(componentMarkerType)
}

type isErasedImmutableComponent interface {
	isImmutableComponent(componentMarkerType)
}

type IsSupportsChangeDetectionComponent[C IsComponent[C]] interface {
	IsComponent[C]
	supportsChangeDetection(componentMarkerType)
}
