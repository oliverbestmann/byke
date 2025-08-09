package earcut

import (
	"embed"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

//go:embed viewer/testdata/*.json
var files embed.FS

func BenchmarkEarCut(b *testing.B) {
	buf, _ := fs.ReadFile(files, "viewer/testdata/water-huge.json")

	var data [][][2]float64
	_ = json.Unmarshal(buf, &data)

	outer, holes := dataToPoints(data)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = Triangulate(outer, holes)
	}
}

func TestEarCut(t *testing.T) {
	entries, _ := files.ReadDir("viewer/testdata")
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".json")
		t.Run(name, func(t *testing.T) {
			fp, err := files.Open("viewer/testdata/" + entry.Name())
			require.NoError(t, err)

			var data [][][2]float64
			err = json.NewDecoder(fp).Decode(&data)
			require.NoError(t, err)

			outer, holes := dataToPoints(data)

			_, indices := Triangulate(outer, holes)

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
