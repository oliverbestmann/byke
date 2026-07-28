package ktx2

import (
	"bytes"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

//go:generate stringer -type SupercompressionScheme
type SupercompressionScheme uint32

const SupercompressionSchemeNone SupercompressionScheme = 0
const SupercompressionSchemeBasisLZ SupercompressionScheme = 1
const SupercompressionSchemeZSTD SupercompressionScheme = 2
const SupercompressionSchemeZLIB SupercompressionScheme = 3

func decompress(scheme SupercompressionScheme, buf []byte) ([]byte, error) {
	switch scheme {
	case SupercompressionSchemeNone:
		return buf, nil

	case SupercompressionSchemeZSTD:
		r, _ := zstd.NewReader(bytes.NewReader(buf))
		return io.ReadAll(r)

	default:
		return nil, fmt.Errorf("unsupported compression: %q", scheme)
	}
}
