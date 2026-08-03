package httpfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"slices"
	"time"
)

type FS struct {
	ctx     context.Context
	client  *http.Client
	headers http.Header
	base    *url.URL
}

// New provides a filesystem (an fs.FS) for the HTTP (or HTTPS) endpoint
// rooted at u. This filesystem is suitable for use with the 'http' or
// 'https' URL schemes. All reads are made with the GET method, while stat calls
// are made with the HEAD method (with a fallback to GET).
//
// A context can be given by using WithContext.
// HTTP Headers can be provided by using WithHeader.
func New(u *url.URL) *FS {
	return &FS{
		ctx:     context.Background(),
		client:  http.DefaultClient,
		headers: http.Header{},
		base:    u,
	}
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.SubFS      = (*FS)(nil)
)

func (f *FS) URL() string {
	return f.base.String()
}

func (f *FS) WithContext(ctx context.Context) fs.FS {
	if ctx == nil {
		return f
	}

	fsys := *f
	fsys.ctx = ctx

	return &fsys
}

func (f *FS) WithHeader(headers http.Header) fs.FS {
	if headers == nil {
		return f
	}

	fsys := *f
	if len(fsys.headers) == 0 {
		fsys.headers = headers
	} else {
		for k, vs := range headers {
			for _, v := range vs {
				fsys.headers.Add(k, v)
			}
		}
	}

	return &fsys
}

func (f *FS) WithHTTPClient(client *http.Client) fs.FS {
	if client == nil {
		return f
	}

	fsys := *f
	fsys.client = client

	return new(fsys)
}

func (f *FS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	return &httpFile{
		ctx:    f.ctx,
		u:      f.base.JoinPath(name),
		client: f.client,
		name:   name,
		hdr:    f.headers,
	}, nil
}

func (f *FS) ReadFile(name string) ([]byte, error) {
	opened, err := f.Open(name)
	if err != nil {
		return nil, err
	}

	defer func() { _ = opened.Close() }()

	b, err := io.ReadAll(opened)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (f *FS) Sub(name string) (fs.FS, error) {
	fsys := *f
	fsys.base = f.base.JoinPath(name)
	return new(fsys), nil
}

type httpFile struct {
	ctx    context.Context
	body   io.ReadCloser
	fi     fs.FileInfo
	u      *url.URL
	client *http.Client
	hdr    http.Header
	name   string
}

func (f *httpFile) request(method string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(f.ctx, method, f.u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header = f.hdr

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}

	modTime := time.Time{}
	if mod := resp.Header.Get("Last-Modified"); mod != "" {
		// best-effort - if it can't be parsed, just ignore it...
		modTime, _ = http.ParseTime(mod)
	}

	f.fi = fileInfo(f.name, resp.ContentLength, 0o444, modTime)

	if resp.StatusCode == 0 || resp.StatusCode >= 400 {
		_ = resp.Body.Close()

		return nil, httpError(method, resp.StatusCode)
	}

	// The response body must be closed later
	return resp.Body, nil
}

func (f *httpFile) Close() error {
	if f.body == nil {
		return nil
	}

	return f.body.Close()
}

func (f *httpFile) Read(p []byte) (int, error) {
	if f.body == nil {
		body, err := f.request(http.MethodGet)
		if err != nil {
			return 0, err
		}

		f.body = body
	}

	return f.body.Read(p)
}

func (f *httpFile) Stat() (fs.FileInfo, error) {
	body, err := f.request(http.MethodHead)
	if err == nil {
		_ = body.Close()

		return f.fi, nil
	}

	var he httpErr

	fallbackCodes := []int{http.StatusMethodNotAllowed, http.StatusUnauthorized}
	if !errors.As(err, &he) || !slices.Contains(fallbackCodes, he.StatusCode()) {
		return nil, err
	}

	// fall back to GET if HEAD returned one of fallback codes
	body, err = f.request(http.MethodGet)
	if err != nil {
		return nil, err
	}

	_ = body.Close()

	return f.fi, nil
}

// httpError represents an HTTP error with its status code
func httpError(method string, statusCode int) error {
	return httpErr{
		method:     method,
		statusCode: statusCode,
	}
}

type httpErr struct {
	method     string
	statusCode int
}

func (e httpErr) Error() string {
	return fmt.Sprintf("http %s failed with status %d", e.method, e.statusCode)
}

func (e httpErr) StatusCode() int {
	return e.statusCode
}

// fileInfo creates a static fs.FileInfo with the given properties.
func fileInfo(name string, size int64, mode fs.FileMode, modTime time.Time) fs.FileInfo {
	return &staticFileInfo{
		name:    name,
		size:    size,
		mode:    mode,
		modTime: modTime,
	}
}

type staticFileInfo struct {
	modTime time.Time
	name    string
	size    int64
	mode    fs.FileMode
}

func (fi *staticFileInfo) IsDir() bool        { return fi.Mode().IsDir() }
func (fi *staticFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi *staticFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *staticFileInfo) Name() string       { return fi.name }
func (fi *staticFileInfo) Size() int64        { return fi.size }
func (fi *staticFileInfo) Sys() any           { return nil }
