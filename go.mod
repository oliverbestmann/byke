module github.com/oliverbestmann/byke

go 1.27

require (
	github.com/ebitengine/oto/v3 v3.4.0
	github.com/go-gl/glfw/v3.4/glfw v0.1.0-pre.1.0.20260628091122-0bd588dc30cf
	github.com/go-text/render v0.2.1
	github.com/go-text/typesetting v0.3.4
	github.com/hajimehoshi/ebiten/v2 v2.9.9
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/oliverbestmann/earcut-go v1.0.0
	github.com/oliverbestmann/puffin-go v1.0.0
	github.com/oliverbestmann/webgpu v1.34.0
	github.com/pkg/profile v1.7.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/exp v0.0.0-20260611194520-c48552f49976
	golang.org/x/image v0.40.0
	golang.org/x/mobile v0.0.0-20260520154334-0e4426e1883d
)

require (
	github.com/chewxy/math32 v1.11.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/felixge/fgprof v0.9.5 // indirect
	github.com/google/pprof v0.0.0-20260507013755-92041b743c96 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.4 // indirect
	github.com/jfreymuth/oggvorbis v1.0.5 // indirect
	github.com/jfreymuth/vorbis v1.0.2 // indirect
	github.com/oliverbestmann/mikktspace-go v0.0.0-20260628135113-36b1a30cb1e0 // indirect
	github.com/oliverbestmann/webgpu/libs-android v0.0.0-20260628152806-6b27e30a172e // indirect
	github.com/oliverbestmann/webgpu/libs-darwin v0.0.0-20260628152755-66a5dfa57f8d // indirect
	github.com/oliverbestmann/webgpu/libs-ios v0.0.0-20260628152757-fe2537e7ddac // indirect
	github.com/oliverbestmann/webgpu/libs-linux v0.0.0-20260628152803-421b8a341d08 // indirect
	github.com/oliverbestmann/webgpu/libs-windows v0.0.0-20260628152801-f47d1b682eb8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	github.com/timandy/routine v1.1.6 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
	golang.org/x/tools/cmd/godoc v0.1.0-deprecated // indirect
	golang.org/x/tools/godoc v0.1.0-deprecated // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// The current glfw/v3.4 bindings are broken when trying to use wayland and x11 at the same time.
// This is a fix for that.
// See https://github.com/go-gl/glfw/pull/420 for more information
// replace github.com/go-gl/glfw/v3.4/glfw v0.1.0-pre.1.0.20260406072232-3ac4aa2bb164 => github.com/oliverbestmann/go-gl-glfw/v3.4/glfw v0.0.0-20260510101646-c1f83c493fe1

tool golang.org/x/tools/cmd/godoc
