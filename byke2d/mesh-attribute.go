package byke2d

import (
	"hash/maphash"
	"slices"

	"github.com/oliverbestmann/webgpu/wgpu"
)

type VertexAttribute struct {
	Name     string
	Format   wgpu.VertexFormat
	Location uint32
}

func (v *VertexAttribute) EqualTo(other VertexAttribute) bool {
	return v.Name == other.Name
}

func (v *VertexAttribute) Size() uint32 {
	return v.Format.ByteSize()
}

var VertexAttributePosition = VertexAttribute{
	Name:     "Position",
	Format:   wgpu.VertexFormatFloat32x3,
	Location: 9,
}

var VertexAttributeColor = VertexAttribute{
	Name:     "Color",
	Format:   wgpu.VertexFormatFloat32x4,
	Location: 10,
}

var VertexAttributeUV = VertexAttribute{
	Name:     "UV",
	Format:   wgpu.VertexFormatFloat32x2,
	Location: 11,
}

var VertexAttributeUV1 = VertexAttribute{
	Name:     "UV1",
	Format:   wgpu.VertexFormatFloat32x2,
	Location: 12,
}

var VertexAttributeUV2 = VertexAttribute{
	Name:     "UV2",
	Format:   wgpu.VertexFormatFloat32x2,
	Location: 13,
}

var VertexAttributeNormal = VertexAttribute{
	Name:     "Normal",
	Format:   wgpu.VertexFormatFloat32x3,
	Location: 14,
}

var VertexAttributeTangentSpace = VertexAttribute{
	Name:     "TangentSpace",
	Format:   wgpu.VertexFormatFloat32x4,
	Location: 15,
}

var VertexAttributeJoints = VertexAttribute{
	Name:     "Joints",
	Format:   wgpu.VertexFormatUint16x4,
	Location: 16,
}

var VertexAttributeJointWeights = VertexAttribute{
	Name:     "JointWeights",
	Format:   wgpu.VertexFormatFloat32x4,
	Location: 17,
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

func (v *VertexAttributes) Has(name VertexAttribute) bool {
	for _, attr := range *v {
		if attr.Attribute.EqualTo(name) {
			return true
		}
	}

	return false
}

var seed = maphash.MakeSeed()

type VertexLayoutKey uint64

type VertexLayout struct {
	Attributes []VertexAttribute
	key        VertexLayoutKey
}

func NewVertexLayout(attrs []VertexAttribute) VertexLayout {
	compare := func(lhs, rhs VertexAttribute) int {
		return int(lhs.Location) - int(rhs.Location)
	}

	sortedAttributes := slices.SortedFunc(slices.Values(attrs), compare)

	var h maphash.Hash
	h.SetSeed(seed)
	for _, attr := range sortedAttributes {
		maphash.WriteComparable(&h, attr)
	}

	return VertexLayout{
		Attributes: sortedAttributes,
		key:        VertexLayoutKey(h.Sum64()),
	}
}

func (v VertexLayout) Key() VertexLayoutKey {
	return v.key
}

func (v VertexLayout) Size() (size uint32) {
	for _, attr := range v.Attributes {
		size += attr.Format.ByteSize()
	}

	return
}

func (v VertexLayout) EqualTo(other VertexLayout) bool {
	return slices.Equal(v.Attributes, other.Attributes)
}
