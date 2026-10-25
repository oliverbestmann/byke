module github.com/oliverbestmann/byke

go 1.27

require (
	github.com/chewxy/math32 v1.11.2
	github.com/ebitengine/oto/v3 v3.5.1
	github.com/go-gl/glfw/v3.4/glfw v0.1.0-pre.2
	github.com/go-text/render v0.2.1
	github.com/go-text/typesetting v0.3.5
	github.com/hajimehoshi/go-mp3 v0.3.4
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/jfreymuth/oggvorbis v1.0.5
	github.com/klauspost/compress v1.20.3
	github.com/oliverbestmann/earcut-go v1.1.1
	github.com/oliverbestmann/mikktspace-go v0.0.0-20261021165803-21f42ec57da7
	github.com/oliverbestmann/puffin-go v1.0.0
	github.com/oliverbestmann/webgpu v1.37.0
	github.com/oliverbestmann/wesl-go v1.2.0
	github.com/stretchr/testify v1.11.1
	github.com/x448/float16 v0.8.4
	golang.org/x/exp v0.0.0-20261014143641-f0251eb4aa3e
	golang.org/x/image v0.47.0
	golang.org/x/mobile v0.0.0-20261011034528-197770444867
	golang.org/x/sys v0.49.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2/v2 v2.9.1 // indirect
	github.com/dop251/goja v0.0.0-20261023133238-1622cea953df // indirect
	github.com/ebitengine/purego v0.11.2 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/google/pprof v0.0.0-20261016150608-1facf8f91b50 // indirect
	github.com/jfreymuth/pulse v0.1.3 // indirect
	github.com/jfreymuth/vorbis v1.0.2 // indirect
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
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
	golang.org/x/tools/cmd/godoc v0.1.0-deprecated // indirect
	golang.org/x/tools/godoc v0.1.0-deprecated // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// The current glfw/v3.4 bindings are broken when trying to use wayland and x11 at the same time.
// This is a fix for that.
// See https://github.com/go-gl/glfw/pull/420 for more information
// replace github.com/go-gl/glfw/v3.4/glfw v0.1.0-pre.1.0.20260406072232-3ac4aa2bb164 => github.com/oliverbestmann/go-gl-glfw/v3.4/glfw v0.0.0-20260510101646-c1f83c493fe1

tool (
	golang.org/x/tools/cmd/godoc
	golang.org/x/tools/cmd/stringer
)
