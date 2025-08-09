package main

import (
	"embed"
	"fmt"
	"math"
	"math/rand/v2"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/gm"
)

//go:embed assets
var assets embed.FS

func main() {
	var app App
	app.InsertResource(MakeAssetFS(assets))
	app.AddPlugin(GamePlugin)
	app.AddSystems(Startup, startupSystem)
	app.AddSystems(Update, func(q Query[struct {
		_         With[Shader]
		Transform *Transform
	}]) {
		for item := range q.Items() {
			item.Transform.Rotation += 0.01
		}
	})
	fmt.Println(app.Run())
}

func startupSystem(
	commands *Commands,
	assets *Assets,
) {
	commands.Spawn(
		Camera{},
		OrthographicProjection{
			ViewportOrigin: gm.VecSplat(0.5),
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 200.0},
			Scale:          1,
		},
	)

	var points []gm.Vec
	var hole []gm.Vec

	for r := gm.Rad(0); r < gm.Rad(math.Pi*2); r += 0.1 {
		vec := gm.Vec{X: rand.Float64()*25 + 25}
		points = append(points, vec.Rotated(r))

		vec = gm.Vec{X: rand.Float64()*10 + 10}
		hole = append(hole, vec.Rotated(r))
	}

	polygon := Polygon(points, hole)

	polygon.ComputeUV(func(point gm.Vec) gm.Vec {
		return point.Mul(1 / 50.0)
	})

	commands.Spawn(
		// Sprite{
		// 	Image:      assets.Image("ebiten.png").Await(),
		// 	CustomSize: Some(gm.VecSplat(100.0)),
		// },
		polygon,
		NewTransform().WithRotation(gm.DegToRad(180)),
		assets.Shader("fire.kage").Await(),
		ShaderInput{
			Uniforms: map[string]any{
				"Scale": 1.0,
				"Alpha": 1.0,
			},
		},
	)
}
