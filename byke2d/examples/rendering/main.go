package main

import (
	"embed"
	"log/slog"
	"math/rand/v2"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed assets
var assets embed.FS

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	slog.SetDefault(slog.New(handler))

	var app App

	// configure assets before loading the plugin
	app.InsertResource(MakeAssetFS(assets))

	app.AddPlugin(PluginRender)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, moveSprites)
	app.AddSystems(Update, animateSprite)
	app.AddSystems(Update, System(handleInputSystem).After(animateSprite))

	app.MustRun()
}

type Player struct {
	Component[Player]
	Direction int
	Blocked   bool
}

type Animate struct {
	Component[Animate]
	Timer
}

type Velocity struct {
	Component[Velocity]
	glm.Vec2f
}

var layoutIdle = TextureAtlasLayoutFromGrid(GridOptions{
	Count:  13,
	Width:  32,
	Height: 32,
})

var layoutWalk = TextureAtlasLayoutFromGrid(GridOptions{
	StartRow: 1,
	Count:    8,
	Width:    32,
	Height:   32,
})

var layoutHurt = TextureAtlasLayoutFromGrid(GridOptions{
	StartRow: 6,
	Count:    4,
	Width:    32,
	Height:   32,
})

var layoutDeath = TextureAtlasLayoutFromGrid(GridOptions{
	StartRow: 7,
	Count:    7,
	Width:    32,
	Height:   32,
})

var layoutAttack = []TextureAtlasLayout{
	TextureAtlasLayoutFromGrid(GridOptions{
		StartRow: 2,
		Count:    10,
		Width:    32,
		Height:   32,
	}),
	TextureAtlasLayoutFromGrid(GridOptions{
		StartRow: 3,
		Count:    10,
		Width:    32,
		Height:   32,
	}),
	TextureAtlasLayoutFromGrid(GridOptions{
		StartRow: 4,
		Count:    10,
		Width:    32,
		Height:   32,
	}),
}

func setupSystem(ctx *RenderContext, commands *Commands, assets *Assets) {
	asset := assets.Texture("circle.png").Await()

	nnSettings := &LoadTextureSettings{
		FilterMode: wgpu.FilterModeNearest,
	}

	figure := assets.TextureWithSettings("figure.png", nnSettings).Await()

	cameraTexture := NewTexture2d(ctx, NewTexture2dOptions{
		Label:      "PixelCamera",
		Width:      360,
		Height:     200,
		Format:     wgpu.TextureFormatBGRA8UnormSrgb,
		FilterMode: wgpu.FilterModeNearest,
	})
	commands.Spawn(
		Camera{},
		RenderTarget{Texture: AsRenderTexture(cameraTexture)},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 360},
		},
	)

	commands.Spawn(
		RenderLayersOf(1),
		Camera{},
		ClearColor{Color: ColorBlack},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 360},
		},
	)

	commands.Spawn(
		TransformFromXY(0, 0),
		RenderLayersOf(1),
		Sprite{
			Texture: cameraTexture,
		},
	)

	commands.Spawn(
		TransformFromXYZ(30, 0, 1).WithScaleXY(1, 1),
		Player{},
		Sprite{Texture: figure},
		Animate{Timer: NewTimerWithFrequency(10.0)},
		TextureAtlas{Layout: layoutIdle},
	)

	for range 500 {
		x := (rand.Float32() - 0.5) * 1200
		y := (rand.Float32() - 0.5) * 1200

		vx := (rand.Float32() - 0.5) * 5
		vy := (rand.Float32() - 0.5) * 5

		size := rand.Float32()*32 + 16
		alpha := rand.Float32()*0.1 + 0.05

		commands.Spawn(
			TransformFromXY(x, y),
			Velocity{Vec2f: glm.Vec2f{vx, vy}},
			Sprite{
				Texture:    asset,
				Color:      ColorSRGBA(1, 1, 1, alpha),
				CustomSize: Some(glm.Vec2f{size, size}),
			},
		)
	}
}

func moveSprites(vt VirtualTime, query Query[struct {
	_         With[Sprite]
	Transform *Transform
	Velocity  Velocity
}],
) {
	for item := range query.Items() {
		vel := item.Velocity.Scale(vt.DeltaSecs)
		newValue := item.Transform.Translation.Truncate().Add(vel)
		item.Transform.Translation[0] = newValue[0]
		item.Transform.Translation[1] = newValue[1]
	}
}

func animateSprite(vt *VirtualTime, query Query[struct {
	Animation    *Animate
	TextureAtlas *TextureAtlas
}],
) {
	for item := range query.Items() {
		if item.Animation.Tick(vt.Delta).JustFinished() {
			item.TextureAtlas.Index += 1
		}
	}
}

func handleInputSystem(keys Keys, player Single[struct {
	Player       *Player
	Sprite       *Sprite
	TextureAtlas *TextureAtlas
}],
) {
	var direction int
	var attack, hurt bool

	if player.Value.Player.Blocked {
		if player.Value.TextureAtlas.IsValid() {
			return
		}
	}

	if keys.IsPressed(vyn.KeyArrowLeft) {
		direction -= 1
	}

	if keys.IsPressed(vyn.KeyArrowRight) {
		direction += 1
	}

	if keys.IsJustPressed(vyn.KeySpace) {
		direction = 0
		attack = true
	}

	if keys.IsJustPressed(vyn.KeyH) {
		hurt = true
	}

	player.Value.Player.Blocked = false

	if player.Value.Player.Direction != direction {
		player.Value.TextureAtlas.Index = 0
		player.Value.Player.Direction = direction
	}

	player.Value.TextureAtlas.Wrapping = true

	switch {
	case attack:
		layout := layoutAttack[rand.IntN(len(layoutAttack))]
		player.Value.TextureAtlas.Layout = layout
		player.Value.TextureAtlas.Wrapping = false
		player.Value.TextureAtlas.Index = 0
		player.Value.Player.Blocked = true

	case hurt:
		player.Value.TextureAtlas.Layout = layoutHurt
		player.Value.TextureAtlas.Wrapping = false
		player.Value.TextureAtlas.Index = 0
		player.Value.Player.Blocked = true

	case direction == 0:
		player.Value.TextureAtlas.Layout = layoutIdle

	case direction == 1:
		player.Value.TextureAtlas.Layout = layoutWalk

	case direction == -1:
		player.Value.TextureAtlas.Layout = layoutWalk
	}

	switch {
	case direction == 1:
		player.Value.Sprite.FlipX = false

	case direction == -1:
		player.Value.Sprite.FlipX = true
	}
}
