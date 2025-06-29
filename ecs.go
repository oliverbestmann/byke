package ecs

import (
	"fmt"
	"iter"
	"reflect"
)

type AnyPtr any

type PopulateTarget func(target reflect.Value, ptrToValues []pointerValue)

type PopulateSingleTarget func(target reflect.Value, ptrToValues pointerValue)

type Query[T any] struct {
	erasedQuery
}

type erasedQuery struct {
	values   iter.Seq[[]pointerValue]
	populate PopulateTarget
}

func (*Query[T]) __isQuery() {}

func (*Query[T]) reflectType() reflect.Type {
	return reflect.TypeFor[T]()
}

func (q *Query[T]) setInner(inner erasedQuery) {
	q.erasedQuery = inner
}

func (q *Query[T]) Get() (value *T, ok bool) {
	for value := range q.Items() {
		return &value, true
	}

	return nil, false
}

func (q *Query[T]) Count() int {
	var count int
	for range q.values {
		count += 1
	}

	return count
}

func (q *Query[T]) Items() iter.Seq[T] {
	return func(yield func(T) bool) {
		// is this safe?
		var target T

		for values := range q.values {
			q.populate(reflect.ValueOf(&target).Elem(), values)

			if !yield(target) {
				return
			}
		}
	}
}

type Resource[T any] *T

type IsComponent[T any] interface {
	AnyComponent
	IsComponent(T)
}

type isAnyComponentMarker struct{}

type AnyComponent interface {
	isAnyComponent(isAnyComponentMarker)
}

type RequireComponents interface {
	RequireComponents() []AnyComponent
}

type Component[T IsComponent[T]] struct{}

func (Component[T]) isAnyComponent(isAnyComponentMarker) {}

func (Component[T]) IsComponent(T) {}

type EntityId uint32

type Entity struct {
	Id EntityId

	// pointer to components
	Components map[ComponentType]componentValue
}

type componentValue struct {
	Type       ComponentType
	PtrToValue pointerValue
}

type pointerValue struct {
	reflect.Value
}

type ComponentType struct {
	GoType reflect.Type
}

func componentTypeOf[C IsComponent[C]]() ComponentType {
	var componentInstance C
	return anyComponentTypeOf(componentInstance)
}

func anyComponentTypeOf(component AnyComponent) ComponentType {
	return ComponentType{
		GoType: reflect.TypeOf(component),
	}
}

type resourceValue struct {
	Reflect pointerValue
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
	// A function that executes the system against
	// the given world
	Run func()
}

type World struct {
	entityIdSeq EntityId
	entities    map[EntityId]*Entity
	resources   map[reflect.Type]resourceValue
	schedules   map[ScheduleId][]preparedSystem
}

func NewWorld() World {
	return World{
		entities:  map[EntityId]*Entity{},
		resources: map[reflect.Type]resourceValue{},
		schedules: map[ScheduleId][]preparedSystem{},
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
		system.Run()
	}
}

func (w *World) ReserveEntityId() EntityId {
	entityId := w.entityIdSeq
	w.entityIdSeq += 1

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

		tyComponent := anyComponentTypeOf(queue[idx])

		// maybe skip this one if it already exists on the entity
		if _, exists := entity.Components[tyComponent]; exists && !overwrite {
			continue
		}

		// and add it to the entity
		entity.Components[tyComponent] = componentValue{
			Type:       tyComponent,
			PtrToValue: copyToHeap(queue[idx]),
		}

		// enqueue all required components
		if component, ok := queue[idx].(RequireComponents); ok {
			queue = append(queue, component.RequireComponents()...)
		}
	}
}

func (w *World) NewCommands() *Commands {
	return &Commands{world: w}
}

func (w *World) Exec(commands *Commands) {
	for _, command := range commands.queue {
		command(w)
	}
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

type Commands struct {
	world *World
	queue []Command
}

func (c *Commands) Spawn(components ...AnyComponent) EntityCommands {
	entityId := c.world.ReserveEntityId()

	c.queue = append(c.queue, func(world *World) {
		world.Spawn(entityId, components)
	})

	return EntityCommands{
		entityId: entityId,
		commands: c,
	}
}

func (c *Commands) Entity(entityId EntityId) EntityCommands {
	return EntityCommands{
		entityId: entityId,
		commands: c,
	}
}

func copyToHeap(value any) pointerValue {
	if reflect.TypeOf(value).Kind() == reflect.Pointer {
		panic("we do not want to have double pointers")
	}

	// move the component onto the heap
	ptrToValue := reflect.New(reflect.TypeOf(value))
	ptrToValue.Elem().Set(reflect.ValueOf(value))
	return pointerValue{Value: ptrToValue}
}

type EntityCommands struct {
	entityId EntityId
	commands *Commands
}

func (e EntityCommands) Update(commands ...EntityCommand) EntityCommands {
	e.commands.queue = append(e.commands.queue, func(world *World) {
		entity, ok := world.entities[e.entityId]
		if !ok {
			panic(fmt.Sprintf("entity %d does not exist", e.entityId))
		}

		for _, command := range commands {
			command(world, entity)
		}
	})

	return e
}

func (e EntityCommands) Despawn() {
	e.commands.queue = append(e.commands.queue, func(world *World) {
		if world.entities[e.entityId] == nil {
			fmt.Printf("[warn] cannot despawn entity %d: does not exist\n", e.entityId)
			return
		}

		delete(world.entities, e.entityId)
	})
}

func RemoveComponent[C IsComponent[C]]() EntityCommand {
	componentType := componentTypeOf[C]()

	return func(world *World, entity *Entity) {
		delete(entity.Components, componentType)
	}
}

func InsertComponent[C IsComponent[C]](maybeValue ...C) EntityCommand {
	if len(maybeValue) > 1 {
		panic("InsertComponent must be called with at most one argument")
	}

	var component C
	if len(maybeValue) == 1 {
		component = maybeValue[0]
	}

	return func(world *World, entity *Entity) {
		world.insertComponents(entity, []AnyComponent{component})
	}
}

func ValidateComponent[T IsComponent[T]]() struct{} {
	// TODO mark component as valid somewhere, maybe calculate some
	//  kind of component type id too
	return struct{}{}
}

type Command func(world *World)

type EntityCommand func(world *World, entity *Entity)

type Extractor func(entity *Entity) (pointerValue, bool)

func extractComponentByType(ty ComponentType) Extractor {
	return func(entity *Entity) (pointerValue, bool) {
		value, ok := entity.Components[ty]
		return value.PtrToValue, ok
	}
}

type parsedQueryTarget struct {
	extractors     []Extractor
	populateTarget PopulateTarget
}

type queryValueAccessor struct {
	extractor      Extractor
	populateTarget PopulateSingleTarget
}

func (w *World) queryValuesIter(extractors []Extractor) iter.Seq[[]pointerValue] {
	return func(yield func([]pointerValue) bool) {
		var values []pointerValue

	outer:
		for _, entity := range w.entities {
			values = values[:0]

			for _, extractor := range extractors {
				value, ok := extractor(entity)
				if !ok {
					continue outer
				}

				values = append(values, value)
			}

			if !yield(values) {
				return
			}
		}
	}
}

func parseQueryTarget(tyTarget reflect.Type) parsedQueryTarget {
	isSingleTarget := isComponentType(tyTarget) ||
		tyTarget.Kind() == reflect.Pointer && isComponentType(tyTarget.Elem()) ||
		isOptionType(tyTarget)

	if isSingleTarget {
		return parseSingleQueryTarget(tyTarget)
	}

	if tyTarget.Kind() == reflect.Struct {
		return parseStructQueryTarget(tyTarget)
	}

	panic(fmt.Sprintf("unknown query target type: %s", tyTarget))
}

func parseSingleQueryTarget(tyTarget reflect.Type) parsedQueryTarget {
	value := buildQuerySingleValue(tyTarget)

	return parsedQueryTarget{
		extractors: []Extractor{value.extractor},
		populateTarget: func(target reflect.Value, ptrToValues []pointerValue) {
			value.populateTarget(target, ptrToValues[0])
		},
	}
}

func assertIsPointerType(t reflect.Type) {
	if t.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("expected pointer type, got %s", t))
	}
}

func assertIsNonPointerType(t reflect.Type) {
	if t.Kind() == reflect.Pointer {
		panic(fmt.Sprintf("expected non pointer type, got %s", t))
	}
}

func parseStructQueryTarget(tyTarget reflect.Type) parsedQueryTarget {
	var extractors []Extractor
	var populateSingleTargets []PopulateSingleTarget

	for idx := range tyTarget.NumField() {
		field := tyTarget.Field(idx)
		fieldTy := field.Type

		if !field.IsExported() || field.Anonymous {
			continue
		}

		value := buildQuerySingleValue(fieldTy)
		extractors = append(extractors, value.extractor)
		populateSingleTargets = append(populateSingleTargets, value.populateTarget)
	}

	return parsedQueryTarget{
		extractors: extractors,
		populateTarget: func(target reflect.Value, ptrToValues []pointerValue) {
			if len(ptrToValues) != len(populateSingleTargets) {
				panic("unexpected number of pointers")
			}

			// we expect the type to match
			if target.Type() != tyTarget {
				panic(fmt.Sprintf("target type does not match, expected %s, got %s", tyTarget, target.Type()))
			}

			targetType := target.Type()
			for idx := range target.NumField() {
				field := target.Field(idx)

				// TODO cache this
				if !targetType.Field(idx).IsExported() || targetType.Field(idx).Anonymous {
					continue
				}

				populateSingleTargets[idx](field, ptrToValues[idx])
			}
		},
	}
}

func buildQuerySingleValue(tyTarget reflect.Type) queryValueAccessor {
	switch {
	// the entity id is directly injectable
	case tyTarget == reflect.TypeFor[EntityId]():
		return queryValueAccessor{
			extractor: func(entity *Entity) (pointerValue, bool) {
				return pointerValueOf(&entity.Id), true
			},

			populateTarget: func(target reflect.Value, ptrToValue pointerValue) {
				target.Set(ptrToValue.Elem())
			},
		}

	case isComponentType(tyTarget):
		return queryValueAccessor{
			extractor: extractComponentByType(reflectComponentTypeOf(tyTarget)),

			populateTarget: func(target reflect.Value, ptrToValue pointerValue) {
				assertIsNonPointerType(target.Type())
				assertIsPointerType(ptrToValue.Value.Type())

				// copy value to target
				target.Set(ptrToValue.Value.Elem())
			},
		}

	case tyTarget.Kind() == reflect.Pointer && isComponentType(tyTarget.Elem()):
		return queryValueAccessor{
			extractor: extractComponentByType(reflectComponentTypeOf(tyTarget.Elem())),

			populateTarget: func(target reflect.Value, ptrToValue pointerValue) {
				assertIsPointerType(target.Type())
				assertIsPointerType(ptrToValue.Value.Type())

				// let target point to the component
				target.Set(ptrToValue.Value)
			},
		}

	case isOptionType(tyTarget):
		return parseSingleValueForOption(tyTarget)

	case isHasType(tyTarget):
		return parseSingleValueForHas(tyTarget)

	default:
		panic(fmt.Sprintf("not a type we can extract: %s", tyTarget))
	}
}

func isComponentType(t reflect.Type) bool {
	return t.Kind() != reflect.Pointer && t.Implements(reflect.TypeFor[AnyComponent]())
}

func pointerValueOf(value any) pointerValue {
	var v reflect.Value

	switch reflectValue := value.(type) {
	case reflect.Value:
		v = reflectValue

	default:
		v = reflect.ValueOf(value)
	}

	if v.Kind() != reflect.Pointer {
		panic("not a pointer value")
	}

	return pointerValue{Value: v}
}

func reflectComponentTypeOf(tyComponent reflect.Type) ComponentType {
	return ComponentType{
		GoType: tyComponent,
	}
}

func populateTargetStruct(target reflect.Value, ptrToValues []pointerValue) {
	for idx := range target.NumField() {
		value := ptrToValues[idx].Value
		field := target.Field(idx)
		field.Set(value)
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
	if ps, ok := system.(preparedSystem); ok {
		ps.Run()
		return
	}

	// prepare and execute directly
	prepareSystem(w, system).Run()
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
			world.Exec(commands)
		},
	}
}

func prepareSystem(w *World, system System) preparedSystem {
	rSystem := reflect.ValueOf(system)

	if rSystem.Kind() != reflect.Func {
		panic(fmt.Sprintf("not a function: %s", rSystem.Type()))
	}

	tySystem := rSystem.Type()

	// collect a number of functions that when called will prepare the systems parameters
	var params []systemParameter

	for idx := range tySystem.NumIn() {
		tyIn := tySystem.In(idx)

		resourceCopy, resourceCopyOk := w.resources[reflect.PointerTo(tyIn)]
		resource, resourceOk := w.resources[tyIn]

		switch {
		case reflect.PointerTo(tyIn).Implements(reflect.TypeFor[queryAccessor]()):
			params = append(params, valueSystemParameter(buildQuery(w, tyIn)))

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

	return preparedSystem{
		Run: func() {
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
		},
	}
}

type queryAccessor interface {
	__isQuery()
	reflectType() reflect.Type
	setInner(query erasedQuery)
}

func buildQuery(w *World, tyQuery reflect.Type) reflect.Value {
	var ptrToQuery = reflect.New(tyQuery)
	queryAcc := ptrToQuery.Interface().(queryAccessor)

	// build the query from the target type
	parsed := parseQueryTarget(queryAcc.reflectType())

	inner := erasedQuery{
		values:   w.queryValuesIter(parsed.extractors),
		populate: parsed.populateTarget,
	}

	ptrToQuery = reflect.New(tyQuery)
	ptrToQuery.Interface().(queryAccessor).setInner(inner)

	return ptrToQuery.Elem()
}

type Name string

func (n Name) isAnyComponent(isAnyComponentMarker) {}

func (n Name) IsComponent(Name) {}
