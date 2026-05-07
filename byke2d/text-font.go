package byke2d

import (
	"bytes"
	_ "embed"
	"sync"

	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/shaping"
)

type Faces shaping.Fontmap

//go:embed fonts/NotoSans-VariableFont_wdth,wght.ttf
var fontNormal []byte

//go:embed fonts/NotoSansJP-VariableFont_wght.ttf
var fontJapanese []byte

var DefaultFont = sync.OnceValue(func() Font {
	return Font{Faces: loadDefaultFont()}
})

func loadDefaultFont() Faces {
	fm := fontscan.NewFontMap(nil)

	if err := fm.AddFont(bytes.NewReader(fontNormal), "NotoSans.ttf", ""); err != nil {
		panic(err)
	}

	if err := fm.AddFont(bytes.NewReader(fontJapanese), "NotoSansJP.ttf", ""); err != nil {
		panic(err)
	}

	return fm
}
