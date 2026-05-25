package byke2d

import "github.com/oliverbestmann/webgpu/wgpu"

type vertexAttributeOffsets struct {
	index  uint32
	offset uint64
}

func (o *vertexAttributeOffsets) Inc(size uint64, fmt wgpu.VertexFormat) wgpu.VertexAttribute {
	attr := wgpu.VertexAttribute{
		ShaderLocation: o.index,
		Offset:         o.offset,
		Format:         fmt,
	}

	o.index += 1
	o.offset += size

	return attr
}
