package byke2d

import (
	"bytes"
	_ "embed"
	"fmt"
	"sync"

	"github.com/go-text/typesetting/shaping"
)

type Faces shaping.Fontmap

//go:embed fonts/AtkinsonHyperlegible-Regular.ttf
var fontSans []byte

//go:embed fonts/SpaceMono-Regular.ttf
var fontMonoSpace []byte

var DefaultFontSans = sync.OnceValue(func() Font {
	return Font{Faces: ParseFontFaces(fontSans)}
})

var DefaultFontMono = sync.OnceValue(func() Font {
	return Font{Faces: ParseFontFaces(fontMonoSpace)}
})

func TryParseFontFaces(bufs ...[]byte) (Faces, error) {
	fm := fontscan_NewFontMap()

	for _, buf := range bufs {
		if err := fm.AddFont(bytes.NewReader(buf), "", ""); err != nil {
			return nil, fmt.Errorf("parsing font: %w", err)
		}
	}

	return fm, nil
}

func ParseFontFaces(bufs ...[]byte) Faces {
	faces, err := TryParseFontFaces(bufs...)
	if err != nil {
		panic(err)
	}

	return faces
}
