//go:build !js

package audio

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenStream(t *testing.T) {
	files, _ := filepath.Glob("testdata/sine-*")
	if len(files) == 0 {
		t.Fatal("no test files")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			fp, err := os.Open(file)
			if err != nil {
				t.Fatal(err)
			}

			defer fp.Close()

			stream, err := OpenStream(fp)
			if err != nil {
				t.Fatal(err)
			}

			config := stream.Config()
			if config.SampleRate != 44100 || config.ChannelCount != 2 {
				t.Fatalf("unexpected config %+v", config)
			}

			samples := readAll(t, stream)

			// 50ms of audio, mp3 is padded to full mp3 frames of 1152 samples
			frames := len(samples) / 2
			if frames < 2205 || frames > 4*1152 {
				t.Fatalf("unexpected frame count %d", frames)
			}

			// ffmpeg's sine source has an amplitude of 1/8, about 0.09 after upmixing to stereo.
			// Both channels are equal.
			var peak float32
			for idx := 0; idx < len(samples); idx += 2 {
				if samples[idx] != samples[idx+1] {
					t.Fatalf("channels differ at frame %d", idx/2)
				}

				peak = max(peak, samples[idx])
			}

			if peak < 0.08 || peak > 0.15 {
				t.Fatalf("unexpected peak %f", peak)
			}

			// seeking to a frame must give the same samples. Vorbis and mp3 seek
			// to the exact frame too, so we can compare all formats.
			pos, err := stream.Seek(1000, io.SeekStart)
			if err != nil || pos != 1000 {
				t.Fatalf("seek: %d, %v", pos, err)
			}

			rest := readAll(t, stream)
			if len(rest) != len(samples)-1000 {
				t.Fatalf("read %d samples after seek, expected %d", len(rest), len(samples)-1000)
			}

			for idx := range rest {
				if diff := rest[idx] - samples[1000+idx]; diff > 1e-3 || diff < -1e-3 {
					t.Fatalf("sample %d differs after seek: %f != %f", idx, rest[idx], samples[1000+idx])
				}
			}
		})
	}
}

func readAll(t *testing.T, stream Stream[float32]) []float32 {
	var samples []float32

	buf := make([]float32, 1000)
	for {
		n, err := stream.Read(buf)
		samples = append(samples, buf[:n]...)

		if err == io.EOF || (err == nil && n == 0) {
			return samples
		}

		if err != nil {
			t.Fatal(err)
		}
	}
}
