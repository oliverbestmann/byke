package bykebiten

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"io/fs"
	"log/slog"
	"path"
	"reflect"
	"sync/atomic"
	"time"
)

type assetCache[T any] struct {
	values   map[string]*AsyncAsset[T]
	loading  atomic.Int32
	finished atomic.Int32
}

func (a *assetCache[T]) Get(p string, load func() (T, error)) *AsyncAsset[T] {
	if a.values == nil {
		a.values = make(map[string]*AsyncAsset[T], 64)
	}

	// cleanup path to improve cache hits
	p = path.Clean(p)

	// check cache first
	if cached, ok := a.values[p]; ok {
		return cached
	}

	a.loading.Add(1)

	slog.Info("Start loading asset",
		slog.String("type", reflect.TypeFor[T]().String()),
		slog.String("path", p))

	startTime := time.Now()

	// actually load the asset
	asyncAsset := loadAsync(func() (T, error) {
		defer a.finished.Add(1)
		defer func() {
			slog.Info("Finish loading asset",
				slog.String("type", reflect.TypeFor[T]().String()),
				slog.String("path", p),
				slog.Duration("duration", time.Since(startTime)))
		}()

		return load()
	})

	// and put the promise into the cache
	a.values[p] = asyncAsset

	return asyncAsset
}

type Assets struct {
	FS AssetsFS

	bytes  assetCache[[]byte]
	images assetCache[*ebiten.Image]
}

func (a *Assets) Bytes(path string) *AsyncAsset[[]byte] {
	return a.bytes.Get(path, func() ([]byte, error) {
		return fs.ReadFile(a.FS, path)
	})
}

func (a *Assets) Image(path string) *AsyncAsset[*ebiten.Image] {
	return a.images.Get(path, func() (*ebiten.Image, error) {
		fp, err := a.FS.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open asset %q: %w", path, err)
		}

		img, _, err := image.Decode(fp)
		if err != nil {
			return nil, fmt.Errorf("decode image %q: %w", path, err)
		}

		return ebiten.NewImageFromImage(img), nil
	})
}

func (a *Assets) StartCount() int {
	return int(a.bytes.loading.Load() + a.images.loading.Load())
}

func (a *Assets) FinishCount() int {
	return int(a.bytes.finished.Load() + a.images.finished.Load())
}

func (a *Assets) IsLoading() bool {
	return a.StartCount() > a.FinishCount()
}

type AsyncAsset[T any] struct {
	value atomic.Pointer[T]
	error atomic.Pointer[error]
	done  <-chan struct{}
}

func loadAsync[T any](load func() (T, error)) *AsyncAsset[T] {
	doneCh := make(chan struct{})
	asset := &AsyncAsset[T]{done: doneCh}

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

func (a *AsyncAsset[T]) Poll() (T, error, bool) {
	var tZero T

	if value := a.value.Load(); value != nil {
		return *value, nil, true
	}

	if err := a.error.Load(); err != nil {
		return tZero, *err, true
	}

	return tZero, nil, false
}

func (a *AsyncAsset[T]) PollSuccess() (T, bool) {
	value, err, ok := a.Poll()
	if ok && err != nil {
		panic(fmt.Errorf("failed to load asset: %w", err))
	}

	return value, ok
}

func (a *AsyncAsset[T]) Await() T {
	value, err := a.TryAwait()
	if err != nil {
		panic(fmt.Errorf("failed to load asset: %w", err))
	}

	return value
}

func (a *AsyncAsset[T]) TryAwait() (T, error) {
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
