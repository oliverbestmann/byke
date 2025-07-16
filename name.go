package byke

import "github.com/oliverbestmann/byke/internal/arch"

var _ = ValidateComponent[Name]()

// Name assigns a non unique name to an entity.
// Adding a name can be helpful for debugging.
type Name struct {
	arch.Component[Name]
	Name string
}

// Named creates a new Name component.
func Named(name string) Name {
	return Name{Name: name}
}

func (n Name) String() string {
	return n.Name
}
