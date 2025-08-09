package byke

import (
	"iter"
	"reflect"

	"github.com/oliverbestmann/byke/spoke"
)

type RemovedComponents[C IsComponent[C]] struct {
	reader *EventReader[removedComponentEvent[C]]
}

func (c RemovedComponents[C]) Read() iter.Seq[EntityId] {
	return func(yield func(EntityId) bool) {
		for _, event := range c.reader.Read() {
			if !yield(EntityId(event)) {
				return
			}
		}
	}
}

func (RemovedComponents[C]) addToWorld(w *World) *Events[removedComponentEvent[C]] {
	if events, exists := ResourceOf[Events[removedComponentEvent[C]]](w); exists {
		return events
	}

	registry, ok := ResourceOf[removedComponentsRegistry](w)
	if !ok {
		w.InsertResource(removedComponentsRegistry{
			byComponentType: map[*spoke.ComponentType]func(EntityId){},
		})

		registry, _ = ResourceOf[removedComponentsRegistry](w)
	}

	w.InsertResource(Events[removedComponentEvent[C]]{})
	w.AddSystems(Last, updateEventsSystem[removedComponentEvent[C]])

	events, _ := ResourceOf[Events[removedComponentEvent[C]]](w)

	componentType := spoke.ComponentTypeOf[C]()

	writer := events.Writer()
	registry.byComponentType[componentType] = func(entityId EntityId) {
		writer.Write(removedComponentEvent[C](entityId))
	}

	return events
}

func (c RemovedComponents[C]) init(world *World) SystemParamState {
	events := c.addToWorld(world)

	instance := RemovedComponents[C]{reader: events.Reader()}
	return valueSystemParamState(reflect.ValueOf(instance))
}

type removedComponentEvent[C IsComponent[C]] EntityId

type removedComponentsRegistry struct {
	byComponentType map[*spoke.ComponentType]func(EntityId)
}

func (r *removedComponentsRegistry) ComponentRemoved(entityId EntityId, componentType *spoke.ComponentType) {
	emit, ok := r.byComponentType[componentType]
	if !ok {
		return
	}

	emit(entityId)
}
