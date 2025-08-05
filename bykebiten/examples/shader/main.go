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
		Sprite{
			Image:      assets.Image("ebiten.png").Await(),
			CustomSize: Some(gm.VecSplat(100.0)),
		},
		Shader{Shader: shader},
	)
}

const shaderSrc = `//kage:unit pixels

package main

// Uniform variables.
var Time float
var Cursor vec2

// Fragment is the entry point of the fragment shader.
// Fragment returns the color value for the current position.
func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// You can define variables with a short variable declaration like Go.
	// pos := dstPos.xy - imageDstOrigin()
	// 
	// lightpos := vec3(Cursor, 50)
	// lightdir := normalize(lightpos - vec3(pos, 0))
	// normal := normalize(imageSrc1UnsafeAt(srcPos) - 0.5)
	// const ambient = 0.25
	// diffuse := 0.75 * max(0.0, dot(normal.xyz, lightdir))
	// 
	// // You can treat multiple source images by
	// // imageSrc[N]At or imageSrc[N]UnsafeAt.
	// return imageSrc0UnsafeAt(srcPos) * (ambient + diffuse)

	// pos := (srcPos.xy - imageSrc0Origin()) 
	// pos1 := pos / imageSrc0Size()

	px := imageSrc0UnsafeAt(srcPos.xy)

	return px * color
}
`
