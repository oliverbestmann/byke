package byke

import (
	"fmt"
	"reflect"
)

// In describes an input parameter of a system.
// A system can only accept exactly one input parameter.
type In[T any] struct {
	Value T
}

func (i In[T]) init(*World) SystemParamState {
	wrapper := reflect.ValueOf(&In[T]{}).Elem()
	return &inSystemParamState[T]{
		wrapperValue: wrapper,
		inValue:      wrapper.Field(0),
	}
}

type inSystemParamState[T any] struct {
	wrapperValue, inValue reflect.Value
}

func (i *inSystemParamState[T]) getValue(sc systemContext) (reflect.Value, error) {
	actualValue := reflect.ValueOf(sc.InValue)

	if !actualValue.Type().AssignableTo(i.inValue.Type()) {
		err := fmt.Errorf("can not use param type %s with In[%s]", actualValue.Type(), i.inValue.Type())
		return reflect.Value{}, err
	}

	i.inValue.Set(actualValue)
	return i.wrapperValue, nil
}

func (i *inSystemParamState[T]) cleanupValue() {
	// clear the reference
	i.inValue.SetZero()
}

func (i *inSystemParamState[T]) valueType() reflect.Type {
	return i.wrapperValue.Type()
}
