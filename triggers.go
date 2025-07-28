package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/refl"
	"reflect"
)

type On[E any] struct {
	Target EntityId
	Event  E
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

func (On[E]) isTrigger(isTriggerComponent) {}

// new creates a new value of this type and returns it
func (On[E]) new(target EntityId, event any) isTriggerComponent {
	return On[E]{
		Target: target,
		Event:  event.(E),
	}
}

type onSystemParamState struct {
	onType    reflect.Type
	makeValue func(target EntityId, event any) isTriggerComponent
}

func (o onSystemParamState) getValue(sc systemContext) (reflect.Value, error) {
	return reflect.ValueOf(o.makeValue(sc.Trigger.TargetId, sc.Trigger.EventValue)), nil
}

func (o onSystemParamState) cleanupValue() {}

func (o onSystemParamState) valueType() reflect.Type {
	return o.onType
}

type isTriggerComponent interface {
	isTrigger(isTriggerComponent)
	new(target EntityId, event any) isTriggerComponent
	eventType() reflect.Type
}

var _ isTriggerComponent = On[bool]{}

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
		panic("Observers first parameter must be of type On[Event]")
	}

	triggerType := funcType.In(0)
	if triggerType.Kind() != reflect.Struct || !refl.ImplementsInterfaceDirectly[isTriggerComponent](triggerType) {
		panic(fmt.Sprintf("Observers first parameter must be of type On[Event], got %s", triggerType))
	}

	triggerValue := reflect.New(triggerType).Elem().Interface().(isTriggerComponent)

	return Observer{
		eventType: triggerValue.eventType(),
		callback:  fn,
	}
}

func (o Observer) WatchEntity(entityId EntityId) Observer {
	o.entities = append(o.entities, entityId)
	return o
}
