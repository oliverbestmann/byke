package byke

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
