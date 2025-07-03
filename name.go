package byke

import "github.com/oliverbestmann/byke/internal/arch"

var _ = ValidateComponent[Name]()

type Name struct {
	arch.Component[Name]
	Name string
}

func (n Name) String() string {
	return n.Name
}

func Named(name string) Name {
	return Name{Name: name}
}
