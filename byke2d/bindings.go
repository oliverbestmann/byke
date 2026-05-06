package byke2d

import "github.com/oliverbestmann/webgpu/wgpu"

func Sequential(entries ...wgpu.BindGroupEntry) []wgpu.BindGroupEntry {
	for idx := range entries {
		entries[idx].Binding = uint32(idx)
	}

	return entries
}

func BindingTextureView(value *wgpu.TextureView) wgpu.BindGroupEntry {
	return wgpu.BindGroupEntry{TextureView: value}
}

func BindingSampler(value *wgpu.Sampler) wgpu.BindGroupEntry {
	return wgpu.BindGroupEntry{Sampler: value}
}

func BindingBuffer(value *wgpu.Buffer) wgpu.BindGroupEntry {
	return wgpu.BindGroupEntry{Buffer: value, Size: wgpu.WholeSize}
}

func BindingBufferSize(value *wgpu.Buffer, offset, size uint64) wgpu.BindGroupEntry {
	return wgpu.BindGroupEntry{Buffer: value, Offset: offset, Size: size}
}
