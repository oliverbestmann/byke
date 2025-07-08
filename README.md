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

The `bykebiten` package provides basic integration with [Ebitengine](https://ebitengine.org/):

* Starts and manages a game loop
* Provides resources for window config, screen size, mouse input, and more
* Handles sprite rendering with support for z-ordering, anchor points, and custom sizes
* Propagates transforms through the entity hierarchy

### Example

Check out the [examples](https://github.com/oliverbestmann/byke/blob/main/bykebiten/examples/manysprites/main.go) for a
look at how everything comes together.

---

> Note: This README and some documentation was refined using generative AI,
> but all code in this project is handwritten.
