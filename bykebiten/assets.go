package bykebiten

import (
	"bytes"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke/bykebiten/audio"
	"image"
	"io"
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

type Assets struct {
	fs fs.FS

	bytes  assetCache[[]byte]
	images assetCache[*ebiten.Image]
	audios assetCache[*AudioSource]
}

func (a *Assets) Bytes(path string) *AsyncAsset[[]byte] {
	return a.bytes.Get(path, func() ([]byte, error) {
		return fs.ReadFile(a.fs, path)
	})
}

func (a *Assets) Image(path string) *AsyncAsset[*ebiten.Image] {
	return a.images.Get(path, func() (*ebiten.Image, error) {
		fp, err := a.fs.Open(path)
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

func (a *Assets) Audio(path string) *AsyncAsset[*AudioSource] {
	return a.audios.Get(path, func() (*AudioSource, error) {
		fp, err := a.fs.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open asset %q: %w", path, err)
		}

		buf, err := io.ReadAll(fp)
		if err != nil {
			return nil, fmt.Errorf("read asset %q into memory: %w", path, err)
		}

		// try to parse the file once. if that works, we will just create an instance
		// each time without error checking
		if _, err := audio.OpenStream(bytes.NewReader(buf)); err != nil {
			return nil, fmt.Errorf("create stream for %q: %w", path, err)
		}

		factory := func() audio.AudioStream[float32] {
			stream, err := audio.OpenStream(bytes.NewReader(buf))
			if err != nil {
				err := fmt.Errorf("create stream for %q: %w", path, err)
				panic(err)
			}

			return stream
		}

		return &AudioSource{factory: factory}, nil
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
