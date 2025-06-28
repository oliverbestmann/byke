package ecs

import (
	"fmt"
	"iter"
	"reflect"
)

type AnyPtr any

type PopulateTarget func(target reflect.Value, ptrToValues []pointerValue)

type Query[T any] struct {
	innerQuery
}

type innerQuery struct {
	values   iter.Seq[[]pointerValue]
	populate PopulateTarget
}

func (*Query[T]) __isQuery() {}

func (*Query[T]) reflectType() reflect.Type {
	return reflect.TypeFor[T]()
}

func (q *Query[T]) setInner(inner innerQuery) {
	q.innerQuery = inner
}

func (q *Query[T]) Get() (value *T, ok bool) {
	for value := range q.Items() {
		return &value, true
	}

	return nil, false
}

func (q *Query[T]) Items() iter.Seq[T] {
	return func(yield func(T) bool) {
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
	IsComponent(T)
}

type AnyComponent any

type RequireComponents interface {
	RequireComponents() []AnyComponent
}

type Component[T IsComponent[T]] struct{}

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
	Reflect reflect.Value
	Pointer AnyPtr
}

type Schedule struct {
	// make this a non zero sized type, so that creating a
	// new Schedule will always return a different memory address
	_nonZero uint32
}

type System any

type World struct {
	entityIdSeq EntityId
	entities    map[EntityId]*Entity
	resources   map[reflect.Type]resourceValue
	schedules   map[*Schedule][]System
	queryCache  map[reflect.Type]reflect.Value
}

func NewWorld() World {
	return World{
		entities:   map[EntityId]*Entity{},
		resources:  map[reflect.Type]resourceValue{},
		schedules:  map[*Schedule][]System{},
		queryCache: map[reflect.Type]reflect.Value{},
	}
}

func (w *World) AddSystems(schedule *Schedule, systems ...System) {
	w.schedules[schedule] = append(w.schedules[schedule], systems...)
}

func (w *World) RunSchedule(schedule *Schedule) {
	for _, system := range w.schedules[schedule] {
		w.RunSystem(system)
	}
}

func (w *World) Insert(entity *Entity) {
	w.entities[entity.Id] = entity
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

type Commands struct {
	world *World
	queue []Command
}

func (c *Commands) Spawn(components ...AnyComponent) EntityCommands {
	entityId := c.world.entityIdSeq
	c.world.entityIdSeq += 1

	entity := &Entity{
		Id:         entityId,
		Components: map[ComponentType]componentValue{},
	}

	c.queue = append(c.queue, func(world *World) {
		queue := append([]AnyComponent(nil), components...)

		var added Set[ComponentType]

		for idx := 0; idx < len(queue); idx++ {
			tyComponent := anyComponentTypeOf(queue[idx])
			if added.Has(tyComponent) {
				continue
			}

			added.Insert(tyComponent)

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

		world.Insert(entity)
	})

	return EntityCommands{
		entityId: entityId,
		commands: c,
	}
}

func copyToHeap(component AnyComponent) pointerValue {
	// move the component onto the heap
	ptrToComponent := reflect.New(reflect.TypeOf(component))
	ptrToComponent.Elem().Set(reflect.ValueOf(component))
	return pointerValue{Value: ptrToComponent}
}

type EntityCommands struct {
	entityId EntityId
	commands *Commands
}

func (e EntityCommands) Update(commands ...EntityCommand) EntityCommands {
	e.commands.queue = append(e.commands.queue, func(world *World) {
		entity := world.entities[e.entityId]

		for _, command := range commands {
			command(world, entity)
		}
	})

	return e
}

func (e EntityCommands) Despawn() {
	e.commands.queue = append(e.commands.queue, func(world *World) {
		delete(world.entities, e.entityId)
	})
}

func RemoveComponent[C IsComponent[C]]() EntityCommand {
	componentType := componentTypeOf[C]()

	return func(world *World, entity *Entity) {
		delete(entity.Components, componentType)
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

type parsedQuery struct {
	extractors     []Extractor
	populateValues PopulateTarget
}

func (w *World) queryValues(q parsedQuery) iter.Seq[[]pointerValue] {
	return func(yield func([]pointerValue) bool) {
		var values []pointerValue

	outer:
		for _, entity := range w.entities {
			values = values[:0]

			for _, extractor := range q.extractors {
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

func parseQuery(tyTarget reflect.Type) parsedQuery {
	if isComponentType(tyTarget) {
		// the target is a single component
		tyComponent := reflectComponentTypeOf(tyTarget)
		extractor := extractComponentByType(tyComponent)
		return parsedQuery{
			extractors: []Extractor{extractor},
			populateValues: func(target reflect.Value, ptrToValues []pointerValue) {
				// target contains a target of type tyTarget.

				if ptrToValues[0].Value.Kind() != reflect.Pointer {
					panic(fmt.Sprintf("expected pointer, got %s", ptrToValues[0].Type()))
				}

				ptrToValue := ptrToValues[0].Value
				if target.Kind() != reflect.Ptr {
					ptrToValue = ptrToValue.Elem()
				}

				target.Set(ptrToValue)
			},
		}
	}

	// TODO check tyTarget == struct

	var extractors []Extractor

	for idx := range tyTarget.NumField() {
		field := tyTarget.Field(idx)
		fieldTy := field.Type

		if fieldTy.Kind() == reflect.Ptr {
			// mutable
			fieldTy = fieldTy.Elem()
		}

		var extractor Extractor

		switch {
		case fieldTy == reflect.TypeFor[EntityId]():
			// fill in the entity id
			extractor = func(entity *Entity) (pointerValue, bool) {
				return pointerValueOf(&entity.Id), true
			}

		case isComponentType(fieldTy):
			extractor = extractComponentByType(ComponentType{
				GoType: fieldTy,
			})

		case reflect.PointerTo(fieldTy).Implements(reflect.TypeFor[optionAccessor]()):
			extractor = extractOptionOf(fieldTy)

		default:
			panic(fmt.Sprintf("not a type we can extract: %s", fieldTy))
		}

		extractors = append(extractors, extractor)
	}

	return parsedQuery{
		extractors:     extractors,
		populateValues: populateTargetStruct,
	}
}

func isComponentType(t reflect.Type) bool {
	_, ok := t.MethodByName("IsComponent")
	return ok
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

		if field.Kind() != reflect.Ptr {
			value = value.Elem()
		}

		field.Set(value)
	}
}

func (w *World) InsertResource(res any) {
	ptr := reflect.New(reflect.TypeOf(res))
	ptr.Elem().Set(reflect.ValueOf(res))

	w.resources[ptr.Type()] = resourceValue{
		Reflect: ptr,
		Pointer: ptr.Interface(),
	}
}

func (w *World) RunSystem(system System) {
	runSystem(w, reflect.ValueOf(system))
}

func runSystem(w *World, system reflect.Value) []reflect.Value {
	if system.Kind() != reflect.Func {
		panic(fmt.Sprintf("not a function: %s", system.Type()))
	}

	tySystem := system.Type()

	var params []reflect.Value

	for idx := range tySystem.NumIn() {
		tyIn := tySystem.In(idx)

		resourceCopy, resourceCopyOk := w.resources[reflect.PointerTo(tyIn)]
		resource, resourceOk := w.resources[tyIn]

		switch {
		case reflect.PointerTo(tyIn).Implements(reflect.TypeFor[queryAccessor]()):
			params = append(params, buildQuery(w, tyIn))

		case tyIn == reflect.TypeFor[*Commands]():
			commands := &Commands{world: w}

			// Apply commands in the end
			//goland:noinspection GoDeferInLoop
			defer w.Exec(commands)

			param := reflect.ValueOf(commands)
			params = append(params, param)

		case tyIn == reflect.TypeFor[*World]():
			params = append(params, reflect.ValueOf(w))

		case resourceCopyOk:
			params = append(params, resourceCopy.Reflect.Elem())

		case resourceOk:
			params = append(params, resource.Reflect)

		default:
			panic(fmt.Sprintf("Can not handle system param of type %s", tyIn))
		}
	}

	return system.Call(params)
}

type queryAccessor interface {
	__isQuery()
	reflectType() reflect.Type
	setInner(query innerQuery)
}

func buildQuery(w *World, tyQuery reflect.Type) reflect.Value {
	var query reflect.Value

	if cached := w.queryCache[tyQuery]; cached.IsValid() {
		query = cached

	} else {
		var ptrToQuery = reflect.New(tyQuery)
		queryAcc := ptrToQuery.Interface().(queryAccessor)

		// build the query from the target type
		parsed := parseQuery(queryAcc.reflectType())

		inner := innerQuery{
			values:   w.queryValues(parsed),
			populate: parsed.populateValues,
		}

		ptrToQuery = reflect.New(tyQuery)
		ptrToQuery.Interface().(queryAccessor).setInner(inner)

		query = ptrToQuery.Elem()

		// cache query
		w.queryCache[tyQuery] = query
	}

	return query
}

type Name string

func (n Name) IsComponent(Name) {}
