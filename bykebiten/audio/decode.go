package audio

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"io"
)

func OpenStream(fp io.ReadSeeker) (AudioStream[float32], error) {
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

func VorbisStream(fp io.ReadSeeker) (AudioStream[float32], error) {
	s, err := vorbis.DecodeF32(fp)
	if err != nil {
		return nil, fmt.Errorf("open vorbis stream: %w", err)
	}

	audioStream := AdaptAudioStream[float32](s)
	return audioStream, nil
}

func Mp3Stream(fp io.ReadSeeker) (AudioStream[float32], error) {
	s, err := mp3.DecodeF32(fp)
	if err != nil {
		return nil, fmt.Errorf("open mp3 stream: %w", err)
	}

	audioStream := AdaptAudioStream[float32](s)
	return audioStream, nil
}

func WavStream(fp io.ReadSeeker) (AudioStream[float32], error) {
	s, err := wav.DecodeF32(fp)
	if err != nil {
		return nil, fmt.Errorf("open wav stream: %w", err)
	}

	audioStream := AdaptAudioStream[float32](s)
	return audioStream, nil
}
