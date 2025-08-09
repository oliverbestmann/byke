package bykebiten

import (
	"fmt"
	"io"

	"github.com/hajimehoshi/ebiten/v2"
)

type ShaderAssetLoader struct{}

func (l ShaderAssetLoader) Load(ctx LoadContext, r io.ReadSeekCloser) (any, error) {
	defer func() { _ = r.Close() }()

	source, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read shader source: %w", err)
	}

	shader, err := ebiten.NewShader(source)
	if err != nil {
		return nil, fmt.Errorf("compile shader: %w", err)
	}

	return Shader{Shader: shader}, nil
}

func (l ShaderAssetLoader) Extensions() []string {
	return []string{".kage", ".go"}
}
