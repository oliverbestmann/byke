package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
)

var _ = byke.ValidateComponent[viewUniforms]()

type viewUniforms struct {
	byke.Component[viewUniforms]
	ScreenToNDC   glm.Mat3f
	WorldToScreen glm.Mat3f
}

func (v viewUniforms) ToWGPU() []byte {
	var w wx.StructWriter
	w.AppendMat3f(v.ScreenToNDC)
	w.AppendMat3f(v.WorldToScreen)
	return w.Bytes()
}
