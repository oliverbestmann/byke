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

type MakeColumn func() *ErasedColumn

type ComponentTypeId uint16

type ComponentType struct {
	Name            string
	Type            reflect.Type
	MakeColumn      MakeColumn
	UnsafeSetValue  UnsafeSetValue
	UnsafeCopyValue func(from, to unsafe.Pointer)

	// Maphash is an optional function to calculates the maphash
	// using maphash.Comparable. This is only defined if the type is comparable and
	// the component embeds ComparableComponent.
	Maphash func(ErasedComponent) HashValue

	// MemorySlices define regions that this type is well defined in. If the type has holes
	// due to having padding bytes, we might have multiple memory slices.
	MemorySlices []memorySlice

	// The Id of the type
	Id ComponentTypeId

	// HasPointers indicates that a value of the type contains pointers, e.g.
	// by having a field of type *T, a string, a slice or a map value.
	HasPointers bool

	// TriviallyHashable indicates that the type can be trivially hashed by hashing
	// the types MemorySlices
	TriviallyHashable bool

	// Comparable indicates if the type is comparable
	Comparable bool

	memcmp bool
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
			slog.Debug(
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

	return ensureComponentType[C](ptrToType, makeComponentType[C])
}

func comparableComponentTypeOf[C IsComparableComponent[C]]() *ComponentType {
	reflectType := reflect.TypeFor[C]()
	ptrToType := abiTypePointerTo(reflectType)

	if cached, ok := (*componentTypes.Load())[ptrToType]; ok {
		return cached
	}

	if typeHasPaddingBytes(reflectType) {
		fmt.Printf("[warn] type %s contains padding bytes\n", reflectType)
	}

	return ensureComponentType[C](ptrToType, func(id ComponentTypeId) *ComponentType {
		ty := makeComponentType[C](id)

		ty.MakeColumn = MakeErasedColumn(ty)
		ty.Comparable = true

		ty.memcmp = !typeHasPaddingBytes(reflectType) && len(ty.MemorySlices) == 1

		ty.Maphash = func(component ErasedComponent) HashValue {
			return HashValue(maphash.Comparable(seed, *any(component).(*C)))
		}

		return ty
	})
}

func makeComponentType[C IsComponent[C]](id ComponentTypeId) *ComponentType {
	reflectType := reflect.TypeFor[C]()

	ty := &ComponentType{
		Id:   id,
		Type: reflectType,
		Name: reflectType.String(),
	}

	ty.MakeColumn = MakeErasedColumn(ty)

	ty.UnsafeSetValue = unsafeCopyComponentValue[C]
	ty.UnsafeCopyValue = unsafeCopyValue[C]

	ty.HasPointers = typeHasPointers(reflectType)

	ty.TriviallyHashable = typeIsTriviallyHashable(reflectType)
	if ty.TriviallyHashable {
		ty.MemorySlices = memorySlicesOf(reflectType, 0, nil)
	}

	return ty
}

func unsafeCopyComponentValue[C IsComponent[C]](target unsafe.Pointer, value ErasedComponent) {
	// target points to a C
	*(*C)(target) = *any(value).(*C)
}

func unsafeCopyValue[C IsComponent[C]](to, from unsafe.Pointer) {
	*(*C)(to) = *(*C)(from)
}
