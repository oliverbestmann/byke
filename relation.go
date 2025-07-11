package byke

import (
	"github.com/oliverbestmann/byke/internal/arch"
	"slices"
)

var _ = ValidateComponent[Children]()
var _ = ValidateComponent[ChildOf]()

type IsParentComponent[T IsComparableComponent[T]] interface {
	IsComparableComponent[T]
	isParentComponent
}

type isParentComponent interface {
	ErasedComponent
	RelationChildType() *arch.ComponentType
	Children() []EntityId
	addChild(id EntityId)
	removeChild(id EntityId)
}

type IsChildComponent[T IsComparableComponent[T]] interface {
	IsComparableComponent[T]
	isChildComponent
}

type isChildComponent interface {
	ErasedComponent
	RelationParentType() *arch.ComponentType
	ParentEntityId() EntityId
}

// RelationshipTarget must be embedded on the parent side of a relationship
type RelationshipTarget[Child IsImmutableComponent[Child]] struct {
	_children []EntityId
}

func (*RelationshipTarget[Child]) RelationChildType() *arch.ComponentType {
	return arch.ComponentTypeOf[Child]()
}

func (p *RelationshipTarget[Child]) addChild(childId EntityId) {
	p._children = append(p._children, childId)
}

func (p *RelationshipTarget[Child]) removeChild(childId EntityId) {
	idx := slices.Index(p._children, childId)
	if idx >= 0 {
		p._children = slices.Delete(p._children, idx, idx+1)
	}
}

// Children returns the children in this component.
// You **must not** modify the returned slice.
func (p *RelationshipTarget[Child]) Children() []EntityId {
	return p._children
}

// Relationship must be embedded on the client side of a relationship
type Relationship[Parent IsImmutableComponent[Parent]] struct{}

func (Relationship[Parent]) RelationParentType() *arch.ComponentType {
	return arch.ComponentTypeOf[Parent]()
}

type ChildOf struct {
	ImmutableComponent[ChildOf]
	Relationship[Children]
	Parent EntityId
}

func (c ChildOf) ParentEntityId() EntityId {
	return c.Parent
}

type Children struct {
	ImmutableComponent[Children]
	RelationshipTarget[ChildOf]
}
