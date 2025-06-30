package byke

import "slices"

// ParentComponent must be embedded on the parent side of a relationship
type ParentComponent[Parent IsComponent[Parent], Child IsComponent[Child]] struct {
	Component[Parent]

	// Children holds the EntityIds of the children of this relation.
	// You **must not** update the Children slice manually.
	Children []EntityId
}

func (*ParentComponent[Parent, Child]) RelationChildType() ComponentType {
	return componentTypeOf[Child]()
}

func (*ParentComponent[Parent, Child]) makeChildComponent() childComponent {
	var value Child
	return any(value).(childComponent)
}

func (p *ParentComponent[Parent, Child]) addChild(childId EntityId) {
	if slices.Contains(p.Children, childId) {
		return
	}

	p.Children = append(p.Children, childId)
}

func (p *ParentComponent[Parent, Child]) removeChild(childId EntityId) {
	p.Children = slices.DeleteFunc(p.Children, func(id EntityId) bool {
		return childId == id
	})
}

func (p *ParentComponent[Parent, Child]) children() []EntityId {
	return p.Children
}

// ChildComponent must be embedded on the client side of a relationship
type ChildComponent[Parent IsComponent[Parent], Child IsComponent[Child]] struct {
	Component[Child]
	Parent EntityId
}

func (ChildComponent[Parent, Child]) RelationParentType() ComponentType {
	return componentTypeOf[Parent]()
}

func (c ChildComponent[Parent, Child]) parentId() EntityId {
	return c.Parent
}

type parentComponent interface {
	AnyComponent
	RelationChildType() ComponentType
	addChild(id EntityId)
	removeChild(id EntityId)
	children() []EntityId
}

type childComponent interface {
	AnyComponent
	RelationParentType() ComponentType
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
