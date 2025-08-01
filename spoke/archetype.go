package spoke

import (
	"encoding/binary"
	"fmt"
	"github.com/oliverbestmann/byke/internal/set"
	"hash/maphash"
	"slices"
	"strings"
	"unsafe"
)

type ArchetypeId uint64

type Archetype struct {
	Id    ArchetypeId
	Types []*ComponentType

	entities []EntityId
	index    map[EntityId]Row

	columns       []Column
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

func (a *Archetype) ReplaceComponentValue(tick Tick, entityId EntityId, component ErasedComponent) ErasedComponent {
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

	return column.Get(row)
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
		// move entity from rowSwap to row
		a.entities[row] = a.entities[rowSwap]

		// replace column value at row with rowSwap
		for _, column := range a.columns {
			column.Copy(rowSwap, row)
		}

		// update the index, point to row instead of rowSwap
		a.index[a.entities[row]] = row
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

	return a.getAt(row), true
}

func (a *Archetype) componentAt(row Row, componentType *ComponentType) ErasedComponent {
	column := a.getColumn(componentType)
	if column == nil {
		return nil
	}

	return column.Get(row)
}

func (a *Archetype) changedAt(row Row, componentType *ComponentType) Tick {
	column := a.getColumn(componentType)
	if column == nil {
		return NoTick
	}

	return column.Changed(row)
}

func (a *Archetype) addedAt(row Row, componentType *ComponentType) Tick {
	column := a.getColumn(componentType)
	if column == nil {
		return NoTick
	}

	return column.Added(row)
}

func (a *Archetype) LastAdded(componentType *ComponentType) Tick {
	column := a.getColumn(componentType)
	if column == nil {
		return NoTick
	}

	return column.LastAdded()
}

func (a *Archetype) LastChanged(componentType *ComponentType) Tick {
	column := a.getColumn(componentType)
	if column == nil {
		return NoTick
	}

	return column.LastChanged()
}

func (a *Archetype) getColumn(componentType *ComponentType) Column {
	return a.columnsByType[componentType]
}

func (a *Archetype) getAt(row Row) EntityRef {
	columns := make([]ColumnAccess, len(a.columns))

	for idx, column := range a.columns {
		columns[idx] = column.Access()
	}

	return EntityRef{
		fetch:     unsafeSlice(columns),
		archetype: a,
		row:       row,
	}
}

func (a *Archetype) IterForQuery(query *Query, fetches []ColumnAccess) (EntityIter, []ColumnAccess) {
	fetch := fetches[:0]

	// check what types we can fetch
	for _, typ := range query.Fetch {
		column := a.getColumn(typ.ComponentType)
		fetch = append(fetch, columnAccessOf(column))
	}

	// now we have columns in the same order as the queries fetch values.
	iter := EntityIter{
		Fetch:     fetch,
		entities:  a.entities,
		total:     len(a.entities),
		archetype: a,
	}

	return iter, fetch
}

func columnAccessOf(column Column) ColumnAccess {
	if column == nil {
		return ColumnAccess{}
	} else {
		return column.Access()
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
	if !debug {
		return
	}

	entityCount := len(a.entities)

	for idx, column := range a.columns {
		if column.Len() != entityCount {
			panic(fmt.Sprintf("%s: expected %d values in column %s, got %d", a, entityCount, a.Types[idx], column.Len()))
		}
	}

	if len(a.index) != entityCount {
		panic("entity count/index mismatch")
	}

	for row, entityId := range a.entities {
		rowIndex, ok := a.index[entityId]
		if !ok {
			panic("entity not in index")
		}

		if rowIndex != Row(row) {
			panic("entity index out of sync")
		}
	}

	// double check in reverse
	for entityId, row := range a.index {
		if int(row) >= len(a.entities) {
			panic("entity count/index mismatch")
		}

		if a.entities[row] != entityId {
			panic("entity index of of sync")
		}
	}
}

func (a *Archetype) GetComponent(entityId EntityId, componentType *ComponentType) ErasedComponent {
	row, ok := a.index[entityId]
	if !ok {
		return nil
	}

	column := a.getColumn(componentType)
	if column == nil {
		return nil
	}

	return column.Get(row)
}

func (a *Archetype) componentsAt(row Row) []ErasedComponent {
	components := make([]ErasedComponent, len(a.columns))
	for idx, column := range a.columns {
		components[idx] = column.Get(row)
	}

	return components
}

func (a *Archetype) Len() int {
	return len(a.entities)
}

type EntityIter struct {
	Fetch      []ColumnAccess
	archetype  *Archetype
	entities   []EntityId
	row, total int
}

func (iter *EntityIter) More() bool {
	if iter.row == iter.total {
		return false
	}

	iter.row += 1

	return true
}

func (iter *EntityIter) Current() EntityRef {
	return EntityRef{
		fetch:     unsafeSlice(iter.Fetch),
		archetype: iter.archetype,
		row:       Row(iter.row - 1),
	}
}

type EntityRef struct {
	fetch     fastSlice[ColumnAccess]
	archetype *Archetype
	row       Row
}

func (e *EntityRef) EntityId() EntityId {
	return e.archetype.entities[e.row]
}

func (e *EntityRef) GetAt(idx int) unsafe.Pointer {
	return e.fetch.Get(idx).At(e.row)
}

func (e *EntityRef) Get(ty *ComponentType) ErasedComponent {
	return e.archetype.componentAt(e.row, ty)
}

func (e *EntityRef) Changed(ty *ComponentType) Tick {
	return e.archetype.changedAt(e.row, ty)
}

func (e *EntityRef) Added(ty *ComponentType) Tick {
	return e.archetype.addedAt(e.row, ty)
}

func (e *EntityRef) Components() []ErasedComponent {
	return e.archetype.componentsAt(e.row)
}

type Archetypes struct {
	archetypes []*Archetype
	lookup     map[ArchetypeId]*Archetype
}

func (a *Archetypes) Lookup(types []*ComponentType) (*Archetype, bool) {
	id, sortedTypes := ArchetypeIdOf(types)

	at, ok := a.lookup[id]
	if ok {
		return at, false
	}

	if a.lookup == nil {
		a.lookup = map[ArchetypeId]*Archetype{}
	}

	at = makeArchetype(id, slices.Clone(sortedTypes))
	a.lookup[id] = at
	a.archetypes = append(a.archetypes, at)

	return at, true
}

func (a *Archetypes) All() []*Archetype {
	return a.archetypes
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

type fastSlice[T any] struct {
	values *T
}

func unsafeSlice[T any](slice []T) fastSlice[T] {
	values := unsafe.SliceData(slice)
	return fastSlice[T]{values: values}
}

func (s fastSlice[T]) Get(idx int) *T {
	var tZero T
	return (*T)(unsafe.Add(unsafe.Pointer(s.values), uintptr(idx)*unsafe.Sizeof(tZero)))
}
