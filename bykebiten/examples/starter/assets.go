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

//go:embed ducky.png
var ducky_png []byte

var AssetEbiten = sync.OnceValue(func() *ebiten.Image {
	image, _, _ := ebitenutil.NewImageFromReader(bytes.NewReader(ebiten_png))
	return image
})

var AssetDucky = sync.OnceValue(func() *ebiten.Image {
	image, _, _ := ebitenutil.NewImageFromReader(bytes.NewReader(ducky_png))
	return image
})
