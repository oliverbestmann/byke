//go:build js

package byke2d

import (
	"errors"
	"fmt"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

func fontscan_NewFontMap() fontmap {
	return fontmap{}
}

type fontmap struct {
	faces []*font.Face
}

func (fm *fontmap) ResolveFace(r rune) *font.Face {
	for _, face := range fm.faces {
		_, ok := fm.faces[0].NominalGlyph(r)
		if ok {
			return face
		}
	}

	return nil
}

func (fm *fontmap) AddFont(fontFile font.Resource, fileID, family string) error {
	loaders, err := opentype.NewLoaders(fontFile)
	if err != nil {
		return fmt.Errorf("unsupported font resource: %s", err)
	}

	// eagerly load the faces
	faces, err := font.ParseTTC(fontFile)
	if err != nil {
		return fmt.Errorf("unsupported font resource: %s", err)
	}

	if len(faces) != len(loaders) {
		return errors.New("internal error: inconsistent font descriptors and loader")
	}

	fm.faces = append(fm.faces, faces...)
	return nil
}
