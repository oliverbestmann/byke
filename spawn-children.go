package byke

func SpawnChild(components ...ErasedComponent) ErasedComponent {
	return &spawnChildComponent{
		Components: components,
	}
}

type spawnChildComponent struct {
	Component[spawnChildComponent]
	Components []ErasedComponent
}
