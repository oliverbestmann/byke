package main

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/bykebiten/internal/earcut"
	"github.com/oliverbestmann/byke/gm"
)

func main() {
	var app App
	app.AddPlugin(GamePlugin)
	app.AddSystems(Startup, startupSystem)
	fmt.Println(app.Run())
}

func startupSystem(
	commands *Commands,
) {
	commands.Spawn(
		Camera{},
		OrthographicProjection{
			ViewportOrigin: gm.VecSplat(0.5),
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 200.0},
			Scale:          1,
		},
	)

	var outer []gm.Vec
	var hole []gm.Vec

	for r := gm.Rad(0); r < gm.Rad(math.Pi*2); r += 0.1 {
		vec := gm.Vec{X: rand.Float64()*25 + 25}
		outer = append(outer, vec.Rotated(r))

		vec = gm.Vec{X: rand.Float64()*10 + 10}
		hole = append(hole, vec.Rotated(r))
	}

	pp, indices := earcut.EarCut(outer, [][]earcut.Point{hole})
	polygon := Mesh{
		Indices: indices,
	}

	for _, point := range pp {
		polygon.Vertices = append(polygon.Vertices, ebiten.Vertex{
			DstX:   float32(point.X),
			DstY:   float32(point.Y),
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		})
	}

	commands.Spawn(
		// Sprite{
		// 	Image:      assets.Image("ebiten.png").Await(),
		// 	CustomSize: Some(gm.VecSplat(100.0)),
		// },
		polygon,
		ColorTint{Color: color.RGBA(1, 1, 1, 0.5)},
	)

	var path Path
	for idx := 0; idx < len(indices); idx += 3 {
		path.MoveTo(pp[indices[idx]])
		path.LineTo(pp[indices[idx+1]])
		path.LineTo(pp[indices[idx+2]])
		path.LineTo(pp[indices[idx]])
	}

	commands.Spawn(
		path,
		Layer{Z: 1},
		Stroke{Color: color.RGBA(0, 1, 0, 1), Width: 1.0},
	)

	var pathOriginal Path
	pathOriginal.LineStrip(outer)
	pathOriginal.Close()

	commands.Spawn(
		pathOriginal,
		Stroke{Color: color.RGBA(1, 0, 0, 1), Width: 2.0},
	)

	var pathHole Path
	pathHole.LineStrip(hole)
	pathHole.Close()

	commands.Spawn(
		pathHole,
		Stroke{Color: color.RGBA(0, 0, 1, 1), Width: 2.0},
	)
}
