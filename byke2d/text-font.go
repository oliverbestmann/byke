package byke2d

import (
	_ "embed"
	"sync"

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
