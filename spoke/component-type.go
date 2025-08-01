package spoke

import (
	"fmt"
	"hash/maphash"
	"log/slog"
	"maps"
	"reflect"
	"sync/atomic"
	"unsafe"
)

var seed = maphash.MakeSeed()

type UnsafeSetValue func(target unsafe.Pointer, ref ErasedComponent)

type MakeColumn func() Column

type ComponentTypeId uint16

type ComponentType struct {
	Name       string
	Type       reflect.Type
	MakeColumn MakeColumn

	// The size of the underlying datatype in bytes
	Size uintptr

	// The Id of the type
	Id ComponentTypeId

	// HasPointers indicates that a value of the type contains pointers, e.g.
	// by having a field of type *T, a string, an interface, a slice or a map value.
	HasPointers bool

	// TriviallyHashable indicates that the type can be trivially hashed by hashing
	// the types MemorySlices
	TriviallyHashable bool

	// DirtyTracking indicates if the type is comparable
	DirtyTracking        bool
	IsMarker             bool
	IsImmutableComponent bool
}

func ComponentTypeOf[C IsComponent[C]]() *ComponentType {
	var zeroValue C

	//goland:noinspection GoDfaNilDereference
	return zeroValue.ComponentType()
}

func (c *ComponentType) New() ErasedComponent {
	return reflect.New(c.Type).Interface().(ErasedComponent)
}

func (c *ComponentType) CopyOf(value ErasedComponent) ErasedComponent {
	target := reflect.New(c.Type)
	target.Elem().Set(reflect.ValueOf(value).Elem())
	return target.Interface().(ErasedComponent)
}

func (c *ComponentType) String() string {
	return c.Name
}

func (c *ComponentType) LogValue() slog.Value {
	return slog.StringValue(c.Name)
}

var componentTypes atomic.Pointer[map[unsafe.Pointer]*ComponentType]

func init() {
	// initialize the lookup table
	componentTypes.Store(&map[unsafe.Pointer]*ComponentType{})
}

func ensureComponentType[C IsComponent[C]](ptrToType unsafe.Pointer, makeType func(id ComponentTypeId) *ComponentType) *ComponentType {
	for {
		previousTypes := componentTypes.Load()
		if cached, ok := (*previousTypes)[ptrToType]; ok {
			return cached
		}

		newTypeId := ComponentTypeId(len(*previousTypes) + 1)

		newType := makeType(newTypeId)

		newTypes := maps.Clone(*previousTypes)
		newTypes[ptrToType] = newType

		if componentTypes.CompareAndSwap(previousTypes, &newTypes) {
			slog.Info(
				"New component type registered",
				slog.String("name", newType.Name),
				slog.Int("id", int(newType.Id)),
			)

			// TODO move ValidateComponent into this package and call it here?

			return newType
		}
	}
}

func abiTypePointerTo(t reflect.Type) unsafe.Pointer {
	type eface struct {
		typ, val unsafe.Pointer
	}

	// a reflect.Type is backed by an *rType. The rType contains a abi.Type as
	// its first value. This means, that a *rType can be re-interpreted as *abi.Type
	return (*eface)(unsafe.Pointer(&t)).val
}

func nonComparableComponentTypeOf[C IsComponent[C]]() *ComponentType {
	reflectType := reflect.TypeFor[C]()
	ptrToType := abiTypePointerTo(reflectType)

	if cached, ok := (*componentTypes.Load())[ptrToType]; ok {
		return cached
	}

	if typeHasPaddingBytes(reflectType) {
		fmt.Printf("[warn] type %s contains padding bytes\n", reflectType)
	}

	return ensureComponentType[C](ptrToType, makeNonComparableComponentType[C])
}

func makeComparableComponentTypeOf[C IsComparableComponent[C]]() *ComponentType {
	reflectType := reflect.TypeFor[C]()
	ptrToType := abiTypePointerTo(reflectType)

	if cached, ok := (*componentTypes.Load())[ptrToType]; ok {
		return cached
	}

	if typeHasPaddingBytes(reflectType) {
		fmt.Printf("[warn] type %s contains padding bytes\n", reflectType)
	}

	return ensureComponentType[C](ptrToType, func(id ComponentTypeId) *ComponentType {
		ty := baseComponentTypeOf[C](id)

		switch {
		case ty.IsMarker:
			ty.MakeColumn = NewZeroSizedColumn[C]

		case ty.TriviallyHashable:
			ty.DirtyTracking = true
			ty.MakeColumn = NewShadowComparableColumn[C]

		default:
			ty.DirtyTracking = true
			ty.MakeColumn = NewHashedComparableColumn[C]
		}

		return ty
	})
}

func makeNonComparableComponentType[C IsComponent[C]](id ComponentTypeId) *ComponentType {
	ty := baseComponentTypeOf[C](id)

	switch {
	case ty.IsMarker:
		ty.MakeColumn = NewZeroSizedColumn[C]

	default:
		ty.MakeColumn = NewTypedColumn[C]
	}

	return ty
}

func baseComponentTypeOf[C IsComponent[C]](id ComponentTypeId) *ComponentType {
	reflectType := reflect.TypeFor[C]()

	var cValue C

	_, isImmutableComponent := any(cValue).(IsErasedImmutableComponent)

	return &ComponentType{
		Id:                   id,
		Type:                 reflectType,
		Name:                 reflectType.String(),
		Size:                 unsafe.Sizeof(cValue),
		IsMarker:             unsafe.Sizeof(cValue) == 0,
		HasPointers:          typeHasPointers(reflectType),
		TriviallyHashable:    typeIsTriviallyHashable(reflectType),
		IsImmutableComponent: isImmutableComponent,
	}
}

func unsafeCopyComponentValue[C IsComponent[C]](target unsafe.Pointer, value ErasedComponent) {
	// target points to a C
	*(*C)(target) = *any(value).(*C)
}

func unsafeCopyValue[C IsComponent[C]](to, from unsafe.Pointer) {
	*(*C)(to) = *(*C)(from)
}
