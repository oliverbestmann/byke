package arch

import (
	"encoding/binary"
	"fmt"
	"github.com/oliverbestmann/byke/internal/set"
	"hash/maphash"
	"slices"
	"strings"
)

// all Archetype
var archetypes = map[ArchetypeId]*Archetype{}

type ArchetypeId uint64

type Archetype struct {
	Id    ArchetypeId
	Types []*ComponentType

	entities []EntityId
	columns  []Column
	index    map[EntityId]Row

	columnsByType map[*ComponentType]Column
}

func makeArchetype(id ArchetypeId, sortedTypes []*ComponentType) *Archetype {
	// check that we do not have any duplicates in the types
	var seen set.Set[*ComponentType]
	for _, ty := range sortedTypes {
		if !seen.Insert(ty) {
			panic(fmt.Sprintf("archetype contains duplicate: %s", ty))
		}
	}

	columnsByType := map[*ComponentType]Column{}

	var columns []Column
	for _, ty := range sortedTypes {
		column := ty.MakeColumn()
		columns = append(columns, column)

		// put column into index too
		columnsByType[ty] = column
	}

	return &Archetype{
		Id:            id,
		Types:         sortedTypes,
		entities:      nil,
		columns:       columns,
		columnsByType: columnsByType,
		index:         map[EntityId]Row{},
	}
}

func (a *Archetype) String() string {
	var value strings.Builder

	value.WriteString("Archetype(")
	for idx, ty := range a.Types {
		if idx > 0 {
			value.WriteString(", ")
		}

		value.WriteString(ty.String())
	}

	value.WriteString(")")

	return value.String()
}

func (a *Archetype) ContainsType(componentType *ComponentType) bool {
	for _, ty := range a.Types {
		if ty == componentType {
			return true
		}
	}

	return false
}

func (a *Archetype) Insert(entityId EntityId, components []ComponentValue) {
	if _, exists := a.index[entityId]; exists {
		panic(fmt.Sprintf("archetype %s already contains entity %s", a, entityId))
	}

	// must have the correct number of components
	if len(components) != len(a.Types) {
		panic(fmt.Sprintf("archetype component types do not match"))
	}

	// put entity into index
	idx := len(a.entities)
	a.index[entityId] = Row(idx)

	// add entity
	a.entities = append(a.entities, entityId)

	// add value of each component to the columns
	for _, component := range components {
		componentType := component.ComponentType()

		// get the target component
		column, ok := a.columnsByType[componentType]
		if !ok {
			panic(fmt.Sprintf("unexpected component of type %s", a))
		}

		// and add it to the correct column
		column.Append(component)
	}
}

func (a *Archetype) ReplaceComponentValue(entityId EntityId, component ComponentValue) {
	row, exists := a.index[entityId]
	if !exists {
		panic(fmt.Sprintf("entity %s does not exist", entityId))
	}

	// get the target column
	componentType := component.ComponentType()
	column, ok := a.columnsByType[componentType]
	if !ok {
		panic(fmt.Sprintf("unexpected component of type %s", a))
	}

	// update the existing value
	column.Update(row, component)
}

func (a *Archetype) Remove(entityId EntityId) {
	row, exists := a.index[entityId]
	if !exists {
		panic(fmt.Sprintf("entity %s does not exist", entityId))
	}

	// remove from index
	delete(a.index, entityId)

	// to remove a value, we move the last element into the
	// spot of the one to remove
	rowSwap := Row(len(a.entities) - 1)

	if row != rowSwap {
		// replace entityId at rowSwap
		a.entities[rowSwap] = a.entities[row]

		// replace column value at rowSwap
		for _, column := range a.columns {
			column.Copy(rowSwap, row)
		}
	}

	// now truncate columns & entities
	a.entities = a.entities[:rowSwap]
	for _, column := range a.columns {
		column.Truncate(rowSwap)
	}
}

func (a *Archetype) Get(entityId EntityId) (EntityRef, bool) {
	row, exists := a.index[entityId]
	if !exists {
		return EntityRef{}, false
	}

	return a.rowAt(row, nil), true
}

func (a *Archetype) rowAt(row Row, reuseComponents []ComponentValue) EntityRef {
	// TODO take slice to reuse as parameter?
	values := reuseComponents[:0]
	for _, column := range a.columns {
		value := column.Get(row)
		values = append(values, value)
	}

	return EntityRef{
		EntityId:   a.entities[row],
		Components: values,
	}
}

func (a *Archetype) Iter(scratch *[]ComponentValue) ArchetypeIter {
	return ArchetypeIter{
		archetype: a,
		scratch:   scratch,
	}
}

func (a *Archetype) TransferTo(target *Archetype, entityId EntityId, newComponents ...ComponentValue) {
	entity, ok := a.Get(entityId)
	if !ok {
		panic("entity does not exist")
	}

	// add the new components if any
	components := append(entity.Components, newComponents...)

	// remove all components we don't care about anymore
	components = slices.DeleteFunc(components, func(value ComponentValue) bool {
		return !target.ContainsType(value.ComponentType())
	})

	// remove it from the previous archetype
	a.Remove(entityId)

	// and insert into target
	target.Insert(entityId, components)
}

type ArchetypeIter struct {
	archetype *Archetype
	scratch   *[]ComponentValue
	row       Row
}

func (iter *ArchetypeIter) More() bool {
	return int(iter.row) < len(iter.archetype.entities)
}

func (iter *ArchetypeIter) Next() EntityRef {
	entity := iter.archetype.rowAt(iter.row, (*iter.scratch)[:0])
	iter.row += 1
	*iter.scratch = entity.Components[:0]
	return entity
}

type EntityRef struct {
	EntityId   EntityId
	Components []ComponentValue
}

func LookupArchetype(types []*ComponentType) *Archetype {
	id, sortedTypes := ArchetypeIdOf(types)

	at, ok := archetypes[id]
	if !ok {
		at = makeArchetype(id, slices.Clone(sortedTypes))
		archetypes[id] = at
	}

	return at
}

var typesScratch []*ComponentType

// ArchetypeIdOf returns the ArchetypeId for the given ComponentType slice.
// The return value sortedTypes contains the provided types in a deterministic order.
// The returned slice will be reused at the next call of ArchetypeIdOf and must not be kept around.
func ArchetypeIdOf(types []*ComponentType) (id ArchetypeId, sortedTypes []*ComponentType) {
	// clone the types into our scratch buffer for soring
	types = append(typesScratch[:0], types...)

	// sort slices by id to have a deterministic ordering
	slices.SortFunc(types, compareComponentTypes)

	// hash the types to have an id
	return ArchetypeId(hashTypes(types)), types
}

func hashTypes(types []*ComponentType) HashValue {
	var hash maphash.Hash

	hash.SetSeed(seed)

	for _, ty := range types {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(ty.Id))
		_, _ = hash.Write(buf[:])
	}

	return HashValue(hash.Sum64())
}

func compareComponentTypes(lhs, rhs *ComponentType) int {
	return int(lhs.Id - rhs.Id)
}
