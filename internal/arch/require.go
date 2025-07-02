package arch

type RequireComponents interface {
	RequireComponents() []ErasedComponentValue
}
