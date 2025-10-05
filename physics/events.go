package physics

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
)

type ContactStartedMessage struct {
	A, B     byke.EntityId
	Position gm.Vec
	Normal   gm.Vec
}

type ContactEndedMessage struct {
	A, B byke.EntityId
}

type ContactStarted struct {
	byke.EventTarget
	Other    byke.EntityId
	Position gm.Vec
	Normal   gm.Vec
}

type ContactEnded struct {
	byke.EventTarget
	Other byke.EntityId
}

type SensorStartedMessage struct {
	A, B     byke.EntityId
	Position gm.Vec
}

type SensorEndedMessage struct {
	A, B byke.EntityId
}

type SensorStarted struct {
	byke.EventTarget
	Other    byke.EntityId
	Position gm.Vec
}

type OnSensorEnded struct {
	byke.EventTarget
	Other    byke.EntityId
	Position gm.Vec
}
