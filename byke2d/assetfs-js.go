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

	// resolve like a relative link: drops the page name (index.html) and the query
	assets := base.ResolveReference(&url.URL{Path: "assets/"})
	return MakeAssetFS(httpfs.New(assets))
}
