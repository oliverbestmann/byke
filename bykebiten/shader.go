package bykebiten

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
)

var _ = byke.ValidateComponent[Shader]()
var _ = byke.ValidateComponent[ShaderInput]()

type Shader struct {
	byke.Component[Shader]
	Shader *ebiten.Shader
}

func (Shader) RequireComponents() []byke.ErasedComponent {
	return []byke.ErasedComponent{
		ShaderInput{},
	}
}

type ShaderInput struct {
	byke.Component[ShaderInput]

	// The input images that are provided to the shader. If you use a Shader
	// with a Sprite, the first image will be replaced with to Sprite.Image.
	Images [4]*ebiten.Image

	// The uniforms to provide to the shader
	Uniforms map[string]any
}

// Put sets the value of a uniform value
func (s *ShaderInput) Put(uniform string, value any) {
	if s.Uniforms == nil {
		s.Uniforms = map[string]any{}
	}

	s.Uniforms[uniform] = toUniformValue(value)
}

// DisableAutoUniforms is a marker component to indicate that byke should not
// automatically inject common uniform values such as Time or Transform
type DisableAutoUniforms struct {
	byke.ImmutableComponent[DisableAutoUniforms]
}

// SetUniformsFromStruct takes uniform values from a struct value. It iterates over the fields
// of the struct and sets the values of exported fields in ShaderInput.Uniforms
func (s *ShaderInput) SetUniformsFromStruct(value any) {
	rv := reflect.ValueOf(value)
	ty := rv.Type()

	if ty.Kind() != reflect.Struct {
		err := fmt.Errorf("UpdateUniforms must be called with a struct type, got %s", ty.Kind())
		panic(err)
	}

	for idx := range rv.NumField() {
		field := ty.Field(idx)
		if field.Anonymous || !field.IsExported() {
			continue
		}

		// get the fields value and copy it to the Uniforms
		fieldValue := rv.Field(idx).Interface()
		s.Put(field.Name, fieldValue)
	}
}

func toUniformValue(value any) any {
	// TODO also support arrays, but we need to use reflection to do that

	switch value := value.(type) {
	case gm.Vec:
		return [2]float32{float32(value.X), float32(value.Y)}

	case []gm.Vec:
		// TODO maybe go the not so unsafe way and map to float32?
		ptrToValues := unsafe.SliceData(value)
		return unsafe.Slice((*float64)(unsafe.Pointer(ptrToValues)), len(value)*2)

	case gm.Mat:
		return [4]float32{
			float32(value.XAxis.X), float32(value.XAxis.Y),
			float32(value.YAxis.X), float32(value.YAxis.Y),
		}

	case []gm.Mat:
		ptrToValues := unsafe.SliceData(value)
		return unsafe.Slice((*float64)(unsafe.Pointer(ptrToValues)), len(value)*4)

	case color.Color:
		r, g, b, a := value.Values()
		return [4]float32{r, g, b, a}

	default:
		return value
	}
}

func updateUniformsSystem(
	vt byke.VirtualTime,
	query byke.Query[struct {
		_ byke.Without[DisableAutoUniforms]

		Input           *ShaderInput
		GlobalTransform Transform
	}],
) {
	time := vt.Elapsed.Seconds()
	for item := range query.Items() {
		item.Input.Put("Time", time)
		item.Input.Put("Transform", gm.RotationMat(item.GlobalTransform.Rotation))
	}
}
