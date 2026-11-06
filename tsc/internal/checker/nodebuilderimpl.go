package checker

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/collections"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/debug"
	"github.com/microsoft/TypeScript/tsc/internal/jsnum"
	"github.com/microsoft/TypeScript/tsc/internal/module"
	"github.com/microsoft/TypeScript/tsc/internal/modulespecifiers"
	"github.com/microsoft/TypeScript/tsc/internal/nodebuilder"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
	"github.com/microsoft/TypeScript/tsc/internal/pseudochecker"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/internal/stringutil"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

type CompositeSymbolIdentity struct {
	isConstructorNode bool
	symbolId          ast.SymbolSerial
	nodeId            ast.NodeId
}

type TrackedSymbolArgs struct {
	symbol               ast.Symbol
	enclosingDeclaration ast.Node
	meaning              ast.SymbolFlags
}

type SerializedTypeEntry struct {
	node           ast.Node
	truncating     bool
	addedLength    int
	trackedSymbols []*TrackedSymbolArgs
}

type CompositeTypeCacheIdentity struct {
	typeId              TypeId
	flags               nodebuilder.Flags
	internalFlags       nodebuilder.InternalFlags
	inferTypeParameters CacheHashKey
}

type NodeBuilderLinks struct {
	serializedTypes                  map[CompositeTypeCacheIdentity]*SerializedTypeEntry // Collection of types serialized at this location
	fakeScopeForSignatureDeclaration *string                                             // If present, this is a fake scope injected into an enclosing declaration chain.
}

type NodeBuilderSymbolLinks struct {
	specifierCache module.ModeAwareCache[moduleSpecifierResult]
}

type moduleSpecifierResult struct {
	specifier            tspath.ModuleSpecifier
	importAttributesType *Type
}

type NodeBuilderContext struct {
	host                            Host
	tracker                         nodebuilder.SymbolTracker
	approximateLength               int
	maxTruncationLength             int
	encounteredError                bool
	truncating                      bool
	reportedDiagnostic              bool
	flags                           nodebuilder.Flags
	internalFlags                   nodebuilder.InternalFlags
	depth                           int
	maxExpansionDepth               int // -1 means no expansion, 0+ = verbosity levels
	typeStack                       []TypeId
	canIncreaseExpansionDepth       bool
	expansionTruncated              bool
	enclosingDeclaration            ast.Node
	enclosingFile                   *ast.SourceFile
	inferTypeParameters             []TypeId
	visitedTypes                    collections.Set[TypeId]
	symbolDepth                     map[CompositeSymbolIdentity]int
	trackedSymbols                  []*TrackedSymbolArgs
	mapper                          *TypeMapper
	reverseMappedStack              []ast.SymbolId
	enclosingSymbolTypes            map[ast.SymbolSerial]*Type
	suppressReportInferenceFallback bool
	remappedSymbolReferences        map[ast.SymbolSerial]ast.SymbolId

	// per signature scope state
	typeParameterNames                    collections.CopyOnWriteMap[TypeId, ast.Identifier]
	typeParameterNamesByText              collections.CopyOnWriteSet[ast.SymbolName]
	typeParameterNamesByTextNextNameCount collections.CopyOnWriteMap[ast.SymbolName, int]
	typeParameterSymbolList               collections.CopyOnWriteSet[ast.SymbolSerial]
}

type NodeBuilderImpl struct {
	// host members
	f  *ast.NodeFactory
	ch *Checker
	e  *printer.EmitContext
	pc *pseudochecker.PseudoChecker

	// cache
	links       core.LinkStore[ast.Node, NodeBuilderLinks]
	symbolLinks core.LinkStore[ast.SymbolId, NodeBuilderSymbolLinks]

	// state
	ctx *NodeBuilderContext

	// reusable visitor
	cloneBindingNameVisitor *ast.NodeVisitor

	// symbols for synthesized identifiers, needed for e.g. inlay hints
	idToSymbol map[ast.IdentifierNode]ast.Symbol
}

const (
	defaultMaximumTruncationLength      = 160
	noTruncationMaximumTruncationLength = 1_000_000
)

// Node builder utility functions

func newNodeBuilderImpl(ch *Checker, e *printer.EmitContext, idToSymbol map[ast.IdentifierNode]ast.Symbol) *NodeBuilderImpl {
	if idToSymbol == nil {
		idToSymbol = make(map[ast.IdentifierNode]ast.Symbol)
	}
	b := &NodeBuilderImpl{f: e.Factory.AsNodeFactory(), ch: ch, e: e, idToSymbol: idToSymbol, pc: pseudochecker.NewPseudoChecker(ch.strictNullChecks, ch.exactOptionalPropertyTypes)}
	b.cloneBindingNameVisitor = ast.NewNodeVisitor(b.cloneBindingName, b.f, ast.NodeVisitorHooks{})
	return b
}

func (b *NodeBuilderImpl) saveRestoreFlags() func() {
	flags := b.ctx.flags
	internalFlags := b.ctx.internalFlags
	depth := b.ctx.depth

	return func() {
		b.ctx.flags = flags
		b.ctx.internalFlags = internalFlags
		b.ctx.depth = depth
	}
}

func (b *NodeBuilderImpl) checkTruncationLength() bool {
	if b.ctx.truncating {
		return b.ctx.truncating
	}
	var maxLength int
	if b.ctx.flags&nodebuilder.FlagsNoTruncation != 0 {
		maxLength = noTruncationMaximumTruncationLength
	} else if b.ctx.maxTruncationLength > 0 {
		maxLength = b.ctx.maxTruncationLength
	} else {
		maxLength = defaultMaximumTruncationLength
	}
	b.ctx.truncating = b.ctx.approximateLength > maxLength
	return b.ctx.truncating
}

// checkTruncationLengthIfExpanding returns true if maxExpansionDepth >= 0 and truncation length exceeded.
// When expanding, we need to mark the output as truncated so we know not to offer further expansion.
func (b *NodeBuilderImpl) checkTruncationLengthIfExpanding() bool {
	if b.ctx.maxExpansionDepth >= 0 && b.checkTruncationLength() {
		b.ctx.expansionTruncated = true
		return true
	}
	return false
}

// isExpandableType reports whether t has a named representation that could be inlined
// as its structural form during hover expansion. Filters out lib types.
// When isAlias is true, checks whether t's alias symbol is from user code (not lib).
func (b *NodeBuilderImpl) isExpandableType(t *Type, isAlias bool) bool {
	if isAlias {
		return !b.ch.isLibSymbolForHoverVerbosity(t.Alias(b.ch).Symbol())
	}
	if b.ch.IsLibTypeForHoverVerbosity(t) {
		return false
	}
	objectFlags := t.objectFlags
	if t.flags&TypeFlagsEnumLike != 0 ||
		objectFlags&ObjectFlagsReference != 0 ||
		objectFlags&ObjectFlagsClassOrInterface != 0 {
		return true
	}
	if objectFlags&ObjectFlagsAnonymous != 0 && b.ch.symbolOfType(t) != 0 &&
		b.ch.symbolFlags(b.ch.symbolOfType(t))&(ast.SymbolFlagsClass|ast.SymbolFlagsEnum|ast.SymbolFlagsValueModule|ast.SymbolFlagsFunction|ast.SymbolFlagsMethod) != 0 {
		return true
	}
	return false
}

// isTypeOnStack reports whether t is already being processed in the current expansion,
// excluding the last element (which is the type currently being serialized by typeToTypeNode).
func (b *NodeBuilderImpl) isTypeOnStack(t *Type) bool {
	for i := range len(b.ctx.typeStack) - 1 {
		if b.ch.typ(b.ctx.typeStack[i]) == t {
			return true
		}
	}
	return false
}

// shouldExpandType decides whether to expand this type at the current depth.
// Returns true when depth < maxExpansionDepth (expand now).
// At the boundary (depth == maxExpansionDepth), sets canIncreaseExpansionDepth
// to signal that a higher verbosity level would reveal more detail.
// Returns false when expansion is disabled (maxExpansionDepth < 0), the type
// is not expandable, or the type is cyclic.
func (b *NodeBuilderImpl) shouldExpandType(t *Type, isAlias bool) bool {
	if b.ctx.maxExpansionDepth < 0 {
		return false
	}
	if !b.isExpandableType(t, isAlias) {
		return false
	}
	if b.isTypeOnStack(t) {
		return false
	}
	if b.ctx.depth < b.ctx.maxExpansionDepth {
		return true
	}
	b.ctx.canIncreaseExpansionDepth = true
	return false
}

// isActivelyExpanding reports whether the current depth is below maxExpansionDepth,
// meaning type-node reuse should be skipped so typeToTypeNode can expand named types.
func (b *NodeBuilderImpl) isActivelyExpanding() bool {
	return b.ctx.maxExpansionDepth > 0 && b.ctx.depth < b.ctx.maxExpansionDepth
}

// checkTypeExpandability probes whether a type (or its type arguments) could be expanded,
// for use after type-node reuse where shouldExpandType was never called.
// Delegates to shouldExpandType for the actual check, then recurses into type arguments
// of reference types (e.g., Apple inside Promise<Apple>).
func (b *NodeBuilderImpl) checkTypeExpandability(t *Type) {
	if b.ctx.maxExpansionDepth < 0 || t == nil || b.ctx.canIncreaseExpansionDepth {
		return
	}
	// Push t onto the type stack so shouldExpandType's cycle detection works correctly.
	b.ctx.typeStack = append(b.ctx.typeStack, t.ref())
	defer func() {
		b.ctx.typeStack = b.ctx.typeStack[:len(b.ctx.typeStack)-1]
	}()
	// If t is an ancestor in the current expansion, return early to avoid unbounded recursion.
	if b.isTypeOnStack(t) {
		return
	}
	if t.Alias(b.ch) != nil {
		b.shouldExpandType(t, true)
	}
	if !b.ctx.canIncreaseExpansionDepth {
		b.shouldExpandType(t, false)
	}
	if b.ctx.canIncreaseExpansionDepth {
		return
	}
	// Recurse into type arguments (e.g., check Apple in Promise<Apple>).
	if t.objectFlags&ObjectFlagsReference != 0 {
		for _, argId := range b.ch.getTypeArguments(t) {
			arg := b.ch.typ(argId)
			b.checkTypeExpandability(arg)
			if b.ctx.canIncreaseExpansionDepth {
				return
			}
		}
	}
}

func (b *NodeBuilderImpl) appendReferenceToType(root ast.TypeNode, ref ast.TypeNode) ast.TypeNode {
	if ast.IsImportTypeNode(root) {
		// first shift type arguments

		// !!! In the old emitter, an Identifier could have type arguments for use with quickinfo:
		// typeArguments := root.TypeArguments
		// qualifier := root.AsImportTypeNode().Qualifier
		// if qualifier != nil {
		// 	if ast.IsIdentifier(qualifier) {
		// 		if typeArguments != getIdentifierTypeArguments(qualifier) {
		// 			qualifier = setIdentifierTypeArguments(b.f.CloneNode(qualifier), typeArguments)
		// 		}
		// 	} else {
		// 		if typeArguments != getIdentifierTypeArguments(qualifier.Right) {
		// 			qualifier = b.f.UpdateQualifiedName(qualifier, qualifier.Left, setIdentifierTypeArguments(b.f.cloneNode(qualifier.Right), typeArguments))
		// 		}
		// 	}
		// }
		// !!! Without the above, nested type args are silently elided
		imprt := root.AsImportTypeNode()
		// then move qualifiers
		ids := getAccessStack(ref)
		qualifier := root.AsImportTypeNode().Qualifier()
		for _, id := range ids {
			if !qualifier.IsNil() {
				qualifier = b.f.NewQualifiedName(qualifier, id)
			} else {
				qualifier = id
			}
		}
		return b.f.UpdateImportTypeNode(imprt, imprt.IsTypeOf(), imprt.Argument(), imprt.Attributes(), qualifier, ref.TypeArgumentList())
	} else if ast.IsTypeReferenceNode(root) {
		typeRef := root.AsTypeReferenceNode()
		if b.ctx.flags&nodebuilder.FlagsUseInstantiationExpressions != 0 && !typeRef.TypeArguments().IsNil() && typeRef.TypeArguments().Len() != 0 {
			expr := b.createExpressionWithTypeArguments(b.createAccessExpression(typeRef.TypeName()), typeRef.TypeArguments())
			for _, id := range getAccessStack(ref) {
				expr = b.f.NewPropertyAccessExpression(expr, ast.Node{}, id, ast.NodeFlagsNone)
			}
			return expr
		}
		var typeName ast.Node = typeRef.TypeName()
		for _, id := range getAccessStack(ref) {
			typeName = b.f.NewQualifiedName(typeName, id)
		}
		return b.f.UpdateTypeReferenceNode(root.AsTypeReferenceNode(), typeName, ref.TypeArgumentList())
	}
	expr := b.createAccessExpression(root)
	for _, id := range getAccessStack(ref) {
		expr = b.f.NewPropertyAccessExpression(expr, ast.Node{}, id, ast.NodeFlagsNone)
	}
	return expr
}

func getAccessStack(ref ast.Node) []ast.Node {
	var state ast.Node = ref.AsTypeReferenceNode().TypeName()
	ids := []ast.Node{}
	for !ast.IsIdentifier(state) {
		entity := state.AsQualifiedName()
		ids = append([]ast.Node{entity.Right()}, ids...)
		state = entity.Left()
	}
	ids = append([]ast.Node{state}, ids...)
	return ids
}

func isClassInstanceSide(c *Checker, t *Type) bool {
	return c.symbolOfType(t) != 0 && c.symbolFlags(c.symbolOfType(t))&ast.SymbolFlagsClass != 0 && (t == c.getDeclaredTypeOfClassOrInterface(c.symbolOfType(t)) || (t.flags&TypeFlagsObject != 0 && t.objectFlags&ObjectFlagsIsClassInstanceClone != 0))
}

func (b *NodeBuilderImpl) createElidedInformationPlaceholder() ast.TypeNode {
	b.ctx.approximateLength += 3
	if b.ctx.flags&nodebuilder.FlagsNoTruncation == 0 {
		return b.f.NewTypeReferenceNode(b.f.NewIdentifier("..."), ast.TypeList{} /*typeArguments*/)
	}
	return b.e.AddSyntheticLeadingComment(b.f.NewKeywordTypeNode(ast.KindAnyKeyword), ast.KindMultiLineCommentTrivia, "elided", false /*hasTrailingNewLine*/)
}

func (b *NodeBuilderImpl) mapToTypeNodes(list []TypeId, isBareList bool) ast.NodeList {
	if len(list) == 0 {
		return ast.NodeList{}
	}

	if b.checkTruncationLength() {
		if !isBareList {
			var node ast.Node
			if b.ctx.flags&nodebuilder.FlagsNoTruncation != 0 {
				node = b.e.AddSyntheticLeadingComment(b.f.NewKeywordTypeNode(ast.KindAnyKeyword), ast.KindMultiLineCommentTrivia, "elided", false /*hasTrailingNewLine*/)
			} else {
				node = b.f.NewTypeReferenceNode(b.f.NewIdentifier("..."), ast.TypeList{} /*typeArguments*/)
			}
			return b.f.NewNodeList([]ast.Node{node})
		} else if len(list) > 2 {
			nodes := []ast.Node{
				b.typeToTypeNode(b.ch.typ(list[0])),
				{},
				b.typeToTypeNode(b.ch.typ(list[len(list)-1])),
			}

			if b.ctx.flags&nodebuilder.FlagsNoTruncation != 0 {
				nodes[1] = b.e.AddSyntheticLeadingComment(b.f.NewKeywordTypeNode(ast.KindAnyKeyword), ast.KindMultiLineCommentTrivia, fmt.Sprintf("... %d more elided ...", len(list)-2), false /*hasTrailingNewLine*/)
			} else {
				text := fmt.Sprintf("... %d more ...", len(list)-2)
				nodes[1] = b.f.NewTypeReferenceNode(b.f.NewIdentifier(text), ast.TypeList{} /*typeArguments*/)
			}
			return b.f.NewNodeList(nodes)
		}
	}

	mayHaveNameCollisions := b.ctx.flags&nodebuilder.FlagsUseFullyQualifiedType == 0
	type seenName struct {
		t *Type
		i int
	}
	var seenNames *collections.MultiMap[ast.SymbolName, seenName]
	if mayHaveNameCollisions {
		seenNames = &collections.MultiMap[ast.SymbolName, seenName]{}
	}

	result := make([]ast.Node, 0, len(list))

	for i, tId := range list {
		t := b.ch.typ(tId)
		displayIndex := i + 1
		if b.checkTruncationLength() && (displayIndex+2 < len(list)-1) {
			if b.ctx.flags&nodebuilder.FlagsNoTruncation != 0 {
				result = append(result, b.e.AddSyntheticLeadingComment(b.f.NewKeywordTypeNode(ast.KindAnyKeyword), ast.KindMultiLineCommentTrivia, fmt.Sprintf("... %d more elided ...", len(list)-displayIndex), false /*hasTrailingNewLine*/))
			} else {
				text := fmt.Sprintf("... %d more ...", len(list)-displayIndex)
				result = append(result, b.f.NewTypeReferenceNode(b.f.NewIdentifier(text), ast.TypeList{} /*typeArguments*/))
			}
			typeNode := b.typeToTypeNode(b.ch.typ(list[len(list)-1]))
			if !typeNode.IsNil() {
				result = append(result, typeNode)
			}
			break
		}
		b.ctx.approximateLength += 2 // Account for whitespace + separator
		typeNode := b.typeToTypeNode(t)
		if !typeNode.IsNil() {
			result = append(result, typeNode)
			if seenNames != nil && isIdentifierTypeReference(typeNode) {
				seenNames.Add(typeNode.AsTypeReferenceNode().TypeName().TextSymbolName(), seenName{t, len(result) - 1})
			}
		}
	}

	if seenNames != nil {
		// To avoid printing types like `[Foo, Foo]` or `Bar & Bar` where
		// occurrences of the same name actually come from different
		// namespaces, go through the single-identifier type reference nodes
		// we just generated, and see if any names were generated more than
		// once while referring to different types. If so, regenerate the
		// type node for each entry by that name with the
		// `UseFullyQualifiedType` flag enabled.
		restoreFlags := b.saveRestoreFlags()
		b.ctx.flags |= nodebuilder.FlagsUseFullyQualifiedType
		for types := range seenNames.Values() {
			if !arrayIsHomogeneous(types, func(x, y seenName) bool {
				return b.ch.typesAreSameReference(x.t, y.t)
			}) {
				for _, seen := range types {
					result[seen.i] = b.typeToTypeNode(seen.t)
				}
			}
		}
		restoreFlags()
	}

	return b.f.NewNodeList(result)
}

func (b *NodeBuilderImpl) serializeTypeName(node ast.Node, isTypeOf bool, typeArguments ast.NodeList) ast.Node {
	meaning := ast.SymbolFlagsType
	if isTypeOf {
		meaning = ast.SymbolFlagsValue
	}
	symbol := b.ch.resolveEntityName(node, meaning, true, false, node)
	if symbol == 0 {
		return ast.Node{}
	}

	resolvedSymbol := symbol
	if b.ch.symbolFlags(symbol)&ast.SymbolFlagsAlias != 0 {
		resolvedSymbol = b.ch.resolveAlias(symbol)
	}

	if b.ch.isSymbolAccessible(symbol, b.ctx.enclosingDeclaration, meaning, false).Accessibility != printer.SymbolAccessibilityAccessible {
		return ast.Node{}
	}
	return b.symbolToTypeNode(resolvedSymbol, meaning, typeArguments)
}

func isIdentifierTypeReference(node ast.Node) bool {
	return ast.IsTypeReferenceNode(node) && ast.IsIdentifier(node.AsTypeReferenceNode().TypeName())
}

func arrayIsHomogeneous[T any](array []T, comparer func(a, B T) bool) bool {
	if len(array) < 2 {
		return true
	}
	first := array[0]
	for i := 1; i < len(array); i++ {
		target := array[i]
		if !comparer(first, target) {
			return false
		}
	}
	return true
}

func (c *Checker) typesAreSameReference(a, b *Type) bool {
	return a == b || c.symbolOfType(a) != 0 && c.symbolOfType(a) == c.symbolOfType(b) || a.Alias(c) != nil && a.Alias(c) == b.Alias(c)
}

func (b *NodeBuilderImpl) setCommentRange(node ast.Node, range_ ast.Node) {
	if !range_.IsNil() && b.ctx.enclosingFile != nil && b.ctx.enclosingFile == ast.GetSourceFileOfNode(range_) {
		// Copy comments to node for declaration emit
		b.e.AssignCommentRange(node, range_)
	}
}

func (b *NodeBuilderImpl) typeNodeIsEquivalentToType(annotatedDeclaration ast.Node, t *Type, typeFromTypeNode *Type) bool {
	if typeFromTypeNode == t {
		return true
	}
	if annotatedDeclaration.IsNil() {
		return false
	}
	// !!!
	// used to be hasEffectiveQuestionToken for JSDoc
	if isOptionalDeclaration(annotatedDeclaration) {
		return b.ch.getTypeWithFacts(t, TypeFactsNEUndefined) == typeFromTypeNode
	}
	return false
}

func (b *NodeBuilderImpl) canReuseExistingJSTypeNode(existing ast.TypeNode, t *Type) bool {
	return b.ch.getIntendedTypeFromJSDocTypeReference(existing) == nil && b.existingTypeNodeIsNotReferenceOrIsReferenceWithCompatibleTypeArgumentCount(existing, t)
}

func (b *NodeBuilderImpl) tryGetResolvedSymbolFromTypeNode(node ast.Node) ast.SymbolId {
	if node.IsNil() || node.Parent().IsNil() {
		return 0
	}
	b.ch.getTypeFromTypeNode(node)
	// call to ensure symbol is resolved
	return b.ch.symbolNodeLinks.get(node)
}

func (b *NodeBuilderImpl) existingTypeNodeIsNotReferenceOrIsReferenceWithCompatibleTypeArgumentCount(existing ast.TypeNode, t *Type) bool {
	// In JS, you can say something like `Foo` and get a `Foo<any>` implicitly - we don't want to preserve that original `Foo` in these cases, though.
	if t.objectFlags&ObjectFlagsReference == 0 {
		return true
	}
	if !ast.IsTypeReferenceNode(existing) {
		return true
	}

	symbol := b.tryGetResolvedSymbolFromTypeNode(existing)
	if symbol == 0 {
		return true
	}

	// `type` is a reference type, and `existing` is a type reference node, but we still need to make sure they refer to the _same_ target type
	// before we go comparing their type argument counts.

	existingTarget := b.ch.getDeclaredTypeOfSymbol(symbol)
	if existingTarget == nil || existingTarget != t.AsTypeReference().target(b.ch) {
		return true
	}
	return existing.TypeArgumentList().Len() >= b.ch.getMinTypeArgumentCount(t.AsTypeReference().target(b.ch).AsInterfaceType().TypeParameters(b.ch))
}

func (b *NodeBuilderImpl) tryReuseExistingNonParameterTypeNode(existing ast.TypeNode, t *Type, host ast.Node, annotationType *Type) ast.TypeNode {
	if host.IsNil() {
		host = b.ctx.enclosingDeclaration
	}
	if annotationType == nil {
		annotationType = b.getTypeFromTypeNode(existing, true)
	}
	if annotationType != nil && b.typeNodeIsEquivalentToType(host, t, annotationType) && b.canReuseExistingJSTypeNode(existing, t) {
		result := b.tryReuseExistingNodeHelper(existing)
		if !result.IsNil() {
			return result
		}
	}
	return ast.Node{}
}

func (b *NodeBuilderImpl) getResolvedTypeWithoutAbstractConstructSignatures(t *StructuredType) *Type {
	if len(t.ConstructSignatures(b.ch)) == 0 {
		return t.AsType()
	}
	if t.resolvedExtra(b.ch).objectTypeWithoutAbstractConstructSignatures != 0 {
		return b.ch.typ(t.resolvedExtra(b.ch).objectTypeWithoutAbstractConstructSignatures)
	}
	constructSignatures := core.Filter(t.ConstructSignatures(b.ch), func(signature *Signature) bool {
		return signature.flags&SignatureFlagsAbstract == 0
	})
	if len(constructSignatures) == len(t.ConstructSignatures(b.ch)) {
		t.resolvedExtra(b.ch).objectTypeWithoutAbstractConstructSignatures = t.AsType().ref()
		return t.AsType()
	}
	typeCopy := b.ch.newAnonymousType(b.ch.symbolOfType(t.AsType()), t.getMembers(b.ch), t.CallSignatures(b.ch), core.IfElse(len(constructSignatures) > 0, constructSignatures, []*Signature{}), t.getIndexInfos(b.ch))
	t.resolvedExtra(b.ch).objectTypeWithoutAbstractConstructSignatures = typeCopy.ref()
	typeCopy.AsStructuredType().ensureResolvedExtra(b.ch).objectTypeWithoutAbstractConstructSignatures = typeCopy.ref()
	return typeCopy
}

func (b *NodeBuilderImpl) symbolToNode(symbol ast.SymbolId, meaning ast.SymbolFlags) ast.Node {
	if b.ctx.internalFlags&nodebuilder.InternalFlagsWriteComputedProps != 0 {
		if !b.ch.symbolValueDeclaration(symbol).IsNil() {
			name := ast.GetNameOfDeclaration(b.ch.symbolValueDeclaration(symbol))
			if !name.IsNil() && ast.IsComputedPropertyName(name) {
				return name
			}
		}
		if b.ch.valueSymbolLinks.Has(symbol) {
			nameType := b.ch.valueSymbolLinks.Get(symbol).getNameType(b.ch)
			if nameType != nil && nameType.flags&(TypeFlagsEnumLiteral|TypeFlagsUniqueESSymbol) != 0 {
				oldEnclosing := b.ctx.enclosingDeclaration
				b.ctx.enclosingDeclaration = b.ch.symbolValueDeclaration(b.ch.symbolOfType(nameType))
				result := b.f.NewComputedPropertyName(b.symbolToExpressionWorker(b.ch.symbolOfType(nameType), meaning))
				b.ctx.enclosingDeclaration = oldEnclosing
				return result
			}
		}
	}
	return b.symbolToExpression(symbol, meaning)
}

func (b *NodeBuilderImpl) symbolToName(symbol ast.SymbolId, meaning ast.SymbolFlags, expectsIdentifier bool) ast.Node {
	chain := b.lookupSymbolChain(symbol, meaning, false)
	if expectsIdentifier && len(chain) != 1 && !b.ctx.encounteredError && (b.ctx.flags&nodebuilder.FlagsAllowQualifiedNameInPlaceOfIdentifier != 0) {
		b.ctx.encounteredError = true
	}
	return b.createEntityNameFromSymbolChain(chain, len(chain)-1)
}

func (b *NodeBuilderImpl) createEntityNameFromSymbolChain(chain []ast.SymbolId, index int) ast.Node {
	// typeParameterNodes := b.lookupTypeParameterNodes(chain, index)
	symbol := chain[index]

	if index == 0 {
		b.ctx.flags |= nodebuilder.FlagsInInitialEntityName
	}
	symbolName := b.getNameOfSymbolAsWritten(symbol)
	if index == 0 {
		b.ctx.flags ^= nodebuilder.FlagsInInitialEntityName
	}

	identifier := b.newIdentifier(symbolName, symbol)
	b.e.AddEmitFlags(identifier, printer.EFNoAsciiEscaping)
	// !!! TODO: smuggle type arguments out
	// if (typeParameterNodes) setIdentifierTypeArguments(identifier, factory.createNodeArray<TypeNode | TypeParameterDeclaration>(typeParameterNodes));
	// identifier.symbol = symbol;
	// expression = identifier;
	if index > 0 {
		return b.f.NewQualifiedName(
			b.createEntityNameFromSymbolChain(chain, index-1),
			identifier,
		)
	}
	return identifier
}

// TODO: Audit usages of symbolToEntityNameNode - they should probably all be symbolToName
func (b *NodeBuilderImpl) symbolToEntityNameNode(symbol ast.SymbolId) ast.EntityName {
	identifier := b.newIdentifier(b.ch.symbolName(symbol).String(), symbol)
	if b.ch.symbolParent(symbol) != 0 {
		return b.f.NewQualifiedName(b.symbolToEntityNameNode(b.ch.symbolParent(symbol)), identifier)
	}
	return identifier
}

func (b *NodeBuilderImpl) symbolToTypeNode(symbol ast.SymbolId, mask ast.SymbolFlags, typeArguments ast.NodeList) ast.TypeNode {
	chain := b.lookupSymbolChain(symbol, mask, (b.ctx.flags&nodebuilder.FlagsUseAliasDefinedOutsideCurrentScope == 0)) // If we're using aliases outside the current scope, dont bother with the module
	if len(chain) == 0 {
		return ast.Node{} // TODO: shouldn't be possible, `lookupSymbolChain` should always at least return the input symbol and issue an error
	}
	isTypeOf := mask == ast.SymbolFlagsValue
	if b.ch.symbolDeclarations(chain[0]).Some(hasNonGlobalAugmentationExternalModuleSymbol) {
		// module is root, must use `ImportTypeNode`
		var nonRootParts ast.Node
		if len(chain) > 1 {
			nonRootParts = b.createAccessFromSymbolChain(chain, len(chain)-1, 1, typeArguments)
		}
		typeParameterNodes := typeArguments
		if typeParameterNodes.IsNil() {
			typeParameterNodes = b.lookupTypeParameterNodes(chain, 0)
		}
		contextFile := ast.GetSourceFileOfNode(b.e.MostOriginal(b.ctx.enclosingDeclaration)) // TODO: Just use b.ctx.enclosingFile ? Or is the delayed lookup important for context moves?
		targetFile := ast.GetSourceFileOfModule(b.ch.symbolOf(chain[0]))
		var specifierResult moduleSpecifierResult
		var importModeOverride core.ResolutionMode
		if b.ch.compilerOptions.GetModuleResolutionKind() == core.ModuleResolutionKindNode16 || b.ch.compilerOptions.GetModuleResolutionKind() == core.ModuleResolutionKindNodeNext {
			// An `import` type directed at an esm format file is only going to resolve in esm mode - set the esm mode assertion
			if targetFile != nil && contextFile != nil && b.ch.program.GetEmitModuleFormatOfFile(targetFile) == core.ModuleKindESNext && b.ch.program.GetEmitModuleFormatOfFile(targetFile) != b.ch.program.GetEmitModuleFormatOfFile(contextFile) {
				specifierResult = b.getSpecifierForModuleSymbol(chain[0], core.ModuleKindESNext)
				importModeOverride = core.ModuleKindESNext
			}
		}
		if len(specifierResult.specifier) == 0 {
			specifierResult = b.getSpecifierForModuleSymbol(chain[0], core.ResolutionModeNone)
		}
		if (b.ctx.flags&nodebuilder.FlagsAllowNodeModulesRelativePaths == 0) /* && b.ch.compilerOptions.GetModuleResolutionKind() != core.ModuleResolutionKindClassic */ && strings.Contains(specifierResult.specifier.AsString(), "/node_modules/") {
			oldSpecifierResult := specifierResult

			if b.ch.compilerOptions.GetModuleResolutionKind() == core.ModuleResolutionKindNode16 || b.ch.compilerOptions.GetModuleResolutionKind() == core.ModuleResolutionKindNodeNext {
				// We might be able to write a portable import type using a mode override; try specifier generation again, but with a different mode set
				swappedMode := core.ModuleKindESNext
				if b.ch.program.GetEmitModuleFormatOfFile(contextFile) == core.ModuleKindESNext {
					swappedMode = core.ModuleKindCommonJS
				}
				specifierResult = b.getSpecifierForModuleSymbol(chain[0], swappedMode)

				if strings.Contains(specifierResult.specifier.AsString(), "/node_modules/") {
					// Still unreachable :(
					specifierResult = oldSpecifierResult
				} else {
					importModeOverride = swappedMode
				}
			}

			if importModeOverride == core.ResolutionModeNone {
				// If ultimately we can only name the symbol with a reference that dives into a `node_modules` folder, we should error
				// since declaration files with these kinds of references are liable to fail when published :(
				b.ctx.encounteredError = true
				b.ctx.tracker.ReportLikelyUnsafeImportRequiredError(oldSpecifierResult.specifier.AsString(), b.ch.symbolName(symbol).String())
			}
		}

		attributes := b.createImportAttributesForModuleSpecifier(specifierResult, importModeOverride)
		lit := b.f.NewLiteralTypeNode(b.newStringLiteral(specifierResult.specifier.AsString()))
		b.ctx.approximateLength += len(specifierResult.specifier) + 10 // specifier + import("")
		if nonRootParts.IsNil() || ast.IsEntityName(nonRootParts) {
			if !nonRootParts.IsNil() {
				// !!! TODO: smuggle type arguments out
				// const lastId = isIdentifier(nonRootParts) ? nonRootParts : nonRootParts.right;
				// setIdentifierTypeArguments(lastId, /*typeArguments*/ undefined);
			}
			return b.f.NewImportTypeNode(isTypeOf, lit, attributes, nonRootParts, typeParameterNodes)
		}

		splitNode := getTopmostIndexedAccessType(nonRootParts.AsIndexedAccessTypeNode())
		qualifier := splitNode.ObjectType().AsTypeReferenceNode().TypeName()
		return b.f.NewIndexedAccessTypeNode(
			b.f.NewImportTypeNode(isTypeOf, lit, attributes, qualifier, typeParameterNodes),
			splitNode.IndexType(),
		)

	}

	entityName := b.createAccessFromSymbolChain(chain, len(chain)-1, 0, typeArguments)
	if ast.IsIndexedAccessTypeNode(entityName) {
		return entityName // Indexed accesses can never be `typeof`
	}
	if ast.IsEntityName(entityName) {
		if isTypeOf {
			return b.f.NewTypeQueryNode(entityName, ast.TypeList{})
		}
		return b.f.NewTypeReferenceNode(entityName, typeArguments)
	}
	if isTypeOf && ast.IsExpressionWithTypeArguments(entityName) {
		expr := entityName.AsExpressionWithTypeArguments()
		return b.f.NewTypeQueryNode(b.f.DeepCloneNode(expr.Expression()), expr.TypeArguments())
	}
	return entityName
}

func getTopmostIndexedAccessType(node ast.IndexedAccessTypeNode) ast.IndexedAccessTypeNode {
	if ast.IsIndexedAccessTypeNode(node.ObjectType()) {
		return getTopmostIndexedAccessType(node.ObjectType().AsIndexedAccessTypeNode())
	}
	return node
}

func (b *NodeBuilderImpl) createAccessFromSymbolChain(chain []ast.SymbolId, index int, stopper int, overrideTypeArguments ast.NodeList) ast.Node {
	typeParameterNodes := overrideTypeArguments
	if index != (len(chain) - 1) {
		typeParameterNodes = b.lookupTypeParameterNodes(chain, index)
	}
	symbol := chain[index]
	var parent ast.SymbolId
	if index > 0 {
		parent = chain[index-1]
	}

	var symbolName string
	if index == 0 {
		b.ctx.flags |= nodebuilder.FlagsInInitialEntityName
		symbolName = b.getNameOfSymbolAsWritten(symbol)
		b.ctx.approximateLength += len(symbolName) + 1
		b.ctx.flags ^= nodebuilder.FlagsInInitialEntityName
	} else {
		// lookup a ref to symbol within parent to handle export aliases
		if parent != 0 {
			exports := b.ch.getExportsOfSymbol(parent)
			if !exports.IsNil() {
				// avoid exhaustive iteration in the common case
				res, ok := exports.LookupId(b.ch.symbols, b.ch.symbolName(symbol))
				if b.ch.symbolName(symbol) != ast.InternalSymbolNameExportEquals && !b.ch.symbolName(symbol).IsLateBoundName() && ok && res != 0 && b.ch.getSymbolIfSameReference(res, symbol) != 0 {
					symbolName = b.ch.symbolName(symbol).String()
				} else {
					results := make(map[ast.SymbolId]ast.SymbolName, 1)
					for name, ex := range exports.AllIds(b.ch.symbols) {
						if b.ch.getSymbolIfSameReference(ex, symbol) != 0 && !name.IsLateBoundName() && name != ast.InternalSymbolNameExportEquals {
							results[ex] = name
							// break // must collect all results and sort them - exports are randomly iterated
						}
					}
					resultSymbols := slices.Collect(maps.Keys(results))
					if len(resultSymbols) > 0 {
						b.ch.sortSymbols(resultSymbols)
						symbolName = results[resultSymbols[0]].String()
					}
				}
			}
		}
	}

	if len(symbolName) == 0 {
		var name ast.Node
		for _, d := range b.ch.symbolDeclarations(symbol).All() {
			name = ast.GetNameOfDeclaration(d)
			if !name.IsNil() {
				break
			}
		}
		if !name.IsNil() && ast.IsComputedPropertyName(name) && ast.IsEntityName(name.Expression()) {
			lhs := b.createAccessFromSymbolChain(chain, index-1, stopper, overrideTypeArguments)
			if ast.IsEntityName(lhs) {
				return b.f.NewIndexedAccessTypeNode(
					b.f.NewParenthesizedTypeNode(b.f.NewTypeQueryNode(lhs, ast.TypeList{})),
					b.f.NewTypeQueryNode(name.Expression(), ast.TypeList{}),
				)
			}
			return lhs
		}
		symbolName = b.getNameOfSymbolAsWritten(symbol)
	}
	b.ctx.approximateLength += len(symbolName) + 1

	if (b.ctx.flags&nodebuilder.FlagsForbidIndexedAccessSymbolReferences == 0) && parent != 0 &&
		!b.ch.getMembersOfSymbol(parent).IsNil() && b.ch.getMembersOfSymbol(parent).GetId(b.ch.symbols, b.ch.symbolName(symbol)) != 0 &&
		b.ch.getSymbolIfSameReference(b.ch.getMembersOfSymbol(parent).GetId(b.ch.symbols, b.ch.symbolName(symbol)), symbol) != 0 {
		// Should use an indexed access
		lhs := b.createAccessFromSymbolChain(chain, index-1, stopper, overrideTypeArguments)
		if ast.IsIndexedAccessTypeNode(lhs) {
			return b.f.NewIndexedAccessTypeNode(
				lhs,
				b.f.NewLiteralTypeNode(b.newStringLiteral(symbolName)),
			)
		}
		return b.f.NewIndexedAccessTypeNode(
			b.f.NewTypeReferenceNode(lhs, typeParameterNodes),
			b.f.NewLiteralTypeNode(b.newStringLiteral(symbolName)),
		)
	}

	identifier := b.newIdentifier(symbolName, symbol)
	b.e.AddEmitFlags(identifier, printer.EFNoAsciiEscaping)

	if index > stopper {
		lhs := b.createAccessFromSymbolChain(chain, index-1, stopper, overrideTypeArguments)
		if b.ctx.flags&nodebuilder.FlagsUseInstantiationExpressions == 0 || ast.IsEntityName(lhs) && (typeParameterNodes.IsNil() || typeParameterNodes.Len() == 0) {
			return b.f.NewQualifiedName(lhs, identifier)
		}
		return b.createExpressionWithTypeArguments(b.f.NewPropertyAccessExpression(b.createAccessExpression(lhs), ast.Node{}, identifier, ast.NodeFlagsNone), typeParameterNodes)
	}
	return identifier
}

func (b *NodeBuilderImpl) symbolToExpression(symbol ast.SymbolId, mask ast.SymbolFlags) ast.Expression {
	b.ctx.tracker.TrackSymbol(b.ch.symbolOf(symbol), b.ctx.enclosingDeclaration, mask)
	return b.symbolToExpressionWorker(symbol, mask)
}

func (b *NodeBuilderImpl) symbolToExpressionWorker(symbol ast.SymbolId, mask ast.SymbolFlags) ast.Expression {
	chain := b.lookupSymbolChainWorker(symbol, mask, false)
	return b.createExpressionFromSymbolChain(chain, len(chain)-1)
}

func (b *NodeBuilderImpl) createExpressionFromSymbolChain(chain []ast.SymbolId, index int) ast.Expression {
	typeParameterNodes := b.lookupExpressionChainTypeArgumentNodes(chain, index)
	symbol := chain[index]

	if index == 0 {
		b.ctx.flags |= nodebuilder.FlagsInInitialEntityName
	}
	symbolName := b.getNameOfSymbolAsWritten(symbol)
	if index == 0 {
		b.ctx.flags ^= nodebuilder.FlagsInInitialEntityName
	}

	if startsWithSingleOrDoubleQuote(symbolName) && b.ch.symbolDeclarations(symbol).Some(hasNonGlobalAugmentationExternalModuleSymbol) {
		specifierResult := b.getSpecifierForModuleSymbol(symbol, core.ResolutionModeNone)
		b.ctx.approximateLength += 2 + len(specifierResult.specifier)
		return b.newStringLiteral(specifierResult.specifier.AsString())
	}

	if index == 0 || canUsePropertyAccess(symbolName) {
		identifier := b.newIdentifier(symbolName, symbol)
		b.e.AddEmitFlags(identifier, printer.EFNoAsciiEscaping)
		b.ctx.approximateLength += 1 + len(symbolName)
		if index > 0 {
			result := b.f.NewPropertyAccessExpression(b.createExpressionFromSymbolChain(chain, index-1), ast.Node{}, identifier, ast.NodeFlagsNone)
			b.e.AddEmitFlags(result, printer.EFNoIndentation)
			return b.createExpressionWithTypeArguments(result, typeParameterNodes)
		}
		return b.createExpressionWithTypeArguments(identifier, typeParameterNodes)
	}

	if startsWithSquareBracket(symbolName) {
		symbolName = symbolName[1 : len(symbolName)-1]
	}

	var expression ast.Expression
	if startsWithSingleOrDoubleQuote(symbolName) && b.ch.symbolFlags(symbol)&ast.SymbolFlagsEnumMember == 0 {
		literalText := stringutil.UnquoteString(symbolName)
		b.ctx.approximateLength += len(literalText) + 2
		expression = b.newStringLiteralEx(literalText, symbolName[0] == '\'')
	} else if jsnum.FromString(symbolName).String() == symbolName {
		// TODO: the follwing in strada would assert if the number is negative, but no such assertion exists here
		// Moreover, what's even guaranteeing the name *isn't* -1 here anyway? Needs double-checking.
		b.ctx.approximateLength += len(symbolName)
		expression = b.f.NewNumericLiteral(symbolName, ast.TokenFlagsNone)
	}
	if expression.IsNil() {
		b.ctx.approximateLength += len(symbolName)
		expression = b.newIdentifier(symbolName, symbol)
		b.e.AddEmitFlags(expression, printer.EFNoAsciiEscaping)
	}
	b.ctx.approximateLength += 2 // []
	return b.createExpressionWithTypeArguments(b.f.NewElementAccessExpression(b.createExpressionFromSymbolChain(chain, index-1), ast.Node{}, expression, ast.NodeFlagsNone), typeParameterNodes)
}

func canUsePropertyAccess(name string) bool {
	if len(name) == 0 {
		return false
	}
	// TODO: in strada, this only used `isIdentifierStart` on the first character, while this checks the whole string for validity
	// - possible strada bug?
	if strings.HasPrefix(name, "#") {
		return len(name) > 1 && scanner.IsIdentifierText(name[1:], core.LanguageVariantStandard)
	}
	return scanner.IsIdentifierText(name, core.LanguageVariantStandard)
}

func startsWithSingleOrDoubleQuote(str string) bool {
	return strings.HasPrefix(str, "'") || strings.HasPrefix(str, "\"")
}

func startsWithSquareBracket(str string) bool {
	return strings.HasPrefix(str, "[")
}

func isDefaultBindingContext(location ast.Node) bool {
	return location.Kind() == ast.KindSourceFile || ast.IsAmbientModule(location)
}

func (b *NodeBuilderImpl) getNameOfSymbolFromNameType(symbol ast.SymbolId) string {
	if b.ch.valueSymbolLinks.Has(symbol) {
		nameType := b.ch.valueSymbolLinks.Get(symbol).getNameType(b.ch)
		if nameType == nil {
			return ""
		}
		if nameType.flags&TypeFlagsStringOrNumberLiteral != 0 {
			var name string
			switch v := nameType.AsLiteralType().Value(b.ch).(type) {
			case string:
				name = v
			case jsnum.Number:
				name = v.String()
			}
			if !scanner.IsIdentifierText(name, core.LanguageVariantStandard) && !isNumericLiteralName(name) {
				return b.ch.valueToString(nameType.AsLiteralType().Value(b.ch))
			}
			if isNumericLiteralName(name) && strings.HasPrefix(name, "-") {
				return fmt.Sprintf("[%s]", name)
			}
			return name
		}
		if nameType.flags&TypeFlagsUniqueESSymbol != 0 {
			text := b.getNameOfSymbolAsWritten(b.ch.symbolOfType(nameType))
			return fmt.Sprintf("[%s]", text)
		}
	}
	return ""
}

/**
* Gets a human-readable name for a symbol.
* Should *not* be used for the right-hand side of a `.` -- use `symbolName(symbol)` for that instead.
*
* Unlike `symbolName(symbol)`, this will include quotes if the name is from a string literal.
* It will also use a representation of a number as written instead of a decimal form, e.g. `0o11` instead of `9`.
 */
func (b *NodeBuilderImpl) getNameOfSymbolAsWritten(symbol ast.SymbolId) string {
	result, ok := b.ctx.remappedSymbolReferences[ast.GetSymbolSerial(b.ch.symbolOf(symbol))]
	if ok {
		symbol = result
	}
	if b.ch.symbolName(symbol) == ast.InternalSymbolNameDefault && (b.ctx.flags&nodebuilder.FlagsUseAliasDefinedOutsideCurrentScope == 0) &&
		// If it's not the first part of an entity name, it must print as `default`
		((b.ctx.flags&nodebuilder.FlagsInInitialEntityName == 0) ||
			// if the symbol is synthesized, it will only be referenced externally it must print as `default`
			b.ch.symbolDeclarations(symbol).Len() == 0 ||
			// if not in the same binding context (source file, module declaration), it must print as `default`
			(!b.ctx.enclosingDeclaration.IsNil() && ast.FindAncestor(b.ch.symbolDeclarations(symbol).At(0), isDefaultBindingContext) != ast.FindAncestor(b.ctx.enclosingDeclaration, isDefaultBindingContext))) {
		return "default"
	}
	if b.ch.symbolDeclarations(symbol).Len() > 0 {
		name := core.FirstNonNilSeq(b.ch.symbolDeclarations(symbol).Values(), ast.GetNameOfDeclaration) // Try using a declaration with a name, first
		if !name.IsNil() {
			// !!! TODO: JS Object.defineProperty declarations
			// if ast.IsCallExpression(declaration) && ast.IsBindableObjectDefinePropertyCall(declaration) {
			// 	return symbol.Name
			// }
			if ast.IsComputedPropertyName(name) && b.ch.symbolCheckFlags(symbol)&ast.CheckFlagsLate == 0 {
				if b.ch.valueSymbolLinks.Has(symbol) && b.ch.valueSymbolLinks.Get(symbol).getNameType(b.ch) != nil && b.ch.valueSymbolLinks.Get(symbol).getNameType(b.ch).flags&TypeFlagsStringOrNumberLiteral != 0 {
					result := b.getNameOfSymbolFromNameType(symbol)
					if len(result) > 0 {
						return result
					}
				}
			}
			return scanner.DeclarationNameToString(name)
		}
		declaration := b.ch.symbolDeclarations(symbol).At(0) // Declaration may be nameless, but we'll try anyway
		if !declaration.Parent().IsNil() && declaration.Parent().Kind() == ast.KindVariableDeclaration {
			return scanner.DeclarationNameToString(declaration.Parent().AsVariableDeclaration().Name())
		}
		if ast.IsClassExpression(declaration) || ast.IsFunctionExpression(declaration) || ast.IsArrowFunction(declaration) {
			if b.ctx != nil && !b.ctx.encounteredError && b.ctx.flags&nodebuilder.FlagsAllowAnonymousIdentifier == 0 {
				b.ctx.encounteredError = true
			}
			switch declaration.Kind() {
			case ast.KindClassExpression:
				return "(Anonymous class)"
			case ast.KindFunctionExpression, ast.KindArrowFunction:
				return "(Anonymous function)"
			}
		}
	}
	name := b.getNameOfSymbolFromNameType(symbol)
	if len(name) > 0 {
		return name
	}
	return ast.EscapeInternalSymbolName(b.ch.symbolName(symbol))
}

// The full set of type parameters for a generic class or interface type consists of its outer type parameters plus
// its locally declared type parameters.
func (b *NodeBuilderImpl) getTypeParametersOfClassOrInterface(symbol ast.SymbolId) []TypeId {
	result := make([]TypeId, 0)
	result = append(result, b.ch.getOuterTypeParametersOfClassOrInterface(symbol)...)
	result = append(result, b.ch.getLocalTypeParametersOfClassOrInterfaceOrTypeAlias(symbol)...)
	return result
}

func (b *NodeBuilderImpl) lookupTypeParameterNodes(chain []ast.SymbolId, index int) ast.TypeParameterList {
	debug.Assert(chain != nil && 0 <= index && index < len(chain))
	symbol := chain[index]
	symbolId := ast.GetSymbolSerial(b.ch.symbolOf(symbol))
	if b.ctx.typeParameterSymbolList.Has(symbolId) {
		return ast.TypeParameterList{}
	}
	b.ctx.typeParameterSymbolList.Add(symbolId)

	if b.ctx.flags&nodebuilder.FlagsWriteTypeParametersInQualifiedName != 0 && index < (len(chain)-1) {
		if typeArgumentNodes := b.lookupInstantiatedTypeArgumentNodes(chain, index); !typeArgumentNodes.IsNil() {
			return typeArgumentNodes
		} else {
			typeParameterNodes := b.typeParametersToTypeParameterDeclarations(symbol)
			if len(typeParameterNodes) > 0 {
				return b.f.NewNodeList(typeParameterNodes)
			}
			return ast.TypeParameterList{}
		}
	}

	return ast.TypeParameterList{}
}

// TODO: move `lookupSymbolChain` and co to `symbolaccessibility.go` (but getSpecifierForModuleSymbol uses much context which makes that hard?)
func (b *NodeBuilderImpl) lookupSymbolChain(symbol ast.SymbolId, meaning ast.SymbolFlags, yieldModuleSymbol bool) []ast.SymbolId {
	b.ctx.tracker.TrackSymbol(b.ch.symbolOf(symbol), b.ctx.enclosingDeclaration, meaning)
	return b.lookupSymbolChainWorker(symbol, meaning, yieldModuleSymbol)
}

func (b *NodeBuilderImpl) lookupSymbolChainWorker(symbol ast.SymbolId, meaning ast.SymbolFlags, yieldModuleSymbol bool) []ast.SymbolId {
	// Try to get qualified name if the symbol is not a type parameter and there is an enclosing declaration.
	var chain []ast.SymbolId
	isTypeParameter := b.ch.symbolFlags(symbol)&ast.SymbolFlagsTypeParameter != 0
	if !isTypeParameter && (!b.ctx.enclosingDeclaration.IsNil() || b.ctx.flags&nodebuilder.FlagsUseFullyQualifiedType != 0) && (b.ctx.internalFlags&nodebuilder.InternalFlagsDoNotIncludeSymbolChain == 0) {
		res := b.getSymbolChain(symbol, meaning /*endOfChain*/, true, yieldModuleSymbol)
		chain = res
		debug.Assert(chain != nil)
		debug.Assert(len(chain) > 0)
	} else {
		chain = append(chain, symbol)
	}
	return chain
}

type sortedSymbolNamePair struct {
	sym  ast.SymbolId
	name string
}

/** @param endOfChain Set to false for recursive calls; non-recursive calls should always output something. */
func (b *NodeBuilderImpl) getSymbolChain(symbol ast.SymbolId, meaning ast.SymbolFlags, endOfChain bool, yieldModuleSymbol bool) []ast.SymbolId {
	accessibleSymbolChain := b.ch.getAccessibleSymbolChain(symbol, b.ctx.enclosingDeclaration, meaning, b.ctx.flags&nodebuilder.FlagsUseOnlyExternalAliasing != 0)
	qualifierMeaning := meaning
	if len(accessibleSymbolChain) > 1 {
		qualifierMeaning = getQualifiedLeftMeaning(meaning)
	}
	if len(accessibleSymbolChain) == 0 ||
		b.ch.needsQualification(accessibleSymbolChain[0], b.ctx.enclosingDeclaration, qualifierMeaning) {
		// Go up and add our parent.
		root := symbol
		if len(accessibleSymbolChain) > 0 {
			root = accessibleSymbolChain[0]
		}
		parents := b.ch.getContainersOfSymbol(root, b.ctx.enclosingDeclaration, meaning)
		if len(parents) > 0 {
			parentSpecifiers := core.Map(parents, func(symbol ast.SymbolId) sortedSymbolNamePair {
				if b.ch.symbolDeclarations(symbol).Some(hasNonGlobalAugmentationExternalModuleSymbol) {
					return sortedSymbolNamePair{symbol, b.getSpecifierForModuleSymbol(symbol, core.ResolutionModeNone).specifier.AsString()}
				}
				return sortedSymbolNamePair{symbol, ""}
			})
			slices.SortStableFunc(parentSpecifiers, b.sortByBestName)
			for _, pair := range parentSpecifiers {
				parent := pair.sym
				parentChain := b.getSymbolChain(parent, getQualifiedLeftMeaning(meaning), false, yieldModuleSymbol)
				if len(parentChain) > 0 {
					if !b.ch.symbolExports(parent).IsNil() {
						exported, ok := b.ch.symbolExports(parent).LookupId(b.ch.symbols, ast.InternalSymbolNameExportEquals)
						if ok && b.ch.getSymbolIfSameReference(exported, symbol) != 0 {
							// parentChain root _is_ symbol - symbol is a module export=, so it kinda looks like it's own parent
							// No need to lookup an alias for the symbol in itself
							accessibleSymbolChain = parentChain
							break
						}
					}
					nextSyms := accessibleSymbolChain
					if len(nextSyms) == 0 {
						fallback := b.ch.getAliasForSymbolInContainer(parent, symbol)
						if fallback == 0 {
							fallback = symbol
						}
						nextSyms = append(nextSyms, fallback)
					}
					accessibleSymbolChain = append(parentChain, nextSyms...)
					break
				}
			}
		}
	}
	if len(accessibleSymbolChain) > 0 {
		return accessibleSymbolChain
	}
	if
	// If this is the last part of outputting the symbol, always output. The cases apply only to parent symbols.
	endOfChain ||
		// If a parent symbol is an anonymous type, don't write it.
		(b.ch.symbolFlags(symbol)&(ast.SymbolFlagsTypeLiteral|ast.SymbolFlagsObjectLiteral) == 0) {
		// If a parent symbol is an external module, don't write it. (We prefer just `x` vs `"foo/bar".x`.)
		if !endOfChain && !yieldModuleSymbol && b.ch.symbolDeclarations(symbol).Some(hasNonGlobalAugmentationExternalModuleSymbol) {
			return nil
		}
		return []ast.SymbolId{symbol}
	}
	return nil
}

func (b_ *NodeBuilderImpl) sortByBestName(a sortedSymbolNamePair, b sortedSymbolNamePair) int {
	specifierA := a.name
	specifierB := b.name
	if len(specifierA) > 0 && len(specifierB) > 0 {
		isBRelative := tspath.PathIsRelative(specifierB)
		if tspath.PathIsRelative(specifierA) == isBRelative {
			// Both relative or both non-relative, sort by number of parts
			return modulespecifiers.CountPathComponents(specifierA) - modulespecifiers.CountPathComponents(specifierB)
		}
		if isBRelative {
			// A is non-relative, B is relative: prefer A
			return -1
		}
		// A is relative, B is non-relative: prefer B
		return 1
	}
	return b_.ch.compareSymbols(a.sym, b.sym) // must sort symbols for stable ordering
}

func canHaveModuleSpecifier(node ast.Node) bool {
	if node.IsNil() {
		return false
	}
	switch node.Kind() {
	case ast.KindVariableDeclaration,
		ast.KindBindingElement,
		ast.KindImportDeclaration,
		ast.KindExportDeclaration,
		ast.KindImportEqualsDeclaration,
		ast.KindImportClause,
		ast.KindNamespaceExport,
		ast.KindNamespaceImport,
		ast.KindExportSpecifier,
		ast.KindImportSpecifier,
		ast.KindImportType:
		return true
	}
	return false
}

func TryGetModuleSpecifierFromDeclaration(node ast.Node) ast.Node {
	res := tryGetModuleSpecifierFromDeclarationWorker(node)
	if res.IsNil() || !ast.IsStringLiteral(res) {
		return ast.Node{}
	}
	return res
}

func tryGetModuleSpecifierFromDeclarationWorker(node ast.Node) ast.Node {
	switch node.Kind() {
	case ast.KindVariableDeclaration, ast.KindBindingElement:
		moduleCall := ast.FindAncestor(node.Initializer(), func(node ast.Node) bool {
			return ast.IsRequireCall(node, true /*requireStringLiteralLikeArgument*/) || ast.IsImportCall(node)
		})
		if moduleCall.IsNil() {
			return ast.Node{}
		}
		return moduleCall.ArgumentList().At(0)
	case ast.KindImportDeclaration, ast.KindExportDeclaration, ast.KindJSDocImportTag:
		return node.ModuleSpecifier()
	case ast.KindImportEqualsDeclaration:
		ref := node.AsImportEqualsDeclaration().ModuleReference()
		if ref.Kind() != ast.KindExternalModuleReference {
			return ast.Node{}
		}
		return ref.Expression()
	case ast.KindImportClause:
		if ast.IsImportDeclaration(node.Parent()) {
			return node.Parent().ModuleSpecifier()
		}
		return node.Parent().ModuleSpecifier()
	case ast.KindNamespaceExport:
		return node.Parent().ModuleSpecifier()
	case ast.KindNamespaceImport:
		if ast.IsImportDeclaration(node.Parent().Parent()) {
			return node.Parent().Parent().ModuleSpecifier()
		}
		return node.Parent().Parent().ModuleSpecifier()
	case ast.KindExportSpecifier:
		return node.Parent().Parent().ModuleSpecifier()
	case ast.KindImportSpecifier:
		if ast.IsImportDeclaration(node.Parent().Parent().Parent()) {
			return node.Parent().Parent().Parent().ModuleSpecifier()
		}
		return node.Parent().Parent().Parent().ModuleSpecifier()
	case ast.KindImportType:
		if ast.IsLiteralImportTypeNode(node) {
			return node.AsImportTypeNode().Argument().AsLiteralTypeNode().Literal()
		}
		return ast.Node{}
	default:
		debug.AssertNever(node)
		return ast.Node{}
	}
}

func (b *NodeBuilderImpl) getSpecifierForModuleSymbol(symbol ast.SymbolId, overrideImportMode core.ResolutionMode) moduleSpecifierResult {
	enclosingDeclaration := b.e.MostOriginal(b.ctx.enclosingDeclaration)
	var originalModuleSpecifier ast.Node
	if canHaveModuleSpecifier(enclosingDeclaration) {
		originalModuleSpecifier = TryGetModuleSpecifierFromDeclaration(enclosingDeclaration)
	}
	var originalImportAttributesType *Type
	if !originalModuleSpecifier.IsNil() {
		originalImportAttributesType = b.ch.getImportAttributesTypeForModuleSpecifier(originalModuleSpecifier)
	}

	file := ast.GetDeclarationOfKind(b.ch.symbolOf(symbol), ast.KindSourceFile)
	if file.IsNil() {
		equivalentSymbol := core.FirstNonNilSeq(b.ch.symbolDeclarations(symbol).Values(), func(d ast.Node) ast.SymbolId {
			return b.ch.getFileSymbolIfFileSymbolExportEqualsContainer(d, symbol)
		})
		if equivalentSymbol != 0 {
			file = ast.GetDeclarationOfKind(b.ch.symbolOf(equivalentSymbol), ast.KindSourceFile)
		}
	}

	if file.IsNil() {
		if declaration := b.ch.symbolDeclarations(symbol).Find(ast.IsModuleWithStringLiteralName); !declaration.IsNil() {
			specifier := tspath.ToModuleSpecifier(declaration.Name().Text())
			if originalImportAttributesType != nil && b.moduleSpecifierResolvesToSymbol(specifier, originalImportAttributesType, symbol) {
				return moduleSpecifierResult{specifier: specifier, importAttributesType: originalImportAttributesType}
			}
			return moduleSpecifierResult{specifier: specifier, importAttributesType: b.ch.getTypeOfModuleImportAttributes(symbol)}
		}
		if specifier, ok := ast.TryGetAmbientModuleNameFromSymbolName(b.ch.symbolName(symbol)); ok {
			return moduleSpecifierResult{specifier: tspath.ToModuleSpecifier(specifier)}
		}
	}
	if b.ctx.enclosingFile == nil {
		if specifier, ok := ast.TryGetAmbientModuleNameFromSymbolName(b.ch.symbolName(symbol)); ok {
			return moduleSpecifierResult{specifier: tspath.ToModuleSpecifier(specifier)}
		}
		return moduleSpecifierResult{specifier: ast.GetSourceFileOfModule(b.ch.symbolOf(symbol)).FileName().AsModuleSpecifier()}
	}

	contextFile := b.ctx.enclosingFile
	resolutionMode := overrideImportMode
	if resolutionMode == core.ResolutionModeNone && !originalModuleSpecifier.IsNil() {
		resolutionMode = b.ch.program.GetModeForUsageLocation(contextFile, originalModuleSpecifier)
	} else if resolutionMode == core.ResolutionModeNone && contextFile != nil {
		resolutionMode = b.ch.program.GetDefaultResolutionModeForFile(contextFile)
	}
	cacheKey := module.ModeAwareCacheKey{Name: string(contextFile.PathKey()), Mode: resolutionMode}
	links := b.symbolLinks.Get(symbol)
	if links.specifierCache == nil {
		links.specifierCache = make(module.ModeAwareCache[moduleSpecifierResult])
	}
	result, ok := links.specifierCache[cacheKey]
	if ok {
		return b.moduleSpecifierResultForSymbol(result, originalImportAttributesType, symbol)
	}
	// For declaration bundles, we need to generate absolute paths relative to the common source dir for imports,
	// just like how the declaration emitter does for the ambient module declarations - we can easily accomplish this
	// using the `baseUrl` compiler option (which we would otherwise never use in declaration emit) and a non-relative
	// specifier preference
	host := b.ctx.host
	specifierCompilerOptions := b.ch.compilerOptions
	specifierPref := modulespecifiers.ImportModuleSpecifierPreferenceProjectRelative
	endingPref := modulespecifiers.ImportModuleSpecifierEndingPreferenceNone
	if resolutionMode == core.ResolutionModeESM {
		endingPref = modulespecifiers.ImportModuleSpecifierEndingPreferenceJs
	}
	moduleSpecifiersResult := modulespecifiers.GetModuleSpecifiers(b.ch.symbolOf(symbol), b.ch, specifierCompilerOptions, contextFile, host, modulespecifiers.UserPreferences{
		ImportModuleSpecifierPreference: specifierPref,
		ImportModuleSpecifierEnding:     endingPref,
	}, modulespecifiers.ModuleSpecifierOptions{
		OverrideImportMode: overrideImportMode,
	}, false)
	debug.Assert(len(moduleSpecifiersResult.Specifiers) > 0)
	result = moduleSpecifierResult{specifier: moduleSpecifiersResult.Specifiers[0]}
	if b.ch.symbolId(moduleSpecifiersResult.AmbientModuleSymbol) != 0 {
		result.importAttributesType = b.ch.getTypeOfModuleImportAttributes(b.ch.symbolId(moduleSpecifiersResult.AmbientModuleSymbol))
	}
	links.specifierCache[cacheKey] = result
	return b.moduleSpecifierResultForSymbol(result, originalImportAttributesType, symbol)
}

func (b *NodeBuilderImpl) moduleSpecifierResultForSymbol(result moduleSpecifierResult, importAttributesType *Type, symbol ast.SymbolId) moduleSpecifierResult {
	if importAttributesType != nil && b.moduleSpecifierResolvesToSymbol(result.specifier, importAttributesType, symbol) {
		result.importAttributesType = importAttributesType
	}
	result.specifier = b.rewriteContentMappedModuleSpecifier(result.specifier)
	return result
}

func (b *NodeBuilderImpl) rewriteContentMappedModuleSpecifier(specifier tspath.ModuleSpecifier) tspath.ModuleSpecifier {
	if b.ctx.internalFlags&nodebuilder.InternalFlagsRewriteModuleSpecifiers != 0 && b.ch.compilerOptions.RewriteRelativeImportExtensions.IsTrue() {
		rewrites := b.ctx.host.ContentMapperExtensionRewrites()
		caseSensitivity := b.ctx.host.CaseSensitivity()
		if specifier.ShouldRewriteExtension(rewrites, caseSensitivity) {
			rewritten, _ := specifier.RewriteExtension(rewrites, caseSensitivity)
			return rewritten
		}
	}
	return specifier
}

func (b *NodeBuilderImpl) moduleSpecifierResolvesToSymbol(specifier tspath.ModuleSpecifier, importAttributesType *Type, symbol ast.SymbolId) bool {
	location := b.ctx.enclosingDeclaration
	if location.IsNil() && b.ctx.enclosingFile != nil {
		location = b.ctx.enclosingFile.AsNode()
	}
	if location.IsNil() {
		return false
	}
	resolved := b.ch.resolveExternalModule(location, specifier.AsString(), nil, ast.Node{}, false /*isForAugmentation*/, importAttributesType)
	return resolved != 0 && b.ch.getMergedSymbol(resolved) == b.ch.getMergedSymbol(symbol)
}

func (b *NodeBuilderImpl) createImportAttributesForModuleSpecifier(result moduleSpecifierResult, importModeOverride core.ResolutionMode) ast.Node {
	isEmptyAttributesType := result.importAttributesType == nil || b.ch.isEmptyObjectType(result.importAttributesType)
	if isEmptyAttributesType && importModeOverride == core.ResolutionModeNone {
		return ast.Node{}
	}

	var properties []ast.SymbolId
	if !isEmptyAttributesType {
		properties = slices.Clone(b.ch.getPropertiesOfType(result.importAttributesType))
	}
	slices.SortFunc(properties, func(x, y ast.SymbolId) int {
		return strings.Compare(b.ch.symbolName(x).String(), b.ch.symbolName(y).String())
	})
	attributes := make([]ast.Node, 0, len(properties)+1)
	resolutionMode := ""
	if importModeOverride != core.ResolutionModeNone {
		resolutionMode = core.IfElse(importModeOverride == core.ResolutionModeESM, "import", "require")
		attributes = append(attributes, b.f.NewImportAttribute(b.newStringLiteral("resolution-mode"), b.newStringLiteral(resolutionMode)))
		b.ctx.approximateLength += len("resolution-mode") + len(resolutionMode) + 6 // `"resolution-mode": "value"`
	}
	for _, property := range properties {
		name := b.ch.symbolName(property)
		propertyType := b.ch.getTypeOfSymbol(property)
		if propertyType.flags&TypeFlagsStringLiteral == 0 {
			continue
		}
		value := b.ch.getStringLiteralValue(propertyType)
		var nameNode ast.Node
		if scanner.IsIdentifierText(name.String(), core.LanguageVariantStandard) {
			nameNode = b.f.NewIdentifier(name.String())
		} else {
			nameNode = b.newStringLiteral(name.String())
			b.ctx.approximateLength += 2
		}
		attributes = append(attributes, b.f.NewImportAttribute(nameNode, b.newStringLiteral(value)))
		b.ctx.approximateLength += len(name.String()) + len(value) + 4 // `name: "value"`
	}
	if len(attributes) == 0 {
		return ast.Node{}
	}
	b.ctx.approximateLength += 16 + 2*(len(attributes)-1) // `, { with: { , ... , } }`
	return b.f.NewImportAttributes(ast.KindWithKeyword, b.f.NewNodeList(attributes), false)
}

func (b *NodeBuilderImpl) typeParameterToDeclarationWithConstraint(typeParameter *Type, constraintNode ast.TypeNode) ast.TypeParameterDeclarationNode {
	restoreFlags := b.saveRestoreFlags()
	b.ctx.flags &^= nodebuilder.FlagsWriteTypeParametersInQualifiedName // Avoids potential infinite loop when building for a claimspace with a generic
	modifiers := ast.CreateModifiersFromModifierFlags(b.ch.getTypeParameterModifiers(typeParameter), b.f.NewModifier)
	var modifiersList ast.ModifierList
	if len(modifiers) > 0 {
		modifiersList = b.f.NewModifierList(modifiers)
	}
	name := b.typeParameterToName(typeParameter)
	defaultParameter := b.ch.getDefaultFromTypeParameter(typeParameter)
	var defaultParameterDeclarationNode ast.Node
	if defaultParameter != nil {
		defaultParameterDeclarationNode = b.typeToTypeNode(defaultParameter)
	}
	restoreFlags()
	return b.f.NewTypeParameterDeclaration(
		modifiersList,
		name.AsNode(),
		constraintNode,
		ast.Node{}, // expression
		defaultParameterDeclarationNode,
	)
}

/**
* Unlike the utilities `setTextRange`, this checks if the `location` we're trying to set on `range` is within the
* same file as the active context. If not, the range is not applied. This prevents us from copying ranges across files,
* which will confuse the node printer (as it assumes all node ranges are within the current file).
* Additionally, if `range` _isn't synthetic_, or isn't in the current file, it will _copy_ it to _remove_ its' position
* information.
*
* It also calls `setOriginalNode` to setup a `.original` pointer, since you basically *always* want these in the node builder.
 */
func (b *NodeBuilderImpl) setTextRange(range_ ast.Node, location ast.Node) ast.Node {
	if range_.IsNil() {
		return range_
	}
	if !ast.NodeIsSynthesized(range_) || (range_.Flags()&ast.NodeFlagsSynthesized == 0) || b.ctx.enclosingFile == nil || b.ctx.enclosingFile != ast.GetSourceFileOfNode(b.e.MostOriginal(range_)) {
		original := range_
		range_ = range_.Clone(b.f) // if `range` is synthesized or originates in another file, copy it so it definitely has synthetic positions
		range_.SetLoc(core.NewTextRange(-1, -1))
		if symbol, ok := b.idToSymbol[original]; ok {
			b.idToSymbol[range_] = symbol
		}
	}
	if range_ == location || location.IsNil() {
		return range_
	}
	// Don't overwrite the original node if `range` has an `original` node that points either directly or indirectly to `location`
	original := b.e.Original(range_)
	for !original.IsNil() && original != location {
		original = b.e.Original(original)
	}
	if original.IsNil() {
		b.e.SetOriginalEx(range_, location, true)
	}

	// only set positions if range comes from the same file since copying text across files isn't supported by the emitter
	if b.ctx.enclosingFile != nil && b.ctx.enclosingFile == ast.GetSourceFileOfNode(b.e.MostOriginal(location)) {
		range_.SetLoc(location.Loc())
		return range_
	} else {
		range_.SetLoc(core.NewTextRange(-1, -1))
	}
	return range_
}

func (b *NodeBuilderImpl) typeParameterShadowsOtherTypeParameterInScope(name ast.SymbolName, typeParameter *Type) bool {
	result := b.ch.resolveName(b.ctx.enclosingDeclaration, name, ast.SymbolFlagsType, nil, false, false)
	if result != 0 && b.ch.symbolFlags(result)&ast.SymbolFlagsTypeParameter != 0 {
		return result != b.ch.symbolOfType(typeParameter)
	}
	return false
}

func (b *NodeBuilderImpl) typeParameterToName(typeParameter *Type) ast.Identifier {
	if b.ctx.flags&nodebuilder.FlagsGenerateNamesForShadowedTypeParams != 0 {
		if cached, ok := b.ctx.typeParameterNames.Get(typeParameter.Id()); ok {
			return cached
		}
	}
	result := b.symbolToName(b.ch.symbolOfType(typeParameter), ast.SymbolFlagsType /*expectsIdentifier*/, true)
	if !ast.IsIdentifier(result) {
		return b.f.NewIdentifier("(Missing type parameter)").AsIdentifier()
	}
	if b.ch.symbolOfType(typeParameter) != 0 && b.ch.symbolDeclarations(b.ch.symbolOfType(typeParameter)).Len() > 0 {
		decl := b.ch.symbolDeclarations(b.ch.symbolOfType(typeParameter)).At(0)
		if !decl.IsNil() && ast.IsTypeParameterDeclaration(decl) {
			result = b.setTextRange(result, decl.Name())
		}
	}
	if b.ctx.flags&nodebuilder.FlagsGenerateNamesForShadowedTypeParams != 0 {
		rawName := result.TextSymbolName()
		i, _ := b.ctx.typeParameterNamesByTextNextNameCount.Get(rawName)
		name := rawName

		for true {
			if !b.ctx.typeParameterNamesByText.Has(name) && !b.typeParameterShadowsOtherTypeParameterInScope(name, typeParameter) {
				break
			}
			i++
			name = ast.MakeSymbolName(fmt.Sprintf("%s_%d", rawName.String(), i))
		}

		if name != rawName {
			// !!! TODO: smuggle type arguments out
			// const typeArguments = getIdentifierTypeArguments(result);
			result = b.newIdentifier(name.String(), b.ch.symbolOfType(typeParameter))
			// setIdentifierTypeArguments(result, typeArguments);
		}

		// avoiding iterations of the above loop turns out to be worth it when `i` starts to get large, so we cache the max
		// `i` we've used thus far, to save work later
		b.ctx.typeParameterNamesByTextNextNameCount.Set(rawName, i)
		b.ctx.typeParameterNames.Set(typeParameter.Id(), result.AsIdentifier())
		b.ctx.typeParameterNamesByText.Add(name)
	}

	return result.AsIdentifier()
}

func (b *NodeBuilderImpl) isMappedTypeHomomorphic(mapped *Type) bool {
	return b.ch.getHomomorphicTypeVariable(mapped) != nil
}

func (b *NodeBuilderImpl) isHomomorphicMappedTypeWithNonHomomorphicInstantiation(mapped *MappedType) bool {
	return mapped.target(b.ch) != nil && !b.isMappedTypeHomomorphic(mapped.AsType()) && b.isMappedTypeHomomorphic(mapped.target(b.ch))
}

func (b *NodeBuilderImpl) createMappedTypeNodeFromType(t *Type) ast.TypeNode {
	debug.Assert(t.Flags()&TypeFlagsObject != 0)
	mapped := t.AsMappedType()
	var readonlyToken ast.Node
	if !mapped.mapped(b.ch).declaration.ReadonlyToken().IsNil() {
		readonlyToken = b.f.NewToken(mapped.mapped(b.ch).declaration.ReadonlyToken().Kind())
	}
	var questionToken ast.Node
	if !mapped.mapped(b.ch).declaration.QuestionToken().IsNil() {
		questionToken = b.f.NewToken(mapped.mapped(b.ch).declaration.QuestionToken().Kind())
	}
	var appropriateConstraintTypeNode ast.Node
	var newTypeVariable ast.Node
	templateType := b.ch.getTemplateTypeFromMappedType(t)
	typeParameter := b.ch.getTypeParameterFromMappedType(t)

	// If the mapped type isn't `keyof` constraint-declared, _but_ still has modifiers preserved, and its naive instantiation won't preserve modifiers because its constraint isn't `keyof` constrained, we have work to do
	needsModifierPreservingWrapper := !b.ch.isMappedTypeWithKeyofConstraintDeclaration(t) &&
		b.ch.getModifiersTypeFromMappedType(t).flags&TypeFlagsUnknown == 0 &&
		b.ctx.flags&nodebuilder.FlagsGenerateNamesForShadowedTypeParams != 0 &&
		!(b.ch.getConstraintTypeFromMappedType(t).flags&TypeFlagsTypeParameter != 0 && b.ch.getConstraintOfTypeParameter(b.ch.getConstraintTypeFromMappedType(t)).flags&TypeFlagsIndex != 0)

	if b.ch.isMappedTypeWithKeyofConstraintDeclaration(t) {
		// We have a { [P in keyof T]: X }
		// We do this to ensure we retain the toplevel keyof-ness of the type which may be lost due to keyof distribution during `getConstraintTypeFromMappedType`
		if b.ctx.flags&nodebuilder.FlagsGenerateNamesForShadowedTypeParams != 0 && b.isHomomorphicMappedTypeWithNonHomomorphicInstantiation(mapped) {
			newConstraintParam := b.ch.newTypeParameter(
				b.ch.newSymbol(ast.SymbolFlagsTypeParameter, ast.SymbolNameT),
			)
			name := b.typeParameterToName(newConstraintParam)
			target := t.Target(b.ch)
			newTypeVariable = b.f.NewTypeReferenceNode(name.AsNode(), ast.TypeList{})
			templateType = b.ch.instantiateType(b.ch.getTemplateTypeFromMappedType(target), b.ch.newTypeMapper([]TypeId{b.ch.getTypeParameterFromMappedType(target).ref(), b.ch.getModifiersTypeFromMappedType(target).ref()}, []TypeId{typeParameter.ref(), newConstraintParam.ref()}))
		}
		indexTarget := newTypeVariable
		if indexTarget.IsNil() {
			indexTarget = b.typeToTypeNode(b.ch.getModifiersTypeFromMappedType(t))
		}
		appropriateConstraintTypeNode = b.f.NewTypeOperatorNode(ast.KindKeyOfKeyword, indexTarget)
	} else if needsModifierPreservingWrapper {
		// So, step 1: new type variable
		newParam := b.ch.newTypeParameter(
			b.ch.newSymbol(ast.SymbolFlagsTypeParameter, ast.SymbolNameT),
		)
		name := b.typeParameterToName(newParam)
		newTypeVariable = b.f.NewTypeReferenceNode(name.AsNode(), ast.TypeList{})
		// step 2: make that new type variable itself the constraint node, making the mapped type `{[K in T_1]: Template}`
		appropriateConstraintTypeNode = newTypeVariable
	} else {
		appropriateConstraintTypeNode = b.typeToTypeNode(b.ch.getConstraintTypeFromMappedType(t))
	}

	// nameType and templateType nodes have to be in the new scope
	cleanup := b.enterNewScope(mapped.mapped(b.ch).declaration.AsNode(), nil, []TypeId{b.ch.getTypeParameterFromMappedType(t).ref()}, nil, nil)
	typeParameterDeclarationNode := b.typeParameterToDeclarationWithConstraint(typeParameter, appropriateConstraintTypeNode)
	var nameTypeNode ast.Node
	if !mapped.mapped(b.ch).declaration.NameType().IsNil() {
		nameTypeNode = b.typeToTypeNode(b.ch.getNameTypeFromMappedType(t))
	}
	templateTypeNode := b.typeToTypeNode(b.ch.removeMissingType(
		templateType,
		getMappedTypeModifiers(b.ch, t)&MappedTypeModifiersIncludeOptional != 0,
	))
	cleanup()
	result := b.f.NewMappedTypeNode(
		readonlyToken,
		typeParameterDeclarationNode,
		nameTypeNode,
		questionToken,
		templateTypeNode,
		ast.TypeElementList{},
	)
	b.ctx.approximateLength += 10
	b.e.AddEmitFlags(result, printer.EFSingleLine)

	if b.ctx.flags&nodebuilder.FlagsGenerateNamesForShadowedTypeParams != 0 && b.isHomomorphicMappedTypeWithNonHomomorphicInstantiation(mapped) {
		// homomorphic mapped type with a non-homomorphic naive inlining
		// wrap it with a conditional like `SomeModifiersType extends infer U ? {..the mapped type...} : never` to ensure the resulting
		// type stays homomorphic

		rawConstraintTypeFromDeclaration := b.getTypeFromTypeNode(mapped.mapped(b.ch).declaration.TypeParameter().AsTypeParameterDeclaration().Constraint().Type(), false)
		if rawConstraintTypeFromDeclaration != nil {
			rawConstraintTypeFromDeclaration = b.ch.getConstraintOfTypeParameter(rawConstraintTypeFromDeclaration)
		}
		if rawConstraintTypeFromDeclaration == nil {
			rawConstraintTypeFromDeclaration = b.ch.unknownType
		}
		originalConstraint := b.ch.instantiateType(rawConstraintTypeFromDeclaration, mapped.mapper(b.ch))

		var originalConstraintNode ast.Node
		if originalConstraint.flags&TypeFlagsUnknown == 0 {
			originalConstraintNode = b.typeToTypeNode(originalConstraint)
		}

		return b.f.NewConditionalTypeNode(
			b.typeToTypeNode(b.ch.getModifiersTypeFromMappedType(t)),
			b.f.NewInferTypeNode(b.f.NewTypeParameterDeclaration(ast.ModifierList{}, newTypeVariable.AsTypeReferenceNode().TypeName().Clone(b.f), originalConstraintNode, ast.Node{}, ast.Node{})),
			result,
			b.f.NewKeywordTypeNode(ast.KindNeverKeyword),
		)
	} else if needsModifierPreservingWrapper {
		// and step 3: once the mapped type is reconstructed, create a `ConstraintType extends infer T_1 extends keyof ModifiersType ? {[K in T_1]: Template} : never`
		// subtly different from the `keyof` constraint case, by including the `keyof` constraint on the `infer` type parameter, it doesn't rely on the constraint type being itself
		// constrained to a `keyof` type to preserve its modifier-preserving behavior. This is all basically because we preserve modifiers for a wider set of mapped types than
		// just homomorphic ones.
		return b.f.NewConditionalTypeNode(
			b.typeToTypeNode(b.ch.getConstraintTypeFromMappedType(t)),
			b.f.NewInferTypeNode(b.f.NewTypeParameterDeclaration(ast.ModifierList{}, newTypeVariable.AsTypeReferenceNode().TypeName().Clone(b.f), b.f.NewTypeOperatorNode(ast.KindKeyOfKeyword, b.typeToTypeNode(b.ch.getModifiersTypeFromMappedType(t))), ast.Node{}, ast.Node{})),
			result,
			b.f.NewKeywordTypeNode(ast.KindNeverKeyword),
		)
	}

	return result
}

func (b *NodeBuilderImpl) typePredicateToTypePredicateNode(predicate *TypePredicate) ast.Node {
	var assertsModifier ast.Node
	if predicate.kind == TypePredicateKindAssertsIdentifier || predicate.kind == TypePredicateKindAssertsThis {
		assertsModifier = b.f.NewToken(ast.KindAssertsKeyword)
	}
	var parameterName ast.Node
	if predicate.kind == TypePredicateKindIdentifier || predicate.kind == TypePredicateKindAssertsIdentifier {
		parameterName = b.f.NewIdentifier(predicate.parameterName.String())
		b.e.AddEmitFlags(parameterName, printer.EFNoAsciiEscaping)
	} else {
		parameterName = b.f.NewThisTypeNode()
	}
	var typeNode ast.Node
	if predicate.t != 0 {
		typeNode = b.typeToTypeNode(b.ch.typ(predicate.t))
	}
	return b.f.NewTypePredicateNode(
		assertsModifier,
		parameterName,
		typeNode,
	)
}

func (b *NodeBuilderImpl) typeToTypeNodeHelperWithPossibleReusableTypeNode(t *Type, typeNode ast.TypeNode) ast.TypeNode {
	if t == nil {
		return b.f.NewKeywordTypeNode(ast.KindAnyKeyword)
	}
	if !b.isActivelyExpanding() && !typeNode.IsNil() && b.getTypeFromTypeNode(typeNode, false) == t {
		reused := b.tryReuseExistingNodeHelper(typeNode)
		if !reused.IsNil() {
			b.checkTypeExpandability(t)
			return reused
		}
	}
	return b.typeToTypeNode(t)
}

func (b *NodeBuilderImpl) typeParameterToDeclaration(parameter *Type) ast.Node {
	constraint := b.ch.getConstraintOfTypeParameter(parameter)
	var constraintNode ast.Node
	if constraint != nil {
		constraintNode = b.typeToTypeNodeHelperWithPossibleReusableTypeNode(constraint, b.ch.getConstraintDeclaration(parameter))
	}
	return b.typeParameterToDeclarationWithConstraint(parameter, constraintNode)
}

func (b *NodeBuilderImpl) symbolToTypeParameterDeclarations(symbol ast.SymbolId) []ast.Node {
	return b.typeParametersToTypeParameterDeclarations(symbol)
}

func (b *NodeBuilderImpl) typeParametersToTypeParameterDeclarations(symbol ast.SymbolId) []ast.Node {
	targetSymbol := b.ch.getTargetSymbol(symbol)
	if b.ch.symbolFlags(targetSymbol)&(ast.SymbolFlagsClass|ast.SymbolFlagsInterface|ast.SymbolFlagsAlias) != 0 {
		var results []ast.Node
		params := b.ch.getLocalTypeParametersOfClassOrInterfaceOrTypeAlias(symbol)
		for _, paramId := range params {
			param := b.ch.typ(paramId)
			results = append(results, b.typeParameterToDeclaration(param))
		}
		return results
	} else if b.ch.symbolFlags(targetSymbol)&ast.SymbolFlagsFunction != 0 {
		var results []ast.Node
		for _, paramId := range b.ch.getTypeParametersFromDeclaration(b.ch.symbolValueDeclaration(symbol)) {
			param := b.ch.typ(paramId)
			results = append(results, b.typeParameterToDeclaration(param))
		}
		return results
	}
	return nil
}

func (c *Checker) getEffectiveParameterDeclaration(symbol ast.SymbolId) ast.Node {
	parameterDeclaration := ast.GetDeclarationOfKind(c.symbolOf(symbol), ast.KindParameter)
	if !parameterDeclaration.IsNil() {
		return parameterDeclaration
	}
	if c.symbolFlags(symbol)&ast.SymbolFlagsTransient == 0 {
		return ast.GetDeclarationOfKind(c.symbolOf(symbol), ast.KindJSDocParameterTag)
	}
	return ast.Node{}
}

func (b *NodeBuilderImpl) symbolToParameterDeclaration(parameterSymbol ast.SymbolId, preserveModifierFlags bool) ast.Node {
	parameterDeclaration := b.ch.getEffectiveParameterDeclaration(parameterSymbol)

	parameterType := b.ch.getTypeOfSymbol(parameterSymbol)
	parameterTypeNode := b.serializeTypeForDeclaration(parameterDeclaration, parameterType, parameterSymbol, true)
	var modifiers ast.ModifierList
	if b.ctx.flags&nodebuilder.FlagsOmitParameterModifiers == 0 && preserveModifierFlags && !parameterDeclaration.IsNil() && ast.CanHaveModifiers(parameterDeclaration) {
		originals := core.Filter(parameterDeclaration.ModifierNodes(), ast.IsModifier)
		clones := core.Map(originals, func(node ast.Node) ast.Node { return node.Clone(b.f) })
		if len(clones) > 0 {
			modifiers = b.f.NewModifierList(clones)
		}
	}
	isRest := !parameterDeclaration.IsNil() && isRestParameter(parameterDeclaration) || b.ch.symbolCheckFlags(parameterSymbol)&ast.CheckFlagsRestParameter != 0
	var dotDotDotToken ast.Node
	if isRest {
		dotDotDotToken = b.f.NewToken(ast.KindDotDotDotToken)
	}
	name := b.parameterToParameterDeclarationName(parameterSymbol, parameterDeclaration)
	isOptional := !parameterDeclaration.IsNil() && b.ch.isOptionalParameter(parameterDeclaration) || b.ch.symbolCheckFlags(parameterSymbol)&ast.CheckFlagsOptionalParameter != 0
	var questionToken ast.Node
	if isOptional {
		questionToken = b.f.NewToken(ast.KindQuestionToken)
	}

	parameterNode := b.f.NewParameterDeclaration(
		modifiers,
		dotDotDotToken,
		name,
		questionToken,
		parameterTypeNode,
		/*initializer*/ ast.Node{},
	)
	b.ctx.approximateLength += len(b.ch.symbolName(parameterSymbol).String()) + 3
	return parameterNode
}

func (b *NodeBuilderImpl) parameterToParameterDeclarationName(parameterSymbol ast.SymbolId, parameterDeclaration ast.Node) ast.Node {
	if parameterDeclaration.IsNil() || parameterDeclaration.Name().IsNil() {
		return b.newIdentifier(b.ch.symbolName(parameterSymbol).String(), parameterSymbol)
	}

	name := parameterDeclaration.Name()
	switch name.Kind() {
	case ast.KindIdentifier:
		cloned := b.f.DeepCloneNode(name)
		b.e.SetEmitFlags(cloned, printer.EFNoAsciiEscaping)
		b.idToSymbol[cloned] = b.ch.symbolOf(parameterSymbol)
		return cloned
	case ast.KindQualifiedName:
		cloned := b.f.DeepCloneNode(name.AsQualifiedName().Right())
		b.e.SetEmitFlags(cloned, printer.EFNoAsciiEscaping)
		b.idToSymbol[cloned] = b.ch.symbolOf(parameterSymbol)
		return cloned
	default:
		return b.cloneBindingName(name)
	}
}

func (b *NodeBuilderImpl) cloneBindingName(node ast.Node) ast.Node {
	if ast.IsComputedPropertyName(node) && b.ch.isLateBindableName(node) {
		b.trackComputedName(node.Expression(), b.ctx.enclosingDeclaration)
	}

	visited := b.cloneBindingNameVisitor.VisitEachChild(node)

	if ast.IsBindingElement(visited) {
		bindingElement := visited.AsBindingElement()
		visited = b.f.UpdateBindingElement(
			bindingElement,
			bindingElement.DotDotDotToken(),
			bindingElement.PropertyName(),
			bindingElement.Name(),
			ast.Node{}, // remove initializer
		)
	}

	if !ast.NodeIsSynthesized(visited) {
		visited = b.f.DeepCloneNode(visited)
	}

	b.e.SetEmitFlags(visited, printer.EFSingleLine|printer.EFNoAsciiEscaping)
	return visited
}

func (b *NodeBuilderImpl) serializeTypeForExpression(expr ast.Node) ast.Node {
	// !!! TODO: shim, add node reuse
	t := b.ch.instantiateType(b.ch.getWidenedType(b.ch.getRegularTypeOfExpression(expr)), b.ctx.mapper)
	return b.typeToTypeNode(t)
}

func (b *NodeBuilderImpl) serializeInferredReturnTypeForSignature(signature *Signature, returnType *Type) ast.Node {
	oldSuppressReportInferenceFallback := b.ctx.suppressReportInferenceFallback
	b.ctx.suppressReportInferenceFallback = true
	typePredicate := b.ch.getTypePredicateOfSignature(signature)
	var returnTypeNode ast.Node
	if typePredicate != nil {
		var predicate *TypePredicate
		if b.ctx.mapper != nil {
			predicate = b.ch.instantiateTypePredicate(typePredicate, b.ctx.mapper)
		} else {
			predicate = typePredicate
		}
		returnTypeNode = b.typePredicateToTypePredicateNodeHelper(predicate)
	} else {
		returnTypeNode = b.typeToTypeNode(returnType)
	}
	b.ctx.suppressReportInferenceFallback = oldSuppressReportInferenceFallback
	return returnTypeNode
}

func (b *NodeBuilderImpl) typePredicateToTypePredicateNodeHelper(typePredicate *TypePredicate) ast.Node {
	var assertsModifier ast.Node
	if typePredicate.kind == TypePredicateKindAssertsThis || typePredicate.kind == TypePredicateKindAssertsIdentifier {
		assertsModifier = b.f.NewToken(ast.KindAssertsKeyword)
	} else {
		assertsModifier = ast.Node{}
	}
	var parameterName ast.Node
	if typePredicate.kind == TypePredicateKindIdentifier || typePredicate.kind == TypePredicateKindAssertsIdentifier {
		parameterName = b.newIdentifier(typePredicate.parameterName.String(), 0 /*symbol*/)
		b.e.SetEmitFlags(parameterName, printer.EFNoAsciiEscaping)
	} else {
		parameterName = b.f.NewThisTypeNode()
	}
	var typeNode ast.Node
	if typePredicate.t != 0 {
		typeNode = b.typeToTypeNode(b.ch.typ(typePredicate.t))
	}
	return b.f.NewTypePredicateNode(assertsModifier, parameterName, typeNode)
}

type SignatureToSignatureDeclarationOptions struct {
	modifiers     []ast.Node
	name          ast.PropertyName
	questionToken ast.Node
}

func (b *NodeBuilderImpl) signatureToSignatureDeclarationHelper(signature *Signature, kind ast.Kind, options *SignatureToSignatureDeclarationOptions) ast.Node {
	var typeParameters []ast.Node

	expandedParams, cleanup := b.enterSignatureScope(signature)
	b.ctx.approximateLength += 3
	// Usually a signature contributes a few more characters than this, but 3 is the minimum

	if b.ctx.flags&nodebuilder.FlagsWriteTypeArgumentsOfSignature != 0 && signature.target != 0 && signature.mapper != 0 && len(b.ch.signatureOf(signature.target).TypeParameters(b.ch)) != 0 {
		for _, parameterId := range b.ch.signatureOf(signature.target).TypeParameters(b.ch) {
			parameter := b.ch.typ(parameterId)
			typeParameters = append(typeParameters, b.typeToTypeNode(b.ch.instantiateType(parameter, b.ch.mapperOf(signature.mapper))))
		}
	} else {
		for _, parameterId := range signature.TypeParameters(b.ch) {
			parameter := b.ch.typ(parameterId)
			typeParameters = append(typeParameters, b.typeParameterToDeclaration(parameter))
		}
	}

	restoreFlags := b.saveRestoreFlags()
	b.ctx.flags &^= nodebuilder.FlagsSuppressAnyReturnType
	// If the expanded parameter list had a variadic in a non-trailing position, don't expand it
	parameters := core.Map(core.IfElse(core.Some(expandedParams, func(p ast.SymbolId) bool {
		return p != expandedParams[len(expandedParams)-1] && b.ch.symbolCheckFlags(p)&ast.CheckFlagsRestParameter != 0
	}), signature.Parameters(b.ch), expandedParams), func(parameter ast.SymbolId) ast.Node {
		return b.symbolToParameterDeclaration(parameter, kind == ast.KindConstructor)
	})
	var thisParameter ast.Node
	if b.ctx.flags&nodebuilder.FlagsOmitThisParameter != 0 {
		thisParameter = ast.Node{}
	} else {
		thisParameter = b.tryGetThisParameterDeclaration(signature)
	}
	if !thisParameter.IsNil() {
		parameters = append([]ast.Node{thisParameter}, parameters...)
	}
	restoreFlags()

	returnTypeNode := b.serializeReturnTypeForSignature(signature, true)

	var modifiers []ast.Node
	if options != nil {
		modifiers = options.modifiers
	}
	if (kind == ast.KindConstructorType) && signature.flags&SignatureFlagsAbstract != 0 {
		flags := ast.ModifiersToFlags(modifiers)
		modifiers = ast.CreateModifiersFromModifierFlags(flags|ast.ModifierFlagsAbstract, b.f.NewModifier)
	}

	paramList := b.f.NewNodeList(parameters)
	var typeParamList ast.NodeList
	if len(typeParameters) != 0 {
		typeParamList = b.f.NewNodeList(typeParameters)
	}
	var modifierList ast.ModifierList
	if len(modifiers) > 0 {
		modifierList = b.f.NewModifierList(modifiers)
	}
	var name ast.Node
	if options != nil {
		name = options.name
	}
	if name.IsNil() {
		name = b.f.NewIdentifier("")
	}

	var node ast.Node
	switch {
	case kind == ast.KindCallSignature:
		node = b.f.NewCallSignatureDeclaration(typeParamList, paramList, returnTypeNode)
	case kind == ast.KindConstructSignature:
		node = b.f.NewConstructSignatureDeclaration(typeParamList, paramList, returnTypeNode)
	case kind == ast.KindMethodSignature:
		var questionToken ast.Node
		if options != nil {
			questionToken = options.questionToken
		}
		node = b.f.NewMethodSignatureDeclaration(modifierList, name, questionToken, typeParamList, paramList, returnTypeNode)
	case kind == ast.KindMethodDeclaration:
		node = b.f.NewMethodDeclaration(modifierList, ast.Node{} /*asteriskToken*/, name, ast.Node{} /*questionToken*/, typeParamList, paramList, returnTypeNode, ast.Node{} /*fullSignature*/, ast.Node{} /*body*/)
	case kind == ast.KindConstructor:
		node = b.f.NewConstructorDeclaration(modifierList, ast.TypeParameterList{} /*typeParamList*/, paramList, ast.Node{} /*returnTypeNode*/, ast.Node{} /*fullSignature*/, ast.Node{} /*body*/)
	case kind == ast.KindGetAccessor:
		node = b.f.NewGetAccessorDeclaration(modifierList, name, ast.TypeParameterList{} /*typeParamList*/, paramList, returnTypeNode, ast.Node{} /*fullSignature*/, ast.Node{} /*body*/)
	case kind == ast.KindSetAccessor:
		node = b.f.NewSetAccessorDeclaration(modifierList, name, ast.TypeParameterList{} /*typeParamList*/, paramList, ast.Node{} /*returnTypeNode*/, ast.Node{} /*fullSignature*/, ast.Node{} /*body*/)
	case kind == ast.KindIndexSignature:
		node = b.f.NewIndexSignatureDeclaration(modifierList, paramList, returnTypeNode)
	// !!! JSDoc Support
	// case kind == ast.KindJSDocFunctionType:
	// 	node = b.f.NewJSDocFunctionType(parameters, returnTypeNode)
	case kind == ast.KindFunctionType:
		if returnTypeNode.IsNil() {
			returnTypeNode = b.f.NewTypeReferenceNode(b.f.NewIdentifier(""), ast.TypeList{})
		}
		node = b.f.NewFunctionTypeNode(typeParamList, paramList, returnTypeNode)
	case kind == ast.KindConstructorType:
		if returnTypeNode.IsNil() {
			returnTypeNode = b.f.NewTypeReferenceNode(b.f.NewIdentifier(""), ast.TypeList{})
		}
		node = b.f.NewConstructorTypeNode(modifierList, typeParamList, paramList, returnTypeNode)
	case kind == ast.KindFunctionDeclaration:
		// TODO: assert name is Identifier
		node = b.f.NewFunctionDeclaration(modifierList, ast.Node{} /*asteriskToken*/, name, typeParamList, paramList, returnTypeNode, ast.Node{} /*fullSignature*/, ast.Node{} /*body*/)
	case kind == ast.KindFunctionExpression:
		// TODO: assert name is Identifier
		node = b.f.NewFunctionExpression(modifierList, ast.Node{} /*asteriskToken*/, name, typeParamList, paramList, returnTypeNode, ast.Node{} /*fullSignature*/, b.f.NewBlock(b.f.NewNodeList([]ast.Node{}), false))
	case kind == ast.KindArrowFunction:
		node = b.f.NewArrowFunction(modifierList, typeParamList, paramList, returnTypeNode, ast.Node{} /*fullSignature*/, ast.Node{} /*equalsGreaterThanToken*/, b.f.NewBlock(b.f.NewNodeList([]ast.Node{}), false))
	default:
		panic("Unhandled kind in signatureToSignatureDeclarationHelper")
	}

	// !!! TODO: Smuggle type arguments of signatures out for quickinfo
	// if typeArguments != nil {
	// 	node.TypeArguments = b.f.NewNodeList(typeArguments)
	// }

	cleanup()
	return node
}

func (c *Checker) getExpandedParameters(sig *Signature, skipUnionExpanding bool) [][]ast.SymbolId {
	if signatureHasRestParameter(sig) {
		restIndex := len(sig.Parameters(c)) - 1
		restSymbol := sig.Parameters(c)[restIndex]
		restType := c.getTypeOfSymbol(restSymbol)
		getUniqAssociatedNamesFromTupleType := func(t *Type, restSymbol ast.SymbolId) []ast.SymbolName {
			names := core.MapIndex(t.Target(c).AsTupleType().tuple(c).elementInfos, func(info TupleElementInfo, i int) ast.SymbolName {
				return c.getTupleElementLabel(info, restSymbol, i)
			})
			if len(names) > 0 {
				duplicates := []int{}
				uniqueNames := make(map[ast.SymbolName]bool)
				for i, name := range names {
					_, ok := uniqueNames[name]
					if ok {
						duplicates = append(duplicates, i)
					} else {
						uniqueNames[name] = true
					}
				}
				counters := make(map[ast.SymbolName]int)
				for _, i := range duplicates {
					counter, ok := counters[names[i]]
					if !ok {
						counter = 1
					}
					var name ast.SymbolName
					for true {
						name = ast.MakeSymbolName(fmt.Sprintf("%s_%d", names[i].String(), counter))
						_, ok := uniqueNames[name]
						if ok {
							counter++
							continue
						} else {
							uniqueNames[name] = true
							break
						}
					}
					names[i] = name
					counters[names[i]] = counter + 1
				}
			}
			return names
		}
		expandSignatureParametersWithTupleMembers := func(restType *Type, restIndex int, restSymbol ast.SymbolId) []ast.SymbolId {
			elementTypes := c.getTypeArguments(restType)
			associatedNames := getUniqAssociatedNamesFromTupleType(restType, restSymbol)
			restParams := typesMapIndexTo(c, elementTypes, func(t *Type, i int) ast.SymbolId {
				// Lookup the label from the individual tuple passed in before falling back to the signature `rest` parameter name
				// TODO: getTupleElementLabel can no longer fail, investigate if this lack of falliability meaningfully changes output
				// var name *string
				// if associatedNames != nil && associatedNames[i] != nil {
				// 	name = associatedNames[i]
				// } else {
				// 	name = c.getParameterNameAtPosition(sig, restIndex+i, restType)
				// }
				name := associatedNames[i]
				flags := restType.Target(c).AsTupleType().tuple(c).elementInfos[i].flags
				var checkFlags ast.CheckFlags
				switch {
				case flags&ElementFlagsVariable != 0:
					checkFlags = ast.CheckFlagsRestParameter
				case flags&ElementFlagsOptional != 0:
					checkFlags = ast.CheckFlagsOptionalParameter
				}
				symbol := c.newSymbolEx(ast.SymbolFlagsFunctionScopedVariable, name, checkFlags)
				links := c.valueSymbolLinks.Get(symbol)
				if flags&ElementFlagsRest != 0 {
					links.resolvedType = c.createArrayType(t).ref()
				} else {
					links.resolvedType = t.ref()
				}
				return symbol
			})
			return core.Concatenate(sig.Parameters(c)[0:restIndex], restParams)
		}

		if c.isTupleType(restType) {
			return [][]ast.SymbolId{expandSignatureParametersWithTupleMembers(restType, restIndex, restSymbol)}
		} else if !skipUnionExpanding && restType.flags&TypeFlagsUnion != 0 && c.typesEvery(restType.AsUnionType().types(c), c.isTupleType) {
			return typesMapTo(c, restType.AsUnionType().types(c), func(t *Type) []ast.SymbolId {
				return expandSignatureParametersWithTupleMembers(t, restIndex, restSymbol)
			})
		}
	}
	return [][]ast.SymbolId{sig.Parameters(c)}
}

func (b *NodeBuilderImpl) tryGetThisParameterDeclaration(signature *Signature) ast.Node {
	if signature.ThisParameter(b.ch) != 0 {
		return b.symbolToParameterDeclaration(signature.ThisParameter(b.ch), false)
	}
	if !signature.declaration.IsNil() && ast.IsInJSFile(b.ch.nodeOf(signature.declaration)) {
		// !!! JSDoc Support
		// thisTag := getJSDocThisTag(signature.declaration)
		// if (thisTag && thisTag.typeExpression) {
		// 	return factory.createParameterDeclaration(
		// 		/*modifiers*/ undefined,
		// 		/*dotDotDotToken*/ undefined,
		// 		"this",
		// 		/*questionToken*/ undefined,
		// 		typeToTypeNodeHelper(getTypeFromTypeNode(context, thisTag.typeExpression), context),
		// 	);
		// }
	}
	return ast.Node{}
}

/**
* Serializes the return type of the signature by first trying to use the syntactic printer if possible and falling back to the checker type if not.
 */
func (b *NodeBuilderImpl) serializeReturnTypeForSignature(signature *Signature, tryReuse bool) ast.Node {
	suppressAny := b.ctx.flags&nodebuilder.FlagsSuppressAnyReturnType != 0
	restoreFlags := b.saveRestoreFlags()
	if suppressAny {
		b.ctx.flags &^= nodebuilder.FlagsSuppressAnyReturnType // suppress only toplevel `any`s
	}
	var returnTypeNode ast.Node

	var returnType *Type
	if !signature.declaration.IsNil() && !ast.NodeIsSynthesized(b.ch.nodeOf(signature.declaration)) {
		symbol := b.ch.getSymbolOfDeclaration(b.ch.nodeOf(signature.declaration))
		var ok bool
		returnType, ok = b.ctx.enclosingSymbolTypes[ast.GetSymbolSerial(b.ch.symbolOf(symbol))]
		if !ok || returnType == nil {
			returnType = b.ch.instantiateType(b.ch.getReturnTypeOfSignature(signature), b.ctx.mapper)
		}
	} else {
		returnType = b.ch.getReturnTypeOfSignature(signature)
	}
	if !(suppressAny && IsTypeAny(returnType)) {
		if !b.isActivelyExpanding() && tryReuse && !b.ctx.enclosingDeclaration.IsNil() && !signature.declaration.IsNil() && !ast.NodeIsSynthesized(b.ch.nodeOf(signature.declaration)) {
			declarationSymbol := b.ch.getSymbolOfDeclaration(b.ch.nodeOf(signature.declaration))
			restore := b.addSymbolTypeToContext(declarationSymbol, returnType)
			pt := b.pc.GetReturnTypeOfSignature(b.ch.nodeOf(signature.declaration))
			if b.pseudoTypeEquivalentToType(pt, returnType, false, !b.ctx.suppressReportInferenceFallback) {
				// Also verify the pseudo type captures any inferred type predicate, not just the boolean return type.
				// The pseudochecker is unaware of inferred type predicates, so it produces boolean where
				// the checker infers e.g. `x is string`.
				typePredicate := b.ch.getTypePredicateOfSignature(signature)
				if typePredicate != nil && !b.pseudoReturnTypeMatchesPredicate(pt, typePredicate) {
					if !b.ctx.suppressReportInferenceFallback {
						b.ctx.tracker.ReportInferenceFallback(b.ch.nodeOf(signature.declaration))
					}
					pt = nil
				}
				if pt != nil {
					// !!! TODO: If annotated type node is a reference with insufficient type arguments, we should still fall back to type serialization
					// see: canReuseTypeNodeAnnotation in strada for context
					returnTypeNode = b.pseudoTypeToNodeWithCheckerFallback(pt, returnType)
				}
			}
			restore()
		}
		if returnTypeNode.IsNil() {
			returnTypeNode = b.serializeInferredReturnTypeForSignature(signature, returnType)
		}
	}

	if returnTypeNode.IsNil() && !suppressAny {
		returnTypeNode = b.f.NewKeywordTypeNode(ast.KindAnyKeyword)
	}
	restoreFlags()
	return returnTypeNode
}

func (b *NodeBuilderImpl) isTriviallySerializableComputedName(e ast.Node) bool {
	shapeGood := !e.IsNil() && !e.Name().IsNil() && ast.IsComputedPropertyName(e.Name()) && ast.IsEntityNameExpression(e.Name().Expression())
	if !shapeGood {
		return false
	}
	return b.ch.isEntityNameVisible(e.Name().Expression(), b.ctx.enclosingDeclaration, false).Accessibility == printer.SymbolAccessibilityAccessible
}

func (b *NodeBuilderImpl) indexInfoToObjectComputedNamesOrSignatureDeclaration(indexInfo *IndexInfo, typeNode ast.TypeNode) []ast.Node {
	if len(indexInfo.components) > 0 {
		// Index info is derived from object or class computed property names (plus explicit named members) - we can clone those instead of writing out the result computed index signature
		allComponentComputedNamesSerializable := !b.ctx.enclosingDeclaration.IsNil() && core.Every(indexInfo.components, b.isTriviallySerializableComputedName)
		if allComponentComputedNamesSerializable {
			// Only use computed name serialization form if all components are visible and take the `a.b.c` form
			newComponents := core.Filter(indexInfo.components, func(c ast.Node) bool {
				// skip late bound props that contribute to the index signature - they'll be created by property creation anyway
				return !b.ch.hasLateBindableName(c)
			})
			bailed := false
			results := core.Map(newComponents, func(e ast.Node) ast.Node {
				name := b.reuseNode(e.Name())
				if !name.IsNil() {
					// Still need to track visibility even if we've already checked it to paint references as used
					b.trackComputedName(e.Name().Expression(), b.ctx.enclosingDeclaration)
					var mods ast.ModifierList
					if indexInfo.isReadonly {
						mods = b.f.NewModifierList([]ast.Node{b.f.NewModifier(ast.KindReadonlyKeyword)})
					}
					var postfixToken ast.Node
					if !e.PostfixToken().IsNil() {
						postfixToken = e.PostfixToken().Clone(b.f)
					}
					var currentTypeNode ast.TypeNode
					if !typeNode.IsNil() {
						currentTypeNode = b.f.DeepCloneNode(typeNode)
					} else {
						currentTypeNode = b.typeToTypeNode(b.ch.getTypeOfSymbol(b.ch.nodeSymbol(e)))
					}
					sig := b.f.NewPropertySignatureDeclaration(
						mods,
						name,
						postfixToken,
						currentTypeNode,
						ast.Node{},
					)
					sig.SetLoc(e.Loc())
					return sig
				}
				bailed = true
				return ast.Node{}
			})
			if !bailed {
				return results
			}
		}
	}
	return []ast.Node{b.indexInfoToIndexSignatureDeclarationHelper(indexInfo, typeNode)}
}

func (b *NodeBuilderImpl) indexInfoToIndexSignatureDeclarationHelper(indexInfo *IndexInfo, typeNode ast.TypeNode) ast.Node {
	name := getNameFromIndexInfo(indexInfo)
	indexerTypeNode := b.typeToTypeNode(b.ch.typ(indexInfo.keyType))

	indexingParameter := b.f.NewParameterDeclaration(ast.ModifierList{}, ast.Node{}, b.newIdentifier(name, 0 /*symbol*/), ast.Node{}, indexerTypeNode, ast.Node{})
	if typeNode.IsNil() {
		if indexInfo.valueType == 0 {
			typeNode = b.f.NewKeywordTypeNode(ast.KindAnyKeyword)
		} else {
			typeNode = b.typeToTypeNode(b.ch.typ(indexInfo.valueType))
		}
	}
	if indexInfo.valueType == 0 && b.ctx.flags&nodebuilder.FlagsAllowEmptyIndexInfoType == 0 {
		b.ctx.encounteredError = true
	}
	b.ctx.approximateLength += len(name) + 4
	var modifiers ast.ModifierList
	if indexInfo.isReadonly {
		b.ctx.approximateLength += 9
		modifiers = b.f.NewModifierList([]ast.Node{b.f.NewModifier(ast.KindReadonlyKeyword)})
	}
	return b.f.NewIndexSignatureDeclaration(modifiers, b.f.NewNodeList([]ast.Node{indexingParameter}), typeNode)
}

func hasTypeAnnotation(declaration ast.Declaration) bool {
	if declaration.IsNil() || declaration.Type().IsNil() {
		return false
	}
	// Type alias declarations have a .Type() that is their type definition, not a type annotation on a value.
	// Exclude them so callers don't mistake them for annotated value declarations.
	if ast.IsTypeAliasDeclaration(declaration) || ast.IsJSTypeAliasDeclaration(declaration) {
		return false
	}
	return true
}

/**
* Unlike `typeToTypeNodeHelper`, this handles setting up the `AllowUniqueESSymbolType` flag
* so a `unique symbol` is returned when appropriate for the input symbol, rather than `typeof sym`
* @param declaration - The preferred declaration to pull existing type nodes from (the symbol will be used as a fallback to find any annotated declaration)
* @param type - The type to write; an existing annotation must match this type if it's used, otherwise this is the type serialized as a new type node
* @param symbol - The symbol is used both to find an existing annotation if declaration is not provided, and to determine if `unique symbol` should be printed
 */
func (b *NodeBuilderImpl) serializeTypeForDeclaration(declaration ast.Declaration, t *Type, symbol ast.SymbolId, tryReuse bool) ast.Node {
	if declaration.IsNil() {
		if symbol != 0 {
			declaration = b.ch.symbolValueDeclaration(symbol)
			if declaration.IsNil() {
				// TODO: prefer annotated declarations like in strada (but does this ever even matter in practice? All callers should supply a declaration!)
				declaration = b.ch.symbolDeclarations(symbol).FirstOrNil()
			}
		}
	}
	if symbol == 0 {
		symbol = b.ch.getSymbolOfDeclaration(declaration)
	}
	if t == nil {
		if symbol == 0 {
			if ast.IsVariableLike(declaration) {
				t = b.ch.getTypeForVariableLikeDeclaration(declaration, false, CheckModeNormal)
			} else {
				t = b.ch.errorType
			}
		} else {
			t = b.ctx.enclosingSymbolTypes[ast.GetSymbolSerial(b.ch.symbolOf(symbol))]
			if t == nil {
				if b.ch.symbolFlags(symbol)&ast.SymbolFlagsAccessor != 0 && declaration.Kind() == ast.KindSetAccessor {
					t = b.ch.instantiateType(b.ch.getWriteTypeOfSymbol(symbol), b.ctx.mapper)
				} else if symbol != 0 && (b.ch.symbolFlags(symbol)&(ast.SymbolFlagsTypeLiteral|ast.SymbolFlagsSignature) == 0) {
					t = b.ch.instantiateType(b.ch.getWidenedLiteralType(b.ch.getTypeOfSymbol(symbol)), b.ctx.mapper)
				} else {
					t = b.ch.errorType
				}
			}
		}
	}

	requiresAddingUndefined := !declaration.IsNil() && (ast.IsParameterDeclaration(declaration) || ast.IsPropertySignatureDeclaration(declaration) || ast.IsPropertyDeclaration(declaration)) && b.ch.requiresAddingImplicitUndefined(declaration, symbol, b.ctx.enclosingDeclaration)
	addUndefinedForParameter := requiresAddingUndefined && (ast.IsParameterDeclaration(declaration) /*|| ast.IsJSDocParameterTag(declaration)*/)
	if addUndefinedForParameter {
		t = b.ch.getOptionalType(t, false)
	}

	restoreFlags := b.saveRestoreFlags()
	if t.flags&TypeFlagsUniqueESSymbol != 0 && b.ch.symbolOfType(t) == symbol && (b.ctx.enclosingDeclaration.IsNil() || b.ch.symbolDeclarations(symbol).Some(func(d ast.Declaration) bool {
		return ast.GetSourceFileOfNode(d) == b.ctx.enclosingFile
	})) {
		b.ctx.flags |= nodebuilder.FlagsAllowUniqueESSymbolType
	}
	var result ast.Node
	var reportedInferenceFallback bool
	// !!! expandable hover support
	if !b.isActivelyExpanding() && tryReuse && !b.ctx.enclosingDeclaration.IsNil() && !declaration.IsNil() && (ast.IsAccessor(declaration) || (ast.HasInferredType(declaration) && !ast.NodeIsSynthesized(declaration) && (t.ObjectFlags()&ObjectFlagsRequiresWidening) == 0)) {
		var remove func()
		if symbol != 0 {
			remove = b.addSymbolTypeToContext(symbol, t)
		}
		var pt *pseudochecker.PseudoType
		if ast.IsAccessor(declaration) {
			pt = b.pc.GetTypeOfAccessor(declaration)
		} else {
			pt = b.pc.GetTypeOfDeclaration(declaration)
		}
		if (pt == nil || pt.Kind == pseudochecker.PseudoTypeKindNoResult) && ast.IsBinaryExpression(declaration) && symbol != 0 {
			if decl := b.ch.symbolDeclarations(symbol).Find(hasTypeAnnotation); !decl.IsNil() {
				// Binary expressions have a first-in-wins type annotation system. The first one with an annotation supplies the type for the rest.
				pt = b.pc.GetTypeOfDeclaration(decl)
			}
		}
		reportErrors := !b.ctx.suppressReportInferenceFallback
		if b.pseudoTypeEquivalentToType(pt, t, !requiresAddingUndefined && (ast.IsParameterDeclaration(declaration) || ast.IsPropertySignatureDeclaration(declaration) || ast.IsPropertyDeclaration(declaration)) && isOptionalDeclaration(declaration), reportErrors) {
			// !!! TODO: If annotated type node is a reference with insufficient type arguments, we should still fall back to type serialization
			// see: canReuseTypeNodeAnnotation in strada for context
			ptt := b.pseudoTypeToType(pt)
			if ptt != nil && requiresAddingUndefined && containsNonMissingUndefinedType(b.ch, t) && !containsNonMissingUndefinedType(b.ch, ptt) {
				pt = pseudochecker.NewPseudoTypeUnion([]*pseudochecker.PseudoType{pt, pseudochecker.PseudoTypeUndefined})
			}
			result = b.pseudoTypeToNodeWithCheckerFallback(pt, t)
		} else {
			// Equivalence failed; if errors from inferred-with-errors pseudo types were
			// reported, note it so we can suppress nested errors during the fallback
			// typeToTypeNode serialization (mirroring the suppression that
			// pseudoTypeToNodeWithCheckerFallback provides).
			reportedInferenceFallback = reportErrors && pt.Kind == pseudochecker.PseudoTypeKindInferred && len(pt.AsPseudoTypeInferred().ErrorNodes) > 0
			shouldAddUndefined := false
			if requiresAddingUndefined {
				if ptt := b.pseudoTypeToType(pt); ptt != nil {
					shouldAddUndefined = !containsNonMissingUndefinedType(b.ch, ptt)
				} else {
					shouldAddUndefined = !pseudochecker.CouldAlreadyReferToUndefinedType(pt)
				}
			}
			if shouldAddUndefined {
				pt = pseudochecker.NewPseudoTypeUnion([]*pseudochecker.PseudoType{pt, pseudochecker.PseudoTypeUndefined})
				if b.pseudoTypeEquivalentToType(pt, t, false, reportErrors) {
					result = b.pseudoTypeToNodeWithCheckerFallback(pt, t)
					reportedInferenceFallback = false
				}
			}
		}
		if remove != nil {
			remove()
		}
	}
	if result.IsNil() {
		if reportedInferenceFallback {
			oldSuppress := b.ctx.suppressReportInferenceFallback
			b.ctx.suppressReportInferenceFallback = true
			result = b.typeToTypeNode(t)
			b.ctx.suppressReportInferenceFallback = oldSuppress
		} else {
			result = b.typeToTypeNode(t)
		}
	}
	restoreFlags()
	if result.IsNil() {
		return b.f.NewKeywordTypeNode(ast.KindAnyKeyword)
	}
	return result
}

const MAX_REVERSE_MAPPED_NESTING_INSPECTION_DEPTH = 3

func (b *NodeBuilderImpl) shouldUsePlaceholderForProperty(propertySymbol ast.SymbolId) bool {
	// Reverse mapped type placeholders are for display, not declaration emit.
	if b.ctx.flags&nodebuilder.FlagsAllowAnonymousIdentifier == 0 {
		return false
	}
	// Use placeholders for reverse mapped types we've either
	// (1) already descended into, or
	// (2) are nested reverse mappings within a mapping over a non-anonymous type, or
	// (3) are deeply nested properties that originate from the same mapped type.
	// Condition (2) is a restriction mostly just to
	// reduce the blowup in printback size from doing, eg, a deep reverse mapping over `Window`.
	// Since anonymous types usually come from expressions, this allows us to preserve the output
	// for deep mappings which likely come from expressions, while truncating those parts which
	// come from mappings over library functions.
	// Condition (3) limits printing of possibly infinitely deep reverse mapped types.
	if b.ch.symbolCheckFlags(propertySymbol)&ast.CheckFlagsReverseMapped == 0 {
		return false
	}
	// (1)
	if slices.Contains(b.ctx.reverseMappedStack, propertySymbol) {
		return true
	}
	// (2)
	if len(b.ctx.reverseMappedStack) > 0 {
		last := b.ctx.reverseMappedStack[len(b.ctx.reverseMappedStack)-1]
		if b.ch.reverseMappedSymbolLinks.Has(last) {
			links := b.ch.reverseMappedSymbolLinks.TryGet(last)
			propertyType := b.ch.typ(links.propertyType)
			if propertyType != nil && propertyType.objectFlags&ObjectFlagsAnonymous == 0 {
				return true
			}
		}
	}
	// (3) - we only inspect the last MAX_REVERSE_MAPPED_NESTING_INSPECTION_DEPTH elements of the
	// stack for approximate matches to catch tight infinite loops
	// TODO: Why? Reasoning lost to time. this could probably stand to be improved?
	if len(b.ctx.reverseMappedStack) < MAX_REVERSE_MAPPED_NESTING_INSPECTION_DEPTH {
		return false
	}
	if !b.ch.reverseMappedSymbolLinks.Has(propertySymbol) {
		return false
	}
	propertyLinks := b.ch.reverseMappedSymbolLinks.TryGet(propertySymbol)
	propMappedType := b.ch.typ(propertyLinks.mappedType)
	if propMappedType == nil || b.ch.symbolOfType(propMappedType) == 0 {
		return false
	}
	for i := range b.ctx.reverseMappedStack {
		if i > MAX_REVERSE_MAPPED_NESTING_INSPECTION_DEPTH {
			break
		}
		prop := b.ctx.reverseMappedStack[len(b.ctx.reverseMappedStack)-1-i]
		if b.ch.reverseMappedSymbolLinks.Has(prop) {
			links := b.ch.reverseMappedSymbolLinks.TryGet(prop)
			mappedType := b.ch.typ(links.mappedType)
			if mappedType != nil && b.ch.symbolOfType(mappedType) == b.ch.symbolOfType(propMappedType) {
				return true
			}
		}
	}
	return false
}

func (b *NodeBuilderImpl) trackComputedName(accessExpression ast.Node, enclosingDeclaration ast.Node) {
	// get symbol of the first identifier of the entityName
	firstIdentifier := ast.GetFirstIdentifier(accessExpression)
	name := b.ch.resolveName(enclosingDeclaration, firstIdentifier.TextSymbolName(), ast.SymbolFlagsValue|ast.SymbolFlagsExportValue, nil /*nameNotFoundMessage*/, true /*isUse*/, false)
	if name != 0 {
		b.ctx.tracker.TrackSymbol(b.ch.symbolOf(name), enclosingDeclaration, ast.SymbolFlagsValue)
	} else {
		// Name does not resolve at target location, track symbol at dest location (should be inaccessible)
		fallback := b.ch.resolveName(firstIdentifier, firstIdentifier.TextSymbolName(), ast.SymbolFlagsValue|ast.SymbolFlagsExportValue, nil /*nameNotFoundMessage*/, true /*isUse*/, false)
		if fallback != 0 {
			b.ctx.tracker.TrackSymbol(b.ch.symbolOf(fallback), enclosingDeclaration, ast.SymbolFlagsValue)
		}
	}
}

type propertyNameNodeKind int

const (
	propertyNameNodeKindIdentifier propertyNameNodeKind = iota
	propertyNameNodeKindNumericLiteral
	propertyNameNodeKindStringLiteral
)

func classifyPropertyName(name string, stringNamed bool, isMethod bool) propertyNameNodeKind {
	if isMethod && name == "new" {
		return propertyNameNodeKindStringLiteral
	}
	if scanner.IsIdentifierText(name, core.LanguageVariantStandard) {
		return propertyNameNodeKindIdentifier
	}
	return core.IfElse(!stringNamed && isNumericLiteralName(name) && jsnum.FromString(name) >= 0, propertyNameNodeKindNumericLiteral, propertyNameNodeKindStringLiteral)
}

func (b *NodeBuilderImpl) createPropertyNameNodeForIdentifierOrLiteral(name string, singleQuote bool, stringNamed bool, isMethod bool, symbol ast.SymbolId) ast.Node {
	switch classifyPropertyName(name, stringNamed, isMethod) {
	case propertyNameNodeKindIdentifier:
		return b.newIdentifier(name, symbol)
	case propertyNameNodeKindNumericLiteral:
		return b.f.NewNumericLiteral(name, ast.TokenFlagsNone)
	default:
		return b.f.NewStringLiteral(name, core.IfElse(singleQuote, ast.TokenFlagsSingleQuote, ast.TokenFlagsNone))
	}
}

func (b *NodeBuilderImpl) isStringNamed(d ast.Declaration) bool {
	name := ast.GetNameOfDeclaration(d)
	if name.IsNil() {
		return false
	}
	if ast.IsComputedPropertyName(name) {
		t := b.ch.checkExpression(name.Expression())
		return t.flags&TypeFlagsStringLike != 0
	}
	if ast.IsElementAccessExpression(name) {
		t := b.ch.checkExpression(name.AsElementAccessExpression().ArgumentExpression())
		return t.flags&TypeFlagsStringLike != 0
	}
	return ast.IsStringLiteral(name)
}

func (b *NodeBuilderImpl) isSingleQuotedStringNamed(d ast.Declaration) bool {
	name := ast.GetNameOfDeclaration(d)
	return !name.IsNil() && ast.IsStringLiteral(name) && name.AsStringLiteral().TokenFlags()&ast.TokenFlagsSingleQuote != 0
}

func (b *NodeBuilderImpl) getPropertyNameNodeForSymbol(symbol ast.SymbolId, enclosingDeclaration ast.Node) ast.Node {
	// For hash-private names, clone the original private identifier from the declaration
	if !b.ch.symbolValueDeclaration(symbol).IsNil() {
		declName := b.ch.symbolValueDeclaration(symbol).Name()
		if !declName.IsNil() && ast.IsPrivateIdentifier(declName) {
			return b.f.DeepCloneNode(declName)
		}
	}
	stringNamed := b.ch.symbolDeclarations(symbol).Len() != 0 && b.ch.symbolDeclarations(symbol).Every(b.isStringNamed)
	singleQuote := b.ch.symbolDeclarations(symbol).Len() != 0 && b.ch.symbolDeclarations(symbol).Every(b.isSingleQuotedStringNamed)
	isMethod := b.ch.symbolFlags(symbol)&ast.SymbolFlagsMethod != 0
	fromNameType := b.getPropertyNameNodeForSymbolFromNameType(symbol, enclosingDeclaration, singleQuote, stringNamed, isMethod)
	if !fromNameType.IsNil() {
		return fromNameType
	}

	var name string
	if description, ok := b.ch.symbolName(symbol).PrivateNameDescription(); ok {
		// symbol IDs are unstable - replace #nnn# with #private#
		name = "__#private@" + description
	} else {
		name = b.ch.symbolName(symbol).String()
	}

	return b.createPropertyNameNodeForIdentifierOrLiteral(name, singleQuote, stringNamed, isMethod, symbol)
}

// See getNameForSymbolFromNameType for a stringy equivalent
func (b *NodeBuilderImpl) getPropertyNameNodeForSymbolFromNameType(symbol ast.SymbolId, enclosingDeclaration ast.Node, singleQuote bool, stringNamed bool, isMethod bool) ast.Node {
	if !b.ch.valueSymbolLinks.Has(symbol) {
		return ast.Node{}
	}
	nameType := b.ch.valueSymbolLinks.TryGet(symbol).getNameType(b.ch)
	if nameType == nil {
		return ast.Node{}
	}
	enumEnclosingDeclaration := enclosingDeclaration
	if enumEnclosingDeclaration.IsNil() && b.ctx.enclosingFile != nil {
		enumEnclosingDeclaration = b.ctx.enclosingFile.AsNode()
	}
	if nameType.flags&TypeFlagsEnumLiteral != 0 {
		enumSymbol := b.ch.symbolParent(b.ch.symbolOfType(nameType))
		if enumSymbol == 0 {
			enumSymbol = b.ch.symbolOfType(nameType)
		}
		if !enumEnclosingDeclaration.IsNil() &&
			b.ch.isSymbolAccessibleByFlags(enumSymbol, enumEnclosingDeclaration, ast.SymbolFlagsValue) {
			saveEnclosingDeclaration := b.ctx.enclosingDeclaration
			b.ctx.enclosingDeclaration = enumEnclosingDeclaration
			result := b.f.NewComputedPropertyName(b.symbolToExpression(b.ch.symbolOfType(nameType), ast.SymbolFlagsValue))
			b.ctx.enclosingDeclaration = saveEnclosingDeclaration
			return result
		}
	}
	if nameType.flags&TypeFlagsStringOrNumberLiteral != 0 {
		var name string
		switch nameType.AsLiteralType().Value(b.ch).(type) {
		case jsnum.Number:
			name = nameType.AsLiteralType().Value(b.ch).(jsnum.Number).String()
		case string:
			name = nameType.AsLiteralType().Value(b.ch).(string)
		}
		if !scanner.IsIdentifierText(name, core.LanguageVariantStandard) && (stringNamed || !isNumericLiteralName(name)) {
			node := b.f.NewStringLiteral(name, core.IfElse(singleQuote, ast.TokenFlagsSingleQuote, ast.TokenFlagsNone))
			return node
		}
		if isNumericLiteralName(name) && name[0] == '-' {
			return b.f.NewComputedPropertyName(b.f.NewPrefixUnaryExpression(ast.KindMinusToken, b.f.NewNumericLiteral(name[1:], ast.TokenFlagsNone)))
		}
		return b.createPropertyNameNodeForIdentifierOrLiteral(name, singleQuote, stringNamed, isMethod, symbol)
	}
	if nameType.flags&TypeFlagsUniqueESSymbol != 0 {
		// The reference was tracked in the destination scope by trackComputedName.
		// Reconstructing its spelling in the source scope must not paint that scope's declarations visible.
		return b.f.NewComputedPropertyName(b.symbolToExpressionWorker(b.ch.symbolOfType(nameType), ast.SymbolFlagsValue))
	}
	return ast.Node{}
}

func (b *NodeBuilderImpl) addPropertyToElementList(propertySymbol ast.SymbolId, typeElements []ast.TypeElement) []ast.TypeElement {
	propertyIsReverseMapped := b.ch.symbolCheckFlags(propertySymbol)&ast.CheckFlagsReverseMapped != 0
	var propertyType *Type
	if b.shouldUsePlaceholderForProperty(propertySymbol) {
		propertyType = b.ch.anyType
	} else {
		propertyType = b.ch.getNonMissingTypeOfSymbol(propertySymbol)
	}
	saveEnclosingDeclaration := b.ctx.enclosingDeclaration
	b.ctx.enclosingDeclaration = ast.Node{}
	if b.ch.symbolName(propertySymbol).IsLateBoundName() {
		if b.ch.symbolDeclarations(propertySymbol).Len() > 0 {
			decl := b.ch.symbolDeclarations(propertySymbol).At(0)
			if b.ch.hasLateBindableName(decl) {
				if ast.IsBinaryExpression(decl) {
					name := ast.GetNameOfDeclaration(decl)
					if !name.IsNil() && ast.IsElementAccessExpression(name) && ast.IsPropertyAccessEntityNameExpression(name.AsElementAccessExpression().ArgumentExpression(), false /*allowJs*/) {
						b.trackComputedName(name.AsElementAccessExpression().ArgumentExpression(), saveEnclosingDeclaration)
					}
				} else {
					b.trackComputedName(decl.Name().Expression(), saveEnclosingDeclaration)
				}
			}
		} else {
			b.ctx.tracker.ReportNonSerializableProperty(b.ch.symbolToString(propertySymbol))
		}
	}
	if !b.ch.symbolValueDeclaration(propertySymbol).IsNil() {
		b.ctx.enclosingDeclaration = b.ch.symbolValueDeclaration(propertySymbol)
	} else if b.ch.symbolDeclarations(propertySymbol).Len() > 0 && !b.ch.symbolDeclarations(propertySymbol).At(0).IsNil() {
		b.ctx.enclosingDeclaration = b.ch.symbolDeclarations(propertySymbol).At(0)
	} else {
		b.ctx.enclosingDeclaration = saveEnclosingDeclaration
	}
	propertyName := b.getPropertyNameNodeForSymbol(propertySymbol, saveEnclosingDeclaration)
	b.ctx.enclosingDeclaration = saveEnclosingDeclaration
	b.ctx.approximateLength += len(ast.SymbolNameText(b.ch.symbolOf(propertySymbol))) + 1

	if b.ch.symbolFlags(propertySymbol)&ast.SymbolFlagsAccessor != 0 {
		writeType := b.ch.getWriteTypeOfSymbol(propertySymbol)
		if !b.ch.isErrorType(propertyType) && !b.ch.isErrorType(writeType) {
			propDeclaration := ast.GetDeclarationOfKind(b.ch.symbolOf(propertySymbol), ast.KindPropertyDeclaration)
			if propertyType != writeType || b.ch.symbolParent(propertySymbol) != 0 && b.ch.symbolFlags(b.ch.symbolParent(propertySymbol))&ast.SymbolFlagsClass != 0 && propDeclaration.IsNil() {
				symbolMapper := b.ch.valueSymbolLinks.Get(propertySymbol).mapper
				if getterDeclaration := ast.GetDeclarationOfKind(b.ch.symbolOf(propertySymbol), ast.KindGetAccessor); !getterDeclaration.IsNil() {
					getterSignature := b.ch.getSignatureFromDeclaration(getterDeclaration)
					if symbolMapper != 0 {
						getterSignature = b.ch.instantiateSignature(getterSignature, b.ch.mapperOf(symbolMapper))
					}
					getter := b.signatureToSignatureDeclarationHelper(getterSignature, ast.KindGetAccessor, &SignatureToSignatureDeclarationOptions{
						name: propertyName,
					})
					b.setCommentRange(getter, getterDeclaration)
					typeElements = append(typeElements, getter)
				}
				if setterDeclaration := ast.GetDeclarationOfKind(b.ch.symbolOf(propertySymbol), ast.KindSetAccessor); !setterDeclaration.IsNil() {
					setterSignature := b.ch.getSignatureFromDeclaration(setterDeclaration)
					if symbolMapper != 0 {
						setterSignature = b.ch.instantiateSignature(setterSignature, b.ch.mapperOf(symbolMapper))
					}
					setter := b.signatureToSignatureDeclarationHelper(setterSignature, ast.KindSetAccessor, &SignatureToSignatureDeclarationOptions{
						name: propertyName,
					})
					b.setCommentRange(setter, setterDeclaration)
					typeElements = append(typeElements, setter)
				}
				return typeElements
			} else if b.ch.symbolParent(propertySymbol) != 0 && b.ch.symbolFlags(b.ch.symbolParent(propertySymbol))&ast.SymbolFlagsClass != 0 && !propDeclaration.IsNil() && !core.Find(propDeclaration.ModifierNodes(), func(m ast.Node) bool {
				return m.Kind() == ast.KindAccessorKeyword
			}).IsNil() {
				fakeGetterSignature := b.ch.newSignature(SignatureFlagsNone, ast.Node{}, nil, 0, nil, propertyType, nil, 0)
				fakeGetterDeclaration := b.signatureToSignatureDeclarationHelper(fakeGetterSignature, ast.KindGetAccessor, &SignatureToSignatureDeclarationOptions{
					name: propertyName,
				})
				b.setCommentRange(fakeGetterDeclaration, propDeclaration)
				typeElements = append(typeElements, fakeGetterDeclaration)

				setterParam := b.ch.newSymbol(ast.SymbolFlagsFunctionScopedVariable, ast.SymbolNameArg)
				b.ch.valueSymbolLinks.Get(setterParam).resolvedType = writeType.ref()
				fakeSetterSignature := b.ch.newSignature(SignatureFlagsNone, ast.Node{}, nil, 0, []ast.SymbolId{setterParam}, b.ch.voidType, nil, 0)
				fakeSetterDeclaration := b.signatureToSignatureDeclarationHelper(fakeSetterSignature, ast.KindSetAccessor, &SignatureToSignatureDeclarationOptions{
					name: propertyName,
				})
				typeElements = append(typeElements, fakeSetterDeclaration)
				return typeElements
			}
		}
	}

	var optionalToken ast.Node
	if b.ch.symbolFlags(propertySymbol)&ast.SymbolFlagsOptional != 0 {
		optionalToken = b.f.NewToken(ast.KindQuestionToken)
	} else {
		optionalToken = ast.Node{}
	}
	if b.ch.symbolFlags(propertySymbol)&(ast.SymbolFlagsFunction|ast.SymbolFlagsMethod) != 0 && len(b.ch.getPropertiesOfObjectType(propertyType)) == 0 && !b.ch.isReadonlySymbol(propertySymbol) {
		signatures := b.ch.getSignaturesOfType(b.ch.filterType(propertyType, func(t *Type) bool {
			return t.flags&TypeFlagsUndefined == 0
		}), SignatureKindCall)
		for _, signature := range signatures {
			methodDeclaration := b.signatureToSignatureDeclarationHelper(signature, ast.KindMethodSignature, &SignatureToSignatureDeclarationOptions{
				name:          propertyName,
				questionToken: optionalToken,
			})
			b.setCommentRange(methodDeclaration, core.Coalesce(b.ch.nodeOf(signature.declaration), b.ch.symbolValueDeclaration(propertySymbol)))
			typeElements = append(typeElements, methodDeclaration)
		}
		if len(signatures) != 0 || optionalToken.IsNil() {
			return typeElements
		}
	}
	var propertyTypeNode ast.TypeNode
	if b.shouldUsePlaceholderForProperty(propertySymbol) {
		propertyTypeNode = b.createElidedInformationPlaceholder()
	} else {
		if propertyIsReverseMapped {
			b.ctx.reverseMappedStack = append(b.ctx.reverseMappedStack, propertySymbol)
		}
		if propertyType != nil {
			propertyTypeNode = b.serializeTypeForDeclaration(ast.Node{} /*declaration*/, propertyType, propertySymbol, true)
		} else {
			propertyTypeNode = b.f.NewKeywordTypeNode(ast.KindAnyKeyword)
		}
		if propertyIsReverseMapped {
			b.ctx.reverseMappedStack = b.ctx.reverseMappedStack[:len(b.ctx.reverseMappedStack)-1]
		}
	}

	var modifiers ast.ModifierList
	if b.ch.isReadonlySymbol(propertySymbol) {
		modifiers = b.f.NewModifierList([]ast.Node{b.f.NewModifier(ast.KindReadonlyKeyword)})
		b.ctx.approximateLength += 9
	}
	propertySignature := b.f.NewPropertySignatureDeclaration(modifiers, propertyName, optionalToken, propertyTypeNode, ast.Node{})

	b.setCommentRange(propertySignature, b.ch.symbolValueDeclaration(propertySymbol))
	typeElements = append(typeElements, propertySignature)

	return typeElements
}

func (b *NodeBuilderImpl) createTypeNodesFromResolvedType(resolvedType *StructuredType) ast.NodeList {
	if b.checkTruncationLength() {
		if b.ctx.flags&nodebuilder.FlagsNoTruncation != 0 {
			elem := b.f.NewNotEmittedTypeElement()
			return b.f.NewNodeList([]ast.TypeElement{b.e.AddSyntheticTrailingComment(elem, ast.KindMultiLineCommentTrivia, "elided", false /*hasTrailingNewLine*/)})
		}
		return b.f.NewNodeList([]ast.Node{b.f.NewPropertySignatureDeclaration(ast.ModifierList{}, b.f.NewIdentifier("..."), ast.Node{}, ast.Node{}, ast.Node{})})
	}
	var typeElements []ast.TypeElement
	for _, signature := range resolvedType.CallSignatures(b.ch) {
		typeElements = append(typeElements, b.signatureToSignatureDeclarationHelper(signature, ast.KindCallSignature, nil))
	}
	for _, signature := range resolvedType.ConstructSignatures(b.ch) {
		if signature.flags&SignatureFlagsAbstract != 0 {
			continue
		}
		typeElements = append(typeElements, b.signatureToSignatureDeclarationHelper(signature, ast.KindConstructSignature, nil))
	}
	for _, info := range resolvedType.getIndexInfos(b.ch) {
		var typeNode ast.TypeNode
		if resolvedType.objectFlags&ObjectFlagsReverseMapped != 0 && b.ctx.flags&nodebuilder.FlagsAllowAnonymousIdentifier != 0 {
			typeNode = b.createElidedInformationPlaceholder()
		}
		typeElements = slices.Concat(typeElements, b.indexInfoToObjectComputedNamesOrSignatureDeclaration(info, typeNode))
	}

	properties := resolvedType.getProperties(b.ch)
	if len(properties) == 0 {
		return b.f.NewNodeList(typeElements)
	}

	i := 0
	for _, propertySymbol := range properties {
		if isExpanding(b.ctx) && b.ch.symbolFlags(propertySymbol)&ast.SymbolFlagsPrototype != 0 {
			continue
		}
		i++
		if b.ctx.flags&nodebuilder.FlagsWriteClassExpressionAsTypeLiteral != 0 {
			if b.ch.symbolFlags(propertySymbol)&ast.SymbolFlagsPrototype != 0 {
				continue
			}
			if b.ch.getDeclarationModifierFlagsFromSymbol(propertySymbol)&(ast.ModifierFlagsPrivate|ast.ModifierFlagsProtected) != 0 {
				b.ctx.tracker.ReportPrivateInBaseOfClassExpression(b.ch.symbolName(propertySymbol).String())
			}
			if b.ch.isPrivateIdentifierSymbol(propertySymbol) {
				b.ctx.tracker.ReportPrivateInBaseOfClassExpression(ast.SymbolNameText(b.ch.symbolOf(propertySymbol)))
			}
		}
		if b.checkTruncationLength() && (i+2 < len(properties)-1) {
			if b.ctx.flags&nodebuilder.FlagsNoTruncation != 0 {
				typeElements[len(typeElements)-1] = b.e.AddSyntheticTrailingComment(typeElements[len(typeElements)-1], ast.KindMultiLineCommentTrivia, fmt.Sprintf("... %d more elided ...", len(properties)-i), false /*hasTrailingNewLine*/)
			} else {
				text := fmt.Sprintf("... %d more ...", len(properties)-i)
				typeElements = append(typeElements, b.f.NewPropertySignatureDeclaration(ast.ModifierList{}, b.f.NewIdentifier(text), ast.Node{}, ast.Node{}, ast.Node{}))
			}
			typeElements = b.addPropertyToElementList(properties[len(properties)-1], typeElements)
			break
		}
		typeElements = b.addPropertyToElementList(propertySymbol, typeElements)
	}
	if len(typeElements) != 0 {
		return b.f.NewNodeList(typeElements)
	} else {
		return ast.NodeList{}
	}
}

func (b *NodeBuilderImpl) createTypeNodeFromObjectType(t *Type) ast.TypeNode {
	if b.ch.isGenericMappedType(t) || (t.objectFlags&ObjectFlagsMapped != 0 && t.AsMappedType().mapped(b.ch).containsError) {
		return b.createMappedTypeNodeFromType(t)
	}

	resolved := b.ch.resolveStructuredTypeMembers(t)
	callSigs := resolved.CallSignatures(b.ch)
	ctorSigs := resolved.ConstructSignatures(b.ch)
	if len(resolved.getProperties(b.ch)) == 0 && len(resolved.getIndexInfos(b.ch)) == 0 {
		if len(callSigs) == 0 && len(ctorSigs) == 0 {
			b.ctx.approximateLength += 2
			result := b.f.NewTypeLiteralNode(b.f.NewNodeList([]ast.Node{}))
			b.e.SetEmitFlags(result, printer.EFSingleLine)
			return result
		}

		if len(callSigs) == 1 && len(ctorSigs) == 0 {
			signature := callSigs[0]
			signatureNode := b.signatureToSignatureDeclarationHelper(signature, ast.KindFunctionType, nil)
			return signatureNode
		}

		if len(ctorSigs) == 1 && len(callSigs) == 0 {
			signature := ctorSigs[0]
			signatureNode := b.signatureToSignatureDeclarationHelper(signature, ast.KindConstructorType, nil)
			return signatureNode
		}
	}

	abstractSignatures := core.Filter(ctorSigs, func(signature *Signature) bool {
		return signature.flags&SignatureFlagsAbstract != 0
	})
	if len(abstractSignatures) > 0 {
		types := mapToTypeIds(abstractSignatures, func(s *Signature) *Type {
			return b.ch.getOrCreateTypeFromSignature(s)
		})
		// count the number of type elements excluding abstract constructors
		typeElementCount := len(callSigs) + (len(ctorSigs) - len(abstractSignatures)) + len(resolved.getIndexInfos(b.ch)) + core.IfElse(b.ctx.flags&nodebuilder.FlagsWriteClassExpressionAsTypeLiteral != 0, core.CountWhere(resolved.getProperties(b.ch), func(p ast.SymbolId) bool {
			return b.ch.symbolFlags(p)&ast.SymbolFlagsPrototype == 0
		}), len(resolved.getProperties(b.ch)))
		// don't include an empty object literal if there were no other static-side
		// properties to write, i.e. `abstract class C { }` becomes `abstract new () => {}`
		// and not `(abstract new () => {}) & {}`
		if typeElementCount != 0 {
			// create a copy of the object type without any abstract construct signatures.
			types = append(types, b.getResolvedTypeWithoutAbstractConstructSignatures(resolved).ref())
		}
		return b.typeToTypeNode(b.ch.getIntersectionType(types))
	}

	restoreFlags := b.saveRestoreFlags()
	b.ctx.flags |= nodebuilder.FlagsInObjectTypeLiteral
	members := b.createTypeNodesFromResolvedType(resolved)
	restoreFlags()
	typeLiteralNode := b.f.NewTypeLiteralNode(members)
	b.ctx.approximateLength += 2
	b.e.SetEmitFlags(typeLiteralNode, core.IfElse((b.ctx.flags&nodebuilder.FlagsMultilineObjectLiterals != 0), 0, printer.EFSingleLine))
	return typeLiteralNode
}

func getTypeAliasForTypeLiteral(c *Checker, t *Type) ast.SymbolId {
	if c.symbolOfType(t) != 0 && c.symbolFlags(c.symbolOfType(t))&ast.SymbolFlagsTypeLiteral != 0 && c.symbolDeclarations(c.symbolOfType(t)).Len() != 0 {
		node := ast.WalkUpParenthesizedTypes(c.symbolDeclarations(c.symbolOfType(t)).At(0).Parent())
		if ast.IsTypeAliasDeclaration(node) {
			return c.getSymbolOfDeclaration(node)
		}
	}
	return 0
}

func (b *NodeBuilderImpl) shouldWriteTypeOfFunctionSymbol(symbol ast.SymbolId, typeId TypeId) (bool, ast.SymbolId) {
	// `typeof C.name` can only be written when the member name is a valid identifier
	isStaticMethodSymbol := b.ch.symbolFlags(symbol)&ast.SymbolFlagsMethod != 0 && scanner.IsIdentifierText(b.ch.symbolName(symbol).String(), core.LanguageVariantStandard) && b.ch.symbolDeclarations(symbol).Some(func(declaration ast.Node) bool {
		return ast.IsStatic(declaration) && !b.ch.isLateBindableIndexSignature(ast.GetNameOfDeclaration(declaration))
	})
	isNonLocalFunctionSymbol := false
	isFunctionExpressionSymbol := false
	if b.ch.symbolFlags(symbol)&ast.SymbolFlagsFunction != 0 {
		if b.ch.symbolParent(symbol) != 0 {
			isNonLocalFunctionSymbol = true
		} else {
			for _, declaration := range b.ch.symbolDeclarations(symbol).All() {
				if declaration.Parent().Kind() == ast.KindSourceFile || declaration.Parent().Kind() == ast.KindModuleBlock {
					isNonLocalFunctionSymbol = true
					break
				}
				if ast.IsFunctionExpressionOrArrowFunction(declaration) && ast.IsVariableDeclaration(declaration.Parent()) &&
					ast.IsVariableDeclarationList(declaration.Parent().Parent()) && ast.IsVariableStatement(declaration.Parent().Parent().Parent()) &&
					!declaration.Parent().Parent().Parent().Parent().IsNil() &&
					(declaration.Parent().Parent().Parent().Parent().Kind() == ast.KindSourceFile || declaration.Parent().Parent().Parent().Parent().Kind() == ast.KindModuleBlock) {
					isNonLocalFunctionSymbol = true
					isFunctionExpressionSymbol = true
					break
				}
			}
		}
	}
	if isStaticMethodSymbol || isNonLocalFunctionSymbol {
		if isFunctionExpressionSymbol && !b.ch.symbolValueDeclaration(symbol).IsNil() && !b.ch.symbolValueDeclaration(symbol).Parent().IsNil() && b.ch.symbolValueDeclaration(symbol).Parent() != b.ctx.enclosingDeclaration {
			symbol = b.ch.getMergedSymbol(b.ch.nodeSymbol(b.ch.symbolValueDeclaration(symbol).Parent()))
		}
		// typeof is allowed only for static/non local functions
		return (b.ctx.flags&nodebuilder.FlagsUseTypeOfFunction != 0 || b.ctx.visitedTypes.Has(typeId)) && // it is type of the symbol uses itself recursively
			(b.ctx.flags&nodebuilder.FlagsUseStructuralFallback == 0 || b.ch.isValueSymbolAccessible(symbol, b.ctx.enclosingDeclaration)), symbol // And the build is going to succeed without visibility error or there is no structural fallback allowed
	}
	return false, symbol
}

func (b *NodeBuilderImpl) createAnonymousTypeNode(t *Type) ast.TypeNode {
	return b.createAnonymousTypeNodeEx(t, false, false)
}

func (b *NodeBuilderImpl) shouldEmitTypeOfSymbol(forceExpansion bool, forceClassExpansion bool, isInstanceType ast.SymbolFlags, symbol ast.SymbolId, typeId TypeId) (bool, ast.SymbolId) {
	if forceExpansion {
		return false, symbol
	}
	nonFunctionResult := b.ch.symbolFlags(symbol)&ast.SymbolFlagsClass != 0 && !forceClassExpansion && b.ch.getBaseTypeVariableOfClass(symbol) == nil && !(!b.ch.symbolValueDeclaration(symbol).IsNil() && ast.IsClassLike(b.ch.symbolValueDeclaration(symbol)) && b.ctx.flags&nodebuilder.FlagsWriteClassExpressionAsTypeLiteral != 0 && (!ast.IsClassDeclaration(b.ch.symbolValueDeclaration(symbol)) || b.ch.isSymbolAccessible(symbol, b.ctx.enclosingDeclaration, isInstanceType, false /*shouldComputeAliasesToMakeVisible*/).Accessibility != printer.SymbolAccessibilityAccessible)) || b.ch.symbolFlags(symbol)&(ast.SymbolFlagsEnum|ast.SymbolFlagsValueModule) != 0
	if nonFunctionResult {
		return true, symbol
	}
	return b.shouldWriteTypeOfFunctionSymbol(symbol, typeId)
}

func (b *NodeBuilderImpl) createAnonymousTypeNodeEx(t *Type, forceClassExpansion bool, forceExpansion bool) ast.TypeNode {
	typeId := t.Id()
	symbol := b.ch.symbolOfType(t)
	if symbol != 0 {
		isInstantiationExpressionType := t.objectFlags&ObjectFlagsInstantiationExpressionType != 0
		if isInstantiationExpressionType {
			instantiationExpressionType := t.AsInstantiationExpressionType()
			existing := b.ch.nodeOf(instantiationExpressionType.node())
			//  instantiationExpressionType.node is unreliable for constituents of unions and intersections.
			// declare const Err: typeof ErrImpl & (<T>() => T);
			// type ErrAlias<U> = typeof Err<U>;
			// declare const e: ErrAlias<number>;
			// ErrAlias<number> = typeof Err<number> = typeof ErrImpl & (<number>() => number)
			// The problem is each constituent of the intersection will be associated with typeof Err<number>
			// And when extracting a type for typeof ErrImpl from typeof Err<number> does not make sense.
			if ast.IsTypeQueryNode(existing) && b.getTypeFromTypeNode(existing, false) == t {
				// Guard against unbounded recursion when the existing typeof node fails to be reused
				// (e.g. its entity name isn't accessible from this scope) and the recovery boundary's
				// fallback re-enters typeToTypeNode with the very same instantiation type, which would
				// in turn try to reuse the same node again. Mark the type as visited around the reuse
				// attempt so the inner recursion bottoms out via the visitedTypes guard below.
				if b.ctx.visitedTypes.Has(typeId) {
					return b.createCyclicStructurePlaceholder()
				}
				b.ctx.visitedTypes.Add(typeId)
				typeNode := b.tryReuseExistingNonParameterTypeNode(existing, t, ast.Node{}, nil)
				b.ctx.visitedTypes.Delete(typeId)
				if !typeNode.IsNil() {
					return typeNode
				}
			}
			if b.ctx.visitedTypes.Has(typeId) {
				return b.createCyclicStructurePlaceholder()
			}
			return b.visitAndTransformType(t, (*NodeBuilderImpl).createTypeNodeFromObjectType)
		}
		var isInstanceType ast.SymbolFlags
		if isClassInstanceSide(b.ch, t) {
			isInstanceType = ast.SymbolFlagsType
		} else {
			isInstanceType = ast.SymbolFlagsValue
		}

		// !!! JS support
		// if c.isJSConstructor(symbol.ValueDeclaration) {
		// 	// Instance and static types share the same symbol; only add 'typeof' for the static side.
		// 	return b.symbolToTypeNode(symbol, isInstanceType, nil)
		// } else
		if ok, symbol := b.shouldEmitTypeOfSymbol(forceExpansion, forceClassExpansion, isInstanceType, symbol, typeId); ok {
			if b.shouldExpandType(t, false /*isAlias*/) {
				b.ctx.depth++
			} else {
				return b.symbolToTypeNode(symbol, isInstanceType, ast.NodeList{})
			}
		}
		if b.ctx.visitedTypes.Has(typeId) {
			// If type is an anonymous type literal in a type alias declaration, use type alias name
			typeAlias := getTypeAliasForTypeLiteral(b.ch, t)
			if typeAlias != 0 {
				// The specified symbol flags need to be reinterpreted as type flags
				return b.symbolToTypeNode(typeAlias, ast.SymbolFlagsType, ast.NodeList{})
			} else {
				return b.createCyclicStructurePlaceholder()
			}
		} else {
			return b.visitAndTransformType(t, (*NodeBuilderImpl).createTypeNodeFromObjectType)
		}
	} else if t.objectFlags&ObjectFlagsReverseMapped != 0 && b.ctx.flags&nodebuilder.FlagsAllowAnonymousIdentifier == 0 {
		if b.ctx.visitedTypes.Has(typeId) {
			return b.createCyclicStructurePlaceholder()
		}
		return b.visitAndTransformType(t, (*NodeBuilderImpl).createTypeNodeFromObjectType)
	} else {
		// Reverse mapped types use property and index signature placeholders for display.
		return b.createTypeNodeFromObjectType(t)
	}
}

func (b *NodeBuilderImpl) getTypeFromTypeNode(node ast.TypeNode, noMappedTypes bool) *Type {
	// !!! noMappedTypes optional param support
	if node.Parent().IsNil() {
		return b.ch.errorType
	}
	t := b.ch.getTypeFromTypeNode(node)
	if b.ctx.mapper == nil {
		return t
	}

	instantiated := b.ch.instantiateType(t, b.ctx.mapper)
	if noMappedTypes && instantiated != t {
		return nil
	}
	return instantiated
}

func (b *NodeBuilderImpl) typeToTypeNodeOrCircularityElision(t *Type) ast.TypeNode {
	if t.flags&TypeFlagsUnion != 0 {
		if b.ctx.visitedTypes.Has(t.Id()) {
			return b.createCyclicStructurePlaceholder()
		}
		return b.visitAndTransformType(t, (*NodeBuilderImpl).typeToTypeNode)
	}
	return b.typeToTypeNode(t)
}

func (b *NodeBuilderImpl) createCyclicStructurePlaceholder() ast.TypeNode {
	if b.ctx.flags&nodebuilder.FlagsAllowAnonymousIdentifier == 0 {
		b.ctx.encounteredError = true
		b.ctx.tracker.ReportCyclicStructureError()
	}
	return b.createElidedInformationPlaceholder()
}

func (b *NodeBuilderImpl) conditionalTypeToTypeNode(_t *Type) ast.TypeNode {
	if b.checkTruncationLength() {
		return b.createElidedInformationPlaceholder()
	}
	t := _t.AsConditionalType()
	checkTypeNode := b.typeToTypeNode(t.checkType(b.ch))
	b.ctx.approximateLength += 15
	if b.ctx.flags&nodebuilder.FlagsGenerateNamesForShadowedTypeParams != 0 && t.cond(b.ch).root.isDistributive && t.checkType(b.ch).flags&TypeFlagsTypeParameter == 0 {
		newParam := b.ch.newTypeParameter(b.ch.newSymbol(ast.SymbolFlagsTypeParameter, ast.SymbolNameT /* as __String */))
		name := b.typeParameterToName(newParam)
		newTypeVariable := b.f.NewTypeReferenceNode(name.AsNode(), ast.TypeList{})
		b.ctx.approximateLength += 37
		// 15 each for two added conditionals, 7 for an added infer type
		newMapper := prependTypeMapping(b.ch, b.ch.typ(t.cond(b.ch).root.checkType), newParam, b.ch.mapperOf(t.cond(b.ch).mapper))
		saveInferTypeParameters := b.ctx.inferTypeParameters
		b.ctx.inferTypeParameters = t.cond(b.ch).root.inferTypeParameters
		extendsTypeNode := b.typeToTypeNode(b.ch.instantiateType(b.ch.typ(t.cond(b.ch).root.extendsType), newMapper))
		b.ctx.inferTypeParameters = saveInferTypeParameters
		trueTypeNode := b.typeToTypeNodeOrCircularityElision(b.ch.instantiateType(b.getTypeFromTypeNode(t.cond(b.ch).root.node.TrueType(), false), newMapper))
		falseTypeNode := b.typeToTypeNodeOrCircularityElision(b.ch.instantiateType(b.getTypeFromTypeNode(t.cond(b.ch).root.node.FalseType(), false), newMapper))

		// outermost conditional makes `T` a type parameter, allowing the inner conditionals to be distributive
		// second conditional makes `T` have `T & checkType` substitution, so it is correctly usable as the checkType
		// inner conditional runs the check the user provided on the check type (distributively) and returns the result
		// checkType extends infer T ? T extends checkType ? T extends extendsType<T> ? trueType<T> : falseType<T> : never : never;
		// this is potentially simplifiable to
		// checkType extends infer T ? T extends checkType & extendsType<T> ? trueType<T> : falseType<T> : never;
		// but that may confuse users who read the output more.
		// On the other hand,
		// checkType extends infer T extends checkType ? T extends extendsType<T> ? trueType<T> : falseType<T> : never;
		// may also work with `infer ... extends ...` in, but would produce declarations only compatible with the latest TS.
		newId := newTypeVariable.AsTypeReferenceNode().TypeName().AsIdentifier().Clone(b.f)
		syntheticExtendsNode := b.f.NewInferTypeNode(b.f.NewTypeParameterDeclaration(ast.ModifierList{}, newId, ast.Node{}, ast.Node{}, ast.Node{}))
		innerCheckConditionalNode := b.f.NewConditionalTypeNode(newTypeVariable, extendsTypeNode, trueTypeNode, falseTypeNode)
		syntheticTrueNode := b.f.NewConditionalTypeNode(b.f.NewTypeReferenceNode(name.Clone(b.f), ast.TypeList{}), b.f.DeepCloneNode(checkTypeNode), innerCheckConditionalNode, b.f.NewKeywordTypeNode(ast.KindNeverKeyword))
		return b.f.NewConditionalTypeNode(checkTypeNode, syntheticExtendsNode, syntheticTrueNode, b.f.NewKeywordTypeNode(ast.KindNeverKeyword))
	}
	saveInferTypeParameters := b.ctx.inferTypeParameters
	b.ctx.inferTypeParameters = t.cond(b.ch).root.inferTypeParameters
	extendsTypeNode := b.typeToTypeNode(t.extendsType(b.ch))
	b.ctx.inferTypeParameters = saveInferTypeParameters
	trueTypeNode := b.typeToTypeNodeOrCircularityElision(b.ch.getTrueTypeFromConditionalType(_t))
	falseTypeNode := b.typeToTypeNodeOrCircularityElision(b.ch.getFalseTypeFromConditionalType(_t))
	return b.f.NewConditionalTypeNode(checkTypeNode, extendsTypeNode, trueTypeNode, falseTypeNode)
}

func (b *NodeBuilderImpl) getParentSymbolOfTypeParameter(typeParameter *TypeParameter) ast.SymbolId {
	tp := ast.GetDeclarationOfKind(b.ch.symbolOf(b.ch.symbolOfType(typeParameter.AsType())), ast.KindTypeParameter)
	var host ast.Node
	// !!! JSDoc support
	// if ast.IsJSDocTemplateTag(tp.Parent) {
	// 	host = getEffectiveContainerForJSDocTemplateTag(tp.Parent)
	// } else {
	host = tp.Parent()
	// }
	if host.IsNil() {
		return 0
	}
	return b.ch.getSymbolOfNode(host)
}

func (b *NodeBuilderImpl) arrayOrTupleTypeToNode(t *Type) ast.TypeNode {
	var typeArguments []TypeId = b.ch.getTypeArguments(t)
	if t.Target(b.ch) == b.ch.globalArrayType || t.Target(b.ch) == b.ch.globalReadonlyArrayType {
		if b.ctx.flags&nodebuilder.FlagsWriteArrayAsGenericType != 0 {
			typeArgumentNode := b.typeToTypeNode(b.ch.typ(typeArguments[0]))
			return b.f.NewTypeReferenceNode(
				b.newIdentifier(core.IfElse(t.Target(b.ch) == b.ch.globalArrayType, "Array", "ReadonlyArray"), b.ch.symbolOfType(t.Target(b.ch))),
				b.f.NewNodeList([]ast.TypeNode{typeArgumentNode}),
			)
		}
		elementType := b.typeToTypeNode(b.ch.typ(typeArguments[0]))
		arrayType := b.f.NewArrayTypeNode(elementType)
		if t.Target(b.ch) == b.ch.globalArrayType {
			return arrayType
		} else {
			return b.f.NewTypeOperatorNode(ast.KindReadonlyKeyword, arrayType)
		}
	} else {
		debug.Assert(t.Target(b.ch).objectFlags&ObjectFlagsTuple != 0)
		typeArguments = b.ch.typesSameMapIndex(typeArguments, func(arg *Type, i int) *Type {
			isOptional := false
			if i < len(t.Target(b.ch).AsTupleType().tuple(b.ch).elementInfos) {
				isOptional = t.Target(b.ch).AsTupleType().tuple(b.ch).elementInfos[i].flags&ElementFlagsOptional != 0
			}
			return b.ch.removeMissingType(arg, isOptional)
		})
		if len(typeArguments) > 0 {
			arity := b.ch.getTypeReferenceArity(t)
			tupleConstituentNodes := b.mapToTypeNodes(typeArguments[0:arity], false /*isBareList*/)
			if !tupleConstituentNodes.IsNil() {
				for i := range tupleConstituentNodes.Len() {
					flags := t.Target(b.ch).AsTupleType().tuple(b.ch).elementInfos[i].flags
					labeledElementDeclaration := t.Target(b.ch).AsTupleType().tuple(b.ch).elementInfos[i].labeledDeclaration

					if !labeledElementDeclaration.IsNil() {
						tupleConstituentNodes.SetAt(i, b.f.NewNamedTupleMember(
							core.IfElse(flags&ElementFlagsVariable != 0, b.f.NewToken(ast.KindDotDotDotToken), ast.Node{}),
							b.newIdentifier(b.ch.getTupleElementLabel(t.Target(b.ch).AsTupleType().tuple(b.ch).elementInfos[i], 0, i).String(), 0 /*symbol*/),
							core.IfElse(flags&ElementFlagsOptional != 0, b.f.NewToken(ast.KindQuestionToken), ast.Node{}),
							core.IfElse(flags&ElementFlagsRest != 0, b.f.NewArrayTypeNode(tupleConstituentNodes.At(i)), tupleConstituentNodes.At(i)),
						))
					} else {
						switch {
						case flags&ElementFlagsVariable != 0:
							tupleConstituentNodes.SetAt(i, b.f.NewRestTypeNode(core.IfElse(flags&ElementFlagsRest != 0, b.f.NewArrayTypeNode(tupleConstituentNodes.At(i)), tupleConstituentNodes.At(i))))
						case flags&ElementFlagsOptional != 0:
							tupleConstituentNodes.SetAt(i, b.f.NewOptionalTypeNode(tupleConstituentNodes.At(i)))
						}
					}
				}
				tupleTypeNode := b.f.NewTupleTypeNode(tupleConstituentNodes)
				b.e.SetEmitFlags(tupleTypeNode, printer.EFSingleLine)
				if t.Target(b.ch).AsTupleType().tuple(b.ch).readonly {
					return b.f.NewTypeOperatorNode(ast.KindReadonlyKeyword, tupleTypeNode)
				} else {
					return tupleTypeNode
				}
			}
		}
		if b.ctx.encounteredError || (b.ctx.flags&nodebuilder.FlagsAllowEmptyTuple != 0) {
			tupleTypeNode := b.f.NewTupleTypeNode(b.f.NewNodeList([]ast.TypeNode{}))
			b.e.SetEmitFlags(tupleTypeNode, printer.EFSingleLine)
			if t.Target(b.ch).AsTupleType().tuple(b.ch).readonly {
				return b.f.NewTypeOperatorNode(ast.KindReadonlyKeyword, tupleTypeNode)
			} else {
				return tupleTypeNode
			}
		}
		b.ctx.encounteredError = true
		return ast.Node{}
		// TODO: GH#18217
	}
}

func (b *NodeBuilderImpl) typeReferenceToTypeNode(t *Type) ast.TypeNode {
	var typeArguments []TypeId = b.ch.getTypeArguments(t)
	if b.ctx.flags&nodebuilder.FlagsWriteClassExpressionAsTypeLiteral != 0 && !b.ch.symbolValueDeclaration(b.ch.symbolOfType(t)).IsNil() && ast.IsClassLike(b.ch.symbolValueDeclaration(b.ch.symbolOfType(t))) && !b.ch.isValueSymbolAccessible(b.ch.symbolOfType(t), b.ctx.enclosingDeclaration) {
		return b.createAnonymousTypeNode(t)
	} else {
		outerTypeParameters := t.Target(b.ch).AsInterfaceType().OuterTypeParameters(b.ch)
		i := 0
		var resultType ast.TypeNode
		if outerTypeParameters != nil {
			length := len(outerTypeParameters)
			for i < length {
				// Find group of type arguments for type parameters with the same declaring container.
				start := i
				parent := b.getParentSymbolOfTypeParameter(b.ch.typ(outerTypeParameters[i]).AsTypeParameter())
				for ok := true; ok; ok = i < length && b.getParentSymbolOfTypeParameter(b.ch.typ(outerTypeParameters[i]).AsTypeParameter()) == parent { // do-while loop
					i++
				}
				// When type parameters are their own type arguments for the whole group (i.e. we have
				// the default outer type arguments), we don't show the group.

				if !slices.Equal(outerTypeParameters[start:i], typeArguments[start:i]) {
					typeArgumentSlice := b.mapToTypeNodes(typeArguments[start:i], false /*isBareList*/)
					restoreFlags := b.saveRestoreFlags()
					b.ctx.flags |= nodebuilder.FlagsForbidIndexedAccessSymbolReferences
					ref := b.symbolToTypeNode(parent, ast.SymbolFlagsType, typeArgumentSlice)
					restoreFlags()
					if resultType.IsNil() {
						resultType = ref
					} else {
						resultType = b.appendReferenceToType(resultType, ref)
					}
				}
			}
		}
		var typeArgumentNodes ast.NodeList
		if len(typeArguments) > 0 {
			typeParameterCount := 0
			typeParams := t.Target(b.ch).AsInterfaceType().TypeParameters(b.ch)
			if typeParams != nil {
				typeParameterCount = min(len(typeParams), len(typeArguments))

				// Maybe we should do this for more types, but for now we only elide type arguments that are
				// identical to their associated type parameters' defaults for `Iterable`, `IterableIterator`,
				// `AsyncIterable`, and `AsyncIterableIterator` to provide backwards-compatible .d.ts emit due
				// to each now having three type parameters instead of only one.
				if b.ch.isReferenceToType(t, b.ch.getGlobalIterableType()) || b.ch.isReferenceToType(t, b.ch.getGlobalIterableIteratorType()) || b.ch.isReferenceToType(t, b.ch.getGlobalAsyncIterableType()) || b.ch.isReferenceToType(t, b.ch.getGlobalAsyncIterableIteratorType()) {
					if b.ch.nodeOf(t.AsTypeReference().node(b.ch)).IsNil() || !ast.IsTypeReferenceNode(b.ch.nodeOf(t.AsTypeReference().node(b.ch))) || b.ch.nodeOf(t.AsTypeReference().node(b.ch)).TypeArguments() == nil || b.ch.nodeOf(t.AsTypeReference().node(b.ch)).TypeArgumentList().Len() < typeParameterCount {
						for typeParameterCount > 0 {
							typeArgument := b.ch.typ(typeArguments[typeParameterCount-1])
							typeParameter := b.ch.typ(t.Target(b.ch).AsInterfaceType().TypeParameters(b.ch)[typeParameterCount-1])
							defaultType := b.ch.getDefaultFromTypeParameter(typeParameter)
							if defaultType == nil || !b.ch.isTypeIdenticalTo(typeArgument, defaultType) {
								break
							}
							typeParameterCount--
						}
					}
				}
			}

			typeArgumentNodes = b.mapToTypeNodes(typeArguments[i:typeParameterCount], false /*isBareList*/)
		}
		restoreFlags := b.saveRestoreFlags()
		b.ctx.flags |= nodebuilder.FlagsForbidIndexedAccessSymbolReferences
		finalRef := b.symbolToTypeNode(b.ch.symbolOfType(t), ast.SymbolFlagsType, typeArgumentNodes)
		restoreFlags()
		if resultType.IsNil() {
			return finalRef
		} else {
			return b.appendReferenceToType(resultType, finalRef)
		}
	}
}

func (b *NodeBuilderImpl) visitAndTransformType(t *Type, transform func(b *NodeBuilderImpl, t *Type) ast.TypeNode) ast.TypeNode {
	if b.checkTruncationLength() {
		return b.createElidedInformationPlaceholder()
	}

	typeId := t.Id()
	isArrayOrTuple := b.ch.isArrayOrTupleType(t)
	if isArrayOrTuple {
		// Deferred and regular references share a cycle identity.
		typeId = b.ch.createTypeReference(t.Target(b.ch), b.ch.getTypeArguments(t)).Id()
	}
	if b.ctx.visitedTypes.Has(typeId) {
		return b.createCyclicStructurePlaceholder()
	}

	isConstructorObject := t.objectFlags&ObjectFlagsAnonymous != 0 && b.ch.symbolOfType(t) != 0 && b.ch.symbolFlags(b.ch.symbolOfType(t))&ast.SymbolFlagsClass != 0
	var id *CompositeSymbolIdentity
	switch {
	case isArrayOrTuple:
		// Do not bound finite container nesting by the shared Array symbol or tuple origin.
		id = nil
	case t.objectFlags&ObjectFlagsReference != 0 && !b.ch.nodeOf(t.AsTypeReference().node(b.ch)).IsNil():
		id = &CompositeSymbolIdentity{false, 0, ast.GetNodeId(b.ch.nodeOf(t.AsTypeReference().node(b.ch)))}
	case t.flags&TypeFlagsConditional != 0:
		id = &CompositeSymbolIdentity{false, 0, ast.GetNodeId(t.AsConditionalType().cond(b.ch).root.node.AsNode())}
	case b.ch.symbolOfType(t) != 0:
		id = &CompositeSymbolIdentity{isConstructorObject, ast.GetSymbolSerial(b.ch.symbolOf(b.ch.symbolOfType(t))), 0}
	default:
		id = nil
	}
	// Since instantiations of the same anonymous type have the same symbol, tracking symbols instead
	// of types allows us to catch circular references to instantiations of the same anonymous type

	key := CompositeTypeCacheIdentity{
		typeId:        typeId,
		flags:         b.ctx.flags,
		internalFlags: b.ctx.internalFlags,
	}
	if len(b.ctx.inferTypeParameters) != 0 {
		key.inferTypeParameters = getTypeListKey(b.ctx.inferTypeParameters)
	}
	// Don't rely on type cache if we're expanding a type, because we need to compute `canIncreaseExpansionDepth`.
	canUseCache := b.ctx.maxExpansionDepth < 0
	if canUseCache && !b.ctx.enclosingDeclaration.IsNil() && b.links.Has(b.ctx.enclosingDeclaration) {
		links := b.links.Get(b.ctx.enclosingDeclaration)
		cachedResult, ok := links.serializedTypes[key]
		if ok {
			// TODO:: check if we instead store late painted statements associated with this?
			for _, arg := range cachedResult.trackedSymbols {
				b.ctx.tracker.TrackSymbol(b.ch.symbolOf(b.ch.symbolId(arg.symbol)), arg.enclosingDeclaration, arg.meaning)
			}
			if cachedResult.truncating {
				b.ctx.truncating = true
			}
			b.ctx.approximateLength += cachedResult.addedLength
			return b.f.DeepCloneNode(cachedResult.node)
		}
	}

	if t.objectFlags&ObjectFlagsReverseMapped != 0 {
		// Growing type arguments can prevent a reverse mapped type from repeating.
		// Bound expansion by its mapped declaration as well as its type identity.
		origin := CompositeSymbolIdentity{nodeId: ast.GetNodeId(t.AsReverseMappedType().mappedType(b.ch).AsMappedType().mapped(b.ch).declaration.AsNode())}
		depth := b.ctx.symbolDepth[origin]
		if depth >= 100 {
			b.ctx.truncating = true
			return b.createElidedInformationPlaceholder()
		}
		b.ctx.symbolDepth[origin] = depth + 1
		defer func() { b.ctx.symbolDepth[origin] = depth }()
	}

	var depth int
	if id != nil {
		depth = b.ctx.symbolDepth[*id]
		if depth > 10 {
			b.ctx.truncating = true
			return b.createElidedInformationPlaceholder()
		}
		b.ctx.symbolDepth[*id] = depth + 1
	}
	b.ctx.visitedTypes.Add(typeId)
	prevTrackedSymbols := b.ctx.trackedSymbols
	b.ctx.trackedSymbols = nil
	startLength := b.ctx.approximateLength
	result := transform(b, t)
	addedLength := b.ctx.approximateLength - startLength
	if canUseCache && !b.ctx.reportedDiagnostic && !b.ctx.encounteredError {
		links := b.links.Get(b.ctx.enclosingDeclaration)
		if links.serializedTypes == nil {
			links.serializedTypes = make(map[CompositeTypeCacheIdentity]*SerializedTypeEntry)
		}
		links.serializedTypes[key] = &SerializedTypeEntry{
			node:           result,
			truncating:     b.ctx.truncating,
			addedLength:    addedLength,
			trackedSymbols: b.ctx.trackedSymbols,
		}
	}
	b.ctx.visitedTypes.Delete(typeId)
	if id != nil {
		b.ctx.symbolDepth[*id] = depth
	}
	b.ctx.trackedSymbols = prevTrackedSymbols
	return result

	// !!! TODO: Attempt node reuse or parse nodes to minimize copying once text range setting is set up
	// deepCloneOrReuseNode := func(node T) T {
	// 	if !nodeIsSynthesized(node) && getParseTreeNode(node) == node {
	// 		return node
	// 	}
	// 	return setTextRange(b.ctx, b.f.cloneNode(visitEachChildWorker(node, deepCloneOrReuseNode, nil /*b.ctx*/, deepCloneOrReuseNodes, deepCloneOrReuseNode)), node)
	// }

	// deepCloneOrReuseNodes := func(nodes *NodeArray[ast.Node], visitor Visitor, test func(node ast.Node) bool, start number, count number) *NodeArray[ast.Node] {
	// 	if nodes != nil && nodes.length == 0 {
	// 		// Ensure we explicitly make a copy of an empty array; visitNodes will not do this unless the array has elements,
	// 		// which can lead to us reusing the same empty NodeArray more than once within the same AST during type noding.
	// 		return setTextRangeWorker(b.f.NewNodeArray(nil, nodes.hasTrailingComma), nodes)
	// 	}
	// 	return visitNodes(nodes, visitor, test, start, count)
	// }
}

func (b *NodeBuilderImpl) typeToTypeNode(t *Type) ast.TypeNode {
	inTypeAlias := b.ctx.flags & nodebuilder.FlagsInTypeAlias
	b.ctx.flags &^= nodebuilder.FlagsInTypeAlias

	if t == nil {
		if b.ctx.flags&nodebuilder.FlagsAllowEmptyUnionOrIntersection == 0 {
			b.ctx.encounteredError = true
			return ast.Node{}
			// TODO: GH#18217
		}
		b.ctx.approximateLength += 3
		return b.f.NewKeywordTypeNode(ast.KindAnyKeyword)
	}

	t = getNonDistributedTypeParameter(b.ch, t)

	// Push type onto typeStack for expansion depth tracking
	if b.ctx.maxExpansionDepth >= 0 {
		b.ctx.typeStack = append(b.ctx.typeStack, t.ref())
		defer func() {
			b.ctx.typeStack = b.ctx.typeStack[:len(b.ctx.typeStack)-1]
		}()
	}

	if b.ctx.flags&nodebuilder.FlagsNoTypeReduction == 0 {
		t = b.ch.getReducedType(t)
	}

	if t.flags&TypeFlagsAny != 0 {
		if t.Alias(b.ch) != nil {
			return t.Alias(b.ch).ToTypeReferenceNode(b)
		}
		if t == b.ch.unresolvedType {
			return b.e.AddSyntheticLeadingComment(b.f.NewKeywordTypeNode(ast.KindAnyKeyword), ast.KindMultiLineCommentTrivia, "unresolved", false /*hasTrailingNewLine*/)
		}
		b.ctx.approximateLength += 3
		return b.f.NewKeywordTypeNode(core.IfElse(t == b.ch.intrinsicMarkerType, ast.KindIntrinsicKeyword, ast.KindAnyKeyword))
	}
	if t.flags&TypeFlagsUnknown != 0 {
		return b.f.NewKeywordTypeNode(ast.KindUnknownKeyword)
	}
	if t.flags&TypeFlagsString != 0 {
		b.ctx.approximateLength += 6
		return b.f.NewKeywordTypeNode(ast.KindStringKeyword)
	}
	if t.flags&TypeFlagsNumber != 0 {
		b.ctx.approximateLength += 6
		return b.f.NewKeywordTypeNode(ast.KindNumberKeyword)
	}
	if t.flags&TypeFlagsBigInt != 0 {
		b.ctx.approximateLength += 6
		return b.f.NewKeywordTypeNode(ast.KindBigIntKeyword)
	}
	if t.flags&TypeFlagsBoolean != 0 && t.Alias(b.ch) == nil {
		b.ctx.approximateLength += 7
		return b.f.NewKeywordTypeNode(ast.KindBooleanKeyword)
	}
	expandingEnum := false
	if t.flags&TypeFlagsEnumLike != 0 {
		if b.ch.symbolFlags(b.ch.symbolOfType(t))&ast.SymbolFlagsEnumMember != 0 {
			parentSymbol := b.ch.getParentOfSymbol(b.ch.symbolOfType(t))
			parentName := b.symbolToTypeNode(parentSymbol, ast.SymbolFlagsType, ast.NodeList{})
			if b.ch.getDeclaredTypeOfSymbol(parentSymbol) == t {
				return parentName
			}
			memberName := ast.SymbolNameText(b.ch.symbolOf(b.ch.symbolOfType(t)))
			if scanner.IsIdentifierText(memberName, core.LanguageVariantStandard) {
				return b.appendReferenceToType(parentName /* as TypeReference | ImportTypeNode */, b.f.NewTypeReferenceNode(b.f.NewIdentifier(memberName), ast.TypeList{} /*typeArguments*/))
			}
			if ast.IsImportTypeNode(parentName) {
				parentName.AsImportTypeNode().SetIsTypeOf(true)
				// mutably update, node is freshly manufactured anyhow
				return b.f.NewIndexedAccessTypeNode(parentName, b.f.NewLiteralTypeNode(b.newStringLiteral(memberName)))
			} else if ast.IsTypeReferenceNode(parentName) {
				return b.f.NewIndexedAccessTypeNode(b.f.NewTypeQueryNode(parentName.AsTypeReferenceNode().TypeName(), ast.TypeList{}), b.f.NewLiteralTypeNode(b.newStringLiteral(memberName)))
			} else {
				panic("Unhandled type node kind returned from `symbolToTypeNode`.")
			}
		}
		if t.flags&TypeFlagsUnion == 0 || !b.shouldExpandType(t, false /*isAlias*/) {
			return b.symbolToTypeNode(b.ch.symbolOfType(t), ast.SymbolFlagsType, ast.NodeList{})
		}
		expandingEnum = true
	}
	if t.flags&TypeFlagsStringLiteral != 0 {
		b.ctx.approximateLength += len(t.AsLiteralType().Value(b.ch).(string)) + 2
		lit := b.newStringLiteral(t.AsLiteralType().Value(b.ch).(string))
		b.e.AddEmitFlags(lit, printer.EFNoAsciiEscaping)
		return b.f.NewLiteralTypeNode(lit)
	}
	if t.flags&TypeFlagsNumberLiteral != 0 {
		value := t.AsLiteralType().Value(b.ch).(jsnum.Number)
		b.ctx.approximateLength += len(value.String())
		if value < 0 {
			return b.f.NewLiteralTypeNode(b.f.NewPrefixUnaryExpression(ast.KindMinusToken, b.f.NewNumericLiteral(value.String()[1:], ast.TokenFlagsNone)))
		} else {
			return b.f.NewLiteralTypeNode(b.f.NewNumericLiteral(value.String(), ast.TokenFlagsNone))
		}
	}
	if t.flags&TypeFlagsBigIntLiteral != 0 {
		b.ctx.approximateLength += len(pseudoBigIntToString(getBigIntLiteralValue(b.ch, t))) + 1
		return b.f.NewLiteralTypeNode(b.f.NewBigIntLiteral(pseudoBigIntToString(getBigIntLiteralValue(b.ch, t))+"n", ast.TokenFlagsNone))
	}
	if t.flags&TypeFlagsBooleanLiteral != 0 {
		if t.AsLiteralType().Value(b.ch).(bool) {
			b.ctx.approximateLength += 4
			return b.f.NewLiteralTypeNode(b.f.NewKeywordExpression(ast.KindTrueKeyword))
		} else {
			b.ctx.approximateLength += 5
			return b.f.NewLiteralTypeNode(b.f.NewKeywordExpression(ast.KindFalseKeyword))
		}
	}
	if t.flags&TypeFlagsUniqueESSymbol != 0 {
		if b.ctx.flags&nodebuilder.FlagsAllowUniqueESSymbolType == 0 {
			if b.ch.isValueSymbolAccessible(b.ch.symbolOfType(t), b.ctx.enclosingDeclaration) {
				b.ctx.approximateLength += 6
				return b.symbolToTypeNode(b.ch.symbolOfType(t), ast.SymbolFlagsValue, ast.NodeList{})
			}
			b.ctx.tracker.ReportInaccessibleUniqueSymbolError()
		}
		b.ctx.approximateLength += 13
		return b.f.NewTypeOperatorNode(ast.KindUniqueKeyword, b.f.NewKeywordTypeNode(ast.KindSymbolKeyword))
	}
	if t.flags&TypeFlagsVoid != 0 {
		b.ctx.approximateLength += 4
		return b.f.NewKeywordTypeNode(ast.KindVoidKeyword)
	}
	if t.flags&TypeFlagsUndefined != 0 {
		b.ctx.approximateLength += 9
		return b.f.NewKeywordTypeNode(ast.KindUndefinedKeyword)
	}
	if t.flags&TypeFlagsNull != 0 {
		b.ctx.approximateLength += 4
		return b.f.NewLiteralTypeNode(b.f.NewKeywordExpression(ast.KindNullKeyword))
	}
	if t.flags&TypeFlagsNever != 0 {
		b.ctx.approximateLength += 5
		return b.f.NewKeywordTypeNode(ast.KindNeverKeyword)
	}
	if t.flags&TypeFlagsESSymbol != 0 {
		b.ctx.approximateLength += 6
		return b.f.NewKeywordTypeNode(ast.KindSymbolKeyword)
	}
	if t.flags&TypeFlagsNonPrimitive != 0 {
		b.ctx.approximateLength += 6
		return b.f.NewKeywordTypeNode(ast.KindObjectKeyword)
	}
	if isThisTypeParameter(t) {
		if b.ctx.flags&nodebuilder.FlagsInObjectTypeLiteral != 0 {
			if !b.ctx.encounteredError && b.ctx.flags&nodebuilder.FlagsAllowThisInObjectLiteral == 0 {
				b.ctx.encounteredError = true
			}
			b.ctx.tracker.ReportInaccessibleThisError()
		}
		b.ctx.approximateLength += 4
		return b.f.NewThisTypeNode()
	}

	if inTypeAlias == 0 && t.Alias(b.ch) != nil && (b.ctx.flags&nodebuilder.FlagsUseAliasDefinedOutsideCurrentScope != 0 || b.ch.isTypeSymbolAccessible(t.Alias(b.ch).Symbol(), b.ctx.enclosingDeclaration)) {
		// If we should expand this type alias, skip the alias and fall through to expand the underlying type
		if !b.shouldExpandType(t, true /*isAlias*/) {
			sym := t.Alias(b.ch).Symbol()
			typeArgumentNodes := b.mapToTypeNodes(t.Alias(b.ch).TypeArguments(), false /*isBareList*/)
			if b.ch.symbolName(sym).IsReservedMemberName() && b.ch.symbolFlags(sym)&ast.SymbolFlagsClass == 0 {
				return b.f.NewTypeReferenceNode(b.f.NewIdentifier(""), typeArgumentNodes)
			}
			if !typeArgumentNodes.IsNil() && typeArgumentNodes.Len() == 1 && sym == b.ch.symbolOfType(b.ch.globalArrayType) {
				return b.f.NewArrayTypeNode(typeArgumentNodes.At(0))
			}
			return b.symbolToTypeNode(sym, ast.SymbolFlagsType, typeArgumentNodes)
		}
		// Expanding: increment depth and process the underlying type
		b.ctx.depth++
		defer func() { b.ctx.depth-- }()
	}

	objectFlags := t.objectFlags

	if objectFlags&ObjectFlagsReference != 0 {
		debug.Assert(t.Flags()&TypeFlagsObject != 0)
		// When expanding, expand type references to their structural form
		if b.shouldExpandType(t, false /*isAlias*/) {
			b.ctx.depth++
			result := b.createAnonymousTypeNodeEx(t, true /*forceClassExpansion*/, true /*forceExpansion*/)
			b.ctx.depth--
			return result
		}
		if b.ch.isArrayOrTupleType(t) {
			return b.visitAndTransformType(t, (*NodeBuilderImpl).arrayOrTupleTypeToNode)
		} else if !b.ch.nodeOf(t.AsTypeReference().node(b.ch)).IsNil() {
			return b.visitAndTransformType(t, (*NodeBuilderImpl).typeReferenceToTypeNode)
		} else {
			return b.typeReferenceToTypeNode(t)
		}
	}
	if t.flags&TypeFlagsTypeParameter != 0 || objectFlags&ObjectFlagsClassOrInterface != 0 {
		// When expanding class or interface types, show their structural form
		if objectFlags&ObjectFlagsClassOrInterface != 0 && b.shouldExpandType(t, false /*isAlias*/) {
			b.ctx.depth++
			result := b.createAnonymousTypeNodeEx(t, true /*forceClassExpansion*/, true /*forceExpansion*/)
			b.ctx.depth--
			return result
		}
		if t.flags&TypeFlagsTypeParameter != 0 && slices.Contains(b.ctx.inferTypeParameters, t.ref()) {
			b.ctx.approximateLength += len(ast.SymbolNameText(b.ch.symbolOf(b.ch.symbolOfType(t)))) + 6
			var constraintNode ast.TypeNode
			constraint := b.ch.getConstraintOfTypeParameter(t)
			if constraint != nil {
				// If the infer type has a constraint that is not the same as the constraint
				// we would have normally inferred based on b, we emit the constraint
				// using `infer T extends ?`. We omit inferred constraints from type references
				// as they may be elided.
				inferredConstraint := b.ch.getInferredTypeParameterConstraint(t, true /*omitTypeReferences*/)
				if !(inferredConstraint != nil && b.ch.isTypeIdenticalTo(constraint, inferredConstraint)) {
					b.ctx.approximateLength += 9
					constraintNode = b.typeToTypeNode(constraint)
				}
			}
			return b.f.NewInferTypeNode(b.typeParameterToDeclarationWithConstraint(t, constraintNode))
		}
		if b.ctx.flags&nodebuilder.FlagsGenerateNamesForShadowedTypeParams != 0 && t.flags&TypeFlagsTypeParameter != 0 {
			name := b.typeParameterToName(t)
			b.ctx.approximateLength += len(name.Text().String())
			return b.f.NewTypeReferenceNode(b.newIdentifier(name.Text().String(), b.ch.symbolOfType(t)), ast.TypeList{} /*typeArguments*/)
		}
		// Ignore constraint/default when creating a usage (as opposed to declaration) of a type parameter.
		if b.ch.symbolOfType(t) != 0 {
			return b.symbolToTypeNode(b.ch.symbolOfType(t), ast.SymbolFlagsType, ast.NodeList{})
		}
		var name string
		if (t == b.ch.markerSuperTypeForCheck || t == b.ch.markerSubTypeForCheck) && b.ch.varianceTypeParameter != nil && b.ch.symbolOfType(b.ch.varianceTypeParameter) != 0 {
			name = core.IfElse(t == b.ch.markerSubTypeForCheck, "sub-", "super-") + ast.SymbolNameText(b.ch.symbolOf(b.ch.symbolOfType(b.ch.varianceTypeParameter)))
		} else {
			name = "?"
		}
		return b.f.NewTypeReferenceNode(b.newIdentifier(name, 0 /*symbol*/), ast.TypeList{} /*typeArguments*/)
	}
	if t.flags&TypeFlagsUnion != 0 && t.AsUnionType().origin(b.ch) != nil {
		t = t.AsUnionType().origin(b.ch)
	}
	if t.flags&(TypeFlagsUnion|TypeFlagsIntersection) != 0 {
		var types []TypeId
		if t.flags&TypeFlagsUnion != 0 {
			types = b.ch.formatUnionTypes(t.AsUnionType().types(b.ch), expandingEnum)
		} else {
			types = t.AsIntersectionType().types(b.ch)
		}
		if len(types) == 1 {
			return b.typeToTypeNode(b.ch.typ(types[0]))
		}
		typeNodes := b.mapToTypeNodes(types, true /*isBareList*/)
		if !typeNodes.IsNil() && typeNodes.Len() > 0 {
			if t.flags&TypeFlagsUnion != 0 {
				return b.f.NewUnionTypeNode(typeNodes)
			} else {
				return b.f.NewIntersectionTypeNode(typeNodes)
			}
		} else {
			if !b.ctx.encounteredError && b.ctx.flags&nodebuilder.FlagsAllowEmptyUnionOrIntersection == 0 {
				b.ctx.encounteredError = true
			}
			return ast.Node{}
			// TODO: GH#18217
		}
	}
	if objectFlags&(ObjectFlagsAnonymous|ObjectFlagsMapped) != 0 {
		debug.Assert(t.Flags()&TypeFlagsObject != 0)
		// The type is an object literal type.
		return b.createAnonymousTypeNode(t)
	}
	if t.flags&TypeFlagsIndex != 0 {
		indexedType := t.Target(b.ch)
		b.ctx.approximateLength += 6
		indexTypeNode := b.typeToTypeNode(indexedType)
		return b.f.NewTypeOperatorNode(ast.KindKeyOfKeyword, indexTypeNode)
	}
	if t.flags&TypeFlagsTemplateLiteral != 0 {
		texts := t.AsTemplateLiteralType().texts(b.ch)
		types := t.AsTemplateLiteralType().types(b.ch)
		templateHead := b.f.NewTemplateHead(texts[0], "", ast.TokenFlagsNone)
		b.e.AddEmitFlags(templateHead, printer.EFNoAsciiEscaping)
		templateSpans := b.f.NewNodeList(typesMapIndexTo(b.ch, types, func(t *Type, i int) ast.Node {
			var res ast.TemplateMiddleOrTail
			if i < len(types)-1 {
				res = b.f.NewTemplateMiddle(texts[i+1], "", ast.TokenFlagsNone)
			} else {
				res = b.f.NewTemplateTail(texts[i+1], "", ast.TokenFlagsNone)
			}
			b.e.AddEmitFlags(res, printer.EFNoAsciiEscaping)
			return b.f.NewTemplateLiteralTypeSpan(b.typeToTypeNode(t), res)
		}))
		b.ctx.approximateLength += 2
		return b.f.NewTemplateLiteralTypeNode(templateHead, templateSpans)
	}
	if t.flags&TypeFlagsStringMapping != 0 {
		typeNode := b.typeToTypeNode(t.Target(b.ch))
		return b.symbolToTypeNode(b.ch.symbolOfType(t), ast.SymbolFlagsType, b.f.NewNodeList([]ast.Node{typeNode}))
	}
	if t.flags&TypeFlagsIndexedAccess != 0 {
		objectTypeNode := b.typeToTypeNode(t.AsIndexedAccessType().objectType(b.ch))
		indexTypeNode := b.typeToTypeNode(t.AsIndexedAccessType().indexType(b.ch))
		b.ctx.approximateLength += 2
		return b.f.NewIndexedAccessTypeNode(objectTypeNode, indexTypeNode)
	}
	if t.flags&TypeFlagsConditional != 0 {
		return b.visitAndTransformType(t, (*NodeBuilderImpl).conditionalTypeToTypeNode)
	}
	if t.flags&TypeFlagsSubstitution != 0 {
		typeNode := b.typeToTypeNode(t.AsSubstitutionType().baseType(b.ch))
		if !b.ch.isNoInferType(t) {
			return typeNode
		}
		noInferSymbol := b.ch.getGlobalTypeAliasSymbol(ast.SymbolNameNoInfer, 1, false)
		if noInferSymbol != 0 {
			return b.symbolToTypeNode(noInferSymbol, ast.SymbolFlagsType, b.f.NewNodeList([]ast.Node{typeNode}))
		} else {
			return typeNode
		}
	}

	panic("Should be unreachable.")
}

func (b *NodeBuilderImpl) newStringLiteral(text string) ast.Node {
	return b.newStringLiteralEx(text, false /*isSingleQuote*/)
}

func (b *NodeBuilderImpl) newStringLiteralEx(text string, isSingleQuote bool) ast.Node {
	flags := ast.TokenFlagsNone
	if isSingleQuote || b.ctx.flags&nodebuilder.FlagsUseSingleQuotesForStringLiteralType != 0 {
		flags |= ast.TokenFlagsSingleQuote
	}
	node := b.f.NewStringLiteral(text, flags)
	return node
}

// Direct serialization core functions for types, type aliases, and symbols

func (t *TypeAlias) ToTypeReferenceNode(b *NodeBuilderImpl) ast.Node {
	return b.f.NewTypeReferenceNode(b.symbolToEntityNameNode(t.Symbol()), b.mapToTypeNodes(t.TypeArguments(), false /*isBareList*/))
}

func (b *NodeBuilderImpl) newIdentifier(text string, symbol ast.SymbolId) ast.Node {
	id := b.f.NewIdentifier(text)
	if symbol != 0 {
		b.idToSymbol[id] = b.ch.symbolOf(symbol)
	}
	return id
}

func (b *NodeBuilderImpl) createAccessExpression(node ast.Node) ast.Expression {
	switch {
	case ast.IsQualifiedName(node):
		return b.f.NewPropertyAccessExpression(b.createAccessExpression(node.AsQualifiedName().Left()), ast.Node{} /*questionDotToken*/, b.f.DeepCloneNode(node.AsQualifiedName().Right()), ast.NodeFlagsNone)
	case ast.IsIdentifier(node), ast.IsPropertyAccessExpression(node), ast.IsExpressionWithTypeArguments(node):
		return b.f.DeepCloneNode(node)
	default:
		panic("unexpected access node kind: " + node.Kind().String())
	}
}

func (b *NodeBuilderImpl) createExpressionWithTypeArguments(expr ast.Expression, typeArguments ast.NodeList) ast.Expression {
	if typeArguments.IsNil() || typeArguments.Len() == 0 {
		return expr
	}
	return b.f.NewExpressionWithTypeArguments(expr, typeArguments)
}

func (b *NodeBuilderImpl) lookupInstantiatedTypeArgumentNodes(chain []ast.SymbolId, index int) ast.TypeParameterList {
	if b.shouldWriteTypeParametersInQualifiedName(chain, index) {
		symbol := chain[index]
		nextSymbol := chain[index+1]
		if b.ch.symbolCheckFlags(nextSymbol)&ast.CheckFlagsInstantiated == 0 {
			return ast.TypeParameterList{}
		}

		targetSymbol := symbol
		if b.ch.symbolFlags(symbol)&ast.SymbolFlagsAlias != 0 && !b.ch.canGetTypeParametersOfClassOrInterface(symbol) {
			targetSymbol = b.ch.resolveAlias(symbol)
		}

		if !b.ch.canGetTypeParametersOfClassOrInterface(targetSymbol) {
			return ast.TypeParameterList{}
		}

		params := b.getTypeParametersOfClassOrInterface(targetSymbol)
		targetMapper := b.ch.mapperOf(b.ch.valueSymbolLinks.Get(nextSymbol).mapper)
		if targetMapper != nil {
			params = b.ch.typesMap(params, func(t *Type) *Type { return targetMapper.Map(b.ch, t) })
		}
		return b.mapToTypeNodes(params, false /*isBareList*/)
	}
	return ast.TypeParameterList{}
}

func (b *NodeBuilderImpl) lookupExpressionChainTypeArgumentNodes(chain []ast.SymbolId, index int) ast.TypeParameterList {
	if b.shouldWriteTypeParametersInQualifiedName(chain, index) {
		symbol := chain[index]
		symbolId := ast.GetSymbolSerial(b.ch.symbolOf(symbol))
		if b.ctx.typeParameterSymbolList.Has(symbolId) {
			return ast.TypeParameterList{}
		}

		b.ctx.typeParameterSymbolList.Add(symbolId)
		if typeArgumentNodes := b.lookupInstantiatedTypeArgumentNodes(chain, index); !typeArgumentNodes.IsNil() {
			return typeArgumentNodes
		}
		typeParameterNodes := b.typeParametersToTypeParameterDeclarations(symbol)
		if len(typeParameterNodes) > 0 {
			return b.f.NewNodeList(typeParameterNodes)
		}
	}
	return ast.TypeParameterList{}
}

func (b *NodeBuilderImpl) shouldWriteTypeParametersInQualifiedName(chain []ast.SymbolId, index int) bool {
	return b.ctx.flags&nodebuilder.FlagsWriteTypeParametersInQualifiedName != 0 && index < len(chain)-1
}
