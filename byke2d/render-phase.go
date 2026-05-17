package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/radix"
	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func pluginRenderPhases(app *byke.App) {
	app.InsertResource(renderPhaseSortCache{})
	app.AddSystems(Core2d, byke.System(dispatchRenderSystem).InSet(RenderPhaseExecute))
	app.AddSystems(Render, byke.System(sortRenderPhasesSystem).InSet(RenderPhaseSort))
	app.AddSystems(Render, byke.System(cleanupRenderPhaseSystem).InSet(RenderPhaseCleanup))
}

type Draw func(world *byke.World, pass *wgpu.RenderPassEncoder, item RenderPhaseItem) (ok bool)

type RenderPhase struct {
	byke.Component[RenderPhase]
	items     []RenderPhaseItem
	index     []radix.Value
	sortCache radix.Cache
}

func (r *RenderPhase) Reset() {
	r.items = r.items[:0]
	r.index = r.index[:0]
}

func (r *RenderPhase) Append(item RenderPhaseItem) {
	r.items = append(r.items, item)
}

func (r *RenderPhase) Len() uint32 {
	return uint32(len(r.items))
}

func (r *RenderPhase) Get(idx uint32) *RenderPhaseItem {
	return &r.items[r.index[idx].Index]
}

func (r *RenderPhase) IsEmpty() bool {
	return len(r.items) == 0
}

type RenderPhaseItem struct {
	Draw Draw
	Type any

	// e.g. the z value or distance from camera
	SortValue      float32
	BatchBegin     uint32
	BatchCount     uint32
	ExtractedIndex uint32
}

func dispatchRenderSystem(
	world *byke.World,
	ctx *RenderContext,
	viewQuery ViewQuery[struct {
		Camera     *Camera
		ViewTarget *ViewTarget
		Phase      *RenderPhase
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
	})
	defer pass.Release()

	phase := view.Phase
	for idx := uint32(0); idx < phase.Len(); idx++ {
		item := phase.Get(idx)
		item.Draw(world, pass, *item)

		if item.BatchCount > 0 {
			idx += item.BatchCount - 1
		}
	}

	pass.End()

	buf := enc.Finish(nil)
	ctx.Submit(buf)
}

type renderPhaseSortCache struct {
	SortCache radix.Cache
}

func sortRenderPhasesSystem(phases byke.Query[*RenderPhase]) {
	for phase := range phases.Items() {
		sortRenderPhase(phase)
	}
}

func sortRenderPhase(phase *RenderPhase) {
	defer puffin.NewScope("Sort RenderPhase").End()

	n := len(phase.items)
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
		phase.index[idx].Key = phase.items[idx].SortValue
		phase.index[idx].Index = uint32(idx)
	}

	radix.Sort(&phase.sortCache, phase.index)
}

func cleanupRenderPhaseSystem(
	phases byke.Query[*RenderPhase]) {
	for phase := range phases.Items() {
		phase.Reset()
	}
}
