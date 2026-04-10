package byke2d

import (
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
)

type viewUniform struct {
	ScreenToNDC   glm.Mat3f
	WorldToScreen glm.Mat3f
}

func (v *viewUniform) ToWGPU() []byte {
	var w wx.StructWriter
	w.AppendMat3f(v.ScreenToNDC)
	w.AppendMat3f(v.WorldToScreen)
	return w.Bytes()
}
