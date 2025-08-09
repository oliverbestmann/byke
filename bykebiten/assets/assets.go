package assets

import (
	"bytes"
	_ "embed"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed FiraMono-subset.ttf
var firamono_subset_ttf []byte

var FiraMono = sync.OnceValue(func() *text.GoTextFaceSource {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(firamono_subset_ttf))
	if err != nil {
		panic(err)
	}

	return font
})
