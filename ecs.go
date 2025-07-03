package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/oliverbestmann/byke/internal/assert"
	"github.com/oliverbestmann/byke/internal/query"
	"reflect"
)

type Tick = arch.Tick
type EntityId = arch.EntityId

type IsComponent[T any] = arch.IsComponent[T]
type IsComparableComponent[T comparable] = arch.IsComparableComponent[T]

type Component[T IsComponent[T]] = arch.Component[T]
type ComparableComponent[T IsComparableComponent[T]] = arch.ComparableComponent[T]

type ErasedComponent = arch.ErasedComponent

type Option[C IsComponent[C]] = query.Option[C]
type OptionMut[C IsComponent[C]] = query.OptionMut[C]

type Has[C IsComponent[C]] = query.Has[C]
type With[C IsComponent[C]] = query.With[C]
type Without[C IsComponent[C]] = query.Without[C]
type Added[C IsComparableComponent[C]] = query.Added[C]
type Changed[C IsComparableComponent[C]] = query.Changed[C]

type ScheduleId any

type resourceValue struct {
	Reflect ptrValue
	Pointer AnyPtr
}

type ptrValue struct {
	reflect.Value
}

type AnyPtr = any

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
	storage     *arch.Storage
	entityIdSeq EntityId
	resources   map[reflect.Type]resourceValue
	schedules   map[ScheduleId][]*preparedSystem
	currentTick Tick
}

func NewWorld() *World {
	return &World{
		storage:   arch.NewStorage(),
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

func (w *World) Spawn(entityId EntityId, components []ErasedComponent) {
	w.storage.Spawn(w.currentTick, entityId)
	w.insertComponents(entityId, components)
}

func (w *World) insertComponents(entityId EntityId, components []ErasedComponent) {
	queue := append([]ErasedComponent{}, components...)

	tick := w.currentTick

	for idx := 0; idx < len(queue); idx++ {
		// if in question we'll overwrite the components if they
		// where specified directly
		overwrite := idx < len(components)

		component := queue[idx]
		componentType := component.ComponentType()

		// maybe skip this one if it already exists on the entity
		exists := w.storage.HasComponent(entityId, componentType)
		if exists && !overwrite {
			continue
		}

		// TODO must not be inserted if it is a parentComponent
		// if _, ok := component.(parentComponent); ok {
		// 	panic(fmt.Sprintf(
		// 		"you may not insert a byke.ParentComponent yourself: %C", component,
		// 	))
		// }

		// and add it to the entity
		component = copyComponent(component)
		w.storage.InsertComponent(tick, entityId, component)

		// trigger hooks for this component
		w.onComponentInsert(entityId, component)

		// enqueue all required components
		if req, ok := component.(arch.RequireComponents); ok {
			queue = append(queue, req.RequireComponents()...)
		}
	}
}

func (w *World) onComponentInsert(entityId EntityId, component ErasedComponent) {
	// TODO update relations if needed
	// if parentComponent, ok := w.parentComponentOf(component); ok {
	// 	parentComponent.addChild(entity.Id)
	// }
}

func (w *World) onComponentRemoved(entity EntityId, component arch.ComponentValue) {
	// TODO update relations if needed
	// if parentComponent, ok := w.parentComponentOf(component); ok {
	// 	parentComponent.removeChild(entity.Id)
	// }
}

func (w *World) parentComponentOf(component ErasedComponent) (parentComponent, bool) {
	return nil, false

	// TODO
	//child, ok := component.(childComponent)
	//if !ok {
	//	return nil, false
	//}
	//
	//parentId := child.parentId()
	//parent, ok := w.entities[parentId]
	//if !ok {
	//	panic(fmt.Sprintf("parent entity %s does not exist", parentId))
	//}
	//
	//parentType := child.RelationParentType()
	//parentComponentValue, ok := parent.Components[parentType]
	//if !ok {
	//	parentComponentValue = componentValue{
	//		PtrToValue: ptrValueOf(reflect.New(parentType.Type)).Interface().(ErasedComponent),
	//	}
	//
	//	parent.Components[parentType] = parentComponentValue
	//}
	//
	//parentComponent := parentComponentValue.PtrToValue.(parentComponent)
	//return parentComponent, true
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

func copyComponent(value ErasedComponent) ErasedComponent {
	componentType := value.ComponentType()
	ptrToValue := componentType.New()
	reflect.ValueOf(ptrToValue).Elem().Set(reflect.ValueOf(value))
	return ptrToValue
}

func ValidateComponent[C IsComponent[C]]() struct{} {
	componentType := arch.ComponentTypeOf[C]()

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

func (w *World) InsertResource(resource any) {
	resType := reflect.PointerTo(reflect.TypeOf(resource))

	if existing, ok := w.resources[resType]; ok {
		// update existing value
		existing.Reflect.Elem().Set(reflect.ValueOf(resource))
		return
	}

	// allocate the resource on the heap and set it
	ptr := reflect.New(resType)
	ptr.Elem().Set(reflect.ValueOf(resource))

	w.resources[ptr.Type()] = resourceValue{
		Reflect: pointerValueOf(ptr),
		Pointer: ptr.Interface(),
	}
}

func pointerValueOf(ptr reflect.Value) ptrValue {
	assert.IsPointerType(ptr.Type())
	return ptrValue{Value: ptr}
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

		entity, ok := w.storage.Get(entityId)
		if !ok {
			fmt.Printf("[warn] cannot despawn entity %d: does not exist\n", entityId)
			continue
		}

		_ = entity

		// TODO
		// update relationships
		//for _, component := range entity.Components {
		//	value := component.PtrToValue
		//	if parentComponent, ok := w.parentComponentOf(value); ok {
		//		parentComponent.removeChild(entityId)
		//	}
		//
		//	// despawn child entities too
		//	if parentComponent, ok := value.(parentComponent); ok {
		//		for _, entityId := range parentComponent.Children() {
		//			queue = append(queue, entityId)
		//		}
		//	}
		//}
	}

	for _, entityId := range queue {
		w.storage.Despawn(entityId)
	}
}

func (w *World) removeComponent(entityId EntityId, componentType *arch.ComponentType) {
	component, ok := w.storage.GetComponent(entityId, componentType)
	if !ok {
		return
	}

	w.storage.RemoveComponent(w.currentTick, entityId, componentType)
	w.onComponentRemoved(entityId, component)
}

func (w *World) recheckComponents(componentTypes []*arch.ComponentType) {
	w.storage.CheckChanged(w.currentTick, componentTypes)
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

func querySystemParameter(world *World, queryType reflect.Type, system *preparedSystem) systemParameter {
	ptrToQueryInstance := reflect.New(queryType)

	queryAccessor := ptrToQueryInstance.Interface().(queryAccessor)

	parsed, err := queryAccessor.parse()
	if err != nil {
		panic(fmt.Sprintf("failed to parse query of type %s: %s", queryType, err))
	}

	theQuery := &parsed.Query

	inner := &innerQuery{
		ParsedQuery: parsed,
		Storage:     world.storage,
		Iter:        world.storage.IterQuery(theQuery),
	}

	queryAccessor.set(inner)

	return systemParameter{
		GetValue: func() reflect.Value {
			theQuery.LastRun = system.LastRun
			return ptrToQueryInstance.Elem()
		},

		Cleanup: func(value reflect.Value) {
			world.recheckComponents(parsed.Mutable)
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
			params = append(params, querySystemParameter(w, tyIn, preparedSystem))

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
