package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"reflect"
)

type ParsedQuery struct {
	Query   arch.Query
	Mutable []*arch.ComponentType
}

func ParseQuery(queryType reflect.Type) (ParsedQuery, error) {
	var parsed ParsedQuery

	if err := buildQuery(queryType, &parsed); err != nil {
		return ParsedQuery{}, err
	}

	return parsed, nil
}

func buildQuery(queryType reflect.Type, result *ParsedQuery) error {
	query := &result.Query

	switch {
	case isComponent(queryType):
		query.Fetch = append(query.Fetch, componentTypeOf(queryType))
		return nil

	case isMutableComponent(queryType):
		componentType := componentTypeOf(queryType.Elem())
		query.Fetch = append(query.Fetch, componentType)
		result.Mutable = append(result.Mutable, componentType)
		return nil

	case isFilter(queryType):
		filter := reflect.New(queryType).Interface().(Filter)
		filter.applyTo(result)
		return nil

	case isStructQuery(queryType):
		return buildStructQuery(queryType, result)

	default:
		return fmt.Errorf("invalid query type: %s", queryType)
	}
}

func buildStructQuery(queryType reflect.Type, result *ParsedQuery) error {
	for field := range fieldsOf(queryType) {
		if err := buildQuery(field.Type, result); err != nil {
			return err
		}
	}

	return nil
}

func isStructQuery(ty reflect.Type) bool {
	return ty.Kind() == reflect.Struct
}

func isComponent(ty reflect.Type) bool {
	if ty.Kind() != reflect.Struct {
		return false
	}

	if !ty.Implements(reflect.TypeFor[arch.ErasedComponent]()) {
		return false
	}

	// a component must embed arch.Component or arch.ComparableComponent
	var count int
	for field := range fieldsOf(ty) {
		if implementsInterfaceDirectly[arch.ErasedComponent](field.Type) {
			count += 1
		}
	}

	// expect to have exactly one
	return count == 1
}

func isMutableComponent(ty reflect.Type) bool {
	return ty.Kind() == reflect.Pointer && isComponent(ty.Elem())
}

func isFilter(ty reflect.Type) bool {
	return ty.Kind() != reflect.Pointer && implementsInterfaceDirectly[Filter](ty)
}
