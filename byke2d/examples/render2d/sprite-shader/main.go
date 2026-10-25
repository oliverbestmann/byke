package main

import (
	_ "embed"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// Draws sprites with custom fragment shaders. The left sprite uses the default shader,
// the middle one adds noise and the right one fades out towards the edges.

//go:embed assets/noise.wgsl
var noisyShaderCode string

//go:embed assets/fade.wgsl
var fadeShaderCode string

var noisyShader = &ShaderDef{
	Source:        noisyShaderCode,
	FragmentEntry: "noisy_sprite",
}

var fadeShader = &ShaderDef{
	Source:        fadeShaderCode,
	FragmentEntry: "fade_sprite",
}

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(Camera{})

	texture := assets.TextureWithSettings("marker.png", &LoadTextureSettings{
		FilterMode: wgpu.FilterModeNearest,
	}).Await()

	sprite := Sprite{Texture: texture, CustomSize: Some(glm.Vec2f{160, 160})}

	commands.Spawn(TransformFromXY(-200, 0), sprite)
	commands.Spawn(TransformFromXY(0, 0), sprite, CustomShader{Shader: noisyShader})
	commands.Spawn(TransformFromXY(200, 0), sprite, CustomShader{Shader: fadeShader})
}
