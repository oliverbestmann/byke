package byke

// noCopy can be embedded to provide "go vet" linting
// when a type should not - but is - be copied
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
