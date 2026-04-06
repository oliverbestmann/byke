package byke2d

type AppExit struct {
	error
}

var AppExitSuccess = AppExit{}
