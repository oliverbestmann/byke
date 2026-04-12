package byke2d

import "github.com/oliverbestmann/byke"

func clearViewTargetSystem(
	renderContext *RenderContext,
	viewTarget *ViewTarget,
	cameras byke.Query[struct {
		Camera     Camera
		ClearColor ClearColor
	}],
) {
	for camera := range cameras.Items() {
		if camera.Camera.Inactive {
			continue
		}

		// clearTexture(renderContext, viewTarget.Target, camera.ClearColor.Color)
		break
	}
}
