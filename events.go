package byke

import (
	"fmt"
	"reflect"
)

func EventType[E any]() NewEventType {
	return newEvent[E]{}
}

type NewEventType interface {
	configureEventIn(app *App)
}

type newEvent[E any] struct{}

func (newEvent[E]) configureEventIn(app *App) {
	app.InsertResource(Events[E]{})
	app.AddSystems(Last, updateEventsSystem[E])
}

func updateEventsSystem[E any](events *Events[E]) {
	events.Update()
}

type EventId int

type EventWithId[E any] struct {
	Id    EventId
	Event E
}

type Events[E any] struct {
	_ noCopy

	prevId EventId
	curr   []EventWithId[E]
	prev   []EventWithId[E]
}

func (e *Events[E]) AppendTo(target []EventWithId[E]) []EventWithId[E] {
	target = append(target, e.prev...)
	target = append(target, e.curr...)
	return target
}

func (e *Events[E]) Send(event E) {
	e.prevId += 1

	e.curr = append(e.curr, EventWithId[E]{
		Id:    e.prevId,
		Event: event,
	})
}

func (e *Events[E]) Update() {
	e.curr, e.prev = e.prev, e.curr

	// reuse the memory of the current buffer
	clear(e.curr)
	e.curr = e.curr[:0]
}

func (e *Events[E]) Reader() *EventReader[E] {
	return &EventReader[E]{events: e}
}

func (e *Events[E]) Writer() *EventWriter[E] {
	return &EventWriter[E]{events: e}
}

type EventWriter[E any] struct {
	_ noCopy

	events *Events[E]
}

func (w *EventWriter[E]) Write(event E) {
	w.events.Send(event)
}

func (w *EventWriter[E]) init(world *World) SystemParamState {
	events, ok := ResourceOf[Events[E]](world)
	if !ok {
		var eZero E
		panic(fmt.Sprintf("event %T not registered", eZero))
	}

	reader := events.Writer()
	return valueSystemParamState(reflect.ValueOf(reader))
}

type EventReader[E any] struct {
	_ noCopy

	events *Events[E]
	lastId EventId

	scratch       []E
	scratchWithId []EventWithId[E]
}

func (r *EventReader[E]) Read() []E {
	r.scratchWithId = r.events.AppendTo(r.scratchWithId[:0])

	buffer := r.scratchWithId

	// limit buffer to only the events we've already read
	for len(buffer) > 0 {
		if buffer[0].Id > r.lastId {
			break
		}

		buffer = buffer[1:]
	}

	if len(buffer) > 0 {
		// store the last id we've seen
		r.lastId = buffer[len(buffer)-1].Id
	}

	// convert to event slice, reuse scratch buffer
	events := r.scratch[:0]
	for _, event := range buffer {
		events = append(events, event.Event)
	}

	// keep scratch buffer for reuse
	r.scratch = events

	return events
}

func (r *EventReader[E]) init(world *World) SystemParamState {
	events, ok := ResourceOf[Events[E]](world)
	if !ok {
		var eZero E
		panic(fmt.Sprintf("event %T not registered", eZero))
	}

	reader := events.Reader()
	return valueSystemParamState(reflect.ValueOf(reader))
}
