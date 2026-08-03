//go:build js

package byke2d

import (
	"net/url"
	"syscall/js"

	"github.com/oliverbestmann/byke/byke2d/httpfs"
)

func initializeAssetFS() AssetFS {
	base, err := url.Parse(js.Global().Get("location").Get("href").String())
	if err != nil {
		panic("failed to parse location.href")
	}

	assets := base.JoinPath("assets/")
	return MakeAssetFS(httpfs.New(assets))
}
