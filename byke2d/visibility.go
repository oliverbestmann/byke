package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
)

var (
	_ = byke.ValidateComponent[Visibility]()
	_ = byke.ValidateComponent[ComputedVisibility]()
)

type visibility uint8

const (
	visibilityInherit   = 0
	visibilityVisible   = 1
	visibilityInvisible = 2
)

var (
	InheritVisibility = Visibility{value: visibilityInherit}
	Visible           = Visibility{value: visibilityVisible}
	Invisible         = Visibility{value: visibilityInvisible}
)

type Visibility struct {
	byke.ComparableComponent[Visibility]
	value visibility
}

func (*Visibility) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{ComputedVisibility{}}
}

func (v *Visibility) Compute(parentVisibility ComputedVisibility) ComputedVisibility {
	if v.value == visibilityInherit {
		return parentVisibility
	}

	return ComputedVisibility{Visible: v.value == visibilityVisible}
}

func (v *Visibility) SetVisible() {
	v.value = visibilityVisible
}

func (v *Visibility) SetInvisible() {
	v.value = visibilityInvisible
}

func (v *Visibility) SetInherit() {
	v.value = visibilityInherit
}

type ComputedVisibility struct {
	byke.Component[ComputedVisibility]
	Visible bool
}
