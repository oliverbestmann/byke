//go:build js

package audio

import (
	"fmt"
	"io"
	"syscall/js"

	"github.com/oliverbestmann/webgpu/jsx"
)

// browserSampleRate is the rate the browser resamples to, it matches byke2d.StereoSamplesPerSecond.
const browserSampleRate = 48000

// OpenStream decodes the complete file with the browser's decodeAudioData
// into stereo samples at browserSampleRate.
func OpenStream(fp io.ReadSeeker) (Stream[float32], error) {
	data, err := io.ReadAll(fp)
	if err != nil {
		return nil, fmt.Errorf("read audio file: %w", err)
	}

	bufIn := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(bufIn, data)

	result, ok := jsx.Await(decodeAudioAsync.Invoke(bufIn, browserSampleRate))
	if !ok {
		return nil, fmt.Errorf("decode audio: %w", js.Error{Value: result})
	}

	samples := make([]float32, result.Get("length").Int()/4)
	js.CopyBytesToGo(SamplesAsBytes(samples), result)

	config := StreamConfig{
		SampleRate:         browserSampleRate,
		ChannelCount:       2,
		ChannelSampleCount: len(samples) / 2,
	}

	return FromSamples(config, samples), nil
}

var decodeAudioAsync = js.Global().Get("Function").New("bytes", "sampleRate", `
	return (async function() {
		const ctx = new OfflineAudioContext(2, 1, sampleRate);
		const audio = await ctx.decodeAudioData(bytes.buffer);

		const left = audio.getChannelData(0);
		const right = audio.numberOfChannels > 1 ? audio.getChannelData(1) : left;

		const out = new Float32Array(audio.length * 2);
		for (let idx = 0; idx < audio.length; idx++) {
			out[2 * idx] = left[idx];
			out[2 * idx + 1] = right[idx];
		}

		return new Uint8Array(out.buffer);
	})();
`)
