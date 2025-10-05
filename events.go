package byke

// Event is an untargeted event and can be used with global observers
// registered with app.add_observer().
type Event interface{}

// EntityEvent is an event targeted at a specific entity.
type EntityEvent interface {
	Event
	TargetEntityId() EntityId
}

type EventTarget EntityId

func (t EventTarget) TargetEntityId() EntityId {
	return EntityId(t)
}
