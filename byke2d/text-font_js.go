//go:build js

package byke2d

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

func loadDefaultFont() Faces {
	var fm fontmap

	if err := fm.AddFont(bytes.NewReader(fontNormal)); err != nil {
		panic(err)
	}

	if err := fm.AddFont(bytes.NewReader(fontJapanese)); err != nil {
		panic(err)
	}

	return &fm
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

func (fm *fontmap) AddFont(fontFile font.Resource) error {
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
