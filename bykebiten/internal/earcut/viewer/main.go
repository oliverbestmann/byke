package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func main() {
	ebiten.RunGame(viewerGame{})
}

type viewerGame struct{}

func (v viewerGame) Draw(screen *ebiten.Image) {
	x0, y0, x1, y1 := 691.1673640167942, 187.84811580883337, 690.519293351987, 187.62247242648044

	var path vector.Path
	path.LineTo(float32(x0), float32(y0))
	path.LineTo(float32(x1), float32(y1))
	path.Close()

	vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: 1}, nil)
}

func (v viewerGame) Update() error {
	return nil
}

func (v viewerGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}
