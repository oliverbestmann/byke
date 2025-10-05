package byke

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/oliverbestmann/byke/internal/refl"
)

type On[E Event] struct {
	Event E
}

func (On[E]) init(*World) SystemParamState {
	return onSystemParamState{
		onType:    reflect.TypeFor[On[E]](),
		makeValue: On[E]{}.new,
	}
}

func (On[E]) eventType() reflect.Type {
	return reflect.TypeFor[E]()
}

func (On[E]) isOn(isOn) {}

// new creates a new value of this type and returns it
func (On[E]) new(event Event) isOn {
	return On[E]{
		Event: event.(E),
	}
}

type onSystemParamState struct {
	onType    reflect.Type
	makeValue func(event Event) isOn
}

func (o onSystemParamState) getValue(sc systemContext) (reflect.Value, error) {
	return reflect.ValueOf(o.makeValue(sc.Trigger.EventValue)), nil
}

func (o onSystemParamState) cleanupValue() {}

func (o onSystemParamState) valueType() reflect.Type {
	return o.onType
}

type isOn interface {
	isOn(isOn)
	new(event Event) isOn
	eventType() reflect.Type
}

var _ isOn = On[bool]{}

type Observer struct {
	Component[Observer]
	eventType reflect.Type
	callback  AnySystem
	entities  []EntityId
	system    *preparedSystem
}

func NewObserver(fn any) Observer {
	value := reflect.ValueOf(fn)

	if value.Kind() != reflect.Func {
		panic("Observer must be a function")
	}

	funcType := value.Type()
	if funcType.NumIn() < 1 {
		panic("Observers first parameter must be of type On[Message]")
	}

	triggerType := funcType.In(0)
	if triggerType.Kind() != reflect.Struct || !refl.ImplementsInterfaceDirectly[isOn](triggerType) {
		panic(fmt.Sprintf("Observers first parameter must be of type On[Message], got %s", triggerType))
	}

	triggerValue := reflect.New(triggerType).Elem().Interface().(isOn)

	return Observer{
		eventType: triggerValue.eventType(),
		callback:  fn,
	}
}

func (o Observer) WatchEntity(entityId EntityId) Observer {
	o.entities = append(o.entities, entityId)
	return o
}

func (o Observer) ObservesType(ty reflect.Type) bool {
	return o.eventType == ty
}

func (o Observer) IsScoped() bool {
	return len(o.entities) > 0
}

func (o Observer) Observes(id EntityId) bool {
	return slices.Contains(o.entities, id)
}
