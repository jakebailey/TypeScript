package checker

import (
	"encoding/binary"
	"math/rand/v2"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/testutil/race"
	"github.com/zeebo/xxh3"
	"gotest.tools/v3/assert"
)

func TestCheckerNodeLinkStores(t *testing.T) {
	t.Parallel()
	f := ast.NewNodeFactory(nil, ast.NodeFactoryHooks{})
	table := ast.NewFileTable(nil)
	first := &Checker{}
	first.initNodeLinks(ast.NewRefCache(table))
	second := &Checker{}
	second.initNodeLinks(ast.NewRefCache(table))
	nodes := make([]ast.Node, 4096)
	for i := range nodes {
		node := f.NewIdentifier(strconv.Itoa(i))
		if i%2 == 0 {
			ast.GetNodeId(node)
		}
		nodes[i] = node
		assert.Assert(t, first.resolvedTypes.get(node) == nil)
		assert.Equal(t, first.resolvedSignatures.get(node), SignatureId(0))
		first.resolvedTypes.set(node, &Type{id: TypeId(i + 1)})
		first.resolvedSignatures.set(node, SignatureId(i+1))
		second.resolvedTypes.set(node, &Type{id: TypeId(i + 2)})
		second.resolvedSignatures.set(node, SignatureId(i+2))
	}
	runtime.GC()
	for i, node := range nodes {
		assert.Equal(t, first.resolvedTypes.get(node).Id(), TypeId(i+1))
		assert.Equal(t, first.resolvedSignatures.get(node), SignatureId(i+1))
		assert.Equal(t, second.resolvedTypes.get(node).Id(), TypeId(i+2))
		assert.Equal(t, second.resolvedSignatures.get(node), SignatureId(i+2))
	}
	clone := nodes[0].Clone(f)
	assert.Assert(t, first.resolvedTypes.get(clone) == nil)
	assert.Equal(t, first.resolvedSignatures.get(clone), SignatureId(0))
}

func TestTypeCasts(t *testing.T) {
	t.Parallel()
	type bases interface {
		AsType() *Type
		AsConstrainedType() *ConstrainedType
		AsStructuredType() *StructuredType
		AsObjectType() *ObjectType
		AsTypeReference() *TypeReference
		AsInterfaceType() *InterfaceType
		AsUnionOrIntersectionType() *UnionOrIntersectionType
	}
	tests := []struct {
		kind typeKind
		data bases
		cast func(*Type) any
	}{
		{typeKindIntrinsic, &IntrinsicType{}, func(t *Type) any { return t.AsIntrinsicType() }},
		{typeKindLiteral, &LiteralType{}, func(t *Type) any { return t.AsLiteralType() }},
		{typeKindUniqueESSymbol, &UniqueESSymbolType{}, func(t *Type) any { return t.AsUniqueESSymbolType() }},
		{typeKindTypeParameter, &TypeParameter{}, func(t *Type) any { return t.AsTypeParameter() }},
		{typeKindIndex, &IndexType{}, func(t *Type) any { return t.AsIndexType() }},
		{typeKindIndexedAccess, &IndexedAccessType{}, func(t *Type) any { return t.AsIndexedAccessType() }},
		{typeKindTemplateLiteral, &TemplateLiteralType{}, func(t *Type) any { return t.AsTemplateLiteralType() }},
		{typeKindStringMapping, &StringMappingType{}, func(t *Type) any { return t.AsStringMappingType() }},
		{typeKindSubstitution, &SubstitutionType{}, func(t *Type) any { return t.AsSubstitutionType() }},
		{typeKindConditional, &ConditionalType{}, func(t *Type) any { return t.AsConditionalType() }},
		{typeKindObject, &ObjectType{}, func(t *Type) any { return t.AsObjectType() }},
		{typeKindTypeReference, &TypeReference{}, func(t *Type) any { return t.AsTypeReference() }},
		{typeKindInterface, &InterfaceType{}, func(t *Type) any { return t.AsInterfaceType() }},
		{typeKindTuple, &TupleType{}, func(t *Type) any { return t.AsTupleType() }},
		{typeKindInstantiationExpression, &InstantiationExpressionType{}, func(t *Type) any { return t.AsInstantiationExpressionType() }},
		{typeKindMapped, &MappedType{}, func(t *Type) any { return t.AsMappedType() }},
		{typeKindReverseMapped, &ReverseMappedType{}, func(t *Type) any { return t.AsReverseMappedType() }},
		{typeKindEvolvingArray, &EvolvingArrayType{}, func(t *Type) any { return t.AsEvolvingArrayType() }},
		{typeKindUnion, &UnionType{}, func(t *Type) any { return t.AsUnionType() }},
		{typeKindIntersection, &IntersectionType{}, func(t *Type) any { return t.AsIntersectionType() }},
	}
	for _, test := range tests {
		c := &Checker{}
		test.data.AsType().id = TypeId(test.kind)<<typeKindShift | 1
		typ := c.newType(TypeFlagsAny, ObjectFlagsNone, test.data.AsType(), test.kind)
		runtime.GC()
		assert.Equal(t, test.cast(typ), any(test.data))
		assert.Equal(t, typ.AsConstrainedType(), test.data.AsConstrainedType())
		assert.Equal(t, typ.AsStructuredType(), test.data.AsStructuredType())
		assert.Equal(t, typ.AsObjectType(), test.data.AsObjectType())
		assert.Equal(t, typ.AsTypeReference(), test.data.AsTypeReference())
		assert.Equal(t, typ.AsInterfaceType(), test.data.AsInterfaceType())
		assert.Equal(t, typ.AsUnionOrIntersectionType(), test.data.AsUnionOrIntersectionType())
		typ.flags = TypeFlagsNever
		typ.objectFlags = ObjectFlagsReference
		assert.Equal(t, test.cast(typ), any(test.data))
		assert.Assert(t, didPanic(func() { typ.AsLiteralType() }) == (test.kind != typeKindLiteral))
	}
}

func didPanic(f func()) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	f()
	return false
}

func TestTypeHeaderSize(t *testing.T) {
	t.Parallel()
	// Seven 32-bit header words and an eight-word payload, with no pointers.
	assert.Equal(t, unsafe.Sizeof(Type{}), uintptr(56))
}

func TestKeyBuilder(t *testing.T) {
	t.Parallel()
	for _, count := range []int{0, 1, 46, 47, 256, 4096} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			t.Parallel()
			types := make([]TypeId, count)
			var expected []byte
			prefix := strings.Repeat("x", count)
			expected = append(expected, prefix...)
			expected = append(expected, 17)
			expected = binary.LittleEndian.AppendUint32(expected, 123)
			expected = binary.LittleEndian.AppendUint64(expected, uint64(count))
			for i := range types {
				types[i] = (&Type{id: TypeId(i + 1)}).ref()
				expected = binary.LittleEndian.AppendUint32(expected, uint32(i+1))
			}
			expected = append(expected, "suffix"...)
			var builder keyBuilder
			builder.writeString(prefix)
			builder.writeByte(17)
			builder.writeUint32(123)
			builder.writeTypes(types)
			builder.writeString("suffix")
			assert.Equal(t, builder.hash(), CacheHashKey(xxh3.Hash128(expected)))
			assert.Equal(t, builder.hash(), CacheHashKey(xxh3.Hash128(expected)))
		})
	}
}

func TestTypeListKeyAllocations(t *testing.T) { //nolint:paralleltest
	if race.Enabled {
		t.Skip("race instrumentation changes hashing allocations")
	}
	for _, count := range []int{8, 46, 47, 256, 4096} {
		types := make([]TypeId, count)
		for i := range types {
			types[i] = (&Type{id: TypeId(i + 1)}).ref()
		}
		allocations := testing.AllocsPerRun(10, func() { getTypeListKey(types) })
		expected := float64(0)
		if count > 46 {
			expected = 1
		}
		assert.Equal(t, allocations, expected, "type list length %d", count)
	}
}

func BenchmarkTypeListKey(b *testing.B) {
	for _, count := range []int{8, 46, 47, 256, 4096} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			types := make([]TypeId, count)
			for i := range types {
				types[i] = (&Type{id: TypeId(i + 1)}).ref()
			}
			b.ReportAllocs()
			for b.Loop() {
				getTypeListKey(types)
			}
		})
	}
}

func newCompositeTestChecker() (*Checker, []TypeId) {
	c := &Checker{
		strictNullChecks:         true,
		unionTypes:               make(map[CacheHashKey]TypeId),
		unionOfUnionTypes:        make(map[uint64]TypeId),
		aliasedUnionOfUnionTypes: make(map[UnionOfUnionKey]TypeId),
		intersectionTypes:        make(map[CacheHashKey]TypeId),
		subtypeReductionCache:    make(map[CacheHashKey][]TypeId),
	}
	c.compareSymbols = c.compareSymbolsWorker
	c.initSymbols(nil)
	c.neverType = c.newIntrinsicType(TypeFlagsNever, "never")
	c.unknownType = c.newIntrinsicType(TypeFlagsUnknown, "unknown")
	types := []TypeId{
		c.newIntrinsicType(TypeFlagsString, "string").ref(),
		c.newIntrinsicType(TypeFlagsNumber, "number").ref(),
		c.newIntrinsicType(TypeFlagsESSymbol, "symbol").ref(),
	}
	return c, types
}

func TestCompositeTypeOwnsConstituents(t *testing.T) {
	t.Parallel()
	c, types := newCompositeTestChecker()
	borrowed := slices.Clone(types)
	union := c.getUnionTypeFromSortedList(borrowed, ObjectFlagsNone, nil, nil)
	borrowed[0] = c.neverType.ref()
	assert.Assert(t, slices.Equal(union.Types(c), types))
	objects := []TypeId{
		c.newObjectType(ObjectFlagsInterface, 0).ref(),
		c.newObjectType(ObjectFlagsInterface, 0).ref(),
		c.newObjectType(ObjectFlagsInterface, 0).ref(),
	}
	intersection := c.getIntersectionType(objects)
	objects[0] = c.neverType.ref()
	assert.Assert(t, c.typ(intersection.Types(c)[0]) != c.neverType)
	reduced := c.removeSubtypes(borrowed, false)
	expected := slices.Clone(reduced)
	borrowed[0] = types[0]
	assert.Assert(t, slices.Equal(reduced, expected))
}

func TestCompositeTypeCacheHitAllocations(t *testing.T) { //nolint:paralleltest
	if race.Enabled {
		t.Skip("race instrumentation changes hashing allocations")
	}
	c, types := newCompositeTestChecker()
	union := c.getUnionType(types)
	filtered := c.removeType(union, c.typ(types[1]))
	objects := []TypeId{
		c.newObjectType(ObjectFlagsInterface, 0).ref(),
		c.newObjectType(ObjectFlagsInterface, 0).ref(),
		c.newObjectType(ObjectFlagsInterface, 0).ref(),
	}
	intersection := c.getIntersectionType(objects)
	assert.Equal(t, testing.AllocsPerRun(10, func() { c.getUnionType(types) }), float64(0))
	assert.Equal(t, testing.AllocsPerRun(10, func() { c.getIntersectionType(objects) }), float64(0))
	assert.Equal(t, testing.AllocsPerRun(10, func() { c.mapType(union, func(t *Type) *Type { return t }) }), float64(0))
	assert.Equal(t, testing.AllocsPerRun(10, func() {
		c.mapType(union, func(t *Type) *Type {
			if t == c.typ(types[1]) {
				return nil
			}
			return t
		})
	}), float64(0))
	assert.Equal(t, testing.AllocsPerRun(10, func() { c.removeType(union, c.typ(types[1])) }), float64(0))
	assert.Equal(t, c.removeType(union, c.typ(types[1])), filtered)
	assert.Equal(t, c.getIntersectionType(objects), intersection)
}

func TestTemporaryTypeSetReuse(t *testing.T) {
	t.Parallel()
	c, primitives := newCompositeTestChecker()
	for _, count := range []int{3, 17, 80, 300, 5000, 3} {
		objects := make([]TypeId, count)
		for i := range objects {
			objects[i] = c.newObjectType(ObjectFlagsInterface, 0).ref()
		}
		intersection := c.getIntersectionType(objects)
		assert.Assert(t, slices.Equal(intersection.Types(c), objects))
		union := c.getUnionType(objects)
		assert.Assert(t, slices.Equal(union.Types(c), objects))
		mapped := c.mapType(union, func(typ *Type) *Type {
			c.getIntersectionType(objects[:2])
			if typ == c.typ(objects[1]) {
				return nil
			}
			return typ
		})
		expected := slices.Clone(objects)
		expected = slices.Delete(expected, 1, 2)
		assert.Assert(t, slices.Equal(mapped.Types(c), expected))
		assert.Equal(t, c.removeType(union, c.typ(objects[1])), mapped)
		assert.Assert(t, c.mapType(union, func(*Type) *Type { return nil }) == nil)
		assert.Equal(t, c.getIntersectionType(objects), intersection)
		for set := c.freeTypeSets; set != nil; set = set.next {
			assert.Equal(t, len(set.values), 0)
			assert.Equal(t, len(set.valuesByKey), 0)
			assert.Assert(t, cap(set.values) <= 4096)
			for _, typId := range set.values[:cap(set.values)] {
				typ := c.typ(typId)
				assert.Assert(t, typ == nil)
			}
			for _, typ := range set.inline {
				assert.Assert(t, typ == 0)
			}
		}
	}
	left := c.getUnionType(primitives[:2])
	right := c.getUnionType(primitives[1:])
	assert.Equal(t, c.getIntersectionType([]TypeId{left.ref(), right.ref()}), c.typ(primitives[1]))
}

func TestIntersectionReductionAllocations(t *testing.T) { //nolint:paralleltest
	if race.Enabled {
		t.Skip("race instrumentation changes hashing allocations")
	}
	c, _ := newCompositeTestChecker()
	members := ast.NewSymbolTable()
	for i := range 128 {
		name := ast.MakeSymbolName(strconv.Itoa(i))
		members.SetId(c.symbols, name, c.newSymbol(ast.SymbolFlagsProperty, name))
	}
	object := c.newAnonymousType(0, members, nil, nil, nil)
	intersection := c.newIntersectionType(ObjectFlagsNone, []TypeId{object.ref(), object.ref()})
	assert.Assert(t, !c.somePropertyReducesToNever(intersection))
	assert.Equal(t, testing.AllocsPerRun(10, func() {
		assert.Assert(t, !c.somePropertyReducesToNever(intersection))
	}), float64(0))
}

func TestTemporaryPropertyCounts(t *testing.T) {
	t.Parallel()
	c := &Checker{}
	names := func(p *temporaryPropertyCounts) []string {
		var result []string
		for _, e := range p.entries {
			result = append(result, e.name.String())
		}
		return result
	}
	outer := c.getTemporaryPropertyCounts()
	outer.Add(ast.MakeSymbolName("second"))
	outer.Add(ast.MakeSymbolName("first"))
	outer.Add(ast.MakeSymbolName("second"))
	assert.Assert(t, slices.Equal(names(outer), []string{"second", "first"}))
	inner := c.getTemporaryPropertyCounts()
	assert.Assert(t, inner != outer)
	inner.Add(ast.MakeSymbolName("first"))
	assert.Equal(t, outer.Count(ast.MakeSymbolName("first")), 1)
	assert.Equal(t, outer.Count(ast.MakeSymbolName("second")), 2)
	c.releaseTemporaryPropertyCounts(inner)
	assert.Equal(t, inner.Size(), 0)
	assert.Equal(t, inner.Count(ast.MakeSymbolName("first")), 0)
	reused := c.getTemporaryPropertyCounts()
	assert.Equal(t, reused, inner)
	reused.Add(ast.MakeSymbolName("second"))
	reused.Add(ast.MakeSymbolName("first"))
	assert.Assert(t, slices.Equal(names(reused), []string{"second", "first"}))
	assert.Equal(t, outer.Count(ast.MakeSymbolName("first")), 1)
	c.releaseTemporaryPropertyCounts(reused)
	c.releaseTemporaryPropertyCounts(outer)
	runtime.GC()
	reused = c.getTemporaryPropertyCounts()
	assert.Equal(t, reused.Size(), 0)
	assert.Equal(t, reused.Count(ast.MakeSymbolName("second")), 0)
	for i := range 4097 {
		reused.Add(ast.MakeSymbolName(strconv.Itoa(i)))
	}
	c.releaseTemporaryPropertyCounts(reused)
	for counts := c.freePropertyCounts; counts != nil; counts = counts.next {
		assert.Assert(t, counts != reused)
		assert.Equal(t, counts.Size(), 0)
	}
	assert.Assert(t, c.getTemporaryPropertyCounts() != reused)
	other := &Checker{}
	assert.Assert(t, other.getTemporaryPropertyCounts() != outer)
}

func TestIntersectionReductionReentrancy(t *testing.T) {
	t.Parallel()
	c, _ := newCompositeTestChecker()
	property := c.newSymbol(ast.SymbolFlagsProperty, ast.MakeSymbolName("value"))
	object := c.newAnonymousType(0, c.symbols.TableOfId(ast.MakeSymbolName("value"), property), nil, nil, nil)
	inner := c.newIntersectionType(ObjectFlagsNone, []TypeId{object.ref(), object.ref()})
	outer := c.newIntersectionType(ObjectFlagsNone, []TypeId{inner.ref(), object.ref()})
	assert.Equal(t, c.getReducedType(outer), outer)
	assert.Equal(t, c.getReducedType(inner), inner)
	assert.Equal(t, c.getPropertyOfUnionOrIntersectionType(outer, ast.MakeSymbolName("value"), true), property)
	assert.Assert(t, c.freePropertyCounts != nil)
	assert.Assert(t, c.freePropertyCounts.next != nil)
	workspace := c.freePropertyCounts
	conflicting := c.newSymbolEx(ast.SymbolFlagsProperty, ast.MakeSymbolName("value"), ast.CheckFlagsContainsPrivate)
	outer.AsUnionOrIntersectionType().propertyCacheWithoutFunctionPropertyAugment(c).SetId(c.symbols, ast.MakeSymbolName("value"), conflicting)
	assert.Assert(t, c.somePropertyReducesToNever(outer))
	assert.Equal(t, c.freePropertyCounts, workspace)
	for counts := c.freePropertyCounts; counts != nil; counts = counts.next {
		assert.Equal(t, counts.Size(), 0)
		assert.Equal(t, counts.Count(ast.MakeSymbolName("value")), 0)
	}
}

func BenchmarkCompositeTypeCacheHit(b *testing.B) {
	c, types := newCompositeTestChecker()
	union := c.getUnionType(types)
	objects := []TypeId{
		c.newObjectType(ObjectFlagsInterface, 0).ref(),
		c.newObjectType(ObjectFlagsInterface, 0).ref(),
		c.newObjectType(ObjectFlagsInterface, 0).ref(),
	}

	c.getIntersectionType(objects)
	c.removeType(union, c.typ(types[1]))
	b.ReportAllocs()
	for b.Loop() {
		c.getUnionType(types)
		c.getIntersectionType(objects)
		c.mapType(union, func(t *Type) *Type { return t })
		c.removeType(union, c.typ(types[1]))
	}
}

func TestTypeMapperDispatch(t *testing.T) {
	t.Parallel()
	c, types := newCompositeTestChecker()
	source, target, other := c.typ(types[0]), c.typ(types[1]), c.typ(types[2])
	simple := newSimpleTypeMapper(source, target)
	array := c.newArrayTypeMapper([]TypeId{source.ref(), other.ref()}, []TypeId{target.ref(), source.ref()})
	merged := newMergedTypeMapper(simple, array)
	for _, test := range []struct {
		mapper *TypeMapper
		kind   TypeMapperKind
		result *Type
	}{
		{simple, TypeMapperKindSimple, target},
		{array, TypeMapperKindArray, target},
		{merged, TypeMapperKindMerged, target},
		{c.newArrayToSingleTypeMapper([]TypeId{source.ref()}, target), TypeMapperKindArrayToSingle, target},
		{newDeferredTypeMapper([]TypeId{source.ref()}, []func() *Type{func() *Type { return target }}), TypeMapperKindDeferred, target},
		{newFunctionTypeMapper(func(typ *Type) *Type {
			if typ == source {
				return target
			}
			return typ
		}), TypeMapperKindFunction, target},
		{newCompositeTypeMapper(c, newSimpleTypeMapper(other, target), simple), TypeMapperKindComposite, target},
	} {
		assert.Equal(t, test.mapper.Kind(), test.kind)
		assert.Equal(t, test.mapper.Map(c, source), test.result)
		assert.Assert(t, !test.mapper.MapsThisOnly(c))
	}
	assert.Equal(t, c.compareTypeMappers(simple, newSimpleTypeMapper(source, target)), 0)
	assert.Equal(t, c.compareTypeMappers(array, c.newArrayTypeMapper([]TypeId{source.ref(), other.ref()}, []TypeId{target.ref(), source.ref()})), 0)
	assert.Equal(t, c.compareTypeMappers(merged, newMergedTypeMapper(simple, array)), 0)
	this := c.newTypeParameter(0)
	this.AsTypeParameter().setIsThisType(true)
	for _, mapper := range []*TypeMapper{
		newSimpleTypeMapper(this, target),
		c.newArrayTypeMapper([]TypeId{this.ref()}, []TypeId{target.ref()}),
		c.newArrayToSingleTypeMapper([]TypeId{this.ref()}, target),
		newDeferredTypeMapper([]TypeId{this.ref()}, []func() *Type{func() *Type { return target }}),
	} {
		assert.Assert(t, mapper.MapsThisOnly(c))
		assert.Equal(t, mapper.Map(c, this), target)
		assert.Equal(t, mapper.Map(c, other), other)
	}
}

func TestCompactTypeMapperLayout(t *testing.T) {
	t.Parallel()
	assert.Assert(t, unsafe.Sizeof(SimpleTypeMapper{}) <= 3*unsafe.Sizeof(uintptr(0)))
	assert.Assert(t, unsafe.Sizeof(MergedTypeMapper{}) <= 3*unsafe.Sizeof(uintptr(0)))
	assert.Equal(t, unsafe.Sizeof(ArrayTypeMapper{}), uintptr(20))
	assert.Equal(t, unsafe.Sizeof(ArrayToSingleTypeMapper{}), uintptr(20))
	c, types := newCompositeTestChecker()
	mapper := newSimpleTypeMapper(c.typ(types[0]), c.typ(types[1]))
	assert.Assert(t, c.typ(castTypeMapper[SimpleTypeMapper](mapper, TypeMapperKindSimple).source) == c.typ(types[0]))
	func() {
		defer func() { assert.Assert(t, recover() != nil) }()
		castTypeMapper[ArrayTypeMapper](mapper, TypeMapperKindArray)
	}()
	runtime.GC()
	assert.Equal(t, mapper.Map(c, c.typ(types[0])), c.typ(types[1]))
	runtime.KeepAlive(c)
}

func TestInferenceContextMappers(t *testing.T) { //nolint:paralleltest
	c, types := newCompositeTestChecker()
	c.errorType = c.typ(types[0])
	info := newInferenceInfo(c.typ(types[0]))
	info.inferredType = c.typ(types[1])
	info.candidates = []TypeId{types[1]}
	context := c.newInferenceContextWorker([]*InferenceInfo{info}, nil, InferenceFlagsNone, nil)
	assert.Equal(t, context.nonFixingMapper.Map(c, c.typ(types[0])), c.typ(types[1]))
	assert.Assert(t, !info.isFixed)
	clone := c.cloneInferenceContext(context, InferenceFlagsNone)
	clone.inferences[0].candidates[0] = types[2]
	assert.Equal(t, c.typ(info.candidates[0]), c.typ(types[1]))
	assert.Equal(t, context.mapper.Map(c, c.typ(types[0])), c.typ(types[0]))
	assert.Assert(t, info.isFixed)
	assert.Assert(t, !clone.inferences[0].isFixed)
	assert.Equal(t, context.mapper.Map(c, c.typ(types[2])), c.typ(types[2]))
	runtime.GC()
	assert.Equal(t, context.nonFixingMapper.Map(c, c.typ(types[0])), c.typ(types[0]))
	if !race.Enabled {
		assert.Equal(t, testing.AllocsPerRun(10, func() {
			c.newInferenceContextWorker([]*InferenceInfo{info}, nil, InferenceFlagsNone, nil)
		}), float64(2))
	}
}

func TestCompactValueSymbolLinksLayout(t *testing.T) {
	t.Parallel()
	assert.Equal(t, unsafe.Sizeof(ValueSymbolLinks{}), 4*unsafe.Sizeof(TypeId(0)))
	assert.Equal(t, unsafe.Sizeof(valueSymbolExtra{}), 3*unsafe.Sizeof(TypeId(0)))
	c, types := newCompositeTestChecker()
	var links ValueSymbolLinks
	assert.Assert(t, links.getWriteType(c) == nil)
	assert.Assert(t, links.getNameType(c) == nil)
	assert.Assert(t, links.getContainingType(c) == nil)
	links.setWriteType(c, nil)
	links.setNameType(c, nil)
	links.setContainingType(c, nil)
	assert.Assert(t, links.extra == 0)
	links.setWriteType(c, c.typ(types[0]))
	extra := links.extra
	links.setNameType(c, c.typ(types[1]))
	links.setContainingType(c, c.typ(types[2]))
	assert.Equal(t, links.extra, extra)
	for range 1024 {
		var other ValueSymbolLinks
		other.setNameType(c, c.typ(types[2]))
	}
	runtime.GC()
	assert.Equal(t, links.getWriteType(c), c.typ(types[0]))
	assert.Equal(t, links.getNameType(c), c.typ(types[1]))
	assert.Equal(t, links.getContainingType(c), c.typ(types[2]))
	links.setNameType(c, nil)
	assert.Assert(t, links.getNameType(c) == nil)
	assert.Equal(t, links.getWriteType(c), c.typ(types[0]))
	assert.Equal(t, links.getContainingType(c), c.typ(types[2]))
}

func TestUnionIntersectionPropertyCacheSharing(t *testing.T) {
	t.Parallel()
	for _, intersection := range []bool{false, true} {
		for _, augmentedFirst := range []bool{false, true} {
			t.Run(strconv.FormatBool(intersection)+"/"+strconv.FormatBool(augmentedFirst), func(t *testing.T) {
				t.Parallel()
				c, _ := newCompositeTestChecker()
				members := ast.NewSymbolTable()
				for i := range 300 {
					name := ast.MakeSymbolName(strconv.Itoa(i))
					members.SetId(c.symbols, name, c.newSymbol(ast.SymbolFlagsProperty, name))
				}
				members.SetId(c.symbols, ast.EmptySymbolName, c.newSymbol(ast.SymbolFlagsProperty, ast.EmptySymbolName))
				object := c.newAnonymousType(0, members, nil, nil, nil)
				var typ *Type
				if intersection {
					typ = c.newIntersectionType(ObjectFlagsNone, []TypeId{object.ref(), object.ref()})
				} else {
					typ = c.newUnionType(ObjectFlagsNone, []TypeId{object.ref(), object.ref()})
				}
				data := typ.AsUnionOrIntersectionType()
				for name, symbol := range members.AllIds(c.symbols) {
					assert.Equal(t, c.getUnionOrIntersectionProperty(typ, name, !augmentedFirst), symbol)
					assert.Equal(t, c.getUnionOrIntersectionProperty(typ, name, augmentedFirst), symbol)
				}
				assert.Equal(t, data.propertyCacheWithoutFunctionPropertyAugment(c).Len(), members.Len())
				if !augmentedFirst {
					assert.Assert(t, data.propertyCache(c).IsNil())
				}
				runtime.GC()
				for name, symbol := range members.AllIds(c.symbols) {
					assert.Equal(t, c.getUnionOrIntersectionProperty(typ, name, true), symbol)
					assert.Equal(t, c.getUnionOrIntersectionProperty(typ, name, false), symbol)
				}
				runtime.KeepAlive(c)
			})
		}
	}
}

func TestUnionIntersectionPropertyCacheModes(t *testing.T) {
	t.Parallel()
	for _, intersection := range []bool{false, true} {
		t.Run(strconv.FormatBool(intersection), func(t *testing.T) {
			t.Parallel()
			c, _ := newCompositeTestChecker()
			var typ *Type
			if intersection {
				typ = c.newIntersectionType(ObjectFlagsNone, nil)
			} else {
				typ = c.newUnionType(ObjectFlagsNone, nil)
			}
			data := typ.AsUnionOrIntersectionType()
			assert.Assert(t, c.getUnionOrIntersectionProperty(typ, ast.MakeSymbolName("missing"), true) == 0)
			assert.Assert(t, c.getUnionOrIntersectionProperty(typ, ast.MakeSymbolName("missing"), false) == 0)
			assert.Assert(t, data.propertyCache(c).IsNil())
			assert.Assert(t, data.propertyCacheWithoutFunctionPropertyAugment(c).IsNil())
			data.setPropertyCacheWithoutFunctionPropertyAugment(c, ast.NewSymbolTable())
			data.setPropertyCache(c, ast.NewSymbolTable())
			for _, flags := range []ast.CheckFlags{ast.CheckFlagsReadPartial, ast.CheckFlagsWritePartial} {
				name := ast.MakeSymbolName(strconv.Itoa(int(flags)))
				partial := c.newSymbolEx(ast.SymbolFlagsProperty, name, flags)
				complete := c.newSymbol(ast.SymbolFlagsProperty, name)
				data.propertyCacheWithoutFunctionPropertyAugment(c).SetId(c.symbols, name, partial)
				assert.Equal(t, c.getUnionOrIntersectionProperty(typ, name, true), partial)
				assert.Assert(t, c.getUnionOrIntersectionProperty(typ, name, false) == 0)
				assert.Equal(t, c.getPropertyOfUnionOrIntersectionType(typ, name, true) == 0, flags&ast.CheckFlagsReadPartial != 0)
				data.propertyCache(c).SetId(c.symbols, name, complete)
				assert.Equal(t, c.getUnionOrIntersectionProperty(typ, name, false), complete)
			}
			name := ast.MakeSymbolName("different")
			nonAugmented := c.newSymbol(ast.SymbolFlagsProperty, name)
			augmented := c.newSymbol(ast.SymbolFlagsProperty, name)
			data.propertyCacheWithoutFunctionPropertyAugment(c).SetId(c.symbols, name, nonAugmented)
			data.propertyCache(c).SetId(c.symbols, name, augmented)
			assert.Equal(t, c.getUnionOrIntersectionProperty(typ, name, true), nonAugmented)
			assert.Equal(t, c.getUnionOrIntersectionProperty(typ, name, false), augmented)
		})
	}
}

func TestStructuredTypeExtra(t *testing.T) {
	t.Parallel()
	c, types := newCompositeTestChecker()
	data := c.newAnonymousType(0, ast.SymbolTable{}, nil, nil, nil).AsStructuredType()
	assert.Assert(t, data.resolvedExtra(c) == nil)
	assert.Assert(t, data.CallSignatures(c) == nil)
	assert.Assert(t, data.ConstructSignatures(c) == nil)
	assert.Assert(t, data.getIndexInfos(c) == nil)
	c.setStructuredTypeSignatures(data, nil, nil, nil)
	assert.Assert(t, data.resolvedExtra(c) == nil)

	call := &Signature{}
	construct := &Signature{}
	info := &IndexInfo{keyType: c.typ(types[0]).ref()}
	c.setStructuredTypeSignatures(data, []*Signature{call}, []*Signature{construct}, []*IndexInfo{info})
	assert.Equal(t, data.CallSignatures(c)[0], call)
	assert.Equal(t, data.ConstructSignatures(c)[0], construct)
	assert.Equal(t, cap(data.CallSignatures(c)), 1)
	assert.Equal(t, cap(data.ConstructSignatures(c)), 1)
	assert.Equal(t, data.getIndexInfos(c)[0], info)
	extra := data.resolvedExtra(c)
	extra.objectTypeWithoutAbstractConstructSignatures = c.typ(types[2]).ref()
	for range 1024 {
		other := c.newAnonymousType(0, ast.SymbolTable{}, []*Signature{call}, nil, nil)
		assert.Assert(t, other.AsStructuredType().resolvedExtra(c) != extra)
	}
	runtime.GC()
	assert.Equal(t, data.CallSignatures(c)[0], call)
	assert.Equal(t, data.ConstructSignatures(c)[0], construct)
	assert.Equal(t, data.getIndexInfos(c)[0], info)
	assert.Equal(t, c.typ(data.resolvedExtra(c).objectTypeWithoutAbstractConstructSignatures), c.typ(types[2]))
	c.setStructuredTypeSignatures(data, nil, []*Signature{construct}, nil)
	assert.Equal(t, len(data.CallSignatures(c)), 0)
	assert.Equal(t, data.ConstructSignatures(c)[0], construct)
	assert.Assert(t, data.getIndexInfos(c) == nil)
	c.setStructuredTypeSignatures(data, nil, nil, nil)
	assert.Assert(t, data.CallSignatures(c) == nil)
	assert.Assert(t, data.ConstructSignatures(c) == nil)
	assert.Equal(t, data.resolvedExtra(c), extra)
	assert.Equal(t, c.typ(extra.objectTypeWithoutAbstractConstructSignatures), c.typ(types[2]))
	c.setStructuredTypeSignatures(data, nil, nil, []*IndexInfo{})
	assert.Assert(t, data.getIndexInfos(c) != nil)
	runtime.KeepAlive(c)
}

func TestCompactSignatureLayout(t *testing.T) {
	t.Parallel()
	assert.Equal(t, unsafe.Sizeof(Signature{}), uintptr(48))
	assert.Equal(t, unsafe.Sizeof(signatureExtra{}), uintptr(24))
}

func TestSignatureExtra(t *testing.T) {
	t.Parallel()
	c, types := newCompositeTestChecker()
	c.noTypePredicate = c.allocTypePredicate(TypePredicateKindThis, ast.MakeSymbolName(""), 0, 0)
	sig := c.newSignature(SignatureFlagsHasRestParameter, ast.Node{}, nil, 0, nil, c.typ(types[0]), nil, 1)
	assert.Equal(t, sig.extra, uint32(0))
	assert.Assert(t, sig.TypeParameters(c) == nil)
	assert.Assert(t, sig.ThisParameter(c) == 0)
	assert.Assert(t, sig.getComposite(c) == nil)
	assert.Assert(t, sig.getIsolatedSignatureType(c) == nil)
	assert.Assert(t, sig.getResolvedTypePredicate(c) == nil)
	assert.Equal(t, sig.resolvedMinArgumentCount, int32(-1))
	sig.setTypeParameters(c, nil)
	sig.setThisParameter(c, 0)
	sig.setResolvedTypePredicate(c, nil)
	assert.Equal(t, sig.extra, uint32(0))
	assert.Assert(t, c.getTypePredicateOfSignature(sig) == nil)
	assert.Equal(t, sig.getResolvedTypePredicate(c), c.noTypePredicate)
	assert.Equal(t, sig.extra, uint32(0))
	assert.Assert(t, sig.flags&SignatureFlagsHasRestParameter != 0)
	assert.Equal(t, sig.Flags(), SignatureFlagsHasRestParameter)
	sig.setResolvedTypePredicate(c, nil)
	assert.Assert(t, sig.getResolvedTypePredicate(c) == nil)
	sig.setTypeParameters(c, []TypeId{})
	assert.Assert(t, sig.TypeParameters(c) != nil)
	entry := sig.extra
	list := slices.Clone(types)
	sig.setTypeParameters(c, list)
	list[0] = types[1]
	assert.Equal(t, c.typ(sig.TypeParameters(c)[0]), c.typ(types[0]))
	parameter := c.symbolId(ast.NewSymbol())
	sig.setThisParameter(c, parameter)
	predicate := c.newTypePredicate(TypePredicateKindIdentifier, ast.MakeSymbolName("p"), 0, c.typ(types[0]))
	sig.setResolvedTypePredicate(c, predicate)
	assert.Equal(t, c.getTypePredicateOfSignature(sig), predicate)
	sig.setComposite(c, true, []*Signature{sig})
	composite := sig.getComposite(c)
	assert.Assert(t, composite.isUnion)
	assert.Equal(t, composite.signatures(c)[0], sig)
	sig.ensureExtra(c).isolatedSignatureType = c.typ(types[2]).ref()
	clone := c.cloneSignature(sig)
	assert.Assert(t, clone.extra != entry)
	assert.Equal(t, clone.getComposite(c), composite)
	assert.Equal(t, clone.ThisParameter(c), parameter)
	assert.Equal(t, c.typ(clone.TypeParameters(c)[0]), c.typ(types[0]))
	assert.Assert(t, clone.getResolvedTypePredicate(c) == nil)
	assert.Assert(t, clone.getIsolatedSignatureType(c) == nil)
	clone.setThisParameter(c, 0)
	assert.Equal(t, sig.ThisParameter(c), parameter)
	sig.setResolvedTypePredicate(c, c.noTypePredicate)
	assert.Assert(t, c.getTypePredicateOfSignature(sig) == nil)
	assert.Equal(t, sig.ext(c).resolvedTypePredicate, uint32(0))
	sig.setResolvedTypePredicate(c, predicate)
	for range 1024 {
		other := c.newSignature(SignatureFlagsNone, ast.Node{}, types, parameter, nil, nil, predicate, 0)
		other.shareComposite(c, sig)
	}
	runtime.GC()
	assert.Equal(t, sig.extra, entry)
	assert.Equal(t, sig.getResolvedTypePredicate(c), predicate)
	assert.Equal(t, sig.Flags(), SignatureFlagsHasRestParameter)
	assert.Equal(t, sig.getIsolatedSignatureType(c), c.typ(types[2]))
	sig.setTypeParameters(c, nil)
	sig.setThisParameter(c, 0)
	assert.Equal(t, sig.extra, entry)
	assert.Assert(t, sig.TypeParameters(c) == nil)
	assert.Equal(t, c.typ(sig.resolvedReturnType), c.typ(types[0]))
	assert.Equal(t, sig.minArgumentCount, int32(1))
	assert.Equal(t, sig.getResolvedTypePredicate(c), predicate)
	runtime.KeepAlive(c)
}

func TestRelationTable(t *testing.T) {
	t.Parallel()
	var table relationTable[uint32]
	want := map[RelationKey]uint32{}
	rng := rand.New(rand.NewPCG(1, 2))
	for i := range 20000 {
		key := RelationKey(rng.IntN(3000) + 1)
		if rng.IntN(3) == 0 {
			table.delete(key)
			delete(want, key)
		} else {
			table.set(key, uint32(i))
			want[key] = uint32(i)
		}
	}
	assert.Equal(t, table.count, len(want))
	for key := RelationKey(1); key <= 3000; key++ {
		assert.Equal(t, table.get(key), want[key])
		_, ok := want[key]
		assert.Equal(t, table.has(key), ok)
	}
	table.clear()
	assert.Assert(t, !table.has(1))
}
