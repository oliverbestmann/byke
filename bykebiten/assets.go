package bykebiten

import (
	"bytes"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
)

type LoadAssetSettings interface {
	IsLoadSettings()
}

type LoadContext struct {
	// The path to the file we're currently loading
	Path string

	// A load specific settings object
	Settings LoadAssetSettings
}

type AssetLoader interface {
	Load(ctx LoadContext, r io.ReadSeekCloser) (any, error)
	Extensions() []string
}

type Assets struct {
	fs fs.FS

	loaders map[string]AssetLoader
	generic *assetCache[any]

	bytes *assetCache[[]byte]
}

func makeAssets(fs fs.FS, loaders ...AssetLoader) Assets {
	assets := Assets{
		fs:      fs,
		loaders: make(map[string]AssetLoader, 8),
		generic: &assetCache[any]{},
		bytes:   &assetCache[[]byte]{},
	}

	for _, l := range loaders {
		assets.RegisterLoader(l)
	}

	return assets
}

func (a *Assets) RegisterLoader(l AssetLoader) {
	for _, ext := range l.Extensions() {
		ext = strings.ToLower(ext)
		a.loaders[ext] = l
	}
}

func (a *Assets) Load(path string) AsyncAsset[any] {
	return a.LoadWithSettings(path, nil)
}

func (a *Assets) LoadWithSettings(path string, settings LoadAssetSettings) AsyncAsset[any] {
	if a.generic == nil {
		a.generic = &assetCache[any]{}
	}

	ext := strings.ToLower(filepath.Ext(path))

	loader, ok := a.loaders[ext]
	if !ok {
		err := fmt.Errorf("no loader for extension %q", ext)
		panic(err)
	}

	return a.generic.Get(path, func() (any, error) {
		fp, err := a.fs.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open asset %q: %w", path, err)
		}

		ctx := LoadContext{
			Path:     path,
			Settings: settings,
		}

		asset, err := loader.Load(ctx, readSeekerOf(fp))
		if err != nil {
			return nil, fmt.Errorf("loading asset %q with loader %T: %w", path, loader, err)
		}

		return asset, nil
	})
}

func (a *Assets) Bytes(path string) AsyncAsset[[]byte] {
	return a.bytes.Get(path, func() ([]byte, error) {
		return fs.ReadFile(a.fs, path)
	})
}

func (a *Assets) Image(path string) AsyncAsset[*ebiten.Image] {
	return asTypedAsyncAsset[*ebiten.Image](a.Load(path))
}

func (a *Assets) Audio(path string) AsyncAsset[*AudioSource] {
	return asTypedAsyncAsset[*AudioSource](a.Load(path))
}

func (a *Assets) StartCount() int {
	return int(a.bytes.Loading() + a.generic.Loading())
}

func (a *Assets) FinishCount() int {
	return int(a.bytes.Finished() + a.generic.Finished())
}

func (a *Assets) IsLoading() bool {
	return a.StartCount() > a.FinishCount()
}

type asyncAsset[T any] struct {
	value atomic.Pointer[T]
	error atomic.Pointer[error]
	done  <-chan struct{}
}

func loadAsync[T any](load func() (T, error)) *asyncAsset[T] {
	doneCh := make(chan struct{})
	asset := &asyncAsset[T]{done: doneCh}

	// spawn the go routine to load the actual asset
	go func() {
		defer close(doneCh)

		defer func() {
			// we got a panic, propagate to the error
			if p := recover(); p != nil {
				err := fmt.Errorf("loading asset panicked: %v", p)
				asset.error.Store(&err)
			}
		}()

		// load the value
		value, err := load()

		if err != nil {
			asset.error.Store(&err)
			return
		}

		asset.value.Store(&value)
	}()

	return asset
}

func (a *asyncAsset[T]) Poll() (T, error, bool) {
	var tZero T

	if value := a.value.Load(); value != nil {
		return *value, nil, true
	}

	if err := a.error.Load(); err != nil {
		return tZero, *err, true
	}

	return tZero, nil, false
}

func (a *asyncAsset[T]) PollSuccess() (T, bool) {
	value, err, ok := a.Poll()
	if ok && err != nil {
		panic(fmt.Errorf("failed to load asset: %w", err))
	}

	return value, ok
}

func (a *asyncAsset[T]) Await() T {
	value, err := a.TryAwait()
	if err != nil {
		panic(fmt.Errorf("failed to load asset: %w", err))
	}

	return value
}

func (a *asyncAsset[T]) TryAwait() (T, error) {
	for {
		if value := a.value.Load(); value != nil {
			return *value, nil
		}

		if err := a.error.Load(); err != nil {
			var tZero T
			return tZero, *err
		}

		// wait for the channel to close
		<-a.done
	}
}

type AsyncAsset[T any] interface {
	Await() T
	TryAwait() (T, error)
	PollSuccess() (T, bool)
	Poll() (T, error, bool)
}

func asTypedAsyncAsset[T any](asset AsyncAsset[any]) AsyncAsset[T] {
	return &typedAsyncAsset[T]{Asset: asset}
}

type typedAsyncAsset[T any] struct {
	Asset AsyncAsset[any]
}

func (t *typedAsyncAsset[T]) Await() T {
	return t.Asset.Await().(T)
}

func (t *typedAsyncAsset[T]) TryAwait() (T, error) {
	value, err := t.Asset.TryAwait()
	if err != nil {
		var tZero T
		return tZero, err
	}

	return value.(T), nil
}

func (t *typedAsyncAsset[T]) PollSuccess() (T, bool) {
	value, _, ok := t.Poll()
	return value, ok
}

func (t *typedAsyncAsset[T]) Poll() (T, error, bool) {
	value, err, ok := t.Asset.Poll()
	if !ok {
		var tZero T
		return tZero, nil, false
	}

	if err != nil {
		var tZero T
		return tZero, err, true
	}

	return value.(T), nil, true
}

type assetCache[T any] struct {
	values   map[string]*asyncAsset[T]
	loading  atomic.Int32
	finished atomic.Int32
}

func (a *assetCache[T]) Loading() int32 {
	if a == nil {
		return 0
	}

	return a.loading.Load()
}

func (a *assetCache[T]) Finished() int32 {
	if a == nil {
		return 0
	}

	return a.finished.Load()
}

func (a *assetCache[T]) Get(p string, load func() (T, error)) *asyncAsset[T] {
	if a.values == nil {
		a.values = make(map[string]*asyncAsset[T], 64)
	}

	// cleanup path to improve cache hits
	p = path.Clean(p)

	// check cache first
	if cached, ok := a.values[p]; ok {
		return cached
	}

	a.loading.Add(1)

	slog.Debug("Start loading asset",
		slog.String("type", reflect.TypeFor[T]().String()),
		slog.String("path", p))

	startTime := time.Now()

	// actually load the asset
	asyncAsset := loadAsync(func() (value T, err error) {
		defer a.finished.Add(1)
		defer func() {
			if err != nil {
				slog.Warn("Failed to load asset",
					slog.String("type", reflect.TypeFor[T]().String()),
					slog.String("path", p),
					slog.Duration("duration", time.Since(startTime)),
					slog.String("error", err.Error()))
			} else {
				slog.Debug("Finish loading asset",
					slog.String("type", reflect.TypeFor[T]().String()),
					slog.String("path", p),
					slog.Duration("duration", time.Since(startTime)))
			}
		}()

		return load()
	})

	// and put the promise into the cache
	a.values[p] = asyncAsset

	return asyncAsset
}

func readSeekerOf(r io.ReadCloser) io.ReadSeekCloser {
	if rs, ok := r.(io.ReadSeekCloser); ok {
		return rs
	}

	defer func() { _ = r.Close() }()

	buf, err := io.ReadAll(r)
	if err != nil {
		return errRead{error: err}
	}

	type RSC struct {
		io.ReadSeeker
		io.Closer
	}

	reader := bytes.NewReader(buf)

	return RSC{
		ReadSeeker: reader,
		Closer:     io.NopCloser(reader),
	}
}

type errRead struct {
	error error
}

func (e errRead) Read(p []byte) (n int, err error) {
	return 0, e.error
}

func (e errRead) Seek(offset int64, whence int) (int64, error) {
	return 0, e.error
}

func (e errRead) Close() error {
	return nil
}
