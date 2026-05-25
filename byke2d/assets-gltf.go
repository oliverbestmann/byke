package byke2d

import (
	"fmt"
	"io"

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

	// nothing yet
	_ = settings

	h, err := gltf.Load(r)
	if err != nil {
		return nil, fmt.Errorf("load gltf: %w", err)
	}

	return h, nil
}

func (i GLTFLoader) Extensions() []string {
	return []string{".glb"}
}

func (a *Assets) GLTF(path string) AsyncAsset[*gltf.Handle] {
	return asTypedAsyncAsset[*gltf.Handle](a.Load(path))
}

func (a *Assets) GLTFWithSettings(path string, settings *LoadGLTFSettings) AsyncAsset[*gltf.Handle] {
	return asTypedAsyncAsset[*gltf.Handle](a.LoadWithSettings(path, settings))
}
