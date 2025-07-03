package byke

import (
	"github.com/oliverbestmann/byke/internal/arch"
	"slices"
)

// ParentComponent must be embedded on the parent side of a relationship
type ParentComponent[Parent IsComponent[Parent], Child IsComparableComponent[Child]] struct {
	Component[Parent]

	// _children holds the EntityIds of the children of this relation.
	_children []EntityId
}

func (ParentComponent[Parent, Child]) isParentComponent(component markerIsParentComponent) {}

type markerIsParentComponent interface {
	isParentComponent(markerIsParentComponent)
}

func (*ParentComponent[Parent, Child]) RelationChildType() *arch.ComponentType {
	return arch.ComponentTypeOf[Child]()
}

func (*ParentComponent[Parent, Child]) makeChildComponent() childComponent {
	var value Child
	return any(value).(childComponent)
}

func (p *ParentComponent[Parent, Child]) addChild(childId EntityId) {
	if slices.Contains(p._children, childId) {
		return
	}

	p._children = append(p._children, childId)
}

func (p *ParentComponent[Parent, Child]) removeChild(childId EntityId) {
	idx := slices.Index(p._children, childId)
	if idx >= 0 {
		p._children = slices.Delete(p._children, idx, idx+1)
	}
}

// Children returns the children in this component.
// You **must not** modify the returned slice.
func (p *ParentComponent[Parent, Child]) Children() []EntityId {
	return p._children
}

// ChildComponent must be embedded on the client side of a relationship
type ChildComponent[Parent IsComponent[Parent], Child IsComparableComponent[Child]] struct {
	ComparableComponent[Child]
	Parent EntityId
}

func (ChildComponent[Parent, Child]) RelationParentType() *arch.ComponentType {
	return arch.ComponentTypeOf[Parent]()
}

func (c ChildComponent[Parent, Child]) parentId() EntityId {
	return c.Parent
}

type parentComponent interface {
	ErasedComponent
	RelationChildType() *arch.ComponentType
	addChild(id EntityId)
	removeChild(id EntityId)
	Children() []EntityId
}

type childComponent interface {
	ErasedComponent
	RelationParentType() *arch.ComponentType
	parentId() EntityId
}

type ChildOf struct {
	ChildComponent[Children, ChildOf]
}

type Children struct {
	ParentComponent[Children, ChildOf]
}

var _ = ValidateComponent[Children]()
var _ = ValidateComponent[ChildOf]()
