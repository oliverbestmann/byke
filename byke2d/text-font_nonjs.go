//go:build !js

package byke2d

import (
	"github.com/go-text/typesetting/fontscan"
)

func fontscan_NewFontMap() *fontscan.FontMap {
	return fontscan.NewFontMap(nil)
}
