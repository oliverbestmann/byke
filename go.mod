module github.com/oliverbestmann/byke

go 1.26.1

require (
	github.com/Kaidzen-62/radixsort v0.2.0
	github.com/ebitengine/oto/v3 v3.4.0
	github.com/go-text/render v0.2.1
	github.com/go-text/typesetting v0.3.4
	github.com/hajimehoshi/ebiten/v2 v2.9.9
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/oliverbestmann/puffin-go v1.0.0
	github.com/oliverbestmann/pulse v0.0.0-20260511173117-ca9506c88200
	github.com/oliverbestmann/webgpu v1.33.2
	github.com/pkg/profile v1.7.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/image v0.40.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/felixge/fgprof v0.9.5 // indirect
	github.com/go-gl/glfw/v3.4/glfw v0.1.0-pre.1.0.20260406072232-3ac4aa2bb164 // indirect
	github.com/google/pprof v0.0.0-20260507013755-92041b743c96 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.4 // indirect
	github.com/jfreymuth/oggvorbis v1.0.5 // indirect
	github.com/jfreymuth/vorbis v1.0.2 // indirect
	github.com/oliverbestmann/webgpu/libs-android v0.0.0-20260509160813-48db59792a15 // indirect
	github.com/oliverbestmann/webgpu/libs-darwin v0.0.0-20260509160802-b09403b07cd3 // indirect
	github.com/oliverbestmann/webgpu/libs-ios v0.0.0-20260509160803-765e39d2a48b // indirect
	github.com/oliverbestmann/webgpu/libs-linux v0.0.0-20260509160809-2fefaf7c9ead // indirect
	github.com/oliverbestmann/webgpu/libs-windows v0.0.0-20260509160807-0bc32b12c7bc // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sagernet/sing v0.7.10 // indirect
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	github.com/timandy/routine v1.1.6 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/exp v0.0.0-20260508232706-74f9aab9d74a // indirect
	golang.org/x/mobile v0.0.0-20260508232728-bebd421c7fa8 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	golang.org/x/tools/cmd/godoc v0.1.0-deprecated // indirect
	golang.org/x/tools/godoc v0.1.0-deprecated // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// The current glfw/v3.4 bindings are broken when trying to use wayland and x11 at the same time.
// This is a fix for that.
// See https://github.com/go-gl/glfw/pull/420 for more information
replace github.com/go-gl/glfw/v3.4/glfw v0.1.0-pre.1.0.20260406072232-3ac4aa2bb164 => github.com/oliverbestmann/go-gl-glfw/v3.4/glfw v0.0.0-20260510101646-c1f83c493fe1

tool golang.org/x/tools/cmd/godoc
