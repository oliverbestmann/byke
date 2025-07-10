package main

import (
	"bytes"
	_ "embed"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"sync"

	_ "image/png"
)

//go:embed ebiten.png
var ebiten_png []byte

var EbitenPNG = sync.OnceValue(func() *ebiten.Image {
	image, _, _ := ebitenutil.NewImageFromReader(bytes.NewReader(ebiten_png))
	return image
})
