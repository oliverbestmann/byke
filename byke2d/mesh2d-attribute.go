package byke2d

import "github.com/oliverbestmann/webgpu/wgpu"

type VertexAttribute struct {
	Name   string
	Format wgpu.VertexFormat
}

func (v *VertexAttribute) EqualTo(other VertexAttribute) bool {
	return v.Name == other.Name
}

var VertexAttributeColor = VertexAttribute{
	Name:   "Color",
	Format: wgpu.VertexFormatFloat32x4,
}

var VertexAttributeUV = VertexAttribute{
	Name:   "UV",
	Format: wgpu.VertexFormatFloat32x2,
}

type VertexAttributeValue struct {
	Attribute VertexAttribute
	Value     []byte
}

type VertexAttributes []VertexAttributeValue

func (v *VertexAttributes) Values() []VertexAttributeValue {
	return *v
}

func (v *VertexAttributes) Insert(attr VertexAttribute, values []byte) {
	if existing := v.Get(attr); existing != nil {
		existing.Value = values
		return
	}

	*v = append(*v, VertexAttributeValue{
		Attribute: attr,
		Value:     values,
	})
}

func (v *VertexAttributes) Get(name VertexAttribute) *VertexAttributeValue {
	values := v.Values()
	for idx := range values {
		item := &values[idx]
		if item.Attribute.EqualTo(name) {
			return item
		}
	}

	return nil
}
