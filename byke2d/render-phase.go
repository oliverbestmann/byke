package byke2d

import (
	"iter"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/radix"
	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[BinnedRenderPhase[Opaque]]()
var _ = byke.ValidateComponent[SortableRenderPhase[Transparent]]()

type Transparent struct{}
type Opaque struct{}

func pluginRenderPhases(app *byke.App) {
	app.AddSystems(Core2d, byke.System(dispatchOpaqueRenderSystem).InSet(Core2dOpaque))
	app.AddSystems(Render, byke.System(cleanupBinnedRenderPhaseSystem[Opaque]).InSet(RenderPhaseCleanup))

	app.AddSystems(Core2d, byke.System(dispatchTransparentRenderSystem).InSet(Core2dTransparent))
	app.AddSystems(Render, byke.System(sortRenderPhasesSystem[Transparent]).InSet(RenderPhaseSort))
	app.AddSystems(Render, byke.System(cleanupSortableRenderPhaseSystem[Transparent]).InSet(RenderPhaseCleanup))

}

type Draw func(world *byke.World, pass *TrackedRenderPassEncoder, item RenderItem) (ok bool)

type SortableRenderPhase[M any] struct {
	byke.Component[SortableRenderPhase[M]]
	items      []RenderItem
	sortValues []float32
	index      []radix.Value
	sortCache  radix.Cache
}

//goland:noinspection GoMixedReceiverTypes
func (r SortableRenderPhase[M]) Dispatch(world *byke.World, pass *TrackedRenderPassEncoder) {
	for idx := uint32(0); idx < r.Len(); idx++ {
		item := r.Get(idx)
		item.Draw(world, pass, *item)

		if item.BatchCount > 0 {
			idx += item.BatchCount - 1
		}
	}
}

func (r *SortableRenderPhase[M]) Reset() {
	clear(r.items)

	r.index = r.index[:0]
	r.items = r.items[:0]
	r.sortValues = r.sortValues[:0]
}

func (r *SortableRenderPhase[M]) Append(item RenderItem, sortValue float32) {
	r.items = append(r.items, item)
	r.sortValues = append(r.sortValues, sortValue)
}

func (r *SortableRenderPhase[M]) Len() uint32 {
	return uint32(len(r.items))
}

func (r *SortableRenderPhase[M]) Get(idx uint32) *RenderItem {
	return &r.items[r.index[idx].Index]
}

func (r *SortableRenderPhase[M]) IsEmpty() bool {
	return len(r.items) == 0
}

type BinnedRenderPhase[M any] struct {
	byke.Component[BinnedRenderPhase[M]]
	items map[any][]RenderItem
	count int
}

//goland:noinspection GoMixedReceiverTypes
func (r BinnedRenderPhase[M]) Dispatch(world *byke.World, pass *TrackedRenderPassEncoder) {
	for _, values := range r.items {
		if len(values) == 0 {
			continue
		}

		values[0].Draw(world, pass, values[0])
	}
}

func (r *BinnedRenderPhase[M]) Reset() {
	for key, value := range r.items {
		clear(value)
		r.items[key] = value[:0]
	}

	r.count = 0
}

func (r *BinnedRenderPhase[M]) Append(item RenderItem, key any) {
	if r.items == nil {
		r.items = map[any][]RenderItem{}
	}

	r.items[key] = append(r.items[key], item)
	r.count += 1
}

func (r *BinnedRenderPhase[M]) Batches() iter.Seq2[any, []RenderItem] {
	return func(yield func(any, []RenderItem) bool) {
		for key, items := range r.items {
			if len(items) == 0 {
				continue
			}

			if !yield(key, items) {
				continue
			}
		}
	}
}

func (r *BinnedRenderPhase[M]) IsEmpty() bool {
	return r.count == 0
}

type RenderItem struct {
	Draw Draw
	Type any

	// e.g. the z value or distance from camera
	BatchBegin     uint32
	BatchCount     uint32
	ExtractedIndex uint32
}

type RenderPhase interface {
	Dispatch(world *byke.World, pass *wgpu.RenderPassEncoder)
}

func dispatchTransparentRenderSystem(
	world *byke.World,
	ctx *RenderContext,
	viewQuery ViewQuery[struct {
		Camera           *Camera
		ViewTarget       *ViewTarget
		ViewDepthTexture *ViewDepthTexture
		Phase            *SortableRenderPhase[Transparent]
	}],
) {
	view := viewQuery.Get()

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "MainRender"})
	defer enc.Release()

	pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		Label: "Sprites",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			view.ViewTarget.Attachment(),
		},
		DepthStencilAttachment: new(view.ViewDepthTexture.ReadOnly()),
	})
	defer pass.Release()

	tracked := &TrackedRenderPassEncoder{RenderPassEncoder: pass}
	view.Phase.Dispatch(world, tracked)

	pass.End()

	buf := enc.Finish(nil)
	defer buf.Release()

	ctx.Submit(buf)
}

func dispatchOpaqueRenderSystem(
	world *byke.World,
	ctx *RenderContext,
	viewQuery ViewQuery[struct {
		Camera           *Camera
		ViewTarget       *ViewTarget
		ViewDepthTexture *ViewDepthTexture
		Phase            *BinnedRenderPhase[Opaque]
	}],
) {
	view := viewQuery.Get()

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "MainRender"})
	defer enc.Release()

	pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		Label: "Sprites",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			view.ViewTarget.Attachment(),
		},
		DepthStencilAttachment: new(view.ViewDepthTexture.ReadWrite()),
	})
	defer pass.Release()

	tracked := &TrackedRenderPassEncoder{RenderPassEncoder: pass}

	view.Phase.Dispatch(world, tracked)

	pass.End()

	buf := enc.Finish(nil)
	defer buf.Release()

	ctx.Submit(buf)
}

func sortRenderPhasesSystem[M any](phases byke.Query[*SortableRenderPhase[M]]) {
	for phase := range phases.Items() {
		sortRenderPhase(phase)
	}
}

func sortRenderPhase[M any](phase *SortableRenderPhase[M]) {
	defer puffin.NewScope("Sort RenderPhase").End()

	n := len(phase.sortValues)
	if n == 0 {
		return
	}

	if cap(phase.index) < n {
		// not enough space, need to allocate
		phase.index = make([]radix.Value, n)
	} else {
		// enough space, we can re-use
		phase.index = phase.index[:n]
	}

	// remove bounds-check in loop below
	_ = phase.index[n-1]

	for idx := range n {
		phase.index[idx].Key = phase.sortValues[idx]
		phase.index[idx].Index = uint32(idx)
	}

	radix.Sort(&phase.sortCache, phase.index)
}

func cleanupSortableRenderPhaseSystem[M any](
	phases byke.Query[*SortableRenderPhase[M]]) {
	for phase := range phases.Items() {
		phase.Reset()
	}
}

func cleanupBinnedRenderPhaseSystem[M any](
	phases byke.Query[*BinnedRenderPhase[M]]) {
	for phase := range phases.Items() {
		phase.Reset()
	}
}
