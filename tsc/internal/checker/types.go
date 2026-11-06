package checker

import (
	"math/bits"
	"slices"
	"strings"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/collections"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/evaluator"
)

//go:generate npx hereby generate:checker

// ParseFlags

type ParseFlags uint32

const (
	ParseFlagsNone                   ParseFlags = 0
	ParseFlagsYield                  ParseFlags = 1 << 0
	ParseFlagsAwait                  ParseFlags = 1 << 1
	ParseFlagsType                   ParseFlags = 1 << 2
	ParseFlagsIgnoreMissingOpenBrace ParseFlags = 1 << 4
	ParseFlagsJSDoc                  ParseFlags = 1 << 5
)

type SignatureKind int32

const (
	SignatureKindCall SignatureKind = iota
	SignatureKindConstruct
)

type IndexKind int32

const (
	IndexKindString IndexKind = iota
	IndexKindNumber
)

type MemberOverrideStatus int32

const (
	MemberOverrideStatusNone MemberOverrideStatus = iota
	MemberOverrideStatusNeedsOverride
	MemberOverrideStatusHasInvalidOverride
)

type ContextFlags uint32

const (
	ContextFlagsNone                 ContextFlags = 0
	ContextFlagsSignature            ContextFlags = 1 << 0 // Obtaining contextual signature
	ContextFlagsNoConstraints        ContextFlags = 1 << 1 // Don't obtain type variable constraints
	ContextFlagsIgnoreNodeInferences ContextFlags = 1 << 2 // Ignore inference to current node and parent nodes out to the containing call for, for example, completions
	ContextFlagsSkipBindingPatterns  ContextFlags = 1 << 3 // Ignore contextual types applied by binding patterns
)

type TypeFormatFlags uint32

const (
	TypeFormatFlagsNone                               TypeFormatFlags = 0
	TypeFormatFlagsNoTruncation                       TypeFormatFlags = 1 << 0 // Don't truncate typeToString result
	TypeFormatFlagsWriteArrayAsGenericType            TypeFormatFlags = 1 << 1 // Write Array<T> instead T[]
	TypeFormatFlagsGenerateNamesForShadowedTypeParams TypeFormatFlags = 1 << 2 // When a type parameter T is shadowing another T, generate a name for it so it can still be referenced
	TypeFormatFlagsUseStructuralFallback              TypeFormatFlags = 1 << 3 // When an alias cannot be named by its symbol, rather than report an error, fallback to a structural printout if possible
	// hole because there's a hole in node builder flags
	TypeFormatFlagsWriteTypeArgumentsOfSignature TypeFormatFlags = 1 << 5 // Write the type arguments instead of type parameters of the signature
	TypeFormatFlagsUseFullyQualifiedType         TypeFormatFlags = 1 << 6 // Write out the fully qualified type name (eg. Module.Type, instead of Type)
	// hole because `UseOnlyExternalAliasing` is here in node builder flags, but functions which take old flags use `SymbolFormatFlags` instead
	TypeFormatFlagsSuppressAnyReturnType TypeFormatFlags = 1 << 8 // If the return type is any-like, don't offer a return type.
	// hole because `WriteTypeParametersInQualifiedName` is here in node builder flags, but functions which take old flags use `SymbolFormatFlags` for this instead
	TypeFormatFlagsMultilineObjectLiterals             TypeFormatFlags = 1 << 10 // Always print object literals across multiple lines (only used to map into node builder flags)
	TypeFormatFlagsWriteClassExpressionAsTypeLiteral   TypeFormatFlags = 1 << 11 // Write a type literal instead of (Anonymous class)
	TypeFormatFlagsUseTypeOfFunction                   TypeFormatFlags = 1 << 12 // Write typeof instead of function type literal
	TypeFormatFlagsOmitParameterModifiers              TypeFormatFlags = 1 << 13 // Omit modifiers on parameters
	TypeFormatFlagsUseAliasDefinedOutsideCurrentScope  TypeFormatFlags = 1 << 14 // For a `type T = ... ` defined in a different file, write `T` instead of its value, even though `T` can't be accessed in the current scope.
	TypeFormatFlagsUseSingleQuotesForStringLiteralType TypeFormatFlags = 1 << 28 // Use single quotes for string literal type
	TypeFormatFlagsNoTypeReduction                     TypeFormatFlags = 1 << 29 // Don't call getReducedType
	TypeFormatFlagsUseInstantiationExpressions         TypeFormatFlags = 1 << 30 // Use instantiation expressions for qualified instantiated names like Foo<string>.Bar
	TypeFormatFlagsOmitThisParameter                   TypeFormatFlags = 1 << 25
	TypeFormatFlagsWriteCallStyleSignature             TypeFormatFlags = 1 << 27 // Write construct signatures as call style signatures
	// Error Handling
	TypeFormatFlagsAllowUniqueESSymbolType TypeFormatFlags = 1 << 20 // This is bit 20 to align with the same bit in `NodeBuilderFlags`
	// TypeFormatFlags exclusive
	TypeFormatFlagsAddUndefined             TypeFormatFlags = 1 << 17 // Add undefined to types of initialized, non-optional parameters
	TypeFormatFlagsWriteArrowStyleSignature TypeFormatFlags = 1 << 18 // Write arrow style signature
	// State
	TypeFormatFlagsInArrayType         TypeFormatFlags = 1 << 19 // Writing an array element type
	TypeFormatFlagsInElementType       TypeFormatFlags = 1 << 21 // Writing an array or union element type
	TypeFormatFlagsInFirstTypeArgument TypeFormatFlags = 1 << 22 // Writing first type argument of the instantiated type
	TypeFormatFlagsInTypeAlias         TypeFormatFlags = 1 << 23 // Writing type in type alias declaration

	TypeFormatFlagsNodeBuilderFlagsMask = TypeFormatFlagsNoTruncation | TypeFormatFlagsWriteArrayAsGenericType | TypeFormatFlagsGenerateNamesForShadowedTypeParams | TypeFormatFlagsUseStructuralFallback | TypeFormatFlagsWriteTypeArgumentsOfSignature |
		TypeFormatFlagsUseFullyQualifiedType | TypeFormatFlagsSuppressAnyReturnType | TypeFormatFlagsMultilineObjectLiterals | TypeFormatFlagsWriteClassExpressionAsTypeLiteral |
		TypeFormatFlagsUseTypeOfFunction | TypeFormatFlagsOmitParameterModifiers | TypeFormatFlagsUseAliasDefinedOutsideCurrentScope | TypeFormatFlagsAllowUniqueESSymbolType | TypeFormatFlagsInTypeAlias |
		TypeFormatFlagsUseInstantiationExpressions | TypeFormatFlagsUseSingleQuotesForStringLiteralType | TypeFormatFlagsNoTypeReduction | TypeFormatFlagsOmitThisParameter
)

type SymbolFormatFlags uint32

const (
	SymbolFormatFlagsNone SymbolFormatFlags = 0
	// Write symbols's type argument if it is instantiated symbol
	// eg. class C<T> { p: T }   <-- Show p as C<T>.p here
	//     var a: C<number>;
	//     var p = a.p; <--- Here p is property of C<number> so show it as C<number>.p instead of just C.p
	SymbolFormatFlagsWriteTypeParametersOrArguments SymbolFormatFlags = 1 << 0
	// Use only external alias information to get the symbol name in the given context
	// eg.  module m { export class c { } } import x = m.c;
	// When this flag is specified m.c will be used to refer to the class instead of alias symbol x
	SymbolFormatFlagsUseOnlyExternalAliasing SymbolFormatFlags = 1 << 1
	// Build symbol name using any nodes needed, instead of just components of an entity name
	SymbolFormatFlagsAllowAnyNodeKind SymbolFormatFlags = 1 << 2
	// Prefer aliases which are not directly visible
	SymbolFormatFlagsUseAliasDefinedOutsideCurrentScope SymbolFormatFlags = 1 << 3
	// { [E.A]: 1 }
	/** @internal */
	SymbolFormatFlagsWriteComputedProps SymbolFormatFlags = 1 << 4
	// Skip building an accessible symbol chain
	/** @internal */
	SymbolFormatFlagsDoNotIncludeSymbolChain SymbolFormatFlags = 1 << 5
)

type ExternalEmitHelpers uint32

const (
	ExternalEmitHelpersRest                                     ExternalEmitHelpers           = 1 << iota // __rest (used by ESNext object rest transformation)
	ExternalEmitHelpersDecorate                                                                           // __decorate (used by TypeScript decorators transformation)
	ExternalEmitHelpersMetadata                                                                           // __metadata (used by TypeScript decorators transformation)
	ExternalEmitHelpersParam                                                                              // __param (used by TypeScript decorators transformation)
	ExternalEmitHelpersAwaiter                                                                            // __awaiter (used by ES2017 async functions transformation)
	ExternalEmitHelpersAwait                                                                              // __await (used by ES2017 async generator transformation)
	ExternalEmitHelpersAsyncGenerator                                                                     // __asyncGenerator (used by ES2017 async generator transformation)
	ExternalEmitHelpersAsyncDelegator                                                                     // __asyncDelegator (used by ES2017 async generator yield* transformation)
	ExternalEmitHelpersAsyncValues                                                                        // __asyncValues (used by ES2017 for..await..of transformation)
	ExternalEmitHelpersExportStar                                                                         // __exportStar (used by CommonJS/AMD/UMD module transformation)
	ExternalEmitHelpersImportStar                                                                         // __importStar (used by CommonJS/AMD/UMD module transformation)
	ExternalEmitHelpersImportDefault                                                                      // __importDefault (used by CommonJS/AMD/UMD module transformation)
	ExternalEmitHelpersMakeTemplateObject                                                                 // __makeTemplateObject (used for constructing template string array objects)
	ExternalEmitHelpersClassPrivateFieldGet                                                               // __classPrivateFieldGet (used by the class private field transformation)
	ExternalEmitHelpersClassPrivateFieldSet                                                               // __classPrivateFieldSet (used by the class private field transformation)
	ExternalEmitHelpersClassPrivateFieldIn                                                                // __classPrivateFieldIn (used by the class private field transformation)
	ExternalEmitHelpersSetFunctionName                                                                    // __setFunctionName (used by class fields and ECMAScript decorators)
	ExternalEmitHelpersPropKey                                                                            // __propKey (used by class fields and ECMAScript decorators)
	ExternalEmitHelpersAddDisposableResourceAndDisposeResources                                           // __addDisposableResource and __disposeResources (used by ESNext transformations)
	ExternalEmitHelpersRewriteRelativeImportExtension                                                     // __rewriteRelativeImportExtension (used by --rewriteRelativeImportExtensions)
	ExternalEmitHelpersESDecorateAndRunInitializers             = ExternalEmitHelpersDecorate             // __esDecorate and __runInitializers (used by ECMAScript decorators transformation)

	ExternalEmitHelpersFirstEmitHelper = ExternalEmitHelpersRest
	ExternalEmitHelpersLastEmitHelper  = ExternalEmitHelpersRewriteRelativeImportExtension

	// Helpers included by ES2017 for..await..of
	ExternalEmitHelpersForAwaitOfIncludes = ExternalEmitHelpersAsyncValues

	// Helpers included by ES2017 async generators
	ExternalEmitHelpersAsyncGeneratorIncludes = ExternalEmitHelpersAwait | ExternalEmitHelpersAsyncGenerator

	// Helpers included by yield* in ES2017 async generators
	ExternalEmitHelpersAsyncDelegatorIncludes = ExternalEmitHelpersAwait | ExternalEmitHelpersAsyncDelegator | ExternalEmitHelpersAsyncValues
)

const externalHelpersModuleNameText = "tslib"

// Ids

type (
	TypeId      uint32
	SignatureId uint32
)

// Links for value symbols

type ValueSymbolLinks struct {
	resolvedType TypeId // Type of value symbol
	target       ast.SymbolId
	mapper       TypeMapperId
	extra        uint32 // Index+1 into Checker.valueSymbolExtras, 0 = none
}

type valueSymbolExtra struct {
	writeType      TypeId
	nameType       TypeId
	containingType TypeId // Mapped type for mapped type property, containing union or intersection type for synthetic property
}

func (links *ValueSymbolLinks) getWriteType(c *Checker) *Type {
	if links.extra == 0 {
		return nil
	}
	return c.typ(pageAt(&c.valueSymbolExtras, links.extra).writeType)
}

func (links *ValueSymbolLinks) getNameType(c *Checker) *Type {
	if links.extra == 0 {
		return nil
	}
	return c.typ(pageAt(&c.valueSymbolExtras, links.extra).nameType)
}

func (links *ValueSymbolLinks) getContainingType(c *Checker) *Type {
	if links.extra == 0 {
		return nil
	}
	return c.typ(pageAt(&c.valueSymbolExtras, links.extra).containingType)
}

func (links *ValueSymbolLinks) ensureExtra(c *Checker) *valueSymbolExtra {
	if links.extra == 0 {
		i, extra := pageAdd(&c.valueSymbolExtras)
		links.extra = i
		return extra
	}
	return pageAt(&c.valueSymbolExtras, links.extra)
}

func (links *ValueSymbolLinks) setWriteType(c *Checker, typ *Type) {
	if links.extra == 0 && typ == nil {
		return
	}
	links.ensureExtra(c).writeType = typ.ref()
}

func (links *ValueSymbolLinks) setNameType(c *Checker, typ *Type) {
	if links.extra == 0 && typ == nil {
		return
	}
	links.ensureExtra(c).nameType = typ.ref()
}

func (links *ValueSymbolLinks) setContainingType(c *Checker, typ *Type) {
	if links.extra == 0 && typ == nil {
		return
	}
	links.ensureExtra(c).containingType = typ.ref()
}

// Additional links for mapped symbols

type MappedSymbolLinks struct {
	keyType         TypeId       // Key type for mapped type member
	syntheticOrigin ast.SymbolId // For a property on a mapped or spread type, points back to the original property
}

// Additional links for deferred type symbols

type DeferredSymbolLinks struct {
	parent            TypeId   // Source union/intersection of a deferred type
	constituents      []TypeId // Calculated list of constituents for a deferred type
	writeConstituents []TypeId // Constituents of a deferred `writeType`
}

// Links for alias symbols

type AliasSymbolLinks struct {
	immediateTarget     ast.SymbolId // Immediate target of an alias. May be another alias. Do not access directly, use `checker.getImmediateAliasedSymbol` instead.
	aliasTarget         ast.SymbolId // Resolved (non-alias) target of an alias
	referenced          bool         // True if alias symbol has been referenced as a value that can be emitted
	typeOnlyDeclaration ast.Node     // First resolved alias declaration that makes the symbol only usable in type constructs
}

// Links for module symbols

type ModuleSymbolLinks struct {
	resolvedExports       ast.SymbolTable             // Resolved exports of module or combined early- and late-bound static members of a class.
	typeOnlyExportStarMap map[ast.SymbolName]ast.Node // Set on a module symbol when some of its exports were resolved through a 'export type * from "mod"' declaration
	exportsChecked        bool
}

type ReverseMappedSymbolLinks struct {
	propertyType   TypeId
	mappedType     TypeId // References a mapped type
	constraintType TypeId // References an index type
}

// Links for late-bound symbols

type LateBoundLinks struct {
	lateSymbol ast.SymbolId
}

// Links for export type symbols

type ExportTypeLinks struct {
	target            ast.SymbolId // Target symbol
	originatingImport ast.Node     // Import declaration which produced the symbol, present if the symbol is marked as uncallable but had call signatures in `resolveESModuleSymbol`
}

// Links for type aliases

type TypeAliasLinks struct {
	declaredType                  TypeId
	typeParameters                []TypeId                // Type parameters of type alias (undefined if non-generic)
	instantiations                map[CacheHashKey]TypeId // Instantiations of generic type alias (undefined if non-generic)
	isConstructorDeclaredProperty bool
}

// Links for declared types (type parameters, class types, interface types, enums)

type DeclaredTypeLinks struct {
	declaredType           TypeId
	interfaceChecked       bool
	indexSignaturesChecked bool
	typeParametersChecked  bool
	enumChecked            bool
}

// Links for switch clauses

type ExhaustiveState byte

const (
	ExhaustiveStateUnknown   ExhaustiveState = iota // Exhaustive state not computed
	ExhaustiveStateComputing                        // Exhaustive state computation in progress
	ExhaustiveStateFalse                            // Switch statement is not exhaustive
	ExhaustiveStateTrue                             // Switch statement is exhaustive
)

type SwitchStatementLinks struct {
	exhaustiveState     ExhaustiveState // Switch statement exhaustiveness
	switchTypesComputed bool
	witnessesComputed   bool
	switchTypes         []TypeId
	witnesses           []string
}

type ArrayLiteralLinks struct {
	indicesComputed  bool
	firstSpreadIndex int // Index of first spread expression (or -1 if none)
	lastSpreadIndex  int // Index of last spread expression (or -1 if none)
}

// Links for late-binding containers

type MembersOrExportsResolutionKind int

const (
	MembersOrExportsResolutionKindResolvedExports MembersOrExportsResolutionKind = 0
	MembersOrExportsResolutionKindResolvedMembers MembersOrExportsResolutionKind = 1
)

type MembersAndExportsLinks [2]ast.SymbolTable // Indexed by MembersOrExportsResolutionKind

// Links for synthetic spread properties

type SpreadLinks struct {
	leftSpread  ast.SymbolId // Left source for synthetic spread property
	rightSpread ast.SymbolId // Right source for synthetic spread property
}

// Links for variances of type aliases and interface types

type VarianceLinks struct {
	variances []VarianceFlags
}

type VarianceFlags uint32

const (
	VarianceFlagsInvariant                VarianceFlags = 0                                                                                                       // Neither covariant nor contravariant
	VarianceFlagsCovariant                VarianceFlags = 1 << 0                                                                                                  // Covariant
	VarianceFlagsContravariant            VarianceFlags = 1 << 1                                                                                                  // Contravariant
	VarianceFlagsBivariant                VarianceFlags = VarianceFlagsCovariant | VarianceFlagsContravariant                                                     // Both covariant and contravariant
	VarianceFlagsIndependent              VarianceFlags = 1 << 2                                                                                                  // Unwitnessed type parameter
	VarianceFlagsVarianceMask             VarianceFlags = VarianceFlagsInvariant | VarianceFlagsCovariant | VarianceFlagsContravariant | VarianceFlagsIndependent // Mask containing all measured variances without the unmeasurable flag
	VarianceFlagsUnmeasurable             VarianceFlags = 1 << 3                                                                                                  // Variance result is unusable - relationship relies on structural comparisons which are not reflected in generic relationships
	VarianceFlagsUnreliable               VarianceFlags = 1 << 4                                                                                                  // Variance result is unreliable - checking may produce false negatives, but not false positives
	VarianceFlagsAllowsStructuralFallback               = VarianceFlagsUnmeasurable | VarianceFlagsUnreliable
)

type MarkedAssignmentSymbolLinks struct {
	lastAssignmentPos     int32
	hasDefiniteAssignment bool // Symbol is definitely assigned somewhere
}

type accessibleChainCacheKey struct {
	useOnlyExternalAliasing bool
	location                ast.Node
	meaning                 ast.SymbolFlags
}

type ContainingSymbolLinks struct {
	extendedContainersByFile map[ast.NodeId][]ast.SymbolId // Symbols of nodes which which logically contain this one, cached by file the request is made within
	extendedContainers       *[]ast.SymbolId               // Containers (other than the parent) which this symbol is aliased in
	accessibleChainCache     map[accessibleChainCacheKey][]ast.SymbolId
}

type AccessFlags uint32

const (
	AccessFlagsNone                       AccessFlags = 0
	AccessFlagsIncludeUndefined           AccessFlags = 1 << 0
	AccessFlagsNoIndexSignatures          AccessFlags = 1 << 1
	AccessFlagsWriting                    AccessFlags = 1 << 2
	AccessFlagsCacheSymbol                AccessFlags = 1 << 3
	AccessFlagsAllowMissing               AccessFlags = 1 << 4
	AccessFlagsExpressionPosition         AccessFlags = 1 << 5
	AccessFlagsReportDeprecated           AccessFlags = 1 << 6
	AccessFlagsSuppressNoImplicitAnyError AccessFlags = 1 << 7
	AccessFlagsContextual                 AccessFlags = 1 << 8
	AccessFlagsPersistent                             = AccessFlagsIncludeUndefined
)

type NodeCheckFlags uint32

const (
	NodeCheckFlagsNone                                     NodeCheckFlags = 0
	NodeCheckFlagsTypeChecked                              NodeCheckFlags = 1 << 0  // Node has been type checked
	NodeCheckFlagsContextChecked                           NodeCheckFlags = 1 << 6  // Contextual types have been assigned
	NodeCheckFlagsEnumValuesComputed                       NodeCheckFlags = 1 << 10 // Values for enum members have been computed, and any errors have been reported for them.
	NodeCheckFlagsAssignmentsMarked                        NodeCheckFlags = 1 << 17 // Parameter assignments have been marked
	NodeCheckFlagsContainsClassWithPrivateIdentifiers      NodeCheckFlags = 1 << 20 // Marked on all block-scoped containers containing a class with private identifiers.
	NodeCheckFlagsContainsSuperPropertyInStaticInitializer NodeCheckFlags = 1 << 21 // Marked on all block-scoped containers containing a static initializer with 'super.x' or 'super[x]'.
	NodeCheckFlagsInCheckIdentifier                        NodeCheckFlags = 1 << 22
	NodeCheckFlagsInitializerIsUndefined                   NodeCheckFlags = 1 << 24
	NodeCheckFlagsInitializerIsUndefinedComputed           NodeCheckFlags = 1 << 25
)

// Common links

type NodeLinks struct {
	flags                                NodeCheckFlags // Set of flags specific to Node
	declarationRequiresScopeChange       core.Tristate  // Set by `useOuterVariableScopeInParameter` in checker when downlevel emit would change the name resolution scope inside of a parameter.
	hasReportedStatementInAmbientContext bool           // Cache boolean if we report statements in ambient context
}

type ComputedNameNodeLinks struct {
	hasName core.Tristate  // If the node has a computable name
	name    ast.SymbolName // Resolved name associated with the type of the node
}

// Links for enum members

type EnumMemberLinks struct {
	value evaluator.Result // Constant value of enum member
}

// Links for assertion expressions

type AssertionLinks struct {
	exprType TypeId // Assertion expression type
}

// SourceFile links

type SourceFileLinks struct {
	typeChecked                  bool
	unusedChecked                bool
	externalHelpersModule        ast.SymbolId
	requestedExternalEmitHelpers ExternalEmitHelpers
	deferredNodes                collections.OrderedSet[ast.Node]
	identifierCheckNodes         []ast.Node
	localJsxNamespace            string
	localJsxFragmentNamespace    string
	localJsxFactory              ast.EntityName
	localJsxFragmentFactory      ast.EntityName
	jsxFragmentType              TypeId
}

type TypeFlags uint32

// Note that for types of different kinds, the numeric values of TypeFlags determine the order
// computed by the c.CompareTypes function and therefore the order of constituent types in union types.
// Since union type processing often bails out early when a result is known, it is important to order
// TypeFlags in increasing order of potential type complexity. In particular, indexed access and
// conditional types should sort last as those types are potentially recursive and possibly infinite.

const (
	TypeFlagsNone            TypeFlags = 0
	TypeFlagsAny             TypeFlags = 1 << 0
	TypeFlagsUnknown         TypeFlags = 1 << 1
	TypeFlagsUndefined       TypeFlags = 1 << 2
	TypeFlagsNull            TypeFlags = 1 << 3
	TypeFlagsVoid            TypeFlags = 1 << 4
	TypeFlagsString          TypeFlags = 1 << 5
	TypeFlagsNumber          TypeFlags = 1 << 6
	TypeFlagsBigInt          TypeFlags = 1 << 7
	TypeFlagsBoolean         TypeFlags = 1 << 8
	TypeFlagsESSymbol        TypeFlags = 1 << 9 // Type of symbol primitive introduced in ES6
	TypeFlagsStringLiteral   TypeFlags = 1 << 10
	TypeFlagsNumberLiteral   TypeFlags = 1 << 11
	TypeFlagsBigIntLiteral   TypeFlags = 1 << 12
	TypeFlagsBooleanLiteral  TypeFlags = 1 << 13
	TypeFlagsUniqueESSymbol  TypeFlags = 1 << 14 // unique symbol
	TypeFlagsEnumLiteral     TypeFlags = 1 << 15 // Always combined with StringLiteral, NumberLiteral, or Union
	TypeFlagsEnum            TypeFlags = 1 << 16 // Numeric computed enum member value (must be right after EnumLiteral, see getSortOrderFlags)
	TypeFlagsNonPrimitive    TypeFlags = 1 << 17 // intrinsic object type
	TypeFlagsNever           TypeFlags = 1 << 18 // Never type
	TypeFlagsTypeParameter   TypeFlags = 1 << 19 // Type parameter
	TypeFlagsObject          TypeFlags = 1 << 20 // Object type
	TypeFlagsIndex           TypeFlags = 1 << 21 // keyof T
	TypeFlagsTemplateLiteral TypeFlags = 1 << 22 // Template literal type
	TypeFlagsStringMapping   TypeFlags = 1 << 23 // Uppercase/Lowercase type
	TypeFlagsSubstitution    TypeFlags = 1 << 24 // Type parameter substitution
	TypeFlagsIndexedAccess   TypeFlags = 1 << 25 // T[K]
	TypeFlagsConditional     TypeFlags = 1 << 26 // T extends U ? X : Y
	TypeFlagsUnion           TypeFlags = 1 << 27 // Union (T | U)
	TypeFlagsIntersection    TypeFlags = 1 << 28 // Intersection (T & U)
	TypeFlagsReserved1       TypeFlags = 1 << 29 // Used by union/intersection type construction
	TypeFlagsReserved2       TypeFlags = 1 << 30 // Used by union/intersection type construction
	TypeFlagsReserved3       TypeFlags = 1 << 31

	TypeFlagsAnyOrUnknown                  = TypeFlagsAny | TypeFlagsUnknown
	TypeFlagsNullable                      = TypeFlagsUndefined | TypeFlagsNull
	TypeFlagsLiteral                       = TypeFlagsStringLiteral | TypeFlagsNumberLiteral | TypeFlagsBigIntLiteral | TypeFlagsBooleanLiteral
	TypeFlagsUnit                          = TypeFlagsEnum | TypeFlagsLiteral | TypeFlagsUniqueESSymbol | TypeFlagsNullable
	TypeFlagsFreshable                     = TypeFlagsEnum | TypeFlagsLiteral
	TypeFlagsStringOrNumberLiteral         = TypeFlagsStringLiteral | TypeFlagsNumberLiteral
	TypeFlagsStringOrNumberLiteralOrUnique = TypeFlagsStringLiteral | TypeFlagsNumberLiteral | TypeFlagsUniqueESSymbol
	TypeFlagsDefinitelyFalsy               = TypeFlagsStringLiteral | TypeFlagsNumberLiteral | TypeFlagsBigIntLiteral | TypeFlagsBooleanLiteral | TypeFlagsVoid | TypeFlagsUndefined | TypeFlagsNull
	TypeFlagsPossiblyFalsy                 = TypeFlagsDefinitelyFalsy | TypeFlagsString | TypeFlagsNumber | TypeFlagsBigInt | TypeFlagsBoolean
	TypeFlagsIntrinsic                     = TypeFlagsAny | TypeFlagsUnknown | TypeFlagsString | TypeFlagsNumber | TypeFlagsBigInt | TypeFlagsESSymbol | TypeFlagsVoid | TypeFlagsUndefined | TypeFlagsNull | TypeFlagsNever | TypeFlagsNonPrimitive
	TypeFlagsStringLike                    = TypeFlagsString | TypeFlagsStringLiteral | TypeFlagsTemplateLiteral | TypeFlagsStringMapping
	TypeFlagsNumberLike                    = TypeFlagsNumber | TypeFlagsNumberLiteral | TypeFlagsEnum
	TypeFlagsBigIntLike                    = TypeFlagsBigInt | TypeFlagsBigIntLiteral
	TypeFlagsBooleanLike                   = TypeFlagsBoolean | TypeFlagsBooleanLiteral
	TypeFlagsEnumLike                      = TypeFlagsEnum | TypeFlagsEnumLiteral
	TypeFlagsESSymbolLike                  = TypeFlagsESSymbol | TypeFlagsUniqueESSymbol
	TypeFlagsVoidLike                      = TypeFlagsVoid | TypeFlagsUndefined
	TypeFlagsPrimitive                     = TypeFlagsStringLike | TypeFlagsNumberLike | TypeFlagsBigIntLike | TypeFlagsBooleanLike | TypeFlagsEnumLike | TypeFlagsESSymbolLike | TypeFlagsVoidLike | TypeFlagsNull
	TypeFlagsDefinitelyNonNullable         = TypeFlagsStringLike | TypeFlagsNumberLike | TypeFlagsBigIntLike | TypeFlagsBooleanLike | TypeFlagsEnumLike | TypeFlagsESSymbolLike | TypeFlagsObject | TypeFlagsNonPrimitive
	TypeFlagsDisjointDomains               = TypeFlagsNonPrimitive | TypeFlagsStringLike | TypeFlagsNumberLike | TypeFlagsBigIntLike | TypeFlagsBooleanLike | TypeFlagsESSymbolLike | TypeFlagsVoidLike | TypeFlagsNull
	TypeFlagsUnionOrIntersection           = TypeFlagsUnion | TypeFlagsIntersection
	TypeFlagsStructuredType                = TypeFlagsObject | TypeFlagsUnion | TypeFlagsIntersection
	TypeFlagsTypeVariable                  = TypeFlagsTypeParameter | TypeFlagsIndexedAccess
	TypeFlagsInstantiableNonPrimitive      = TypeFlagsTypeVariable | TypeFlagsConditional | TypeFlagsSubstitution
	TypeFlagsInstantiablePrimitive         = TypeFlagsIndex | TypeFlagsTemplateLiteral | TypeFlagsStringMapping
	TypeFlagsInstantiable                  = TypeFlagsInstantiableNonPrimitive | TypeFlagsInstantiablePrimitive
	TypeFlagsStructuredOrInstantiable      = TypeFlagsStructuredType | TypeFlagsInstantiable
	TypeFlagsObjectFlagsType               = TypeFlagsAny | TypeFlagsNullable | TypeFlagsNever | TypeFlagsObject | TypeFlagsUnion | TypeFlagsIntersection
	TypeFlagsSimplifiable                  = TypeFlagsIndexedAccess | TypeFlagsConditional | TypeFlagsIndex
	TypeFlagsSingleton                     = TypeFlagsAny | TypeFlagsUnknown | TypeFlagsString | TypeFlagsNumber | TypeFlagsBoolean | TypeFlagsBigInt | TypeFlagsESSymbol | TypeFlagsVoid | TypeFlagsUndefined | TypeFlagsNull | TypeFlagsNever | TypeFlagsNonPrimitive
	// 'TypeFlagsNarrowable' types are types where narrowing actually narrows.
	// This *should* be every type other than null, undefined, void, and never
	TypeFlagsNarrowable = TypeFlagsAny | TypeFlagsUnknown | TypeFlagsStructuredOrInstantiable | TypeFlagsStringLike | TypeFlagsNumberLike | TypeFlagsBigIntLike | TypeFlagsBooleanLike | TypeFlagsESSymbol | TypeFlagsUniqueESSymbol | TypeFlagsNonPrimitive
	// The following flags are aggregated during union and intersection type construction
	TypeFlagsIncludesMask = TypeFlagsAny | TypeFlagsUnknown | TypeFlagsPrimitive | TypeFlagsNever | TypeFlagsObject | TypeFlagsUnion | TypeFlagsIntersection | TypeFlagsNonPrimitive | TypeFlagsTemplateLiteral | TypeFlagsStringMapping
	// The following flags are used for different purposes during union and intersection type construction
	TypeFlagsIncludesMissingType             = TypeFlagsTypeParameter
	TypeFlagsIncludesNonWideningType         = TypeFlagsIndex
	TypeFlagsIncludesWildcard                = TypeFlagsIndexedAccess
	TypeFlagsIncludesEmptyObject             = TypeFlagsConditional
	TypeFlagsIncludesInstantiable            = TypeFlagsSubstitution
	TypeFlagsIncludesConstrainedTypeVariable = TypeFlagsReserved1
	TypeFlagsIncludesError                   = TypeFlagsReserved2
	TypeFlagsNotPrimitiveUnion               = TypeFlagsAny | TypeFlagsUnknown | TypeFlagsVoid | TypeFlagsNever | TypeFlagsObject | TypeFlagsIntersection | TypeFlagsIncludesInstantiable
)

var typeFlagNames = [...]struct {
	flag TypeFlags
	name string
}{
	{TypeFlagsAny, "Any"},
	{TypeFlagsUnknown, "Unknown"},
	{TypeFlagsUndefined, "Undefined"},
	{TypeFlagsNull, "Null"},
	{TypeFlagsVoid, "Void"},
	{TypeFlagsString, "String"},
	{TypeFlagsNumber, "Number"},
	{TypeFlagsBigInt, "BigInt"},
	{TypeFlagsBoolean, "Boolean"},
	{TypeFlagsESSymbol, "ESSymbol"},
	{TypeFlagsStringLiteral, "StringLiteral"},
	{TypeFlagsNumberLiteral, "NumberLiteral"},
	{TypeFlagsBigIntLiteral, "BigIntLiteral"},
	{TypeFlagsBooleanLiteral, "BooleanLiteral"},
	{TypeFlagsUniqueESSymbol, "UniqueESSymbol"},
	{TypeFlagsEnumLiteral, "EnumLiteral"},
	{TypeFlagsEnum, "Enum"},
	{TypeFlagsNonPrimitive, "NonPrimitive"},
	{TypeFlagsNever, "Never"},
	{TypeFlagsTypeParameter, "TypeParameter"},
	{TypeFlagsObject, "Object"},
	{TypeFlagsIndex, "Index"},
	{TypeFlagsTemplateLiteral, "TemplateLiteral"},
	{TypeFlagsStringMapping, "StringMapping"},
	{TypeFlagsSubstitution, "Substitution"},
	{TypeFlagsIndexedAccess, "IndexedAccess"},
	{TypeFlagsConditional, "Conditional"},
	{TypeFlagsUnion, "Union"},
	{TypeFlagsIntersection, "Intersection"},
}

// FormatTypeFlags returns the individual flag names as a slice of strings.
func FormatTypeFlags(flags TypeFlags) []string {
	result := make([]string, 0, bits.OnesCount32(uint32(flags)))
	for _, fn := range typeFlagNames {
		if flags&fn.flag != 0 {
			result = append(result, fn.name)
		}
	}
	if len(result) == 0 {
		result = append(result, "None")
	}
	return result
}

// String returns a pipe-separated string of flag names.
func (f TypeFlags) String() string {
	return strings.Join(FormatTypeFlags(f), "|")
}

func (v VarianceFlags) String() string {
	variance := v & VarianceFlagsVarianceMask
	var result string
	switch variance {
	case VarianceFlagsInvariant:
		result = "in out"
	case VarianceFlagsBivariant:
		result = "[bivariant]"
	case VarianceFlagsContravariant:
		result = "in"
	case VarianceFlagsCovariant:
		result = "out"
	case VarianceFlagsIndependent:
		result = "[independent]"
	default:
		result = ""
	}
	if v&VarianceFlagsUnmeasurable != 0 {
		result += " (unmeasurable)"
	} else if v&VarianceFlagsUnreliable != 0 {
		result += " (unreliable)"
	}
	return result
}

type ObjectFlags uint32

// Types included in TypeFlags.ObjectFlagsType have an objectFlags property. Some ObjectFlags
// are specific to certain types and reuse the same bit position. Those ObjectFlags require a check
// for a certain TypeFlags value to determine their meaning.
const (
	ObjectFlagsNone                                       ObjectFlags = 0
	ObjectFlagsClass                                      ObjectFlags = 1 << 0  // Class
	ObjectFlagsInterface                                  ObjectFlags = 1 << 1  // Interface
	ObjectFlagsReference                                  ObjectFlags = 1 << 2  // Generic type reference
	ObjectFlagsTuple                                      ObjectFlags = 1 << 3  // Synthesized generic tuple type
	ObjectFlagsAnonymous                                  ObjectFlags = 1 << 4  // Anonymous
	ObjectFlagsMapped                                     ObjectFlags = 1 << 5  // Mapped
	ObjectFlagsInstantiated                               ObjectFlags = 1 << 6  // Instantiated anonymous or mapped type
	ObjectFlagsObjectLiteral                              ObjectFlags = 1 << 7  // Originates in an object literal
	ObjectFlagsEvolvingArray                              ObjectFlags = 1 << 8  // Evolving array type
	ObjectFlagsObjectLiteralPatternWithComputedProperties ObjectFlags = 1 << 9  // Object literal pattern with computed properties
	ObjectFlagsReverseMapped                              ObjectFlags = 1 << 10 // Object contains a property from a reverse-mapped type
	ObjectFlagsJsxAttributes                              ObjectFlags = 1 << 11 // Jsx attributes type
	ObjectFlagsJSLiteral                                  ObjectFlags = 1 << 12 // Object type declared in JS - disables errors on read/write of nonexisting members
	ObjectFlagsFreshLiteral                               ObjectFlags = 1 << 13 // Fresh object literal
	ObjectFlagsArrayLiteral                               ObjectFlags = 1 << 14 // Originates in an array literal
	ObjectFlagsPrimitiveUnion                             ObjectFlags = 1 << 15 // Union of only primitive types
	ObjectFlagsContainsWideningType                       ObjectFlags = 1 << 16 // Type is or contains undefined or null widening type
	ObjectFlagsContainsObjectOrArrayLiteral               ObjectFlags = 1 << 17 // Type is or contains object literal type
	ObjectFlagsNonInferrableType                          ObjectFlags = 1 << 18 // Type is or contains anyFunctionType or silentNeverType
	ObjectFlagsCouldContainTypeVariablesComputed          ObjectFlags = 1 << 19 // CouldContainTypeVariables flag has been computed
	ObjectFlagsCouldContainTypeVariables                  ObjectFlags = 1 << 20 // Type could contain a type variable
	ObjectFlagsMembersResolved                            ObjectFlags = 1 << 21 // Members have been resolved

	ObjectFlagsClassOrInterface   = ObjectFlagsClass | ObjectFlagsInterface
	ObjectFlagsRequiresWidening   = ObjectFlagsContainsWideningType | ObjectFlagsContainsObjectOrArrayLiteral
	ObjectFlagsPropagatingFlags   = ObjectFlagsContainsWideningType | ObjectFlagsContainsObjectOrArrayLiteral | ObjectFlagsNonInferrableType
	ObjectFlagsInstantiatedMapped = ObjectFlagsMapped | ObjectFlagsInstantiated
	// Object flags that uniquely identify the kind of ObjectType
	ObjectFlagsObjectTypeKindMask = ObjectFlagsClassOrInterface | ObjectFlagsReference | ObjectFlagsTuple | ObjectFlagsAnonymous | ObjectFlagsMapped | ObjectFlagsReverseMapped | ObjectFlagsEvolvingArray | ObjectFlagsInstantiationExpressionType | ObjectFlagsSingleSignatureType
	// Flags that require TypeFlags.Object
	ObjectFlagsContainsSpread              = 1 << 22 // Object literal contains spread operation
	ObjectFlagsObjectRestType              = 1 << 23 // Originates in object rest declaration
	ObjectFlagsInstantiationExpressionType = 1 << 24 // Originates in instantiation expression
	ObjectFlagsSingleSignatureType         = 1 << 25 // A single signature type extracted from a potentially broader type
	ObjectFlagsIsClassInstanceClone        = 1 << 26 // Type is a clone of a class instance type
	// Flags that require TypeFlags.Object and ObjectFlags.Reference
	ObjectFlagsIdenticalBaseTypeCalculated = 1 << 27 // has had `getSingleBaseForNonAugmentingSubtype` invoked on it already
	ObjectFlagsIdenticalBaseTypeExists     = 1 << 28 // has a defined cachedEquivalentBaseType member
	ObjectFlagsFromTypeNode                = 1 << 29 // Originates in resolution of AST type node
	// Flags that require TypeFlags.UnionOrIntersection or TypeFlags.Substitution
	ObjectFlagsIsGenericTypeComputed = 1 << 22 // IsGenericObjectType flag has been computed
	ObjectFlagsIsGenericObjectType   = 1 << 23 // Union or intersection contains generic object type
	ObjectFlagsIsGenericIndexType    = 1 << 24 // Union or intersection contains generic index type
	ObjectFlagsIsGenericType         = ObjectFlagsIsGenericObjectType | ObjectFlagsIsGenericIndexType
	// Flags that require TypeFlags.Union
	ObjectFlagsContainsIntersections      = 1 << 25 // Union contains intersections
	ObjectFlagsIsUnknownLikeUnionComputed = 1 << 26 // IsUnknownLikeUnion flag has been computed
	ObjectFlagsIsUnknownLikeUnion         = 1 << 27 // Union of null, undefined, and empty object type
	ObjectFlagsIsUniformEnumComputed      = 1 << 28 // IsUniformEnum flag has been computed
	ObjectFlagsIsUniformEnum              = 1 << 29 // Union contains uniform literal types
	// Flags that require TypeFlags.Intersection
	ObjectFlagsIsNeverIntersectionComputed = 1 << 25 // IsNeverLike flag has been computed
	ObjectFlagsIsNeverIntersection         = 1 << 26 // Intersection reduces to never
	ObjectFlagsIsConstrainedTypeVariable   = 1 << 27 // T & C, where T's constraint and C are primitives, object, or {}
)

// TypeAlias

type TypeAlias struct {
	symbol        ast.SymbolId
	typeArguments []TypeId
}

func (a *TypeAlias) Symbol() ast.SymbolId {
	if a == nil {
		return 0
	}
	return a.symbol
}

func (a *TypeAlias) TypeArguments() []TypeId {
	if a == nil {
		return nil
	}
	return a.typeArguments
}

// Type

// The concrete kind names the layout of a type's payload. It is kept in the high bits of the type's id. All concrete
// types are views of the same 64-byte Type record, so a kind replaces the self-referential interface. The ordering
// groups types by their embedded bases for the base casts below.
type typeKind uint8

const (
	typeKindIntrinsic typeKind = iota + 1
	typeKindLiteral
	typeKindUniqueESSymbol
	typeKindTypeParameter
	typeKindIndex
	typeKindIndexedAccess
	typeKindTemplateLiteral
	typeKindStringMapping
	typeKindSubstitution
	typeKindConditional
	typeKindObject
	typeKindTypeReference
	typeKindInterface
	typeKindTuple
	typeKindInstantiationExpression
	typeKindMapped
	typeKindReverseMapped
	typeKindEvolvingArray
	typeKindUnion
	typeKindIntersection
)

const (
	typeKindShift = 27
	typeIdMask    = 1<<typeKindShift - 1
)

// A Type is a 56-byte record in a checker-owned slab, with no Go pointers. The header holds what every type has;
// resolved names the members, properties, signatures and index infos of object, union and intersection types once
// they are resolved; extra names a record of fields few types have; and data is a 32-byte payload whose meaning
// depends on the kind. Kinds with more state than fits keep it in a side record whose index the payload holds.
type Type struct {
	flags       TypeFlags
	objectFlags ObjectFlags
	id          TypeId // The type's id, with its typeKind in the high bits
	symbol      ast.SymbolId
	resolved    uint32 // Resolved members: an index in the checker's resolved records plus one, 0 for none
	extra       uint32 // Rarely set fields: an index in the checker's type extras plus one, 0 for none
	data        typeData
}

// typeData is a type's per-kind payload: eight 32-bit words holding type ids, mapper ids, list handles and indices of
// side records, laid out per kind as described at each kind. It holds no Go pointers, so the type slabs are not
// scanned by the garbage collector.
type typeData [8]uint32

// Enforce the layout required by the pointer conversions, on every architecture: every concrete type is a Type at
// offset zero with no further fields.
var (
	_ [0 - unsafe.Offsetof(IntrinsicType{}.Type)]byte
	_ [unsafe.Sizeof(IntrinsicType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(IntrinsicType{})]byte
	_ [0 - unsafe.Offsetof(LiteralType{}.Type)]byte
	_ [unsafe.Sizeof(LiteralType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(LiteralType{})]byte
	_ [0 - unsafe.Offsetof(UniqueESSymbolType{}.Type)]byte
	_ [unsafe.Sizeof(UniqueESSymbolType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(UniqueESSymbolType{})]byte
	_ [0 - unsafe.Offsetof(TypeParameter{}.Type)]byte
	_ [unsafe.Sizeof(TypeParameter{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(TypeParameter{})]byte
	_ [0 - unsafe.Offsetof(IndexType{}.Type)]byte
	_ [unsafe.Sizeof(IndexType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(IndexType{})]byte
	_ [0 - unsafe.Offsetof(IndexedAccessType{}.Type)]byte
	_ [unsafe.Sizeof(IndexedAccessType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(IndexedAccessType{})]byte
	_ [0 - unsafe.Offsetof(TemplateLiteralType{}.Type)]byte
	_ [unsafe.Sizeof(TemplateLiteralType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(TemplateLiteralType{})]byte
	_ [0 - unsafe.Offsetof(StringMappingType{}.Type)]byte
	_ [unsafe.Sizeof(StringMappingType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(StringMappingType{})]byte
	_ [0 - unsafe.Offsetof(SubstitutionType{}.Type)]byte
	_ [unsafe.Sizeof(SubstitutionType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(SubstitutionType{})]byte
	_ [0 - unsafe.Offsetof(ConditionalType{}.Type)]byte
	_ [unsafe.Sizeof(ConditionalType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(ConditionalType{})]byte
	_ [0 - unsafe.Offsetof(ObjectType{}.Type)]byte
	_ [unsafe.Sizeof(ObjectType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(ObjectType{})]byte
	_ [0 - unsafe.Offsetof(TypeReference{}.Type)]byte
	_ [unsafe.Sizeof(TypeReference{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(TypeReference{})]byte
	_ [0 - unsafe.Offsetof(InterfaceType{}.Type)]byte
	_ [unsafe.Sizeof(InterfaceType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(InterfaceType{})]byte
	_ [0 - unsafe.Offsetof(TupleType{}.Type)]byte
	_ [unsafe.Sizeof(TupleType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(TupleType{})]byte
	_ [0 - unsafe.Offsetof(InstantiationExpressionType{}.Type)]byte
	_ [unsafe.Sizeof(InstantiationExpressionType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(InstantiationExpressionType{})]byte
	_ [0 - unsafe.Offsetof(MappedType{}.Type)]byte
	_ [unsafe.Sizeof(MappedType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(MappedType{})]byte
	_ [0 - unsafe.Offsetof(ReverseMappedType{}.Type)]byte
	_ [unsafe.Sizeof(ReverseMappedType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(ReverseMappedType{})]byte
	_ [0 - unsafe.Offsetof(EvolvingArrayType{}.Type)]byte
	_ [unsafe.Sizeof(EvolvingArrayType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(EvolvingArrayType{})]byte
	_ [0 - unsafe.Offsetof(UnionType{}.Type)]byte
	_ [unsafe.Sizeof(UnionType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(UnionType{})]byte
	_ [0 - unsafe.Offsetof(IntersectionType{}.Type)]byte
	_ [unsafe.Sizeof(IntersectionType{}) - unsafe.Sizeof(Type{})]byte
	_ [unsafe.Sizeof(Type{}) - unsafe.Sizeof(IntersectionType{})]byte
)

// typeExtra holds the fields of a type that few types set.
type typeExtra struct {
	alias                  *TypeAlias
	resolvedBaseConstraint TypeId
	instantiations         map[CacheHashKey]TypeId // Map of type instantiations
	mapper                 TypeMapperId            // Type mapper of a deferred type reference
	node                   ast.NodeRef             // TypeReferenceNode | ArrayTypeNode | TupleTypeNode of a deferred type reference
	resolvedDefaultType    TypeId                  // Default of a type parameter
	distributedType        TypeId                  // Distributed form of a type parameter
}

// structuredMembers holds the resolved members of an object, union or intersection type.
//
// The record holds no pointers: the member table is an index (plus one) into the checker's member tables, the
// properties a list in its symbol list store, and the rarer fields an index (plus one) into its extra records.
type structuredMembers struct {
	members uint32
	propPos uint32
	propLen uint32
	extra   uint32
}

type structuredTypeExtra struct {
	signatures         []*Signature // Signatures (call + construct)
	callSignatureCount int          // Count of call signatures
	indexInfos         []*IndexInfo

	objectTypeWithoutAbstractConstructSignatures TypeId
}

// typeSlab holds a checker's types in fixed chunks behind a directory, so a TypeId (a type's position, from 1)
// finds its record with one load from the directory and one from the chunk. Chunks never move, so a *Type stays
// valid for the checker's life.
type typeSlab struct {
	dir []*typeChunk
	n   uint32
}

const (
	typeChunkShift = 12
	typeChunkSize  = 1 << typeChunkShift
	typeChunkMask  = typeChunkSize - 1
)

type typeChunk [typeChunkSize]Type

func (s *typeSlab) add(kind typeKind) *Type {
	s.n++
	id := s.n
	if id > typeIdMask {
		panic("Too many types")
	}
	k := int(id >> typeChunkShift)
	if k == len(s.dir) {
		s.dir = append(s.dir, new(typeChunk))
	}
	t := &s.dir[k][id&typeChunkMask]
	t.id = TypeId(kind)<<typeKindShift | TypeId(id)
	return t
}

// typ returns the type with the given id, or nil for 0.
func (c *Checker) typ(id TypeId) *Type {
	if id == 0 {
		return nil
	}
	return &c.types.dir[id>>typeChunkShift][id&typeChunkMask]
}

// ref returns the id of t, or 0 for nil.
func (t *Type) ref() TypeId {
	if t == nil {
		return 0
	}
	return t.id & typeIdMask
}

func castType[T any](t *Type, kind typeKind) *T {
	if t.kind() != kind {
		panic("Invalid type cast")
	}
	// SAFETY: Every concrete type is a struct that embeds Type and adds no fields, and every caller pairs T with its kind.
	return (*T)(unsafe.Pointer(t))
}

func (t *Type) kind() typeKind {
	return typeKind(t.id >> typeKindShift)
}

func (t *Type) Id() TypeId {
	return t.id & typeIdMask
}

func (t *Type) Flags() TypeFlags {
	return t.flags
}

func (t *Type) ObjectFlags() ObjectFlags {
	return t.objectFlags
}

// ext returns the type's rarely set fields, or nil if none is set.
func (t *Type) ext(c *Checker) *typeExtra {
	return pageAt(&c.typeExtras, t.extra)
}

func (t *Type) ensureExtra(c *Checker) *typeExtra {
	if t.extra == 0 {
		var e *typeExtra
		t.extra, e = pageAdd(&c.typeExtras)
		return e
	}
	return c.typeExtras.At(t.extra - 1)
}

// res returns the type's resolved members, or nil if none is set.
func (t *Type) res(c *Checker) *structuredMembers {
	return pageAt(&c.typeResolved, t.resolved)
}

func (t *Type) ensureResolved(c *Checker) *structuredMembers {
	if t.resolved == 0 {
		var r *structuredMembers
		t.resolved, r = pageAdd(&c.typeResolved)
		return r
	}
	return c.typeResolved.At(t.resolved - 1)
}

// pageAt returns the record at a stored index (position plus one), or nil for 0.
func pageAt[T any](p *core.Pages[T], i uint32) *T {
	if i == 0 {
		return nil
	}
	return p.At(i - 1)
}

// pageAdd appends a zero record and returns its stored index (position plus one) and address.
func pageAdd[T any](p *core.Pages[T]) (uint32, *T) {
	i, r := p.Add()
	return i + 1, r
}

// symbolListOf copies a list of symbols into the checker's symbol list store and returns its handle. A nil list stays
// nil and an empty one stays empty.
func (c *Checker) symbolListOf(s []ast.SymbolId) (uint32, uint32) {
	if s == nil {
		return 0, 0
	}
	pos, room := c.symbolLists.alloc(len(s))
	copy(room, s)
	return pos, uint32(len(s))
}

// typeListOf copies a list of types into the checker's list store and returns its handle. A nil list stays nil and an
// empty one stays empty.
func (c *Checker) typeListOf(s []TypeId) (uint32, uint32) {
	if s == nil {
		return 0, 0
	}
	pos, list := c.typeLists.alloc(len(s))
	copy(list, s)
	return pos, uint32(len(s))
}

// Casts for concrete struct types

func (t *Type) AsIntrinsicType() *IntrinsicType { return castType[IntrinsicType](t, typeKindIntrinsic) }
func (t *Type) AsLiteralType() *LiteralType     { return castType[LiteralType](t, typeKindLiteral) }
func (t *Type) AsUniqueESSymbolType() *UniqueESSymbolType {
	return castType[UniqueESSymbolType](t, typeKindUniqueESSymbol)
}
func (t *Type) AsTupleType() *TupleType { return castType[TupleType](t, typeKindTuple) }
func (t *Type) AsInstantiationExpressionType() *InstantiationExpressionType {
	return castType[InstantiationExpressionType](t, typeKindInstantiationExpression)
}
func (t *Type) AsMappedType() *MappedType { return castType[MappedType](t, typeKindMapped) }
func (t *Type) AsReverseMappedType() *ReverseMappedType {
	return castType[ReverseMappedType](t, typeKindReverseMapped)
}

func (t *Type) AsEvolvingArrayType() *EvolvingArrayType {
	return castType[EvolvingArrayType](t, typeKindEvolvingArray)
}

func (t *Type) AsTypeParameter() *TypeParameter {
	return castType[TypeParameter](t, typeKindTypeParameter)
}
func (t *Type) AsUnionType() *UnionType { return castType[UnionType](t, typeKindUnion) }
func (t *Type) AsIntersectionType() *IntersectionType {
	return castType[IntersectionType](t, typeKindIntersection)
}
func (t *Type) AsIndexType() *IndexType { return castType[IndexType](t, typeKindIndex) }
func (t *Type) AsIndexedAccessType() *IndexedAccessType {
	return castType[IndexedAccessType](t, typeKindIndexedAccess)
}

func (t *Type) AsTemplateLiteralType() *TemplateLiteralType {
	return castType[TemplateLiteralType](t, typeKindTemplateLiteral)
}

func (t *Type) AsStringMappingType() *StringMappingType {
	return castType[StringMappingType](t, typeKindStringMapping)
}

func (t *Type) AsSubstitutionType() *SubstitutionType {
	return castType[SubstitutionType](t, typeKindSubstitution)
}

func (t *Type) AsConditionalType() *ConditionalType {
	return castType[ConditionalType](t, typeKindConditional)
}

// Casts for embedded struct types

func (t *Type) AsConstrainedType() *ConstrainedType {
	if kind := t.kind(); kind >= typeKindTypeParameter && kind <= typeKindIntersection {
		// SAFETY: Types of these kinds are structs that embed ConstrainedType at offset zero.
		return (*ConstrainedType)(unsafe.Pointer(t))
	}
	return nil
}

func (t *Type) AsStructuredType() *StructuredType {
	if kind := t.kind(); kind >= typeKindObject && kind <= typeKindIntersection {
		// SAFETY: Types of these kinds are structs that embed StructuredType at offset zero.
		return (*StructuredType)(unsafe.Pointer(t))
	}
	return nil
}

func (t *Type) AsObjectType() *ObjectType {
	if kind := t.kind(); kind >= typeKindObject && kind <= typeKindEvolvingArray {
		// SAFETY: Types of these kinds are structs that embed ObjectType at offset zero.
		return (*ObjectType)(unsafe.Pointer(t))
	}
	return nil
}

func (t *Type) AsTypeReference() *TypeReference {
	if kind := t.kind(); kind >= typeKindTypeReference && kind <= typeKindTuple {
		// SAFETY: Types of these kinds are structs that embed TypeReference at offset zero.
		return (*TypeReference)(unsafe.Pointer(t))
	}
	return nil
}

func (t *Type) AsInterfaceType() *InterfaceType {
	if kind := t.kind(); kind >= typeKindInterface && kind <= typeKindTuple {
		// SAFETY: Types of these kinds are structs that embed InterfaceType at offset zero.
		return (*InterfaceType)(unsafe.Pointer(t))
	}
	return nil
}

func (t *Type) AsUnionOrIntersectionType() *UnionOrIntersectionType {
	if kind := t.kind(); kind >= typeKindUnion && kind <= typeKindIntersection {
		// SAFETY: Types of these kinds are structs that embed UnionOrIntersectionType at offset zero.
		return (*UnionOrIntersectionType)(unsafe.Pointer(t))
	}
	return nil
}

func (t *Type) Distributed(c *Checker) []TypeId {
	switch {
	case t.flags&TypeFlagsUnion != 0:
		return t.AsUnionType().types(c)
	case t.flags&TypeFlagsNever != 0:
		return nil
	}
	return []TypeId{t.ref()}
}

// Common accessors

func (t *Type) Target(c *Checker) *Type {
	switch {
	case t.flags&TypeFlagsObject != 0:
		return c.typ(t.AsObjectType().targetId())
	case t.flags&TypeFlagsTypeParameter != 0:
		return c.typ(t.AsTypeParameter().targetId())
	case t.flags&TypeFlagsIndex != 0:
		return c.typ(t.AsIndexType().targetId())
	case t.flags&TypeFlagsStringMapping != 0:
		return c.typ(t.AsStringMappingType().targetId())
	}
	panic("Unhandled case in Type.Target")
}

func (t *Type) Mapper(c *Checker) *TypeMapper {
	switch {
	case t.flags&TypeFlagsObject != 0:
		return t.AsObjectType().mapper(c)
	case t.flags&TypeFlagsTypeParameter != 0:
		return t.AsTypeParameter().mapper(c)
	case t.flags&TypeFlagsConditional != 0:
		return c.mapperOf(t.AsConditionalType().cond(c).mapper)
	}
	panic("Unhandled case in Type.Mapper")
}

func (t *Type) Types(c *Checker) []TypeId {
	switch {
	case t.flags&TypeFlagsUnionOrIntersection != 0:
		return t.AsUnionOrIntersectionType().types(c)
	case t.flags&TypeFlagsTemplateLiteral != 0:
		return t.AsTemplateLiteralType().types(c)
	}
	panic("Unhandled case in Type.Types")
}

func (t *Type) TargetInterfaceType(c *Checker) *InterfaceType {
	return c.typ(t.AsTypeReference().targetId()).AsInterfaceType()
}

func (t *Type) TargetTupleType(c *Checker) *TupleType {
	return c.typ(t.AsTypeReference().targetId()).AsTupleType()
}

// SymbolOfType returns the symbol of a type, or the zero Symbol if it has none.
func (c *Checker) symbolOfTypeId(t *Type) ast.SymbolId {
	return c.symbolOfType(t)
}

func (c *Checker) symbolOfType(t *Type) ast.SymbolId {
	if t.symbol == 0 {
		return 0
	}
	return c.symbolId(c.symbolIds.symbol(t.symbol))
}

func (t *Type) setSymbol(c *Checker, symbol ast.SymbolId) {
	t.symbol = symbol
}

func (t *Type) Alias(c *Checker) *TypeAlias {
	if t.extra == 0 {
		return nil
	}
	return t.ext(c).alias
}

func (t *Type) setAlias(c *Checker, alias *TypeAlias) {
	if alias == nil && t.extra == 0 {
		return
	}
	t.ensureExtra(c).alias = alias
}

func (t *Type) IsUnion() bool {
	return t.flags&TypeFlagsUnion != 0
}

func (t *Type) IsString() bool {
	return t.flags&TypeFlagsString != 0
}

func (t *Type) IsIntersection() bool {
	return t.flags&TypeFlagsIntersection != 0
}

func (t *Type) IsStringLiteral() bool {
	return t.flags&TypeFlagsStringLiteral != 0
}

func (t *Type) IsNumberLiteral() bool {
	return t.flags&TypeFlagsNumberLiteral != 0
}

func (t *Type) IsBigIntLiteral() bool {
	return t.flags&TypeFlagsBigIntLiteral != 0
}

func (t *Type) IsEnumLiteral() bool {
	return t.flags&TypeFlagsEnumLiteral != 0
}

func (t *Type) IsBooleanLike() bool {
	return t.flags&TypeFlagsBooleanLike != 0
}

func (t *Type) IsStringLike() bool {
	return t.flags&TypeFlagsStringLike != 0
}

func (t *Type) IsClass() bool {
	return t.objectFlags&ObjectFlagsClass != 0
}

func (t *Type) IsTypeParameter() bool {
	return t.flags&TypeFlagsTypeParameter != 0
}

func (t *Type) IsIndex() bool {
	return t.flags&TypeFlagsIndex != 0
}

func (t *Type) IsTupleType(c *Checker) bool {
	return c.isTupleType(t)
}

// TypeBase

type TypeBase struct {
	Type
}

func (t *TypeBase) AsType() *Type                                       { return &t.Type }
func (t *TypeBase) AsConstrainedType() *ConstrainedType                 { return nil }
func (t *TypeBase) AsStructuredType() *StructuredType                   { return nil }
func (t *TypeBase) AsObjectType() *ObjectType                           { return nil }
func (t *TypeBase) AsTypeReference() *TypeReference                     { return nil }
func (t *TypeBase) AsInterfaceType() *InterfaceType                     { return nil }
func (t *TypeBase) AsUnionOrIntersectionType() *UnionOrIntersectionType { return nil }

// IntrinsicType: in word 0 the index of the name in the checker's intrinsic names.

type IntrinsicType struct {
	TypeBase
}

func (t *IntrinsicType) IntrinsicName(c *Checker) string {
	return c.intrinsicNames[t.data[0]]
}

func (t *IntrinsicType) setIntrinsicName(c *Checker, name string) {
	t.data[0] = uint32(len(c.intrinsicNames))
	c.intrinsicNames = append(c.intrinsicNames, name)
}

// LiteralType: in word 0 the index of the value (string | jsnum.Number | bool | PseudoBigInt) in the checker's literal
// values plus one, or 0 for the nil value of a computed enum; in word 1 the regular version of a fresh type or the
// fresh version of a regular type; bit 0 of word 2 marks a fresh type.

type LiteralType struct {
	TypeBase
}

const literalTypeFresh = 1

func (t *LiteralType) Value(c *Checker) any {
	if t.data[0] == 0 {
		return nil
	}
	return *c.literalValues.At(t.data[0] - 1)
}

func (t *LiteralType) setValue(c *Checker, value any) {
	if value == nil {
		t.data[0] = 0
		return
	}
	var v *any
	t.data[0], v = pageAdd(&c.literalValues)
	*v = value
}

// FreshType returns the fresh version of the type.
func (t *LiteralType) FreshType(c *Checker) *Type {
	if t.data[2]&literalTypeFresh != 0 {
		return &t.Type
	}
	return c.typ(TypeId(t.data[1]))
}

// RegularType returns the regular version of the type.
func (t *LiteralType) RegularType(c *Checker) *Type {
	if t.data[2]&literalTypeFresh != 0 {
		return c.typ(TypeId(t.data[1]))
	}
	return &t.Type
}

func (t *LiteralType) setFreshType(fresh *Type) {
	if fresh == &t.Type {
		t.data[2] |= literalTypeFresh
	} else {
		t.data[1] = uint32(fresh.ref())
	}
}

func (t *LiteralType) setRegularType(regular *Type) {
	if regular == &t.Type {
		t.data[2] &^= literalTypeFresh
	} else {
		t.data[2] |= literalTypeFresh
		t.data[1] = uint32(regular.ref())
	}
}

// UniqueESSymbolType: the name in words 0 and 1.

type UniqueESSymbolType struct {
	TypeBase
}

func (t *UniqueESSymbolType) name() ast.SymbolName {
	return ast.SymbolName(uint64(t.data[1])<<32 | uint64(t.data[0]))
}

func (t *UniqueESSymbolType) setName(name ast.SymbolName) {
	t.data[0] = uint32(name)
	t.data[1] = uint32(name >> 32)
}

// ConstrainedType (type with computed base constraint)

type ConstrainedType struct {
	TypeBase
}

func (t *ConstrainedType) AsConstrainedType() *ConstrainedType { return t }

func (t *ConstrainedType) resolvedBaseConstraint(c *Checker) *Type {
	if t.extra == 0 {
		return nil
	}
	return c.typ(t.ext(c).resolvedBaseConstraint)
}

func (t *ConstrainedType) setResolvedBaseConstraint(c *Checker, constraint *Type) {
	if constraint == nil && t.extra == 0 {
		return
	}
	t.ensureExtra(c).resolvedBaseConstraint = constraint.ref()
}

// StructuredType (base of all types with members)

type StructuredType struct {
	ConstrainedType
}

func (t *StructuredType) AsStructuredType() *StructuredType { return t }

func (t *StructuredType) members(c *Checker) ast.SymbolTable {
	if t.resolved == 0 {
		return nil
	}
	if i := t.res(c).members; i != 0 {
		return *c.memberTables.At(i - 1)
	}
	return nil
}

func (t *StructuredType) setMembers(c *Checker, members ast.SymbolTable) {
	if members == nil {
		if t.resolved != 0 {
			t.res(c).members = 0
		}
		return
	}
	r := t.ensureResolved(c)
	if r.members != 0 {
		*c.memberTables.At(r.members - 1) = members
		return
	}
	i, m := pageAdd(&c.memberTables)
	*m = members
	r.members = i
}

func (t *StructuredType) properties(c *Checker) []ast.SymbolId {
	if t.resolved == 0 {
		return nil
	}
	r := t.res(c)
	return c.symbolLists.slice(r.propPos, r.propLen)
}

// setProperties stores a copy of properties.
func (t *StructuredType) setProperties(c *Checker, properties []ast.SymbolId) {
	if properties == nil && t.resolved == 0 {
		return
	}
	r := t.ensureResolved(c)
	r.propPos, r.propLen = c.symbolListOf(properties)
}

// shareProperties makes t's properties the same list as source's.
func (t *StructuredType) shareProperties(c *Checker, source *StructuredType) {
	if source.resolved == 0 {
		t.setProperties(c, nil)
		return
	}
	pos, n := source.res(c).propPos, source.res(c).propLen
	if pos == 0 && t.resolved == 0 {
		return
	}
	r := t.ensureResolved(c)
	r.propPos, r.propLen = pos, n
}

func (t *StructuredType) resolvedExtra(c *Checker) *structuredTypeExtra {
	if t.resolved == 0 {
		return nil
	}
	return pageAt(&c.structuredTypeExtras, t.res(c).extra)
}

func (t *StructuredType) ensureResolvedExtra(c *Checker) *structuredTypeExtra {
	resolved := t.ensureResolved(c)
	if resolved.extra == 0 {
		i, extra := pageAdd(&c.structuredTypeExtras)
		resolved.extra = i
		return extra
	}
	return pageAt(&c.structuredTypeExtras, resolved.extra)
}

func (t *StructuredType) CallSignatures(c *Checker) []*Signature {
	extra := t.resolvedExtra(c)
	if extra == nil {
		return nil
	}
	return slices.Clip(extra.signatures[:extra.callSignatureCount])
}

func (t *StructuredType) ConstructSignatures(c *Checker) []*Signature {
	extra := t.resolvedExtra(c)
	if extra == nil {
		return nil
	}
	return slices.Clip(extra.signatures[extra.callSignatureCount:])
}

func (t *StructuredType) getMember(c *Checker, name ast.SymbolName) ast.SymbolId {
	if members := t.members(c); members != nil {
		return members.GetId(c.symbols, name)
	}
	lazy := t.getLazyState(c)
	if lazy == nil {
		return 0
	}
	index := lazy.layout.members.Find(name)
	if index < 0 {
		return 0
	}
	return t.getPropertyAt(c, index)
}

func (t *StructuredType) getMembers(c *Checker) ast.SymbolTable {
	if t.members(c) == nil {
		if properties := t.getProperties(c); len(properties) != 0 {
			members := ast.NewSymbolTableWithCapacity(len(properties))
			for _, member := range properties {
				members.SetId(c.symbols, c.symbolName(member), member)
			}
			t.setMembers(c, members)
		}
	}
	return t.members(c)
}

func (t *StructuredType) getPropertyAt(c *Checker, index int) ast.SymbolId {
	lazy := t.getLazyState(c)
	if lazy == nil {
		return t.properties(c)[index]
	}
	if t.properties(c) == nil {
		t.setProperties(c, make([]ast.SymbolId, lazy.layout.members.Len()))
	}
	properties := t.properties(c)
	member := properties[index]
	if member == 0 {
		symbol := lazy.layout.members.IdAt(c.symbols, index)
		baseIndex := lazy.layout.baseIndices[index]
		if baseIndex < 0 {
			member = c.instantiateSymbol(symbol, c.mapperOf(lazy.mapper))
		} else {
			member = c.getPropertyOfType(c.typ(lazy.baseTypes[baseIndex]), c.symbolName(symbol))
		}
		properties[index] = member
	}
	return member
}

func (t *StructuredType) getProperties(c *Checker) []ast.SymbolId {
	if lazy := t.getLazyState(c); lazy != nil && !lazy.propertiesResolved {
		for i := range lazy.layout.members.Len() {
			t.getPropertyAt(c, i)
		}
		lazy.propertiesResolved = true
	}
	return t.properties(c)
}

func (t *StructuredType) hasProperties(c *Checker) bool {
	if len(t.properties(c)) != 0 {
		return true
	}
	lazy := t.getLazyState(c)
	if lazy == nil {
		return false
	}
	return lazy.layout.members.Len() != 0
}

func (t *StructuredType) getLazyState(c *Checker) *LazyState {
	if t.objectFlags&(ObjectFlagsClassOrInterface|ObjectFlagsReference) != 0 {
		return t.AsType().AsTypeReference().lazy(c)
	}
	return nil
}

func (t *StructuredType) getSignatures(c *Checker) []*Signature {
	extra := t.resolvedExtra(c)
	if extra == nil {
		return nil
	}
	return extra.signatures
}

func (t *StructuredType) getIndexInfos(c *Checker) []*IndexInfo {
	extra := t.resolvedExtra(c)
	if extra == nil {
		return nil
	}
	return extra.indexInfos
}

func (t *StructuredType) propertiesId(c *Checker) []ast.SymbolId {
	return t.getProperties(c)
}

// Except for tuple type references and reverse mapped types, all object types have an associated symbol.
// Possible object type instances are listed in the following.

// InterfaceType:
// ObjectFlagsClass: Originating non-generic class type
// ObjectFlagsClass|ObjectFlagsReference: Originating generic class type
// ObjectFlagsInterface: Originating non-generic interface type
// ObjectFlagsInterface|ObjectFlagsReference: Originating generic interface type

// TupleType:
// ObjectFlagsReference|ObjectFlagsTuple: Originating generic tuple type (synthesized)

// TypeReference
// ObjectFlagsReference: Instantiated generic class, interface, or tuple type

// ObjectType:
// ObjectFlagsAnonymous: Originating anonymous object type
// ObjectFlagsAnonymous|ObjectFlagsInstantiated: Instantiated anonymous object type

// MappedType:
// ObjectFlagsMapped: Originating mapped type
// ObjectFlagsMapped|ObjectFlagsInstantiated: Instantiated mapped type

// InstantiationExpressionType:
// ObjectFlagsAnonymous|ObjectFlagsInstantiationExpression: Originating instantiation expression type
// ObjectFlagsAnonymous|ObjectFlagsInstantiated|ObjectFlagsInstantiationExpression: Instantiated instantiation expression type

// ReverseMappedType:
// ObjectFlagsAnonymous|ObjectFlagsReverseMapped: Reverse mapped type

// EvolvingArrayType:
// ObjectFlagsEvolvingArray: Evolving array type

// ObjectType: the target of an instantiated type in word 0, except for reverse mapped and evolving array types, and
// the mapper of an anonymous, mapped or instantiation expression type in word 1. A deferred type reference keeps its
// mapper in extra.

type ObjectType struct {
	StructuredType
}

func (t *ObjectType) AsObjectType() *ObjectType { return t }

// Target of instantiated type
func (t *ObjectType) target(c *Checker) *Type {
	return c.typ(t.targetId())
}

func (t *ObjectType) targetId() TypeId {
	if t.kind() >= typeKindReverseMapped {
		return 0
	}
	return TypeId(t.data[0])
}

func (t *ObjectType) setTarget(target *Type) {
	if t.kind() >= typeKindReverseMapped {
		panic("Type has no target")
	}
	t.data[0] = uint32(target.ref())
}

// Type mapper for instantiated type
func (t *ObjectType) mapper(c *Checker) *TypeMapper {
	switch t.kind() {
	case typeKindObject, typeKindInstantiationExpression, typeKindMapped:
		return c.mapperOf(TypeMapperId(t.data[1]))
	case typeKindTypeReference:
		if t.extra != 0 {
			return c.mapperOf(t.ext(c).mapper)
		}
	}
	return nil
}

func (t *ObjectType) setMapper(c *Checker, mapper *TypeMapper) {
	switch t.kind() {
	case typeKindObject, typeKindInstantiationExpression, typeKindMapped:
		t.data[1] = uint32(c.mapperId(mapper))
	case typeKindTypeReference:
		if mapper != nil || t.extra != 0 {
			t.ensureExtra(c).mapper = c.mapperId(mapper)
		}
	default:
		if mapper != nil {
			panic("Type has no mapper")
		}
	}
}

// Map of type instantiations
func (t *ObjectType) instantiations(c *Checker) map[CacheHashKey]TypeId {
	if t.extra == 0 {
		return nil
	}
	return t.ext(c).instantiations
}

func (t *ObjectType) setInstantiations(c *Checker, instantiations map[CacheHashKey]TypeId) {
	if instantiations == nil && t.extra == 0 {
		return
	}
	t.ensureExtra(c).instantiations = instantiations
}

// TypeReference (instantiation of an InterfaceType): the target in word 0, the type arguments' list handle in words 2
// and 3, and in word 4 the index of the lazy state plus one. Interface and tuple types add their own record's index
// plus one in word 5.

type TypeReference struct {
	ObjectType
}

type LazyState struct {
	layout             *MemberLayout
	mapper             TypeMapperId
	baseTypes          []TypeId
	propertiesResolved bool
}

type MemberLayout struct {
	id               uint32          // Unique identifier for the member layout
	members          ast.SymbolTable // MemberInfos keyed by property name
	baseIndices      []int32         // Indices of base types for each member (-1 if locally declared)
	layoutProperties []ast.SymbolId  // Lazily created list of layout properties (the symbols in members)
}

func (t *TypeReference) AsTypeReference() *TypeReference { return t }

// TypeReferenceNode | ArrayTypeNode | TupleTypeNode when deferred, else the zero NodeRef
func (t *TypeReference) node(c *Checker) ast.NodeRef {
	if t.extra == 0 {
		return ast.NodeRef{}
	}
	return t.ext(c).node
}

func (t *TypeReference) setNode(c *Checker, node ast.NodeRef) {
	if node == (ast.NodeRef{}) && t.extra == 0 {
		return
	}
	t.ensureExtra(c).node = node
}

func (t *TypeReference) resolvedTypeArguments(c *Checker) []TypeId {
	return c.typeLists.slice(t.data[2], t.data[3])
}

func (t *TypeReference) setResolvedTypeArguments(c *Checker, typeArguments []TypeId) {
	t.data[2], t.data[3] = c.typeListOf(typeArguments)
}

// Resolved member layout and instantiation state
func (t *TypeReference) lazy(c *Checker) *LazyState {
	return pageAt(&c.lazyStates, t.data[4])
}

// newLazy gives the type a zero lazy state and returns it.
func (t *TypeReference) newLazy(c *Checker) *LazyState {
	var lazy *LazyState
	t.data[4], lazy = pageAdd(&c.lazyStates)
	return lazy
}

// InterfaceType (when generic, serves as reference to instantiation of itself): its own record in p2.

type InterfaceType struct {
	TypeReference
}

type interfaceData struct {
	allTypeParameters           []TypeId // Type parameters (outer + local + thisType)
	outerTypeParameterCount     int      // Count of outer type parameters
	thisType                    TypeId   // The "this" type (nil if none)
	baseTypesResolved           bool
	declaredMembersResolved     bool
	resolvedBaseConstructorType TypeId
	resolvedBaseTypes           []TypeId
	declaredMembers             ast.SymbolTable // Declared members
	declaredCallSignatures      []*Signature    // Declared call signatures
	declaredConstructSignatures []*Signature    // Declared construct signatures
	declaredIndexInfos          []*IndexInfo    // Declared index signatures
}

func (t *InterfaceType) AsInterfaceType() *InterfaceType { return t }

func (t *InterfaceType) iface(c *Checker) *interfaceData {
	if t.kind() == typeKindTuple {
		return &c.tupleDatas.At(t.data[5] - 1).interfaceData
	}
	return c.interfaceDatas.At(t.data[5] - 1)
}

func (t *InterfaceType) OuterTypeParameters(c *Checker) []TypeId {
	d := t.iface(c)
	if len(d.allTypeParameters) == 0 {
		return nil
	}
	return slices.Clip(d.allTypeParameters[:d.outerTypeParameterCount])
}

func (t *InterfaceType) LocalTypeParameters(c *Checker) []TypeId {
	d := t.iface(c)
	if len(d.allTypeParameters) == 0 {
		return nil
	}
	return slices.Clip(d.allTypeParameters[d.outerTypeParameterCount : len(d.allTypeParameters)-1])
}

func (t *InterfaceType) TypeParameters(c *Checker) []TypeId {
	d := t.iface(c)
	if len(d.allTypeParameters) == 0 {
		return nil
	}
	return slices.Clip(d.allTypeParameters[:len(d.allTypeParameters)-1])
}

func (t *InterfaceType) ThisType(c *Checker) *Type {
	return c.typ(t.iface(c).thisType)
}

// TupleType

type ElementFlags uint32

const (
	ElementFlagsNone        ElementFlags = 0
	ElementFlagsRequired    ElementFlags = 1 << 0 // T
	ElementFlagsOptional    ElementFlags = 1 << 1 // T?
	ElementFlagsRest        ElementFlags = 1 << 2 // ...T[]
	ElementFlagsVariadic    ElementFlags = 1 << 3 // ...T
	ElementFlagsFixed                    = ElementFlagsRequired | ElementFlagsOptional
	ElementFlagsVariable                 = ElementFlagsRest | ElementFlagsVariadic
	ElementFlagsNonRequired              = ElementFlagsOptional | ElementFlagsRest | ElementFlagsVariadic
	ElementFlagsNonRest                  = ElementFlagsRequired | ElementFlagsOptional | ElementFlagsVariadic
)

type TupleElementInfo struct {
	flags              ElementFlags
	labeledDeclaration ast.Node // NamedTupleMember | ParameterDeclaration | nil
}

func (t *TupleElementInfo) TupleElementFlags() ElementFlags { return t.flags }
func (t *TupleElementInfo) LabeledDeclaration() ast.Node    { return t.labeledDeclaration }

// TupleType: like InterfaceType, with a larger record.

type TupleType struct {
	InterfaceType
}

type tupleData struct {
	interfaceData
	elementInfos  []TupleElementInfo
	minLength     int // Number of required or variadic elements
	fixedLength   int // Number of initial required or optional elements
	combinedFlags ElementFlags
	readonly      bool
}

func (t *TupleType) tuple(c *Checker) *tupleData {
	return c.tupleDatas.At(t.data[5] - 1)
}

func (t *TupleType) FixedLength(c *Checker) int { return t.tuple(c).fixedLength }
func (t *TupleType) IsReadonly(c *Checker) bool { return t.tuple(c).readonly }
func (t *TupleType) ElementFlags(c *Checker) []ElementFlags {
	elementInfos := t.tuple(c).elementInfos
	elementFlags := make([]ElementFlags, len(elementInfos))
	for i, info := range elementInfos {
		elementFlags[i] = info.flags
	}
	return elementFlags
}
func (t *TupleType) ElementInfos(c *Checker) []TupleElementInfo { return t.tuple(c).elementInfos }

// InstantiationExpressionType: the node in words 2 and 3.

type InstantiationExpressionType struct {
	ObjectType
}

func (t *InstantiationExpressionType) node() ast.NodeRef {
	// SAFETY: Words 2 and 3 are two adjacent 32-bit words, laid out like a NodeRef.
	return *(*ast.NodeRef)(unsafe.Pointer(&t.data[2]))
}

func (t *InstantiationExpressionType) setNode(node ast.NodeRef) {
	// SAFETY: Words 2 and 3 are two adjacent 32-bit words, laid out like a NodeRef.
	*(*ast.NodeRef)(unsafe.Pointer(&t.data[2])) = node
}

// MappedType: its own record's index plus one in word 5.

type MappedType struct {
	ObjectType
}

type mappedData struct {
	declaration          ast.MappedTypeNode
	typeParameter        TypeId
	constraintType       TypeId
	nameType             TypeId
	templateType         TypeId
	modifiersType        TypeId
	resolvedApparentType TypeId
	containsError        bool
}

func (t *MappedType) mapped(c *Checker) *mappedData {
	return c.mappedDatas.At(t.data[5] - 1)
}

func (t *MappedType) TypeParameter(c *Checker) *Type  { return c.typ(t.mapped(c).typeParameter) }
func (t *MappedType) ConstraintType(c *Checker) *Type { return c.typ(t.mapped(c).constraintType) }
func (t *MappedType) NameType(c *Checker) *Type       { return c.typ(t.mapped(c).nameType) }
func (t *MappedType) TemplateType(c *Checker) *Type   { return c.typ(t.mapped(c).templateType) }
func (t *MappedType) ResolveComponents(c *Checker, typ *Type) {
	c.getTypeParameterFromMappedType(typ)
	c.getConstraintTypeFromMappedType(typ)
	c.getNameTypeFromMappedType(typ)
	c.getTemplateTypeFromMappedType(typ)
}

// ReverseMappedType: the source, mapped type and constraint type in words 0, 1 and 2.

type ReverseMappedType struct {
	ObjectType
}

func (t *ReverseMappedType) source(c *Checker) *Type         { return c.typ(TypeId(t.data[0])) }
func (t *ReverseMappedType) setSource(v *Type)               { t.data[0] = uint32(v.ref()) }
func (t *ReverseMappedType) mappedType(c *Checker) *Type     { return c.typ(TypeId(t.data[1])) }
func (t *ReverseMappedType) setMappedType(v *Type)           { t.data[1] = uint32(v.ref()) }
func (t *ReverseMappedType) constraintType(c *Checker) *Type { return c.typ(TypeId(t.data[2])) }
func (t *ReverseMappedType) setConstraintType(v *Type)       { t.data[2] = uint32(v.ref()) }

// EvolvingArrayType: the element type and final array type in words 0 and 1.

type EvolvingArrayType struct {
	ObjectType
}

func (t *EvolvingArrayType) elementType(c *Checker) *Type    { return c.typ(TypeId(t.data[0])) }
func (t *EvolvingArrayType) setElementType(v *Type)          { t.data[0] = uint32(v.ref()) }
func (t *EvolvingArrayType) finalArrayType(c *Checker) *Type { return c.typ(TypeId(t.data[1])) }
func (t *EvolvingArrayType) setFinalArrayType(v *Type)       { t.data[1] = uint32(v.ref()) }

// UnionOrIntersectionType: the types' list handle in words 0 and 1, and in word 3 the index plus one of a record of
// rarer fields allocated on first use.

type UnionOrIntersectionType struct {
	StructuredType
}

type unionOrIntersectionData struct {
	propertyCache                               ast.SymbolTable // Augmented entries, except complete ones already in the non-augmented cache.
	propertyCacheWithoutFunctionPropertyAugment ast.SymbolTable
	resolvedProperties                          []ast.SymbolId
}

func (t *UnionOrIntersectionType) AsUnionOrIntersectionType() *UnionOrIntersectionType { return t }

func (t *UnionOrIntersectionType) types(c *Checker) []TypeId {
	return c.typeLists.slice(t.data[0], t.data[1])
}

func (t *UnionOrIntersectionType) setTypes(c *Checker, types []TypeId) {
	t.data[0], t.data[1] = c.typeListOf(types)
}

func (t *UnionOrIntersectionType) Types(c *Checker) []TypeId {
	return t.types(c)
}

func (t *UnionOrIntersectionType) more(c *Checker) *unionOrIntersectionData {
	if t.data[3] == 0 {
		return nil
	}
	if t.kind() == typeKindUnion {
		return &c.unionDatas.At(t.data[3] - 1).unionOrIntersectionData
	}
	return &c.intersectionDatas.At(t.data[3] - 1).unionOrIntersectionData
}

func (t *UnionOrIntersectionType) ensureMore(c *Checker) *unionOrIntersectionData {
	if t.data[3] == 0 {
		if t.kind() == typeKindUnion {
			t.data[3], _ = pageAdd(&c.unionDatas)
		} else {
			t.data[3], _ = pageAdd(&c.intersectionDatas)
		}
	}
	return t.more(c)
}

func (t *UnionOrIntersectionType) propertyCache(c *Checker) ast.SymbolTable {
	if more := t.more(c); more != nil {
		return more.propertyCache
	}
	return nil
}

func (t *UnionOrIntersectionType) setPropertyCache(c *Checker, v ast.SymbolTable) {
	t.ensureMore(c).propertyCache = v
}

func (t *UnionOrIntersectionType) propertyCacheWithoutFunctionPropertyAugment(c *Checker) ast.SymbolTable {
	if more := t.more(c); more != nil {
		return more.propertyCacheWithoutFunctionPropertyAugment
	}
	return nil
}

func (t *UnionOrIntersectionType) setPropertyCacheWithoutFunctionPropertyAugment(c *Checker, v ast.SymbolTable) {
	t.ensureMore(c).propertyCacheWithoutFunctionPropertyAugment = v
}

func (t *UnionOrIntersectionType) resolvedProperties(c *Checker) []ast.SymbolId {
	if more := t.more(c); more != nil {
		return more.resolvedProperties
	}
	return nil
}

func (t *UnionOrIntersectionType) setResolvedProperties(c *Checker, v []ast.SymbolId) {
	t.ensureMore(c).resolvedProperties = v
}

// UnionType: the origin in word 2.

type UnionType struct {
	UnionOrIntersectionType
}

type unionData struct {
	unionOrIntersectionData
	resolvedReducedType TypeId
	regularType         TypeId
	keyPropertyName     ast.SymbolName    // Property with unique unit type that exists in every object/intersection in union type
	constituentMap      map[TypeId]TypeId // Constituents keyed by unit type discriminants
}

func (t *UnionType) union(c *Checker) *unionData {
	return pageAt(&c.unionDatas, t.data[3])
}

// Denormalized union, intersection, or index type in which union originates
func (t *UnionType) origin(c *Checker) *Type { return c.typ(TypeId(t.data[2])) }
func (t *UnionType) setOrigin(v *Type)       { t.data[2] = uint32(v.ref()) }

func (t *UnionType) resolvedReducedType(c *Checker) *Type {
	if d := t.union(c); d != nil {
		return c.typ(d.resolvedReducedType)
	}
	return nil
}

func (t *UnionType) setResolvedReducedType(c *Checker, v *Type) {
	t.ensureMore(c)
	t.union(c).resolvedReducedType = v.ref()
}

func (t *UnionType) regularType(c *Checker) *Type {
	if d := t.union(c); d != nil {
		return c.typ(d.regularType)
	}
	return nil
}

func (t *UnionType) setRegularType(c *Checker, v *Type) {
	t.ensureMore(c)
	t.union(c).regularType = v.ref()
}

func (t *UnionType) keyPropertyName(c *Checker) ast.SymbolName {
	if d := t.union(c); d != nil {
		return d.keyPropertyName
	}
	return ast.SymbolName(0)
}

func (t *UnionType) setKeyPropertyName(c *Checker, v ast.SymbolName) {
	t.ensureMore(c)
	t.union(c).keyPropertyName = v
}

func (t *UnionType) constituentMap(c *Checker) map[TypeId]TypeId {
	if d := t.union(c); d != nil {
		return d.constituentMap
	}
	return nil
}

func (t *UnionType) setConstituentMap(c *Checker, v map[TypeId]TypeId) {
	t.ensureMore(c)
	t.union(c).constituentMap = v
}

// IntersectionType: the resolved apparent type in word 2.

type IntersectionType struct {
	UnionOrIntersectionType
}

type intersectionData struct {
	unionOrIntersectionData
	uniqueLiteralFilledInstantiation TypeId // Instantiation with type parameters mapped to never type
}

func (t *IntersectionType) resolvedApparentType(c *Checker) *Type { return c.typ(TypeId(t.data[2])) }
func (t *IntersectionType) setResolvedApparentType(v *Type)       { t.data[2] = uint32(v.ref()) }

func (t *IntersectionType) uniqueLiteralFilledInstantiation(c *Checker) *Type {
	if d := pageAt(&c.intersectionDatas, t.data[3]); d != nil {
		return c.typ(d.uniqueLiteralFilledInstantiation)
	}
	return nil
}

func (t *IntersectionType) setUniqueLiteralFilledInstantiation(c *Checker, v *Type) {
	t.ensureMore(c)
	c.intersectionDatas.At(t.data[3] - 1).uniqueLiteralFilledInstantiation = v.ref()
}

// TypeParameter: the constraint, target and mapper in words 0, 1 and 2, and flags in word 3. The default and
// distributed type are in extra.

type TypeParameter struct {
	ConstrainedType
}

const (
	typeParameterIsThisType = 1 << iota
	typeParameterIsDistributed
)

func (t *TypeParameter) constraint(c *Checker) *Type         { return c.typ(TypeId(t.data[0])) }
func (t *TypeParameter) setConstraint(v *Type)               { t.data[0] = uint32(v.ref()) }
func (t *TypeParameter) target(c *Checker) *Type             { return c.typ(t.targetId()) }
func (t *TypeParameter) targetId() TypeId                    { return TypeId(t.data[1]) }
func (t *TypeParameter) setTarget(v *Type)                   { t.data[1] = uint32(v.ref()) }
func (t *TypeParameter) mapper(c *Checker) *TypeMapper       { return c.mapperOf(TypeMapperId(t.data[2])) }
func (t *TypeParameter) setMapper(c *Checker, v *TypeMapper) { t.data[2] = uint32(c.mapperId(v)) }
func (t *TypeParameter) IsThisType() bool                    { return t.data[3]&typeParameterIsThisType != 0 }
func (t *TypeParameter) isDistributed() bool                 { return t.data[3]&typeParameterIsDistributed != 0 }
func (t *TypeParameter) setIsThisType(v bool)                { t.setFlag(typeParameterIsThisType, v) }
func (t *TypeParameter) setIsDistributed(v bool)             { t.setFlag(typeParameterIsDistributed, v) }

func (t *TypeParameter) setFlag(flag uint32, v bool) {
	if v {
		t.data[3] |= flag
	} else {
		t.data[3] &^= flag
	}
}

func (t *TypeParameter) resolvedDefaultType(c *Checker) *Type {
	if t.extra == 0 {
		return nil
	}
	return c.typ(t.ext(c).resolvedDefaultType)
}

func (t *TypeParameter) setResolvedDefaultType(c *Checker, v *Type) {
	if v == nil && t.extra == 0 {
		return
	}
	t.ensureExtra(c).resolvedDefaultType = v.ref()
}

func (t *TypeParameter) distributedType(c *Checker) *Type {
	if t.extra == 0 {
		return nil
	}
	return c.typ(t.ext(c).distributedType)
}

func (t *TypeParameter) setDistributedType(c *Checker, v *Type) {
	if v == nil && t.extra == 0 {
		return
	}
	t.ensureExtra(c).distributedType = v.ref()
}

// IndexFlags

type IndexFlags uint32

const (
	IndexFlagsNone              IndexFlags = 0
	IndexFlagsStringsOnly       IndexFlags = 1 << 0
	IndexFlagsNoIndexSignatures IndexFlags = 1 << 1
	IndexFlagsNoReducibleCheck  IndexFlags = 1 << 2
)

// IndexType: the target in word 0 and the flags in word 1.

type IndexType struct {
	ConstrainedType
}

func (t *IndexType) Target(c *Checker) *Type    { return t.target(c) }
func (t *IndexType) target(c *Checker) *Type    { return c.typ(t.targetId()) }
func (t *IndexType) targetId() TypeId           { return TypeId(t.data[0]) }
func (t *IndexType) setTarget(v *Type)          { t.data[0] = uint32(v.ref()) }
func (t *IndexType) indexFlags() IndexFlags     { return IndexFlags(t.data[1]) }
func (t *IndexType) setIndexFlags(v IndexFlags) { t.data[1] = uint32(v) }

// IndexedAccessType: the object and index types in words 0 and 1, and the access flags (only AccessFlags.Persistent)
// in word 2.

type IndexedAccessType struct {
	ConstrainedType
}

func (t *IndexedAccessType) ObjectType(c *Checker) *Type  { return t.objectType(c) }
func (t *IndexedAccessType) IndexType(c *Checker) *Type   { return t.indexType(c) }
func (t *IndexedAccessType) objectType(c *Checker) *Type  { return c.typ(TypeId(t.data[0])) }
func (t *IndexedAccessType) setObjectType(v *Type)        { t.data[0] = uint32(v.ref()) }
func (t *IndexedAccessType) indexType(c *Checker) *Type   { return c.typ(TypeId(t.data[1])) }
func (t *IndexedAccessType) setIndexType(v *Type)         { t.data[1] = uint32(v.ref()) }
func (t *IndexedAccessType) accessFlags() AccessFlags     { return AccessFlags(t.data[2]) }
func (t *IndexedAccessType) setAccessFlags(v AccessFlags) { t.data[2] = uint32(v) }

// TemplateLiteralType: in word 0 the index plus one of the texts (always one element longer than the types) in the
// checker's template texts, and the types' list handle (always at least one) in words 1 and 2.

type TemplateLiteralType struct {
	ConstrainedType
}

func (t *TemplateLiteralType) Texts(c *Checker) []string { return t.texts(c) }
func (t *TemplateLiteralType) Types(c *Checker) []TypeId { return t.types(c) }

func (t *TemplateLiteralType) texts(c *Checker) []string {
	return *c.templateTexts.At(t.data[0] - 1)
}

func (t *TemplateLiteralType) setTexts(c *Checker, texts []string) {
	var p *[]string
	t.data[0], p = pageAdd(&c.templateTexts)
	*p = texts
}

func (t *TemplateLiteralType) types(c *Checker) []TypeId {
	return c.typeLists.slice(t.data[1], t.data[2])
}

func (t *TemplateLiteralType) setTypes(c *Checker, types []TypeId) {
	t.data[1], t.data[2] = c.typeListOf(types)
}

// StringMappingType: the target in word 0.

type StringMappingType struct {
	ConstrainedType
}

func (t *StringMappingType) Target(c *Checker) *Type { return t.target(c) }
func (t *StringMappingType) target(c *Checker) *Type { return c.typ(t.targetId()) }
func (t *StringMappingType) targetId() TypeId        { return TypeId(t.data[0]) }
func (t *StringMappingType) setTarget(v *Type)       { t.data[0] = uint32(v.ref()) }

// SubstitutionType: the base type (the target type) and the constraint it is known to satisfy in words 0 and 1.

type SubstitutionType struct {
	ConstrainedType
}

func (t *SubstitutionType) BaseType(c *Checker) *Type        { return t.baseType(c) }
func (t *SubstitutionType) SubstConstraint(c *Checker) *Type { return t.constraint(c) }
func (t *SubstitutionType) baseType(c *Checker) *Type        { return c.typ(TypeId(t.data[0])) }
func (t *SubstitutionType) setBaseType(v *Type)              { t.data[0] = uint32(v.ref()) }
func (t *SubstitutionType) constraint(c *Checker) *Type      { return c.typ(TypeId(t.data[1])) }
func (t *SubstitutionType) setConstraint(v *Type)            { t.data[1] = uint32(v.ref()) }

type ConditionalRoot struct {
	node                ast.ConditionalTypeNode
	checkType           TypeId
	extendsType         TypeId
	isDistributive      bool
	inferTypeParameters []TypeId
	outerTypeParameters []TypeId
	instantiations      map[CacheHashKey]TypeId
	alias               *TypeAlias
}

// ConditionalType: its own record's index plus one in word 0, and the check and extends types in words 1 and 2.

type ConditionalType struct {
	ConstrainedType
}

type conditionalData struct {
	root                             *ConditionalRoot
	resolvedTrueType                 TypeId
	resolvedFalseType                TypeId
	resolvedInferredTrueType         TypeId // The `trueType` instantiated with the `combinedMapper`, if present
	resolvedDefaultConstraint        TypeId
	resolvedConstraintOfDistributive TypeId
	mapper                           TypeMapperId
	combinedMapper                   TypeMapperId
}

func (t *ConditionalType) cond(c *Checker) *conditionalData {
	return c.conditionalDatas.At(t.data[0] - 1)
}
func (t *ConditionalType) CheckType(c *Checker) *Type   { return t.checkType(c) }
func (t *ConditionalType) ExtendsType(c *Checker) *Type { return t.extendsType(c) }
func (t *ConditionalType) checkType(c *Checker) *Type   { return c.typ(TypeId(t.data[1])) }
func (t *ConditionalType) setCheckType(v *Type)         { t.data[1] = uint32(v.ref()) }
func (t *ConditionalType) extendsType(c *Checker) *Type { return c.typ(TypeId(t.data[2])) }
func (t *ConditionalType) setExtendsType(v *Type)       { t.data[2] = uint32(v.ref()) }

// SignatureFlags

type SignatureFlags uint32

const (
	SignatureFlagsNone SignatureFlags = 0
	// Propagating flags
	SignatureFlagsHasRestParameter SignatureFlags = 1 << 0 // Indicates last parameter is rest parameter
	SignatureFlagsHasLiteralTypes  SignatureFlags = 1 << 1 // Indicates signature is specialized
	SignatureFlagsConstruct        SignatureFlags = 1 << 2 // Indicates signature is a construct signature
	SignatureFlagsAbstract         SignatureFlags = 1 << 3 // Indicates signature comes from an abstract class, abstract construct signature, or abstract constructor type
	// Non-propagating flags
	SignatureFlagsIsInnerCallChain                       SignatureFlags = 1 << 4 // Indicates signature comes from a CallChain nested in an outer OptionalChain
	SignatureFlagsIsOuterCallChain                       SignatureFlags = 1 << 5 // Indicates signature comes from a CallChain that is the outermost chain of an optional expression
	SignatureFlagsIsUntypedSignatureInJSFile             SignatureFlags = 1 << 6 // Indicates signature is from a js file and has no types
	SignatureFlagsIsNonInferrable                        SignatureFlags = 1 << 7 // Indicates signature comes from a non-inferrable type
	SignatureFlagsIsSignatureCandidateForOverloadFailure SignatureFlags = 1 << 8
	signatureFlagsNoTypePredicate                        SignatureFlags = 1 << 9
	// We do not propagate `IsInnerCallChain` or `IsOuterCallChain` to instantiated signatures, as that would result in us
	// attempting to add `| undefined` on each recursive call to `getReturnTypeOfSignature` when
	// instantiating the return type.
	SignatureFlagsPropagatingFlags = SignatureFlagsHasRestParameter | SignatureFlagsHasLiteralTypes | SignatureFlagsConstruct | SignatureFlagsAbstract | SignatureFlagsIsUntypedSignatureInJSFile | SignatureFlagsIsSignatureCandidateForOverloadFailure
	SignatureFlagsCallChainFlags   = SignatureFlagsIsInnerCallChain | SignatureFlagsIsOuterCallChain
)

// Signature

type Signature struct {
	id                       SignatureId
	flags                    SignatureFlags
	minArgumentCount         int32
	resolvedMinArgumentCount int32
	declaration              ast.NodeRef
	parameters               []ast.SymbolId
	resolvedReturnType       TypeId
	target                   SignatureId
	mapper                   TypeMapperId
	extra                    *signatureExtra
}

type signatureExtra struct {
	typeParameters        []TypeId
	thisParameter         ast.SymbolId
	resolvedTypePredicate *TypePredicate
	isolatedSignatureType TypeId
	composite             *CompositeSignature
}

func (s *Signature) ensureExtra(c *Checker) *signatureExtra {
	if s.extra == nil {
		s.extra = c.signatureExtraArena.New()
	}
	return s.extra
}

func (s *Signature) setTypeParameters(c *Checker, types []TypeId) {
	if types != nil || s.extra != nil {
		s.ensureExtra(c).typeParameters = types
	}
}

func (s *Signature) setThisParameter(c *Checker, parameter ast.SymbolId) {
	if parameter != 0 || s.extra != nil {
		s.ensureExtra(c).thisParameter = parameter
	}
}

func (s *Signature) getResolvedTypePredicate(c *Checker) *TypePredicate {
	if s.flags&signatureFlagsNoTypePredicate != 0 {
		return c.noTypePredicate
	}
	if s.extra == nil {
		return nil
	}
	return s.extra.resolvedTypePredicate
}

func (s *Signature) setResolvedTypePredicate(c *Checker, predicate *TypePredicate) {
	s.flags &^= signatureFlagsNoTypePredicate
	if predicate != nil && predicate == c.noTypePredicate {
		s.flags |= signatureFlagsNoTypePredicate
		predicate = nil
	}
	if predicate != nil || s.extra != nil {
		s.ensureExtra(c).resolvedTypePredicate = predicate
	}
}

func (s *Signature) getComposite() *CompositeSignature {
	if s.extra == nil {
		return nil
	}
	return s.extra.composite
}

func (s *Signature) setComposite(c *Checker, composite *CompositeSignature) {
	if composite != nil || s.extra != nil {
		s.ensureExtra(c).composite = composite
	}
}

func (s *Signature) getIsolatedSignatureType(c *Checker) *Type {
	if s.extra == nil {
		return nil
	}
	return c.typ(s.extra.isolatedSignatureType)
}

func (s *Signature) Id() SignatureId {
	return s.id
}

func (s *Signature) Flags() SignatureFlags {
	return s.flags &^ signatureFlagsNoTypePredicate
}

func (s *Signature) TypeParameters() []TypeId {
	if s.extra == nil {
		return nil
	}
	return s.extra.typeParameters
}

func (s *Signature) Target(c *Checker) *Signature {
	return c.signatureOf(s.target)
}

func (s *Signature) ThisParameter() ast.SymbolId {
	if s.extra == nil {
		return 0
	}
	return s.extra.thisParameter
}

func (s *Signature) Parameters() []ast.SymbolId {
	return s.parameters
}

func (s *Signature) HasRestParameter() bool {
	return s.flags&SignatureFlagsHasRestParameter != 0
}

func (s *Signature) MinArgumentCount() int {
	return int(s.minArgumentCount)
}

type CompositeSignature struct {
	isUnion    bool         // True for union, false for intersection
	signatures []*Signature // Individual signatures
}

type TypePredicateKind int32

const (
	TypePredicateKindThis TypePredicateKind = iota
	TypePredicateKindIdentifier
	TypePredicateKindAssertsThis
	TypePredicateKindAssertsIdentifier
)

type TypePredicate struct {
	kind           TypePredicateKind
	parameterIndex int32
	parameterName  ast.SymbolName
	t              TypeId
}

func (typePredicate *TypePredicate) Type(c *Checker) *Type {
	return c.typ(typePredicate.t)
}

func (typePredicate *TypePredicate) Kind() TypePredicateKind {
	return typePredicate.kind
}

func (typePredicate *TypePredicate) ParameterIndex() int32 {
	return typePredicate.parameterIndex
}

func (typePredicate *TypePredicate) ParameterName() ast.SymbolName {
	return typePredicate.parameterName
}

// IndexInfo

// IndexInfoId names an index info: its index in the checker's index infos plus one, 0 for none.
type IndexInfoId uint32

type IndexInfo struct {
	keyType     TypeId
	valueType   TypeId
	id          IndexInfoId
	isReadonly  bool
	declaration ast.Node     // IndexSignatureDeclaration
	indexSymbol ast.SymbolId // Synthetic property symbol for this index signature
	components  []ast.Node   // ElementWithComputedPropertyName
}

func (info *IndexInfo) KeyType(c *Checker) *Type {
	return c.typ(info.keyType)
}

func (info *IndexInfo) ValueType(c *Checker) *Type {
	return c.typ(info.valueType)
}

func (info *IndexInfo) IsReadonly() bool {
	return info.isReadonly
}

func (info *IndexInfo) Declaration() ast.Node {
	return info.declaration
}

/**
 * Ternary values are defined such that
 * x & y picks the lesser in the order False < Unknown < Maybe < True, and
 * x | y picks the greater in the order False < Unknown < Maybe < True.
 * Generally, Ternary.Maybe is used as the result of a relation that depends on itself, and
 * Ternary.Unknown is used as the result of a variance check that depends on itself. We make
 * a distinction because we don't want to cache circular variance check results.
 */
type Ternary int8

const (
	TernaryFalse   Ternary = 0
	TernaryUnknown Ternary = 1
	TernaryMaybe   Ternary = 3
	TernaryTrue    Ternary = -1
)

type TypeComparer func(s *Type, t *Type, reportErrors bool) Ternary

type LanguageFeatureMinimumTargetMap struct {
	Exponentiation                    core.ScriptTarget
	AsyncFunctions                    core.ScriptTarget
	ForAwaitOf                        core.ScriptTarget
	AsyncGenerators                   core.ScriptTarget
	AsyncIteration                    core.ScriptTarget
	ObjectSpreadRest                  core.ScriptTarget
	RegularExpressionFlagsDotAll      core.ScriptTarget
	BindinglessCatch                  core.ScriptTarget
	BigInt                            core.ScriptTarget
	NullishCoalesce                   core.ScriptTarget
	OptionalChaining                  core.ScriptTarget
	LogicalAssignment                 core.ScriptTarget
	TopLevelAwait                     core.ScriptTarget
	ClassFields                       core.ScriptTarget
	PrivateNamesAndClassStaticBlocks  core.ScriptTarget
	RegularExpressionFlagsHasIndices  core.ScriptTarget
	ShebangComments                   core.ScriptTarget
	UsingAndAwaitUsing                core.ScriptTarget
	ClassAndClassElementDecorators    core.ScriptTarget
	RegularExpressionFlagsUnicodeSets core.ScriptTarget
}

var LanguageFeatureMinimumTarget = LanguageFeatureMinimumTargetMap{
	Exponentiation:                    core.ScriptTargetES2016,
	AsyncFunctions:                    core.ScriptTargetES2017,
	ForAwaitOf:                        core.ScriptTargetES2018,
	AsyncGenerators:                   core.ScriptTargetES2018,
	AsyncIteration:                    core.ScriptTargetES2018,
	ObjectSpreadRest:                  core.ScriptTargetES2018,
	RegularExpressionFlagsDotAll:      core.ScriptTargetES2018,
	BindinglessCatch:                  core.ScriptTargetES2019,
	BigInt:                            core.ScriptTargetES2020,
	NullishCoalesce:                   core.ScriptTargetES2020,
	OptionalChaining:                  core.ScriptTargetES2020,
	LogicalAssignment:                 core.ScriptTargetES2021,
	TopLevelAwait:                     core.ScriptTargetES2022,
	ClassFields:                       core.ScriptTargetES2022,
	PrivateNamesAndClassStaticBlocks:  core.ScriptTargetES2022,
	RegularExpressionFlagsHasIndices:  core.ScriptTargetES2022,
	ShebangComments:                   core.ScriptTargetESNext,
	UsingAndAwaitUsing:                core.ScriptTargetESNext,
	ClassAndClassElementDecorators:    core.ScriptTargetESNext,
	RegularExpressionFlagsUnicodeSets: core.ScriptTargetESNext,
}

// Aliases for types
type StringLiteralType = Type

func (c *Checker) SymbolOfType(t *Type) ast.Symbol {
	return c.symbolOf(c.symbolOfTypeId(t))
}

func (t *StructuredType) Properties(c *Checker) []ast.Symbol {
	return c.symbolsOf(t.propertiesId(c))
}
