package byke

import (
	"fmt"
	spoke2 "github.com/oliverbestmann/byke/spoke"
	"reflect"
	"slices"
)

const NoEntityId = EntityId(0)

type resourceValue struct {
	// Value is of kind Pointer and points to the value of the resource.
	Value reflect.Value
}

type ptrValue struct {
	reflect.Value
}

type AnyPtr = any

// World holds all entities and resources, schedules, systems, etc.
// While an empty World can be created using NewWorld, it is normally created and configured
// by using the App api.
type World struct {
	storage     *spoke2.Storage
	entityIdSeq EntityId
	resources   map[reflect.Type]resourceValue
	schedules   map[ScheduleId]*schedule
	systems     map[SystemId]*preparedSystem
	currentTick spoke2.Tick
}

// NewWorld creates a new empty world.
// You probably want to use the App api instead.
func NewWorld() *World {
	return &World{
		storage:   spoke2.NewStorage(),
		resources: map[reflect.Type]resourceValue{},
		schedules: map[ScheduleId]*schedule{},
		systems:   map[SystemId]*preparedSystem{},
	}
}

// AddSystems adds systems to a schedule within the world.
func (w *World) AddSystems(scheduleId ScheduleId, firstSystem AnySystem, systems ...AnySystem) {
	schedule := w.scheduleOf(scheduleId)

	systems = append([]AnySystem{firstSystem}, systems...)

	for _, system := range asSystemConfigs(systems...) {
		preparedSystem := w.prepareSystem(system)
		schedule.AddSystem(preparedSystem)
	}

	if err := schedule.UpdateSystemOrdering(); err != nil {
		panic(err)
	}
}

// RunSystem runs a system within the world.
func (w *World) RunSystem(system AnySystem) {
	systemConfig := asSystemConfig(system)
	preparedSystem := w.prepareSystem(systemConfig)
	w.runSystem(preparedSystem, systemContext{})
}

func (w *World) ConfigureSystemSets(scheduleId ScheduleId, systemSets ...*SystemSet) {
	schedule := w.scheduleOf(scheduleId)

	for _, systemSet := range systemSets {
		schedule.AddSystemSet(systemSet)
	}

	if err := schedule.UpdateSystemOrdering(); err != nil {
		panic(err)
	}
}

func (w *World) timingStats() *TimingStats {
	stats, _ := ResourceOf[TimingStats](w)
	return stats
}

func (w *World) scheduleOf(scheduleId ScheduleId) *schedule {
	schedule, ok := w.schedules[scheduleId]
	if !ok {
		schedule = newSchedule(scheduleId)
		w.schedules[scheduleId] = schedule
	}

	return schedule
}

func (w *World) runSystem(system *preparedSystem, ctx systemContext) any {
	for _, predicate := range system.Predicates {
		result := w.runSystem(predicate, systemContext{})
		if result == nil || !result.(bool) {
			// predicate evaluated to "do not run", stop execution here
			return nil
		}
	}

	if timings := w.timingStats(); timings != nil {
		defer timings.MeasureSystem(system).Stop()
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

// RunSchedule runs the schedule identified by the given ScheduleId.
// If no schedule with this id exists, no action is performed.
func (w *World) RunSchedule(scheduleId ScheduleId) {
	schedule, ok := w.schedules[scheduleId]
	if !ok {
		return
	}

	// remove the schedule while it is executed
	delete(w.schedules, scheduleId)

	// add the schedule back once it has finished executing
	defer func() {
		if _, exists := w.schedules[scheduleId]; exists {
			panic(fmt.Sprintf("The schedule %q was modified while it is being executed", scheduleId))
		}

		w.schedules[scheduleId] = schedule
	}()

	if timings := w.timingStats(); timings != nil {
		defer timings.MeasureSchedule(scheduleId).Stop()
	}

	for _, system := range schedule.systems {
		w.runSystem(system, systemContext{})
	}
}

// AddObserver adds a new observer.
// Observers are entities containing the Observer component.
func (w *World) AddObserver(observer Observer) EntityId {
	// prepare system here. this will also panic if the systems parameters
	// are not well formed.
	observer.system = w.prepareSystem(asSystemConfig(observer.callback))

	return w.Spawn([]ErasedComponent{observer})
}

// TriggerObserver triggers all observers listening on the given target (or all targets) for the
// given event value.
//
// TODO observer event propagation is not yet implemented.
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

// Spawn spawns a new entity with the given components.
func (w *World) Spawn(components []ErasedComponent) EntityId {
	return w.spawnWithEntityId(w.reserveEntityId(), components)
}

func (w *World) reserveEntityId() EntityId {
	w.entityIdSeq += 1
	entityId := w.entityIdSeq

	return entityId

}

func (w *World) spawnWithEntityId(entityId EntityId, components []ErasedComponent) EntityId {
	if entityId == 0 {
		entityId = w.reserveEntityId()
	}

	w.storage.Spawn(w.currentTick, entityId)
	w.insertComponents(entityId, components)
	return entityId
}

// TODO improve this by actually adding all components to storage at the same time.
func (w *World) insertComponents(entityId EntityId, components []ErasedComponent) {
	queue := flattenComponents(nil, components...)

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
		if _, ok := component.(isRelationshipTargetType); ok {
			panic(fmt.Sprintf(
				"you may not insert a byke.RelationshipTarget yourself: %T", component,
			))
		}

		// move it to the heap and add it to the entity
		component = copyComponent(component)
		component = w.storage.InsertComponent(tick, entityId, component)

		// trigger hooks for this component
		w.onComponentInsert(entityId, component)

		// enqueue all required components
		if req, ok := component.(spoke2.RequireComponents); ok {
			queue = append(queue, req.RequireComponents()...)
		}
	}

	for _, spawnChild := range spawnChildren {
		components := append(spawnChild.Components, ChildOf{Parent: entityId})
		w.spawnWithEntityId(w.reserveEntityId(), components)
	}
}

func (w *World) onComponentInsert(entityId EntityId, component ErasedComponent) {
	if targetComponent, targetId, targetType, ok := w.relationshipTargetComponentOf(component); ok {
		if targetComponent == nil {
			// create a new instance of the component
			targetComponent = targetType.New().(isRelationshipTargetType)
		} else {
			// create a copy of the component
			targetComponent = copyComponent(targetComponent).(isRelationshipTargetType)
		}

		// add the child to the relationship target
		targetComponent.addChild(entityId)

		// and replace its value by inserting it again
		w.storage.InsertComponent(w.currentTick, targetId, targetComponent)
	}
}

func (w *World) onComponentRemoved(entityId EntityId, component ErasedComponent) {
	w.removeEntityFromParentComponentOf(entityId, component)

	if registry, ok := ResourceOf[removedComponentsRegistry](w); ok {
		registry.ComponentRemoved(entityId, component.ComponentType())
	}
}

func (w *World) removeEntityFromParentComponentOf(entityId EntityId, component ErasedComponent) {
	if targetComponent, targetId, _, ok := w.relationshipTargetComponentOf(component); ok && targetComponent != nil {

		children := targetComponent.Children()

		if len(children) == 1 && children[0] == entityId {
			// would need to remove the last element.
			// in that case, we can just remove the component itself
			w.storage.RemoveComponent(w.currentTick, targetId, targetComponent.ComponentType())
		} else {
			// create a copy of the component without the child
			targetComponent = copyComponent(targetComponent).(isRelationshipTargetType)
			targetComponent.removeChild(entityId)

			// and replace its value by inserting it again
			w.storage.InsertComponent(w.currentTick, targetId, targetComponent)
		}
	}
}

func (w *World) relationshipTargetComponentOf(component ErasedComponent) (isRelationshipTargetType, EntityId, *spoke2.ComponentType, bool) {
	child, ok := component.(isRelationshipComponent)
	if !ok {
		return nil, 0, nil, false
	}

	parentId := child.RelationshipEntityId()

	parent, ok := w.storage.Get(parentId)
	if !ok {
		panic(fmt.Sprintf("parent entity %s does not exist", parentId))
	}

	parentType := child.RelationshipTargetType()
	parentComponentValue := parent.Get(parentType)
	if parentComponentValue != nil {
		return parentComponentValue.(isRelationshipTargetType), parentId, nil, true
	}

	// there is no component in the parent
	return nil, parentId, parentType, true
}

func (w *World) NewCommands() *Commands {
	return &Commands{world: w}
}

// InsertResource inserts a new resource into the world.
// The resource should be provided as a non-pointer type.
//
// If the resource does not yet exist, a new value of the resources type will
// be allocated on the heap and the value provided will be copied into that memory location.
//
// If the world already contains a resource of the same type, this value will
// just be updated with the newly provided one.
func (w *World) InsertResource(resource any) {
	resType := reflect.PointerTo(reflect.TypeOf(resource))

	if existing, ok := w.resources[resType]; ok {
		// update existing value in place
		existing.Value.Elem().Set(reflect.ValueOf(resource))
		return
	}

	// allocate the resource on the heap and copy the provided value to it
	ptr := reflect.New(resType.Elem())
	ptr.Elem().Set(reflect.ValueOf(resource))

	w.resources[ptr.Type()] = resourceValue{
		Value: ptr,
	}
}

// RemoveResource removes a resource previously added with InsertResource.
func (w *World) RemoveResource(resourceType reflect.Type) {
	resType := reflect.PointerTo(resourceType)
	delete(w.resources, resType)
}

// Despawn recursively despawns the given entity following Children relations.
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
			w.onComponentRemoved(entityId, component)

			// despawn child entities too
			if parentComponent, ok := component.(isRelationshipTargetType); ok {
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

// Resource returns a pointer to the resource of the given reflect type.
// The type must be the non-pointer type of the resource, i.e. the type of the resource
// as it was passed to InsertResource.
func (w *World) Resource(ty reflect.Type) (AnyPtr, bool) {
	resValue, ok := w.resources[reflect.PointerTo(ty)]
	if !ok {
		return nil, false
	}

	return resValue.Value.Interface(), true
}

// ResourceOf is a typed version of World.Resource.
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

	// get the actual value of the source
	sourceValue := reflect.ValueOf(value)
	for sourceValue.Kind() == reflect.Pointer {
		sourceValue = sourceValue.Elem()
	}

	// copy the source to the newly allocated component
	reflect.ValueOf(ptrToValue).Elem().Set(sourceValue)
	return ptrToValue
}

func (w *World) removeComponent(entityId EntityId, componentType *spoke2.ComponentType) {
	component, ok := w.storage.RemoveComponent(w.currentTick, entityId, componentType)
	if !ok {
		return
	}

	w.onComponentRemoved(entityId, component)
}

func (w *World) recheckComponents(query *spoke2.Query, componentTypes []*spoke2.ComponentType) {
	w.storage.CheckChanged(w.currentTick, query, componentTypes)
}
