package byke

// NoCopy can be embedded to provide "go vet" linting
// when a type should not - but is - be copied
type NoCopy struct{}

func (*NoCopy) Lock()   {}
func (*NoCopy) Unlock() {}
