package byke

import (
	"github.com/oliverbestmann/byke/internal/arch"
	"hash/maphash"
	"slices"
	"unsafe"
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

// ParentComponent must be embedded on the parent side of a relationship
type ParentComponent[Child IsComparableComponent[Child]] struct {
	_children EntitySet
}

func (*ParentComponent[Child]) RelationChildType() *arch.ComponentType {
	return arch.ComponentTypeOf[Child]()
}

func (p *ParentComponent[Child]) addChild(childId EntityId) {
	p._children.Insert(childId)
}

func (p *ParentComponent[Child]) removeChild(childId EntityId) {
	p._children.Remove(childId)
}

// Children returns the children in this component.
// You **must not** modify the returned slice.
func (p ParentComponent[Child]) Children() []EntityId {
	return p._children.Slice()
}

// ChildComponent must be embedded on the client side of a relationship
type ChildComponent[Parent IsComparableComponent[Parent]] struct{}

func (ChildComponent[Parent]) RelationParentType() *arch.ComponentType {
	return arch.ComponentTypeOf[Parent]()
}

type ChildOf struct {
	ComparableComponent[ChildOf]
	ChildComponent[Children]
	Parent EntityId
}

func (c ChildOf) ParentEntityId() EntityId {
	return c.Parent
}

type Children struct {
	ComparableComponent[Children]
	ParentComponent[ChildOf]
}

// EntitySet is a comparable set of EntityId values
type EntitySet struct {
	values *[]EntityId
	hash   arch.HashValue
}

var entitySetSeed = maphash.MakeSeed()

func (e *EntitySet) Insert(entityId EntityId) bool {
	// check if value is in the set
	if slices.Contains(e.Slice(), entityId) {
		return false
	}

	// add to the list
	e.update(append(e.Slice(), entityId))

	return true
}

func (e *EntitySet) Remove(entityId EntityId) bool {
	// check if value is in the set
	idx := slices.Index(e.Slice(), entityId)
	if idx == -1 {
		return false
	}

	e.update(slices.Delete(e.Slice(), idx, idx+1))

	return true
}

func (e *EntitySet) Slice() []EntityId {
	if e.values == nil {
		return nil
	}

	return *e.values
}

func (e *EntitySet) update(newValues []EntityId) {
	if e.values == nil {
		e.values = &newValues
	} else {
		*e.values = newValues
	}

	e.rehash()
}

func (e *EntitySet) rehash() {
	if len(*e.values) == 0 {
		e.hash = arch.HashValue(0)
		return
	}

	bytes := (*byte)(unsafe.Pointer(unsafe.SliceData(*e.values)))
	byteSlice := unsafe.Slice(bytes, len(*e.values)*int(unsafe.Sizeof(EntityId(0))))
	e.hash = arch.HashValue(maphash.Bytes(entitySetSeed, byteSlice))
}
