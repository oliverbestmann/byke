package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
)

type Clickable struct {
	Component[Clickable]
}

type Clicked struct{}

func checkClickSystem(commands *Commands, mouseCursor MouseCursor, buttons MouseButtons, query Query[struct {
	With[Clickable]
	EntityId
	Size            Size
	Anchor          Option[Anchor]
	GlobalTransform GlobalTransform
}]) {
	// TODO make this a predicate
	if !buttons.IsJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	for item := range query.Items() {
		// TODO implement check if click was in bounding box
		pos := item.GlobalTransform.AsAffine().Inverse().Transform(mouseCursor.Vec)
		_ = pos

		commands.Entity(item.EntityId).Trigger(Clicked{})

		break
	}
}
