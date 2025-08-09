package byke

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type MyEvent int

func TestEvents(t *testing.T) {
	var app App

	app.AddEvent(EventType[MyEvent]())

	var eventsWritten int
	var expected []MyEvent

	writer := func(w *EventWriter[MyEvent]) {
		eventsWritten += 1
		w.Write(MyEvent(eventsWritten))
	}

	reader := func(w *EventReader[MyEvent]) {
		require.Equal(t, expected, w.Read())
	}

	w := app.World()

	// start of frame 1
	w.RunSystem(writer)
	w.RunSystem(writer)

	expected = []MyEvent{1, 2}
	w.RunSystem(reader)

	// second call should not read anything
	expected = []MyEvent{}
	w.RunSystem(reader)

	// send another one in the same frame, should be received
	w.RunSystem(writer)

	expected = []MyEvent{3}
	w.RunSystem(reader)

	// send some more to be read in the next frame
	w.RunSystem(writer)
	w.RunSystem(writer)

	// end of frame 1
	w.RunSchedule(Last)

	// frame 2
	expected = []MyEvent{4, 5}
	w.RunSystem(reader)

	// send some more to be read in the next frame
	w.RunSystem(writer) // 6
	w.RunSystem(writer) // 7

	// end of frame 2
	w.RunSchedule(Last)

	// frame 3
	// just write one event and then end the frame
	w.RunSystem(writer) // 8
	w.RunSchedule(Last)

	// frame 4
	// events from frame 2 were not picked up, should be gone now,
	// events from frame 3 should be readable
	expected = []MyEvent{8}
	w.RunSystem(reader)
}
