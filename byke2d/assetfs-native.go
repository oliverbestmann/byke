//go:build !js

package byke2d

import (
	"log/slog"
	"os"
	"path/filepath"
)

func initializeAssetFS() AssetFS {
	var dir = "."

	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}

	slog.Info("Configure assets directory", slog.String("path", dir))
	return MakeSubAssetFS(os.DirFS(dir))
}
