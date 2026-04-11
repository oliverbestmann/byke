package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
)

type TextureAtlas struct {
	byke.Component[TextureAtlas]
	Layout TextureAtlasLayout
	Index  int
}

type TextureAtlasLayout []wx.Rectangle2u

func TextureAtlasFromRect(rect wx.Rectangle2u) TextureAtlas {
	return TextureAtlas{
		Layout: TextureAtlasLayout{rect},
	}
}

type GridOptions struct {
	// Defaults to Width * Height if not defined, otherwise this will
	// be the number of frames from the Grid, left to right
	Count uint32

	Columns uint16
	Rows    uint16
	Width   uint16
	Height  uint16
	OffsetX uint16
	OffsetY uint16
	GapX    uint16
	GapY    uint16
}

func TextureAtlasFromGrid(opts GridOptions) TextureAtlas {
	// calculate number of layout
	count := opts.Count
	if count == 0 {
		count = uint32(opts.Width) * uint32(opts.Height)
	}

	var layout TextureAtlasLayout

	for n := range count {
		// clamp n into the valid range
		n = min(max(n, 0), count)

		row := n / uint32(opts.Columns)
		column := n % uint32(opts.Columns)

		x0 := column*uint32(opts.Width+opts.GapX) + uint32(opts.OffsetX)
		y0 := row*uint32(opts.Height+opts.GapY) + uint32(opts.OffsetY)
		x1 := x0 + uint32(opts.Width)
		y1 := y0 + uint32(opts.Height)

		layout = append(layout, wx.RectangleFromPoints(
			glm.Vec2u{x0, y0},
			glm.Vec2u{x1, y1},
		))
	}

	return TextureAtlas{Layout: layout}
}
