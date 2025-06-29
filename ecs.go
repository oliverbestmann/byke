package byke

import (
	"fmt"
	"reflect"
)

type AnyPtr any

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
	PtrToValue ptrValue
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

func copyToHeap(value any) ptrValue {
	if reflect.TypeOf(value).Kind() == reflect.Pointer {
		panic("we do not want to have double pointers")
	}

	// move the component onto the heap
	ptrToValue := reflect.New(reflect.TypeOf(value))
	ptrToValue.Elem().Set(reflect.ValueOf(value))
	return ptrValueOf(ptrToValue)
}

func ValidateComponent[T IsComponent[T]]() struct{} {
	// TODO mark component as valid somewhere, maybe calculate some
	//  kind of component type id too
	return struct{}{}
}

func reflectComponentTypeOf(tyComponent reflect.Type) ComponentType {
	return ComponentType{
		GoType: tyComponent,
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

type Name string

func (n Name) isAnyComponent(isAnyComponentMarker) {}

func (n Name) IsComponent(Name) {}
