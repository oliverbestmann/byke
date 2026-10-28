// Command gallery builds a static site of the byke2d examples: a thumbnail grid
// grouped by category and one page per example with its wasm build and source.
//
// Thumbnails are cached in -thumbs, named by a hash of the example directory,
// so an example is only rendered again after it changed.
package main

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type example struct {
	Category  string
	Name      string
	Dir       string
	Thumbnail string
	Sources   []source

	// NoWasm is set by a "//byke:no wasm" comment in a source file. The example is shown as source only.
	NoWasm bool
}

type source struct {
	Path     string
	Language string
	Code     string

	// HTML holds the highlighted Go code, otherwise highlight.js handles Code
	HTML template.HTML
}

type category struct {
	Name     string
	Examples []*example
}

func main() {
	examplesDir := flag.String("examples", "byke2d/examples", "directory with <category>/<example>/main.go")
	thumbsDir := flag.String("thumbs", "gallery-thumbs", "cache directory for thumbnails")
	outDir := flag.String("out", "site", "output directory")
	tinygo := flag.Bool("tinygo", false, "build the wasm with TinyGo (byke-cli build-web --tinygo)")
	flag.Parse()

	if err := run(*examplesDir, *thumbsDir, *outDir, *tinygo); err != nil {
		slog.Error("Build gallery", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(examplesDir, thumbsDir, outDir string, tinygo bool) error {
	mains, err := filepath.Glob(filepath.Join(examplesDir, "*", "*", "main.go"))
	if err != nil {
		return err
	}

	if err := os.MkdirAll(thumbsDir, 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(outDir, "thumbs"), 0o755); err != nil {
		return err
	}

	usedThumbs := map[string]bool{}

	var categories []*category

	for _, main := range mains {
		dir := filepath.Dir(main)

		ex := &example{
			Category: filepath.Base(filepath.Dir(dir)),
			Name:     filepath.Base(dir),
			Dir:      dir,
		}

		slog.Info("Build example", slog.String("category", ex.Category), slog.String("name", ex.Name))

		sources, err := readSources(dir)
		if err != nil {
			return err
		}

		ex.Sources = sources
		ex.NoWasm = slices.ContainsFunc(sources, func(s source) bool { return strings.Contains(s.Code, "//byke:no wasm") })

		pageDir := filepath.Join(outDir, ex.Category, ex.Name)

		if !ex.NoWasm {
			thumbnail, err := thumbnail(ex, thumbsDir, outDir)
			if err != nil {
				// a missing thumbnail should not keep the gallery from deploying
				slog.Warn("Render thumbnail", slog.String("dir", dir), slog.String("error", err.Error()))
			}

			if thumbnail != "" {
				usedThumbs[thumbnail] = true
				ex.Thumbnail = "thumbs/" + thumbnail
			}

			if err := buildWasm(dir, filepath.Join(pageDir, "wasm"), tinygo); err != nil {
				return fmt.Errorf("build wasm of %s: %w", dir, err)
			}
		}

		if err := writeTemplate(filepath.Join(pageDir, "index.html"), examplePage, ex); err != nil {
			return err
		}

		idx := slices.IndexFunc(categories, func(c *category) bool { return c.Name == ex.Category })
		if idx < 0 {
			categories = append(categories, &category{Name: ex.Category})
			idx = len(categories) - 1
		}

		categories[idx].Examples = append(categories[idx].Examples, ex)
	}

	slices.SortFunc(categories, func(a, b *category) int { return cmp.Compare(a.Name, b.Name) })

	if err := writeTemplate(filepath.Join(outDir, "index.html"), indexPage, categories); err != nil {
		return err
	}

	// drop thumbnails of old example versions, so the cache does not grow
	entries, err := os.ReadDir(thumbsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !usedThumbs[entry.Name()] {
			slog.Info("Remove old thumbnail", slog.String("name", entry.Name()))
			_ = os.Remove(filepath.Join(thumbsDir, entry.Name()))
		}
	}

	return nil
}

// thumbnail renders the example offscreen unless a cached thumbnail exists,
// copies it to the output and returns its file name.
func thumbnail(ex *example, thumbsDir, outDir string) (string, error) {
	hash, err := hashDir(ex.Dir)
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s-%s-%s.jpg", ex.Category, ex.Name, hash)

	cached, err := filepath.Abs(filepath.Join(thumbsDir, name))
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(cached); errors.Is(err, fs.ErrNotExist) {
		// render to a temporary file, so a failed run does not leave a broken thumbnail in the cache
		tmp := cached + ".tmp"
		defer os.Remove(tmp)

		cmd := exec.Command("go", "run", "--tags", "fakeClock", ".")
		cmd.Dir = ex.Dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(),
			"BYKE_RUN_OFFSCREEN_TEST=true",
			"BYKE_SCREENSHOT="+tmp,
			"WGPU_FORCE_FALLBACK_ADAPTER=1",
		)

		if err := cmd.Run(); err != nil {
			return "", err
		}

		if err := os.Rename(tmp, cached); err != nil {
			return "", err
		}
	}

	data, err := os.ReadFile(cached)
	if err != nil {
		return "", err
	}

	return name, os.WriteFile(filepath.Join(outDir, "thumbs", name), data, 0o644)
}

// hashDir hashes the files of an example, without build output.
func hashDir(dir string) (string, error) {
	h := sha256.New()

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && d.Name() == "dist" {
			return filepath.SkipDir
		}

		if d.IsDir() || d.Name() == "log" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(dir, path)
		_, _ = fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), len(data))
		_, _ = h.Write(data)
		return nil
	})

	return hex.EncodeToString(h.Sum(nil))[:16], err
}

func buildWasm(dir, target string, tinygo bool) error {
	args := []string{"build-web", "--clean"}
	if tinygo {
		args = append(args, "--tinygo")
	}

	cmd := exec.Command("byke-cli", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	// CopyFS does not overwrite files of a previous run
	if err := os.RemoveAll(target); err != nil {
		return err
	}

	return os.CopyFS(target, os.DirFS(filepath.Join(dir, "dist", "wasm")))
}

func readSources(dir string) ([]source, error) {
	goFiles, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}

	shaders, err := filepath.Glob(filepath.Join(dir, "assets", "*.wgsl"))
	if err != nil {
		return nil, err
	}

	var sources []source
	for _, path := range slices.Concat(goFiles, shaders) {
		code, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		rel, _ := filepath.Rel(dir, path)
		src := source{Path: filepath.ToSlash(rel), Code: string(code)}

		if strings.HasSuffix(path, ".go") {
			src.HTML = highlightGo(code)
		} else {
			// shortcut: highlight.js has no wgsl, rust is close enough
			src.Language = "rust"
		}

		sources = append(sources, src)
	}

	return sources, nil
}

func writeTemplate(path string, tmpl *template.Template, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	fp, err := os.Create(path)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(fp, data); err != nil {
		_ = fp.Close()
		return err
	}

	return fp.Close()
}

const style = `{{define "style"}}<style>
  body { margin: 0 auto; padding: 1rem; max-width: 1280px; background: #111; color: #ddd; font-family: sans-serif; }
  a { color: #8cf; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 1rem; }
  .grid a { display: block; text-decoration: none; background: #222; border-radius: 4px; overflow: hidden; }
  .grid img, .grid .missing { display: block; width: 100%; aspect-ratio: 4 / 3; object-fit: cover; background: #333; }
  .grid span { display: block; padding: 0.5rem; }
  iframe { display: block; width: 100%; aspect-ratio: 16 / 9; border: 0; background: black; }
  pre code.hljs { border-radius: 4px; }
  .hljs-type { color: #ffa657; }
</style>{{end}}`

var indexPage = template.Must(template.Must(template.New("index").Parse(style)).Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>byke examples</title>
  {{template "style"}}
</head>
<body>
<h1>byke examples</h1>
<p>
  <a href="https://github.com/oliverbestmann/byke">byke</a> is an entity component system (ECS) for Go: a game holds its state as entities with data components, and systems are functions that query and update them. It is modeled on the
  <a href="https://bevy.org/">Bevy</a> game engine. byke2d adds 2D and 3D rendering with WebGPU, audio, input, and asset loading.
  Each example page shows the source code and runs the example in your browser. The ecs examples print to the console, so their pages show only the source.
</p>
{{range .}}
<h2>{{.Name}}</h2>
<div class="grid">
  {{range .Examples}}
  <a href="{{.Category}}/{{.Name}}/">
    {{if .Thumbnail}}<img src="{{.Thumbnail}}" alt="" loading="lazy">{{else}}<div class="missing"></div>{{end}}
    <span>{{.Name}}</span>
  </a>
  {{end}}
</div>
{{end}}
</body>
</html>
`))

var examplePage = template.Must(template.Must(template.New("example").Parse(style)).Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>{{.Name}} - byke examples</title>
  {{template "style"}}
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.11.1/styles/github-dark.min.css">
  <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.11.1/highlight.min.js"></script>
  <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.11.1/languages/rust.min.js"></script>
</head>
<body>
<p><a href="../../">All examples</a></p>
<h1>{{.Category}} / {{.Name}}</h1>
<p><a href="https://github.com/oliverbestmann/byke/tree/main/byke2d/examples/{{.Category}}/{{.Name}}">Source on GitHub</a></p>
{{if not .NoWasm}}<iframe src="wasm/" allowfullscreen></iframe>{{end}}
{{range .Sources}}
<h2>{{.Path}}</h2>
{{if .HTML}}<pre><code class="hljs">{{.HTML}}</code></pre>
{{else}}<pre><code class="language-{{.Language}}">{{.Code}}</code></pre>{{end}}
{{end}}
<script>document.querySelectorAll("code[class^=language-]").forEach(el => hljs.highlightElement(el))</script>
</body>
</html>
`))
