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

func Bundle(components ...ErasedComponent) ErasedComponent {
	return &bundleComponent{Components: components}
}

type bundleComponent struct {
	Component[bundleComponent]
	Components []ErasedComponent
}

func flattenComponents(target []ErasedComponent, components ...ErasedComponent) []ErasedComponent {
	for _, component := range components {
		if bundle, ok := component.(*bundleComponent); ok {
			// recurse into the bundle and flatten its components
			target = flattenComponents(target, bundle.Components...)
		} else {
			target = append(target, component)
		}
	}

	return target
}
