[![Reference byke](https://pkg.go.dev/badge/github.com/oliverbestmann/byke.svg)](https://pkg.go.dev/github.com/oliverbestmann/byke)
[![Reference bykebiten](https://pkg.go.dev/badge/github.com/oliverbestmann/byke/bykebiten.svg)](https://pkg.go.dev/github.com/oliverbestmann/byke/bykebiten)

# byke

**byke** is an Entity Component System (ECS) library for Go, inspired by the [Bevy](https://bevy.org/) API.

> Although still under development, it already includes a wide range of features.
> Documentation and examples will improve in the near feature.

With a background in Bevy, you'll find Byke straightforward.
The `App` type is the main entry point - just add plugins, resources, and systems.

```golang
func main() {
   var app App

   app.AddPlugin(GamePlugin)
   app.AddSystems(Startup, spawnCamera, spawnWorld)
   app.AddSystems(Update, Systems(moveObjectsSystem, otherSystem).Chain())
   app.MustRun()
}
```

Components are defined by embedding the zero-sized `Component[T]` type.
System parameters, such as resources, `Local` or `Query`, are automatically injected.
Use `Query[T]` for data retrieval. Byke offers standard query filters such as `With`, `Without`, `Changed`, and more.

```golang
type Velocity struct {
   Component[Velocity]
   Linear Vec
}

func moveObjectsSystem(vt VirtualTime, query Query[struct {
   Velocity  Velocity
   Transform *Transform
}]) {
   for item := range query.Items() {
      item.Transform.Translation = item.Transform.Translation.
         Add(item.Velocity.Linear.Mul(vt.DeltaSecs))
   }
}
```

### Core Features

* **Schedules and SystemSets**: Organize systems and define execution order.
   * `Local[T]` local state for systems
   * `In[T]` to pass a value when invoking a system
* **Resources**: Inject shared data into systems.
* **Queries**: Supports filters like `With`, `Without`, `Added`, and `Changed`. Also supports `Option[T]` and
  `OptionMut[T]`. Automatic mapping to struct types.
* **Events**: Use `EventWriter[E]` and `EventReader[E]` to send and receive events.
* **Observers**: Support bevy style (Entity-)Observers
* **States**: Manage application state with `State[S]` and `NextState[S]`.

    * Supports `OnEnter(state)` and `OnExit(state)` schedules.
    * Emits `StateTransitionEvent[S]` during transitions.
    * Allows state-scoped entities via `DespawnOnExitState(TitleScreen)`.
* **Commands**: Spawn/despawn entities, trigger observers and add/remove components.
* **Change detection**: Components marked as `Comparable` support automatic change detection.
* **Type Safety**: Avoids the need for type assertions like `value.(MyType)`.
* **Entity Hierarchies**: Support for parent-child relationships between entities.
* **Fixed Timestep**: Execute game logic or physics systems with a fixed timestep interval.


### Ebitengine Integration

The `bykebiten` package provides integration with [Ebitengine](https://ebitengine.org/):
* Initializes and manages the game loop
* Configures window settings, screen size, and input
* Applies transforms through entity hierarchy
* Manages rendering with z-order, anchors, and sizes
* Sprites, also supports sprite sheets
* Text rendering with custom fonts
* Handles vectors with filled and stroked paths
* Supports meshes: circles, rectangles, and triangulated polygons
* Shaders for sprites and meshes with uniforms and image inputs
* Multi-camera functionality
* Custom render targets via *ebiten.Image
* Asset loading:
  * AssetLoader support
  * Tracks asset loads
  * Custom fs.FS: embedded, http, local support
* Audio playback with looping and despawning
* Spatial audio (see `astroids` example)

### Example

Check out the [examples](https://github.com/oliverbestmann/byke/blob/main/bykebiten/examples/) for a
look at how everything comes together.

---

> Note: This README and some documentation was refined using generative AI,
> but all code in this project is handwritten.
