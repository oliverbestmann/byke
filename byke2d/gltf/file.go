package gltf

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"unsafe"

	"github.com/oliverbestmann/byke/byke2d/glm"
)

var bin = &binary.LittleEndian

type Ref uint32

func (r Ref) LogValue() slog.Value {
	return slog.IntValue(int(r))
}

type OptionRef struct {
	IsSet bool
	Value Ref
}

func (o *OptionRef) UnmarshalJSON(bytes []byte) error {
	if err := json.Unmarshal(bytes, &o.Value); err != nil {
		return fmt.Errorf("read OptionRef: %w", err)
	}

	o.IsSet = true
	return nil
}

func (o *OptionRef) IsNil() bool {
	return !o.IsSet
}

func (o *OptionRef) Get() Ref {
	if !o.IsSet {
		panic("optional ref not set")
	}

	return o.Value
}

type Asset struct {
	Generator string `json:"generator"`
	Version   string `json:"version"`
}

type Node struct {
	Id          Ref          `json:"-"`
	Name        string       `json:"name"`
	Mesh        OptionRef    `json:"mesh"`
	Skin        OptionRef    `json:"skin"`
	Camera      OptionRef    `json:"camera"`
	Matrix      *[16]float32 `json:"matrix"`
	Rotation    *[4]float32  `json:"rotation"`
	Scale       *[3]float32  `json:"scale"`
	Translation *[3]float32  `json:"translation"`
	Children    []Ref        `json:"children"`
	Extensions  Extensions   `json:"extensions"`
}

func (n *Node) TransformComponents() (translation, scale glm.Vec3f, rotation glm.Quat) {
	scale = glm.Vec3f{1, 1, 1}
	rotation = glm.IdentityQuat()

	if t := n.Translation; t != nil {
		translation = *t
	}

	if s := n.Scale; s != nil {
		scale = *s
	}

	if r := n.Rotation; r != nil {
		rotation = glm.QuatOf(r[0], r[1], r[2], r[3])
	}

	return
}

func (n *Node) Transform() glm.Mat4f {
	if m := n.Matrix; m != nil {
		return glm.Mat4f{
			{m[0], m[1], m[2], m[3]},
			{m[4], m[5], m[6], m[7]},
			{m[8], m[9], m[10], m[11]},
			{m[12], m[13], m[15], m[15]},
		}
	}

	translation, scale, rotation := n.TransformComponents()

	return glm.TranslationMat4f(translation.XYZ()).
		Mul(rotation.ToMat4()).
		Scale(scale.XYZ())
}

type Image struct {
	Name       string `json:"name"`
	MimeType   string `json:"mimeType"`
	Uri        string `json:"uri"`
	BufferView Ref    `json:"bufferView"`
}

type Texture struct {
	Name    string `json:"name"`
	Sampler Ref    `json:"sampler"`
	Source  Ref    `json:"source"`
}

type TextureInfo struct {
	Index    Ref     `json:"index"`
	TexCoord uint32  `json:"texCoord"`
	Scale    float32 `json:"scale"`
}

type Sampler struct {
	Name      string `json:"name"`
	MagFilter uint32 `json:"magFilter"`
	MinFilter uint32 `json:"minFilter"`
	WrapS     uint32 `json:"wrapS"`
	WrapT     uint32 `json:"wrapT"`
}

type Scene struct {
	Name  string `json:"name"`
	Nodes []Ref  `json:"nodes"`
}

type Camera struct {
	Name string `json:"name"`
}

type Mesh struct {
	Name       string          `json:"name"`
	Primitives []MeshPrimitive `json:"primitives"`
	Weights    []float32
	Extras     MeshExtras `json:"extras"`
}

type MeshExtras struct {
	TargetNames []string `json:"targetNames"`
}

type MorphTarget struct {
	Positions OptionRef `json:"POSITION"`
	Normals   OptionRef `json:"NORMAL"`
	Tangents  OptionRef `json:"TANGENT"`
}

type MeshPrimitive struct {
	Attributes map[string]Ref `json:"attributes"`
	Indices    OptionRef      `json:"indices"`
	Material   OptionRef      `json:"material"`
	Targets    []MorphTarget  `json:"targets"`
	Mode       uint32         `json:"mode"`
}

func (p MeshPrimitive) MustGet(key string) Ref {
	id, ok := p.Attributes[key]
	if !ok {
		panic(fmt.Errorf("attribute %q not found", key))
	}

	return id
}

type Accessor struct {
	Name          string `json:"name"`
	BufferView    Ref    `json:"bufferView"`
	ByteOffset    uint32 `json:"byteOffset"`
	ComponentType uint32 `json:"componentType"`
	Count         uint32 `json:"count"`
	Type          string `json:"type"`
}

func (h *Handle) Scene(id Ref) []Node {
	var nodes []Node
	for _, nid := range h.Scenes[id].Nodes {
		nodes = append(nodes, h.Nodes[nid])
	}

	return nodes
}

func (h *Handle) ChildNodes(parent Node) []Node {
	var nodes []Node
	for _, nid := range parent.Children {
		nodes = append(nodes, h.Nodes[nid])
	}

	return nodes
}

func (h *Handle) Buffer(viewId Ref) []byte {
	bufferView := &h.BufferViews[viewId]
	offset := bufferView.ByteOffset
	return h.binary[offset : offset+bufferView.ByteLength]
}

//goland:noinspection DuplicatedCode
func (h *Handle) Resolve(aid Ref) any {
	acc := &h.Accessors[aid]

	const Byte = 5120
	const UnsignedByte = 5121
	const Short = 5122
	const UnsignedShort = 5123
	const UnsignedInt = 5125
	const Float = 5126

	count := acc.Count

	if acc.ComponentType == Float {
		if acc.Type == "SCALAR" {
			buf := h.BytesForAccessor(acc, 4)
			return castToType[float32](buf, count)
		}

		if acc.Type == "VEC2" {
			buf := h.BytesForAccessor(acc, 8)
			return castToType[glm.Vec2f](buf, count)
		}

		if acc.Type == "VEC3" {
			buf := h.BytesForAccessor(acc, 12)
			return castToType[glm.Vec3f](buf, count)
		}

		if acc.Type == "VEC4" {
			buf := h.BytesForAccessor(acc, 16)
			return castToType[glm.Vec4f](buf, count)
		}

		if acc.Type == "MAT2" {
			buf := h.BytesForAccessor(acc, 16)
			return castToType[glm.Mat2f](buf, count)
		}

		if acc.Type == "MAT3" {
			buf := h.BytesForAccessor(acc, 36)
			return castToType[glm.Mat3f](buf, count)
		}

		if acc.Type == "MAT4" {
			buf := h.BytesForAccessor(acc, 64)
			return castToType[glm.Mat4f](buf, count)
		}
	}

	if acc.ComponentType == UnsignedInt {
		if acc.Type == "SCALAR" {
			buf := h.BytesForAccessor(acc, 4)
			return castToType[uint32](buf, count)
		}

		if acc.Type == "VEC2" {
			buf := h.BytesForAccessor(acc, 8)
			return castToType[glm.Vec2u](buf, count)
		}

		if acc.Type == "VEC3" {
			buf := h.BytesForAccessor(acc, 12)
			return castToType[glm.Vec3u](buf, count)
		}

		if acc.Type == "VEC4" {
			buf := h.BytesForAccessor(acc, 16)
			return castToType[glm.Vec4u](buf, count)
		}
	}

	if acc.ComponentType == UnsignedShort {
		if acc.Type == "SCALAR" {
			buf := h.BytesForAccessor(acc, 2)
			return castToType[uint16](buf, count)
		}

		if acc.Type == "VEC2" {
			buf := h.BytesForAccessor(acc, 4)
			return castToType[glm.Vec2uh](buf, count)
		}

		if acc.Type == "VEC3" {
			buf := h.BytesForAccessor(acc, 6)
			return castToType[glm.Vec3uh](buf, count)
		}

		if acc.Type == "VEC4" {
			buf := h.BytesForAccessor(acc, 8)
			return castToType[glm.Vec4uh](buf, count)
		}
	}

	panic(fmt.Errorf("can not resolve type=%q, format=%d", acc.Type, acc.ComponentType))
}

func (h *Handle) BytesForAccessor(acc *Accessor, expectedStride uint32) []byte {

	bufferView := &h.BufferViews[acc.BufferView]
	if expectedStride > 0 && bufferView.ByteStride > 0 && bufferView.ByteStride != expectedStride {
		panic(fmt.Errorf("expected byteStride %d, got %d", expectedStride, bufferView.ByteStride))
	}

	offset := bufferView.ByteOffset + acc.ByteOffset
	return h.binary[offset : offset+bufferView.ByteLength]
}

func castToType[T any](buf []byte, count uint32) []T {
	var tZero T
	tSize := int(unsafe.Sizeof(tZero))
	if len(buf)%tSize != 0 {
		panic(fmt.Errorf("failed to cast buffer of length %d to %T", len(buf), tZero))
	}

	n := len(buf) / tSize
	if int(count) > n {
		panic(fmt.Errorf("expected at least %d elements, got %d", count, n))
	}

	ptr := unsafe.Pointer(unsafe.SliceData(buf))
	return unsafe.Slice((*T)(ptr), count)
}

type BufferView struct {
	Buffer     uint32
	ByteOffset uint32
	ByteLength uint32
	ByteStride uint32
}

type Material struct {
	Name              string             `json:"name"`
	MetallicRoughness *MetallicRoughness `json:"pbrMetallicRoughness"`
	EmissiveFactor    glm.Vec3f          `json:"emissiveFactor"`
	NormalTexture     *TextureInfo       `json:"normalTexture"`
	OcclusionTexture  *TextureInfo       `json:"occlusionTexture"`
}

func (m *Material) BaseColor() glm.Vec4f {
	if m.MetallicRoughness == nil {
		return glm.Vec4f{1, 1, 1, 1}
	}

	if m.MetallicRoughness.BaseColorFactor == nil {
		return glm.Vec4f{1, 1, 1, 1}
	}

	return *m.MetallicRoughness.BaseColorFactor
}

type MetallicRoughness struct {
	BaseColorFactor  *[4]float32  `json:"baseColorFactor"`
	BaseColorTexture *TextureInfo `json:"baseColorTexture"`
}

type Animation struct {
	Name     string             `json:"name"`
	Channels []AnimationChannel `json:"channels"`
	Samplers []AnimationSampler `json:"samplers"`
}

type AnimationChannel struct {
	Target  AnimationTarget `json:"target"`
	Sampler Ref             `json:"sampler"`
}

type AnimationTarget struct {
	Path string `json:"path"`
	Node Ref    `json:"node"`
}

type AnimationSampler struct {
	Interpolation string `json:"interpolation"`
	Input         Ref    `json:"input"`
	Output        Ref    `json:"output"`
}

type Skin struct {
	Name                string    `json:"name"`
	Skeleton            OptionRef `json:"skeleton"`
	Joints              []Ref     `json:"joints"`
	InverseBindMatrices OptionRef `json:"inverseBindMatrices"`
}

type fileContent struct {
	Asset Asset `json:"asset"`

	Animations  []Animation     `json:"animations"`
	Accessors   []Accessor      `json:"accessors"`
	BufferViews []BufferView    `json:"bufferViews"`
	Cameras     []Camera        `json:"cameras"`
	Images      []Image         `json:"images"`
	Materials   []Material      `json:"materials"`
	Meshes      []Mesh          `json:"meshes"`
	Nodes       []Node          `json:"nodes"`
	Samplers    []Sampler       `json:"samplers"`
	Scene       Ref             `json:"scene"`
	Scenes      []Scene         `json:"scenes"`
	Textures    []Texture       `json:"textures"`
	Skins       []Skin          `json:"skins"`
	Extensions  json.RawMessage `json:"extensions"`
}

type Handle struct {
	fileContent
	binary []byte
}

func Load(r io.Reader) (*Handle, error) {
	remaining, err := readHeader(r)
	if err != nil {
		return nil, err
	}

	// remove header from remaining bytes
	remaining -= 12

	// first chunk must be the json chunk
	jsonChunk, err := readChunk(r, 0x4E4F534A)
	if err != nil {
		return nil, fmt.Errorf("read json chunk: %w", err)
	}

	remaining -= 8 + uint32(len(jsonChunk))

	var binaryChunk []byte
	if remaining > 0 {
		chunk, err := readChunk(r, 0x004E4942)
		if err != nil {
			return nil, fmt.Errorf("read binary chunk: %w", err)
		}

		binaryChunk = chunk
	}

	// parse the json chunk
	var content fileContent
	if err := json.Unmarshal(jsonChunk, &content); err != nil {
		return nil, fmt.Errorf("decode gltf json: %w", err)
	}

	// set node ids
	for idx := range content.Nodes {
		content.Nodes[idx].Id = Ref(idx)
	}

	handle := &Handle{
		fileContent: content,
		binary:      binaryChunk,
	}

	return handle, nil
}

func readChunk(r io.Reader, expectedChunkType uint32) ([]byte, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("read chunk header: %w", err)
	}

	chunkSize := bin.Uint32(header[0:4])
	chunkType := bin.Uint32(header[4:8])

	if chunkType != expectedChunkType {
		return nil, fmt.Errorf("unexpected chunk type %#x", chunkType)
	}

	buf := make([]byte, int(chunkSize))
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read chunk %d bytes of data: %w", chunkSize, err)
	}

	return buf, nil
}

func readHeader(r io.Reader) (length uint32, err error) {
	var header [12]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}

	magic := bin.Uint32(header[0:4])
	if magic != 0x46546C67 {
		return 0, errors.New("not a glTF file")
	}

	version := bin.Uint32(header[4:8])
	if version != 2 {
		return 0, fmt.Errorf("expected version 2, got %d", version)
	}

	length = bin.Uint32(header[8:12])
	return length, nil
}

type Light struct {
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Color     [3]float32 `json:"color"`
	Intensity float32    `json:"intensity"`
}

type KHRLightsPunctualInFile struct {
	Lights []Light `json:"lights"`
}

type KHRLightsPunctualInNode struct {
	Light Ref `json:"light"`
}

type Extensions map[string]json.RawMessage

func ExtensionOf[T any](e Extensions, name string) (*T, error) {
	encoded, ok := e[name]
	if !ok {
		return nil, nil
	}

	var tValue T

	if err := json.Unmarshal(encoded, &tValue); err != nil {
		return nil, fmt.Errorf("decode %T: %w", tValue, err)
	}

	return new(tValue), nil
}

type AllExtensions struct {
	Lights KHRLightsPunctualInFile `json:"KHR_lights_punctual"`
}
