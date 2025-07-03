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
	_, exists := a.columnsByType[componentType]
	return exists
}

func (a *Archetype) Insert(tick Tick, entityId EntityId, components []ErasedComponent) {
	defer a.assertInvariants()

	if _, exists := a.index[entityId]; exists {
		panic(fmt.Sprintf("archetype %s already contains entity %s", a, entityId))
	}

	// must have the correct number of components
	if len(components) != len(a.Types) {
		panic(fmt.Sprintf("archetype component types do not match"))
	}

	// add value of each component to the columns
	for _, component := range components {
		componentType := component.ComponentType()

		// get the target component
		column, ok := a.columnsByType[componentType]
		if !ok {
			panic(fmt.Sprintf("unexpected component of type %s", a))
		}

		// and add it to the correct column
		column.Append(tick, component)
	}

	// add the entity
	a.addEntity(entityId)
}

func (a *Archetype) addEntity(entityId EntityId) {
	// put entity into index
	idx := len(a.entities)
	a.index[entityId] = Row(idx)

	// add entity
	a.entities = append(a.entities, entityId)
}

func (a *Archetype) ReplaceComponentValue(tick Tick, entityId EntityId, component ErasedComponent) {
	defer a.assertInvariants()

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
	column.Update(tick, row, component)
}

func (a *Archetype) Remove(entityId EntityId) {
	defer a.assertInvariants()

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

func (a *Archetype) Import(tick Tick, source *Archetype, entityId EntityId, newComponents ...ErasedComponent) {
	target := a

	defer target.assertInvariants()

	rowInSource, exists := source.index[entityId]
	if !exists {
		panic("entity does not exist")
	}

	// go through the columns we have and import them into the target
	for idx, sourceColumn := range source.columns {
		ty := source.Types[idx]

		targetColumn, ok := target.columnsByType[ty]
		if !ok {
			// looks like this component type is removed during the transfer
			continue
		}

		// import source
		targetColumn.Import(sourceColumn, rowInSource)
	}

	// now add the new components
	for _, component := range newComponents {
		componentType := component.ComponentType()
		targetColumn, ok := target.columnsByType[componentType]
		if !ok {
			panic(fmt.Sprintf("target column does not exist: %s", componentType))
		}

		// add it to the column
		targetColumn.Append(tick, component)
	}

	// add the entity to the index
	target.addEntity(entityId)
}

func (a *Archetype) CheckChanged(tick Tick, componentType *ComponentType) {
	column, ok := a.columnsByType[componentType]
	if !ok {
		panic(fmt.Sprintf("type not in archetype: %s", componentType))
	}

	column.CheckChanged(tick)
}

func (a *Archetype) assertInvariants() {
	entityCount := len(a.entities)

	for idx, column := range a.columns {
		if column.Len() != entityCount {
			panic(fmt.Sprintf("%s: expected %d values in column %s, got %d", a, entityCount, a.Types[idx], column.Len()))
		}
	}

	for row, entityId := range a.entities {
		if a.index[entityId] != Row(row) {
			panic("entity index out of sync")
		}
	}
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

func (e EntityRef) Get(ty *ComponentType) (*ComponentValue, bool) {
	for idx := range e.Components {
		if e.Components[idx].Type == ty {
			return &e.Components[idx], true
		}
	}

	return nil, false
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
