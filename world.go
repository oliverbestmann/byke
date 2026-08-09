package byke

import (
	"fmt"
	"log/slog"
	"reflect"
	"sync/atomic"

	"github.com/oliverbestmann/byke/internal/set"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/puffin-go"
)

const NoEntityId = EntityId(0)

type AnyPtr = any

// World holds all entities and resources, schedules, systems, etc.
// While an empty World can be created using NewWorld, it is normally created and configured
// by using the App api.
type World struct {
	resourceContainer

	storage          *spoke.Storage
	entityIdSeq      EntityId
	schedules        map[ScheduleId]*schedule
	systems          map[SystemId]*preparedSystem
	makeSystemParams makeSystemParams

	currentTick   spoke.Tick
	activeQueries atomic.Int32

	commands CommandQueue
}

// NewWorld creates a new empty world.
// You probably want to use the App api instead.
func NewWorld() *World {
	flushComponentValidations()

	defaultMakeSystemParams := makeSystemParams{
		makeWorldSystemParamState,
		makeCommandsSystemStateParam,
		ForwardToNewStateOnPointer[queryT],
		forwardToNewState[localT],
		forwardToNewState[messageWriterT],
		forwardToNewState[messageReaderT],
		forwardToNewState[singleT],
		forwardToNewState[inT],
		forwardToNewState[onT],
		forwardToNewState[resT],
		forwardToNewState[resOptionT],
		forwardToNewState[removedComponentsT],
	}

	return &World{
		resourceContainer: resourceContainer{},
		storage:           spoke.NewStorage(),
		schedules:         map[ScheduleId]*schedule{},
		systems:           map[SystemId]*preparedSystem{},
		makeSystemParams:  defaultMakeSystemParams,
		currentTick:       1,
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
	w.RunSystemWithInValue(system, nil)
}

func (w *World) RunSystemWithInValue(system AnySystem, inValue any) {
	w.runSystemWithValue(system, inValue)
}

func (w *World) runSystemWithValue(system AnySystem, inValue any) {
	systemConfig := asSystemConfig(system)
	preparedSystem := w.prepareSystem(systemConfig)
	w.runSystem(preparedSystem, SystemContext{InValue: inValue})
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

func (w *World) AddMakeSystemParam(msp MakeSystemParam) {
	w.makeSystemParams = append(w.makeSystemParams, msp)
}

func (w *World) timingStats() *TimingStats {
	stats, _ := w.ResourceOf[TimingStats]()
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

func (w *World) runSystem(system *preparedSystem, ctx SystemContext) any {
	checkpoint := w.commands.Checkpoint()

	result := w.runSystemWithoutApplyingCommands(system, ctx)

	// TODO do we need this? This should only be possible if we run
	//  World.RunSchedule or World.RunSystem while iterating a query.
	//  In bevy this can not happen due to needing &mut World to run a system
	//  or a schedule, which forbids you from iterating a query at the same time.
	//
	if count := int(w.activeQueries.Load()); count != 0 {
		// slog.Warn("Flushing commands delayed, active query exists", slog.Int("count", count))
		panic(fmt.Errorf("queries are still active: %d", count))
	}

	w.applyCommands(checkpoint)

	return result
}

func (w *World) runSystemWithoutApplyingCommands(system *preparedSystem, ctx SystemContext) any {
	defer puffin.NewScopeWithValue("byke.RunSystem", system.Name).End()

	for _, predicate := range system.Predicates {
		result := w.runSystemWithoutApplyingCommands(predicate, SystemContext{})
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

	defer puffin.NewScopeWithValue("RunSchedule", scheduleId.String()).End()

	// all added commands should be handled already
	checkpoint := w.commands.Checkpoint()
	defer assertIsEmpty(w.commands.DrainAt(checkpoint))

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

	for _, system := range schedule.Systems() {
		w.runSystem(system, SystemContext{})
	}
}

func assertIsEmpty(slice []Command) {
	if len(slice) > 0 {
		panic(fmt.Errorf("expected to have no pending commands, got %#v", slice))
	}
}

// AddObserver adds a new observer.
// Observers are entities containing the Observer component.
func (w *World) AddObserver(observer Observer) EntityId {
	// prepare system here. this will also panic if the systems parameters
	// are not wellformed.
	observer.system = w.prepareSystem(asSystemConfig(observer.callback))

	return w.Spawn([]ErasedComponent{observer})
}

// TriggerObserver triggers all observers listening on the given target (or all targets) for the
// given event value.
//
// TODO observer event propagation is not yet implemented.
func (w *World) TriggerObserver(eventValue Event) {
	// get the event type first
	eventType := reflect.TypeOf(eventValue)

	checkpoint := w.commands.Checkpoint()
	defer assertIsEmpty(w.commands.DrainAt(checkpoint))

	w.RunSystemWithInValue(triggerObserverSystem, triggerObserverIn{
		ObserverType: eventType,
		EventValue:   eventValue,
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
	if entityId == NoEntityId {
		entityId = w.reserveEntityId()
	}

	components, spawnChildren := w.prepareComponents(entityId, components)

	w.storage.Spawn(w.currentTick, entityId, components)
	w.onComponentsInsert(entityId, components)

	// now spawn all childrens as necessary
	for _, spawnChild := range spawnChildren {
		components := append(spawnChild.Components, ChildOf{Parent: entityId})
		w.spawnWithEntityId(w.reserveEntityId(), components)
	}

	return entityId
}

func (w *World) insertComponents(entityId EntityId, components []ErasedComponent) {
	components, spawnChildren := w.prepareComponents(entityId, components)

	w.storage.InsertComponents(w.currentTick, entityId, components)
	w.onComponentsInsert(entityId, components)

	// now spawn all childrens as necessary
	for _, spawnChild := range spawnChildren {
		components := append(spawnChild.Components, ChildOf{Parent: entityId})
		w.spawnWithEntityId(w.reserveEntityId(), components)
	}
}

func (w *World) prepareComponents(entityId EntityId, components []ErasedComponent) (collectedComponents []ErasedComponent, spawnChildren []*spawnChildComponent) {
	queue := flattenComponents(nil, components...)
	var inserted set.Set[*spoke.ComponentType]

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

		// skip if we've already added the component type to the queue
		if !inserted.Insert(componentType) {
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

		collectedComponents = append(collectedComponents, component)

		// enqueue all required components
		queue = append(queue, componentType.RequiredComponents()...)
	}

	return
}

func (w *World) onComponentsInsert(id EntityId, components []ErasedComponent) {
	for _, component := range components {
		w.onComponentInsert(id, component)
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

	if registry, ok := w.ResourceOf[removedComponentsRegistry](); ok {
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

func (w *World) relationshipTargetComponentOf(component ErasedComponent) (isRelationshipTargetType, EntityId, *spoke.ComponentType, bool) {
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

// Despawn recursively despawns the given entity following Children relations.
func (w *World) Despawn(entityId EntityId) {
	queue := []EntityId{entityId}

	for idx := 0; idx < len(queue); idx++ {
		entityId = queue[idx]

		entity, ok := w.storage.Get(entityId)
		if !ok {
			slog.Warn(
				"cannot despawn entity: entity does not exist",
				slog.Any("entityId", entityId),
			)

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

func (w *World) Query[T any]() Query[T] {
	state := NewQuerySystemParamState[T](w)
	value, err := state.GetValue(SystemContext{})
	if err != nil {
		panic(err)
	}

	return value.Interface().(Query[T])
}

func (w *World) PrintSystems() {
	for id, schedule := range w.schedules {
		fmt.Println()
		fmt.Printf("Schedule %q:\n", id)

		for _, sys := range schedule.Systems() {
			fmt.Println(" ->", sys.Name)
		}
	}
}

func (w *World) removeComponent(entityId EntityId, componentType *spoke.ComponentType) {
	component, ok := w.storage.RemoveComponent(w.currentTick, entityId, componentType)
	if !ok {
		return
	}

	w.onComponentRemoved(entityId, component)
}

func (w *World) recheckComponents(query *spoke.CachedQuery, componentTypes []*spoke.ComponentType) {
	w.storage.CheckChanged(w.currentTick, query, componentTypes)
}

// RegisterComponentHooks returns a new RegisterComponentHooks instance that can be used
// to register component hooks
func (w *World) RegisterComponentHooks[T IsComponent[T]]() RegisterComponentHooks {
	componentType := spoke.ComponentTypeOf[T]()

	return RegisterComponentHooks{
		world:    w,
		delegate: w.storage.RegisterComponentHooks(componentType),
	}
}

func (w *World) Commands() *CommandQueue {
	return &w.commands
}

func (w *World) dispatchHookWithWorld(entity spoke.EntityRef, component *spoke.ComponentType, hook ComponentHook) {
	var dw DeferredWorld = w
	hook(dw, entity, component)
}

func (w *World) applyCommands(checkpoint Checkpoint) {
	if w.activeQueries.Load() != 0 {
		panic("cannot apply commands, queries are still running")
	}

	commands := w.commands.DrainAt(checkpoint)
	if len(commands) > 0 {
		defer puffin.NewScope("byke.FlushCommands").End()
	}

	for _, command := range commands {
		command.Apply(w)
	}
}

func copyComponent(value ErasedComponent) ErasedComponent {
	return value.ComponentType().CopyOf(value)
}

type triggerObserverIn struct {
	ObserverType reflect.Type
	EventValue   Event
}

func triggerObserverSystem(
	w *World,
	observers Query[*Observer],
	in In[triggerObserverIn],
) {
	params := &in.Value

	targetId := NoEntityId
	if params.EventValue != nil {
		if ev, ok := params.EventValue.(EntityEvent); ok {
			targetId = ev.TargetEntityId()
		}
	}

	checkpoint := w.commands.Checkpoint()

	for observer := range observers.Items() {
		if !observer.ObservesType(params.ObserverType) {
			continue
		}

		if targetId == NoEntityId && observer.IsScoped() {
			continue
		}

		if targetId != NoEntityId && !observer.Observes(targetId) {
			continue
		}

		// we found a match, trigger the observer
		w.runSystemWithoutApplyingCommands(observer.system, SystemContext{
			Trigger: systemTrigger{
				EventValue: params.EventValue,
			},
		})
	}

	w.applyCommands(checkpoint)
}
