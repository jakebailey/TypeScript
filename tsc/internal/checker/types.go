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

// Links for referenced symbols

type SymbolReferenceLinks struct {
	referenceKinds ast.SymbolFlags // Flags for the meanings of the symbol that were referenced
}

// Links for value symbols

type ValueSymbolLinks struct {
	resolvedType *Type // Type of value symbol
	target       ast.SymbolId
	mapper       *TypeMapper
	extra        *valueSymbolExtra
}

type valueSymbolExtra struct {
	writeType      *Type
	nameType       *Type
	containingType *Type // Mapped type for mapped type property, containing union or intersection type for synthetic property
}

func (links *ValueSymbolLinks) getWriteType() *Type {
	if links.extra == nil {
		return nil
	}
	return links.extra.writeType
}

func (links *ValueSymbolLinks) getNameType() *Type {
	if links.extra == nil {
		return nil
	}
	return links.extra.nameType
}

func (links *ValueSymbolLinks) getContainingType() *Type {
	if links.extra == nil {
		return nil
	}
	return links.extra.containingType
}

func (links *ValueSymbolLinks) setWriteType(c *Checker, typ *Type) {
	if links.extra == nil {
		if typ == nil {
			return
		}
		links.extra = c.valueSymbolExtraArena.New()
	}
	links.extra.writeType = typ
}

func (links *ValueSymbolLinks) setNameType(c *Checker, typ *Type) {
	if links.extra == nil {
		if typ == nil {
			return
		}
		links.extra = c.valueSymbolExtraArena.New()
	}
	links.extra.nameType = typ
}

func (links *ValueSymbolLinks) setContainingType(c *Checker, typ *Type) {
	if links.extra == nil {
		if typ == nil {
			return
		}
		links.extra = c.valueSymbolExtraArena.New()
	}
	links.extra.containingType = typ
}

// Additional links for mapped symbols

type MappedSymbolLinks struct {
	keyType         *Type      // Key type for mapped type member
	syntheticOrigin ast.Symbol // For a property on a mapped or spread type, points back to the original property
}

// Additional links for deferred type symbols

type DeferredSymbolLinks struct {
	parent            *Type   // Source union/intersection of a deferred type
	constituents      []*Type // Calculated list of constituents for a deferred type
	writeConstituents []*Type // Constituents of a deferred `writeType`
}

// Links for alias symbols

type AliasSymbolLinks struct {
	immediateTarget     ast.Symbol // Immediate target of an alias. May be another alias. Do not access directly, use `checker.getImmediateAliasedSymbol` instead.
	aliasTarget         ast.Symbol // Resolved (non-alias) target of an alias
	referenced          bool       // True if alias symbol has been referenced as a value that can be emitted
	typeOnlyDeclaration ast.Node   // First resolved alias declaration that makes the symbol only usable in type constructs
}

// Links for module symbols

type ModuleSymbolLinks struct {
	resolvedExports       ast.SymbolTable             // Resolved exports of module or combined early- and late-bound static members of a class.
	typeOnlyExportStarMap map[ast.SymbolName]ast.Node // Set on a module symbol when some of its exports were resolved through a 'export type * from "mod"' declaration
	exportsChecked        bool
}

type ReverseMappedSymbolLinks struct {
	propertyType   *Type
	mappedType     *Type // References a mapped type
	constraintType *Type // References an index type
}

// Links for late-bound symbols

type LateBoundLinks struct {
	lateSymbol ast.Symbol
}

// Links for export type symbols

type ExportTypeLinks struct {
	target            ast.Symbol // Target symbol
	originatingImport ast.Node   // Import declaration which produced the symbol, present if the symbol is marked as uncallable but had call signatures in `resolveESModuleSymbol`
}

// Links for type aliases

type TypeAliasLinks struct {
	declaredType                  *Type
	typeParameters                []*Type                // Type parameters of type alias (undefined if non-generic)
	instantiations                map[CacheHashKey]*Type // Instantiations of generic type alias (undefined if non-generic)
	isConstructorDeclaredProperty bool
}

// Links for declared types (type parameters, class types, interface types, enums)

type DeclaredTypeLinks struct {
	declaredType           *Type
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
	switchTypes         []*Type
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
	leftSpread  ast.Symbol // Left source for synthetic spread property
	rightSpread ast.Symbol // Right source for synthetic spread property
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
	extendedContainersByFile map[ast.NodeId][]ast.Symbol // Symbols of nodes which which logically contain this one, cached by file the request is made within
	extendedContainers       *[]ast.Symbol               // Containers (other than the parent) which this symbol is aliased in
	accessibleChainCache     map[accessibleChainCacheKey][]ast.Symbol
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

type TypeNodeLinks struct {
	resolvedType        *Type    // Resolved type associated with node
	outerTypeParameters *[]*Type // Outer type parameters of anonymous object type
}

func (links *TypeNodeLinks) getOuterTypeParameters() []*Type {
	if links.outerTypeParameters == nil {
		return nil
	}
	return *links.outerTypeParameters
}

func (links *TypeNodeLinks) setOuterTypeParameters(c *Checker, types []*Type) {
	if types == nil {
		links.outerTypeParameters = nil
		return
	}
	if links.outerTypeParameters == nil {
		links.outerTypeParameters = c.outerTypeParameterListArena.New()
	}
	*links.outerTypeParameters = types
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
	exprType *Type // Assertion expression type
}

// SourceFile links

type SourceFileLinks struct {
	typeChecked                  bool
	unusedChecked                bool
	externalHelpersModule        ast.Symbol
	requestedExternalEmitHelpers ExternalEmitHelpers
	deferredNodes                collections.OrderedSet[ast.Node]
	identifierCheckNodes         []ast.Node
	localJsxNamespace            string
	localJsxFragmentNamespace    string
	localJsxFactory              ast.EntityName
	localJsxFragmentFactory      ast.EntityName
	jsxFragmentType              *Type
}

// Signature specific links

type SignatureLinks struct {
	resolvedSignature  *Signature // Cached signature of signature node or call expression
	effectsSignature   *Signature // Signature with possible control flow effects
	decoratorSignature *Signature // Signature for decorator as if invoked by the runtime
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
	symbol        ast.Symbol
	typeArguments []*Type
}

func (a *TypeAlias) Symbol() ast.Symbol {
	if a == nil {
		return ast.Symbol{}
	}
	return a.symbol
}

func (a *TypeAlias) TypeArguments() []*Type {
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

// A Type is a 64-byte record in a checker-owned slab. The header holds what every type has; resolved holds the
// members, properties, signatures and index infos of object, union and intersection types once they are resolved;
// extra holds fields few types have; and data is a 32-byte payload whose meaning depends on the kind. Kinds with more
// state than fits keep it in a record the payload points to.
type Type struct {
	flags       TypeFlags
	objectFlags ObjectFlags
	id          TypeId // The type's id, with its typeKind in the high bits
	symbol      ast.SymbolId
	resolved    *structuredMembers
	extra       *typeExtra
	data        typeData
}

// typeData is a type's per-kind payload. The garbage collector must know which words of a record hold pointers, so
// every kind keeps its pointers in p0, p1 and p2 and its scalars in w0 and w1.
type typeData struct {
	p0, p1, p2 unsafe.Pointer
	w0, w1     uint32
}

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
	resolvedBaseConstraint *Type
	instantiations         map[CacheHashKey]*Type // Map of type instantiations
	mapper                 *TypeMapper            // Type mapper of a deferred type reference
	node                   ast.NodeRef            // TypeReferenceNode | ArrayTypeNode | TupleTypeNode of a deferred type reference
	resolvedDefaultType    *Type                  // Default of a type parameter
	distributedType        *Type                  // Distributed form of a type parameter
}

// structuredMembers holds the resolved members of an object, union or intersection type.
type structuredMembers struct {
	members    ast.SymbolTable
	properties []ast.Symbol
	extra      *structuredTypeExtra
}

type structuredTypeExtra struct {
	signatures         []*Signature // Signatures (call + construct)
	callSignatureCount int          // Count of call signatures
	indexInfos         []*IndexInfo

	objectTypeWithoutAbstractConstructSignatures *Type
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

func (t *Type) ensureExtra(c *Checker) *typeExtra {
	if t.extra == nil {
		t.extra = c.typeExtraArena.New()
	}
	return t.extra
}

func (t *Type) ensureResolved(c *Checker) *structuredMembers {
	if t.resolved == nil {
		// Allocated individually: in a shared chunk, one live record would keep its neighbours' member tables alive
		// after their types become garbage.
		t.resolved = new(structuredMembers)
	}
	return t.resolved
}

// fromPayload reads a pointer word of a type's payload.
func fromPayload[T any](p unsafe.Pointer) *T {
	// SAFETY: Each kind keeps pointers of a single type in each pointer word, and only the accessors of that kind,
	// which pair the word with that type, read or write it.
	return (*T)(p)
}

// toPayload converts a pointer for storage in a pointer word of a type's payload.
func toPayload[T any](v *T) unsafe.Pointer {
	// SAFETY: The word keeps the pointer visible to the garbage collector, and fromPayload converts it back to *T.
	return unsafe.Pointer(v)
}

// sliceFromPayload rebuilds a slice from a pointer word and a length word of a type's payload. It returns nil for a
// nil pointer, which keeps a nil slice distinct from an empty one.
func sliceFromPayload[T any](p unsafe.Pointer, n uint32) []T {
	if p == nil {
		return nil
	}
	// SAFETY: sliceToPayload stored the data pointer and length of a []T, and the length and capacity are equal.
	return unsafe.Slice(fromPayload[T](p), n)
}

// sliceToPayload splits a slice into a pointer word and a length word of a type's payload. The capacity is
// dropped, so the slice must not be appended to through the payload.
func sliceToPayload[T any](s []T) (unsafe.Pointer, uint32) {
	// SAFETY: The data pointer keeps the backing array alive, and sliceFromPayload rebuilds the slice from it.
	return toPayload(unsafe.SliceData(s)), uint32(len(s))
}

func typeSlice(p unsafe.Pointer, n uint32) []*Type {
	return sliceFromPayload[*Type](p, n)
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

func (t *Type) Distributed() []*Type {
	switch {
	case t.flags&TypeFlagsUnion != 0:
		return t.AsUnionType().types()
	case t.flags&TypeFlagsNever != 0:
		return nil
	}
	return []*Type{t}
}

// Common accessors

func (t *Type) Target() *Type {
	switch {
	case t.flags&TypeFlagsObject != 0:
		return t.AsObjectType().target()
	case t.flags&TypeFlagsTypeParameter != 0:
		return t.AsTypeParameter().target()
	case t.flags&TypeFlagsIndex != 0:
		return t.AsIndexType().target()
	case t.flags&TypeFlagsStringMapping != 0:
		return t.AsStringMappingType().target()
	}
	panic("Unhandled case in Type.Target")
}

func (t *Type) Mapper() *TypeMapper {
	switch {
	case t.flags&TypeFlagsObject != 0:
		return t.AsObjectType().mapper()
	case t.flags&TypeFlagsTypeParameter != 0:
		return t.AsTypeParameter().mapper()
	case t.flags&TypeFlagsConditional != 0:
		return t.AsConditionalType().cond().mapper
	}
	panic("Unhandled case in Type.Mapper")
}

func (t *Type) Types() []*Type {
	switch {
	case t.flags&TypeFlagsUnionOrIntersection != 0:
		return t.AsUnionOrIntersectionType().types()
	case t.flags&TypeFlagsTemplateLiteral != 0:
		return t.AsTemplateLiteralType().types()
	}
	panic("Unhandled case in Type.Types")
}

func (t *Type) TargetInterfaceType() *InterfaceType {
	return t.AsTypeReference().target().AsInterfaceType()
}

func (t *Type) TargetTupleType() *TupleType {
	return t.AsTypeReference().target().AsTupleType()
}

// SymbolOfType returns the symbol of a type, or the zero Symbol if it has none.
func (c *Checker) SymbolOfType(t *Type) ast.Symbol {
	return c.symbolOfType(t)
}

func (c *Checker) symbolOfType(t *Type) ast.Symbol {
	if t.symbol == 0 {
		return ast.Symbol{}
	}
	return c.symbolIds.symbol(t.symbol)
}

func (t *Type) setSymbol(c *Checker, symbol ast.Symbol) {
	t.symbol = c.symbolId(symbol)
}

func (t *Type) Alias() *TypeAlias {
	if t.extra == nil {
		return nil
	}
	return t.extra.alias
}

func (t *Type) setAlias(c *Checker, alias *TypeAlias) {
	if alias == nil && t.extra == nil {
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

func (t *Type) IsTupleType() bool {
	return isTupleType(t)
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

// IntrinsicType: the name in p0 and w0.

type IntrinsicType struct {
	TypeBase
}

func (t *IntrinsicType) IntrinsicName() string {
	// SAFETY: setIntrinsicName stored the data pointer and length of an immutable string.
	return unsafe.String(fromPayload[byte](t.data.p0), t.data.w0)
}

func (t *IntrinsicType) setIntrinsicName(name string) {
	// SAFETY: Strings are immutable, and the data pointer keeps the bytes alive.
	t.data.p0 = toPayload(unsafe.StringData(name))
	t.data.w0 = uint32(len(name))
}

// LiteralType: the value (string | jsnum.Number | bool | PseudoBigInt | nil for a computed enum) in p0 and p1, and in
// p2 the regular version of a fresh type or the fresh version of a regular type. Bit 0 of w0 marks a fresh type.

type LiteralType struct {
	TypeBase
}

const literalTypeFresh = 1

func (t *LiteralType) Value() any {
	// SAFETY: p0 and p1 are two adjacent pointer words, laid out like an interface value.
	return *(*any)(unsafe.Pointer(&t.data.p0))
}

func (t *LiteralType) setValue(value any) {
	// SAFETY: p0 and p1 are two adjacent pointer words, laid out like an interface value.
	*(*any)(unsafe.Pointer(&t.data.p0)) = value
}

// FreshType returns the fresh version of the type.
func (t *LiteralType) FreshType() *Type {
	if t.data.w0&literalTypeFresh != 0 {
		return &t.Type
	}
	return fromPayload[Type](t.data.p2)
}

// RegularType returns the regular version of the type.
func (t *LiteralType) RegularType() *Type {
	if t.data.w0&literalTypeFresh != 0 {
		return fromPayload[Type](t.data.p2)
	}
	return &t.Type
}

func (t *LiteralType) setFreshType(fresh *Type) {
	if fresh == &t.Type {
		t.data.w0 |= literalTypeFresh
	} else {
		t.data.p2 = toPayload(fresh)
	}
}

func (t *LiteralType) setRegularType(regular *Type) {
	if regular == &t.Type {
		t.data.w0 &^= literalTypeFresh
	} else {
		t.data.w0 |= literalTypeFresh
		t.data.p2 = toPayload(regular)
	}
}

func (t *LiteralType) String() string {
	return ValueToString(t.Value())
}

// UniqueESSymbolType: the name in w0 and w1.

type UniqueESSymbolType struct {
	TypeBase
}

func (t *UniqueESSymbolType) name() ast.SymbolName {
	return ast.SymbolName(uint64(t.data.w1)<<32 | uint64(t.data.w0))
}

func (t *UniqueESSymbolType) setName(name ast.SymbolName) {
	t.data.w0 = uint32(name)
	t.data.w1 = uint32(name >> 32)
}

// ConstrainedType (type with computed base constraint)

type ConstrainedType struct {
	TypeBase
}

func (t *ConstrainedType) AsConstrainedType() *ConstrainedType { return t }

func (t *ConstrainedType) resolvedBaseConstraint() *Type {
	if t.extra == nil {
		return nil
	}
	return t.extra.resolvedBaseConstraint
}

func (t *ConstrainedType) setResolvedBaseConstraint(c *Checker, constraint *Type) {
	if constraint == nil && t.extra == nil {
		return
	}
	t.ensureExtra(c).resolvedBaseConstraint = constraint
}

// StructuredType (base of all types with members)

type StructuredType struct {
	ConstrainedType
}

func (t *StructuredType) AsStructuredType() *StructuredType { return t }

func (t *StructuredType) members() ast.SymbolTable {
	if t.resolved == nil {
		return nil
	}
	return t.resolved.members
}

func (t *StructuredType) setMembers(c *Checker, members ast.SymbolTable) {
	if members == nil && t.resolved == nil {
		return
	}
	t.ensureResolved(c).members = members
}

func (t *StructuredType) properties() []ast.Symbol {
	if t.resolved == nil {
		return nil
	}
	return t.resolved.properties
}

func (t *StructuredType) setProperties(c *Checker, properties []ast.Symbol) {
	if properties == nil && t.resolved == nil {
		return
	}
	t.ensureResolved(c).properties = properties
}

func (t *StructuredType) resolvedExtra() *structuredTypeExtra {
	if t.resolved == nil {
		return nil
	}
	return t.resolved.extra
}

func (t *StructuredType) setResolvedExtra(c *Checker, extra *structuredTypeExtra) {
	if extra == nil && t.resolved == nil {
		return
	}
	t.ensureResolved(c).extra = extra
}

func (t *StructuredType) ensureResolvedExtra(c *Checker) *structuredTypeExtra {
	resolved := t.ensureResolved(c)
	if resolved.extra == nil {
		resolved.extra = c.structuredTypeExtraArena.New()
	}
	return resolved.extra
}

func (t *StructuredType) CallSignatures() []*Signature {
	extra := t.resolvedExtra()
	if extra == nil {
		return nil
	}
	return slices.Clip(extra.signatures[:extra.callSignatureCount])
}

func (t *StructuredType) ConstructSignatures() []*Signature {
	extra := t.resolvedExtra()
	if extra == nil {
		return nil
	}
	return slices.Clip(extra.signatures[extra.callSignatureCount:])
}

func (t *StructuredType) getMember(c *Checker, name ast.SymbolName) ast.Symbol {
	if members := t.members(); members != nil {
		return members.Get(name)
	}
	lazy := t.getLazyState()
	if lazy == nil {
		return ast.Symbol{}
	}
	index := lazy.layout.members.Find(name)
	if index < 0 {
		return ast.Symbol{}
	}
	return t.getPropertyAt(c, index)
}

func (t *StructuredType) getMembers(c *Checker) ast.SymbolTable {
	if t.members() == nil {
		if properties := t.getProperties(c); len(properties) != 0 {
			members := ast.NewSymbolTableWithCapacity(len(properties))
			for _, member := range properties {
				members.Set(member.Name(), member)
			}
			t.setMembers(c, members)
		}
	}
	return t.members()
}

func (t *StructuredType) getPropertyAt(c *Checker, index int) ast.Symbol {
	lazy := t.getLazyState()
	if lazy == nil {
		return t.properties()[index]
	}
	if t.properties() == nil {
		t.setProperties(c, make([]ast.Symbol, lazy.layout.members.Len()))
	}
	properties := t.properties()
	member := properties[index]
	if member.IsNil() {
		symbol := lazy.layout.members.SymbolAt(index)
		baseIndex := lazy.layout.baseIndices[index]
		if baseIndex < 0 {
			member = c.instantiateSymbol(symbol, lazy.mapper)
		} else {
			member = c.getPropertyOfType(lazy.baseTypes[baseIndex], symbol.Name())
		}
		properties[index] = member
	}
	return member
}

func (t *StructuredType) getProperties(c *Checker) []ast.Symbol {
	if lazy := t.getLazyState(); lazy != nil && !lazy.propertiesResolved {
		for i := range lazy.layout.members.Len() {
			t.getPropertyAt(c, i)
		}
		lazy.propertiesResolved = true
	}
	return t.properties()
}

func (t *StructuredType) hasProperties() bool {
	if len(t.properties()) != 0 {
		return true
	}
	lazy := t.getLazyState()
	if lazy == nil {
		return false
	}
	return lazy.layout.members.Len() != 0
}

func (t *StructuredType) getLazyState() *LazyState {
	if t.objectFlags&(ObjectFlagsClassOrInterface|ObjectFlagsReference) != 0 {
		return t.AsType().AsTypeReference().lazy()
	}
	return nil
}

func (t *StructuredType) getSignatures() []*Signature {
	extra := t.resolvedExtra()
	if extra == nil {
		return nil
	}
	return extra.signatures
}

func (t *StructuredType) getIndexInfos() []*IndexInfo {
	extra := t.resolvedExtra()
	if extra == nil {
		return nil
	}
	return extra.indexInfos
}

func (t *StructuredType) Properties(c *Checker) []ast.Symbol {
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

// ObjectType: the target of an instantiated type in p0, except for reverse mapped and evolving array types, and the
// mapper of an anonymous, mapped or instantiation expression type in p1. A deferred type reference keeps its mapper
// in extra.

type ObjectType struct {
	StructuredType
}

func (t *ObjectType) AsObjectType() *ObjectType { return t }

// Target of instantiated type
func (t *ObjectType) target() *Type {
	if t.kind() >= typeKindReverseMapped {
		return nil
	}
	return fromPayload[Type](t.data.p0)
}

func (t *ObjectType) setTarget(target *Type) {
	if t.kind() >= typeKindReverseMapped {
		panic("Type has no target")
	}
	t.data.p0 = toPayload(target)
}

// Type mapper for instantiated type
func (t *ObjectType) mapper() *TypeMapper {
	switch t.kind() {
	case typeKindObject, typeKindInstantiationExpression, typeKindMapped:
		return fromPayload[TypeMapper](t.data.p1)
	case typeKindTypeReference:
		if t.extra != nil {
			return t.extra.mapper
		}
	}
	return nil
}

func (t *ObjectType) setMapper(c *Checker, mapper *TypeMapper) {
	switch t.kind() {
	case typeKindObject, typeKindInstantiationExpression, typeKindMapped:
		t.data.p1 = toPayload(mapper)
	case typeKindTypeReference:
		if mapper != nil || t.extra != nil {
			t.ensureExtra(c).mapper = mapper
		}
	default:
		if mapper != nil {
			panic("Type has no mapper")
		}
	}
}

// Map of type instantiations
func (t *ObjectType) instantiations() map[CacheHashKey]*Type {
	if t.extra == nil {
		return nil
	}
	return t.extra.instantiations
}

func (t *ObjectType) setInstantiations(c *Checker, instantiations map[CacheHashKey]*Type) {
	if instantiations == nil && t.extra == nil {
		return
	}
	t.ensureExtra(c).instantiations = instantiations
}

// TypeReference (instantiation of an InterfaceType): the target in p0, the type arguments in p1 and w0, and the lazy
// state in p2. Interface and tuple types keep the lazy state in their own record in p2.

type TypeReference struct {
	ObjectType
}

type LazyState struct {
	layout             *MemberLayout
	mapper             *TypeMapper
	baseTypes          []*Type
	propertiesResolved bool
}

type MemberLayout struct {
	id               uint32          // Unique identifier for the member layout
	members          ast.SymbolTable // MemberInfos keyed by property name
	baseIndices      []int32         // Indices of base types for each member (-1 if locally declared)
	layoutProperties []ast.Symbol    // Lazily created list of layout properties (the symbols in members)
}

func (t *TypeReference) AsTypeReference() *TypeReference { return t }

// TypeReferenceNode | ArrayTypeNode | TupleTypeNode when deferred, else the zero NodeRef
func (t *TypeReference) node() ast.NodeRef {
	if t.extra == nil {
		return ast.NodeRef{}
	}
	return t.extra.node
}

func (t *TypeReference) setNode(c *Checker, node ast.NodeRef) {
	if node == (ast.NodeRef{}) && t.extra == nil {
		return
	}
	t.ensureExtra(c).node = node
}

func (t *TypeReference) resolvedTypeArguments() []*Type {
	return typeSlice(t.data.p1, t.data.w0)
}

func (t *TypeReference) setResolvedTypeArguments(typeArguments []*Type) {
	t.data.p1, t.data.w0 = sliceToPayload(typeArguments)
}

// Resolved member layout and instantiation state
func (t *TypeReference) lazy() *LazyState {
	if t.kind() == typeKindTypeReference {
		return fromPayload[LazyState](t.data.p2)
	}
	return fromPayload[interfaceData](t.data.p2).lazy
}

func (t *TypeReference) setLazy(lazy *LazyState) {
	if t.kind() == typeKindTypeReference {
		t.data.p2 = toPayload(lazy)
	} else {
		fromPayload[interfaceData](t.data.p2).lazy = lazy
	}
}

// InterfaceType (when generic, serves as reference to instantiation of itself): its own record in p2.

type InterfaceType struct {
	TypeReference
}

type interfaceData struct {
	lazy                        *LazyState
	allTypeParameters           []*Type // Type parameters (outer + local + thisType)
	outerTypeParameterCount     int     // Count of outer type parameters
	thisType                    *Type   // The "this" type (nil if none)
	baseTypesResolved           bool
	declaredMembersResolved     bool
	resolvedBaseConstructorType *Type
	resolvedBaseTypes           []*Type
	declaredMembers             ast.SymbolTable // Declared members
	declaredCallSignatures      []*Signature    // Declared call signatures
	declaredConstructSignatures []*Signature    // Declared construct signatures
	declaredIndexInfos          []*IndexInfo    // Declared index signatures
}

func (t *InterfaceType) AsInterfaceType() *InterfaceType { return t }

func (t *InterfaceType) iface() *interfaceData {
	return fromPayload[interfaceData](t.data.p2)
}

func (t *InterfaceType) OuterTypeParameters() []*Type {
	d := t.iface()
	if len(d.allTypeParameters) == 0 {
		return nil
	}
	return slices.Clip(d.allTypeParameters[:d.outerTypeParameterCount])
}

func (t *InterfaceType) LocalTypeParameters() []*Type {
	d := t.iface()
	if len(d.allTypeParameters) == 0 {
		return nil
	}
	return slices.Clip(d.allTypeParameters[d.outerTypeParameterCount : len(d.allTypeParameters)-1])
}

func (t *InterfaceType) TypeParameters() []*Type {
	d := t.iface()
	if len(d.allTypeParameters) == 0 {
		return nil
	}
	return slices.Clip(d.allTypeParameters[:len(d.allTypeParameters)-1])
}

func (t *InterfaceType) ThisType() *Type {
	return t.iface().thisType
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

// TupleType: like InterfaceType, with a larger record in p2.

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

func (t *TupleType) tuple() *tupleData {
	return fromPayload[tupleData](t.data.p2)
}

func (t *TupleType) FixedLength() int { return t.tuple().fixedLength }
func (t *TupleType) IsReadonly() bool { return t.tuple().readonly }
func (t *TupleType) ElementFlags() []ElementFlags {
	elementInfos := t.tuple().elementInfos
	elementFlags := make([]ElementFlags, len(elementInfos))
	for i, info := range elementInfos {
		elementFlags[i] = info.flags
	}
	return elementFlags
}
func (t *TupleType) ElementInfos() []TupleElementInfo { return t.tuple().elementInfos }

// InstantiationExpressionType: the node in w0 and w1.

type InstantiationExpressionType struct {
	ObjectType
}

func (t *InstantiationExpressionType) node() ast.NodeRef {
	// SAFETY: w0 and w1 are two adjacent 32-bit scalar words, laid out like a NodeRef.
	return *(*ast.NodeRef)(unsafe.Pointer(&t.data.w0))
}

func (t *InstantiationExpressionType) setNode(node ast.NodeRef) {
	// SAFETY: w0 and w1 are two adjacent 32-bit scalar words, laid out like a NodeRef.
	*(*ast.NodeRef)(unsafe.Pointer(&t.data.w0)) = node
}

// MappedType: its own record in p2.

type MappedType struct {
	ObjectType
}

type mappedData struct {
	declaration          ast.MappedTypeNode
	typeParameter        *Type
	constraintType       *Type
	nameType             *Type
	templateType         *Type
	modifiersType        *Type
	resolvedApparentType *Type
	containsError        bool
}

func (t *MappedType) mapped() *mappedData {
	return fromPayload[mappedData](t.data.p2)
}

func (t *MappedType) TypeParameter() *Type  { return t.mapped().typeParameter }
func (t *MappedType) ConstraintType() *Type { return t.mapped().constraintType }
func (t *MappedType) NameType() *Type       { return t.mapped().nameType }
func (t *MappedType) TemplateType() *Type   { return t.mapped().templateType }
func (t *MappedType) ResolveComponents(c *Checker, typ *Type) {
	c.getTypeParameterFromMappedType(typ)
	c.getConstraintTypeFromMappedType(typ)
	c.getNameTypeFromMappedType(typ)
	c.getTemplateTypeFromMappedType(typ)
}

// ReverseMappedType: the source, mapped type and constraint type in p0, p1 and p2.

type ReverseMappedType struct {
	ObjectType
}

func (t *ReverseMappedType) source() *Type             { return fromPayload[Type](t.data.p0) }
func (t *ReverseMappedType) setSource(v *Type)         { t.data.p0 = toPayload(v) }
func (t *ReverseMappedType) mappedType() *Type         { return fromPayload[Type](t.data.p1) }
func (t *ReverseMappedType) setMappedType(v *Type)     { t.data.p1 = toPayload(v) }
func (t *ReverseMappedType) constraintType() *Type     { return fromPayload[Type](t.data.p2) }
func (t *ReverseMappedType) setConstraintType(v *Type) { t.data.p2 = toPayload(v) }

// EvolvingArrayType: the element type and final array type in p0 and p1.

type EvolvingArrayType struct {
	ObjectType
}

func (t *EvolvingArrayType) elementType() *Type        { return fromPayload[Type](t.data.p0) }
func (t *EvolvingArrayType) setElementType(v *Type)    { t.data.p0 = toPayload(v) }
func (t *EvolvingArrayType) finalArrayType() *Type     { return fromPayload[Type](t.data.p1) }
func (t *EvolvingArrayType) setFinalArrayType(v *Type) { t.data.p1 = toPayload(v) }

// UnionOrIntersectionType: the types in p0 and w0, and in p2 a record of rarer fields allocated on first use.

type UnionOrIntersectionType struct {
	StructuredType
}

type unionOrIntersectionData struct {
	propertyCache                               ast.SymbolTable // Augmented entries, except complete ones already in the non-augmented cache.
	propertyCacheWithoutFunctionPropertyAugment ast.SymbolTable
	resolvedProperties                          []ast.Symbol
}

func (t *UnionOrIntersectionType) AsUnionOrIntersectionType() *UnionOrIntersectionType { return t }

func (t *UnionOrIntersectionType) types() []*Type {
	return typeSlice(t.data.p0, t.data.w0)
}

func (t *UnionOrIntersectionType) setTypes(types []*Type) {
	t.data.p0, t.data.w0 = sliceToPayload(types)
}

func (t *UnionOrIntersectionType) Types() []*Type {
	return t.types()
}

func (t *UnionOrIntersectionType) more() *unionOrIntersectionData {
	return fromPayload[unionOrIntersectionData](t.data.p2)
}

func (t *UnionOrIntersectionType) ensureMore(c *Checker) *unionOrIntersectionData {
	if t.data.p2 == nil {
		if t.kind() == typeKindUnion {
			t.data.p2 = toPayload(c.unionDataArena.New())
		} else {
			t.data.p2 = toPayload(c.intersectionDataArena.New())
		}
	}
	return t.more()
}

func (t *UnionOrIntersectionType) propertyCache() ast.SymbolTable {
	if more := t.more(); more != nil {
		return more.propertyCache
	}
	return nil
}

func (t *UnionOrIntersectionType) setPropertyCache(c *Checker, v ast.SymbolTable) {
	t.ensureMore(c).propertyCache = v
}

func (t *UnionOrIntersectionType) propertyCacheWithoutFunctionPropertyAugment() ast.SymbolTable {
	if more := t.more(); more != nil {
		return more.propertyCacheWithoutFunctionPropertyAugment
	}
	return nil
}

func (t *UnionOrIntersectionType) setPropertyCacheWithoutFunctionPropertyAugment(c *Checker, v ast.SymbolTable) {
	t.ensureMore(c).propertyCacheWithoutFunctionPropertyAugment = v
}

func (t *UnionOrIntersectionType) resolvedProperties() []ast.Symbol {
	if more := t.more(); more != nil {
		return more.resolvedProperties
	}
	return nil
}

func (t *UnionOrIntersectionType) setResolvedProperties(c *Checker, v []ast.Symbol) {
	t.ensureMore(c).resolvedProperties = v
}

// UnionType: the origin in p1.

type UnionType struct {
	UnionOrIntersectionType
}

type unionData struct {
	unionOrIntersectionData
	resolvedReducedType *Type
	regularType         *Type
	keyPropertyName     ast.SymbolName  // Property with unique unit type that exists in every object/intersection in union type
	constituentMap      map[*Type]*Type // Constituents keyed by unit type discriminants
}

func (t *UnionType) union() *unionData {
	return fromPayload[unionData](t.data.p2)
}

// Denormalized union, intersection, or index type in which union originates
func (t *UnionType) origin() *Type     { return fromPayload[Type](t.data.p1) }
func (t *UnionType) setOrigin(v *Type) { t.data.p1 = toPayload(v) }

func (t *UnionType) resolvedReducedType() *Type {
	if d := t.union(); d != nil {
		return d.resolvedReducedType
	}
	return nil
}

func (t *UnionType) setResolvedReducedType(c *Checker, v *Type) {
	t.ensureMore(c)
	t.union().resolvedReducedType = v
}

func (t *UnionType) regularType() *Type {
	if d := t.union(); d != nil {
		return d.regularType
	}
	return nil
}

func (t *UnionType) setRegularType(c *Checker, v *Type) {
	t.ensureMore(c)
	t.union().regularType = v
}

func (t *UnionType) keyPropertyName() ast.SymbolName {
	if d := t.union(); d != nil {
		return d.keyPropertyName
	}
	return ast.SymbolName(0)
}

func (t *UnionType) setKeyPropertyName(c *Checker, v ast.SymbolName) {
	t.ensureMore(c)
	t.union().keyPropertyName = v
}

func (t *UnionType) constituentMap() map[*Type]*Type {
	if d := t.union(); d != nil {
		return d.constituentMap
	}
	return nil
}

func (t *UnionType) setConstituentMap(c *Checker, v map[*Type]*Type) {
	t.ensureMore(c)
	t.union().constituentMap = v
}

// IntersectionType: the resolved apparent type in p1.

type IntersectionType struct {
	UnionOrIntersectionType
}

type intersectionData struct {
	unionOrIntersectionData
	uniqueLiteralFilledInstantiation *Type // Instantiation with type parameters mapped to never type
}

func (t *IntersectionType) resolvedApparentType() *Type     { return fromPayload[Type](t.data.p1) }
func (t *IntersectionType) setResolvedApparentType(v *Type) { t.data.p1 = toPayload(v) }

func (t *IntersectionType) uniqueLiteralFilledInstantiation() *Type {
	if d := fromPayload[intersectionData](t.data.p2); d != nil {
		return d.uniqueLiteralFilledInstantiation
	}
	return nil
}

func (t *IntersectionType) setUniqueLiteralFilledInstantiation(c *Checker, v *Type) {
	t.ensureMore(c)
	fromPayload[intersectionData](t.data.p2).uniqueLiteralFilledInstantiation = v
}

// TypeParameter: the constraint, target and mapper in p0, p1 and p2, and flags in w0. The default and distributed
// type are in extra.

type TypeParameter struct {
	ConstrainedType
}

const (
	typeParameterIsThisType = 1 << iota
	typeParameterIsDistributed
)

func (t *TypeParameter) constraint() *Type       { return fromPayload[Type](t.data.p0) }
func (t *TypeParameter) setConstraint(v *Type)   { t.data.p0 = toPayload(v) }
func (t *TypeParameter) target() *Type           { return fromPayload[Type](t.data.p1) }
func (t *TypeParameter) setTarget(v *Type)       { t.data.p1 = toPayload(v) }
func (t *TypeParameter) mapper() *TypeMapper     { return fromPayload[TypeMapper](t.data.p2) }
func (t *TypeParameter) setMapper(v *TypeMapper) { t.data.p2 = toPayload(v) }
func (t *TypeParameter) IsThisType() bool        { return t.data.w0&typeParameterIsThisType != 0 }
func (t *TypeParameter) isDistributed() bool     { return t.data.w0&typeParameterIsDistributed != 0 }
func (t *TypeParameter) setIsThisType(v bool)    { t.setFlag(typeParameterIsThisType, v) }
func (t *TypeParameter) setIsDistributed(v bool) { t.setFlag(typeParameterIsDistributed, v) }

func (t *TypeParameter) setFlag(flag uint32, v bool) {
	if v {
		t.data.w0 |= flag
	} else {
		t.data.w0 &^= flag
	}
}

func (t *TypeParameter) resolvedDefaultType() *Type {
	if t.extra == nil {
		return nil
	}
	return t.extra.resolvedDefaultType
}

func (t *TypeParameter) setResolvedDefaultType(c *Checker, v *Type) {
	if v == nil && t.extra == nil {
		return
	}
	t.ensureExtra(c).resolvedDefaultType = v
}

func (t *TypeParameter) distributedType() *Type {
	if t.extra == nil {
		return nil
	}
	return t.extra.distributedType
}

func (t *TypeParameter) setDistributedType(c *Checker, v *Type) {
	if v == nil && t.extra == nil {
		return
	}
	t.ensureExtra(c).distributedType = v
}

// IndexFlags

type IndexFlags uint32

const (
	IndexFlagsNone              IndexFlags = 0
	IndexFlagsStringsOnly       IndexFlags = 1 << 0
	IndexFlagsNoIndexSignatures IndexFlags = 1 << 1
	IndexFlagsNoReducibleCheck  IndexFlags = 1 << 2
)

// IndexType: the target in p0 and the flags in w0.

type IndexType struct {
	ConstrainedType
}

func (t *IndexType) Target() *Type              { return t.target() }
func (t *IndexType) target() *Type              { return fromPayload[Type](t.data.p0) }
func (t *IndexType) setTarget(v *Type)          { t.data.p0 = toPayload(v) }
func (t *IndexType) indexFlags() IndexFlags     { return IndexFlags(t.data.w0) }
func (t *IndexType) setIndexFlags(v IndexFlags) { t.data.w0 = uint32(v) }

// IndexedAccessType: the object and index types in p0 and p1, and the access flags (only AccessFlags.Persistent) in w0.

type IndexedAccessType struct {
	ConstrainedType
}

func (t *IndexedAccessType) ObjectType() *Type            { return t.objectType() }
func (t *IndexedAccessType) IndexType() *Type             { return t.indexType() }
func (t *IndexedAccessType) objectType() *Type            { return fromPayload[Type](t.data.p0) }
func (t *IndexedAccessType) setObjectType(v *Type)        { t.data.p0 = toPayload(v) }
func (t *IndexedAccessType) indexType() *Type             { return fromPayload[Type](t.data.p1) }
func (t *IndexedAccessType) setIndexType(v *Type)         { t.data.p1 = toPayload(v) }
func (t *IndexedAccessType) accessFlags() AccessFlags     { return AccessFlags(t.data.w0) }
func (t *IndexedAccessType) setAccessFlags(v AccessFlags) { t.data.w0 = uint32(v) }

// TemplateLiteralType: the texts in p0 and w0 (always one element longer than the types), the types (always at least
// one) in p1 and w1.

type TemplateLiteralType struct {
	ConstrainedType
}

func (t *TemplateLiteralType) Texts() []string { return t.texts() }
func (t *TemplateLiteralType) Types() []*Type  { return t.types() }

func (t *TemplateLiteralType) texts() []string {
	return sliceFromPayload[string](t.data.p0, t.data.w0)
}

func (t *TemplateLiteralType) setTexts(texts []string) {
	t.data.p0, t.data.w0 = sliceToPayload(texts)
}

func (t *TemplateLiteralType) types() []*Type {
	return typeSlice(t.data.p1, t.data.w1)
}

func (t *TemplateLiteralType) setTypes(types []*Type) {
	t.data.p1, t.data.w1 = sliceToPayload(types)
}

// StringMappingType: the target in p0.

type StringMappingType struct {
	ConstrainedType
}

func (t *StringMappingType) Target() *Type     { return t.target() }
func (t *StringMappingType) target() *Type     { return fromPayload[Type](t.data.p0) }
func (t *StringMappingType) setTarget(v *Type) { t.data.p0 = toPayload(v) }

// SubstitutionType: the base type (the target type) and the constraint it is known to satisfy in p0 and p1.

type SubstitutionType struct {
	ConstrainedType
}

func (t *SubstitutionType) BaseType() *Type        { return t.baseType() }
func (t *SubstitutionType) SubstConstraint() *Type { return t.constraint() }
func (t *SubstitutionType) baseType() *Type        { return fromPayload[Type](t.data.p0) }
func (t *SubstitutionType) setBaseType(v *Type)    { t.data.p0 = toPayload(v) }
func (t *SubstitutionType) constraint() *Type      { return fromPayload[Type](t.data.p1) }
func (t *SubstitutionType) setConstraint(v *Type)  { t.data.p1 = toPayload(v) }

type ConditionalRoot struct {
	node                ast.ConditionalTypeNode
	checkType           *Type
	extendsType         *Type
	isDistributive      bool
	inferTypeParameters []*Type
	outerTypeParameters []*Type
	instantiations      map[CacheHashKey]*Type
	alias               *TypeAlias
}

// ConditionalType: its own record in p0, and the check and extends types in p1 and p2.

type ConditionalType struct {
	ConstrainedType
}

type conditionalData struct {
	root                             *ConditionalRoot
	resolvedTrueType                 *Type
	resolvedFalseType                *Type
	resolvedInferredTrueType         *Type // The `trueType` instantiated with the `combinedMapper`, if present
	resolvedDefaultConstraint        *Type
	resolvedConstraintOfDistributive *Type
	mapper                           *TypeMapper
	combinedMapper                   *TypeMapper
}

func (t *ConditionalType) cond() *conditionalData { return fromPayload[conditionalData](t.data.p0) }
func (t *ConditionalType) CheckType() *Type       { return t.checkType() }
func (t *ConditionalType) ExtendsType() *Type     { return t.extendsType() }
func (t *ConditionalType) checkType() *Type       { return fromPayload[Type](t.data.p1) }
func (t *ConditionalType) setCheckType(v *Type)   { t.data.p1 = toPayload(v) }
func (t *ConditionalType) extendsType() *Type     { return fromPayload[Type](t.data.p2) }
func (t *ConditionalType) setExtendsType(v *Type) { t.data.p2 = toPayload(v) }

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
	parameters               []ast.Symbol
	resolvedReturnType       *Type
	target                   *Signature
	mapper                   *TypeMapper
	extra                    *signatureExtra
}

type signatureExtra struct {
	typeParameters        []*Type
	thisParameter         ast.Symbol
	resolvedTypePredicate *TypePredicate
	isolatedSignatureType *Type
	composite             *CompositeSignature
}

func (s *Signature) ensureExtra(c *Checker) *signatureExtra {
	if s.extra == nil {
		s.extra = c.signatureExtraArena.New()
	}
	return s.extra
}

func (s *Signature) setTypeParameters(c *Checker, types []*Type) {
	if types != nil || s.extra != nil {
		s.ensureExtra(c).typeParameters = types
	}
}

func (s *Signature) setThisParameter(c *Checker, parameter ast.Symbol) {
	if !parameter.IsNil() || s.extra != nil {
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

func (s *Signature) getIsolatedSignatureType() *Type {
	if s.extra == nil {
		return nil
	}
	return s.extra.isolatedSignatureType
}

func (s *Signature) Id() SignatureId {
	return s.id
}

func (s *Signature) Flags() SignatureFlags {
	return s.flags &^ signatureFlagsNoTypePredicate
}

func (s *Signature) TypeParameters() []*Type {
	if s.extra == nil {
		return nil
	}
	return s.extra.typeParameters
}

func (s *Signature) Target() *Signature {
	return s.target
}

func (s *Signature) ThisParameter() ast.Symbol {
	if s.extra == nil {
		return ast.Symbol{}
	}
	return s.extra.thisParameter
}

func (s *Signature) Parameters() []ast.Symbol {
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
	t              *Type
}

func (typePredicate *TypePredicate) Type() *Type {
	return typePredicate.t
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

type IndexInfo struct {
	keyType     *Type
	valueType   *Type
	isReadonly  bool
	declaration ast.Node   // IndexSignatureDeclaration
	indexSymbol ast.Symbol // Synthetic property symbol for this index signature
	components  []ast.Node // ElementWithComputedPropertyName
}

func (info *IndexInfo) KeyType() *Type {
	return info.keyType
}

func (info *IndexInfo) ValueType() *Type {
	return info.valueType
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
