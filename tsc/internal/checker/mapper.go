package checker

import (
	"slices"
	"unsafe"
)

// TypeMapper

type TypeMapper struct {
	kind TypeMapperKind `customlint:"protected"`
	id   TypeMapperId   // Assigned when a record first stores the mapper
}

// TypeMapperId names a mapper stored in a checker record: its index in the checker's mapper table plus one, 0 for
// none.
type TypeMapperId uint32

type TypeMapperKind uint8

const (
	TypeMapperKindUnknown TypeMapperKind = iota
	TypeMapperKindSimple
	TypeMapperKindArray
	TypeMapperKindArrayToSingle
	TypeMapperKindDeferred
	TypeMapperKindFunction
	TypeMapperKindMerged
	TypeMapperKindComposite
	TypeMapperKindInference
)

// Each concrete mapper embeds the immutable kind header at offset zero.
func castTypeMapper[T any](m *TypeMapper, kind TypeMapperKind) *T {
	if m.kind != kind {
		panic("Incorrect type mapper kind")
	}
	// SAFETY: Each kind is only assigned by the constructor of its concrete mapper, which
	// embeds TypeMapper at offset zero, and every caller pairs T with its kind.
	return (*T)(unsafe.Pointer(m))
}

var (
	_ [0 - unsafe.Offsetof(SimpleTypeMapper{}.TypeMapperBase)]byte
	_ [0 - unsafe.Offsetof(ArrayTypeMapper{}.TypeMapperBase)]byte
	_ [0 - unsafe.Offsetof(ArrayToSingleTypeMapper{}.TypeMapperBase)]byte
	_ [0 - unsafe.Offsetof(DeferredTypeMapper{}.TypeMapperBase)]byte
	_ [0 - unsafe.Offsetof(FunctionTypeMapper{}.TypeMapperBase)]byte
	_ [0 - unsafe.Offsetof(MergedTypeMapper{}.TypeMapperBase)]byte
	_ [0 - unsafe.Offsetof(CompositeTypeMapper{}.TypeMapperBase)]byte
	_ [0 - unsafe.Offsetof(InferenceTypeMapper{}.TypeMapperBase)]byte
	_ [0 - unsafe.Offsetof(TypeMapperBase{}.TypeMapper)]byte
)

func (m *TypeMapper) Map(c *Checker, t *Type) *Type {
	switch m.kind {
	case TypeMapperKindSimple:
		return castTypeMapper[SimpleTypeMapper](m, TypeMapperKindSimple).Map(c, t)
	case TypeMapperKindArray:
		return castTypeMapper[ArrayTypeMapper](m, TypeMapperKindArray).Map(c, t)
	case TypeMapperKindArrayToSingle:
		return castTypeMapper[ArrayToSingleTypeMapper](m, TypeMapperKindArrayToSingle).Map(c, t)
	case TypeMapperKindDeferred:
		return castTypeMapper[DeferredTypeMapper](m, TypeMapperKindDeferred).Map(c, t)
	case TypeMapperKindFunction:
		return castTypeMapper[FunctionTypeMapper](m, TypeMapperKindFunction).Map(c, t)
	case TypeMapperKindMerged:
		return castTypeMapper[MergedTypeMapper](m, TypeMapperKindMerged).Map(c, t)
	case TypeMapperKindComposite:
		return castTypeMapper[CompositeTypeMapper](m, TypeMapperKindComposite).Map(c, t)
	case TypeMapperKindInference:
		return castTypeMapper[InferenceTypeMapper](m, TypeMapperKindInference).Map(c, t)
	default:
		panic("Invalid type mapper kind")
	}
}

func (m *TypeMapper) Kind() TypeMapperKind { return m.kind }

func (m *TypeMapper) MapsThisOnly(c *Checker) bool {
	switch m.kind {
	case TypeMapperKindSimple:
		return castTypeMapper[SimpleTypeMapper](m, TypeMapperKindSimple).MapsThisOnly(c)
	case TypeMapperKindArray:
		return castTypeMapper[ArrayTypeMapper](m, TypeMapperKindArray).MapsThisOnly(c)
	case TypeMapperKindArrayToSingle:
		return castTypeMapper[ArrayToSingleTypeMapper](m, TypeMapperKindArrayToSingle).MapsThisOnly(c)
	case TypeMapperKindDeferred:
		return castTypeMapper[DeferredTypeMapper](m, TypeMapperKindDeferred).MapsThisOnly(c)
	default:
		return false
	}
}

// Factory functions

func (c *Checker) getMappedType(t *Type, mapper *TypeMapper) *Type {
	return mapper.Map(c, getNonDistributedTypeParameter(c, t))
}

func (c *Checker) newTypeMapper(sources []TypeId, targets []TypeId) *TypeMapper {
	if len(sources) == 1 {
		return newSimpleTypeMapper(c.typ(sources[0]), c.typ(targets[0]))
	}
	return c.newArrayTypeMapper(sources, targets)
}

func (c *Checker) combineTypeMappers(m1 *TypeMapper, m2 *TypeMapper) *TypeMapper {
	if m1 != nil {
		return newCompositeTypeMapper(c, m1, m2)
	}
	return m2
}

func (c *Checker) mapTypeWithCompositeMapper(t *Type, m1 *TypeMapper, m2 *TypeMapper) *Type {
	if m1 == nil {
		return c.getMappedType(t, m2)
	}
	t1 := c.getMappedType(t, m1)
	if t1 != t {
		return c.instantiateType(t1, m2)
	}
	return c.getMappedType(t, m2)
}

func mergeTypeMappers(m1 *TypeMapper, m2 *TypeMapper) *TypeMapper {
	if m1 != nil {
		return newMergedTypeMapper(m1, m2)
	}
	return m2
}

func prependTypeMapping(c *Checker, source *Type, target *Type, mapper *TypeMapper) *TypeMapper {
	if mapper == nil {
		return newSimpleTypeMapper(getNonDistributedTypeParameter(c, source), target)
	}
	return newMergedTypeMapper(newSimpleTypeMapper(getNonDistributedTypeParameter(c, source), target), mapper)
}

func appendTypeMapping(c *Checker, mapper *TypeMapper, source *Type, target *Type) *TypeMapper {
	if mapper == nil {
		return newSimpleTypeMapper(getNonDistributedTypeParameter(c, source), target)
	}
	return newMergedTypeMapper(mapper, newSimpleTypeMapper(getNonDistributedTypeParameter(c, source), target))
}

// Maps forward-references to later types parameters to the empty object type.
// This is used during inference when instantiating type parameter defaults.
func (c *Checker) newBackreferenceMapper(context *InferenceContext, index int) *TypeMapper {
	forwardInferences := context.inferences[index:]
	typeParameters := mapToTypeIds(forwardInferences, func(i *InferenceInfo) *Type {
		return i.typeParameter
	})
	return c.newArrayToSingleTypeMapper(typeParameters, c.unknownType)
}

// TypeMapperBase

type TypeMapperBase struct {
	TypeMapper
}

func (m *TypeMapperBase) Map(c *Checker, t *Type) *Type { return t }
func (m *TypeMapperBase) MapsThisOnly(c *Checker) bool  { return false }

// SimpleTypeMapper

type SimpleTypeMapper struct {
	TypeMapperBase
	source TypeId
	target TypeId
}

func newSimpleTypeMapper(source *Type, target *Type) *TypeMapper {
	m := &SimpleTypeMapper{}
	m.kind = TypeMapperKindSimple //nolint:customlint
	m.source = source.ref()
	m.target = target.ref()
	return &m.TypeMapper
}

func (m *SimpleTypeMapper) Map(c *Checker, t *Type) *Type {
	if t == c.typ(m.source) {
		return c.typ(m.target)
	}
	return t
}

func (m *SimpleTypeMapper) MapsThisOnly(c *Checker) bool {
	return isThisTypeParameter(c.typ(m.source))
}

// ArrayTypeMapper

// The lists of an array mapper are handles into the checker's list store, so the mapper holds no pointers and the
// collector doesn't scan it. Most lists, such as a declaration's type parameters and a reference's type arguments,
// already live there and keep their handle; a list built elsewhere is copied in.
type ArrayTypeMapper struct {
	TypeMapperBase
	sources uint32
	targets uint32
	n       uint32
}

// mapperListOf returns the position of list in the checker's list store, copying it in if it isn't stored there.
func (c *Checker) mapperListOf(list []TypeId) uint32 {
	if pos, ok := c.typeLists.find(list); ok {
		return pos
	}
	pos, _ := c.typeListOf(list)
	return pos
}

func (c *Checker) newArrayTypeMapper(sources []TypeId, targets []TypeId) *TypeMapper {
	m := &ArrayTypeMapper{}
	m.kind = TypeMapperKindArray //nolint:customlint
	m.sources = c.mapperListOf(sources)
	m.targets = c.mapperListOf(targets)
	m.n = uint32(len(sources))
	return &m.TypeMapper
}

func (m *ArrayTypeMapper) Sources(c *Checker) []TypeId { return c.typeLists.slice(m.sources, m.n) }
func (m *ArrayTypeMapper) Targets(c *Checker) []TypeId { return c.typeLists.slice(m.targets, m.n) }

func (m *ArrayTypeMapper) Map(c *Checker, t *Type) *Type {
	id := t.ref()
	for i, s := range m.Sources(c) {
		if s == id {
			return c.typ(m.Targets(c)[i])
		}
	}
	return t
}

func (m *ArrayTypeMapper) MapsThisOnly(c *Checker) bool {
	return m.n == 1 && isThisTypeParameter(c.typ(m.Sources(c)[0]))
}

// ArrayToSingleTypeMapper

type ArrayToSingleTypeMapper struct {
	TypeMapperBase
	sources uint32
	n       uint32
	target  TypeId
}

func (c *Checker) newArrayToSingleTypeMapper(sources []TypeId, target *Type) *TypeMapper {
	m := &ArrayToSingleTypeMapper{}
	m.kind = TypeMapperKindArrayToSingle //nolint:customlint
	m.sources = c.mapperListOf(sources)
	m.n = uint32(len(sources))
	m.target = target.ref()
	return &m.TypeMapper
}

func (m *ArrayToSingleTypeMapper) Sources(c *Checker) []TypeId { return c.typeLists.slice(m.sources, m.n) }

func (m *ArrayToSingleTypeMapper) Map(c *Checker, t *Type) *Type {
	if slices.Contains(m.Sources(c), t.ref()) {
		return c.typ(m.target)
	}
	return t
}

func (m *ArrayToSingleTypeMapper) MapsThisOnly(c *Checker) bool {
	return m.n == 1 && isThisTypeParameter(c.typ(m.Sources(c)[0]))
}

// DeferredTypeMapper

type DeferredTypeMapper struct {
	TypeMapperBase
	sources []TypeId
	targets []func() *Type
}

func newDeferredTypeMapper(sources []TypeId, targets []func() *Type) *TypeMapper {
	m := &DeferredTypeMapper{}
	m.kind = TypeMapperKindDeferred //nolint:customlint
	m.sources = sources
	m.targets = targets
	return &m.TypeMapper
}

func (m *DeferredTypeMapper) Map(c *Checker, t *Type) *Type {
	for i, sId := range m.sources {
		s := c.typ(sId)
		if t == s {
			return m.targets[i]()
		}
	}
	return t
}

func (m *DeferredTypeMapper) MapsThisOnly(c *Checker) bool {
	return len(m.sources) == 1 && isThisTypeParameter(c.typ(m.sources[0]))
}

// FunctionTypeMapper

type FunctionTypeMapper struct {
	TypeMapperBase
	fn func(*Type) *Type
}

func newFunctionTypeMapper(fn func(*Type) *Type) *TypeMapper {
	m := &FunctionTypeMapper{}
	m.kind = TypeMapperKindFunction //nolint:customlint
	m.fn = fn
	return &m.TypeMapper
}

func (m *FunctionTypeMapper) Map(c *Checker, t *Type) *Type {
	return m.fn(t)
}

// MergedTypeMapper

type MergedTypeMapper struct {
	TypeMapperBase
	m1 *TypeMapper
	m2 *TypeMapper
}

func newMergedTypeMapper(m1 *TypeMapper, m2 *TypeMapper) *TypeMapper {
	m := &MergedTypeMapper{}
	m.kind = TypeMapperKindMerged //nolint:customlint
	m.m1 = m1
	m.m2 = m2
	return &m.TypeMapper
}

func (m *MergedTypeMapper) Map(c *Checker, t *Type) *Type {
	return m.m2.Map(c, m.m1.Map(c, t))
}

// CompositeTypeMapper

type CompositeTypeMapper struct {
	TypeMapperBase
	c  *Checker
	m1 *TypeMapper
	m2 *TypeMapper
}

func newCompositeTypeMapper(c *Checker, m1 *TypeMapper, m2 *TypeMapper) *TypeMapper {
	m := &CompositeTypeMapper{}
	m.kind = TypeMapperKindComposite //nolint:customlint
	m.c = c
	m.m1 = m1
	m.m2 = m2
	return &m.TypeMapper
}

func (m *CompositeTypeMapper) Map(c *Checker, t *Type) *Type {
	t1 := m.m1.Map(c, t)
	if t1 != t {
		return m.c.instantiateType(t1, m.m2)
	}
	return m.m2.Map(c, t)
}

// InferenceTypeMapper

type InferenceTypeMapper struct {
	TypeMapperBase
	c      *Checker
	n      *InferenceContext
	fixing bool
}

func (c *Checker) initInferenceTypeMapper(m *InferenceTypeMapper, n *InferenceContext, fixing bool) *TypeMapper {
	m.kind = TypeMapperKindInference //nolint:customlint
	m.c = c
	m.n = n
	m.fixing = fixing
	return &m.TypeMapper
}

func (m *InferenceTypeMapper) Map(c *Checker, t *Type) *Type {
	for i, inference := range m.n.inferences {
		if t == inference.typeParameter {
			if m.fixing && !inference.isFixed {
				// Before we commit to a particular inference (and thus lock out any further inferences),
				// we infer from any intra-expression inference sites we have collected.
				m.c.inferFromIntraExpressionSites(m.n)
				clearCachedInferences(m.n.inferences)
				inference.isFixed = true
			}
			return m.c.getInferredType(m.n, i)
		}
	}
	return t
}
