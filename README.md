# byke

**byke** is an Entity Component System (ECS) library for Go, inspired by the [Bevy](https://bevy.org/) API.

Although still under development, it already includes a wide range of features.
Documentation and examples will improve in the near feature.

### Core Features

* **Schedules and SystemSets**: Organize systems and define execution order.
* **Resources**: Inject shared data into systems.
* **Queries**: Supports filters like `With`, `Without`, `Added`, and `Changed`. Also supports `Option[T]` and
  `OptionMut[T]`.
* **Events**: Use `EventWriter[E]` and `EventReader[E]` to send and receive events.
* **States**: Manage application state with `State[S]` and `NextState[S]`.

    * Supports `OnEnter(state)` and `OnExit(state)` schedules.
    * Emits `StateTransitionEvent[S]` during transitions.
    * Allows state-scoped entities via `DespawnOnExitState(TitleScreen)`.
* **Commands**: Spawn/despawn entities and add/remove components.
* **Type Safety**: Avoids the need for type assertions like `value.(MyType)`.
* **Entity Hierarchies**: Support for parent-child relationships between entities.

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
