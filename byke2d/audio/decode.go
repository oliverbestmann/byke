//go:build !js

package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/oggvorbis"
)

func OpenStream(fp io.ReadSeeker) (Stream[float32], error) {
	var buf [12]byte

	if _, err := io.ReadFull(fp, buf[:]); err != nil {
		return nil, fmt.Errorf("detecting file format: %w", err)
	}

	_, err := fp.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("reset reader: %w", err)
	}

	if bytes.Equal([]byte("OggS"), buf[:4]) {
		return VorbisStream(fp)
	}

	if buf[0] == 0xff && (buf[1] == 0xFB || buf[1] == 0xF3 || buf[2] == 0xF2) {
		return Mp3Stream(fp)
	}

	if bytes.Equal([]byte("ID3"), buf[:3]) {
		return Mp3Stream(fp)
	}

	if bytes.Equal([]byte("RIFF"), buf[0:4]) && bytes.Equal([]byte("WAVE"), buf[8:12]) {
		return WavStream(fp)
	}

	return nil, errors.New("failed to detect audio file format")
}

func VorbisStream(fp io.ReadSeeker) (Stream[float32], error) {
	r, err := oggvorbis.NewReader(fp)
	if err != nil {
		return nil, fmt.Errorf("open vorbis stream: %w", err)
	}

	return newStereoStream(r.Channels(), r.SampleRate(), r.Length(), r.Read, r.SetPosition)
}

func Mp3Stream(fp io.ReadSeeker) (Stream[float32], error) {
	d, err := mp3.NewDecoder(fp)
	if err != nil {
		return nil, fmt.Errorf("open mp3 stream: %w", err)
	}

	// go-mp3 always decodes to 16 bit stereo, 4 bytes per frame
	const frameSize = 4

	return newStereoStream(2, d.SampleRate(), d.Length()/frameSize,
		pcmReader(d, func(b []byte) float32 { return pcm16(b) }, 2),
		func(frame int64) error {
			_, err := d.Seek(frame*frameSize, io.SeekStart)
			return err
		},
	)
}

func WavStream(fp io.ReadSeeker) (Stream[float32], error) {
	var riff [12]byte
	if _, err := io.ReadFull(fp, riff[:]); err != nil {
		return nil, fmt.Errorf("read wav header: %w", err)
	}

	var format, channels, bits uint16
	var sampleRate uint32

	// walk the chunks until we find the data chunk, fmt must come before it
	for {
		var header [8]byte
		if _, err := io.ReadFull(fp, header[:]); err != nil {
			return nil, fmt.Errorf("read wav chunk: %w", err)
		}

		id := string(header[:4])
		size := int64(binary.LittleEndian.Uint32(header[4:]))

		if id == "fmt " {
			chunk := make([]byte, size)
			if _, err := io.ReadFull(fp, chunk); err != nil || size < 16 {
				return nil, fmt.Errorf("read wav fmt chunk: %w", errors.Join(err, io.ErrUnexpectedEOF))
			}

			format = binary.LittleEndian.Uint16(chunk[0:])
			channels = binary.LittleEndian.Uint16(chunk[2:])
			sampleRate = binary.LittleEndian.Uint32(chunk[4:])
			bits = binary.LittleEndian.Uint16(chunk[14:])

			// WAVE_FORMAT_EXTENSIBLE stores the real format at the start of the sub format GUID
			if format == 0xfffe && size >= 26 {
				format = binary.LittleEndian.Uint16(chunk[24:])
			}

			if size%2 == 1 {
				_, _ = fp.Seek(1, io.SeekCurrent)
			}

			continue
		}

		if id != "data" {
			// chunks are padded to an even size
			if _, err := fp.Seek(size+size%2, io.SeekCurrent); err != nil {
				return nil, fmt.Errorf("skip wav chunk %q: %w", id, err)
			}

			continue
		}

		var decode func([]byte) float32
		switch {
		case format == 1 && bits == 8:
			decode = func(b []byte) float32 { return (float32(b[0]) - 128) / 128 }
		case format == 1 && bits == 16:
			decode = pcm16
		case format == 1 && bits == 24:
			decode = func(b []byte) float32 {
				return float32(int32(uint32(b[0])<<8|uint32(b[1])<<16|uint32(b[2])<<24)) / (1 << 31)
			}
		case format == 3 && bits == 32:
			decode = func(b []byte) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b)) }
		default:
			return nil, fmt.Errorf("unsupported wav format %d with %d bits", format, bits)
		}

		start, err := fp.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, fmt.Errorf("get wav data offset: %w", err)
		}

		frameSize := int64(bits/8) * int64(channels)
		if frameSize == 0 {
			return nil, errors.New("wav has no channels")
		}

		data := io.NewSectionReader(readerAtFromSeeker(fp), start, size)

		return newStereoStream(int(channels), int(sampleRate), size/frameSize,
			pcmReader(data, decode, int(bits/8)),
			func(frame int64) error {
				_, err := data.Seek(frame*frameSize, io.SeekStart)
				return err
			},
		)
	}
}

func pcm16(b []byte) float32 {
	return float32(int16(binary.LittleEndian.Uint16(b))) / (1 << 15)
}

// pcmReader reads interleaved samples of the given byte size and converts them to float32.
func pcmReader(r io.Reader, decode func([]byte) float32, sampleSize int) func([]float32) (int, error) {
	var buf []byte

	return func(samples []float32) (int, error) {
		buf = append(buf[:0], make([]byte, len(samples)*sampleSize)...)

		n, err := io.ReadFull(r, buf)
		if errors.Is(err, io.ErrUnexpectedEOF) {
			err = nil
		}

		n /= sampleSize
		for idx := range n {
			samples[idx] = decode(buf[idx*sampleSize:])
		}

		return n, err
	}
}

type seekerReaderAt struct {
	io.ReadSeeker
}

// readerAtFromSeeker implements io.ReaderAt by seeking. Not safe for concurrent use,
// which is fine as every stream has its own reader.
func readerAtFromSeeker(r io.ReadSeeker) io.ReaderAt {
	if r, ok := r.(io.ReaderAt); ok {
		return r
	}

	return seekerReaderAt{r}
}

func (s seekerReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if _, err := s.Seek(off, io.SeekStart); err != nil {
		return 0, err
	}

	return io.ReadFull(s, p)
}

// stereoStream converts an interleaved mono or stereo source into a stereo Stream.
type stereoStream struct {
	channels int
	config   StreamConfig

	read func([]float32) (int, error)
	seek func(frame int64) error

	// current position in frames
	frame int64
	buf   []float32
}

func newStereoStream(
	channels, sampleRate int, frameCount int64,
	read func([]float32) (int, error),
	seek func(frame int64) error,
) (Stream[float32], error) {
	if channels != 1 && channels != 2 {
		return nil, fmt.Errorf("unsupported channel count %d", channels)
	}

	return &stereoStream{
		channels: channels,
		read:     read,
		seek:     seek,
		config: StreamConfig{
			SampleRate:         sampleRate,
			ChannelCount:       2,
			ChannelSampleCount: int(max(0, frameCount)),
		},
	}, nil
}

func (s *stereoStream) Read(samples []float32) (int, error) {
	frames := len(samples) / 2

	// read into samples directly for stereo, into a smaller buffer for mono
	target := samples[:frames*2]
	if s.channels == 1 {
		s.buf = append(s.buf[:0], make([]float32, frames)...)
		target = s.buf
	}

	// fill the target completely unless the source ends
	var n int
	var err error
	for n < len(target) && err == nil {
		var c int
		c, err = s.read(target[n:])
		n += c

		if c == 0 && err == nil {
			err = io.EOF
		}
	}

	if n > 0 && errors.Is(err, io.EOF) {
		err = nil
	}

	if s.channels == 1 {
		for idx := n - 1; idx >= 0; idx-- {
			samples[2*idx] = s.buf[idx]
			samples[2*idx+1] = s.buf[idx]
		}

		n *= 2
	}

	s.frame += int64(n / 2)

	return n, err
}

func (s *stereoStream) Seek(offset int64, whence int) (int64, error) {
	var base int64
	switch whence {
	case io.SeekCurrent:
		base = s.frame * 2
	case io.SeekEnd:
		base = int64(s.config.SampleCount())
	}

	frame := (base + offset) / 2
	if frame < 0 {
		return 0, errors.New("seek before start of stream")
	}

	if err := s.seek(frame); err != nil {
		return 0, err
	}

	s.frame = frame
	return frame * 2, nil
}

func (s *stereoStream) Config() StreamConfig {
	return s.config
}
