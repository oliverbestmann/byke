//go:build !js

package byke2d

import (
	"bytes"

	"github.com/go-text/typesetting/fontscan"
)

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
