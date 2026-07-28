package ktx2

import (
	"fmt"
	"io"
	"slices"
	"unsafe"
)

type HeaderValues struct {
	Identifier             [12]byte
	VkFormat               VkFormat
	TypeSize               uint32
	PixelWidth             uint32
	PixelHeight            uint32
	PixelDepth             uint32
	LayerCount             uint32
	FaceCount              uint32
	LevelCount             uint32
	SupercompressionScheme SupercompressionScheme
}

type Index struct {
	DataFormatDescByteOffset             uint32
	DataFormatDescByteLength             uint32
	KeyValueDescByteOffset               uint32
	KeyValueDescByteLength               uint32
	SuperCompressionGlobalDataByteOffset uint64
	SuperCompressionGlobalDataByteLength uint64
}

type LevelIndex struct {
	ByteOffset             uint64
	ByteLength             uint64
	UncompressedByteLength uint64
}

func Open(r io.ReadSeeker) (Reader, error) {
	var header Reader

	header.read = r

	{
		var buf [9*4 + 12]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return Reader{}, fmt.Errorf("read header: %w", err)
		}

		header.Header = *(*HeaderValues)(unsafe.Pointer(&buf[0]))
	}

	{
		var buf [4*4 + 2*8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return Reader{}, fmt.Errorf("read index: %w", err)
		}

		header.index = *(*Index)(unsafe.Pointer(&buf[0]))
	}

	{
		n := max(1, header.Header.LevelCount)

		var buf = make([]byte, n*3*8)
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return Reader{}, fmt.Errorf("read level index: %w", err)
		}

		header.levels = unsafe.Slice((*LevelIndex)(unsafe.Pointer(&buf[0])), n)
	}

	return header, nil
}

type Face struct {
	Level     uint32
	Layer     uint32
	Face      uint32
	Width     uint32
	Height    uint32
	Depth     uint32
	RowStride uint32
	Layer3d   uint32
	Buffer    []byte
}

type Reader struct {
	Header HeaderValues
	index  Index
	levels []LevelIndex

	read io.ReadSeeker
}

func (r Reader) Faces() ([]Face, error) {
	var faces []Face

	bpp, ok := vkFormatToBytePerPixel[r.Header.VkFormat]
	if !ok {
		return nil, fmt.Errorf("unknown number of byte per pixel for format %q", r.Header.VkFormat)
	}

	for level, levelConfig := range slices.Backward(r.levelsWithSizes()) {
		if _, err := r.read.Seek(int64(levelConfig.ByteOffset), io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek to level %d: %w", level, err)
		}

		buf := make([]byte, levelConfig.ByteLength)
		if _, err := io.ReadFull(r.read, buf); err != nil {
			return nil, fmt.Errorf("read %d bytes for level %d: %w", levelConfig.ByteLength, level, err)
		}

		buf, err := decompress(r.Header.SupercompressionScheme, buf)
		if err != nil {
			return nil, fmt.Errorf("decompress data in level %d: %w", level, err)
		}

		rowStride := bpp * levelConfig.Width
		faceSize := rowStride * levelConfig.Height * levelConfig.Depth

		for layer := range max(1, r.Header.LayerCount) {
			for face := range r.Header.FaceCount {
				layer3d := layer*r.Header.FaceCount + face
				offset := faceSize * layer3d

				faces = append(faces, Face{
					Level:     uint32(level),
					Layer:     layer,
					Layer3d:   layer3d,
					Face:      face,
					Width:     levelConfig.Width,
					Height:    levelConfig.Height,
					Depth:     levelConfig.Depth,
					RowStride: rowStride,
					Buffer:    buf[offset:][:faceSize],
				})
			}
		}
	}

	return faces, nil
}

func (r Reader) levelsWithSizes() []levelWithSize {
	width := r.Header.PixelWidth
	height := r.Header.PixelHeight
	depth := r.Header.PixelDepth

	var levels []levelWithSize
	for _, level := range r.levels {
		levels = append(levels, levelWithSize{
			LevelIndex: level,
			Width:      width,
			Height:     max(height, 1),
			Depth:      max(depth, 1),
		})

		width /= 2
		height /= 2
		depth /= 2
	}

	return levels
}

type levelWithSize struct {
	LevelIndex

	Width  uint32
	Height uint32
	Depth  uint32
}
