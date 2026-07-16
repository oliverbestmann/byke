package byke2d

import (
	"bytes"
	_ "embed"
	"fmt"
	"sync"

	"github.com/go-text/typesetting/shaping"
)

type Faces shaping.Fontmap

//go:embed fonts/NotoSans-VariableFont_wdth,wght.ttf
var notoSans []byte

//go:embed fonts/NotoSansJP-VariableFont_wght.ttf
var notoSansJapanese []byte

//go:embed fonts/NotoSansMono-Regular.ttf
var notoSansMono []byte

var DefaultFontSans = sync.OnceValue(func() Font {
	return Font{Faces: ParseFontFaces(notoSans, notoSansJapanese)}
})

var DefaultFontMono = sync.OnceValue(func() Font {
	return Font{Faces: ParseFontFaces(notoSansMono)}
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
