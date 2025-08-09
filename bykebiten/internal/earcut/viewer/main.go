package main

import (
	"embed"
	"encoding/json"
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

	points, indices := earcut.Triangulate(outer, holes)

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.RunGame(viewerGame{
		G:     g,
		Outer: outer,
		Holes: holes,

		Vertices: points,
		Indices:  indices,
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

	// draw triangulation
	var triangles vector.Path
	for idx := 0; idx < len(v.Indices); idx += 3 {
		i0, i1, i2 := v.Indices[idx], v.Indices[idx+1], v.Indices[idx+2]
		v0, v1, v2 := v.Vertices[i0], v.Vertices[i1], v.Vertices[i2]

		x0, y0 := toScreen.Apply(v0.XY())
		x1, y1 := toScreen.Apply(v1.XY())
		x2, y2 := toScreen.Apply(v2.XY())

		triangles.MoveTo(float32(x0), float32(y0))
		triangles.LineTo(float32(x1), float32(y1))
		triangles.LineTo(float32(x2), float32(y2))
		triangles.LineTo(float32(x0), float32(y0))
	}

	dop := &vector.DrawPathOptions{}
	dop.ColorScale.ScaleWithColor(color.NRGBA{R: 128, G: 255, B: 128, A: 50})
	vector.FillPath(screen, &triangles, nil, dop)

	v.drawPolygon(screen, toScreen, v.Outer, color.White)

	for _, hole := range v.Holes {
		v.drawPolygon(screen, toScreen, hole, color.NRGBA{R: 255, G: 128, B: 0, A: 128})
	}

	dop = &vector.DrawPathOptions{}
	dop.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 128})
	vector.StrokePath(screen, &triangles, &vector.StrokeOptions{Width: 1}, dop)
}

func (v viewerGame) drawPolygon(screen *ebiten.Image, toScreen ebiten.GeoM, points []earcut.Point, color color.Color) {
	outer := v.pathOf(points, toScreen)

	dop := &vector.DrawPathOptions{}
	dop.ColorScale.ScaleWithColor(color)
	vector.StrokePath(screen, &outer, &vector.StrokeOptions{Width: 2}, dop)
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

func dataToPoints(data [][][2]float64) ([]earcut.Point, [][]earcut.Point) {
	outer := pointsOf(data[0])

	var holes [][]earcut.Point
	for _, hole := range data[1:] {
		holes = append(holes, pointsOf(hole))
	}
	return outer, holes
}

func pointsOf(points [][2]float64) []earcut.Point {
	data := unsafe.SliceData(points)
	return unsafe.Slice((*earcut.Point)(unsafe.Pointer(data)), len(points))
}
