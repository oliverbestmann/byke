package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/oliverbestmann/byke/internal/assert"
	"github.com/oliverbestmann/byke/internal/query"
	"reflect"
	"slices"
)

type Tick = arch.Tick
type EntityId = arch.EntityId
type IsComponent[T any] = arch.IsComponent[T]
type IsImmutableComponent[T IsComponent[T]] = arch.IsImmutableComponent[T]
type IsComparableComponent[T IsComponent[T]] = arch.IsComparableComponent[T]

type Component[T IsComponent[T]] = arch.Component[T]
type ImmutableComponent[T arch.IsImmutableComponent[T]] = arch.ImmutableComponent[T]
type ComparableComponent[T IsComparableComponent[T]] = arch.ComparableComponent[T]

type ErasedComponent = arch.ErasedComponent

type Option[C IsComponent[C]] = query.Option[C]
type OptionMut[C IsComponent[C]] = query.OptionMut[C]

type Has[C IsComponent[C]] = query.Has[C]
type With[C IsComponent[C]] = query.With[C]
type Without[C IsComponent[C]] = query.Without[C]
type Added[C IsComparableComponent[C]] = query.Added[C]
type Changed[C arch.IsSupportsChangeDetectionComponent[C]] = query.Changed[C]

type Or[A, B query.Filter] = query.Or[A, B]

const NoEntityId = EntityId(0)

type ScheduleId interface {
	isSchedule()
}

type resourceValue struct {
	Reflect ptrValue
	Pointer AnyPtr
}

type ptrValue struct {
	reflect.Value
}

type AnyPtr = any

type World struct {
	storage     *arch.Storage
	entityIdSeq EntityId
	resources   map[reflect.Type]resourceValue
	schedules   map[ScheduleId]*Schedule
	systems     map[SystemId]*preparedSystem
	currentTick Tick
}

func NewWorld() *World {
	return &World{
		storage:   arch.NewStorage(),
		resources: map[reflect.Type]resourceValue{},
		schedules: map[ScheduleId]*Schedule{},
		systems:   map[SystemId]*preparedSystem{},
	}
}

func (w *World) ConfigureSystemSets(scheduleId ScheduleId, systemSets ...*SystemSet) {
	schedule, ok := w.schedules[scheduleId]
	if !ok {
		schedule = NewSchedule()
		w.schedules[scheduleId] = schedule
	}

	for _, systemSet := range systemSets {
		// TODO error handling
		schedule.addSystemSet(systemSet)
	}
}

func (w *World) AddSystems(scheduleId ScheduleId, firstSystem AnySystem, systems ...AnySystem) {
	schedule, ok := w.schedules[scheduleId]
	if !ok {
		schedule = NewSchedule()
		w.schedules[scheduleId] = schedule
	}

	systems = append([]AnySystem{firstSystem}, systems...)

	for _, system := range asSystemConfigs(systems...) {
		preparedSystem := w.prepareSystem(system)

		if err := schedule.addSystem(preparedSystem); err != nil {
			// TODO make nicer
			panic(err)
		}
	}
}

func (w *World) RunSystem(system AnySystem) {
	systemConfig := asSystemConfig(system)
	preparedSystem := w.prepareSystem(systemConfig)
	w.runSystem(preparedSystem, systemContext{})
}

func (w *World) runSystem(system *preparedSystem, ctx systemContext) any {
	for _, predicate := range system.Predicates {
		result := w.runSystem(predicate, systemContext{})
		if result == nil || !result.(bool) {
			// predicate evaluated to "do not run", stop execution here
			return nil
		}
	}

	w.currentTick += 1

	ctx.LastRun = system.LastRun
	result := system.RawSystem(ctx)

	// update last run so we can calculate changed components
	// at the next run
	system.LastRun = w.currentTick

	return result
}

func (w *World) prepareSystem(systemConfig *systemConfig) *preparedSystem {
	// check cache first
	prepared, ok := w.systems[systemConfig.Id]
	if ok {
		return prepared
	}

	// need to prepare the system
	prepared = w.prepareSystemUncached(*systemConfig)
	w.systems[systemConfig.Id] = prepared

	return prepared
}

func (w *World) RunSchedule(scheduleId ScheduleId) {
	schedule, ok := w.schedules[scheduleId]
	if !ok {
		return
	}

	for _, system := range schedule.systems {
		w.runSystem(system, systemContext{})
	}
}

func (w *World) ReserveEntityId() EntityId {
	w.entityIdSeq += 1
	entityId := w.entityIdSeq

	return entityId

}

func (w *World) Spawn(entityId EntityId, components []ErasedComponent) EntityId {
	if entityId == 0 {
		entityId = w.ReserveEntityId()
	}

	w.storage.Spawn(w.currentTick, entityId)
	w.insertComponents(entityId, components)
	return entityId
}

func (w *World) insertComponents(entityId EntityId, components []ErasedComponent) {
	queue := append([]ErasedComponent{}, components...)

	tick := w.currentTick

	var spawnChildren []*spawnChildComponent

	for idx := 0; idx < len(queue); idx++ {
		// if in question we'll overwrite the components if they
		// where specified directly
		overwrite := idx < len(components)

		component := queue[idx]
		componentType := component.ComponentType()

		// special handling for spawn child components. do not add them to
		// the entity, but put them into a list that we go through at the
		// end to spawn children
		if spawnChild, ok := component.(*spawnChildComponent); ok {
			spawnChildren = append(spawnChildren, spawnChild)
			continue
		}

		// maybe skip this one if it already exists on the entity
		exists := w.storage.HasComponent(entityId, componentType)
		if exists && !overwrite {
			continue
		}

		// must not be inserted if it is a parentComponent
		if _, ok := component.(isParentComponent); ok {
			panic(fmt.Sprintf(
				"you may not insert a byke.ParentComponent yourself: %T", component,
			))
		}

		// move it to the heap and add it to the entity
		component = copyComponent(component)
		component = w.storage.InsertComponent(tick, entityId, component)

		// trigger hooks for this component
		w.onComponentInsert(entityId, component)

		// enqueue all required components
		if req, ok := component.(arch.RequireComponents); ok {
			queue = append(queue, req.RequireComponents()...)
		}
	}

	for _, spawnChild := range spawnChildren {
		components := append(spawnChild.Components, ChildOf{Parent: entityId})
		w.Spawn(w.ReserveEntityId(), components)
	}
}

func (w *World) onComponentInsert(entityId EntityId, component ErasedComponent) {
	if parentComponent, parentId, ok := w.parentComponentOf(component, true); ok {
		// create a copy of the component
		parentComponent = copyComponent(parentComponent).(isParentComponent)
		parentComponent.addChild(entityId)

		// and replace its value by inserting it again
		w.storage.InsertComponent(w.currentTick, parentId, parentComponent)
	}
}

func (w *World) onComponentRemoved(entityId EntityId, component ErasedComponent) {
	w.removeEntityFromParentComponentOf(entityId, component)
}

func (w *World) removeEntityFromParentComponentOf(entityId EntityId, component ErasedComponent) {
	if parentComponent, parentId, ok := w.parentComponentOf(component, false); ok {
		if parentComponent != nil {

			// check if would need to remove the last element.
			// in that case, we can just remove the component itself
			children := parentComponent.Children()
			if len(children) == 1 && children[0] == entityId {
				w.storage.RemoveComponent(w.currentTick, parentId, parentComponent.ComponentType())
			} else {
				// create a copy of the component without the child
				parentComponent = copyComponent(parentComponent).(isParentComponent)
				parentComponent.removeChild(entityId)

				// and replace its value by inserting it again
				w.storage.InsertComponent(w.currentTick, parentId, parentComponent)
			}
		}
	}
}

func (w *World) parentComponentOf(component ErasedComponent, create bool) (isParentComponent, EntityId, bool) {
	child, ok := component.(isChildComponent)
	if !ok {
		return nil, 0, false
	}

	parentId := child.ParentEntityId()

	parent, ok := w.storage.Get(parentId)
	if !ok {
		panic(fmt.Sprintf("parent entity %s does not exist", parentId))
	}

	parentType := child.RelationParentType()
	parentComponentValue, ok := parent.Get(parentType)
	if ok {
		return parentComponentValue.Value.(isParentComponent), parentId, true
	}

	if !create {
		// there is no component in the parent
		return nil, parentId, true
	}

	// create a new parent component value
	parentComponent := w.storage.InsertComponent(w.currentTick, parentId, parentType.New())
	w.onComponentInsert(parentId, parentComponent)

	return parentComponent.(isParentComponent), parentId, true
}

func (w *World) AddObserver(observer Observer) {
	// prepare system here. this will also panic if the systems parameters
	// are not well formed.
	observer.system = w.prepareSystem(asSystemConfig(observer.callback))

	w.Spawn(NoEntityId, []ErasedComponent{observer})
}

func (w *World) TriggerObserver(targetId EntityId, eventValue any) {
	// get the event type first
	eventType := reflect.TypeOf(eventValue)

	// TODO maybe check for valid event? Better introduce an Event interface
	w.RunSystem(func(observers Query[*Observer], commands *Commands) {
		for observer := range observers.Items() {
			if observer.eventType != eventType {
				continue
			}

			if len(observer.entities) > 0 && !slices.Contains(observer.entities, targetId) {
				continue
			}

			// we found a match, trigger observer
			w.runSystem(observer.system, systemContext{
				Trigger: systemTrigger{
					TargetId:   targetId,
					EventValue: eventValue,
				},
			})
		}
	})
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

	sourceValue := reflect.ValueOf(value)
	for sourceValue.Kind() == reflect.Pointer {
		sourceValue = sourceValue.Elem()
	}

	reflect.ValueOf(ptrToValue).Elem().Set(sourceValue)
	return ptrToValue
}

func ValidateComponent[C IsComponent[C]]() struct{} {
	componentType := arch.ComponentTypeOf[C]()

	var zero C

	if parent, ok := any(zero).(isParentComponent); ok {
		// check if the child type points to us
		childType := parent.RelationChildType()
		instance := reflect.New(childType.Type).Elem().Interface()

		child, ok := instance.(isChildComponent)
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

	if child, ok := any(zero).(isChildComponent); ok {
		// check if the child type points to us
		parentType := child.RelationParentType()

		parentComponent := parentType.New()
		parent, ok := parentComponent.(isParentComponent)
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
	ptr := reflect.New(resType.Elem())
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

func (w *World) Despawn(entityId EntityId) {
	queue := []EntityId{entityId}

	for idx := 0; idx < len(queue); idx++ {
		entityId = queue[idx]

		entity, ok := w.storage.Get(entityId)
		if !ok {
			fmt.Printf("[warn] cannot despawn entity %d: does not exist\n", entityId)
			continue
		}

		// update relationships
		for _, component := range entity.Components() {
			w.removeEntityFromParentComponentOf(entity.EntityId, component.Value)

			// despawn child entities too
			if parentComponent, ok := component.Value.(isParentComponent); ok {
				for _, entityId := range parentComponent.Children() {
					queue = append(queue, entityId)
				}
			}
		}
	}

	for _, entityId := range queue {
		w.storage.Despawn(entityId)
	}
}

func (w *World) removeComponent(entityId EntityId, componentType *arch.ComponentType) {
	component, ok := w.storage.RemoveComponent(w.currentTick, entityId, componentType)
	if !ok {
		return
	}

	w.onComponentRemoved(entityId, component)
}

func (w *World) recheckComponents(componentTypes []*arch.ComponentType) {
	w.storage.CheckChanged(w.currentTick, componentTypes)
}
