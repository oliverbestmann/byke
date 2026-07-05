package byke2d

import (
	"fmt"
	"io"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/oliverbestmann/byke/byke2d/gltf"
)

type LoadGLTFSettings struct {
}

func (*LoadGLTFSettings) IsLoadSettings() {}

type GLTFLoader struct{}

func (i GLTFLoader) Load(ctx LoadContext, r io.ReadSeekCloser) (any, error) {
	defer func() { _ = r.Close() }()

	var settings LoadGLTFSettings
	if ctx.Settings != nil {
		settings = *ctx.Settings.(*LoadGLTFSettings)
	}

	// get the assets fs from the world
	assets := ctx.World.RequireResourceOf[AssetFS]()

	// nothing yet
	_ = settings

	switch {
	case strings.HasSuffix(strings.ToLower(ctx.Path), ".glb"):
		h, err := gltf.GLB(r)
		if err != nil {
			return nil, fmt.Errorf("load glb: %w", err)
		}

		return h, nil

	case strings.HasSuffix(strings.ToLower(ctx.Path), ".gltf"):
		h, err := gltf.GLTF(assets, r)
		if err != nil {
			return nil, fmt.Errorf("load gltf: %w", err)
		}

		return h, nil

	default:
		panic("unreachable")
	}
}

func (i GLTFLoader) Extensions() []string {
	return []string{".glb", ".gltf"}
}

func (a *Assets) GLTF(path string) AsyncAsset[*gltf.Handle] {
	return asTypedAsyncAsset[*gltf.Handle](a.Load(path))
}

func (a *Assets) GLTFWithSettings(path string, settings *LoadGLTFSettings) AsyncAsset[*gltf.Handle] {
	return asTypedAsyncAsset[*gltf.Handle](a.LoadWithSettings(path, settings))
}
