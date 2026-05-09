package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/glm"
)

var _ = byke.ValidateComponent[TextureAtlas]()

type TextureAtlas struct {
	byke.Component[TextureAtlas]
	Layout TextureAtlasLayout
	Index  int

	// Wrap index around if it is out of rane
	Wrapping bool
}

func (t TextureAtlas) IsValid() bool {
	if len(t.Layout) == 0 {
		return false
	}

	return t.Wrapping || (t.Index >= 0 && t.Index < len(t.Layout))
}

func (t TextureAtlas) Current() (glm.Rect2u, bool) {
	if !t.IsValid() {
		return glm.Rect2u{}, false
	}

	rect := t.Layout[t.Index%len(t.Layout)]
	return rect, true
}

type TextureAtlasLayout []glm.Rect2u

func TextureAtlasLayoutFromRect(rect glm.Rect2u) TextureAtlasLayout {
	return TextureAtlasLayout{rect}
}

type GridOptions struct {
	// Total number of Columns & Rows in the Grid. Columns and Rows can each be derived
	// if Count is set to a non zero value.
	Columns, Rows uint

	// Defaults to Columns * Rows if not defined, otherwise this will
	// be the number of tiles to take from the Grid in the order
	// of left to right, top to bottom.
	Count uint

	// The StartColumn and StartRow indices the row & column
	// to start the grid at.
	StartColumn, StartRow uint

	// Width and Height of a single grid cell in pixels.
	Width, Height uint

	// Offset in pixels from the images origin.
	OffsetX, OffsetY uint

	// Gap in pixels between two adjecent tiles.
	GapX, GapY uint
}

func TextureAtlasLayoutFromGrid(opts GridOptions) TextureAtlasLayout {
	count := opts.Count
	columns := opts.Columns

	// if no count is defined,
	// we derive if from the number of rows & columns.
	if count == 0 {
		count = opts.Rows * opts.Columns
	}

	if count == 0 {
		panic("number of tiles in the grid must be positive")
	}

	// if the number of Columns is not defined, we set derive it from
	// the count of images.
	if columns == 0 {
		columns = count
	}

	var layout TextureAtlasLayout

	for n := range count {
		// clamp n into the valid range
		n = min(max(n, 0), count)

		// calculate the index of the tile
		row := n/columns + opts.StartRow
		column := n%columns + opts.StartColumn

		// calculate the tiles pixel position
		x0 := column*opts.Width + opts.GapX + opts.OffsetX
		y0 := row*opts.Height + opts.GapY + opts.OffsetY
		x1 := x0 + opts.Width
		y1 := y0 + opts.Height

		layout = append(layout, glm.RectFromPoints(
			glm.Vec2u{uint32(x0), uint32(y0)},
			glm.Vec2u{uint32(x1), uint32(y1)},
		))
	}

	return layout
}
