package physics

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
)

type ContactStarted struct {
	A, B     byke.EntityId
	Position gm.Vec
	Normal   gm.Vec
}

type ContactEnded struct {
	A, B byke.EntityId
}

type OnContactStarted struct {
	Other    byke.EntityId
	Position gm.Vec
	Normal   gm.Vec
}

type OnContactEnded struct {
	Other byke.EntityId
}

type SensorStarted struct {
	A, B     byke.EntityId
	Position gm.Vec
}

type SensorEnded struct {
	A, B byke.EntityId
}

type OnSensorStarted struct {
	Other    byke.EntityId
	Position gm.Vec
}

type OnSensorEnded struct {
	Other    byke.EntityId
	Position gm.Vec
}
