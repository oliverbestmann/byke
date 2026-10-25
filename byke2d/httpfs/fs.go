//go:build js

// Package httpfs provides an fs.FS that reads files with the browser's fetch API.
// net/http would add several MB of HTTP and TLS code to the wasm binary.
package httpfs

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"syscall/js"
	"time"
)

type FS struct {
	base *url.URL
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.SubFS      = (*FS)(nil)
)

// New provides a filesystem rooted at u. Every Open fetches the whole file.
func New(u *url.URL) *FS {
	return &FS{base: u}
}

func (f *FS) Open(name string) (fs.File, error) {
	data, err := f.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return &file{Reader: bytes.NewReader(data), name: path.Base(name)}, nil
}

func (f *FS) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	data, err := fetch(f.base.JoinPath(name).String())
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	return data, nil
}

func (f *FS) Sub(dir string) (fs.FS, error) {
	return &FS{base: f.base.JoinPath(dir)}, nil
}

func fetch(url string) ([]byte, error) {
	resp, err := await(js.Global().Call("fetch", url))
	if err != nil {
		return nil, err
	}

	if status := resp.Get("status").Int(); !resp.Get("ok").Bool() {
		if status == 404 {
			return nil, fs.ErrNotExist
		}

		return nil, fmt.Errorf("fetch failed with status %d", status)
	}

	buf, err := await(resp.Call("arrayBuffer"))
	if err != nil {
		return nil, err
	}

	array := js.Global().Get("Uint8Array").New(buf)
	data := make([]byte, array.Length())
	js.CopyBytesToGo(data, array)

	return data, nil
}

// await blocks the calling goroutine until the promise settles.
func await(promise js.Value) (js.Value, error) {
	type result struct {
		value js.Value
		err   error
	}

	ch := make(chan result, 1)

	onResolve := js.FuncOf(func(_ js.Value, args []js.Value) any {
		ch <- result{value: args[0]}
		return nil
	})
	defer onResolve.Release()

	onReject := js.FuncOf(func(_ js.Value, args []js.Value) any {
		ch <- result{err: js.Error{Value: args[0]}}
		return nil
	})
	defer onReject.Release()

	promise.Call("then", onResolve, onReject)

	r := <-ch
	return r.value, r.err
}

// file is an in-memory file that is also its own fs.FileInfo.
type file struct {
	*bytes.Reader
	name string
}

func (f *file) Stat() (fs.FileInfo, error) { return f, nil }
func (f *file) Close() error               { return nil }
func (f *file) Name() string               { return f.name }
func (f *file) Mode() fs.FileMode          { return 0o444 }
func (f *file) ModTime() time.Time         { return time.Time{} }
func (f *file) IsDir() bool                { return false }
func (f *file) Sys() any                   { return nil }
