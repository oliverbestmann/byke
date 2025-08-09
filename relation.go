package byke

import (
	"slices"

	"github.com/oliverbestmann/byke/spoke"
)

var _ = ValidateComponent[Children]()
var _ = ValidateComponent[ChildOf]()

type IsRelationshipTargetComponent[T IsImmutableComponent[T]] interface {
	IsImmutableComponent[T]
	isRelationshipTargetType
}

type isRelationshipTargetType interface {
	ErasedComponent
	RelationshipType() *spoke.ComponentType
	Children() []EntityId
	addChild(id EntityId)
	removeChild(id EntityId)
}

type IsRelationshipComponent[T IsImmutableComponent[T]] interface {
	IsImmutableComponent[T]
	isRelationshipComponent
}

type isRelationshipComponent interface {
	ErasedComponent
	RelationshipTargetType() *spoke.ComponentType
	RelationshipEntityId() EntityId
}

// RelationshipTarget must be embedded on the parent side of a relationship
type RelationshipTarget[Child IsImmutableComponent[Child]] struct {
	_children []EntityId
}

func (*RelationshipTarget[Child]) RelationshipType() *spoke.ComponentType {
	return spoke.ComponentTypeOf[Child]()
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

func (Relationship[Parent]) RelationshipTargetType() *spoke.ComponentType {
	return spoke.ComponentTypeOf[Parent]()
}

type ChildOf struct {
	ImmutableComponent[ChildOf]
	Relationship[Children]
	Parent EntityId
}

func (c ChildOf) RelationshipEntityId() EntityId {
	return c.Parent
}

type Children struct {
	ImmutableComponent[Children]
	RelationshipTarget[ChildOf]
}
