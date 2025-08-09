package main

import (
	"embed"
	_ "embed"
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
	. "github.com/oliverbestmann/byke/gm"
)

//go:embed assets/*
var assets embed.FS

func main() {
	var app App

	app.InsertResource(MakeAssetFS(assets))

	app.AddPlugin(GamePlugin)

	// optional: configure the game window
	app.InsertResource(WindowConfig{
		Title:  "Example",
		Width:  800,
		Height: 600,
	})

	app.AddSystems(Startup, startupSystem)
	app.AddSystems(Update, updateTransform)

	fmt.Println(app.Run())
}

type CameraView struct {
	Component[CameraView]
}

type Player struct {
	Component[Player]
}

func startupSystem(commands *Commands) {
	var rect Path
	rect.Rectangle(RectWithCenterAndSize(VecZero, VecSplat(5.0)))
	rect.Rectangle(RectWithOriginAndSize(VecOf(0.0, -0.5), VecOf(5.0, 1.0)))

	commands.Spawn(
		Camera{},
		OrthographicProjection{
			Scale:          1,
			ViewportOrigin: VecSplat(0.5),
			// ScalingMode:    ScalingModeFixed{Viewport: VecSplat(100.0)},
			ScalingMode: ScalingModeAutoMin{MinWidth: 100, MinHeight: 100},
			// ScalingMode: ScalingModeWindowSize{},
		},
		// NewTransform().WithTranslation(VecSplat(20.0)),
	)

	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	whiteImage := img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	commands.Spawn(
		AnchorTopLeft,
		Sprite{
			Image:      whiteImage,
			CustomSize: Some(VecSplat(10.0)),
		},
	)

	commands.Spawn(
		AnchorCenter,
		Player{},
		rect,
		Fill{Color: color.RGB(1, 0, 1)},
		Layer{Z: 1},
	)

	commands.Spawn(
		AnchorBottomRight,
		Sprite{
			Image:      whiteImage,
			CustomSize: Some(VecSplat(10.0)),
		},
		ColorTint{
			Color: color.RGB(1, 0, 0),
		},
	)

	commands.Spawn(
		AnchorCenter,
		TransformFromXY(-50, 0),
		Sprite{
			Image:      whiteImage,
			CustomSize: Some(VecSplat(2.0)),
		},
		SpawnChild(
			Text{Text: "-50"},
			NewTransform().WithScale(VecSplat(0.2)).WithTranslation(Vec{Y: 3}),
		),
	)

	commands.Spawn(
		AnchorCenter,
		TransformFromXY(50, 0),
		Sprite{
			Image:      whiteImage,
			CustomSize: Some(VecSplat(2.0)),
		},
		SpawnChild(
			Text{Text: "+50"},
			NewTransform().WithScale(VecSplat(0.2)).WithTranslation(Vec{Y: 3}),
		),
	)

	commands.Spawn(
		AnchorCenter,
		CameraView{},
		Sprite{
			Image:      whiteImage,
			CustomSize: Some(VecSplat(100.0)),
		},
		ColorTint{Color: color.RGBA(1, 1, 1, 0.1)},
	)
}

func updateTransform(vt VirtualTime, keys Keys,
	query Query[struct {
		Camera     Camera
		Projection *OrthographicProjection
		Transform  *Transform
	}],
	players Query[struct {
		_         With[Player]
		Transform *Transform
	}],
	cameraViewQuery Query[struct {
		_         With[CameraView]
		Transform *Transform
	}],
) {
	player := players.MustFirst()

	if keys.IsPressed(ebiten.KeyArrowUp) {
		dir := RotationMat(player.Transform.Rotation).Transform(Vec{X: 1.0})
		player.Transform.Translation = player.Transform.Translation.Add(dir.Mul(50 * vt.DeltaSecs))
	}
	if keys.IsPressed(ebiten.KeyArrowLeft) {
		player.Transform.Rotation -= Rad(5 * vt.DeltaSecs)
	}
	if keys.IsPressed(ebiten.KeyArrowRight) {
		player.Transform.Rotation += Rad(5 * vt.DeltaSecs)
	}

	for item := range query.Items() {
		item.Transform.Rotation = player.Transform.Rotation
		item.Transform.Translation = player.Transform.Translation

		if keys.IsPressed(ebiten.KeyS) {
			item.Projection.Scale += 0.5 * vt.DeltaSecs
		}

		if keys.IsPressed(ebiten.KeyA) {
			item.Projection.Scale = max(0.1, item.Projection.Scale-0.5*vt.DeltaSecs)
		}

		for view := range cameraViewQuery.Items() {
			*view.Transform = *item.Transform
			view.Transform.Rotation = item.Transform.Rotation
			view.Transform.Scale = VecSplat(item.Projection.Scale)
		}
	}

}
