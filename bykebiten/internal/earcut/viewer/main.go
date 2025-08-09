package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"image/color"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/oliverbestmann/byke/bykebiten/internal/earcut"
)

//go:embed testdata/*.json
var files embed.FS

func main() {
	testCase := "water-huge"

	fp, _ := files.Open("testdata/" + testCase + ".json")

	var data [][][2]float64
	_ = json.NewDecoder(fp).Decode(&data)

	outer := pointsOf(data[0])

	var holes [][]earcut.Point
	for _, hole := range data[1:] {
		holes = append(holes, pointsOf(hole))
	}

	// calculate bounding box of the outer polygon
	minX, minY := outer[0].XY()
	maxX, maxY := outer[0].XY()
	for _, point := range outer {
		minX = min(minX, point.X)
		maxX = max(maxX, point.X)
		minY = min(minY, point.Y)
		maxY = max(maxY, point.Y)
	}

	var g ebiten.GeoM
	g.Translate(-minX, -minY)
	g.Scale(1/(maxX-minX), 1/(maxY-minY))

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.RunGame(viewerGame{
		G:     g,
		Outer: outer,
		Holes: holes,
	})
}

type viewerGame struct {
	Outer []earcut.Point
	Holes [][]earcut.Point
	G     ebiten.GeoM

	Vertices []earcut.Point
	Indices  []uint32
}

func (v viewerGame) Update() error {
	return nil
}

func (v viewerGame) Draw(screen *ebiten.Image) {
	sw, sh := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())

	toScreen := v.G
	toScreen.Scale(0.9*sw, 0.9*sh)
	toScreen.Translate(0.05*sw, 0.05*sh)

	// draw the white line
	v.drawPolygon(screen, toScreen, v.Outer, color.White)

	var maxW, maxH int
	for _, hole := range v.Holes {
		// just for measuring
		h := v.pathOf(hole, toScreen)
		maxW = max(maxW, h.Bounds().Dx())
		maxH = max(maxH, h.Bounds().Dy())

		// draw pink lines
		v.drawPolygon(screen, toScreen, hole, color.NRGBA{R: 255, G: 0, B: 255, A: 255})
	}

	fmt.Println("Max hole size", maxW, maxH)
}

func (v viewerGame) drawPolygon(screen *ebiten.Image, toScreen ebiten.GeoM, points []earcut.Point, color color.Color) {
	outer := v.pathOf(points, toScreen)

	dop := vector.DrawPathOptions{}
	dop.ColorScale.ScaleWithColor(color)

	vector.StrokePath(screen, &outer, &vector.StrokeOptions{Width: 1}, &dop)
}

func (v viewerGame) pathOf(points []earcut.Point, toScreen ebiten.GeoM) vector.Path {
	var outer vector.Path
	for _, point := range points {
		x, y := toScreen.Apply(point.X, point.Y)
		outer.LineTo(float32(x), float32(y))
	}

	outer.Close()
	return outer
}

func (v viewerGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func pointsOf(points [][2]float64) []earcut.Point {
	data := unsafe.SliceData(points)
	return unsafe.Slice((*earcut.Point)(unsafe.Pointer(data)), len(points))
}
