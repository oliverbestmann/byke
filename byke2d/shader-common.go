package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
)

var _ = byke.ValidateComponent[ViewUniforms]()

type ViewUniforms struct {
	byke.Component[ViewUniforms]
	ScreenToNDC   glm.Mat4f
	WorldToScreen glm.Mat4f
}

func (v ViewUniforms) ToWGPU() []byte {
	var w wx.StructWriter
	w.AppendMat4f(v.ScreenToNDC)
	w.AppendMat4f(v.WorldToScreen)
	return w.Bytes()
}
