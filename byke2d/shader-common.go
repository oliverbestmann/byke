package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
)

var _ = byke.ValidateComponent[ViewUniforms]()

type ViewUniforms struct {
	byke.Component[ViewUniforms]

	ViewportOrigin glm.Vec2f
	ViewportSize   glm.Vec2f

	CameraToScreen    glm.Mat4f
	CameraToScreenInv glm.Mat4f
	WorldToCamera     glm.Mat4f
	WorldToCameraInv  glm.Mat4f
	WorldToScreen     glm.Mat4f
	WorldToScreenInv  glm.Mat4f
}

func (v ViewUniforms) ToWGPU() []byte {
	var w wgsl.StructWriter

	w.AppendVec4f(glm.Vec4f{
		v.ViewportOrigin[0], v.ViewportOrigin[1],
		v.ViewportSize[0], v.ViewportSize[1],
	})

	w.AppendMat4f(v.CameraToScreen)
	w.AppendMat4f(v.CameraToScreenInv)
	w.AppendMat4f(v.WorldToCamera)
	w.AppendMat4f(v.WorldToCameraInv)
	w.AppendMat4f(v.WorldToScreen)
	w.AppendMat4f(v.WorldToScreenInv)

	return w.Bytes()
}
