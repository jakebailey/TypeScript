package checker_test

import (
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/repo"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/vfstest"
	"gotest.tools/v3/assert"
)

func TestGetSymbolAtLocation(t *testing.T) {
	t.Parallel()

	content := `interface Foo {
  bar: string;
}
declare const foo: Foo;
foo.bar;`
	fs := vfstest.FromMap(map[string]string{
		"/foo.ts": content,
		"/tsconfig.json": `
				{
					"compilerOptions": {},
					"files": ["foo.ts"]
				}
			`,
	}, tspath.CaseInsensitive /*caseSensitivity*/)
	fs = bundled.WrapFS(fs)

	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)
	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile("/tsconfig.json", &core.CompilerOptions{}, nil, tsoptions.NewDefaultParseConfigHost(fs), nil)
	assert.Equal(t, len(errors), 0, "Expected no errors in parsed command line")

	p := compiler.NewProgram(compiler.ProgramOptions{
		Config: parsed,
		Host:   host,
	})
	p.BindSourceFiles()
	c, done := p.GetTypeChecker(t.Context())
	defer done()
	file := p.GetSourceFile("/foo.ts")
	interfaceId := file.Statements.Nodes[0].Name()
	varId := file.Statements.Nodes[1].AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes[0].Name()
	propAccess := file.Statements.Nodes[2].Expression()
	nodes := []*ast.Node{interfaceId, varId, propAccess}
	for _, node := range nodes {
		symbol := c.GetSymbolAtLocation(node)
		if symbol == nil {
			t.Fatalf("Expected symbol to be non-nil")
		}
	}
}

func TestGetTypeAtLocationOfTypeOnlyImportClause(t *testing.T) {
	t.Parallel()

	fs := vfstest.FromMap(map[string]string{
		"/types.ts": `export type U = number;
export default interface D { x: number }`,
		"/main.ts": `import type { U } from "./types";
import type * as types from "./types";
import { U as V } from "./types";
import type D from "./types";
export const u: U = 1;
export const v: V = 1;
export type W = types.U;
export type E = D;`,
		"/tsconfig.json": `
				{
					"compilerOptions": {},
					"files": ["types.ts", "main.ts"]
				}
			`,
	}, tspath.CaseInsensitive)
	fs = bundled.WrapFS(fs)

	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)

	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile("/tsconfig.json", &core.CompilerOptions{}, nil, tsoptions.NewDefaultParseConfigHost(fs), nil)
	assert.Equal(t, len(errors), 0, "Expected no errors in parsed command line")

	p := compiler.NewProgram(compiler.ProgramOptions{
		Config: parsed,
		Host:   host,
	})
	p.BindSourceFiles()
	c, done := p.GetTypeChecker(t.Context())
	defer done()
	file := p.GetSourceFile("/main.ts")
	importClauseAt := func(index int) *ast.Node {
		return file.Statements.Nodes[index].AsImportDeclaration().ImportClause
	}
	// An import clause without a default binding has no symbol of its own. A type-only one
	// should get the same type as the equivalent regular import instead of crashing.
	regular := c.GetTypeAtLocation(importClauseAt(2))
	for _, index := range []int{0, 1} {
		typ := c.GetTypeAtLocation(importClauseAt(index))
		if typ == nil {
			t.Fatalf("Expected type of import clause %d to be non-nil", index)
		}
		assert.Equal(t, typ, regular)
	}

	defaultClause := c.GetTypeAtLocation(importClauseAt(3))
	defaultReference := c.GetTypeAtLocation(file.Statements.Nodes[7].AsTypeAliasDeclaration().Type)
	assert.Equal(t, defaultClause, defaultReference)
}

func BenchmarkNewChecker(b *testing.B) {
	fs := bundled.WrapFS(osvfs.FS())
	rootPath := tspath.RootedDirectoryPathFromAbsolute(filepath.Join(repo.TestDataPath(), "fixtures/compiler"))
	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)
	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile(rootPath.ResolveFile("tsconfig.json"), &core.CompilerOptions{}, nil, tsoptions.NewDefaultParseConfigHost(fs), nil)
	assert.Equal(b, len(errors), 0, "Expected no errors in parsed command line")
	program := compiler.NewProgram(compiler.ProgramOptions{
		Config: parsed,
		Host:   host,
	})

	b.ReportAllocs()
	for b.Loop() {
		checker.NewChecker(program, nil)
	}
}

func BenchmarkCheck(b *testing.B) {
	fs := bundled.WrapFS(osvfs.FS())
	rootPath := tspath.RootedDirectoryPathFromAbsolute(filepath.Join(repo.TestDataPath(), "fixtures/compiler"))
	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)
	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile(rootPath.ResolveFile("tsconfig.json"), &core.CompilerOptions{}, nil, tsoptions.NewDefaultParseConfigHost(fs), nil)
	assert.Equal(b, len(errors), 0, "Expected no errors in parsed command line")
	program := compiler.NewProgram(compiler.ProgramOptions{
		Config: parsed,
		Host:   host,
	})
	program.BindSourceFiles()
	ctx := b.Context()

	b.ReportAllocs()
	for b.Loop() {
		c, _ := checker.NewChecker(program, nil)
		for _, file := range program.GetSourceFiles() {
			c.GetDiagnostics(ctx, file)
		}
		c.GetGlobalDiagnostics()
	}
}

func BenchmarkCheckFreshProgram(b *testing.B) {
	fs := bundled.WrapFS(osvfs.FS())
	rootPath := tspath.RootedDirectoryPathFromAbsolute(filepath.Join(repo.TestDataPath(), "fixtures/compiler"))
	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)
	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile(rootPath.ResolveFile("tsconfig.json"), &core.CompilerOptions{}, nil, tsoptions.NewDefaultParseConfigHost(fs), nil)
	assert.Equal(b, len(errors), 0, "Expected no errors in parsed command line")
	ctx := b.Context()

	b.ReportAllocs()
	for b.Loop() {
		program := compiler.NewProgram(compiler.ProgramOptions{Config: parsed, Host: host})
		program.BindSourceFiles()
		c, _ := checker.NewChecker(program, nil)
		for _, file := range program.GetSourceFiles() {
			assert.Equal(b, len(c.GetDiagnostics(ctx, file)), 0)
		}
		assert.Equal(b, len(c.GetGlobalDiagnostics()), 0)
	}
}

func TestTypeAllocationAcrossCheckerGrowth(t *testing.T) {
	t.Parallel()
	var content strings.Builder
	content.WriteString(`interface Box<T> { value: T }
export type Initial = Box<"first">;
`)
	for i := range 1024 {
		fmt.Fprintf(&content, "export type T%d = Box<\"value%d\">;\n", i, i)
	}
	fs := bundled.WrapFS(vfstest.FromMap(map[string]string{
		"/foo.ts": content.String(),
	}, tspath.CaseSensitive))
	opts := &core.CompilerOptions{NoLib: core.TSTrue}
	program := compiler.NewProgram(compiler.ProgramOptions{
		Config: tsoptions.NewParsedCommandLine(opts, []tspath.RootedFilePath{"/foo.ts"}, nil, "/", fs.CaseSensitivity()),
		Host:   compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil),
	})
	program.BindSourceFiles()
	c, done := program.GetTypeChecker(t.Context())
	defer done()
	file := program.GetSourceFile("/foo.ts")
	initial := c.GetTypeAtLocation(file.Statements.Nodes[1].AsTypeAliasDeclaration().Type)
	for i := range 1024 {
		node := file.Statements.Nodes[i+2].AsTypeAliasDeclaration().Type
		typ := c.GetTypeAtLocation(node)
		assert.Equal(t, c.TypeToString(typ), fmt.Sprintf("T%d", i))
	}
	assert.Equal(t, c.GetTypeAtLocation(file.Statements.Nodes[1].AsTypeAliasDeclaration().Type), initial)
	assert.Equal(t, c.TypeToString(initial), "Initial")
	assert.Equal(t, len(c.GetDiagnostics(t.Context(), file)), 0)
}

// Memoized results must only be accessed through memo methods, which account
// for the effects of computing them.
func TestMemoValueOnlyAccessedThroughMethods(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	assert.NilError(t, err)
	for _, name := range files {
		if name == "memo.go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		assert.NilError(t, err)
		goast.Inspect(file, func(n goast.Node) bool {
			if sel, ok := n.(*goast.SelectorExpr); ok && sel.Sel.Name == "memoValue" {
				t.Errorf("%s: memo value accessed directly; use get or set", fset.Position(sel.Pos()))
			}
			return true
		})
	}
}

// Mutable cached state must live in memos, which account for the effects of
// computing it. Fields that are not memos must be listed in
// unmemoizedCacheFields with the reason they need not be.
func TestCachedStateIsMemoized(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	assert.NilError(t, err)
	seen := make(map[string]bool)
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		assert.NilError(t, err)
		goast.Inspect(file, func(n goast.Node) bool {
			spec, ok := n.(*goast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := spec.Type.(*goast.StructType)
			if !ok || slices.Contains(notCacheOwners, spec.Name.Name) || name != "types.go" && !strings.HasSuffix(spec.Name.Name, "Links") && spec.Name.Name != "checkerStore" {
				return true
			}
			for _, field := range st.Fields.List {
				if isMemoType(field.Type) {
					continue
				}
				for _, fieldName := range field.Names {
					key := spec.Name.Name + "." + fieldName.Name
					seen[key] = true
					if _, ok := unmemoizedCacheFields[key]; !ok {
						t.Errorf("%s: %s is not a memo; make it one or list it in unmemoizedCacheFields", fset.Position(fieldName.Pos()), key)
					}
				}
			}
			return false
		})
	}
	for key := range unmemoizedCacheFields {
		if !seen[key] {
			t.Errorf("unmemoizedCacheFields lists %s, which is not an unmemoized field", key)
		}
	}
}

func isMemoType(expr goast.Expr) bool {
	switch e := expr.(type) {
	case *goast.IndexExpr:
		expr = e.X
	case *goast.IndexListExpr:
		expr = e.X
	}
	ident, ok := expr.(*goast.Ident)
	return ok && strings.HasPrefix(ident.Name, "memo")
}

const (
	unmemoizedImmutable   = "Set when the store is created"
	unmemoizedContainer   = "Holds links, whose fields are checked separately"
	unmemoizedCounter     = "Allocation counter for IDs and statistics"
	unmemoizedInterned    = "Interned type whose creation has no effects"
	unmemoizedProgram     = "Derived from the program only; computing it has no effects"
	unmemoizedMemoState   = "State of the memos themselves"
	unmemoizedReporting   = "Written only by reporting activations"
	unmemoizedCreated     = "Set when the entity owning the links is created"
	unmemoizedSyntax      = "Derived from syntax and options only; computing it has no effects"
	unmemoizedGuarded     = "Written only within the memoized computation it is part of"
	unmemoizedEmit        = "Emit resolver state, written only while resolving for emit"
	unmemoizedMark        = "Written only by reporting activations and emit marking, and read only by emit"
	unmemoizedNodeBuilder = "Node builder state, scoped to one node builder"
	unmemoizedNodeFlags   = "Each flag is memo-guarded, reporting-only, or set whenever a node is checked and read only by deferred diagnostics, after the whole file is checked"
	unmemoizedKeyed       = "Keyed cell, accounted for like a memo under a key naming its entity and field"
	unmemoizedEntityFlags = "Each flag is set when the entity is created, or is part of a keyed cell"
)

// notCacheOwners are the structs declared in types.go that are plain values
// rather than entities owning cached state.
var notCacheOwners = []string{"LanguageFeatureMinimumTargetMap", "accessibleChainCacheKey", "computedName", "TupleElementInfo"}

var unmemoizedCacheFields = map[string]string{
	"valueSymbolExtra.nameType":              unmemoizedCreated,
	"valueSymbolExtra.containingType":        unmemoizedCreated,
	"TypeAlias.symbol":                       unmemoizedCreated,
	"TypeAlias.typeArguments":                unmemoizedCreated,
	"Type.flags":                             unmemoizedCreated,
	"Type.objectFlags":                       unmemoizedEntityFlags,
	"Type.id":                                unmemoizedCounter,
	"Type.kind":                              unmemoizedCreated,
	"Type.generation":                        unmemoizedCounter,
	"Type.symbol":                            unmemoizedCreated,
	"Type.alias":                             unmemoizedCreated,
	"Type.checker":                           unmemoizedImmutable,
	"IntrinsicType.intrinsicName":            unmemoizedCreated,
	"LiteralType.value":                      unmemoizedCreated,
	"LiteralType.freshType":                  unmemoizedInterned,
	"LiteralType.regularType":                unmemoizedCreated,
	"UniqueESSymbolType.name":                unmemoizedCreated,
	"StructuredType.members":                 unmemoizedKeyed,
	"StructuredType.properties":              unmemoizedKeyed,
	"StructuredType.extra":                   unmemoizedKeyed,
	"structuredTypeExtra.signatures":         unmemoizedKeyed,
	"structuredTypeExtra.callSignatureCount": unmemoizedKeyed,
	"structuredTypeExtra.indexInfos":         unmemoizedKeyed,
	"structuredTypeExtra.objectTypeWithoutAbstractConstructSignatures": unmemoizedInterned,
	"ObjectType.target":                                    unmemoizedCreated,
	"ObjectType.mapper":                                    unmemoizedCreated,
	"TypeReference.node":                                   unmemoizedCreated,
	"InterfaceType.allTypeParameters":                      unmemoizedCreated,
	"InterfaceType.outerTypeParameterCount":                unmemoizedCreated,
	"InterfaceType.thisType":                               unmemoizedCreated,
	"InterfaceType.baseTypesResolved":                      unmemoizedKeyed,
	"InterfaceType.declaredMembersResolved":                unmemoizedKeyed,
	"InterfaceType.declaredMembers":                        unmemoizedKeyed,
	"InterfaceType.declaredCallSignatures":                 unmemoizedKeyed,
	"InterfaceType.declaredConstructSignatures":            unmemoizedKeyed,
	"InterfaceType.declaredIndexInfos":                     unmemoizedKeyed,
	"TupleType.elementInfos":                               unmemoizedCreated,
	"TupleType.minLength":                                  unmemoizedCreated,
	"TupleType.fixedLength":                                unmemoizedCreated,
	"TupleType.combinedFlags":                              unmemoizedCreated,
	"TupleType.readonly":                                   unmemoizedCreated,
	"InstantiationExpressionType.node":                     unmemoizedCreated,
	"MappedType.declaration":                               unmemoizedCreated,
	"ReverseMappedType.source":                             unmemoizedCreated,
	"ReverseMappedType.mappedType":                         unmemoizedCreated,
	"ReverseMappedType.constraintType":                     unmemoizedCreated,
	"EvolvingArrayType.elementType":                        unmemoizedCreated,
	"UnionOrIntersectionType.types":                        unmemoizedCreated,
	"UnionType.regularType":                                unmemoizedInterned,
	"UnionType.origin":                                     unmemoizedCreated,
	"UnionType.constituentMap":                             unmemoizedGuarded,
	"TypeParameter.target":                                 unmemoizedCreated,
	"TypeParameter.mapper":                                 unmemoizedCreated,
	"TypeParameter.isThisType":                             unmemoizedCreated,
	"TypeParameter.isDistributed":                          unmemoizedCreated,
	"TypeParameter.distributedType":                        unmemoizedInterned,
	"TypeParameter.origin":                                 unmemoizedCreated,
	"IndexType.target":                                     unmemoizedCreated,
	"IndexType.indexFlags":                                 unmemoizedCreated,
	"IndexedAccessType.objectType":                         unmemoizedCreated,
	"IndexedAccessType.indexType":                          unmemoizedCreated,
	"IndexedAccessType.accessFlags":                        unmemoizedCreated,
	"TemplateLiteralType.texts":                            unmemoizedCreated,
	"TemplateLiteralType.types":                            unmemoizedCreated,
	"StringMappingType.target":                             unmemoizedCreated,
	"SubstitutionType.baseType":                            unmemoizedCreated,
	"SubstitutionType.constraint":                          unmemoizedCreated,
	"ConditionalRoot.node":                                 unmemoizedCreated,
	"ConditionalRoot.checkType":                            unmemoizedCreated,
	"ConditionalRoot.extendsType":                          unmemoizedCreated,
	"ConditionalRoot.isDistributive":                       unmemoizedCreated,
	"ConditionalRoot.inferTypeParameters":                  unmemoizedCreated,
	"ConditionalRoot.outerTypeParameters":                  unmemoizedCreated,
	"ConditionalRoot.alias":                                unmemoizedCreated,
	"ConditionalType.root":                                 unmemoizedCreated,
	"ConditionalType.checkType":                            unmemoizedCreated,
	"ConditionalType.extendsType":                          unmemoizedCreated,
	"ConditionalType.mapper":                               unmemoizedCreated,
	"ConditionalType.combinedMapper":                       unmemoizedCreated,
	"Signature.id":                                         unmemoizedCreated,
	"Signature.flags":                                      unmemoizedEntityFlags,
	"Signature.minArgumentCount":                           unmemoizedCreated,
	"Signature.declaration":                                unmemoizedCreated,
	"Signature.parameters":                                 unmemoizedCreated,
	"Signature.target":                                     unmemoizedCreated,
	"Signature.mapper":                                     unmemoizedCreated,
	"Signature.extra":                                      unmemoizedCreated,
	"signatureExtra.typeParameters":                        unmemoizedCreated,
	"signatureExtra.thisParameter":                         unmemoizedCreated,
	"signatureExtra.resolvedTypePredicate":                 unmemoizedKeyed,
	"signatureExtra.isolatedSignatureType":                 unmemoizedInterned,
	"signatureExtra.composite":                             unmemoizedCreated,
	"CompositeSignature.isUnion":                           unmemoizedCreated,
	"CompositeSignature.signatures":                        unmemoizedCreated,
	"TypePredicate.kind":                                   unmemoizedCreated,
	"TypePredicate.parameterIndex":                         unmemoizedCreated,
	"TypePredicate.parameterName":                          unmemoizedCreated,
	"TypePredicate.t":                                      unmemoizedCreated,
	"IndexInfo.keyType":                                    unmemoizedCreated,
	"IndexInfo.valueType":                                  unmemoizedCreated,
	"IndexInfo.isReadonly":                                 unmemoizedCreated,
	"IndexInfo.declaration":                                unmemoizedCreated,
	"IndexInfo.indexSymbol":                                unmemoizedInterned,
	"IndexInfo.components":                                 unmemoizedCreated,
	"checkerStore.root":                                    unmemoizedImmutable,
	"checkerStore.id":                                      unmemoizedImmutable,
	"checkerStore.program":                                 unmemoizedImmutable,
	"checkerStore.compilerOptions":                         unmemoizedImmutable,
	"checkerStore.files":                                   unmemoizedImmutable,
	"checkerStore.fileIndexMap":                            unmemoizedImmutable,
	"checkerStore.languageVersion":                         unmemoizedImmutable,
	"checkerStore.moduleKind":                              unmemoizedImmutable,
	"checkerStore.moduleResolutionKind":                    unmemoizedImmutable,
	"checkerStore.legacyDecorators":                        unmemoizedImmutable,
	"checkerStore.emitStandardClassFields":                 unmemoizedImmutable,
	"checkerStore.strictNullChecks":                        unmemoizedImmutable,
	"checkerStore.strictFunctionTypes":                     unmemoizedImmutable,
	"checkerStore.strictBindCallApply":                     unmemoizedImmutable,
	"checkerStore.strictPropertyInitialization":            unmemoizedImmutable,
	"checkerStore.strictBuiltinIteratorReturn":             unmemoizedImmutable,
	"checkerStore.noImplicitAny":                           unmemoizedImmutable,
	"checkerStore.noImplicitThis":                          unmemoizedImmutable,
	"checkerStore.useUnknownInCatchVariables":              unmemoizedImmutable,
	"checkerStore.exactOptionalPropertyTypes":              unmemoizedImmutable,
	"checkerStore.canCollectSymbolAliasAccessibilityData":  unmemoizedImmutable,
	"checkerStore.arrayVariances":                          unmemoizedImmutable,
	"checkerStore.globals":                                 unmemoizedImmutable,
	"checkerStore.undefinedSymbol":                         unmemoizedImmutable,
	"checkerStore.argumentsSymbol":                         unmemoizedImmutable,
	"checkerStore.requireSymbol":                           unmemoizedImmutable,
	"checkerStore.unknownSymbol":                           unmemoizedImmutable,
	"checkerStore.globalThisSymbol":                        unmemoizedImmutable,
	"checkerStore.mergedSymbols":                           unmemoizedImmutable,
	"checkerStore.mergesFrozen":                            unmemoizedImmutable,
	"checkerStore.anyType":                                 unmemoizedImmutable,
	"checkerStore.autoType":                                unmemoizedImmutable,
	"checkerStore.wildcardType":                            unmemoizedImmutable,
	"checkerStore.blockedStringType":                       unmemoizedImmutable,
	"checkerStore.errorType":                               unmemoizedImmutable,
	"checkerStore.unresolvedType":                          unmemoizedImmutable,
	"checkerStore.nonInferrableAnyType":                    unmemoizedImmutable,
	"checkerStore.intrinsicMarkerType":                     unmemoizedImmutable,
	"checkerStore.unknownType":                             unmemoizedImmutable,
	"checkerStore.undefinedType":                           unmemoizedImmutable,
	"checkerStore.undefinedWideningType":                   unmemoizedImmutable,
	"checkerStore.missingType":                             unmemoizedImmutable,
	"checkerStore.undefinedOrMissingType":                  unmemoizedImmutable,
	"checkerStore.optionalType":                            unmemoizedImmutable,
	"checkerStore.nullType":                                unmemoizedImmutable,
	"checkerStore.nullWideningType":                        unmemoizedImmutable,
	"checkerStore.stringType":                              unmemoizedImmutable,
	"checkerStore.numberType":                              unmemoizedImmutable,
	"checkerStore.bigintType":                              unmemoizedImmutable,
	"checkerStore.regularFalseType":                        unmemoizedImmutable,
	"checkerStore.falseType":                               unmemoizedImmutable,
	"checkerStore.regularTrueType":                         unmemoizedImmutable,
	"checkerStore.trueType":                                unmemoizedImmutable,
	"checkerStore.booleanType":                             unmemoizedImmutable,
	"checkerStore.esSymbolType":                            unmemoizedImmutable,
	"checkerStore.voidType":                                unmemoizedImmutable,
	"checkerStore.neverType":                               unmemoizedImmutable,
	"checkerStore.silentNeverType":                         unmemoizedImmutable,
	"checkerStore.implicitNeverType":                       unmemoizedImmutable,
	"checkerStore.unreachableNeverType":                    unmemoizedImmutable,
	"checkerStore.nonPrimitiveType":                        unmemoizedImmutable,
	"checkerStore.stringOrNumberType":                      unmemoizedImmutable,
	"checkerStore.stringNumberSymbolType":                  unmemoizedImmutable,
	"checkerStore.numberOrBigIntType":                      unmemoizedImmutable,
	"checkerStore.templateConstraintType":                  unmemoizedImmutable,
	"checkerStore.numericStringType":                       unmemoizedImmutable,
	"checkerStore.uniqueLiteralType":                       unmemoizedImmutable,
	"checkerStore.emptyObjectType":                         unmemoizedImmutable,
	"checkerStore.emptyJsxObjectType":                      unmemoizedImmutable,
	"checkerStore.emptyFreshJsxObjectType":                 unmemoizedImmutable,
	"checkerStore.emptyTypeLiteralType":                    unmemoizedImmutable,
	"checkerStore.unknownEmptyObjectType":                  unmemoizedImmutable,
	"checkerStore.unknownUnionType":                        unmemoizedImmutable,
	"checkerStore.emptyGenericType":                        unmemoizedImmutable,
	"checkerStore.anyFunctionType":                         unmemoizedImmutable,
	"checkerStore.noConstraintType":                        unmemoizedImmutable,
	"checkerStore.circularConstraintType":                  unmemoizedImmutable,
	"checkerStore.resolvingDefaultType":                    unmemoizedImmutable,
	"checkerStore.markerSuperType":                         unmemoizedImmutable,
	"checkerStore.markerSubType":                           unmemoizedImmutable,
	"checkerStore.markerOtherType":                         unmemoizedImmutable,
	"checkerStore.markerSuperTypeForCheck":                 unmemoizedImmutable,
	"checkerStore.markerSubTypeForCheck":                   unmemoizedImmutable,
	"checkerStore.noTypePredicate":                         unmemoizedImmutable,
	"checkerStore.anySignature":                            unmemoizedImmutable,
	"checkerStore.unknownSignature":                        unmemoizedImmutable,
	"checkerStore.resolvingSignature":                      unmemoizedImmutable,
	"checkerStore.silentNeverSignature":                    unmemoizedImmutable,
	"checkerStore.enumNumberIndexInfo":                     unmemoizedImmutable,
	"checkerStore.anyBaseTypeIndexInfo":                    unmemoizedImmutable,
	"checkerStore.patternAmbientModules":                   unmemoizedImmutable,
	"checkerStore.patternAmbientModuleAugmentations":       unmemoizedImmutable,
	"checkerStore.patternAmbientModuleAugmentationTargets": unmemoizedImmutable,
	"checkerStore.globalObjectType":                        unmemoizedImmutable,
	"checkerStore.globalFunctionType":                      unmemoizedImmutable,
	"checkerStore.globalCallableFunctionType":              unmemoizedImmutable,
	"checkerStore.globalNewableFunctionType":               unmemoizedImmutable,
	"checkerStore.globalArrayType":                         unmemoizedImmutable,
	"checkerStore.globalReadonlyArrayType":                 unmemoizedImmutable,
	"checkerStore.globalStringType":                        unmemoizedImmutable,
	"checkerStore.globalNumberType":                        unmemoizedImmutable,
	"checkerStore.globalBooleanType":                       unmemoizedImmutable,
	"checkerStore.globalRegExpType":                        unmemoizedImmutable,
	"checkerStore.globalThisType":                          unmemoizedImmutable,
	"checkerStore.anyArrayType":                            unmemoizedImmutable,
	"checkerStore.autoArrayType":                           unmemoizedImmutable,
	"checkerStore.anyReadonlyArrayType":                    unmemoizedImmutable,
	"checkerStore.emptyStringType":                         unmemoizedImmutable,
	"checkerStore.zeroType":                                unmemoizedImmutable,
	"checkerStore.zeroBigIntType":                          unmemoizedImmutable,
	"checkerStore.typeofType":                              unmemoizedImmutable,
	"checkerStore.subtypeRelation":                         unmemoizedImmutable,
	"checkerStore.strictSubtypeRelation":                   unmemoizedImmutable,
	"checkerStore.assignableRelation":                      unmemoizedImmutable,
	"checkerStore.comparableRelation":                      unmemoizedImmutable,
	"checkerStore.identityRelation":                        unmemoizedImmutable,
	"checkerStore.nodeLinks":                               unmemoizedContainer,
	"checkerStore.signatureLinks":                          unmemoizedContainer,
	"checkerStore.symbolNodeLinks":                         unmemoizedContainer,
	"checkerStore.typeNodeLinks":                           unmemoizedContainer,
	"checkerStore.enumMemberLinks":                         unmemoizedContainer,
	"checkerStore.arrayLiteralLinks":                       unmemoizedContainer,
	"checkerStore.switchStatementLinks":                    unmemoizedContainer,
	"checkerStore.jsxElementLinks":                         unmemoizedContainer,
	"checkerStore.computedNameLinks":                       unmemoizedContainer,
	"checkerStore.symbolReferenceLinks":                    unmemoizedContainer,
	"checkerStore.valueSymbolLinks":                        unmemoizedContainer,
	"checkerStore.mappedSymbolLinks":                       unmemoizedContainer,
	"checkerStore.deferredSymbolLinks":                     unmemoizedContainer,
	"checkerStore.aliasSymbolLinks":                        unmemoizedContainer,
	"checkerStore.moduleSymbolLinks":                       unmemoizedContainer,
	"checkerStore.lateBoundLinks":                          unmemoizedContainer,
	"checkerStore.exportTypeLinks":                         unmemoizedContainer,
	"checkerStore.membersAndExportsLinks":                  unmemoizedContainer,
	"checkerStore.typeAliasLinks":                          unmemoizedContainer,
	"checkerStore.declaredTypeLinks":                       unmemoizedContainer,
	"checkerStore.spreadLinks":                             unmemoizedContainer,
	"checkerStore.varianceLinks":                           unmemoizedContainer,
	"checkerStore.ReverseMappedSymbolLinks":                unmemoizedContainer,
	"checkerStore.markedAssignmentSymbolLinks":             unmemoizedContainer,
	"checkerStore.sourceFileLinks":                         unmemoizedContainer,
	"checkerStore.emitResolverLinks":                       unmemoizedContainer,
	"EmitResolverLinks.jsxLinks":                           unmemoizedContainer,
	"EmitResolverLinks.declarationLinks":                   unmemoizedContainer,
	"EmitResolverLinks.declarationFileLinks":               unmemoizedContainer,
	"checkerStore.TypeCount":                               unmemoizedCounter,
	"checkerStore.SymbolCount":                             unmemoizedCounter,
	"checkerStore.SignatureCount":                          unmemoizedCounter,
	"checkerStore.TotalInstantiationCount":                 unmemoizedCounter,
	"checkerStore.nanType":                                 unmemoizedInterned,
	"checkerStore.packagesMap":                             unmemoizedProgram,
	"checkerStore.ambientModulesOnce":                      unmemoizedProgram,
	"checkerStore.ambientModules":                          unmemoizedProgram,
	"checkerStore.memoSlots":                               unmemoizedMemoState,
	"checkerStore.memoPendingCount":                        unmemoizedMemoState,
	"checkerStore.memoReportSlots":                         unmemoizedMemoState,
	"checkerStore.workFilter":                              unmemoizedMemoState,
	"checkerStore.mu":                                      unmemoizedMemoState,
	"checkerStore.diagnostics":                             unmemoizedReporting,
	"checkerStore.suggestionDiagnostics":                   unmemoizedReporting,
	"checkerStore.nonExistentProperties":                   unmemoizedReporting,
	"checkerStore.deferredDiagnosticCallbacks":             unmemoizedReporting,
	"SymbolReferenceLinks.referenceKinds":                  unmemoizedReporting,
	"AliasSymbolLinks.referenced":                          unmemoizedMark,
	"AliasSymbolLinks.referenceChecked":                    unmemoizedReporting,
	"SourceFileLinks.typeChecked":                          unmemoizedReporting,
	"SourceFileLinks.unusedChecked":                        unmemoizedReporting,
	"SourceFileLinks.deferredNodes":                        unmemoizedReporting,
	"SourceFileLinks.identifierCheckNodes":                 unmemoizedReporting,
	"ValueSymbolLinks.target":                              unmemoizedCreated,
	"ValueSymbolLinks.mapper":                              unmemoizedCreated,
	"ValueSymbolLinks.extra":                               unmemoizedCreated,
	"MappedSymbolLinks.keyType":                            unmemoizedCreated,
	"MappedSymbolLinks.syntheticOrigin":                    unmemoizedCreated,
	"DeferredSymbolLinks.parent":                           unmemoizedCreated,
	"DeferredSymbolLinks.constituents":                     unmemoizedCreated,
	"DeferredSymbolLinks.writeConstituents":                unmemoizedCreated,
	"ReverseMappedSymbolLinks.propertyType":                unmemoizedCreated,
	"ReverseMappedSymbolLinks.mappedType":                  unmemoizedCreated,
	"ReverseMappedSymbolLinks.constraintType":              unmemoizedCreated,
	"ExportTypeLinks.target":                               unmemoizedCreated,
	"ExportTypeLinks.originatingImport":                    unmemoizedCreated,
	"SpreadLinks.leftSpread":                               unmemoizedCreated,
	"SpreadLinks.rightSpread":                              unmemoizedCreated,
	"JsxElementLinks.firstJSXTagInFile":                    unmemoizedSyntax,
	"checkerStore._jsxNamespace":                           unmemoizedSyntax,
	"checkerStore._jsxFactoryEntity":                       unmemoizedSyntax,
	"SwitchStatementLinks.witnessesComputed":               unmemoizedSyntax,
	"SwitchStatementLinks.witnesses":                       unmemoizedSyntax,
	"ArrayLiteralLinks.indicesComputed":                    unmemoizedSyntax,
	"ArrayLiteralLinks.firstSpreadIndex":                   unmemoizedSyntax,
	"ArrayLiteralLinks.lastSpreadIndex":                    unmemoizedSyntax,
	"NodeLinks.declarationRequiresScopeChange":             unmemoizedSyntax,
	"SourceFileLinks.localJsxNamespace":                    unmemoizedSyntax,
	"SourceFileLinks.localJsxFragmentNamespace":            unmemoizedSyntax,
	"SourceFileLinks.localJsxFactory":                      unmemoizedSyntax,
	"SourceFileLinks.localJsxFragmentFactory":              unmemoizedSyntax,
	"JsxElementLinks.jsxFlags":                             unmemoizedGuarded,
	"JsxElementLinks.jsxImplicitImportContainer":           unmemoizedGuarded,
	"ModuleSymbolLinks.resolvedExports":                    unmemoizedGuarded,
	"ModuleSymbolLinks.typeOnlyExportStarMap":              unmemoizedGuarded,
	"LateBoundLinks.lateSymbol":                            unmemoizedGuarded,
	"TypeAliasLinks.typeParameters":                        unmemoizedGuarded,
	"MarkedAssignmentSymbolLinks.lastAssignmentPos":        unmemoizedGuarded,
	"MarkedAssignmentSymbolLinks.hasDefiniteAssignment":    unmemoizedGuarded,
	"EnumMemberLinks.value":                                unmemoizedGuarded,
	"JSXLinks.importRef":                                   unmemoizedEmit,
	"DeclarationLinks.isVisible":                           unmemoizedEmit,
	"DeclarationFileLinks.aliasesMarked":                   unmemoizedEmit,
	"NodeBuilderLinks.serializedTypes":                     unmemoizedNodeBuilder,
	"NodeBuilderLinks.fakeScopeForSignatureDeclaration":    unmemoizedNodeBuilder,
	"NodeBuilderSymbolLinks.specifierCache":                unmemoizedNodeBuilder,
	"SourceFileLinks.externalHelpersModule":                unmemoizedReporting,
	"SourceFileLinks.requestedExternalEmitHelpers":         unmemoizedReporting,
	"AliasSymbolLinks.typeOnlyDeclaration":                 unmemoizedGuarded,
	"NodeLinks.flags":                                      unmemoizedNodeFlags,
}
