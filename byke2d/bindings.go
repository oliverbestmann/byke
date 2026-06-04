package byke2d

import (
	"github.com/oliverbestmann/webgpu/wgpu"
)

func Sequential(entries ...wgpu.BindGroupEntry) []wgpu.BindGroupEntry {
	for idx := range entries {
		if entries[idx].Binding == 0 {
			entries[idx].Binding = uint32(idx)
		}
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

func SequentialLayout(entries ...wgpu.BindGroupLayoutEntry) wgpu.BindGroupLayoutDescriptor {
	return SequentialLayoutWithLabel("", entries...)
}

func SequentialLayoutWithLabel(label string, entries ...wgpu.BindGroupLayoutEntry) wgpu.BindGroupLayoutDescriptor {
	for idx := range entries {
		if entries[idx].Binding == 0 {
			entries[idx].Binding = uint32(idx)
		}
	}

	return wgpu.BindGroupLayoutDescriptor{
		Label:   label,
		Entries: entries,
	}
}

type entry interface {
	wgpu.BindGroupLayoutEntry | wgpu.BindGroupEntry
}

func Indexed[T entry](idx uint32, e T) T {
	switch ptrE := any(&e).(type) {
	case *wgpu.BindGroupLayoutEntry:
		ptrE.Binding = idx
		return e

	case *wgpu.BindGroupEntry:
		ptrE.Binding = idx
		return e

	default:
		panic("unreachable")
	}
}

func BindingLayoutTexture1D(sampleType wgpu.TextureSampleType, multisampled bool) wgpu.BindGroupLayoutEntry {
	return wgpu.BindGroupLayoutEntry{
		Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment,
		Texture: wgpu.TextureBindingLayout{
			SampleType:    sampleType,
			ViewDimension: wgpu.TextureViewDimension1D,
			Multisampled:  multisampled,
		},
	}
}

func BindingLayoutTexture2D(sampleType wgpu.TextureSampleType, multisampled bool) wgpu.BindGroupLayoutEntry {
	return wgpu.BindGroupLayoutEntry{
		Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment,
		Texture: wgpu.TextureBindingLayout{
			SampleType:    sampleType,
			ViewDimension: wgpu.TextureViewDimension2D,
			Multisampled:  multisampled,
		},
	}
}

func BindingLayoutTexture3D(sampleType wgpu.TextureSampleType, multisampled bool) wgpu.BindGroupLayoutEntry {
	return wgpu.BindGroupLayoutEntry{
		Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment,
		Texture: wgpu.TextureBindingLayout{
			SampleType:    sampleType,
			ViewDimension: wgpu.TextureViewDimension3D,
			Multisampled:  multisampled,
		},
	}
}

func BindingLayoutSampler(samplerType wgpu.SamplerBindingType) wgpu.BindGroupLayoutEntry {
	return wgpu.BindGroupLayoutEntry{
		Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment,
		Sampler: wgpu.SamplerBindingLayout{
			Type: samplerType,
		},
	}
}

func BindingLayoutBuffer(bindingType wgpu.BufferBindingType, dynamicOffsets bool) wgpu.BindGroupLayoutEntry {
	return wgpu.BindGroupLayoutEntry{
		Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment,
		Buffer: wgpu.BufferBindingLayout{
			Type:             bindingType,
			HasDynamicOffset: dynamicOffsets,
		},
	}
}
