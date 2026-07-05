package byke

import (
	"iter"
	"reflect"

	"github.com/oliverbestmann/byke/spoke"
)

type RemovedComponents[C IsComponent[C]] struct {
	reader *MessageReader[removedComponentEvent[C]]
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

func removedComponentsAddToWorld[C IsComponent[C]](w *World) *Messages[removedComponentEvent[C]] {
	if events, exists := w.ResourceOf[Messages[removedComponentEvent[C]]](); exists {
		return events
	}

	registry, ok := w.ResourceOf[removedComponentsRegistry]()
	if !ok {
		w.InsertResource(removedComponentsRegistry{
			byComponentType: map[*spoke.ComponentType]func(EntityId){},
		})

		registry, _ = w.ResourceOf[removedComponentsRegistry]()
	}

	w.InsertResource(Messages[removedComponentEvent[C]]{})
	w.AddSystems(Last, updateMessagesSystem[removedComponentEvent[C]])

	events, _ := w.ResourceOf[Messages[removedComponentEvent[C]]]()

	componentType := spoke.ComponentTypeOf[C]()

	writer := events.Writer()
	registry.byComponentType[componentType] = func(entityId EntityId) {
		writer.Write(removedComponentEvent[C](entityId))
	}

	return events
}

func (RemovedComponents[C]) newState(world *World, _ removedComponentsT) SystemParamState {
	events := removedComponentsAddToWorld[C](world)
	instance := RemovedComponents[C]{reader: events.Reader()}
	return valueSystemParamState(reflect.ValueOf(instance))
}

type removedComponentsT interface {
	newState(_ *World, _ removedComponentsT) SystemParamState
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
