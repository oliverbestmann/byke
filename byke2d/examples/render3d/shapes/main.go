package main

import (
	"image"
	"slices"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// Adapted from https://bevy.org/examples/3d-rendering/3d-shapes/

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginRotatable)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 100})
}

func setupSystem(commands *Commands, ctx *RenderContext) {
	material := StandardMaterial{
		BaseColor:           ColorSRGB(1, 1, 1),
		PerceptualRoughness: 0.5,
		StandardMaterialBindGroup: StandardMaterialBindGroup{
			Texture: uvDebugTexture(ctx),
			// the flat shapes have only one side
			MaterialValues: MaterialValues{DoubleSided: true},
		},
	}

	shapes := []*Mesh{
		Cube(),
		CubeWithSize(glm.Vec3f{1.5, 0.75, 0.75}),
		Sphere(0.6, 32, 16),
		Sphere(0.6, 8, 6),
		Circle(0.6, 32),
		RegularPolygon(0.6, 6),
		Rectangle(glm.Vec2f{1.2, 0.8}),
	}

	const extent = 12

	for idx, shape := range shapes {
		x := -extent/2 + float32(idx)/float32(len(shapes)-1)*extent

		commands.Spawn(
			TransformFromXYZ(x, 2, 0).WithRotationX(-glm.DegToRad(45)),
			MeshInstance{Mesh: shape},
			material,
			shared.RotatingY{Speed: 0.5},
		)
	}

	// ground plane
	commands.Spawn(
		MeshInstance{Mesh: PlaneWithSize(glm.Vec2f{50, 50})},
		StandardMaterial{BaseColor: ColorSRGB(0.75, 0.75, 0.75), PerceptualRoughness: 0.5},
	)

	commands.Spawn(
		TransformFromXYZ(8, 16, 8),
		PointLight{
			Color:     ColorLinearRGB(1, 1, 1),
			Intensity: 2_000_000,
			Range:     100,

			ShadowsEnabled:   true,
			ShadowDepthBias:  0.2,
			ShadowNormalBias: DefaultPointLight.ShadowNormalBias,
		},
	)

	commands.Spawn(
		Camera3d,
		Camera{},
		HDR{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(0, 7, 14).LookingAt(glm.Vec3f{0, 1, 0}, glm.Vec3f{0, 1, 0}),
	)
}

// uvDebugTexture creates a small colorful texture that shows how uvs map onto a mesh.
func uvDebugTexture(ctx *RenderContext) *Texture {
	const size = 8

	palette := []byte{
		255, 102, 159, 255,
		255, 159, 102, 255,
		236, 255, 102, 255,
		121, 255, 102, 255,
		102, 255, 198, 255,
		102, 198, 255, 255,
		121, 102, 255, 255,
		236, 102, 255, 255,
	}

	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		copy(img.Pix[y*img.Stride:], palette)

		// shift the palette by one pixel for the next row
		palette = slices.Concat(palette[len(palette)-4:], palette[:len(palette)-4])
	}

	return NewTextureFromImage(ctx, img, TextureFromImageOptions{
		Label:         "uv debug texture",
		SamplerConfig: SamplerConfig{FilterMode: wgpu.FilterModeNearest},
	})
}
