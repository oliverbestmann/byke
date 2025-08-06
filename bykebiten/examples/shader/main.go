package main

import (
	"embed"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
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
			item.Transform.Rotation += 0.05
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

	shader, err := ebiten.NewShader([]byte(shaderSrc))
	if err != nil {
		panic(err)
	}

	commands.Spawn(
		// Sprite{
		// 	Image:      assets.Image("ebiten.png").Await(),
		// 	CustomSize: Some(gm.VecSplat(100.0)),
		// },
		RegularPolygon(50, 8),
		NewTransform().WithRotation(gm.DegToRad(180)),
		Shader{Shader: shader},
		ShaderInput{
			Uniforms: map[string]any{
				"Scale": 1.0,
				"Alpha": 1.0,
			},
		},
	)
}

const shaderSrc = `//kage:unit pixels

package main

var Time float
var Scale float
var Alpha float
var Rotation float
var Transform mat2

func rand(n vec2) float {
	return fract(cos(dot(n, vec2(12.9898, 4.1414))) * 43758.5453)
}

func noise(n vec2) float {
	d := vec2(0.0, 1.0);
	b := floor(n)
	f := smoothstep(vec2(0.0), vec2(1.0), fract(n))
	return mix(mix(rand(b), rand(b + d.yx), f.x), mix(rand(b + d.xy), rand(b + d.yy), f.x), f.y);
}

func fbm(n vec2) float {
	total := 0.0
	amplitude := 1.0
	for i := 0; i < 4; i++ {
		total += noise(n) * amplitude
		n += n
		amplitude *= 0.5
	}
	
	return total
}

func Fragment(dp vec4, srcPos vec2, color vec4, uv vec4) vec4 {
	dstPos := dp
	dstPos.xy -= imageDstOrigin() + imageDstSize() * 0.5
	dstPos.xy = Transform * dstPos.xy
	dstPos.xy /= imageDstSize().y

	dstPos.xy *= Scale

	dstPos.xy = uv.xy;

	c1 := vec3(0.5, 0.0, 0.1)
	c2 := vec3(0.9, 0.0, 0.0)
	c3 := vec3(0.2, 0.0, 0.0)
	c4 := vec3(1.0, 0.9, 0.0)
	c5 := vec3(0.1)
	c6 := vec3(0.9)

	speed := vec2(0.7, 0.4);
	shift := 1.6;
	alpha := 1.0;

	t := Time

	p := dstPos.xy * 8.0
	q := fbm(p - t * 0.1);
	r := vec2(fbm(p + q + t * speed.x - p.x - p.y), fbm(p + q - t * speed.y))
	c := mix(c1, c2, fbm(p + r)) + mix(c3, c4, r.x) - mix(c5, c6, r.y)
	return vec4(c * cos(shift * dstPos.y), alpha) * Alpha
}

`
