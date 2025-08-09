package earcut

import (
	"embed"
	"encoding/json"
	"image/color"
	"io/fs"
	"strings"
	"testing"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/stretchr/testify/require"
)

//go:embed _testdata/*.json
var files embed.FS

func TestEarCutViewer(t *testing.T) {
	t.SkipNow()

	testCase := "water-huge2"

	fp, err := files.Open("_testdata/" + testCase + ".json")
	require.NoError(t, err)

	var data [][][2]float64
	err = json.NewDecoder(fp).Decode(&data)
	require.NoError(t, err)

	outer := pointsOf(data[0])

	var holes [][]Point
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

	points, indices := EarCut(outer, holes)

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
	Outer []Point
	Holes [][]Point
	G     ebiten.GeoM

	Vertices []Point
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

	vector.FillPath(screen, &triangles, color.NRGBA{R: 128, G: 255, B: 128, A: 50}, true, vector.FillRuleNonZero)

	v.drawPolygon(screen, toScreen, v.Outer, color.White)

	for _, hole := range v.Holes {
		v.drawPolygon(screen, toScreen, hole, color.NRGBA{R: 255, G: 128, B: 0, A: 128})
	}

	vector.StrokePath(screen, &triangles, color.NRGBA{255, 255, 255, 128}, true, &vector.StrokeOptions{
		Width: 1,
	})

}

func (v viewerGame) drawPolygon(screen *ebiten.Image, toScreen ebiten.GeoM, points []Point, color color.Color) {
	var outer vector.Path
	for _, point := range points {
		x, y := toScreen.Apply(point.X, point.Y)
		outer.LineTo(float32(x), float32(y))
	}

	outer.Close()

	vector.StrokePath(screen, &outer, color, true, &vector.StrokeOptions{
		Width: 2,
	})
}

func (v viewerGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func BenchmarkEarCut(b *testing.B) {
	buf, _ := fs.ReadFile(files, "_testdata/water-huge.json")

	var data [][][2]float64
	_ = json.Unmarshal(buf, &data)

	outer, holes := dataToPoints(data)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = EarCut(outer, holes)
	}
}

func TestEarCut(t *testing.T) {
	entries, _ := files.ReadDir("_testdata")
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".json")
		t.Run(name, func(t *testing.T) {
			fp, err := files.Open("_testdata/" + entry.Name())
			require.NoError(t, err)

			var data [][][2]float64
			err = json.NewDecoder(fp).Decode(&data)
			require.NoError(t, err)

			outer, holes := dataToPoints(data)

			_, indices := EarCut(outer, holes)

			expectedCount, ok := expectedTriangles[name]
			require.True(t, ok)

			require.Equal(t, expectedCount, len(indices)/3)
		})
	}
}

func dataToPoints(data [][][2]float64) ([]Point, [][]Point) {
	outer := pointsOf(data[0])

	var holes [][]Point
	for _, hole := range data[1:] {
		holes = append(holes, pointsOf(hole))
	}
	return outer, holes
}

func pointsOf(points [][2]float64) []Point {
	data := unsafe.SliceData(points)
	return unsafe.Slice((*Point)(unsafe.Pointer(data)), len(points))
}

var expectedTriangles = map[string]int{
	"building":             13,
	"dude":                 106,
	"water":                2482,
	"water2":               1212,
	"water3":               197,
	"water3b":              25,
	"water4":               705,
	"water-huge":           5176,
	"water-huge2":          4462,
	"degenerate":           0,
	"bad-hole":             42,
	"empty-square":         0,
	"issue16":              12,
	"issue17":              11,
	"steiner":              9,
	"issue29":              40,
	"issue34":              139,
	"issue35":              844,
	"self-touching":        124,
	"outside-ring":         64,
	"simplified-us-border": 120,
	"touching-holes":       57,
	"touching-holes2":      10,
	"touching-holes3":      82,
	"touching-holes4":      55,
	"touching-holes5":      133,
	"touching-holes6":      3098,
	"hole-touching-outer":  77,
	"hilbert":              1024,
	"issue45":              10,
	"eberly-3":             73,
	"eberly-6":             1429,
	"issue52":              109,
	"shared-points":        4,
	"bad-diagonals":        7,
	"issue83":              0,
	"issue107":             0,
	"issue111":             18,
	"boxy":                 58,
	"collinear-diagonal":   14,
	"issue119":             18,
	"hourglass":            2,
	"touching2":            8,
	"touching3":            15,
	"touching4":            19,
	"rain":                 2681,
	"issue131":             12,
	"infinite-loop-jhl":    0,
	"filtered-bridge-jhl":  25,
	"issue149":             2,
	"issue142":             4,
	"issue186":             41,
}
