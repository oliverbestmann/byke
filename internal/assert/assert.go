package assert

import (
	"fmt"
	"reflect"
)

func IsPointerType(t reflect.Type) {
	if t.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("expected pointer type, got %s", t))
	}
}

func IsNonPointerType(t reflect.Type) {
	if t.Kind() == reflect.Pointer {
		panic(fmt.Sprintf("expected non pointer type, got %s", t))
	}
}
