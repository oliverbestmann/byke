package physics

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
)

type CollisionStarted struct {
	A, B byke.EntityId
	// Arbiter *cp.Arbiter

	Position gm.Vec
	Normal   gm.Vec
}

type CollisionEnded struct {
	A, B byke.EntityId
	// Arbiter *cp.Arbiter

	Position gm.Vec
	Normal   gm.Vec
}

type OnCollisionStarted struct {
	Other byke.EntityId
	// Arbiter *cp.Arbiter

	Position gm.Vec
	Normal   gm.Vec
}

type OnCollisionEnded struct {
	Other byke.EntityId
	// Arbiter *cp.Arbiter

	Position gm.Vec
	Normal   gm.Vec
}
