package byke

import (
	"fmt"
	"hash/maphash"
	"reflect"
	"strconv"
)

var seed = maphash.MakeSeed()

type AnyPtr any

type Resource[T any] *T

type IsComponent[T any] interface {
	AnyComponent
	IsComponent(T)
}

type isAnyComponentMarker struct{}

type AnyComponent interface {
	isAnyComponent(isAnyComponentMarker)
	ComponentType() ComponentType
}

type RequireComponents interface {
	RequireComponents() []AnyComponent
}

type Component[C IsComponent[C]] struct{}

func (Component[C]) isAnyComponent(isAnyComponentMarker) {}

func (Component[C]) IsComponent(C) {}

func (c Component[C]) ComponentType() ComponentType {
	return componentTypeOf[C]()
}

type IsComparableComponent[T comparable] interface {
	IsComponent[T]
	comparable
}

type ComparableComponent[T IsComparableComponent[T]] struct {
	Component[T]
}

func (c ComparableComponent[T]) hashOf(value AnyComponent) HashValue {
	ptrToValue := any(value).(*T)
	hash := maphash.Comparable(seed, *ptrToValue)
	return HashValue(hash)
}

type erasedComparableComponent interface {
	hashOf(value AnyComponent) HashValue
}

type Tick uint64

type HashValue uint64

type EntityId uint32

func (e EntityId) String() string {
	return strconv.Itoa(int(e))
}

type Entity struct {
	Id EntityId

	// pointer to components
	Components map[ComponentType]componentValue
}

type componentValue struct {
	PtrToValue  AnyComponent
	LastChanged Tick
	Hash        HashValue
}

func (cv *componentValue) Type() ComponentType {
	return cv.PtrToValue.ComponentType()
}

func (cv *componentValue) CheckUpdate() {
	if !cv.Type().Comparable() {
		return
	}

}

// ptrValue is a wrapper around a reflect.Value that holds
// some value of type reflect.Pointer
type ptrValue struct {
	reflect.Value
}

func ptrValueOf(val reflect.Value) ptrValue {
	assertIsPointerType(val.Type())
	return ptrValue{Value: val}
}

type ComponentType struct {
	reflect.Type
}

func componentTypeOf[C IsComponent[C]]() ComponentType {
	var componentInstance C
	return anyComponentTypeOf(componentInstance)
}

func anyComponentTypeOf(component AnyComponent) ComponentType {
	return ComponentType{
		Type: reflect.TypeOf(component),
	}
}

type resourceValue struct {
	Reflect ptrValue
	Pointer AnyPtr
}

type ScheduleId interface {
}

type Schedule struct {
	// make this a non zero sized type, so that creating a
	// new Schedule will always return a different memory address
	_nonZero uint32
}

type System any

type preparedSystem struct {
	LastRun Tick

	// A function that executes the system against
	// the given world
	Run func()
}

type World struct {
	entityIdSeq EntityId
	entities    map[EntityId]*Entity
	resources   map[reflect.Type]resourceValue
	schedules   map[ScheduleId][]*preparedSystem
	currentTick Tick
}

func NewWorld() *World {
	return &World{
		entities:  map[EntityId]*Entity{},
		resources: map[reflect.Type]resourceValue{},
		schedules: map[ScheduleId][]*preparedSystem{},
	}
}

func (w *World) AddSystems(scheduleId ScheduleId, firstSystem System, systems ...System) {
	preparedSystem := prepareSystem(w, firstSystem)
	w.schedules[scheduleId] = append(w.schedules[scheduleId], preparedSystem)

	for _, system := range systems {
		preparedSystem := prepareSystem(w, system)
		w.schedules[scheduleId] = append(w.schedules[scheduleId], preparedSystem)
	}
}

func (w *World) RunSchedule(scheduleId ScheduleId) {
	for _, system := range w.schedules[scheduleId] {
		w.runSystem(system)
	}
}

func (w *World) ReserveEntityId() EntityId {
	w.entityIdSeq += 1
	entityId := w.entityIdSeq

	return entityId

}

func (w *World) Spawn(entityId EntityId, components []AnyComponent) {
	entity := &Entity{
		Id:         entityId,
		Components: map[ComponentType]componentValue{},
	}

	if w.entities[entity.Id] != nil {
		panic(fmt.Sprintf("entity with id %d already exists", entity.Id))
	}

	w.entities[entity.Id] = entity

	w.insertComponents(entity, components)
}

func (w *World) insertComponents(entity *Entity, components []AnyComponent) {
	queue := append([]AnyComponent(nil), components...)

	for idx := 0; idx < len(queue); idx++ {
		// if in question we'll overwrite the components if they
		// where specified directly
		overwrite := idx < len(components)

		component := queue[idx]
		tyComponent := anyComponentTypeOf(component)

		// maybe skip this one if it already exists on the entity
		if _, exists := entity.Components[tyComponent]; exists && !overwrite {
			continue
		}

		// must not be inserted if it is a parentComponent
		if _, ok := component.(parentComponent); ok {
			panic(fmt.Sprintf(
				"you may not insert a byke.ParentComponent yourself: %C", component,
			))
		}

		// and add it to the entity
		heapComponent := copyToHeap(component).Interface().(AnyComponent)
		entity.Components[tyComponent] = componentValue{
			PtrToValue: heapComponent,
			Hash:       hashOf(heapComponent),
		}

		// trigger hooks for this component
		w.onComponentInsert(entity, heapComponent)

		// enqueue all required components
		if req, ok := heapComponent.(RequireComponents); ok {
			queue = append(queue, req.RequireComponents()...)
		}
	}
}

func (w *World) onComponentInsert(entity *Entity, component AnyComponent) {
	// update relations if needed
	if parentComponent, ok := w.parentComponentOf(component); ok {
		parentComponent.addChild(entity.Id)
	}
}

func (w *World) onComponentRemoved(entity *Entity, component AnyComponent) {
	// update relations if needed
	if parentComponent, ok := w.parentComponentOf(component); ok {
		parentComponent.removeChild(entity.Id)
	}
}

func (w *World) parentComponentOf(component AnyComponent) (parentComponent, bool) {
	child, ok := component.(childComponent)
	if !ok {
		return nil, false
	}

	parentId := child.parentId()
	parent, ok := w.entities[parentId]
	if !ok {
		panic(fmt.Sprintf("parent entity %s does not exist", parentId))
	}

	parentType := child.RelationParentType()
	parentComponentValue, ok := parent.Components[parentType]
	if !ok {
		parentComponentValue = componentValue{
			PtrToValue: ptrValueOf(reflect.New(parentType.Type)).Interface().(AnyComponent),
		}

		parent.Components[parentType] = parentComponentValue
	}

	parentComponent := parentComponentValue.PtrToValue.(parentComponent)
	return parentComponent, true
}

func (w *World) NewCommands() *Commands {
	return &Commands{world: w}
}

// Resource returns a pointer to the resource of the given reflect type.
// The type must be the non-pointer type of the resource
func (w *World) Resource(ty reflect.Type) (AnyPtr, bool) {
	resValue, ok := w.resources[reflect.PointerTo(ty)]
	if !ok {
		return nil, false
	}

	return resValue.Pointer, true
}

func ResourceOf[T any](w *World) (*T, bool) {
	value, ok := w.Resource(reflect.TypeFor[T]())
	if !ok {
		return nil, false
	}

	return value.(*T), true
}

func copyToHeap(value any) ptrValue {
	if reflect.TypeOf(value).Kind() == reflect.Pointer {
		panic("we do not want to have double pointers")
	}

	// move the component onto the heap
	ptrToValue := reflect.New(reflect.TypeOf(value))
	ptrToValue.Elem().Set(reflect.ValueOf(value))
	return ptrValueOf(ptrToValue)
}

func ValidateComponent[C IsComponent[C]]() struct{} {
	componentType := componentTypeOf[C]()

	var zero C

	if parent, ok := any(zero).(parentComponent); ok {
		// check if the child type points to us
		childType := parent.RelationChildType()
		instance := reflect.New(childType.Type).Elem().Interface()

		child, ok := instance.(childComponent)
		if !ok {
			panic(fmt.Sprintf(
				"relationship target of %s must point to a component embedding byke.ChildComponent",
				componentType,
			))
		}

		if child.RelationParentType() != componentType {
			panic(fmt.Sprintf(
				"relationship target of %s must point to %s",
				childType, componentType,
			))
		}
	}

	if child, ok := any(zero).(childComponent); ok {
		// check if the child type points to us
		parentType := child.RelationParentType()
		instance := reflect.New(parentType.Type).Interface()

		parent, ok := instance.(parentComponent)
		if !ok {
			panic(fmt.Sprintf(
				"relationship target of %s must point to a component embedding byke.ParentComponent",
				componentType,
			))
		}

		if parent.RelationChildType() != componentType {
			panic(fmt.Sprintf(
				"relationship target of %s must point to %s",
				parentType, componentType,
			))
		}
	}

	// TODO mark component as valid somewhere, maybe calculate some
	//  kind of component type id too
	return struct{}{}
}

func reflectComponentTypeOf(tyComponent reflect.Type) ComponentType {
	assertIsComponentType(tyComponent)

	return ComponentType{
		Type: tyComponent,
	}
}

func (w *World) InsertResource(resource any) {
	resType := reflect.PointerTo(reflect.TypeOf(resource))

	if existing, ok := w.resources[resType]; ok {
		// update existing value
		existing.Reflect.Elem().Set(reflect.ValueOf(resource))
		return
	}

	// create a new pointer to the resource type
	ptr := copyToHeap(resource)

	w.resources[ptr.Type()] = resourceValue{
		Reflect: ptr,
		Pointer: ptr.Interface(),
	}
}

func (w *World) RunSystem(system System) {
	if ps, ok := system.(*preparedSystem); ok {
		w.runSystem(ps)
		return
	}

	// prepare and execute directly
	w.runSystem(prepareSystem(w, system))
}

func (w *World) runSystem(system *preparedSystem) {
	w.currentTick += 1

	system.Run()

	// update last run so we can calculate changed components
	// at the next run
	system.LastRun = w.currentTick
}

func (w *World) Despawn(entityId EntityId) {
	queue := []EntityId{entityId}

	for idx := 0; idx < len(queue); idx++ {
		entityId = queue[idx]

		entity, ok := w.entities[entityId]
		if !ok {
			fmt.Printf("[warn] cannot despawn entity %d: does not exist\n", entityId)
			return
		}

		// update relationships
		for _, component := range entity.Components {
			value := component.PtrToValue
			if parentComponent, ok := w.parentComponentOf(value); ok {
				parentComponent.removeChild(entityId)
			}

			// despawn child entities too
			if parentComponent, ok := value.(parentComponent); ok {
				for _, entityId := range parentComponent.Children() {
					queue = append(queue, entityId)
				}
			}
		}
	}

	for _, entityId := range queue {
		delete(w.entities, entityId)
	}
}

func (w *World) removeComponent(entity *Entity, componentType ComponentType) {
	component, ok := entity.Components[componentType]
	if !ok {
		// component is already gone
		return
	}

	w.onComponentRemoved(entity, component.PtrToValue)

	// remove component
	delete(entity.Components, componentType)
}

func (w *World) recheckComponents(ty ComponentType) {
	isComparable := ty.Implements(reflect.TypeFor[erasedComparableComponent]())
	if !isComparable {
		return
	}

	// TODO optimize, do not walk through all entities
	for _, entity := range w.entities {
		component, ok := entity.Components[ty]
		if !ok {
			continue
		}

		hashValue := hashOf(component.PtrToValue)

		if hashValue != component.Hash {
			component.Hash = hashValue
			component.LastChanged = w.currentTick
			entity.Components[ty] = component
		}
	}
}

func hashOf(component AnyComponent) HashValue {
	erasedComparable, ok := component.(erasedComparableComponent)
	if !ok {
		return 1
	}

	return erasedComparable.hashOf(component)
}

type systemParameter struct {
	// If constant is set, it will be used directly. GetValue will not be called
	Constant reflect.Value

	// GetValue gets the value from somewhere
	GetValue func() reflect.Value

	// Cleanup is an optional function that will be called with the value provided
	// by GetValue or Constant after the system was finished it's work.
	Cleanup func(value reflect.Value)
}

func valueSystemParameter(value reflect.Value) systemParameter {
	return systemParameter{Constant: value}
}

func commandsSystemParameter(world *World) systemParameter {
	return systemParameter{
		GetValue: func() reflect.Value {
			return reflect.ValueOf(&Commands{world: world})
		},

		Cleanup: func(value reflect.Value) {
			commands := value.Interface().(*Commands)
			commands.applyToWorld()
		},
	}
}

func querySystemParameter(world *World, query reflect.Value) systemParameter {
	return systemParameter{
		Constant: query,

		Cleanup: func(value reflect.Value) {
			q := value.Addr().Interface().(queryAccessor)
			for _, ty := range q.get().parsed.mutableComponentTypes {
				world.recheckComponents(ty)
			}
		},
	}
}

func prepareSystem(w *World, system System) *preparedSystem {
	rSystem := reflect.ValueOf(system)

	if rSystem.Kind() != reflect.Func {
		panic(fmt.Sprintf("not a function: %s", rSystem.Type()))
	}

	tySystem := rSystem.Type()

	// collect a number of functions that when called will prepare the systems parameters
	var params []systemParameter

	preparedSystem := &preparedSystem{}

	for idx := range tySystem.NumIn() {
		tyIn := tySystem.In(idx)

		resourceCopy, resourceCopyOk := w.resources[reflect.PointerTo(tyIn)]
		resource, resourceOk := w.resources[tyIn]

		switch {
		case reflect.PointerTo(tyIn).Implements(reflect.TypeFor[queryAccessor]()):
			params = append(params, querySystemParameter(w, buildQuery(w, preparedSystem, tyIn)))

		case tyIn == reflect.TypeFor[*World]():
			params = append(params, valueSystemParameter(reflect.ValueOf(w)))

		case tyIn == reflect.TypeFor[*Commands]():
			params = append(params, commandsSystemParameter(w))

		case resourceCopyOk:
			params = append(params, valueSystemParameter(resourceCopy.Reflect.Elem()))

		case resourceOk:
			params = append(params, valueSystemParameter(resource.Reflect.Value))

		default:
			panic(fmt.Sprintf("Can not handle system param of type %s", tyIn))
		}
	}

	var paramValues []reflect.Value

	preparedSystem.Run = func() {
		paramValues = paramValues[:0]

		for _, param := range params {
			switch {
			case param.Constant.IsValid():
				paramValues = append(paramValues, param.Constant)

			case param.GetValue != nil:
				paramValues = append(paramValues, param.GetValue())

			default:
				panic("systemParameter not valid")
			}
		}

		rSystem.Call(paramValues)

		for idx, param := range params {
			if cleanup := param.Cleanup; cleanup != nil {
				cleanup(paramValues[idx])
			}
		}
	}

	return preparedSystem
}

type Name string

func (n Name) isAnyComponent(isAnyComponentMarker) {}

func (n Name) IsComponent(Name) {}

func (n Name) ComponentType() ComponentType {
	return Component[Name]{}.ComponentType()
}
