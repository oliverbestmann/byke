package byke

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type MyMessage int

func TestMessages(t *testing.T) {
	var app App

	app.AddMessage(MessageType[MyMessage]())

	var messagesWritten int
	var expected []MyMessage

	writer := func(w *MessageWriter[MyMessage]) {
		messagesWritten += 1
		w.Write(MyMessage(messagesWritten))
	}

	reader := func(w *MessageReader[MyMessage]) {
		require.Equal(t, expected, w.Read())
	}

	w := app.World()

	// start of frame 1
	w.RunSystem(writer)
	w.RunSystem(writer)

	expected = []MyMessage{1, 2}
	w.RunSystem(reader)

	// second call should not read anything
	expected = []MyMessage{}
	w.RunSystem(reader)

	// send another one in the same frame, should be received
	w.RunSystem(writer)

	expected = []MyMessage{3}
	w.RunSystem(reader)

	// send some more to be read in the next frame
	w.RunSystem(writer)
	w.RunSystem(writer)

	// end of frame 1
	w.RunSchedule(Last)

	// frame 2
	expected = []MyMessage{4, 5}
	w.RunSystem(reader)

	// send some more to be read in the next frame
	w.RunSystem(writer) // 6
	w.RunSystem(writer) // 7

	// end of frame 2
	w.RunSchedule(Last)

	// frame 3
	// just write one message and then end the frame
	w.RunSystem(writer) // 8
	w.RunSchedule(Last)

	// frame 4
	// messages from frame 2 were not picked up, should be gone now,
	// messages from frame 3 should be readable
	expected = []MyMessage{8}
	w.RunSystem(reader)
}
