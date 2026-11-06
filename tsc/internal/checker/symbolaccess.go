package checker

import (
	"iter"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
)

// The checker handles symbols by their 4-byte SymbolId. Ids below
// binderSymbolCount name the program's binder symbols, whose flags and name
// the program's table copies; the ids above name the checker's own symbols.
// The accessors below read a symbol's fields from its id.

func (c *Checker) symbolFlags(id ast.SymbolId) ast.SymbolFlags {
	if uint32(id) < c.binderSymbolCount {
		return c.binderSymbols.Flags(id)
	}
	return c.symbols.FlagsOf(id)
}

func (c *Checker) symbolName(id ast.SymbolId) ast.SymbolName {
	if uint32(id) < c.binderSymbolCount {
		return c.binderSymbols.Name(id)
	}
	return c.symbols.NameOf(id)
}

func (c *Checker) symbolCheckFlags(id ast.SymbolId) ast.CheckFlags {
	if uint32(id) < c.binderSymbolCount {
		return 0
	}
	return c.symbols.CheckFlagsOf(id)
}

func (c *Checker) symbolParent(id ast.SymbolId) ast.SymbolId {
	return c.symbols.ParentOf(id)
}

func (c *Checker) symbolDeclarations(id ast.SymbolId) ast.Declarations {
	return c.symbols.DeclarationsOf(id)
}

func (c *Checker) symbolValueDeclaration(id ast.SymbolId) ast.Node {
	return c.symbols.ValueDeclarationOf(id)
}

func (c *Checker) symbolMembers(id ast.SymbolId) ast.SymbolTable {
	return c.symbols.MembersOf(id)
}

func (c *Checker) symbolExports(id ast.SymbolId) ast.SymbolTable {
	return c.symbols.ExportsOf(id)
}

func (c *Checker) symbolExportSymbol(id ast.SymbolId) ast.SymbolId {
	return c.symbols.ExportSymbolOf(id)
}

func (c *Checker) setSymbolFlags(id ast.SymbolId, value ast.SymbolFlags) {
	c.symbols.SetFlagsOf(id, value)
}

func (c *Checker) setSymbolCheckFlags(id ast.SymbolId, value ast.CheckFlags) {
	c.symbols.SetCheckFlagsOf(id, value)
}

func (c *Checker) setSymbolName(id ast.SymbolId, value ast.SymbolName) {
	c.symbols.SetNameOf(id, value)
}

func (c *Checker) setSymbolDeclarations(id ast.SymbolId, value ast.Declarations) {
	c.symbols.SetDeclarationsOf(id, value)
}

func (c *Checker) setSymbolValueDeclaration(id ast.SymbolId, value ast.Node) {
	c.symbols.SetValueDeclarationOf(id, value)
}

func (c *Checker) setSymbolParent(id ast.SymbolId, value ast.SymbolId) {
	c.symbols.SetParentOf(id, value)
}

func (c *Checker) setSymbolMembers(id ast.SymbolId, value ast.SymbolTable) {
	c.symbols.SetMembersOf(id, value)
}

func (c *Checker) setSymbolExports(id ast.SymbolId, value ast.SymbolTable) {
	c.symbols.SetExportsOf(id, value)
}

func (c *Checker) setSymbolExportSymbol(id ast.SymbolId, value ast.SymbolId) {
	c.symbols.SetExportSymbolOf(id, value)
}

func (c *Checker) symbolIsExternalModule(id ast.SymbolId) bool {
	return c.symbolFlags(id)&ast.SymbolFlagsModule != 0 && ast.IsAmbientModuleSymbolName(c.symbolName(id))
}

func (c *Checker) symbolIsStatic(id ast.SymbolId) bool {
	valueDeclaration := c.symbolValueDeclaration(id)
	return !valueDeclaration.IsNil() && valueDeclaration.ModifierFlags()&ast.ModifierFlagsStatic != 0
}

// combinedLocalAndExportSymbolFlags: see declareModuleMember in the binder.
func (c *Checker) combinedLocalAndExportSymbolFlags(id ast.SymbolId) ast.SymbolFlags {
	if e := c.symbolExportSymbol(id); e != 0 {
		return c.symbolFlags(id) | c.symbolFlags(e)
	}
	return c.symbolFlags(id)
}

// symbolsOf returns views of the symbols ids name, for callers outside the
// checker.
func (c *Checker) symbolsOf(ids []ast.SymbolId) []ast.Symbol {
	if ids == nil {
		return nil
	}
	symbols := make([]ast.Symbol, len(ids))
	for i, id := range ids {
		symbols[i] = c.symbolOf(id)
	}
	return symbols
}

// symbolIdsOf returns the ids of symbols.
func (c *Checker) symbolIdsOf(symbols []ast.Symbol) []ast.SymbolId {
	if symbols == nil {
		return nil
	}
	ids := make([]ast.SymbolId, len(symbols))
	for i, symbol := range symbols {
		ids[i] = c.symbolId(symbol)
	}
	return ids
}

// symbolIdSeq returns the ids of the symbols seq yields.
func (c *Checker) symbolIdSeq(seq iter.Seq[ast.Symbol]) iter.Seq[ast.SymbolId] {
	return func(yield func(ast.SymbolId) bool) {
		for symbol := range seq {
			if !yield(c.symbolId(symbol)) {
				return
			}
		}
	}
}

// nodeSymbol returns the id of the symbol the binder gave node.
func (c *Checker) nodeSymbol(node ast.Node) ast.SymbolId {
	return c.symbolId(node.Symbol())
}

// getSymbolViewOfDeclaration and symbolViewReferenced serve the name
// resolver, which handles views.
func (c *Checker) getSymbolViewOfDeclaration(node ast.Node) ast.Symbol {
	return c.symbolOf(c.getSymbolOfDeclaration(node))
}

func (c *Checker) symbolViewReferenced(symbol ast.Symbol, meaning ast.SymbolFlags) {
	c.symbolReferenced(c.symbolId(symbol), meaning)
}

// SymbolView returns the view of a symbol id, for callers outside the
// checker.
func (c *Checker) SymbolView(id ast.SymbolId) ast.Symbol {
	return c.symbolOf(id)
}

// SymbolViews returns the views of symbol ids, for callers outside the
// checker.
func (c *Checker) SymbolViews(ids []ast.SymbolId) []ast.Symbol {
	return c.symbolsOf(ids)
}

// SymbolIdOf returns the id of a symbol view, for callers outside the
// checker.
func (c *Checker) SymbolIdOf(symbol ast.Symbol) ast.SymbolId {
	return c.symbolId(symbol)
}
