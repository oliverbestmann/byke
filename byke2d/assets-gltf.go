package byke2d

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"reflect"
	"strings"

	"github.com/oliverbestmann/byke/byke2d/gltf"
)

type LoadGLTFSettings struct{}

func (*LoadGLTFSettings) IsLoadSettings() {}

type GLTFLoader struct{}

func (i GLTFLoader) Type() reflect.Type {
	return reflect.TypeFor[*gltf.Handle]()
}

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

		h.Dir = path.Dir(ctx.Path)

		return h, nil

	case strings.HasSuffix(strings.ToLower(ctx.Path), ".gltf"):
		// uris in the gltf file are relative to the file itself
		dir, err := fs.Sub(assets, path.Dir(ctx.Path))
		if err != nil {
			return nil, fmt.Errorf("load gltf: %w", err)
		}

		h, err := gltf.GLTF(dir, r)
		if err != nil {
			return nil, fmt.Errorf("load gltf: %w", err)
		}

		h.Dir = path.Dir(ctx.Path)

		return h, nil

	default:
		panic("unreachable")
	}
}

func (i GLTFLoader) Extensions() []string {
	return []string{".glb", ".gltf"}
}
