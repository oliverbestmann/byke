package byke2d

import (
	"fmt"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[DynamicOffset[ViewUniforms]]()

func ComponentUniformsPlugin[C WGPUComponent[C]](app *byke.App) {
	app.InsertResource(ComponentUniforms[C]{})

	app.AddSystems(Render, byke.
		System(writeComponentUniformsSystem[C]).
		InSet(RenderPhasePrepareResources))
}

type WGPUComponent[C byke.IsComponent[C]] interface {
	byke.IsComponent[C]
	comparable
	ToWGPU() []byte
}

type DynamicOffset[C WGPUComponent[C]] struct {
	byke.Component[DynamicOffset[C]]
	Offset uint32
}

type ComponentUniforms[C WGPUComponent[C]] struct {
	_ byke.NoCopy

	bytes []byte

	buffer     *wgpu.Buffer
	bufferSize uint64

	elementSize uint64
}

func (c *ComponentUniforms[C]) Binding() wgpu.BindGroupEntry {
	if !c.buffer.IsValid() {
		panic(fmt.Errorf("not initialized: %T", c))
	}

	return BindingBufferSize(c.buffer, 0, c.elementSize)
}

func (c *ComponentUniforms[C]) upload(ctx *RenderContext) {
	requiredSize := uint64(max(len(c.bytes), 4096))

	if c.bufferSize <= requiredSize {
		if c.buffer != nil {
			// release existing buffer reference
			c.buffer.Release()
		}

		var cZero C

		// create a new buffer
		c.buffer = ctx.CreateBuffer(&wgpu.BufferDescriptor{
			Label: fmt.Sprintf("Uniform Buffer %T", cZero),
			Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
			Size:  requiredSize,
		})

		c.bufferSize = requiredSize
	}

	// write data to buffer
	ctx.WriteBuffer(c.buffer, 0, c.bytes)
}

func (c *ComponentUniforms[C]) push(value *C) DynamicOffset[C] {
	// align offset to a multiple of 256
	if v := c.elementSize % 256; v > 0 {
		c.bytes = append(c.bytes, make([]byte, 256-v)...)
	}

	buf := (*value).ToWGPU()

	// remember element size and ensure all elements have the same size
	if len(c.bytes) == 0 {
		c.elementSize = uint64(len(buf))
	} else if uint64(len(buf)) != c.elementSize {
		panic(
			fmt.Errorf(
				"%T: all elements must have the same size, got %d, expected %d",
				*value, len(buf), c.elementSize,
			),
		)
	}

	offset := uint32(len(c.bytes))
	c.bytes = append(c.bytes, buf...)
	return DynamicOffset[C]{Offset: offset}
}

func (c *ComponentUniforms[C]) reset() {
	c.bytes = c.bytes[:0]
	c.elementSize = 0
}

func writeComponentUniformsSystem[C WGPUComponent[C]](
	commands *byke.Commands,
	ctx *RenderContext,
	uniforms *ComponentUniforms[C],
	values byke.Query[struct {
		EntityId byke.EntityId
		ValueRef query.Ref[C]
		Offset   byke.OptionMut[DynamicOffset[C]]
	}],
) {
	uniforms.reset()

	for value := range values.Items() {
		offset := uniforms.push(value.ValueRef.Get())

		mutOffset, ok := value.Offset.Get()
		if ok {
			*mutOffset = offset
		} else {
			commands.Entity(value.EntityId).Insert(offset)
		}
	}

	uniforms.upload(ctx)
}
