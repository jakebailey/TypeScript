package checker

import (
	"cmp"
	"slices"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/collections"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
)

func (c *Checker) IsTypeSymbolAccessible(typeSymbol ast.Symbol, enclosingDeclaration ast.Node) bool {
	return c.isTypeSymbolAccessible(c.symbolId(typeSymbol), enclosingDeclaration)
}

func (c *Checker) isTypeSymbolAccessible(typeSymbol ast.SymbolId, enclosingDeclaration ast.Node) bool {
	access := c.isSymbolAccessibleWorker(typeSymbol, enclosingDeclaration, ast.SymbolFlagsType /*shouldComputeAliasesToMakeVisible*/, false /*allowModules*/, true)
	return access.Accessibility == printer.SymbolAccessibilityAccessible
}

func (c *Checker) IsValueSymbolAccessible(symbol ast.Symbol, enclosingDeclaration ast.Node) bool {
	return c.isValueSymbolAccessible(c.symbolId(symbol), enclosingDeclaration)
}

func (c *Checker) isValueSymbolAccessible(symbol ast.SymbolId, enclosingDeclaration ast.Node) bool {
	access := c.isSymbolAccessibleWorker(symbol, enclosingDeclaration, ast.SymbolFlagsValue /*shouldComputeAliasesToMakeVisible*/, false /*allowModules*/, true)
	return access.Accessibility == printer.SymbolAccessibilityAccessible
}

func (c *Checker) IsSymbolAccessibleByFlags(symbol ast.Symbol, enclosingDeclaration ast.Node, flags ast.SymbolFlags) bool {
	return c.isSymbolAccessibleByFlags(c.symbolId(symbol), enclosingDeclaration, flags)
}

func (c *Checker) isSymbolAccessibleByFlags(symbol ast.SymbolId, enclosingDeclaration ast.Node, flags ast.SymbolFlags) bool {
	access := c.isSymbolAccessibleWorker(symbol, enclosingDeclaration, flags /*shouldComputeAliasesToMakeVisible*/, false /*allowModules*/, false) // TODO: Strada bug? Why is this allowModules: false?
	return access.Accessibility == printer.SymbolAccessibilityAccessible
}

func (c *Checker) IsAnySymbolAccessible(symbols []ast.Symbol, enclosingDeclaration ast.Node, initialSymbol ast.Symbol, meaning ast.SymbolFlags, shouldComputeAliasesToMakeVisible bool, allowModules bool) *printer.SymbolAccessibilityResult {
	return c.isAnySymbolAccessible(c.symbolIdsOf(symbols), enclosingDeclaration, c.symbolId(initialSymbol), meaning, shouldComputeAliasesToMakeVisible, allowModules)
}

func (c *Checker) isAnySymbolAccessible(symbols []ast.SymbolId, enclosingDeclaration ast.Node, initialSymbol ast.SymbolId, meaning ast.SymbolFlags, shouldComputeAliasesToMakeVisible bool, allowModules bool) *printer.SymbolAccessibilityResult {
	if len(symbols) == 0 {
		return nil
	}

	var hadAccessibleChain ast.SymbolId
	earlyModuleBail := false
	for _, symbol := range symbols {
		// Symbol is accessible if it by itself is accessible
		accessibleSymbolChain := c.getAccessibleSymbolChain(symbol, enclosingDeclaration, meaning /*useOnlyExternalAliasing*/, false)
		if len(accessibleSymbolChain) > 0 {
			hadAccessibleChain = symbol
			hasAccessibleDeclarations := c.hasVisibleDeclarations(accessibleSymbolChain[0], shouldComputeAliasesToMakeVisible)
			if hasAccessibleDeclarations != nil {
				return hasAccessibleDeclarations
			}
		}
		if allowModules {
			if c.symbolDeclarations(symbol).Some(hasNonGlobalAugmentationExternalModuleSymbol) {
				if shouldComputeAliasesToMakeVisible {
					earlyModuleBail = true
					// Generally speaking, we want to use the aliases that already exist to refer to a module, if present
					// In order to do so, we need to find those aliases in order to retain them in declaration emit; so
					// if we are in declaration emit, we cannot use the fast path for module visibility until we've exhausted
					// all other visibility options (in order to capture the possible aliases used to reference the module)
					continue
				}
				// Any meaning of a module symbol is always accessible via an `import` type
				return &printer.SymbolAccessibilityResult{
					Accessibility: printer.SymbolAccessibilityAccessible,
				}
			}
		}

		// If we haven't got the accessible symbol, it doesn't mean the symbol is actually inaccessible.
		// It could be a qualified symbol and hence verify the path
		// e.g.:
		// module m {
		//     export class c {
		//     }
		// }
		// const x: typeof m.c
		// In the above example when we start with checking if typeof m.c symbol is accessible,
		// we are going to see if c can be accessed in scope directly.
		// But it can't, hence the accessible is going to be undefined, but that doesn't mean m.c is inaccessible
		// It is accessible if the parent m is accessible because then m.c can be accessed through qualification

		containers := c.getContainersOfSymbol(symbol, enclosingDeclaration, meaning)
		nextMeaning := meaning
		if initialSymbol == symbol {
			nextMeaning = getQualifiedLeftMeaning(meaning)
		}
		parentResult := c.isAnySymbolAccessible(containers, enclosingDeclaration, initialSymbol, nextMeaning, shouldComputeAliasesToMakeVisible, allowModules)
		if parentResult != nil {
			return parentResult
		}
	}

	if earlyModuleBail {
		return &printer.SymbolAccessibilityResult{
			Accessibility: printer.SymbolAccessibilityAccessible,
		}
	}

	if hadAccessibleChain != 0 {
		var moduleName string
		if hadAccessibleChain != initialSymbol {
			moduleName = c.symbolToStringEx(hadAccessibleChain, enclosingDeclaration, ast.SymbolFlagsNamespace, SymbolFormatFlagsAllowAnyNodeKind)
		}
		return &printer.SymbolAccessibilityResult{
			Accessibility:   printer.SymbolAccessibilityNotAccessible,
			ErrorSymbolName: c.symbolToStringEx(initialSymbol, enclosingDeclaration, meaning, SymbolFormatFlagsAllowAnyNodeKind),
			ErrorModuleName: moduleName,
		}
	}
	return nil
}

func hasNonGlobalAugmentationExternalModuleSymbol(declaration ast.Node) bool {
	return ast.IsModuleWithStringLiteralName(declaration) || (declaration.Kind() == ast.KindSourceFile && ast.IsExternalOrCommonJSModule(declaration.AsSourceFile()))
}

func getQualifiedLeftMeaning(rightMeaning ast.SymbolFlags) ast.SymbolFlags {
	// If we are looking in value space, the parent meaning is value, other wise it is namespace
	if rightMeaning == ast.SymbolFlagsValue {
		return ast.SymbolFlagsValue
	}
	return ast.SymbolFlagsNamespace
}

func (c *Checker) getWithAlternativeContainers(container ast.SymbolId, symbol ast.SymbolId, enclosingDeclaration ast.Node, meaning ast.SymbolFlags) []ast.SymbolId {
	additionalContainers := core.MapNonNil(c.symbolDeclarations(container).Nodes(), func(d ast.Node) ast.SymbolId {
		return c.getFileSymbolIfFileSymbolExportEqualsContainer(d, container)
	})
	var reexportContainers []ast.SymbolId
	if !enclosingDeclaration.IsNil() {
		reexportContainers = c.getAlternativeContainingModules(symbol, enclosingDeclaration)
	}
	objectLiteralContainer := c.getVariableDeclarationOfObjectLiteral(container, meaning)
	leftMeaning := getQualifiedLeftMeaning(meaning)
	if !enclosingDeclaration.IsNil() &&
		c.symbolFlags(container)&leftMeaning != 0 &&
		len(c.getAccessibleSymbolChain(container, enclosingDeclaration, ast.SymbolFlagsNamespace /*useOnlyExternalAliasing*/, false)) > 0 {
		// This order expresses a preference for the real container if it is in scope
		res := append(append([]ast.SymbolId{container}, additionalContainers...), reexportContainers...)
		if objectLiteralContainer != 0 {
			res = append(res, objectLiteralContainer)
		}
		return res
	}
	// we potentially have a symbol which is a member of the instance side of something - look for a variable in scope with the container's type
	// which may be acting like a namespace (eg, `Symbol` acts like a namespace when looking up `Symbol.toStringTag`)
	var variableMatches []ast.SymbolId
	if (meaning == ast.SymbolFlagsValue &&
		c.symbolFlags(container)&leftMeaning == 0) &&
		c.symbolFlags(container)&ast.SymbolFlagsType != 0 &&
		c.getDeclaredTypeOfSymbol(container).flags&TypeFlagsObject != 0 {
		c.someSymbolTableInScope(enclosingDeclaration, func(t ast.SymbolTable, _ symbolTableID, _ bool, _ bool, _ ast.Node) bool {
			found := false
			for _, s := range t.AllIds(c.symbols) {
				if c.symbolFlags(s)&leftMeaning != 0 && c.getTypeOfSymbol(s) == c.getDeclaredTypeOfSymbol(container) {
					variableMatches = append(variableMatches, s)
					found = true
				}
			}
			return found
		})
		c.sortSymbols(variableMatches)
	}

	var res []ast.SymbolId
	res = append(res, variableMatches...)
	res = append(res, additionalContainers...)
	res = append(res, container)
	if objectLiteralContainer != 0 {
		res = append(res, objectLiteralContainer)
	}
	res = append(res, reexportContainers...)
	return res
}

func (c *Checker) getAlternativeContainingModules(symbol ast.SymbolId, enclosingDeclaration ast.Node) []ast.SymbolId {
	if enclosingDeclaration.IsNil() {
		return nil
	}
	containingFile := ast.GetSourceFileOfNode(enclosingDeclaration)
	id := ast.GetNodeId(containingFile.AsNode())
	links := c.symbolContainerLinks.Get(symbol)
	if links.extendedContainersByFile == nil {
		links.extendedContainersByFile = make(map[ast.NodeId][]ast.SymbolId)
	}
	existing, ok := links.extendedContainersByFile[id]
	if ok && existing != nil {
		return existing
	}
	var results []ast.SymbolId
	if len(containingFile.Imports()) > 0 {
		// Try to make an import using an import already in the enclosing file, if possible
		for _, importRef := range containingFile.Imports() {
			if ast.NodeIsSynthesized(importRef) {
				// Synthetic names can't be resolved by `resolveExternalModuleName` - they'll cause a debug assert if they error
				continue
			}
			resolvedModule := c.resolveExternalModuleName(enclosingDeclaration, importRef /*ignoreErrors*/, true, c.getImportAttributesTypeForModuleSpecifier(importRef))
			if resolvedModule == 0 {
				continue
			}
			ref := c.getAliasForSymbolInContainer(resolvedModule, symbol)
			if ref == 0 {
				continue
			}
			results = append(results, resolvedModule)
		}
		if len(results) > 0 {
			links.extendedContainersByFile[id] = results
			return results
		}
	}

	if links.extendedContainers != nil {
		return *links.extendedContainers
	}
	// No results from files already being imported by this file - expand search (not location-specific, so cached)
	results = c.getExternalModuleContainers(symbol)
	links.extendedContainers = &results
	return results
}

type externalModuleContainerIndex struct {
	complete           bool
	containersByTarget map[ast.SymbolId][]ast.SymbolId
	moduleOrder        map[ast.SymbolId]int
}

func (index *externalModuleContainerIndex) add(target ast.SymbolId, container ast.SymbolId) {
	// Modules are indexed one at a time, so a repeat of this container is always the last entry.
	if existing := index.containersByTarget[target]; len(existing) == 0 || existing[len(existing)-1] != container {
		index.containersByTarget[target] = append(existing, container)
	}
}

func (c *Checker) getExternalModuleContainers(symbol ast.SymbolId) []ast.SymbolId {
	if c.externalModuleContainers == nil {
		c.buildExternalModuleContainerIndex()
	}
	index := c.externalModuleContainers
	if !index.complete {
		// Re-entered from an alias resolved while building the index; answer this query without it.
		return c.scanExternalModuleContainers(symbol)
	}
	containers := index.containersByTarget[c.getResolvedTarget(symbol)]
	parent := c.getParentOfSymbol(symbol)
	parentOrder, parentIsModule := index.moduleOrder[parent]
	if !parentIsModule {
		return containers
	}
	// The parent module contains the symbol even when the symbol is absent from its exports.
	if at, found := slices.BinarySearchFunc(containers, parentOrder, func(container ast.SymbolId, order int) int {
		return cmp.Compare(index.moduleOrder[container], order)
	}); !found {
		return slices.Insert(slices.Clone(containers), at, parent)
	}
	return containers
}

func (c *Checker) buildExternalModuleContainerIndex() {
	index := &externalModuleContainerIndex{
		containersByTarget: make(map[ast.SymbolId][]ast.SymbolId),
		moduleOrder:        make(map[ast.SymbolId]int, len(c.program.SourceFiles())),
	}
	c.externalModuleContainers = index
	for _, file := range c.program.SourceFiles() {
		if !ast.IsExternalModule(file) {
			continue
		}
		container := c.getSymbolOfDeclaration(file.AsNode())
		index.moduleOrder[container] = len(index.moduleOrder)
		for _, exported := range c.getExportsOfSymbol(container).AllIds(c.symbols) {
			index.add(c.getResolvedTarget(exported), container)
		}
		if exportEquals := c.symbolExports(container).GetId(c.symbols, ast.InternalSymbolNameExportEquals); exportEquals != 0 {
			index.add(c.getResolvedTarget(exportEquals), container)
		}
	}
	index.complete = true
}

func (c *Checker) scanExternalModuleContainers(symbol ast.SymbolId) []ast.SymbolId {
	var containers []ast.SymbolId
	for _, file := range c.program.SourceFiles() {
		if !ast.IsExternalModule(file) {
			continue
		}
		if container := c.getSymbolOfDeclaration(file.AsNode()); c.getAliasForSymbolInContainer(container, symbol) != 0 {
			containers = append(containers, container)
		}
	}
	return containers
}

func (c *Checker) getVariableDeclarationOfObjectLiteral(symbol ast.SymbolId, meaning ast.SymbolFlags) ast.SymbolId {
	// If we're trying to reference some object literal in, eg `var a = { x: 1 }`, the symbol for the literal, `__object`, is distinct
	// from the symbol of the declaration it is being assigned to. Since we can use the declaration to refer to the literal, however,
	// we'd like to make that connection here - potentially causing us to paint the declaration's visibility, and therefore the literal.
	if meaning&ast.SymbolFlagsValue == 0 {
		return 0
	}
	if c.symbolDeclarations(symbol).Len() == 0 {
		return 0
	}
	firstDecl := c.symbolDeclarations(symbol).At(0)
	if firstDecl.Parent().IsNil() {
		return 0
	}
	if !ast.IsVariableDeclaration(firstDecl.Parent()) {
		return 0
	}
	if ast.IsObjectLiteralExpression(firstDecl) && firstDecl == firstDecl.Parent().Initializer() || ast.IsTypeLiteralNode(firstDecl) && firstDecl == firstDecl.Parent().Type() {
		return c.getSymbolOfDeclaration(firstDecl.Parent())
	}
	return 0
}

func hasExternalModuleSymbol(declaration ast.Node) bool {
	return ast.IsAmbientModule(declaration) || (declaration.Kind() == ast.KindSourceFile && ast.IsExternalOrCommonJSModule(declaration.AsSourceFile()))
}

func (c *Checker) getExternalModuleContainer(declaration ast.Node) ast.SymbolId {
	node := ast.FindAncestor(declaration, hasExternalModuleSymbol)
	if node.IsNil() {
		return 0
	}
	return c.getSymbolOfDeclaration(node)
}

func (c *Checker) getFileSymbolIfFileSymbolExportEqualsContainer(d ast.Node, container ast.SymbolId) ast.SymbolId {
	fileSymbol := c.getExternalModuleContainer(d)
	if fileSymbol == 0 || c.symbolExports(fileSymbol).IsNil() {
		return 0
	}
	exported, ok := c.symbolExports(fileSymbol).LookupId(c.symbols, ast.InternalSymbolNameExportEquals)
	if !ok || exported == 0 {
		return 0
	}
	if c.getSymbolIfSameReference(exported, container) != 0 {
		return fileSymbol
	}
	return 0
}

/**
* Attempts to find the symbol corresponding to the container a symbol is in - usually this
* is just its' `.parent`, but for locals, this value is `undefined`
 */
func (c *Checker) getContainersOfSymbol(symbol ast.SymbolId, enclosingDeclaration ast.Node, meaning ast.SymbolFlags) []ast.SymbolId {
	container := c.getParentOfSymbol(symbol)
	// Type parameters end up in the `members` lists but are not externally visible
	if container != 0 && (c.symbolFlags(symbol)&ast.SymbolFlagsTypeParameter == 0) {
		return c.getWithAlternativeContainers(container, symbol, enclosingDeclaration, meaning)
	}
	var candidates []ast.SymbolId
	for _, d := range c.symbolDeclarations(symbol).All() {
		if !ast.IsAmbientModule(d) && !d.Parent().IsNil() {
			// direct children of a module
			if hasNonGlobalAugmentationExternalModuleSymbol(d.Parent()) {
				sym := c.getSymbolOfDeclaration(d.Parent())
				if sym != 0 && !slices.Contains(candidates, sym) {
					candidates = append(candidates, sym)
				}
				continue
			}
			// export ='d member of an ambient module
			if ast.IsModuleBlock(d.Parent()) && !d.Parent().Parent().IsNil() && c.resolveExternalModuleSymbol(c.getSymbolOfDeclaration(d.Parent().Parent()), false) == symbol {
				sym := c.getSymbolOfDeclaration(d.Parent().Parent())
				if sym != 0 && !slices.Contains(candidates, sym) {
					candidates = append(candidates, sym)
				}
				continue
			}
		}
		if ast.IsClassExpression(d) && ast.IsBinaryExpression(d.Parent()) && d.Parent().AsBinaryExpression().OperatorToken().Kind() == ast.KindEqualsToken && ast.IsAccessExpression(d.Parent().AsBinaryExpression().Left()) && ast.IsEntityNameExpression(d.Parent().AsBinaryExpression().Left().Expression()) {
			if ast.IsModuleExportsAccessExpression(d.Parent().AsBinaryExpression().Left()) || ast.IsExportsIdentifier(d.Parent().AsBinaryExpression().Left().Expression()) {
				sym := c.getSymbolOfDeclaration(ast.GetSourceFileOfNode(d).AsNode())
				if sym != 0 && !slices.Contains(candidates, sym) {
					candidates = append(candidates, sym)
				}
				continue
			}
			c.checkExpressionCached(d.Parent().AsBinaryExpression().Left().Expression())
			sym := c.symbolNodeLinks.get(d.Parent().AsBinaryExpression().Left().Expression())
			if sym != 0 && !slices.Contains(candidates, sym) {
				candidates = append(candidates, sym)
			}
			continue
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	var bestContainers []ast.SymbolId
	var alternativeContainers []ast.SymbolId
	for _, container := range candidates {
		if c.getAliasForSymbolInContainer(container, symbol) == 0 {
			continue
		}
		allAlts := c.getWithAlternativeContainers(container, symbol, enclosingDeclaration, meaning)
		if len(allAlts) == 0 {
			continue
		}
		bestContainers = append(bestContainers, allAlts[0])
		alternativeContainers = append(alternativeContainers, allAlts[1:]...)
	}
	return append(bestContainers, alternativeContainers...)
}

func (c *Checker) getAliasForSymbolInContainer(container ast.SymbolId, symbol ast.SymbolId) ast.SymbolId {
	if container == c.getParentOfSymbol(symbol) {
		// fast path, `symbol` is either already the alias or isn't aliased
		return symbol
	}
	// Check if container is a thing with an `export=` which points directly at `symbol`, and if so, return
	// the container itself as the alias for the symbol
	if !c.symbolExports(container).IsNil() {
		exportEquals, ok := c.symbolExports(container).LookupId(c.symbols, ast.InternalSymbolNameExportEquals)
		if ok && exportEquals != 0 && c.getSymbolIfSameReference(exportEquals, symbol) != 0 {
			return container
		}
	}
	exports := c.getExportsOfSymbol(container)
	quick, ok := exports.LookupId(c.symbols, c.symbolName(symbol))
	if ok && quick != 0 && c.getSymbolIfSameReference(quick, symbol) != 0 {
		return quick
	}
	var candidates []ast.SymbolId
	for _, exported := range exports.AllIds(c.symbols) {
		if c.getSymbolIfSameReference(exported, symbol) != 0 {
			candidates = append(candidates, exported)
		}
	}
	if len(candidates) > 0 {
		c.sortSymbols(candidates) // _must_ sort exports for stable results - symbol table is randomly iterated
		return candidates[0]
	}
	return 0
}

func (c *Checker) getAccessibleSymbolChain(
	symbol ast.SymbolId,
	enclosingDeclaration ast.Node,
	meaning ast.SymbolFlags,
	useOnlyExternalAliasing bool,
) []ast.SymbolId {
	return c.getAccessibleSymbolChainEx(accessibleSymbolChainContext{symbol, enclosingDeclaration, meaning, useOnlyExternalAliasing, make(map[ast.SymbolSerial]map[symbolTableID]struct{})})
}

func (c *Checker) getAccessibleSymbolChainId(
	symbol ast.SymbolId,
	enclosingDeclaration ast.Node,
	meaning ast.SymbolFlags,
	useOnlyExternalAliasing bool,
) []ast.SymbolId {
	return c.getAccessibleSymbolChain(symbol, enclosingDeclaration, meaning, useOnlyExternalAliasing)
}

type accessibleSymbolChainContext struct {
	symbol                  ast.SymbolId
	enclosingDeclaration    ast.Node
	meaning                 ast.SymbolFlags
	useOnlyExternalAliasing bool
	visitedSymbolTablesMap  map[ast.SymbolSerial]map[symbolTableID]struct{}
}

// symbolTableID uniquely identifies a symbol table by encoding its source.
// The high 3 bits encode the kind, and the remaining bits encode the
// NodeId or SymbolId of the source.
type symbolTableID uint64

const stKindShift = 61

const (
	stKindLocals symbolTableID = iota << stKindShift
	stKindExports
	stKindMembers
	stKindGlobals
	stKindResolvedExports // resolved/derived exports from getExportsOfSymbol, distinct from raw sym.Exports

	// stKindMask extracts the kind bits from a symbolTableID.
	stKindMask symbolTableID = ^symbolTableID(1<<stKindShift - 1)
)

func symbolTableIDFromLocals(node ast.Node) symbolTableID {
	return stKindLocals | symbolTableID(ast.GetNodeId(node))
}

func (c *Checker) symbolTableIDFromExports(sym ast.SymbolId) symbolTableID {
	return stKindExports | symbolTableID(ast.GetSymbolSerial(c.symbolOf(sym)))
}

// symbolTableIDFromResolvedExports returns an ID for resolved/derived export tables
// (e.g. from getExportsOfSymbol/getExportsOfModule which may include export * resolution
// and late-bound members). This is distinct from symbolTableIDFromExports to prevent
// cache collisions with raw sym.Exports tables passed by someSymbolTableInScope.
func (c *Checker) symbolTableIDFromResolvedExports(sym ast.SymbolId) symbolTableID {
	return stKindResolvedExports | symbolTableID(ast.GetSymbolSerial(c.symbolOf(sym)))
}

func (c *Checker) symbolTableIDFromMembers(sym ast.SymbolId) symbolTableID {
	return stKindMembers | symbolTableID(ast.GetSymbolSerial(c.symbolOf(sym)))
}

func symbolTableIDFromGlobals() symbolTableID {
	return stKindGlobals
}

func (c *Checker) getAccessibleSymbolChainEx(ctx accessibleSymbolChainContext) []ast.SymbolId {
	if ctx.symbol == 0 {
		return nil
	}
	if c.isPropertyOrMethodDeclarationSymbol(ctx.symbol) {
		return nil
	}
	// Go from enclosingDeclaration to the first scope we check, so the cache is keyed off the scope and thus shared more
	var firstRelevantLocation ast.Node
	c.someSymbolTableInScope(ctx.enclosingDeclaration, func(_ ast.SymbolTable, _ symbolTableID, _ bool, _ bool, node ast.Node) bool {
		firstRelevantLocation = node
		return true
	})
	links := c.symbolContainerLinks.Get(ctx.symbol)
	linkKey := accessibleChainCacheKey{ctx.useOnlyExternalAliasing, firstRelevantLocation, ctx.meaning}
	if links.accessibleChainCache == nil {
		links.accessibleChainCache = make(map[accessibleChainCacheKey][]ast.SymbolId)
	}
	existing, ok := links.accessibleChainCache[linkKey]
	if ok {
		return existing
	}

	var result []ast.SymbolId

	c.someSymbolTableInScope(ctx.enclosingDeclaration, func(t ast.SymbolTable, tableId symbolTableID, ignoreQualification bool, isLocalNameLookup bool, _ ast.Node) bool {
		res := c.getAccessibleSymbolChainFromSymbolTable(ctx, t, tableId, ignoreQualification, isLocalNameLookup)
		if len(res) > 0 {
			result = res
			return true
		}
		return false
	})
	links.accessibleChainCache[linkKey] = result
	return result
}

/**
* @param {ignoreQualification} boolean Set when a symbol is being looked for through the exports of another symbol (meaning we have a route to qualify it already)
 */
func (c *Checker) getAccessibleSymbolChainFromSymbolTable(ctx accessibleSymbolChainContext, t ast.SymbolTable, tableId symbolTableID, ignoreQualification bool, isLocalNameLookup bool) []ast.SymbolId {
	symId := ast.GetSymbolSerial(c.symbolOf(ctx.symbol))
	visitedSymbolTables, ok := ctx.visitedSymbolTablesMap[symId]
	if !ok {
		visitedSymbolTables = make(map[symbolTableID]struct{})
		ctx.visitedSymbolTablesMap[symId] = visitedSymbolTables
	}

	_, present := visitedSymbolTables[tableId]
	if present {
		return nil
	}
	visitedSymbolTables[tableId] = struct{}{}

	res := c.trySymbolTable(ctx, t, tableId, ignoreQualification, isLocalNameLookup)

	delete(visitedSymbolTables, tableId)
	return res
}

// getSymbolTableAliases returns only the alias symbols from a symbol table,
// caching the result by tableId to avoid repeated iteration over large tables.
// Members tables are skipped entirely since someSymbolTableInScope filters them
// to SymbolFlagsType & ^SymbolFlagsAssignment, which never includes aliases.
func (c *Checker) getSymbolTableAliases(symbols ast.SymbolTable, tableId symbolTableID) []ast.SymbolId {
	kind := tableId & stKindMask
	// Members tables never contain alias symbols; skip entirely.
	if kind == stKindMembers {
		return nil
	}
	// Cache globals and exports tables (which are large and revisited often).
	// Locals tables are small and per-scope, so they are filtered but not cached.
	if kind == stKindGlobals || kind == stKindExports || kind == stKindResolvedExports {
		if c.symbolTableAliasCache != nil {
			if aliases, ok := c.symbolTableAliasCache[tableId]; ok {
				return aliases
			}
		}
	}
	var aliases []ast.SymbolId
	for _, sym := range symbols.AllIds(c.symbols) {
		if c.symbolFlags(sym)&ast.SymbolFlagsAlias != 0 {
			aliases = append(aliases, sym)
		}
	}
	if kind == stKindGlobals || kind == stKindExports || kind == stKindResolvedExports {
		if c.symbolTableAliasCache == nil {
			c.symbolTableAliasCache = make(map[symbolTableID][]ast.SymbolId)
		}
		c.symbolTableAliasCache[tableId] = aliases
	}
	return aliases
}

func (c *Checker) trySymbolTable(
	ctx accessibleSymbolChainContext,
	symbols ast.SymbolTable,
	tableId symbolTableID,
	ignoreQualification bool,
	isLocalNameLookup bool,
) []ast.SymbolId {
	isGlobals := tableId == stKindGlobals
	// If symbol is directly available by its name in the symbol table
	res, ok := symbols.LookupId(c.symbols, c.symbolName(ctx.symbol))
	if ok && res != 0 && c.isAccessible(ctx, res /*resolvedAliasSymbol*/, 0, ignoreQualification) {
		return []ast.SymbolId{ctx.symbol}
	}

	var candidateChains [][]ast.SymbolId

	// Check for ExportSymbol by direct name lookup rather than discovering it during
	// the alias iteration below (where it would never match, since only alias-flagged
	// symbols are iterated).
	if ok && res != 0 && c.symbolExportSymbol(res) != 0 {
		if c.isAccessible(ctx, c.getMergedSymbol(c.symbolExportSymbol(res)) /*resolvedAliasSymbol*/, 0, ignoreQualification) {
			candidateChains = append(candidateChains, []ast.SymbolId{ctx.symbol})
		}
	}

	// Iterate only alias symbols from the table (cached per tableId).
	// This avoids iterating thousands of non-alias symbols in large tables like globals.
	for _, symbolFromSymbolTable := range c.getSymbolTableAliases(symbols, tableId) {
		// for every non-default, non-export= alias symbol in scope, check if it refers to or can chain to the target symbol
		if c.symbolName(symbolFromSymbolTable) != ast.InternalSymbolNameExportEquals &&
			c.symbolName(symbolFromSymbolTable) != ast.InternalSymbolNameDefault &&
			!(c.isUMDExportSymbol(symbolFromSymbolTable) && !ctx.enclosingDeclaration.IsNil() && ast.IsExternalModule(ast.GetSourceFileOfNode(ctx.enclosingDeclaration))) &&
			// If `!useOnlyExternalAliasing`, we can use any type of alias to get the name
			(!ctx.useOnlyExternalAliasing || c.symbolDeclarations(symbolFromSymbolTable).Some(ast.IsExternalModuleImportEqualsDeclaration)) &&
			// If we're looking up a local name to reference directly, omit namespace reexports, otherwise when we're trawling through an export list to make a dotted name, we can keep it
			(isLocalNameLookup && !c.symbolDeclarations(symbolFromSymbolTable).Some(isNamespaceReexportDeclaration) || !isLocalNameLookup) &&
			// While exports are generally considered to be in scope, export-specifier declared symbols are _not_
			// See similar comment in `resolveName` for details
			(ignoreQualification || len(c.getDeclarationsOfKind(symbolFromSymbolTable, ast.KindExportSpecifier)) == 0) {
			resolvedImportedSymbol := c.resolveAlias(symbolFromSymbolTable)
			candidate := c.getCandidateListForSymbol(ctx, symbolFromSymbolTable, resolvedImportedSymbol, ignoreQualification)
			if len(candidate) > 0 {
				candidateChains = append(candidateChains, candidate)
			}
		}
	}

	if len(candidateChains) > 0 {
		// pick first, shortest
		slices.SortStableFunc(candidateChains, c.compareSymbolChains)
		return candidateChains[0]
	}

	// If there's no result and we're looking at the global symbol table, treat `globalThis` like an alias and try to lookup thru that
	if isGlobals {
		return c.getCandidateListForSymbol(ctx, c.globalThisSymbol, c.globalThisSymbol, ignoreQualification)
	}
	return nil
}

func (c *Checker) compareSymbolChainsWorker(a []ast.SymbolId, b []ast.SymbolId) int {
	chainLen := len(a) - len(b)
	if chainLen != 0 {
		return chainLen
	}

	idx := 0
	for idx < len(a) {
		comparison := c.compareSymbols(a[idx], b[idx])
		if comparison != 0 {
			return comparison
		}
		idx++
	}
	return 0
}

func (c *Checker) isUMDExportSymbol(symbol ast.SymbolId) bool {
	return symbol != 0 && c.symbolDeclarations(symbol).Len() > 0 && !c.symbolDeclarations(symbol).At(0).IsNil() && ast.IsNamespaceExportDeclaration(c.symbolDeclarations(symbol).At(0))
}

func isNamespaceReexportDeclaration(node ast.Node) bool {
	return ast.IsNamespaceExport(node) && !node.Parent().ModuleSpecifier().IsNil()
}

func (c *Checker) getCandidateListForSymbol(
	ctx accessibleSymbolChainContext,
	symbolFromSymbolTable ast.SymbolId,
	resolvedImportedSymbol ast.SymbolId,
	ignoreQualification bool,
) []ast.SymbolId {
	if c.isAccessible(ctx, symbolFromSymbolTable, resolvedImportedSymbol, ignoreQualification) {
		return []ast.SymbolId{symbolFromSymbolTable}
	}

	// Look in the exported members, if we can find accessibleSymbolChain, symbol is accessible using this chain
	// but only if the symbolFromSymbolTable can be qualified
	candidateTable := c.getExportsOfSymbol(resolvedImportedSymbol)
	if candidateTable.IsNil() {
		return nil
	}
	candidateTableId := c.symbolTableIDFromResolvedExports(resolvedImportedSymbol)
	accessibleSymbolsFromExports := c.getAccessibleSymbolChainFromSymbolTable(ctx, candidateTable, candidateTableId /*ignoreQualification*/, true, false)
	if len(accessibleSymbolsFromExports) == 0 {
		return nil
	}
	if !c.canQualifySymbol(ctx, symbolFromSymbolTable, getQualifiedLeftMeaning(ctx.meaning)) {
		return nil
	}
	return append([]ast.SymbolId{symbolFromSymbolTable}, accessibleSymbolsFromExports...)
}

func (c *Checker) isAccessible(
	ctx accessibleSymbolChainContext,
	symbolFromSymbolTable ast.SymbolId,
	resolvedAliasSymbol ast.SymbolId,
	ignoreQualification bool,
) bool {
	likeSymbols := false
	if ctx.symbol == resolvedAliasSymbol {
		likeSymbols = true
	}
	if ctx.symbol == symbolFromSymbolTable {
		likeSymbols = true
	}
	symbol := c.getMergedSymbol(ctx.symbol)
	if symbol == c.getMergedSymbol(resolvedAliasSymbol) {
		likeSymbols = true
	}
	if symbol == c.getMergedSymbol(symbolFromSymbolTable) {
		likeSymbols = true
	}
	if !likeSymbols && resolvedAliasSymbol != 0 && c.symbolFlags(resolvedAliasSymbol)&ast.SymbolFlagsAlias != 0 {
		var seenAliases collections.Set[ast.SymbolId]
		for c.symbolFlags(resolvedAliasSymbol)&ast.SymbolFlagsAlias != 0 && !seenAliases.Has(resolvedAliasSymbol) {
			seenAliases.Add(resolvedAliasSymbol)
			resolvedAliasSymbol = c.getMergedSymbol(c.resolveAlias(resolvedAliasSymbol))
			if symbol == resolvedAliasSymbol {
				likeSymbols = true
				break
			}
		}
	}
	if !likeSymbols {
		return false
	}
	// if the symbolFromSymbolTable is not external module (it could be if it was determined as ambient external module and would be in globals table)
	// and if symbolFromSymbolTable or alias resolution matches the symbol,
	// check the symbol can be qualified, it is only then this symbol is accessible
	return !c.symbolDeclarations(symbolFromSymbolTable).Some(hasNonGlobalAugmentationExternalModuleSymbol) &&
		(ignoreQualification || c.canQualifySymbol(ctx, c.getMergedSymbol(symbolFromSymbolTable), ctx.meaning))
}

func (c *Checker) canQualifySymbol(
	ctx accessibleSymbolChainContext,
	symbolFromSymbolTable ast.SymbolId,
	meaning ast.SymbolFlags,
) bool {
	// If the symbol is equivalent and doesn't need further qualification, this symbol is accessible
	return !c.needsQualification(symbolFromSymbolTable, ctx.enclosingDeclaration, meaning) ||
		// If symbol needs qualification, make sure that parent is accessible, if it is then this symbol is accessible too
		len(c.getAccessibleSymbolChainEx(accessibleSymbolChainContext{c.symbolParent(symbolFromSymbolTable), ctx.enclosingDeclaration, getQualifiedLeftMeaning(meaning), ctx.useOnlyExternalAliasing, ctx.visitedSymbolTablesMap})) > 0
}

func (c *Checker) needsQualification(symbol ast.SymbolId, enclosingDeclaration ast.Node, meaning ast.SymbolFlags) bool {
	qualify := false
	c.someSymbolTableInScope(enclosingDeclaration, func(symbolTable ast.SymbolTable, _ symbolTableID, _ bool, _ bool, _ ast.Node) bool {
		// If symbol of this name is not available in the symbol table we are ok
		res, ok := symbolTable.LookupId(c.symbols, c.symbolName(symbol))
		if !ok || res == 0 {
			return false
		}
		symbolFromSymbolTable := c.getMergedSymbol(res)
		if symbolFromSymbolTable == 0 {
			// Continue to the next symbol table
			return false
		}
		// If the symbol with this name is present it should refer to the symbol
		if symbolFromSymbolTable == symbol {
			// No need to qualify
			return true
		}

		// Qualify if the symbol from symbol table has same meaning as expected
		shouldResolveAlias := c.symbolFlags(symbolFromSymbolTable)&ast.SymbolFlagsAlias != 0 && ast.GetDeclarationOfKind(c.symbolOf(symbolFromSymbolTable), ast.KindExportSpecifier).IsNil()
		if shouldResolveAlias {
			symbolFromSymbolTable = c.resolveAlias(symbolFromSymbolTable)
		}
		flags := c.symbolFlags(symbolFromSymbolTable)
		if shouldResolveAlias {
			flags = c.getSymbolFlags(symbolFromSymbolTable)
		}
		if flags&meaning != 0 {
			qualify = true
			return true
		}

		// Continue to the next symbol table
		return false
	})

	return qualify
}

func (c *Checker) isPropertyOrMethodDeclarationSymbol(symbol ast.SymbolId) bool {
	if c.symbolDeclarations(symbol).Len() > 0 {
		for _, declaration := range c.symbolDeclarations(symbol).All() {
			switch declaration.Kind() {
			case ast.KindPropertyDeclaration,
				ast.KindMethodDeclaration,
				ast.KindGetAccessor,
				ast.KindSetAccessor:
				continue
			default:
				return false
			}
		}
		return true
	}
	return false
}

func (c *Checker) someSymbolTableInScope(
	enclosingDeclaration ast.Node,
	callback func(symbolTable ast.SymbolTable, tableId symbolTableID, ignoreQualification bool, isLocalNameLookup bool, scopeNode ast.Node) bool,
) bool {
	for location := enclosingDeclaration; !location.IsNil(); location = location.Parent() {
		// Locals of a source file are not in scope (because they get merged into the global symbol table)
		if canHaveLocals(location) && !location.Locals().IsNil() && !ast.IsGlobalSourceFile(location) {
			if callback(location.Locals(), symbolTableIDFromLocals(location.AsNode()), false, true, location) {
				return true
			}
		}
		switch location.Kind() {
		case ast.KindSourceFile, ast.KindModuleDeclaration:
			if ast.IsSourceFile(location) && !ast.IsExternalOrCommonJSModule(location.AsSourceFile()) {
				break
			}
			sym := c.getSymbolOfDeclaration(ast.GetReparsedNodeForNode(location))
			if callback(c.symbolExports(sym), c.symbolTableIDFromExports(sym), false, true, location) {
				return true
			}
		case ast.KindClassDeclaration, ast.KindClassExpression, ast.KindInterfaceDeclaration:
			// Type parameters are bound into `members` lists so they can merge across declarations
			// This is troublesome, since in all other respects, they behave like locals :cries:
			// TODO: the below is shared with similar code in `resolveName` - in fact, rephrasing all this symbol
			// lookup logic in terms of `resolveName` would be nice
			// The below is used to lookup type parameters within a class or interface, as they are added to the class/interface locals
			// These can never be latebound, so the symbol's raw members are sufficient. `getMembersOfNode` cannot be used, as it would
			// trigger resolving late-bound names, which we may already be in the process of doing while we're here!
			var table ast.SymbolTable
			sym := c.getSymbolOfDeclaration(location)
			// TODO: Should this filtered table be cached in some way?
			for key, memberSymbol := range c.symbolMembers(sym).AllIds(c.symbols) {
				if c.symbolFlags(memberSymbol)&(ast.SymbolFlagsType & ^ast.SymbolFlagsAssignment) != 0 {
					if table.IsNil() {
						table = c.symbols.NewTable(0)
					}
					table.SetId(c.symbols, key, memberSymbol)
				}
			}
			if !table.IsNil() && callback(table, c.symbolTableIDFromMembers(sym), false, false, location) {
				return true
			}
			// Class expression names (e.g., `B` in `class B {}`) are not stored in any
			// scope table — the binder uses bindAnonymousDeclaration. Expose the name
			// binding here so getAccessibleSymbolChain can resolve self-references.
			// This mirrors the special casing of class expression names in
			// (*NameResolver).Resolve; if class names are ever bound differently
			// (e.g., via class-local type aliases), both sites should be updated.
			if ast.IsClassExpression(location) && !location.AsClassExpression().Name().IsNil() {
				nameTable := c.getClassExpressionNameTable(location)
				if !nameTable.IsNil() && callback(nameTable, symbolTableIDFromLocals(location.AsNode()), false, true, location) {
					return true
				}
			}
		}
	}

	return callback(c.globals, symbolTableIDFromGlobals(), false, true, ast.Node{})
}

// getClassExpressionNameTable returns a cached symbol table containing the class
// expression's name binding. Class expression names are bound via
// bindAnonymousDeclaration and aren't stored in any container's locals, so this
// synthesized table lets someSymbolTableInScope expose them during accessibility checks.
func (c *Checker) getClassExpressionNameTable(location ast.Node) ast.SymbolTable {
	nodeId := ast.GetNodeId(location)
	if c.classExpressionNameTables != nil {
		if table, ok := c.classExpressionNameTables[nodeId]; ok {
			return table
		}
	}
	classSymbol := c.getSymbolOfDeclaration(location)
	nameText := location.AsClassExpression().Name().Text()
	if len(nameText) == 0 || classSymbol == 0 {
		return ast.SymbolTable{}
	}
	table := c.symbols.TableOfId(ast.MakeSymbolName(nameText), classSymbol)
	if c.classExpressionNameTables == nil {
		c.classExpressionNameTables = make(map[ast.NodeId]ast.SymbolTable)
	}
	c.classExpressionNameTables[nodeId] = table
	return table
}

/**
 * Check if the given symbol in given enclosing declaration is accessible and mark all associated alias to be visible if requested
 *
 * @param symbol a Symbol to check if accessible
 * @param enclosingDeclaration a Node containing reference to the symbol
 * @param meaning a SymbolFlags to check if such meaning of the symbol is accessible
 * @param shouldComputeAliasToMakeVisible a boolean value to indicate whether to return aliases to be mark visible in case the symbol is accessible
 */

func (c *Checker) IsSymbolAccessible(symbol ast.Symbol, enclosingDeclaration ast.Node, meaning ast.SymbolFlags, shouldComputeAliasesToMakeVisible bool) printer.SymbolAccessibilityResult {
	return c.isSymbolAccessible(c.symbolId(symbol), enclosingDeclaration, meaning, shouldComputeAliasesToMakeVisible)
}

func (c *Checker) isSymbolAccessible(symbol ast.SymbolId, enclosingDeclaration ast.Node, meaning ast.SymbolFlags, shouldComputeAliasesToMakeVisible bool) printer.SymbolAccessibilityResult {
	return c.isSymbolAccessibleWorker(symbol, enclosingDeclaration, meaning, shouldComputeAliasesToMakeVisible, true /*allowModules*/)
}

func (c *Checker) isSymbolAccessibleWorker(symbol ast.SymbolId, enclosingDeclaration ast.Node, meaning ast.SymbolFlags, shouldComputeAliasesToMakeVisible bool, allowModules bool) printer.SymbolAccessibilityResult {
	if symbol != 0 && !enclosingDeclaration.IsNil() {
		result := c.isAnySymbolAccessible([]ast.SymbolId{symbol}, enclosingDeclaration, symbol, meaning, shouldComputeAliasesToMakeVisible, allowModules)
		if result != nil {
			return *result
		}

		// This could be a symbol that is not exported in the external module
		// or it could be a symbol from different external module that is not aliased and hence cannot be named
		symbolExternalModule := core.FirstNonNilSeq(c.symbolDeclarations(symbol).Values(), c.getExternalModuleContainer)
		if symbolExternalModule != 0 {
			enclosingExternalModule := c.getExternalModuleContainer(enclosingDeclaration)
			if symbolExternalModule != enclosingExternalModule {
				// name from different external module that is not visible
				return printer.SymbolAccessibilityResult{
					Accessibility:   printer.SymbolAccessibilityCannotBeNamed,
					ErrorSymbolName: c.symbolToStringEx(symbol, enclosingDeclaration, meaning, SymbolFormatFlagsAllowAnyNodeKind),
					ErrorModuleName: c.symbolToString(symbolExternalModule),
					ErrorNode:       core.IfElse(ast.IsInJSFile(enclosingDeclaration), enclosingDeclaration, ast.Node{}),
				}
			}
		}

		// Just a local name that is not accessible
		return printer.SymbolAccessibilityResult{
			Accessibility:   printer.SymbolAccessibilityNotAccessible,
			ErrorSymbolName: c.symbolToStringEx(symbol, enclosingDeclaration, meaning, SymbolFormatFlagsAllowAnyNodeKind),
		}
	}

	return printer.SymbolAccessibilityResult{
		Accessibility: printer.SymbolAccessibilityAccessible,
	}
}

func (c *Checker) GetAccessibleSymbolChain(
	symbol ast.Symbol,
	enclosingDeclaration ast.Node,
	meaning ast.SymbolFlags,
	useOnlyExternalAliasing bool,
) []ast.Symbol {
	return c.symbolsOf(c.getAccessibleSymbolChainId(c.symbolId(symbol), enclosingDeclaration, meaning, useOnlyExternalAliasing))
}
