package main

import (
	_ "embed"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Draws sprites with custom, animated fragment shaders:
// a procedural fire, a waving texture and a texture that dissolves and grows back.
// The shaders read the time from the globals uniform.

//go:embed assets/effects.wgsl
var effectsShaderCode string

var (
	fireShader     = &ShaderDef{Source: effectsShaderCode, FragmentEntry: "fire"}
	waveShader     = &ShaderDef{Source: effectsShaderCode, FragmentEntry: "wave"}
	dissolveShader = &ShaderDef{Source: effectsShaderCode, FragmentEntry: "dissolve"}
)

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 60})
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(Camera{})

	marker := assets.Texture("marker.png").Await()
	rocket := assets.Texture("rocket.png").Await()

	commands.Spawn(
		TransformFromXY(-210, 0),
		// the fire shader does not read the texture
		Sprite{Texture: marker, CustomSize: Some(glm.Vec2f{160, 240})},
		CustomShader{Shader: fireShader},
	)

	commands.Spawn(
		TransformFromXY(0, 0),
		Sprite{Texture: marker, CustomSize: Some(glm.Vec2f{180, 180})},
		CustomShader{Shader: waveShader},
	)

	commands.Spawn(
		TransformFromXY(210, 0),
		Sprite{Texture: rocket, CustomSize: Some(glm.Vec2f{180, 180})},
		CustomShader{Shader: dissolveShader},
	)
}
