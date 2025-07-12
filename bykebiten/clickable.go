package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
	"slices"
)

type Interactable struct {
	ImmutableComponent[Interactable]
}

func (i Interactable) RequireComponents() []ErasedComponent {
	return []ErasedComponent{
		InteractionState{None: true},
	}
}

type Clicked struct{}

type PointerOver struct{}

type PointerOut struct{}

type InteractionState struct {
	ImmutableComponent[InteractionState]
	None  bool
	Hover bool
}

var (
	InteractionStateNone  = InteractionState{None: true}
	InteractionStateHover = InteractionState{Hover: true}
)

type interactionQueryItem = struct {
	With[Interactable]

	EntityId

	BBox             BBox
	Anchor           Anchor
	Layer            Layer
	GlobalTransform  GlobalTransform
	InteractionState InteractionState
}

func interactionSystem(
	commands *Commands,
	mouseCursor MouseCursor,
	buttons MouseButtons,
	query Query[interactionQueryItem],
	queryCache *Local[[]interactionQueryItem],
) {
	queryCache.Value = slices.AppendSeq(queryCache.Value[:0], query.Items())

	items := queryCache.Value

	// sort by reverse layer, top most layer will be the first item
	slices.SortFunc(items, func(a, b interactionQueryItem) int {
		switch {
		case a.Layer.Z < b.Layer.Z:
			return 1
		case a.Layer.Z > b.Layer.Z:
			return -1
		default:
			return 0
		}
	})

	for _, item := range items {
		toLocal, ok := item.GlobalTransform.AsAffine().TryInverse()
		if !ok {
			// maybe GlobalTransform wasn't initialized yet
			continue
		}

		// transform mouse position into the local space of the component
		pos := toLocal.Transform(mouseCursor.Vec)

		// check if we hit the bounding box
		hover := item.BBox.Contains(pos)

		if !hover {
			if item.InteractionState == InteractionStateHover {
				commands.Entity(item.EntityId).
					Update(InsertComponent(InteractionStateNone)).
					Trigger(PointerOut{})
			}

			continue
		}

		if item.InteractionState != InteractionStateHover {
			commands.Entity(item.EntityId).
				Update(InsertComponent(InteractionStateHover)).
				Trigger(PointerOver{})
		}

		// check if we have just clicked
		justClicked := buttons.IsJustPressed(ebiten.MouseButtonLeft)

		if justClicked {
			// trigger the Clicked event
			commands.Entity(item.EntityId).Trigger(Clicked{})
		}
	}
}

type QueryCache[T any] struct {
	Local[[]T]
}
