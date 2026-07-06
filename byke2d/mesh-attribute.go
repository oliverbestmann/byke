package byke2d

import (
	"hash/maphash"
	"slices"

	"github.com/oliverbestmann/webgpu/wgpu"
)

// VertexAttribute describes a single attribute of a vertex (e.g., position, normal, color).
// It specifies the data format and shader location for the attribute.
type VertexAttribute struct {
	// Name is the semantic name of this attribute (e.g., "Position", "Normal").
	Name string

	// Format specifies the data type and size of this attribute in the vertex buffer.
	Format wgpu.VertexFormat

	// Location is the shader layout location where this attribute is bound.
	Location uint32
}

// EqualTo reports whether this attribute has the same name as another attribute.
func (v *VertexAttribute) EqualTo(other VertexAttribute) bool {
	return v.Name == other.Name
}

// Size returns the byte size of this attribute value.
func (v *VertexAttribute) Size() uint32 {
	return v.Format.ByteSize()
}

// VertexAttributePosition is the standard vertex position attribute (3 float32 values).
var VertexAttributePosition = VertexAttribute{
	Name:     "Position",
	Format:   wgpu.VertexFormatFloat32x3,
	Location: 7,
}

// VertexAttributeColor is the standard vertex color attribute (4 float32 values).
var VertexAttributeColor = VertexAttribute{
	Name:     "Color",
	Format:   wgpu.VertexFormatFloat32x4,
	Location: 8,
}

// VertexAttributeUV is the standard primary UV texture coordinate attribute (2 float32 values).
var VertexAttributeUV = VertexAttribute{
	Name:     "UV",
	Format:   wgpu.VertexFormatFloat32x2,
	Location: 9,
}

// VertexAttributeUV1 is a secondary UV texture coordinate attribute (2 float32 values).
var VertexAttributeUV1 = VertexAttribute{
	Name:     "UV1",
	Format:   wgpu.VertexFormatFloat32x2,
	Location: 10,
}

// VertexAttributeUV2 is a tertiary UV texture coordinate attribute (2 float32 values).
var VertexAttributeUV2 = VertexAttribute{
	Name:     "UV2",
	Format:   wgpu.VertexFormatFloat32x2,
	Location: 11,
}

// VertexAttributeNormal is the standard vertex surface normal attribute (3 float32 values).
var VertexAttributeNormal = VertexAttribute{
	Name:     "Normal",
	Format:   wgpu.VertexFormatFloat32x3,
	Location: 12,
}

// VertexAttributeTangentSpace stores tangent and bitangent information for normal mapping (4 float32 values).
var VertexAttributeTangentSpace = VertexAttribute{
	Name:     "TangentSpace",
	Format:   wgpu.VertexFormatFloat32x4,
	Location: 13,
}

// VertexAttributeJoints specifies which skeleton joints influence this vertex (4 uint16 values).
var VertexAttributeJoints = VertexAttribute{
	Name:     "Joints",
	Format:   wgpu.VertexFormatUint16x4,
	Location: 14,
}

// VertexAttributeJointWeights specifies the blend weight for each joint influencing this vertex (4 float32 values).
var VertexAttributeJointWeights = VertexAttribute{
	Name:     "JointWeights",
	Format:   wgpu.VertexFormatFloat32x4,
	Location: 15,
}

// VertexAttributeValue pairs a vertex attribute descriptor with its actual byte data.
type VertexAttributeValue struct {
	// Attribute describes the attribute format and location.
	Attribute VertexAttribute

	// Value contains the raw byte data for all vertices of this attribute.
	Value []byte
}

// VertexAttributes is a collection of vertex attribute values for a mesh.
type VertexAttributes []VertexAttributeValue

// Values returns the underlying slice of attribute values.
func (v *VertexAttributes) Values() []VertexAttributeValue {
	return *v
}

// Insert adds or updates an attribute in this collection.
// If the attribute already exists, its values are replaced; otherwise it is appended.
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

// Get retrieves the attribute value for the given attribute descriptor.
// Returns nil if the attribute is not present.
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

// Has reports whether this collection contains the given attribute.
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

// VertexLayout describes the complete structure of vertex data, including all attributes,
// their order, and the total size of a single vertex. Layouts are cached by hash to enable
// efficient deduplication and reuse.
type VertexLayout struct {
	// Attributes are the vertex attributes in this layout, sorted by location.
	Attributes []VertexAttribute

	// key is a hash of the attributes for deduplication and comparison.
	key VertexLayoutKey
}

// NewVertexLayout creates a new vertex layout from a set of attributes.
// Attributes are automatically sorted by location for consistent GPU binding.
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

// Key returns a unique hash key for this layout, used for efficient caching and comparison.
func (v VertexLayout) Key() VertexLayoutKey {
	return v.key
}

// Size returns the total byte size of a single vertex in this layout.
func (v VertexLayout) Size() (size uint32) {
	for _, attr := range v.Attributes {
		size += attr.Format.ByteSize()
	}

	return
}

// EqualTo reports whether this layout is identical to another layout.
func (v VertexLayout) EqualTo(other VertexLayout) bool {
	return v.key == other.key && slices.Equal(v.Attributes, other.Attributes)
}
