package checker

import (
	"slices"
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/astnav"
	"github.com/microsoft/TypeScript/tsc/internal/collections"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/debug"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
)

func (c *Checker) getSymbolsInScopeId(location ast.Node, meaning ast.SymbolFlags) []ast.SymbolId {
	return c.getSymbolsInScope(location, meaning)
}

func (c *Checker) getSymbolsInScope(location ast.Node, meaning ast.SymbolFlags) []ast.SymbolId {
	if location.Flags()&ast.NodeFlagsInWithStatement != 0 {
		// We cannot answer semantic questions within a with block, do not proceed any further
		return nil
	}

	symbols := c.symbols.NewTable(0)
	isStaticSymbol := false

	// Copy the given symbol into symbol tables if the symbol has the given meaning
	// and it doesn't already exists in the symbol table.
	copySymbol := func(symbol ast.SymbolId, meaning ast.SymbolFlags) {
		if c.combinedLocalAndExportSymbolFlags(symbol)&meaning != 0 {
			id := c.symbolName(symbol)
			// We will copy all symbol regardless of its reserved name because
			// symbolsToArray will check whether the key is a reserved name and
			// it will not copy symbol with reserved name to the array
			if _, ok := symbols.LookupId(c.symbols, id); !ok {
				symbols.SetId(c.symbols, id, symbol)
			}
		}
	}

	copySymbols := func(source ast.SymbolTable, meaning ast.SymbolFlags) {
		if meaning != 0 {
			for _, symbol := range source.AllIds(c.symbols) {
				copySymbol(symbol, meaning)
			}
		}
	}

	copyLocallyVisibleExportSymbols := func(source ast.SymbolTable, meaning ast.SymbolFlags) {
		if meaning != 0 {
			for _, symbol := range source.AllIds(c.symbols) {
				// Similar condition as in `resolveNameHelper`
				if ast.GetDeclarationOfKind(c.symbolOf(symbol), ast.KindExportSpecifier).IsNil() &&
					ast.GetDeclarationOfKind(c.symbolOf(symbol), ast.KindNamespaceExport).IsNil() &&
					c.symbolName(symbol) != ast.InternalSymbolNameDefault {
					copySymbol(symbol, meaning)
				}
			}
		}
	}

	populateSymbols := func() {
		var lastLocation ast.Node
		for !location.IsNil() {
			if ast.IsModuleDeclaration(location) &&
				!location.AsModuleDeclaration().Attributes().IsNil() &&
				lastLocation == location.AsModuleDeclaration().Attributes() {
				// Module declaration is not in scope inside its attributes.
				lastLocation = location
				location = location.Parent()
				continue
			}

			if canHaveLocals(location) && location.Locals() != nil && !ast.IsGlobalSourceFile(location) {
				copySymbols(location.Locals(), meaning)
			}

			switch location.Kind() {
			case ast.KindSourceFile:
				if !ast.IsExternalModule(location.AsSourceFile()) {
					break
				}
				fallthrough
			case ast.KindModuleDeclaration:
				copyLocallyVisibleExportSymbols(c.symbolExports(c.getSymbolOfDeclaration(location)), meaning&ast.SymbolFlagsModuleMember)
			case ast.KindEnumDeclaration:
				copySymbols(c.symbolExports(c.getSymbolOfDeclaration(location)), meaning&ast.SymbolFlagsEnumMember)
			case ast.KindClassExpression:
				className := location.AsClassExpression().Name()
				if !className.IsNil() {
					copySymbol(c.nodeSymbol(location), meaning)
				}
				// this fall-through is necessary because we would like to handle
				// type parameter inside class expression similar to how we handle it in classDeclaration and interface Declaration.
				fallthrough
			case ast.KindClassDeclaration, ast.KindInterfaceDeclaration:
				// If we didn't come from static member of class or interface,
				// add the type parameters into the symbol table
				// (type parameters of classDeclaration/classExpression and interface are in member property of the symbol.
				// Note: that the memberFlags come from previous iteration.
				if !isStaticSymbol {
					copySymbols(c.getMembersOfSymbol(c.getSymbolOfDeclaration(location)), meaning&ast.SymbolFlagsType)
				}
			case ast.KindFunctionExpression:
				funcName := location.Name()
				if !funcName.IsNil() {
					copySymbol(c.nodeSymbol(location), meaning)
				}
			}

			if introducesArgumentsExoticObject(location) {
				copySymbol(c.argumentsSymbol, meaning)
			}

			isStaticSymbol = ast.IsStatic(location)
			lastLocation = location
			location = location.Parent()
		}

		copySymbols(c.globals, meaning)
	}

	populateSymbols()

	symbols.Delete(ast.InternalSymbolNameThis) // Not a symbol, a keyword
	return c.symbolsToArray(symbols)
}

func (c *Checker) getExportsOfModuleId(symbol ast.SymbolId) []ast.SymbolId {
	return c.symbolsToArray(c.getExportsOfModule(symbol))
}

func (c *Checker) ForEachExportAndPropertyOfModule(moduleSymbol ast.SymbolId, cb func(ast.SymbolId, ast.SymbolName)) {
	for key, exportedSymbol := range c.getExportsOfModule(moduleSymbol).AllIds(c.symbols) {
		if !key.IsReservedMemberName() {
			cb(exportedSymbol, key)
		}
	}

	exportEquals := c.resolveExternalModuleSymbol(moduleSymbol, false /*dontResolveAlias*/)
	if exportEquals == moduleSymbol {
		return
	}

	typeOfSymbol := c.getTypeOfSymbol(exportEquals)
	if !c.shouldTreatPropertiesOfExternalModuleAsExports(typeOfSymbol) {
		return
	}

	// forEachPropertyOfType
	reducedType := c.getReducedApparentType(typeOfSymbol)
	if reducedType.flags&TypeFlagsStructuredType == 0 {
		return
	}
	for name, symbol := range c.resolveStructuredTypeMembers(reducedType).getMembers(c).AllIds(c.symbols) {
		if c.isNamedMember(symbol, name) {
			cb(symbol, name)
		}
	}
}

func (c *Checker) IsValidPropertyAccess(node ast.Node, propertyName ast.SymbolName) bool {
	return c.isValidPropertyAccess(node, propertyName)
}

func (c *Checker) isValidPropertyAccess(node ast.Node, propertyName ast.SymbolName) bool {
	switch node.Kind() {
	case ast.KindPropertyAccessExpression:
		return c.isValidPropertyAccessWithType(node, node.Expression().Kind() == ast.KindSuperKeyword, propertyName, c.getWidenedType(c.checkExpression(node.Expression())))
	case ast.KindQualifiedName:
		return c.isValidPropertyAccessWithType(node, false /*isSuper*/, propertyName, c.getWidenedType(c.checkExpression(node.AsQualifiedName().Left())))
	case ast.KindImportType:
		return c.isValidPropertyAccessWithType(node, false /*isSuper*/, propertyName, c.getTypeFromTypeNode(node))
	}
	panic("Unexpected node kind in isValidPropertyAccess: " + node.Kind().String())
}

func (c *Checker) isValidPropertyAccessWithType(node ast.Node, isSuper bool, propertyName ast.SymbolName, t *Type) bool {
	// Short-circuiting for improved performance.
	if IsTypeAny(t) {
		return true
	}

	prop := c.getPropertyOfType(t, propertyName)
	return prop != 0 && c.isPropertyAccessible(node, isSuper, false /*isWrite*/, t, prop)
}

// Checks if an existing property access is valid for completions purposes.
// node: a property access-like node where we want to check if we can access a property.
// This node does not need to be an access of the property we are checking.
// e.g. in completions, this node will often be an incomplete property access node, as in `foo.`.
// Besides providing a location (i.e. scope) used to check property accessibility, we use this node for
// computing whether this is a `super` property access.
// type: the type whose property we are checking.
// property: the accessed property's symbol.
func (c *Checker) isValidPropertyAccessForCompletionsId(node ast.Node, t *Type, property ast.SymbolId) bool {
	return c.isPropertyAccessible(
		node,
		node.Kind() == ast.KindPropertyAccessExpression && node.Expression().Kind() == ast.KindSuperKeyword,
		false, /*isWrite*/
		t,
		property,
	)
	// Previously we validated the 'this' type of methods but this adversely affected performance. See #31377 for more context.
}

func (c *Checker) getAllPossiblePropertiesOfTypes(types []TypeId) []ast.SymbolId {
	unionType := c.getUnionType(types)
	if unionType.flags&TypeFlagsUnion == 0 {
		return c.getAugmentedPropertiesOfType(unionType)
	}

	props := c.symbols.NewTable(0)
	for _, memberTypeId := range types {
		memberType := c.typ(memberTypeId)
		augmentedProps := c.getAugmentedPropertiesOfType(memberType)
		for _, p := range augmentedProps {
			if _, ok := props.LookupId(c.symbols, c.symbolName(p)); !ok {
				prop := c.createUnionOrIntersectionProperty(unionType, c.symbolName(p), false /*skipObjectFunctionPropertyAugment*/)
				// May be undefined if the property is private
				if prop != 0 {
					props.SetId(c.symbols, c.symbolName(p), prop)
				}
			}
		}
	}
	return slices.Collect(props.ValueIds(c.symbols))
}

func (c *Checker) isUnknownSymbol(symbol ast.SymbolId) bool {
	return symbol == c.unknownSymbol
}

func (c *Checker) isUndefinedSymbol(symbol ast.SymbolId) bool {
	return symbol == c.undefinedSymbol
}

func (c *Checker) isArgumentsSymbol(symbol ast.SymbolId) bool {
	return symbol == c.argumentsSymbol
}

// Originally from services.ts
func (c *Checker) GetNonOptionalType(t *Type) *Type {
	return c.removeOptionalTypeMarker(t)
}

func (c *Checker) GetStringIndexType(t *Type) *Type {
	return c.getIndexTypeOfType(t, c.stringType)
}

func (c *Checker) GetNumberIndexType(t *Type) *Type {
	return c.getIndexTypeOfType(t, c.numberType)
}

func (c *Checker) GetElementTypeOfArrayType(t *Type) *Type {
	return c.getElementTypeOfArrayType(t)
}

func (c *Checker) GetCallSignatures(t *Type) []*Signature {
	return c.getSignaturesOfType(t, SignatureKindCall)
}

func (c *Checker) GetConstructSignatures(t *Type) []*Signature {
	return c.getSignaturesOfType(t, SignatureKindConstruct)
}

func (c *Checker) getApparentProperties(t *Type) []ast.SymbolId {
	return c.getAugmentedPropertiesOfType(t)
}

func (c *Checker) getAugmentedPropertiesOfType(t *Type) []ast.SymbolId {
	t = c.getApparentType(t)
	propsByName := c.createSymbolTable(c.getPropertiesOfType(t))
	var functionType *Type
	if len(c.getSignaturesOfType(t, SignatureKindCall)) > 0 {
		functionType = c.globalCallableFunctionType
	} else if len(c.getSignaturesOfType(t, SignatureKindConstruct)) > 0 {
		functionType = c.globalNewableFunctionType
	}

	if propsByName == nil {
		propsByName = c.symbols.NewTable(0)
	}
	if functionType != nil {
		for _, p := range c.getPropertiesOfType(functionType) {
			if _, ok := propsByName.LookupId(c.symbols, c.symbolName(p)); !ok {
				propsByName.SetId(c.symbols, c.symbolName(p), p)
			}
		}
	}
	return c.getNamedMembers(propsByName)
}

func (c *Checker) tryGetMemberInModuleExportsAndProperties(memberName ast.SymbolName, moduleSymbol ast.SymbolId) ast.SymbolId {
	symbol := c.tryGetMemberInModuleExports(memberName, moduleSymbol)
	if symbol != 0 {
		return symbol
	}

	exportEquals := c.resolveExternalModuleSymbol(moduleSymbol, false /*dontResolveAlias*/)
	if exportEquals == moduleSymbol {
		return 0
	}

	t := c.getTypeOfSymbol(exportEquals)
	if c.shouldTreatPropertiesOfExternalModuleAsExports(t) {
		return c.getPropertyOfType(t, memberName)
	}
	return 0
}

func (c *Checker) TryGetMemberInModuleExports(memberName ast.SymbolName, moduleSymbol ast.Symbol) ast.Symbol {
	return c.symbolOf(c.tryGetMemberInModuleExports(memberName, c.symbolId(moduleSymbol)))
}

func (c *Checker) tryGetMemberInModuleExports(memberName ast.SymbolName, moduleSymbol ast.SymbolId) ast.SymbolId {
	symbolTable := c.getExportsOfModule(moduleSymbol)
	return symbolTable.GetId(c.symbols, memberName)
}

func (c *Checker) shouldTreatPropertiesOfExternalModuleAsExports(resolvedExternalModuleType *Type) bool {
	return resolvedExternalModuleType.flags&TypeFlagsPrimitive == 0 ||
		resolvedExternalModuleType.objectFlags&ObjectFlagsClass != 0 ||
		// `isArrayOrTupleLikeType` is too expensive to use in this auto-imports hot path.
		c.isArrayType(resolvedExternalModuleType) ||
		c.isTupleType(resolvedExternalModuleType)
}

func (c *Checker) GetContextualType(node ast.Expression, contextFlags ContextFlags) *Type {
	if contextFlags&ContextFlagsIgnoreNodeInferences != 0 {
		return c.runWithInferenceBlockedFromSourceNode(node, func() *Type { return c.getContextualType(node, contextFlags) })
	}
	return c.getContextualType(node, contextFlags)
}

func (c *Checker) runWithInferenceBlockedFromSourceNode[T any](node ast.Node, fn func() T) T {
	containingCall := ast.FindAncestor(node, ast.IsCallLikeExpression)
	if !containingCall.IsNil() {
		toMarkSkip := node
		for {
			c.skipDirectInferenceNodes.Add(c.refs.Ref(toMarkSkip))
			toMarkSkip = toMarkSkip.Parent()
			if toMarkSkip.IsNil() || toMarkSkip == containingCall {
				break
			}
		}
	}

	c.isInferencePartiallyBlocked = true
	result := c.runWithoutResolvedSignatureCaching(node, fn)
	c.isInferencePartiallyBlocked = false

	c.skipDirectInferenceNodes.Clear()
	return result
}

func GetResolvedSignatureForSignatureHelp(node ast.Node, argumentCount int, c *Checker) (*Signature, []*Signature) {
	type result struct {
		signature  *Signature
		candidates []*Signature
	}
	res := c.runWithoutResolvedSignatureCaching(node, func() result {
		signature, candidates := c.getResolvedSignatureWorker(node, CheckModeIsForSignatureHelp, argumentCount)
		return result{signature, candidates}
	})
	return res.signature, res.candidates
}

func (c *Checker) runWithoutResolvedSignatureCaching[T any](node ast.Node, fn func() T) T {
	ancestorNode := ast.FindAncestor(node, ast.IsCallLikeOrFunctionLikeExpression)
	if !ancestorNode.IsNil() {
		cachedResolvedSignatures := make(map[ast.Node]SignatureId)
		cachedTypes := make(map[*ValueSymbolLinks]*Type)
		for !ancestorNode.IsNil() {
			cachedResolvedSignatures[ancestorNode] = c.resolvedSignatures.get(ancestorNode)
			c.resolvedSignatures.delete(ancestorNode)
			if ast.IsFunctionExpressionOrArrowFunction(ancestorNode) {
				symbolLinks := c.valueSymbolLinks.Get(c.getSymbolOfDeclaration(ancestorNode))
				resolvedType := c.typ(symbolLinks.resolvedType)
				cachedTypes[symbolLinks] = resolvedType
				symbolLinks.resolvedType = 0
			}
			ancestorNode = ast.FindAncestor(ancestorNode.Parent(), ast.IsCallLikeOrFunctionLikeExpression)
		}
		result := fn()
		for node, resolvedSignature := range cachedResolvedSignatures {
			if resolvedSignature != 0 {
				c.resolvedSignatures.set(node, resolvedSignature)
			} else {
				c.resolvedSignatures.delete(node)
			}
		}
		for symbolLinks, resolvedType := range cachedTypes {
			symbolLinks.resolvedType = resolvedType.ref()
		}
		return result
	}
	return fn()
}

func (c *Checker) skipAlias(symbol ast.SymbolId) ast.SymbolId {
	if c.symbolFlags(symbol)&ast.SymbolFlagsAlias != 0 {
		return c.getAliasedSymbol(symbol)
	}
	return symbol
}

func (c *Checker) getRootSymbols(symbol ast.SymbolId) []ast.SymbolId {
	roots := c.getImmediateRootSymbols(symbol)
	if len(roots) == 0 {
		return []ast.SymbolId{symbol}
	}
	var result []ast.SymbolId
	for _, root := range roots {
		result = append(result, c.getRootSymbols(root)...)
	}
	return result
}

func (c *Checker) getMappedTypeSymbolOfProperty(symbol ast.SymbolId) ast.SymbolId {
	if valueLinks := c.valueSymbolLinks.TryGet(symbol); valueLinks != nil {
		return c.symbolOfType(valueLinks.getContainingType(c))
	}
	return 0
}

func (c *Checker) getImmediateRootSymbols(symbol ast.SymbolId) []ast.SymbolId {
	if c.symbolCheckFlags(symbol)&ast.CheckFlagsSynthetic != 0 {
		return typesMapNonNilTo(c, c.valueSymbolLinks.Get(symbol).getContainingType(c).Types(c), func(t *Type) ast.SymbolId {
			return c.getPropertyOfType(t, c.symbolName(symbol))
		})
	}
	if c.symbolFlags(symbol)&ast.SymbolFlagsTransient != 0 {
		if c.spreadLinks.Has(symbol) {
			leftSpread := c.spreadLinks.Get(symbol).leftSpread
			rightSpread := c.spreadLinks.Get(symbol).rightSpread
			if leftSpread != 0 {
				return []ast.SymbolId{leftSpread, rightSpread}
			}
		}
		if c.mappedSymbolLinks.Has(symbol) {
			syntheticOrigin := c.mappedSymbolLinks.Get(symbol).syntheticOrigin
			if syntheticOrigin != 0 {
				return []ast.SymbolId{syntheticOrigin}
			}
		}
		target := c.tryGetTarget(symbol)
		if target != 0 {
			return []ast.SymbolId{target}
		}
	}
	return nil
}

func (c *Checker) tryGetTarget(symbol ast.SymbolId) ast.SymbolId {
	var target ast.SymbolId
	next := symbol
	for {
		if c.valueSymbolLinks.Has(next) {
			next = c.valueSymbolLinks.Get(next).target
		} else if c.exportTypeLinks.Has(next) {
			next = c.exportTypeLinks.Get(next).target
		} else {
			next = 0
		}
		if next == 0 {
			break
		}
		target = next
	}
	return target
}

func (c *Checker) getExportSymbolOfSymbol(symbol ast.SymbolId) ast.SymbolId {
	return c.getMergedSymbol(core.IfElse(c.symbolExportSymbol(symbol) != 0, c.symbolExportSymbol(symbol), symbol))
}

func (c *Checker) getExportSpecifierLocalTargetSymbol(node ast.Node) ast.SymbolId {
	// node should be ExportSpecifier | Identifier
	switch node.Kind() {
	case ast.KindExportSpecifier:
		if !node.Parent().Parent().ModuleSpecifier().IsNil() {
			return c.getExternalModuleMember(node.Parent().Parent(), node, false /*dontResolveAlias*/)
		}
		name := node.PropertyNameOrName()
		if name.Kind() == ast.KindStringLiteral {
			// Skip for invalid syntax like this: export { "x" }
			return 0
		}
		return c.resolveEntityName(name, ast.SymbolFlagsValue|ast.SymbolFlagsType|ast.SymbolFlagsNamespace|ast.SymbolFlagsAlias, true /*ignoreErrors*/, false, ast.Node{})
	case ast.KindIdentifier:
		return c.resolveEntityName(node, ast.SymbolFlagsValue|ast.SymbolFlagsType|ast.SymbolFlagsNamespace|ast.SymbolFlagsAlias, true /*ignoreErrors*/, false, ast.Node{})
	}
	panic("Unhandled case in getExportSpecifierLocalTargetSymbol, node should be ExportSpecifier | Identifier")
}

func (c *Checker) getShorthandAssignmentValueSymbol(location ast.Node) ast.SymbolId {
	if !location.IsNil() && location.Kind() == ast.KindShorthandPropertyAssignment {
		return c.resolveEntityName(location.Name(), ast.SymbolFlagsValue|ast.SymbolFlagsAlias, true /*ignoreErrors*/, false, ast.Node{})
	}
	return 0
}

/**
* Get symbols that represent parameter-property-declaration as parameter and as property declaration
* @param parameter a parameterDeclaration node
* @param parameterName a name of the parameter to get the symbols for.
* @return a tuple of two symbols
 */
func (c *Checker) getSymbolsOfParameterPropertyDeclaration(parameter ast.Node /*ParameterPropertyDeclaration*/, parameterName ast.SymbolName) (ast.SymbolId, ast.SymbolId) {
	constructorDeclaration := parameter.Parent()
	classDeclaration := parameter.Parent().Parent()

	parameterSymbol := c.getSymbol(constructorDeclaration.Locals(), parameterName, ast.SymbolFlagsValue)
	propertySymbol := c.getSymbol(c.getMembersOfSymbol(c.nodeSymbol(classDeclaration)), parameterName, ast.SymbolFlagsValue)

	if parameterSymbol != 0 && propertySymbol != 0 {
		return parameterSymbol, propertySymbol
	}

	panic("There should exist two symbols, one as property declaration and one as parameter declaration")
}

// IsDeclarationUsed checks if an import declaration identifier is used in the source file.
// This is primarily used for organizing imports to determine which imports can be removed.
func (c *Checker) IsDeclarationUsed(
	sourceFile *ast.SourceFile,
	identifier ast.Identifier,
	jsxElementsPresent bool,
	jsxModeNeedsExplicitImport bool,
) bool {
	if jsxElementsPresent && jsxModeNeedsExplicitImport {
		jsxNamespace := c.getJsxNamespace(sourceFile.AsNode())
		jsxFragmentFactory := c.GetJsxFragmentFactory(sourceFile.AsNode())
		identifierText := identifier.Text().String()
		if identifierText == jsxNamespace {
			return true
		}
		if jsxFragmentFactory != "" && identifierText == jsxFragmentFactory {
			return true
		}
	}

	symbol := c.getSymbolAtLocationId(identifier.AsNode())
	if symbol == 0 {
		return true
	}

	return c.IsSymbolReferencedInFile(sourceFile, identifier, symbol)
}

// IsSymbolReferencedInFile checks if a symbol is referenced in the source file (besides its definition).
// This is used as a quick check for whether a symbol is used at all in a file.
func (c *Checker) IsSymbolReferencedInFile(
	sourceFile *ast.SourceFile,
	definition ast.Identifier,
	symbol ast.SymbolId,
) bool {
	identifierText := definition.Text()
	for _, token := range getPossibleSymbolReferenceNodes(sourceFile, identifierText.String(), sourceFile.AsNode()) {
		if !ast.IsIdentifier(token) {
			continue
		}
		id := token.AsIdentifier()
		if id == definition || id.Text() != identifierText {
			continue
		}
		refSymbol := c.getSymbolAtLocationId(token)
		if refSymbol == symbol {
			return true
		}
		if !token.Parent().IsNil() && token.Parent().Kind() == ast.KindShorthandPropertyAssignment {
			shorthandSymbol := c.getShorthandAssignmentValueSymbol(token.Parent())
			if shorthandSymbol == symbol {
				return true
			}
		}
		if !token.Parent().IsNil() && ast.IsExportSpecifier(token.Parent()) {
			localSymbol := c.getLocalSymbolForExportSpecifier(token.AsIdentifier(), refSymbol, token.Parent().AsExportSpecifier())
			if localSymbol == symbol {
				return true
			}
		}
	}
	return false
}

// GetReferencesToSymbolInFile returns all identifier nodes in the file that reference the given symbol.
func (c *Checker) GetReferencesToSymbolInFile(sourceFile *ast.SourceFile, symbol ast.Symbol) []ast.Node {
	return c.getReferencesToSymbolInFile(sourceFile, c.symbolId(symbol))
}

func (c *Checker) getReferencesToSymbolInFile(
	sourceFile *ast.SourceFile,
	symbol ast.SymbolId,
) []ast.Node {
	identifierText := c.symbolName(symbol)
	var result []ast.Node
	for _, token := range getPossibleSymbolReferenceNodes(sourceFile, identifierText.String(), sourceFile.AsNode()) {
		if !ast.IsIdentifier(token) {
			continue
		}
		id := token.AsIdentifier()
		if id.Text() != identifierText {
			continue
		}
		refSymbol := c.getSymbolAtLocationId(token)
		if refSymbol == symbol {
			result = append(result, token)
			continue
		}
		if !token.Parent().IsNil() && token.Parent().Kind() == ast.KindShorthandPropertyAssignment {
			shorthandSymbol := c.getShorthandAssignmentValueSymbol(token.Parent())
			if shorthandSymbol == symbol {
				result = append(result, token)
				continue
			}
		}
		if !token.Parent().IsNil() && ast.IsExportSpecifier(token.Parent()) {
			localSymbol := c.getLocalSymbolForExportSpecifier(token.AsIdentifier(), refSymbol, token.Parent().AsExportSpecifier())
			if localSymbol == symbol {
				result = append(result, token)
				continue
			}
		}
	}
	return result
}

func (c *Checker) getLocalSymbolForExportSpecifier(referenceLocation ast.Identifier, referenceSymbol ast.SymbolId, exportSpecifier ast.ExportSpecifier) ast.SymbolId {
	if isExportSpecifierAlias(referenceLocation, exportSpecifier) {
		if symbol := c.getExportSpecifierLocalTargetSymbol(exportSpecifier.AsNode()); symbol != 0 {
			return symbol
		}
	}
	return referenceSymbol
}

func isExportSpecifierAlias(referenceLocation ast.Identifier, exportSpecifier ast.ExportSpecifier) bool {
	debug.Assert(exportSpecifier.PropertyName() == referenceLocation.AsNode() || exportSpecifier.Name() == referenceLocation.AsNode(), "referenceLocation is not export specifier name or property name")
	propertyName := exportSpecifier.PropertyName()
	if !propertyName.IsNil() {
		// Given `export { foo as bar } [from "someModule"]`: It's an alias at `foo`, but at `bar` it's a new symbol.
		return propertyName == referenceLocation.AsNode()
	} else {
		// `export { foo } from "foo"` is a re-export.
		// `export { foo };` is not a re-export, it creates an alias for the local variable `foo`.
		return exportSpecifier.Parent().Parent().ModuleSpecifier().IsNil()
	}
}

func getPossibleSymbolReferenceNodes(sourceFile *ast.SourceFile, symbolName string, container ast.Node) []ast.Node {
	return core.MapNonNil(getPossibleSymbolReferencePositions(sourceFile, symbolName, container), func(pos int) ast.Node {
		if referenceLocation := astnav.GetTouchingPropertyName(sourceFile, pos); referenceLocation != sourceFile.AsNode() {
			return referenceLocation
		}
		return ast.Node{}
	})
}

func getPossibleSymbolReferencePositions(sourceFile *ast.SourceFile, symbolName string, container ast.Node) []int {
	positions := []int{}

	// TODO: Cache symbol existence for files to save text search
	// Also, need to make this work for unicode escapes.

	// Be resilient in the face of a symbol with no name or zero length name
	if symbolName == "" {
		return positions
	}

	text := sourceFile.Text()
	sourceLength := len(text)
	symbolNameLength := len(symbolName)

	if container.IsNil() {
		container = sourceFile.AsNode()
	}

	position := strings.Index(text[container.Pos():], symbolName)
	endPos := container.End()
	for position >= 0 && position < endPos {
		// We found a match.  Make sure it's not part of a larger word (i.e. the char
		// before and after it have to be a non-identifier char).
		endPosition := position + symbolNameLength

		if (position == 0 || !scanner.IsIdentifierPart(rune(text[position-1]))) &&
			(endPosition == sourceLength || !scanner.IsIdentifierPart(rune(text[endPosition]))) {
			// Found a real match.  Keep searching.
			positions = append(positions, position)
		}
		startIndex := position + symbolNameLength + 1
		if startIndex > len(text) {
			break
		}
		if foundIndex := strings.Index(text[startIndex:], symbolName); foundIndex != -1 {
			position = startIndex + foundIndex
		} else {
			break
		}
	}

	return positions
}

func (c *Checker) GetTypeArgumentConstraint(node ast.Node) *Type {
	if !ast.IsTypeNode(node) {
		return nil
	}
	return c.getTypeArgumentConstraint(node)
}

// getUninstantiatedSignatures gets generic signatures from the function's/constructor's type.
func (c *Checker) getUninstantiatedSignatures(node ast.Node) []*Signature {
	switch node.Kind() {
	case ast.KindCallExpression, ast.KindDecorator:
		return c.getSignaturesOfType(c.getTypeOfExpression(node.Expression()), SignatureKindCall)
	case ast.KindNewExpression:
		return c.getSignaturesOfType(c.getTypeOfExpression(node.Expression()), SignatureKindConstruct)
	case ast.KindJsxSelfClosingElement, ast.KindJsxOpeningElement:
		if isJsxIntrinsicTagName(node.TagName()) {
			return nil
		}
		return c.getSignaturesOfType(c.getTypeOfExpression(node.TagName()), SignatureKindCall)
	case ast.KindTaggedTemplateExpression:
		return c.getSignaturesOfType(c.getTypeOfExpression(node.AsTaggedTemplateExpression().Tag()), SignatureKindCall)
	case ast.KindBinaryExpression, ast.KindJsxOpeningFragment:
		return nil
	}
	return nil
}

func (c *Checker) getTypeParameterConstraintForPositionAcrossSignatures(signatures []*Signature, position int) *Type {
	var relevantConstraints []TypeId
	for _, signature := range signatures {
		if position >= len(signature.TypeParameters()) {
			continue
		}
		relevantTypeParameter := c.typ(signature.TypeParameters()[position])
		relevantConstraint := c.getConstraintOfTypeParameter(relevantTypeParameter)
		if relevantConstraint != nil {
			relevantConstraints = append(relevantConstraints, relevantConstraint.ref())
		}
	}
	return c.getUnionType(relevantConstraints)
}

func (c *Checker) getTypeArgumentConstraint(node ast.Node) *Type {
	var typeArgumentPosition int = -1
	if ast.HasTypeArguments(node.Parent()) {
		typeArgs := node.Parent().TypeArguments()
		for i, arg := range typeArgs {
			if arg == node {
				typeArgumentPosition = i
				break
			}
		}
	}

	if typeArgumentPosition >= 0 {
		// The node could be a type argument of a call, a `new` expression, a decorator, an
		// instantiation expression, or a generic type instantiation.

		if ast.IsCallLikeExpression(node.Parent()) {
			return c.getTypeParameterConstraintForPositionAcrossSignatures(
				c.getUninstantiatedSignatures(node.Parent()),
				typeArgumentPosition,
			)
		}

		if ast.IsDecorator(node.Parent().Parent()) {
			return c.getTypeParameterConstraintForPositionAcrossSignatures(
				c.getUninstantiatedSignatures(node.Parent().Parent()),
				typeArgumentPosition,
			)
		}

		if ast.IsExpressionWithTypeArguments(node.Parent()) && ast.IsExpressionStatement(node.Parent().Parent()) {
			uninstantiatedType := c.checkExpression(node.Parent().Expression())

			callConstraint := c.getTypeParameterConstraintForPositionAcrossSignatures(
				c.getSignaturesOfType(uninstantiatedType, SignatureKindCall),
				typeArgumentPosition,
			)
			constructConstraint := c.getTypeParameterConstraintForPositionAcrossSignatures(
				c.getSignaturesOfType(uninstantiatedType, SignatureKindConstruct),
				typeArgumentPosition,
			)

			// An instantiation expression instantiates both call and construct signatures, so
			// if both exist type arguments must be assignable to both constraints.
			if constructConstraint.flags&TypeFlagsNever != 0 {
				return callConstraint
			}
			if callConstraint.flags&TypeFlagsNever != 0 {
				return constructConstraint
			}
			return c.getIntersectionType([]TypeId{callConstraint.ref(), constructConstraint.ref()})
		}

		if ast.IsTypeReferenceType(node.Parent()) {
			typeParameters := c.getTypeParametersForTypeReferenceOrImport(node.Parent())
			if len(typeParameters) == 0 {
				return nil
			}
			if typeArgumentPosition >= len(typeParameters) {
				return nil
			}
			relevantTypeParameter := c.typ(typeParameters[typeArgumentPosition])
			constraint := c.getConstraintOfTypeParameter(relevantTypeParameter)
			if constraint != nil {
				return c.instantiateType(
					constraint,
					c.newTypeMapper(typeParameters, c.getEffectiveTypeArguments(node.Parent(), typeParameters)),
				)
			}
		}
	}
	return nil
}

func (c *Checker) IsTypeInvalidDueToUnionDiscriminant(contextualType *Type, obj ast.Node) bool {
	properties := obj.Properties()
	return core.Some(properties, func(property ast.Node) bool {
		var nameType *Type
		propertyName := property.Name()
		if !propertyName.IsNil() {
			if ast.IsJsxNamespacedName(propertyName) {
				nameType = c.getStringLiteralType(propertyName.Text())
			} else {
				nameType = c.getLiteralTypeFromPropertyName(propertyName)
			}
		}
		name := ast.EmptySymbolName
		if nameType != nil && isTypeUsableAsPropertyName(nameType) {
			name = getPropertyNameFromType(c, nameType)
		}
		var expected *Type
		if name != ast.EmptySymbolName {
			expected = c.getTypeOfPropertyOfType(contextualType, name)
		}
		return expected != nil && c.isLiteralType(expected) && !c.isTypeAssignableTo(c.getTypeOfNode(property), expected)
	})
}

// Unlike `getExportsOfModule`, this includes properties of an `export =` value.
func (c *Checker) getExportsAndPropertiesOfModule(moduleSymbol ast.SymbolId) []ast.SymbolId {
	exports := c.getExportsOfModuleAsArray(moduleSymbol)
	exportEquals := c.resolveExternalModuleSymbol(moduleSymbol, false /*dontResolveAlias*/)
	if exportEquals != moduleSymbol {
		t := c.getTypeOfSymbol(exportEquals)
		if c.shouldTreatPropertiesOfExternalModuleAsExports(t) {
			exports = append(exports, c.getPropertiesOfType(t)...)
		}
	}
	return exports
}

func (c *Checker) getExportsOfModuleAsArray(moduleSymbol ast.SymbolId) []ast.SymbolId {
	return c.symbolsToArray(c.getExportsOfModule(moduleSymbol))
}

// Returns all the properties of the Jsx.IntrinsicElements interface.
func (c *Checker) getJsxIntrinsicTagNamesAt(location ast.Node) []ast.SymbolId {
	intrinsics := c.getJsxType(JsxNames.IntrinsicElements, location)
	if intrinsics == nil {
		return nil
	}
	return c.getPropertiesOfTypeId(intrinsics)
}

func (c *Checker) GetContextualTypeForJsxAttribute(attribute ast.JsxAttributeLike) *Type {
	return c.getContextualTypeForJsxAttribute(attribute, ContextFlagsNone)
}

func (c *Checker) GetConstantValue(node ast.Node) any {
	if node.Kind() == ast.KindEnumMember {
		return c.getEnumMemberValue(node).Value
	}

	if c.symbolNodeLinks.get(node) == 0 {
		c.checkExpressionCached(node) // ensure cached resolved symbol is set
	}
	symbol := c.symbolNodeLinks.get(node)
	if symbol == 0 && ast.IsEntityNameExpression(node) {
		symbol = c.resolveEntityName(
			node,
			ast.SymbolFlagsValue,
			true,       /*ignoreErrors*/
			false,      /*dontResolveAlias*/
			ast.Node{}, /*location*/
		)
	}
	if symbol != 0 && c.symbolFlags(symbol)&ast.SymbolFlagsEnumMember != 0 {
		// inline property\index accesses only for const enums
		member := c.symbolValueDeclaration(symbol)
		if ast.IsEnumConst(member.Parent()) {
			return c.getEnumMemberValue(member).Value
		}
	}

	return nil
}

func (c *Checker) getResolvedSignatureWorker(node ast.Node, checkMode CheckMode, argumentCount int) (*Signature, []*Signature) {
	c.apparentArgumentCount = &argumentCount
	candidatesOutArray := &[]*Signature{}
	var res *Signature
	if !node.IsNil() && ast.IsParseTreeNode(node) {
		res = c.getResolvedSignature(node, candidatesOutArray, checkMode)
	}
	c.apparentArgumentCount = nil
	return res, *candidatesOutArray
}

func (c *Checker) GetCandidateSignaturesForStringLiteralCompletions(call ast.CallLikeExpression, editingArgument ast.Node) []*Signature {
	// first, get candidates when inference is blocked from the source node.
	candidates := c.runWithInferenceBlockedFromSourceNode(editingArgument, func() []*Signature {
		_, blockedInferenceCandidates := c.getResolvedSignatureWorker(call, CheckModeNormal, 0)
		return blockedInferenceCandidates
	})
	candidatesSet := collections.NewSetFromItems(candidates...)

	// next, get candidates where the source node is considered for inference.
	otherCandidates := c.runWithoutResolvedSignatureCaching(editingArgument, func() []*Signature {
		_, inferenceCandidates := c.getResolvedSignatureWorker(call, CheckModeNormal, 0)
		return inferenceCandidates
	})

	for _, candidate := range otherCandidates {
		if candidatesSet.Has(candidate) {
			continue
		}
		candidates = append(candidates, candidate)
	}

	return candidates
}

// GetTypeAtPosition returns the type of a parameter at a given index in a signature.
func (c *Checker) GetTypeAtPosition(s *Signature, pos int) *Type {
	return c.getTypeAtPosition(s, pos)
}

func (c *Checker) GetTypeParameterAtPosition(s *Signature, pos int) *Type {
	t := c.getTypeAtPosition(s, pos)
	if t.IsIndex() && isThisTypeParameter(t.AsIndexType().target(c)) {
		constraint := c.getBaseConstraintOfType(t.AsIndexType().target(c))
		if constraint != nil {
			return c.getIndexType(constraint)
		}
	}
	return t
}

// GetContextualTypeForArrayLiteralAtPosition returns the contextual type for an element at the given position
// in an array with the given contextual type.
func (c *Checker) GetContextualTypeForArrayLiteralAtPosition(contextualArrayType *Type, arrayLiteral ast.Node, position int) *Type {
	if contextualArrayType == nil {
		return nil
	}
	firstSpreadIndex, lastSpreadIndex := -1, -1
	elementIndex := 0
	elements := arrayLiteral.Elements()
	for i, elem := range elements {
		if elem.Pos() < position {
			elementIndex++
		}
		if ast.IsSpreadElement(elem) {
			if firstSpreadIndex == -1 {
				firstSpreadIndex = i
			}
			lastSpreadIndex = i
		}
	}
	// The array may be incomplete, so we don't know its final length.
	return c.getContextualTypeForElementExpression(
		contextualArrayType,
		elementIndex,
		-1, /*length*/
		firstSpreadIndex,
		lastSpreadIndex,
	)
}

var knownGenericTypeNames = map[ast.SymbolName]struct{}{
	ast.SymbolNameArray:            {},
	ast.SymbolNameArrayLike:        {},
	ast.SymbolNameReadonlyArray:    {},
	ast.SymbolNamePromise:          {},
	ast.SymbolNamePromiseLike:      {},
	ast.SymbolNameIterable:         {},
	ast.SymbolNameIterableIterator: {},
	ast.SymbolNameAsyncIterable:    {},
	ast.SymbolNameSet:              {},
	ast.SymbolNameWeakSet:          {},
	ast.SymbolNameReadonlySet:      {},
	ast.SymbolNameMap:              {},
	ast.SymbolNameWeakMap:          {},
	ast.SymbolNameReadonlyMap:      {},
	ast.SymbolNamePartial:          {},
	ast.SymbolNameRequired:         {},
	ast.SymbolNameReadonly:         {},
	ast.SymbolNamePick:             {},
	ast.SymbolNameOmit:             {},
	ast.SymbolNameNonNullable:      {},
}

func isKnownGenericTypeName(name ast.SymbolName) bool {
	_, exists := knownGenericTypeNames[name]
	return exists
}

func (c *Checker) GetFirstTypeArgumentFromKnownType(t *Type) *Type {
	if t.objectFlags&ObjectFlagsReference != 0 && c.symbolOfType(t) != 0 && isKnownGenericTypeName(c.symbolName(c.symbolOfType(t))) {
		symbol := c.getGlobalSymbol(c.symbolName(c.symbolOfType(t)), ast.SymbolFlagsType, nil)
		if symbol != 0 && symbol == c.symbolOfType(t.Target(c)) {
			return c.typ(core.FirstOrNil(c.getTypeArguments(t)))
		}
	}
	if t.Alias(c) != nil && isKnownGenericTypeName(c.symbolName(t.Alias(c).symbol)) {
		symbol := c.getGlobalSymbol(c.symbolName(t.Alias(c).symbol), ast.SymbolFlagsType, nil)
		if symbol != 0 && symbol == t.Alias(c).symbol {
			return c.typ(core.FirstOrNil(t.Alias(c).typeArguments))
		}
	}
	return nil
}

// Gets all symbols for one property. Does not get symbols for every property.
func (c *Checker) getPropertySymbolsFromContextualType(node ast.Node, contextualType *Type, unionSymbolOk bool) []ast.SymbolId {
	name := ast.GetTextOfPropertyName(node.Name())
	if name == "" {
		return nil
	}
	if contextualType.flags&TypeFlagsUnion == 0 {
		if symbol := c.getPropertyOfType(contextualType, ast.MakeSymbolName(name)); symbol != 0 {
			return []ast.SymbolId{symbol}
		}
		return nil
	}
	filteredTypes := contextualType.Types(c)
	if ast.IsObjectLiteralExpression(node.Parent()) || ast.IsJsxAttributes(node.Parent()) {
		filteredTypes = c.typesFilter(filteredTypes, func(t *Type) bool {
			return !c.IsTypeInvalidDueToUnionDiscriminant(t, node.Parent())
		})
	}
	discriminatedPropertySymbols := typesMapNonNilTo(c, filteredTypes, func(t *Type) ast.SymbolId {
		return c.getPropertyOfType(t, ast.MakeSymbolName(name))
	})
	if unionSymbolOk && (len(discriminatedPropertySymbols) == 0 || len(discriminatedPropertySymbols) == len(contextualType.Types(c))) {
		if symbol := c.getPropertyOfType(contextualType, ast.MakeSymbolName(name)); symbol != 0 {
			return []ast.SymbolId{symbol}
		}
	}
	if len(filteredTypes) == 0 && len(discriminatedPropertySymbols) == 0 {
		// Bad discriminant -- do again without discriminating
		return typesMapNonNilTo(c, contextualType.Types(c), func(t *Type) ast.SymbolId {
			return c.getPropertyOfType(t, ast.MakeSymbolName(name))
		})
	}
	// by eliminating duplicates we might even end up with a single symbol
	// that helps with displaying better quick infos on properties of union types
	return core.Deduplicate(discriminatedPropertySymbols)
}

// Gets the property symbol corresponding to the property in destructuring assignment
// 'property1' from
//
//	for ( { property1: a } of elems) {
//	}
//
// 'property1' at location 'a' from:
//
//	[a] = [ property1, property2 ]
func (c *Checker) getPropertySymbolOfDestructuringAssignment(location ast.Node) ast.SymbolId {
	if ast.IsArrayLiteralOrObjectLiteralDestructuringPattern(location.Parent().Parent()) {
		// Get the type of the object or array literal and then look for property of given name in the type
		if typeOfObjectLiteral := c.getTypeOfAssignmentPattern(location.Parent().Parent()); typeOfObjectLiteral != nil {
			return c.getPropertyOfType(typeOfObjectLiteral, location.TextSymbolName())
		}
	}
	return 0
}

// Gets the type of object literal or array literal of destructuring assignment.
// { a } from
//
//	for ( { a } of elems) {
//	}
//
// [ a ] from
//
//	[a] = [ some array ...]
func (c *Checker) getTypeOfAssignmentPattern(expr ast.Node) *Type {
	// If this is from "for of"
	//     for ( { a } of elems) {
	//     }
	if ast.IsForOfStatement(expr.Parent()) {
		iteratedType := c.checkRightHandSideOfForOf(expr.Parent())
		return c.checkDestructuringAssignment(expr, core.OrElse(iteratedType, c.errorType), CheckModeNormal, false)
	}
	// If this is from "for" initializer
	//     for ({a } = elems[0];.....) { }
	if ast.IsBinaryExpression(expr.Parent()) {
		iteratedType := c.getTypeOfExpression(expr.Parent().AsBinaryExpression().Right())
		return c.checkDestructuringAssignment(expr, core.OrElse(iteratedType, c.errorType), CheckModeNormal, false)
	}
	// If this is from nested object binding pattern
	//     for ({ skills: { primary, secondary } } = multiRobot, i = 0; i < 1; i++) {
	if ast.IsPropertyAssignment(expr.Parent()) {
		node := expr.Parent().Parent()
		typeOfParentObjectLiteral := core.OrElse(c.getTypeOfAssignmentPattern(node), c.errorType)
		propertyIndex := slices.Index(node.Properties(), expr.Parent())
		return c.checkObjectLiteralDestructuringPropertyAssignment(node, typeOfParentObjectLiteral, propertyIndex, ast.NodeList{}, false)
	}
	// Array literal assignment - array destructuring pattern
	node := expr.Parent()
	//    [{ property1: p1, property2 }] = elems;
	typeOfArrayLiteral := core.OrElse(c.getTypeOfAssignmentPattern(node), c.errorType)
	elementType := core.OrElse(c.checkIteratedTypeOrElementType(IterationUseDestructuring, typeOfArrayLiteral, c.undefinedType, expr.Parent()), c.errorType)
	return c.checkArrayLiteralDestructuringElementAssignment(node, typeOfArrayLiteral, slices.Index(node.Elements(), expr), elementType, CheckModeNormal)
}

func (c *Checker) GetSignatureFromDeclaration(node ast.Node) *Signature {
	return c.getSignatureFromDeclaration(node)
}

// IsLibSymbolForHoverVerbosity returns true if a symbol is declared in a lib file.
func (c *Checker) IsLibSymbolForHoverVerbosity(symbol ast.Symbol) bool {
	return c.isLibSymbolForHoverVerbosity(c.symbolId(symbol))
}

func (c *Checker) isLibSymbolForHoverVerbosity(symbol ast.SymbolId) bool {
	if symbol == 0 {
		return false
	}
	for _, decl := range c.symbolDeclarations(symbol).All() {
		sf := ast.GetSourceFileOfNode(decl)
		if sf != nil && c.program.IsSourceFileDefaultLibrary(sf.PathKey()) {
			return true
		}
	}
	return false
}

// IsLibTypeForHoverVerbosity returns true if a type is declared in a lib file.
// Don't expand types like Array or Promise, instead treating them as opaque.
func (c *Checker) IsLibTypeForHoverVerbosity(t *Type) bool {
	var symbol ast.SymbolId
	if t.objectFlags&ObjectFlagsReference != 0 {
		symbol = c.symbolOfType(t.Target(c))
	} else {
		symbol = c.symbolOfType(t)
	}
	if c.isLibSymbolForHoverVerbosity(symbol) {
		return true
	}
	return c.isTupleType(t)
}

func (c *Checker) GetSymbolsInScope(location ast.Node, meaning ast.SymbolFlags) []ast.Symbol {
	return c.symbolsOf(c.getSymbolsInScopeId(location, meaning))
}

func (c *Checker) GetExportsOfModule(symbol ast.Symbol) []ast.Symbol {
	return c.symbolsOf(c.getExportsOfModuleId(c.symbolId(symbol)))
}

func (c *Checker) IsValidPropertyAccessForCompletions(node ast.Node, t *Type, property ast.Symbol) bool {
	return c.isValidPropertyAccessForCompletionsId(node, t, c.symbolId(property))
}

func (c *Checker) GetAllPossiblePropertiesOfTypes(types []*Type) []ast.Symbol {
	return c.symbolsOf(c.getAllPossiblePropertiesOfTypes(typeIdsOf(types)))
}

func (c *Checker) IsUnknownSymbol(symbol ast.Symbol) bool {
	return c.isUnknownSymbol(c.symbolId(symbol))
}

func (c *Checker) IsUndefinedSymbol(symbol ast.Symbol) bool {
	return c.isUndefinedSymbol(c.symbolId(symbol))
}

func (c *Checker) IsArgumentsSymbol(symbol ast.Symbol) bool {
	return c.isArgumentsSymbol(c.symbolId(symbol))
}

func (c *Checker) GetApparentProperties(t *Type) []ast.Symbol {
	return c.symbolsOf(c.getApparentProperties(t))
}

func (c *Checker) TryGetMemberInModuleExportsAndProperties(memberName ast.SymbolName, moduleSymbol ast.Symbol) ast.Symbol {
	return c.symbolOf(c.tryGetMemberInModuleExportsAndProperties(memberName, c.symbolId(moduleSymbol)))
}

func (c *Checker) SkipAlias(symbol ast.Symbol) ast.Symbol {
	return c.symbolOf(c.skipAlias(c.symbolId(symbol)))
}

func (c *Checker) GetRootSymbols(symbol ast.Symbol) []ast.Symbol {
	return c.symbolsOf(c.getRootSymbols(c.symbolId(symbol)))
}

func (c *Checker) GetMappedTypeSymbolOfProperty(symbol ast.Symbol) ast.Symbol {
	return c.symbolOf(c.getMappedTypeSymbolOfProperty(c.symbolId(symbol)))
}

func (c *Checker) GetExportSymbolOfSymbol(symbol ast.Symbol) ast.Symbol {
	return c.symbolOf(c.getExportSymbolOfSymbol(c.symbolId(symbol)))
}

func (c *Checker) GetExportSpecifierLocalTargetSymbol(node ast.Node) ast.Symbol {
	return c.symbolOf(c.getExportSpecifierLocalTargetSymbol(node))
}

func (c *Checker) GetShorthandAssignmentValueSymbol(location ast.Node) ast.Symbol {
	return c.symbolOf(c.getShorthandAssignmentValueSymbol(location))
}

func (c *Checker) GetSymbolsOfParameterPropertyDeclaration(parameter ast.Node /*ParameterPropertyDeclaration*/, parameterName ast.SymbolName) (ast.Symbol, ast.Symbol) {
	r0, r1 := c.getSymbolsOfParameterPropertyDeclaration(parameter, parameterName)
	return c.symbolOf(r0), c.symbolOf(r1)
}

func (c *Checker) GetExportsAndPropertiesOfModule(moduleSymbol ast.Symbol) []ast.Symbol {
	return c.symbolsOf(c.getExportsAndPropertiesOfModule(c.symbolId(moduleSymbol)))
}

func (c *Checker) GetJsxIntrinsicTagNamesAt(location ast.Node) []ast.Symbol {
	return c.symbolsOf(c.getJsxIntrinsicTagNamesAt(location))
}

func (c *Checker) GetPropertySymbolsFromContextualType(node ast.Node, contextualType *Type, unionSymbolOk bool) []ast.Symbol {
	return c.symbolsOf(c.getPropertySymbolsFromContextualType(node, contextualType, unionSymbolOk))
}

func (c *Checker) GetPropertySymbolOfDestructuringAssignment(location ast.Node) ast.Symbol {
	return c.symbolOf(c.getPropertySymbolOfDestructuringAssignment(location))
}
