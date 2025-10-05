package byke

import (
	"fmt"
	"reflect"
)

func MessageType[E any]() AddMessageType {
	return newMessage[E]{}
}

type newMessage[E any] struct{}

func (newMessage[E]) configureMessageIn(app *App) {
	app.InsertResource(Messages[E]{})
	app.AddSystems(Last, updateMessagesSystem[E])
}

func updateMessagesSystem[E any](messages *Messages[E]) {
	messages.Update()
}

type MessageId int

type MessageWithId[M any] struct {
	Id      MessageId
	Message M
}

type Messages[E any] struct {
	_ noCopy

	prevId MessageId
	curr   []MessageWithId[E]
	prev   []MessageWithId[E]
}

func (e *Messages[E]) AppendTo(target []MessageWithId[E]) []MessageWithId[E] {
	target = append(target, e.prev...)
	target = append(target, e.curr...)
	return target
}

func (e *Messages[E]) Send(message E) {
	e.prevId += 1

	e.curr = append(e.curr, MessageWithId[E]{
		Id:      e.prevId,
		Message: message,
	})
}

func (e *Messages[E]) Update() {
	e.curr, e.prev = e.prev, e.curr

	// reuse the memory of the current buffer
	clear(e.curr)
	e.curr = e.curr[:0]
}

func (e *Messages[E]) Reader() *MessageReader[E] {
	return &MessageReader[E]{messages: e}
}

func (e *Messages[E]) Writer() *MessageWriter[E] {
	return &MessageWriter[E]{messages: e}
}

type MessageWriter[E any] struct {
	_ noCopy

	messages *Messages[E]
}

func (w *MessageWriter[E]) Write(message E) {
	w.messages.Send(message)
}

func (w *MessageWriter[E]) init(world *World) SystemParamState {
	messages, ok := ResourceOf[Messages[E]](world)
	if !ok {
		var eZero E
		panic(fmt.Sprintf("message type %T not registered", eZero))
	}

	reader := messages.Writer()
	return valueSystemParamState(reflect.ValueOf(reader))
}

type MessageReader[E any] struct {
	_ noCopy

	messages *Messages[E]
	lastId   MessageId

	scratch       []E
	scratchWithId []MessageWithId[E]
}

func (r *MessageReader[E]) Read() []E {
	r.scratchWithId = r.messages.AppendTo(r.scratchWithId[:0])

	buffer := r.scratchWithId

	// limit buffer to only the messages we've already read
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

	// convert to message slice, reuse scratch buffer
	messages := r.scratch[:0]
	for _, message := range buffer {
		messages = append(messages, message.Message)
	}

	// keep scratch buffer for reuse
	r.scratch = messages

	return messages
}

func (r *MessageReader[E]) init(world *World) SystemParamState {
	messages, ok := ResourceOf[Messages[E]](world)
	if !ok {
		var eZero E
		panic(fmt.Sprintf("message %T not registered", eZero))
	}

	reader := messages.Reader()
	return valueSystemParamState(reflect.ValueOf(reader))
}
