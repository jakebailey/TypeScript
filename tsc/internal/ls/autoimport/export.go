package autoimport

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/ls/lsutil"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

//go:generate npx hereby generate:autoimport

type moduleIDKind uint8

const (
	moduleIDKindInvalid moduleIDKind = iota
	moduleIDKindFile
	moduleIDKindAmbient
)

// ModuleID uniquely identifies either a file module or an ambient module.
// For file modules, value is a tspath.PathKey; for ambient modules, a tspath.ModuleSpecifier.
type ModuleID struct {
	value string
	kind  moduleIDKind
}

func fileModuleID(path tspath.PathKey) ModuleID {
	return ModuleID{value: string(path), kind: moduleIDKindFile}
}

func ambientModuleID(specifier string) ModuleID {
	return ModuleID{value: string(tspath.ToModuleSpecifier(specifier)), kind: moduleIDKindAmbient}
}

func (m ModuleID) AsString() string {
	return m.value
}

func (m ModuleID) IsAmbient() bool {
	return m.kind == moduleIDKindAmbient
}

func (m ModuleID) AsPathKey() (tspath.PathKey, bool) {
	if m.kind != moduleIDKindFile {
		return "", false
	}
	return tspath.PathKeyFromCanonical(m.value), true
}

// IsFile reports whether m identifies the file module at path.
func (m ModuleID) IsFile(path tspath.PathKey) bool {
	return m.kind == moduleIDKindFile && m.value == path.AsString()
}

func (m ModuleID) AsModuleSpecifier() (tspath.ModuleSpecifier, bool) {
	if m.kind != moduleIDKindAmbient {
		return "", false
	}
	return tspath.ToModuleSpecifier(m.value), true
}

type ExportID struct {
	ModuleID   ModuleID
	ExportName ast.SymbolName
}

type ExportSyntax uint8

const (
	ExportSyntaxNone ExportSyntax = iota
	// export const x = {}
	ExportSyntaxModifier
	// export { x }
	ExportSyntaxNamed
	// export default function f() {}
	ExportSyntaxDefaultModifier
	// export default f
	ExportSyntaxDefaultDeclaration
	// export = x
	ExportSyntaxEquals
	// export as namespace x
	ExportSyntaxUMD
	// export * from "module"
	ExportSyntaxStar
	// module.exports = {}
	ExportSyntaxCommonJSModuleExports
	// exports.x = {}
	ExportSyntaxCommonJSExportsProperty
)

type Export struct {
	ExportID
	ModuleFileName            tspath.RootedFilePath
	UnresolvedModuleSpecifier tspath.ModuleSpecifier
	Flags                     ast.SymbolFlags
	Syntax                    ExportSyntax
	localName                 ast.SymbolName
	// through is the name of the module symbol's export that this export was found on,
	// either 'export=', InternalSymbolNameExportStar, or unset.
	through ast.SymbolName

	// Checker-set fields

	Target                     ExportID
	ScriptElementKindModifiers lsutil.ScriptElementKindModifier
	ScriptElementKind          lsutil.ScriptElementKind
	IsTypeOnly                 bool

	// The file where the export was found.
	Path tspath.PathKey

	PackageName string
}

func (e *Export) Name() ast.SymbolName {
	if e.localName != ast.EmptySymbolName {
		return e.localName
	}
	if e.ExportName == ast.InternalSymbolNameExportEquals {
		return e.Target.ExportName
	}
	return e.ExportName
}

func (e *Export) IsRenameable() bool {
	return e.ExportName == ast.InternalSymbolNameExportEquals || e.ExportName == ast.InternalSymbolNameDefault
}

func (e *Export) AmbientModuleName() string {
	if e.ModuleID.IsAmbient() {
		return e.ModuleID.AsString()
	}
	return ""
}

func (e *Export) IsUnresolvedAlias() bool {
	return e.Flags == ast.SymbolFlagsAlias
}

func SymbolToExport(symbol ast.Symbol, ch *checker.Checker) *Export {
	if !symbol.Parent().IsNil() && checker.IsExternalModuleSymbol(symbol.Parent()) {
		if moduleID, moduleFileName, ok := tryGetModuleIDAndFileNameOfModuleSymbol(symbol.Parent()); ok {
			return extractFirstExport(symbol, ch, moduleID, moduleFileName, ast.GetSourceFileOfModule(symbol.Parent()))
		}
		return nil
	}

	declaration := symbol.Declarations().FirstOrNil()
	if declaration.IsNil() {
		return nil
	}

	file := ast.GetSourceFileOfNode(declaration)
	if file.Symbol().IsNil() {
		return nil
	}

	moduleSymbol := ch.GetMergedSymbol(file.Symbol())
	moduleID := fileModuleID(file.PathKey())
	moduleFileName := file.FileName()
	target := ch.GetMergedSymbol(ch.SkipAlias(symbol))

	if export := tryGetModuleExport(ast.InternalSymbolNameDefault, target, moduleSymbol, ch, moduleID, moduleFileName, file); export != nil {
		return export
	}
	if export := tryGetModuleExport(ast.InternalSymbolNameExportEquals, target, moduleSymbol, ch, moduleID, moduleFileName, file); export != nil {
		return export
	}
	return tryGetModuleExport(symbol.Name(), target, moduleSymbol, ch, moduleID, moduleFileName, file)
}

func tryGetModuleExport(exportName ast.SymbolName, target ast.Symbol, moduleSymbol ast.Symbol, ch *checker.Checker, moduleID ModuleID, moduleFileName tspath.RootedFilePath, file *ast.SourceFile) *Export {
	exported := ch.TryGetMemberInModuleExportsAndProperties(exportName, moduleSymbol)
	if !exported.IsNil() && ch.GetMergedSymbol(ch.SkipAlias(exported)) == target {
		return extractFirstExport(exported, ch, moduleID, moduleFileName, file)
	}
	return nil
}

func extractFirstExport(symbol ast.Symbol, ch *checker.Checker, moduleID ModuleID, moduleFileName tspath.RootedFilePath, file *ast.SourceFile) *Export {
	var exports []*Export
	extractor := newSymbolExtractor("", ch, tspath.CaseInsensitive, nil)
	extractor.extractFromSymbol(symbol.Name(), symbol, moduleID, moduleFileName, file, &exports)
	return core.FirstOrNil(exports)
}
