package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
)

type Hovered struct {
	ComparableComponent[Hovered]
}

type Interactable struct {
	Component[Interactable]
}

type Clicked struct{}

type PointerOver struct{}

type PointerOut struct{}

type interactionQueryItem = struct {
	With[Interactable]

	BBox            BBox
	Anchor          Anchor
	GlobalTransform GlobalTransform
	EntityId        EntityId
	Hovered         Has[Hovered]
}

func interactionSystem(
	commands *Commands,
	mouseCursor MouseCursor,
	buttons MouseButtons,
	query Query[interactionQueryItem],
) {
	for item := range query.Items() {
		toLocal, ok := item.GlobalTransform.AsAffine().TryInverse()
		if !ok {
			// maybe GlobalTransform wasn't initialized yet
			continue
		}

		// transform mouse position into the local space of the component
		pos := toLocal.Transform(mouseCursor.Vec)

		// check if we hit the bounding box
		hover := item.BBox.Contains(pos)

		if hover && !item.Hovered.Exists {
			commands.Entity(item.EntityId).
				Update(InsertComponent[Hovered]()).
				Trigger(PointerOver{})
		}

		if !hover && item.Hovered.Exists {
			commands.Entity(item.EntityId).
				Update(RemoveComponent[Hovered]()).
				Trigger(PointerOut{})
		}

		// check if we have just clicked
		justClicked := hover && buttons.IsJustPressed(ebiten.MouseButtonLeft)

		if justClicked {
			// trigger the Clicked event
			commands.Entity(item.EntityId).Trigger(Clicked{})
		}

		break
	}
}
