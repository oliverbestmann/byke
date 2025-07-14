package bykebiten

import "github.com/hajimehoshi/ebiten/v2"

type AppExit struct {
	error
}

var AppExitSuccess = AppExit{error: ebiten.Termination}
