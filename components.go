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

func BundleOf(components ...ErasedComponent) ErasedComponent {
	return &Bundle{Components: components}
}

type Bundle struct {
	Component[Bundle]
	Components []ErasedComponent
}

func flattenComponents(target []ErasedComponent, components ...ErasedComponent) []ErasedComponent {
	for _, component := range components {
		if bundle, ok := component.(*Bundle); ok {
			// recurse into the bundle and flatten its components
			target = flattenComponents(target, bundle.Components...)
		} else {
			target = append(target, component)
		}
	}

	return target
}
