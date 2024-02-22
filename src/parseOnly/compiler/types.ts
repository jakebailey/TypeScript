import {
    OptionsNameMap,
} from "./commandLineParser";
import {
    Pattern,
} from "./core";
import {
    MapLike,
} from "./corePublic";

// branded string type used to store absolute, normalized and canonicalized paths
// arbitrary file name can be converted to Path via toPath function
export type Path = string & { __pathBrand: any; };

/** @internal */
export type MatchingKeys<TRecord, TMatch, K extends keyof TRecord = keyof TRecord> = K extends (TRecord[K] extends TMatch ? K : never) ? K : never;

export interface TextRange {
    pos: number;
    end: number;
}

export interface ReadonlyTextRange {
    readonly pos: number;
    readonly end: number;
}

// token > SyntaxKind.Identifier => token is a keyword
// Also, If you add a new SyntaxKind be sure to keep the `Markers` section at the bottom in sync
export const enum SyntaxKind {
    Unknown,
    EndOfFileToken,
    SingleLineCommentTrivia,
    MultiLineCommentTrivia,
    NewLineTrivia,
    WhitespaceTrivia,
    // We detect and preserve #! on the first line
    ShebangTrivia,
    // We detect and provide better error recovery when we encounter a git merge marker.  This
    // allows us to edit files with git-conflict markers in them in a much more pleasant manner.
    ConflictMarkerTrivia,
    // If a file is actually binary, with any luck, we'll get U+FFFD REPLACEMENT CHARACTER
    // in position zero and can just skip what is surely a doomed parse.
    NonTextFileMarkerTrivia,
    // Literals
    NumericLiteral,
    BigIntLiteral,
    StringLiteral,
    JsxText,
    JsxTextAllWhiteSpaces,
    RegularExpressionLiteral,
    NoSubstitutionTemplateLiteral,
    // Pseudo-literals
    TemplateHead,
    TemplateMiddle,
    TemplateTail,
    // Punctuation
    OpenBraceToken,
    CloseBraceToken,
    OpenParenToken,
    CloseParenToken,
    OpenBracketToken,
    CloseBracketToken,
    DotToken,
    DotDotDotToken,
    SemicolonToken,
    CommaToken,
    QuestionDotToken,
    LessThanToken,
    LessThanSlashToken,
    GreaterThanToken,
    LessThanEqualsToken,
    GreaterThanEqualsToken,
    EqualsEqualsToken,
    ExclamationEqualsToken,
    EqualsEqualsEqualsToken,
    ExclamationEqualsEqualsToken,
    EqualsGreaterThanToken,
    PlusToken,
    MinusToken,
    AsteriskToken,
    AsteriskAsteriskToken,
    SlashToken,
    PercentToken,
    PlusPlusToken,
    MinusMinusToken,
    LessThanLessThanToken,
    GreaterThanGreaterThanToken,
    GreaterThanGreaterThanGreaterThanToken,
    AmpersandToken,
    BarToken,
    CaretToken,
    ExclamationToken,
    TildeToken,
    AmpersandAmpersandToken,
    BarBarToken,
    QuestionToken,
    ColonToken,
    AtToken,
    QuestionQuestionToken,
    /** Only the JSDoc scanner produces BacktickToken. The normal scanner produces NoSubstitutionTemplateLiteral and related kinds. */
    BacktickToken,
    /** Only the JSDoc scanner produces HashToken. The normal scanner produces PrivateIdentifier. */
    HashToken,
    // Assignments
    EqualsToken,
    PlusEqualsToken,
    MinusEqualsToken,
    AsteriskEqualsToken,
    AsteriskAsteriskEqualsToken,
    SlashEqualsToken,
    PercentEqualsToken,
    LessThanLessThanEqualsToken,
    GreaterThanGreaterThanEqualsToken,
    GreaterThanGreaterThanGreaterThanEqualsToken,
    AmpersandEqualsToken,
    BarEqualsToken,
    BarBarEqualsToken,
    AmpersandAmpersandEqualsToken,
    QuestionQuestionEqualsToken,
    CaretEqualsToken,
    // Identifiers and PrivateIdentifiers
    Identifier,
    PrivateIdentifier,
    /**
     * Only the special JSDoc comment text scanner produces JSDocCommentTextTokes. One of these tokens spans all text after a tag comment's start and before the next @
     * @internal
     */
    JSDocCommentTextToken,
    // Reserved words
    BreakKeyword,
    CaseKeyword,
    CatchKeyword,
    ClassKeyword,
    ConstKeyword,
    ContinueKeyword,
    DebuggerKeyword,
    DefaultKeyword,
    DeleteKeyword,
    DoKeyword,
    ElseKeyword,
    EnumKeyword,
    ExportKeyword,
    ExtendsKeyword,
    FalseKeyword,
    FinallyKeyword,
    ForKeyword,
    FunctionKeyword,
    IfKeyword,
    ImportKeyword,
    InKeyword,
    InstanceOfKeyword,
    NewKeyword,
    NullKeyword,
    ReturnKeyword,
    SuperKeyword,
    SwitchKeyword,
    ThisKeyword,
    ThrowKeyword,
    TrueKeyword,
    TryKeyword,
    TypeOfKeyword,
    VarKeyword,
    VoidKeyword,
    WhileKeyword,
    WithKeyword,
    // Strict mode reserved words
    ImplementsKeyword,
    InterfaceKeyword,
    LetKeyword,
    PackageKeyword,
    PrivateKeyword,
    ProtectedKeyword,
    PublicKeyword,
    StaticKeyword,
    YieldKeyword,
    // Contextual keywords
    AbstractKeyword,
    AccessorKeyword,
    AsKeyword,
    AssertsKeyword,
    AssertKeyword,
    AnyKeyword,
    AsyncKeyword,
    AwaitKeyword,
    BooleanKeyword,
    ConstructorKeyword,
    DeclareKeyword,
    GetKeyword,
    InferKeyword,
    IntrinsicKeyword,
    IsKeyword,
    KeyOfKeyword,
    ModuleKeyword,
    NamespaceKeyword,
    NeverKeyword,
    OutKeyword,
    ReadonlyKeyword,
    RequireKeyword,
    NumberKeyword,
    ObjectKeyword,
    SatisfiesKeyword,
    SetKeyword,
    StringKeyword,
    SymbolKeyword,
    TypeKeyword,
    UndefinedKeyword,
    UniqueKeyword,
    UnknownKeyword,
    UsingKeyword,
    FromKeyword,
    GlobalKeyword,
    BigIntKeyword,
    OverrideKeyword,
    OfKeyword, // LastKeyword and LastToken and LastContextualKeyword

    // Parse tree nodes

    // Names
    QualifiedName,
    ComputedPropertyName,
    // Signature elements
    TypeParameter,
    Parameter,
    Decorator,
    // TypeMember
    PropertySignature,
    PropertyDeclaration,
    MethodSignature,
    MethodDeclaration,
    ClassStaticBlockDeclaration,
    Constructor,
    GetAccessor,
    SetAccessor,
    CallSignature,
    ConstructSignature,
    IndexSignature,
    // Type
    TypePredicate,
    TypeReference,
    FunctionType,
    ConstructorType,
    TypeQuery,
    TypeLiteral,
    ArrayType,
    TupleType,
    OptionalType,
    RestType,
    UnionType,
    IntersectionType,
    ConditionalType,
    InferType,
    ParenthesizedType,
    ThisType,
    TypeOperator,
    IndexedAccessType,
    MappedType,
    LiteralType,
    NamedTupleMember,
    TemplateLiteralType,
    TemplateLiteralTypeSpan,
    ImportType,
    // Binding patterns
    ObjectBindingPattern,
    ArrayBindingPattern,
    BindingElement,
    // Expression
    ArrayLiteralExpression,
    ObjectLiteralExpression,
    PropertyAccessExpression,
    ElementAccessExpression,
    CallExpression,
    NewExpression,
    TaggedTemplateExpression,
    TypeAssertionExpression,
    ParenthesizedExpression,
    FunctionExpression,
    ArrowFunction,
    DeleteExpression,
    TypeOfExpression,
    VoidExpression,
    AwaitExpression,
    PrefixUnaryExpression,
    PostfixUnaryExpression,
    BinaryExpression,
    ConditionalExpression,
    TemplateExpression,
    YieldExpression,
    SpreadElement,
    ClassExpression,
    OmittedExpression,
    ExpressionWithTypeArguments,
    AsExpression,
    NonNullExpression,
    MetaProperty,
    SyntheticExpression,
    SatisfiesExpression,

    // Misc
    TemplateSpan,
    SemicolonClassElement,
    // Element
    Block,
    EmptyStatement,
    VariableStatement,
    ExpressionStatement,
    IfStatement,
    DoStatement,
    WhileStatement,
    ForStatement,
    ForInStatement,
    ForOfStatement,
    ContinueStatement,
    BreakStatement,
    ReturnStatement,
    WithStatement,
    SwitchStatement,
    LabeledStatement,
    ThrowStatement,
    TryStatement,
    DebuggerStatement,
    VariableDeclaration,
    VariableDeclarationList,
    FunctionDeclaration,
    ClassDeclaration,
    InterfaceDeclaration,
    TypeAliasDeclaration,
    EnumDeclaration,
    ModuleDeclaration,
    ModuleBlock,
    CaseBlock,
    NamespaceExportDeclaration,
    ImportEqualsDeclaration,
    ImportDeclaration,
    ImportClause,
    NamespaceImport,
    NamedImports,
    ImportSpecifier,
    ExportAssignment,
    ExportDeclaration,
    NamedExports,
    NamespaceExport,
    ExportSpecifier,
    MissingDeclaration,

    // Module references
    ExternalModuleReference,

    // JSX
    JsxElement,
    JsxSelfClosingElement,
    JsxOpeningElement,
    JsxClosingElement,
    JsxFragment,
    JsxOpeningFragment,
    JsxClosingFragment,
    JsxAttribute,
    JsxAttributes,
    JsxSpreadAttribute,
    JsxExpression,
    JsxNamespacedName,

    // Clauses
    CaseClause,
    DefaultClause,
    HeritageClause,
    CatchClause,

    ImportAttributes,
    ImportAttribute,
    /** @deprecated */ AssertClause = ImportAttributes,
    /** @deprecated */ AssertEntry = ImportAttribute,
    /** @deprecated */ ImportTypeAssertionContainer,

    // Property assignments
    PropertyAssignment,
    ShorthandPropertyAssignment,
    SpreadAssignment,

    // Enum
    EnumMember,
    // Unparsed
    /** @deprecated */ UnparsedPrologue,
    /** @deprecated */ UnparsedPrepend,
    /** @deprecated */ UnparsedText,
    /** @deprecated */ UnparsedInternalText,
    /** @deprecated */ UnparsedSyntheticReference,

    // Top-level nodes
    SourceFile,
    Bundle,
    /** @deprecated */ UnparsedSource,
    /** @deprecated */ InputFiles,

    // JSDoc nodes
    JSDocTypeExpression,
    JSDocNameReference,
    JSDocMemberName, // C#p
    JSDocAllType, // The * type
    JSDocUnknownType, // The ? type
    JSDocNullableType,
    JSDocNonNullableType,
    JSDocOptionalType,
    JSDocFunctionType,
    JSDocVariadicType,
    JSDocNamepathType, // https://jsdoc.app/about-namepaths.html
    JSDoc,
    /** @deprecated Use SyntaxKind.JSDoc */
    JSDocComment = JSDoc,
    JSDocText,
    JSDocTypeLiteral,
    JSDocSignature,
    JSDocLink,
    JSDocLinkCode,
    JSDocLinkPlain,
    JSDocTag,
    JSDocAugmentsTag,
    JSDocImplementsTag,
    JSDocAuthorTag,
    JSDocDeprecatedTag,
    JSDocClassTag,
    JSDocPublicTag,
    JSDocPrivateTag,
    JSDocProtectedTag,
    JSDocReadonlyTag,
    JSDocOverrideTag,
    JSDocCallbackTag,
    JSDocOverloadTag,
    JSDocEnumTag,
    JSDocParameterTag,
    JSDocReturnTag,
    JSDocThisTag,
    JSDocTypeTag,
    JSDocTemplateTag,
    JSDocTypedefTag,
    JSDocSeeTag,
    JSDocPropertyTag,
    JSDocThrowsTag,
    JSDocSatisfiesTag,

    // Synthesized list
    SyntaxList,

    // Transformation nodes
    NotEmittedStatement,
    PartiallyEmittedExpression,
    CommaListExpression,
    SyntheticReferenceExpression,

    // Enum value count
    Count,

    // Markers
    FirstAssignment = EqualsToken,
    LastAssignment = CaretEqualsToken,
    FirstCompoundAssignment = PlusEqualsToken,
    LastCompoundAssignment = CaretEqualsToken,
    FirstReservedWord = BreakKeyword,
    LastReservedWord = WithKeyword,
    FirstKeyword = BreakKeyword,
    LastKeyword = OfKeyword,
    FirstFutureReservedWord = ImplementsKeyword,
    LastFutureReservedWord = YieldKeyword,
    FirstTypeNode = TypePredicate,
    LastTypeNode = ImportType,
    FirstPunctuation = OpenBraceToken,
    LastPunctuation = CaretEqualsToken,
    FirstToken = Unknown,
    LastToken = LastKeyword,
    FirstTriviaToken = SingleLineCommentTrivia,
    LastTriviaToken = ConflictMarkerTrivia,
    FirstLiteralToken = NumericLiteral,
    LastLiteralToken = NoSubstitutionTemplateLiteral,
    FirstTemplateToken = NoSubstitutionTemplateLiteral,
    LastTemplateToken = TemplateTail,
    FirstBinaryOperator = LessThanToken,
    LastBinaryOperator = CaretEqualsToken,
    FirstStatement = VariableStatement,
    LastStatement = DebuggerStatement,
    FirstNode = QualifiedName,
    FirstJSDocNode = JSDocTypeExpression,
    LastJSDocNode = JSDocSatisfiesTag,
    FirstJSDocTagNode = JSDocTag,
    LastJSDocTagNode = JSDocSatisfiesTag,
    /** @internal */ FirstContextualKeyword = AbstractKeyword,
    /** @internal */ LastContextualKeyword = OfKeyword,
}

export type PunctuationSyntaxKind =
    | SyntaxKind.OpenBraceToken
    | SyntaxKind.CloseBraceToken
    | SyntaxKind.OpenParenToken
    | SyntaxKind.CloseParenToken
    | SyntaxKind.OpenBracketToken
    | SyntaxKind.CloseBracketToken
    | SyntaxKind.DotToken
    | SyntaxKind.DotDotDotToken
    | SyntaxKind.SemicolonToken
    | SyntaxKind.CommaToken
    | SyntaxKind.QuestionDotToken
    | SyntaxKind.LessThanToken
    | SyntaxKind.LessThanSlashToken
    | SyntaxKind.GreaterThanToken
    | SyntaxKind.LessThanEqualsToken
    | SyntaxKind.GreaterThanEqualsToken
    | SyntaxKind.EqualsEqualsToken
    | SyntaxKind.ExclamationEqualsToken
    | SyntaxKind.EqualsEqualsEqualsToken
    | SyntaxKind.ExclamationEqualsEqualsToken
    | SyntaxKind.EqualsGreaterThanToken
    | SyntaxKind.PlusToken
    | SyntaxKind.MinusToken
    | SyntaxKind.AsteriskToken
    | SyntaxKind.AsteriskAsteriskToken
    | SyntaxKind.SlashToken
    | SyntaxKind.PercentToken
    | SyntaxKind.PlusPlusToken
    | SyntaxKind.MinusMinusToken
    | SyntaxKind.LessThanLessThanToken
    | SyntaxKind.GreaterThanGreaterThanToken
    | SyntaxKind.GreaterThanGreaterThanGreaterThanToken
    | SyntaxKind.AmpersandToken
    | SyntaxKind.BarToken
    | SyntaxKind.CaretToken
    | SyntaxKind.ExclamationToken
    | SyntaxKind.TildeToken
    | SyntaxKind.AmpersandAmpersandToken
    | SyntaxKind.AmpersandAmpersandEqualsToken
    | SyntaxKind.BarBarToken
    | SyntaxKind.BarBarEqualsToken
    | SyntaxKind.QuestionQuestionToken
    | SyntaxKind.QuestionQuestionEqualsToken
    | SyntaxKind.QuestionToken
    | SyntaxKind.ColonToken
    | SyntaxKind.AtToken
    | SyntaxKind.BacktickToken
    | SyntaxKind.HashToken
    | SyntaxKind.EqualsToken
    | SyntaxKind.PlusEqualsToken
    | SyntaxKind.MinusEqualsToken
    | SyntaxKind.AsteriskEqualsToken
    | SyntaxKind.AsteriskAsteriskEqualsToken
    | SyntaxKind.SlashEqualsToken
    | SyntaxKind.PercentEqualsToken
    | SyntaxKind.LessThanLessThanEqualsToken
    | SyntaxKind.GreaterThanGreaterThanEqualsToken
    | SyntaxKind.GreaterThanGreaterThanGreaterThanEqualsToken
    | SyntaxKind.AmpersandEqualsToken
    | SyntaxKind.BarEqualsToken
    | SyntaxKind.CaretEqualsToken;

/** @internal */
export type PunctuationOrKeywordSyntaxKind = PunctuationSyntaxKind | KeywordSyntaxKind;

export type KeywordSyntaxKind =
    | SyntaxKind.AbstractKeyword
    | SyntaxKind.AccessorKeyword
    | SyntaxKind.AnyKeyword
    | SyntaxKind.AsKeyword
    | SyntaxKind.AssertsKeyword
    | SyntaxKind.AssertKeyword
    | SyntaxKind.AsyncKeyword
    | SyntaxKind.AwaitKeyword
    | SyntaxKind.BigIntKeyword
    | SyntaxKind.BooleanKeyword
    | SyntaxKind.BreakKeyword
    | SyntaxKind.CaseKeyword
    | SyntaxKind.CatchKeyword
    | SyntaxKind.ClassKeyword
    | SyntaxKind.ConstKeyword
    | SyntaxKind.ConstructorKeyword
    | SyntaxKind.ContinueKeyword
    | SyntaxKind.DebuggerKeyword
    | SyntaxKind.DeclareKeyword
    | SyntaxKind.DefaultKeyword
    | SyntaxKind.DeleteKeyword
    | SyntaxKind.DoKeyword
    | SyntaxKind.ElseKeyword
    | SyntaxKind.EnumKeyword
    | SyntaxKind.ExportKeyword
    | SyntaxKind.ExtendsKeyword
    | SyntaxKind.FalseKeyword
    | SyntaxKind.FinallyKeyword
    | SyntaxKind.ForKeyword
    | SyntaxKind.FromKeyword
    | SyntaxKind.FunctionKeyword
    | SyntaxKind.GetKeyword
    | SyntaxKind.GlobalKeyword
    | SyntaxKind.IfKeyword
    | SyntaxKind.ImplementsKeyword
    | SyntaxKind.ImportKeyword
    | SyntaxKind.InferKeyword
    | SyntaxKind.InKeyword
    | SyntaxKind.InstanceOfKeyword
    | SyntaxKind.InterfaceKeyword
    | SyntaxKind.IntrinsicKeyword
    | SyntaxKind.IsKeyword
    | SyntaxKind.KeyOfKeyword
    | SyntaxKind.LetKeyword
    | SyntaxKind.ModuleKeyword
    | SyntaxKind.NamespaceKeyword
    | SyntaxKind.NeverKeyword
    | SyntaxKind.NewKeyword
    | SyntaxKind.NullKeyword
    | SyntaxKind.NumberKeyword
    | SyntaxKind.ObjectKeyword
    | SyntaxKind.OfKeyword
    | SyntaxKind.PackageKeyword
    | SyntaxKind.PrivateKeyword
    | SyntaxKind.ProtectedKeyword
    | SyntaxKind.PublicKeyword
    | SyntaxKind.ReadonlyKeyword
    | SyntaxKind.OutKeyword
    | SyntaxKind.OverrideKeyword
    | SyntaxKind.RequireKeyword
    | SyntaxKind.ReturnKeyword
    | SyntaxKind.SatisfiesKeyword
    | SyntaxKind.SetKeyword
    | SyntaxKind.StaticKeyword
    | SyntaxKind.StringKeyword
    | SyntaxKind.SuperKeyword
    | SyntaxKind.SwitchKeyword
    | SyntaxKind.SymbolKeyword
    | SyntaxKind.ThisKeyword
    | SyntaxKind.ThrowKeyword
    | SyntaxKind.TrueKeyword
    | SyntaxKind.TryKeyword
    | SyntaxKind.TypeKeyword
    | SyntaxKind.TypeOfKeyword
    | SyntaxKind.UndefinedKeyword
    | SyntaxKind.UniqueKeyword
    | SyntaxKind.UnknownKeyword
    | SyntaxKind.UsingKeyword
    | SyntaxKind.VarKeyword
    | SyntaxKind.VoidKeyword
    | SyntaxKind.WhileKeyword
    | SyntaxKind.WithKeyword
    | SyntaxKind.YieldKeyword;

export type ModifierSyntaxKind =
    | SyntaxKind.AbstractKeyword
    | SyntaxKind.AccessorKeyword
    | SyntaxKind.AsyncKeyword
    | SyntaxKind.ConstKeyword
    | SyntaxKind.DeclareKeyword
    | SyntaxKind.DefaultKeyword
    | SyntaxKind.ExportKeyword
    | SyntaxKind.InKeyword
    | SyntaxKind.PrivateKeyword
    | SyntaxKind.ProtectedKeyword
    | SyntaxKind.PublicKeyword
    | SyntaxKind.ReadonlyKeyword
    | SyntaxKind.OutKeyword
    | SyntaxKind.OverrideKeyword
    | SyntaxKind.StaticKeyword;

export type KeywordTypeSyntaxKind =
    | SyntaxKind.AnyKeyword
    | SyntaxKind.BigIntKeyword
    | SyntaxKind.BooleanKeyword
    | SyntaxKind.IntrinsicKeyword
    | SyntaxKind.NeverKeyword
    | SyntaxKind.NumberKeyword
    | SyntaxKind.ObjectKeyword
    | SyntaxKind.StringKeyword
    | SyntaxKind.SymbolKeyword
    | SyntaxKind.UndefinedKeyword
    | SyntaxKind.UnknownKeyword
    | SyntaxKind.VoidKeyword;

/** @internal */
type TypeNodeSyntaxKind =
    | KeywordTypeSyntaxKind
    | SyntaxKind.TypePredicate
    | SyntaxKind.TypeReference
    | SyntaxKind.FunctionType
    | SyntaxKind.ConstructorType
    | SyntaxKind.TypeQuery
    | SyntaxKind.TypeLiteral
    | SyntaxKind.ArrayType
    | SyntaxKind.TupleType
    | SyntaxKind.NamedTupleMember
    | SyntaxKind.OptionalType
    | SyntaxKind.RestType
    | SyntaxKind.UnionType
    | SyntaxKind.IntersectionType
    | SyntaxKind.ConditionalType
    | SyntaxKind.InferType
    | SyntaxKind.ParenthesizedType
    | SyntaxKind.ThisType
    | SyntaxKind.TypeOperator
    | SyntaxKind.IndexedAccessType
    | SyntaxKind.MappedType
    | SyntaxKind.LiteralType
    | SyntaxKind.TemplateLiteralType
    | SyntaxKind.TemplateLiteralTypeSpan
    | SyntaxKind.ImportType
    | SyntaxKind.ExpressionWithTypeArguments
    | SyntaxKind.JSDocTypeExpression
    | SyntaxKind.JSDocAllType
    | SyntaxKind.JSDocUnknownType
    | SyntaxKind.JSDocNonNullableType
    | SyntaxKind.JSDocNullableType
    | SyntaxKind.JSDocOptionalType
    | SyntaxKind.JSDocFunctionType
    | SyntaxKind.JSDocVariadicType
    | SyntaxKind.JSDocNamepathType
    | SyntaxKind.JSDocSignature
    | SyntaxKind.JSDocTypeLiteral;

export type JsxTokenSyntaxKind =
    | SyntaxKind.LessThanSlashToken
    | SyntaxKind.EndOfFileToken
    | SyntaxKind.ConflictMarkerTrivia
    | SyntaxKind.JsxText
    | SyntaxKind.JsxTextAllWhiteSpaces
    | SyntaxKind.OpenBraceToken
    | SyntaxKind.LessThanToken;

export type JSDocSyntaxKind =
    | SyntaxKind.EndOfFileToken
    | SyntaxKind.WhitespaceTrivia
    | SyntaxKind.AtToken
    | SyntaxKind.NewLineTrivia
    | SyntaxKind.AsteriskToken
    | SyntaxKind.OpenBraceToken
    | SyntaxKind.CloseBraceToken
    | SyntaxKind.LessThanToken
    | SyntaxKind.GreaterThanToken
    | SyntaxKind.OpenBracketToken
    | SyntaxKind.CloseBracketToken
    | SyntaxKind.EqualsToken
    | SyntaxKind.CommaToken
    | SyntaxKind.DotToken
    | SyntaxKind.Identifier
    | SyntaxKind.BacktickToken
    | SyntaxKind.HashToken
    | SyntaxKind.Unknown
    | KeywordSyntaxKind;

// dprint-ignore
export const enum NodeFlags {
    None               = 0,
    Let                = 1 << 0,  // Variable declaration
    Const              = 1 << 1,  // Variable declaration
    Using              = 1 << 2,  // Variable declaration
    AwaitUsing         = Const | Using, // Variable declaration (NOTE: on a single node these flags would otherwise be mutually exclusive)
    NestedNamespace    = 1 << 3,  // Namespace declaration
    Synthesized        = 1 << 4,  // Node was synthesized during transformation
    Namespace          = 1 << 5,  // Namespace declaration
    OptionalChain      = 1 << 6,  // Chained MemberExpression rooted to a pseudo-OptionalExpression
    ExportContext      = 1 << 7,  // Export context (initialized by binding)
    ContainsThis       = 1 << 8,  // Interface contains references to "this"
    HasImplicitReturn  = 1 << 9,  // If function implicitly returns on one of codepaths (initialized by binding)
    HasExplicitReturn  = 1 << 10,  // If function has explicit reachable return on one of codepaths (initialized by binding)
    GlobalAugmentation = 1 << 11,  // Set if module declaration is an augmentation for the global scope
    HasAsyncFunctions  = 1 << 12, // If the file has async functions (initialized by binding)
    DisallowInContext  = 1 << 13, // If node was parsed in a context where 'in-expressions' are not allowed
    YieldContext       = 1 << 14, // If node was parsed in the 'yield' context created when parsing a generator
    DecoratorContext   = 1 << 15, // If node was parsed as part of a decorator
    AwaitContext       = 1 << 16, // If node was parsed in the 'await' context created when parsing an async function
    DisallowConditionalTypesContext = 1 << 17, // If node was parsed in a context where conditional types are not allowed
    ThisNodeHasError   = 1 << 18, // If the parser encountered an error when parsing the code that created this node
    JavaScriptFile     = 1 << 19, // If node was parsed in a JavaScript
    ThisNodeOrAnySubNodesHasError = 1 << 20, // If this node or any of its children had an error
    HasAggregatedChildData = 1 << 21, // If we've computed data from children and cached it in this node

    // These flags will be set when the parser encounters a dynamic import expression or 'import.meta' to avoid
    // walking the tree if the flags are not set. However, these flags are just a approximation
    // (hence why it's named "PossiblyContainsDynamicImport") because once set, the flags never get cleared.
    // During editing, if a dynamic import is removed, incremental parsing will *NOT* clear this flag.
    // This means that the tree will always be traversed during module resolution, or when looking for external module indicators.
    // However, the removal operation should not occur often and in the case of the
    // removal, it is likely that users will add the import anyway.
    // The advantage of this approach is its simplicity. For the case of batch compilation,
    // we guarantee that users won't have to pay the price of walking the tree if a dynamic import isn't used.
    /** @internal */ PossiblyContainsDynamicImport = 1 << 22,
    /** @internal */ PossiblyContainsImportMeta    = 1 << 23,

    JSDoc                                          = 1 << 24, // If node was parsed inside jsdoc
    /** @internal */ Ambient                       = 1 << 25, // If node was inside an ambient context -- a declaration file, or inside something with the `declare` modifier.
    /** @internal */ InWithStatement               = 1 << 26, // If any ancestor of node was the `statement` of a WithStatement (not the `expression`)
    JsonFile                                       = 1 << 27, // If node was parsed in a Json
    /** @internal */ TypeCached                    = 1 << 28, // If a type was cached for node at any point
    /** @internal */ Deprecated                    = 1 << 29, // If has '@deprecated' JSDoc tag

    BlockScoped = Let | Const | Using,
    Constant = Const | Using,

    ReachabilityCheckFlags = HasImplicitReturn | HasExplicitReturn,
    ReachabilityAndEmitFlags = ReachabilityCheckFlags | HasAsyncFunctions,

    // Parsing context flags
    ContextFlags = DisallowInContext | DisallowConditionalTypesContext | YieldContext | DecoratorContext | AwaitContext | JavaScriptFile | InWithStatement | Ambient,

    // Exclude these flags when parsing a Type
    TypeExcludesFlags = YieldContext | AwaitContext,

    // Represents all flags that are potentially set once and
    // never cleared on SourceFiles which get re-used in between incremental parses.
    // See the comment above on `PossiblyContainsDynamicImport` and `PossiblyContainsImportMeta`.
    /** @internal */ PermanentlySetIncrementalFlags = PossiblyContainsDynamicImport | PossiblyContainsImportMeta,

    // The following flags repurpose other NodeFlags as different meanings for Identifier nodes
    /** @internal */ IdentifierHasExtendedUnicodeEscape = ContainsThis, // Indicates whether the identifier contains an extended unicode escape sequence
    /** @internal */ IdentifierIsInJSDocNamespace = HasAsyncFunctions, // Indicates whether the identifier is part of a JSDoc namespace
}

// dprint-ignore
export const enum ModifierFlags {
    None =               0,

    // Syntactic/JSDoc modifiers
    Public =             1 << 0,  // Property/Method
    Private =            1 << 1,  // Property/Method
    Protected =          1 << 2,  // Property/Method
    Readonly =           1 << 3,  // Property/Method
    Override =           1 << 4,  // Override method.

    // Syntactic-only modifiers
    Export =             1 << 5,  // Declarations
    Abstract =           1 << 6,  // Class/Method/ConstructSignature
    Ambient =            1 << 7,  // Declarations
    Static =             1 << 8,  // Property/Method
    Accessor =           1 << 9,  // Property
    Async =              1 << 10, // Property/Method/Function
    Default =            1 << 11, // Function/Class (export default declaration)
    Const =              1 << 12, // Const enum
    In =                 1 << 13, // Contravariance modifier
    Out =                1 << 14, // Covariance modifier
    Decorator =          1 << 15, // Contains a decorator.

    // JSDoc-only modifiers
    Deprecated =         1 << 16, // Deprecated tag.

    // Cache-only JSDoc-modifiers. Should match order of Syntactic/JSDoc modifiers, above.
    /** @internal */ JSDocPublic = 1 << 23, // if this value changes, `selectEffectiveModifierFlags` must change accordingly
    /** @internal */ JSDocPrivate = 1 << 24,
    /** @internal */ JSDocProtected = 1 << 25,
    /** @internal */ JSDocReadonly = 1 << 26,
    /** @internal */ JSDocOverride = 1 << 27,

    /** @internal */ SyntacticOrJSDocModifiers = Public | Private | Protected | Readonly | Override,
    /** @internal */ SyntacticOnlyModifiers = Export | Ambient | Abstract | Static | Accessor | Async | Default | Const | In | Out | Decorator,
    /** @internal */ SyntacticModifiers = SyntacticOrJSDocModifiers | SyntacticOnlyModifiers,
    /** @internal */ JSDocCacheOnlyModifiers = JSDocPublic | JSDocPrivate | JSDocProtected | JSDocReadonly | JSDocOverride,
    /** @internal */ JSDocOnlyModifiers = Deprecated,
    /** @internal */ NonCacheOnlyModifiers = SyntacticOrJSDocModifiers | SyntacticOnlyModifiers | JSDocOnlyModifiers,

    HasComputedJSDocModifiers = 1 << 28, // Indicates the computed modifier flags include modifiers from JSDoc.
    HasComputedFlags =   1 << 29, // Modifier flags have been computed

    AccessibilityModifier = Public | Private | Protected,
    // Accessibility modifiers and 'readonly' can be attached to a parameter in a constructor to make it a property.
    ParameterPropertyModifier = AccessibilityModifier | Readonly | Override,
    NonPublicAccessibilityModifier = Private | Protected,

    TypeScriptModifier = Ambient | Public | Private | Protected | Readonly | Abstract | Const | Override | In | Out,
    ExportDefault = Export | Default,
    All = Export | Ambient | Public | Private | Protected | Static | Readonly | Abstract | Accessor | Async | Default | Const | Deprecated | Override | In | Out | Decorator,
    Modifier = All & ~Decorator,
}

/** @internal */
type NodeId = number;

export interface Node extends ReadonlyTextRange {
    readonly kind: SyntaxKind;
    readonly flags: NodeFlags;
    /** @internal */ modifierFlagsCache: ModifierFlags;
    /** @internal */ readonly transformFlags: TransformFlags; // Flags for transforms
    /** @internal */ id?: NodeId; // Unique id (used to look up NodeLinks)
    readonly parent: Node; // Parent node (initialized by binding)
    /** @internal */ original?: Node; // The original node if this is an updated node.
    /** @internal */ emitNode?: EmitNode; // Associated EmitNode (initialized by transforms)
    // NOTE: `symbol` and `localSymbol` have been moved to `Declaration`
    //       `locals` and `nextContainer` have been moved to `LocalsContainer`
    //       `flowNode` has been moved to `FlowContainer`
    //       see: https://github.com/microsoft/TypeScript/pull/51682
}

export interface JSDocContainer extends Node {
    _jsdocContainerBrand: any;
    /** @internal */ jsDoc?: JSDocArray; // JSDoc that directly precedes this node
}

/** @internal */
export interface JSDocArray extends Array<JSDoc> {
    jsDocCache?: readonly JSDocTag[]; // Cache for getJSDocTags
}

interface LocalsContainer extends Node {
    _localsContainerBrand: any;
}

interface FlowContainer extends Node {
    _flowContainerBrand: any;
}

// Ideally, `ForEachChildNodes` and `VisitEachChildNodes` would not differ.
// However, `forEachChild` currently processes JSDoc comment syntax and missing declarations more thoroughly.
// On the other hand, `visitEachChild` actually processes `Identifier`s (which really *shouldn't* have children,
// but are constructed as if they could for faked-up `QualifiedName`s in the language service.)

/** @internal */
export type ForEachChildNodes =
    | HasChildren
    | MissingDeclaration
    | JSDocTypeExpression
    | JSDocNonNullableType
    | JSDocNullableType
    | JSDocOptionalType
    | JSDocVariadicType
    | JSDocFunctionType
    | JSDoc
    | JSDocSeeTag
    | JSDocNameReference
    | JSDocMemberName
    | JSDocParameterTag
    | JSDocPropertyTag
    | JSDocAuthorTag
    | JSDocImplementsTag
    | JSDocAugmentsTag
    | JSDocTemplateTag
    | JSDocTypedefTag
    | JSDocCallbackTag
    | JSDocReturnTag
    | JSDocTypeTag
    | JSDocThisTag
    | JSDocEnumTag
    | JSDocSignature
    | JSDocLink
    | JSDocLinkCode
    | JSDocLinkPlain
    | JSDocTypeLiteral
    | JSDocUnknownTag
    | JSDocClassTag
    | JSDocPublicTag
    | JSDocPrivateTag
    | JSDocProtectedTag
    | JSDocReadonlyTag
    | JSDocDeprecatedTag
    | JSDocThrowsTag
    | JSDocOverrideTag
    | JSDocSatisfiesTag
    | JSDocOverloadTag;

/** @internal */
type HasChildren =
    | QualifiedName
    | ComputedPropertyName
    | TypeParameterDeclaration
    | ParameterDeclaration
    | Decorator
    | PropertySignature
    | PropertyDeclaration
    | MethodSignature
    | MethodDeclaration
    | ConstructorDeclaration
    | GetAccessorDeclaration
    | SetAccessorDeclaration
    | ClassStaticBlockDeclaration
    | CallSignatureDeclaration
    | ConstructSignatureDeclaration
    | IndexSignatureDeclaration
    | TypePredicateNode
    | TypeReferenceNode
    | FunctionTypeNode
    | ConstructorTypeNode
    | TypeQueryNode
    | TypeLiteralNode
    | ArrayTypeNode
    | TupleTypeNode
    | OptionalTypeNode
    | RestTypeNode
    | UnionTypeNode
    | IntersectionTypeNode
    | ConditionalTypeNode
    | InferTypeNode
    | ImportTypeNode
    | ImportTypeAssertionContainer
    | NamedTupleMember
    | ParenthesizedTypeNode
    | TypeOperatorNode
    | IndexedAccessTypeNode
    | MappedTypeNode
    | LiteralTypeNode
    | TemplateLiteralTypeNode
    | TemplateLiteralTypeSpan
    | ObjectBindingPattern
    | ArrayBindingPattern
    | BindingElement
    | ArrayLiteralExpression
    | ObjectLiteralExpression
    | PropertyAccessExpression
    | ElementAccessExpression
    | CallExpression
    | NewExpression
    | TaggedTemplateExpression
    | TypeAssertion
    | ParenthesizedExpression
    | FunctionExpression
    | ArrowFunction
    | DeleteExpression
    | TypeOfExpression
    | VoidExpression
    | AwaitExpression
    | PrefixUnaryExpression
    | PostfixUnaryExpression
    | BinaryExpression
    | ConditionalExpression
    | TemplateExpression
    | YieldExpression
    | SpreadElement
    | ClassExpression
    | ExpressionWithTypeArguments
    | AsExpression
    | NonNullExpression
    | SatisfiesExpression
    | MetaProperty
    | TemplateSpan
    | Block
    | VariableStatement
    | ExpressionStatement
    | IfStatement
    | DoStatement
    | WhileStatement
    | ForStatement
    | ForInStatement
    | ForOfStatement
    | ContinueStatement
    | BreakStatement
    | ReturnStatement
    | WithStatement
    | SwitchStatement
    | LabeledStatement
    | ThrowStatement
    | TryStatement
    | VariableDeclaration
    | VariableDeclarationList
    | FunctionDeclaration
    | ClassDeclaration
    | InterfaceDeclaration
    | TypeAliasDeclaration
    | EnumDeclaration
    | ModuleDeclaration
    | ModuleBlock
    | CaseBlock
    | NamespaceExportDeclaration
    | ImportEqualsDeclaration
    | ImportDeclaration
    | AssertClause
    | AssertEntry
    | ImportAttributes
    | ImportAttribute
    | ImportClause
    | NamespaceImport
    | NamespaceExport
    | NamedImports
    | ImportSpecifier
    | ExportAssignment
    | ExportDeclaration
    | NamedExports
    | ExportSpecifier
    | ExternalModuleReference
    | JsxElement
    | JsxSelfClosingElement
    | JsxOpeningElement
    | JsxClosingElement
    | JsxFragment
    | JsxAttribute
    | JsxAttributes
    | JsxSpreadAttribute
    | JsxExpression
    | JsxNamespacedName
    | CaseClause
    | DefaultClause
    | HeritageClause
    | CatchClause
    | PropertyAssignment
    | ShorthandPropertyAssignment
    | SpreadAssignment
    | EnumMember
    | SourceFile
    | PartiallyEmittedExpression
    | CommaListExpression;

export type HasJSDoc =
    | AccessorDeclaration
    | ArrowFunction
    | BinaryExpression
    | Block
    | BreakStatement
    | CallSignatureDeclaration
    | CaseClause
    | ClassLikeDeclaration
    | ClassStaticBlockDeclaration
    | ConstructorDeclaration
    | ConstructorTypeNode
    | ConstructSignatureDeclaration
    | ContinueStatement
    | DebuggerStatement
    | DoStatement
    | ElementAccessExpression
    | EmptyStatement
    | EndOfFileToken
    | EnumDeclaration
    | EnumMember
    | ExportAssignment
    | ExportDeclaration
    | ExportSpecifier
    | ExpressionStatement
    | ForInStatement
    | ForOfStatement
    | ForStatement
    | FunctionDeclaration
    | FunctionExpression
    | FunctionTypeNode
    | Identifier
    | IfStatement
    | ImportDeclaration
    | ImportEqualsDeclaration
    | IndexSignatureDeclaration
    | InterfaceDeclaration
    | JSDocFunctionType
    | JSDocSignature
    | LabeledStatement
    | MethodDeclaration
    | MethodSignature
    | ModuleDeclaration
    | NamedTupleMember
    | NamespaceExportDeclaration
    | ObjectLiteralExpression
    | ParameterDeclaration
    | ParenthesizedExpression
    | PropertyAccessExpression
    | PropertyAssignment
    | PropertyDeclaration
    | PropertySignature
    | ReturnStatement
    | SemicolonClassElement
    | ShorthandPropertyAssignment
    | SpreadAssignment
    | SwitchStatement
    | ThrowStatement
    | TryStatement
    | TypeAliasDeclaration
    | TypeParameterDeclaration
    | VariableDeclaration
    | VariableStatement
    | WhileStatement
    | WithStatement;

export type HasType =
    | SignatureDeclaration
    | VariableDeclaration
    | ParameterDeclaration
    | PropertySignature
    | PropertyDeclaration
    | TypePredicateNode
    | ParenthesizedTypeNode
    | TypeOperatorNode
    | MappedTypeNode
    | AssertionExpression
    | TypeAliasDeclaration
    | JSDocTypeExpression
    | JSDocNonNullableType
    | JSDocNullableType
    | JSDocOptionalType
    | JSDocVariadicType;

export type HasInitializer =
    | HasExpressionInitializer
    | ForStatement
    | ForInStatement
    | ForOfStatement
    | JsxAttribute;

type HasExpressionInitializer =
    | VariableDeclaration
    | ParameterDeclaration
    | BindingElement
    | PropertyDeclaration
    | PropertyAssignment
    | EnumMember;

// NOTE: Changing the following list requires changes to:
// - `canHaveDecorators` in factory/utilities.ts
// - `updateModifiers` in factory/nodeFactory.ts
export type HasDecorators =
    | ParameterDeclaration
    | PropertyDeclaration
    | MethodDeclaration
    | GetAccessorDeclaration
    | SetAccessorDeclaration
    | ClassExpression
    | ClassDeclaration;

// NOTE: Changing the following list requires changes to:
// - `canHaveModifiers` in factory/utilitiesPublic.ts
// - `updateModifiers` in factory/nodeFactory.ts
export type HasModifiers =
    | TypeParameterDeclaration
    | ParameterDeclaration
    | ConstructorTypeNode
    | PropertySignature
    | PropertyDeclaration
    | MethodSignature
    | MethodDeclaration
    | ConstructorDeclaration
    | GetAccessorDeclaration
    | SetAccessorDeclaration
    | IndexSignatureDeclaration
    | FunctionExpression
    | ArrowFunction
    | ClassExpression
    | VariableStatement
    | FunctionDeclaration
    | ClassDeclaration
    | InterfaceDeclaration
    | TypeAliasDeclaration
    | EnumDeclaration
    | ModuleDeclaration
    | ImportEqualsDeclaration
    | ImportDeclaration
    | ExportAssignment
    | ExportDeclaration;

/** @internal */
export interface MutableNodeArray<T extends Node> extends Array<T>, TextRange {
    hasTrailingComma: boolean;
    /** @internal */ transformFlags: TransformFlags; // Flags for transforms, possibly undefined
}

export interface NodeArray<T extends Node> extends ReadonlyArray<T>, ReadonlyTextRange {
    readonly hasTrailingComma: boolean;
    /** @internal */ transformFlags: TransformFlags; // Flags for transforms, possibly undefined
}

// TODO(rbuckton): Constraint 'TKind' to 'TokenSyntaxKind'
export interface Token<TKind extends SyntaxKind> extends Node {
    readonly kind: TKind;
}

export type EndOfFileToken = Token<SyntaxKind.EndOfFileToken> & JSDocContainer;

// Punctuation
export interface PunctuationToken<TKind extends PunctuationSyntaxKind> extends Token<TKind> {
}

export type DotDotDotToken = PunctuationToken<SyntaxKind.DotDotDotToken>;
export type QuestionToken = PunctuationToken<SyntaxKind.QuestionToken>;
export type ExclamationToken = PunctuationToken<SyntaxKind.ExclamationToken>;
export type ColonToken = PunctuationToken<SyntaxKind.ColonToken>;
export type EqualsToken = PunctuationToken<SyntaxKind.EqualsToken>;
export type AsteriskToken = PunctuationToken<SyntaxKind.AsteriskToken>;
export type EqualsGreaterThanToken = PunctuationToken<SyntaxKind.EqualsGreaterThanToken>;
export type PlusToken = PunctuationToken<SyntaxKind.PlusToken>;
export type MinusToken = PunctuationToken<SyntaxKind.MinusToken>;
export type QuestionDotToken = PunctuationToken<SyntaxKind.QuestionDotToken>;

// Keywords
export interface KeywordToken<TKind extends KeywordSyntaxKind> extends Token<TKind> {
}

export type AssertsKeyword = KeywordToken<SyntaxKind.AssertsKeyword>;
export type AwaitKeyword = KeywordToken<SyntaxKind.AwaitKeyword>;

export interface ModifierToken<TKind extends ModifierSyntaxKind> extends KeywordToken<TKind> {
}

type AbstractKeyword = ModifierToken<SyntaxKind.AbstractKeyword>;
type AccessorKeyword = ModifierToken<SyntaxKind.AccessorKeyword>;
export type AsyncKeyword = ModifierToken<SyntaxKind.AsyncKeyword>;
type ConstKeyword = ModifierToken<SyntaxKind.ConstKeyword>;
type DeclareKeyword = ModifierToken<SyntaxKind.DeclareKeyword>;
type DefaultKeyword = ModifierToken<SyntaxKind.DefaultKeyword>;
export type ExportKeyword = ModifierToken<SyntaxKind.ExportKeyword>;
type InKeyword = ModifierToken<SyntaxKind.InKeyword>;
type PrivateKeyword = ModifierToken<SyntaxKind.PrivateKeyword>;
type ProtectedKeyword = ModifierToken<SyntaxKind.ProtectedKeyword>;
type PublicKeyword = ModifierToken<SyntaxKind.PublicKeyword>;
export type ReadonlyKeyword = ModifierToken<SyntaxKind.ReadonlyKeyword>;
type OutKeyword = ModifierToken<SyntaxKind.OutKeyword>;
type OverrideKeyword = ModifierToken<SyntaxKind.OverrideKeyword>;
type StaticKeyword = ModifierToken<SyntaxKind.StaticKeyword>;

export type Modifier =
    | AbstractKeyword
    | AccessorKeyword
    | AsyncKeyword
    | ConstKeyword
    | DeclareKeyword
    | DefaultKeyword
    | ExportKeyword
    | InKeyword
    | PrivateKeyword
    | ProtectedKeyword
    | PublicKeyword
    | OutKeyword
    | OverrideKeyword
    | ReadonlyKeyword
    | StaticKeyword;

export type ModifierLike = Modifier | Decorator;

// dprint-ignore
export const enum GeneratedIdentifierFlags {
    // Kinds
    None = 0,                           // Not automatically generated.
    /** @internal */ Auto = 1,             // Automatically generated identifier.
    /** @internal */ Loop = 2,             // Automatically generated identifier with a preference for '_i'.
    /** @internal */ Unique = 3,           // Unique name based on the 'text' property.
    /** @internal */ Node = 4,             // Unique name based on the node in the 'original' property.
    /** @internal */ KindMask = 7,         // Mask to extract the kind of identifier from its flags.

    // Flags
    ReservedInNestedScopes = 1 << 3,    // Reserve the generated name in nested scopes
    Optimistic = 1 << 4,                // First instance won't use '_#' if there's no conflict
    FileLevel = 1 << 5,                 // Use only the file identifiers list and not generated names to search for conflicts
    AllowNameSubstitution = 1 << 6, // Used by `module.ts` to indicate generated nodes which can have substitutions performed upon them (as they were generated by an earlier transform phase)
}

export interface Identifier extends PrimaryExpression, Declaration, JSDocContainer, FlowContainer {
    readonly kind: SyntaxKind.Identifier;
    /**
     * Prefer to use `id.unescapedText`. (Note: This is available only in services, not internally to the TypeScript compiler.)
     * Text of identifier, but if the identifier begins with two underscores, this will begin with three.
     */
    readonly escapedText: __String;
}

// dprint-ignore
/** @internal */
export interface AutoGenerateInfo {
    flags: GeneratedIdentifierFlags;            // Specifies whether to auto-generate the text for an identifier.
    readonly id: number;                        // Ensures unique generated identifiers get unique names, but clones get the same name.
    readonly prefix?: string | GeneratedNamePart;
    readonly suffix?: string;
}

/** @internal */
export interface GeneratedIdentifier extends Identifier {
    readonly emitNode: EmitNode & { autoGenerate: AutoGenerateInfo; };
}

export interface QualifiedName extends Node, FlowContainer {
    readonly kind: SyntaxKind.QualifiedName;
    readonly left: EntityName;
    readonly right: Identifier;
}

export type EntityName = Identifier | QualifiedName;

export type PropertyName = Identifier | StringLiteral | NoSubstitutionTemplateLiteral | NumericLiteral | ComputedPropertyName | PrivateIdentifier;

export type MemberName = Identifier | PrivateIdentifier;

export type DeclarationName =
    | PropertyName
    | JsxAttributeName
    | StringLiteralLike
    | ElementAccessExpression
    | BindingPattern
    | EntityNameExpression;

export interface Declaration extends Node {
    _declarationBrand: any;
}

export interface NamedDeclaration extends Declaration {
    readonly name?: DeclarationName;
}

interface DeclarationStatement extends NamedDeclaration, Statement {
    readonly name?: Identifier | StringLiteral | NumericLiteral;
}

export interface ComputedPropertyName extends Node {
    readonly kind: SyntaxKind.ComputedPropertyName;
    readonly parent: Declaration;
    readonly expression: Expression;
}

// Typed as a PrimaryExpression due to its presence in BinaryExpressions (#field in expr)
export interface PrivateIdentifier extends PrimaryExpression {
    readonly kind: SyntaxKind.PrivateIdentifier;
    // escaping not strictly necessary
    // avoids gotchas in transforms and utils
    readonly escapedText: __String;
}

/** @internal */
export interface GeneratedPrivateIdentifier extends PrivateIdentifier {
    readonly emitNode: EmitNode & { autoGenerate: AutoGenerateInfo; };
}

export interface Decorator extends Node {
    readonly kind: SyntaxKind.Decorator;
    readonly parent: NamedDeclaration;
    readonly expression: LeftHandSideExpression;
}

export interface TypeParameterDeclaration extends NamedDeclaration, JSDocContainer {
    readonly kind: SyntaxKind.TypeParameter;
    readonly parent: DeclarationWithTypeParameterChildren | InferTypeNode;
    readonly modifiers?: NodeArray<Modifier>;
    readonly name: Identifier;
    /** Note: Consider calling `getEffectiveConstraintOfTypeParameter` */
    readonly constraint?: TypeNode;
    readonly default?: TypeNode;

    // For error recovery purposes (see `isGrammarError` in utilities.ts).
    expression?: Expression;
}

export interface SignatureDeclarationBase extends NamedDeclaration, JSDocContainer {
    readonly kind: SignatureDeclaration["kind"];
    readonly name?: PropertyName;
    readonly typeParameters?: NodeArray<TypeParameterDeclaration> | undefined;
    readonly parameters: NodeArray<ParameterDeclaration>;
    readonly type?: TypeNode | undefined;
    /** @internal */ typeArguments?: NodeArray<TypeNode>; // Used for quick info, replaces typeParameters for instantiated signatures
}

export type SignatureDeclaration =
    | CallSignatureDeclaration
    | ConstructSignatureDeclaration
    | MethodSignature
    | IndexSignatureDeclaration
    | FunctionTypeNode
    | ConstructorTypeNode
    | JSDocFunctionType
    | FunctionDeclaration
    | MethodDeclaration
    | ConstructorDeclaration
    | AccessorDeclaration
    | FunctionExpression
    | ArrowFunction;

export interface CallSignatureDeclaration extends SignatureDeclarationBase, TypeElement, LocalsContainer {
    readonly kind: SyntaxKind.CallSignature;
}

export interface ConstructSignatureDeclaration extends SignatureDeclarationBase, TypeElement, LocalsContainer {
    readonly kind: SyntaxKind.ConstructSignature;
}

export type BindingName = Identifier | BindingPattern;

// dprint-ignore
export interface VariableDeclaration extends NamedDeclaration, JSDocContainer {
    readonly kind: SyntaxKind.VariableDeclaration;
    readonly parent: VariableDeclarationList | CatchClause;
    readonly name: BindingName;                    // Declared variable name
    readonly exclamationToken?: ExclamationToken;  // Optional definite assignment assertion
    readonly type?: TypeNode;                      // Optional type annotation
    readonly initializer?: Expression;             // Optional initializer
}

export interface VariableDeclarationList extends Node {
    readonly kind: SyntaxKind.VariableDeclarationList;
    readonly parent: VariableStatement | ForStatement | ForOfStatement | ForInStatement;
    readonly declarations: NodeArray<VariableDeclaration>;
}

// dprint-ignore
export interface ParameterDeclaration extends NamedDeclaration, JSDocContainer {
    readonly kind: SyntaxKind.Parameter;
    readonly parent: SignatureDeclaration;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly dotDotDotToken?: DotDotDotToken;    // Present on rest parameter
    readonly name: BindingName;                  // Declared parameter name.
    readonly questionToken?: QuestionToken;      // Present on optional parameter
    readonly type?: TypeNode;                    // Optional type annotation
    readonly initializer?: Expression;           // Optional initializer
}

// dprint-ignore
export interface BindingElement extends NamedDeclaration, FlowContainer {
    readonly kind: SyntaxKind.BindingElement;
    readonly parent: BindingPattern;
    readonly propertyName?: PropertyName;        // Binding property name (in object binding pattern)
    readonly dotDotDotToken?: DotDotDotToken;    // Present on rest element (in object binding pattern)
    readonly name: BindingName;                  // Declared binding element name
    readonly initializer?: Expression;           // Optional initializer
}

// dprint-ignore
export interface PropertySignature extends TypeElement, JSDocContainer {
    readonly kind: SyntaxKind.PropertySignature;
    readonly parent: TypeLiteralNode | InterfaceDeclaration;
    readonly modifiers?: NodeArray<Modifier>;
    readonly name: PropertyName;                 // Declared property name
    readonly questionToken?: QuestionToken;      // Present on optional property
    readonly type?: TypeNode;                    // Optional type annotation

    // The following properties are used only to report grammar errors (see `isGrammarError` in utilities.ts)
    /** @internal */ readonly initializer?: Expression | undefined; // A property signature cannot have an initializer
}

// dprint-ignore
export interface PropertyDeclaration extends ClassElement, JSDocContainer {
    readonly kind: SyntaxKind.PropertyDeclaration;
    readonly parent: ClassLikeDeclaration;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly name: PropertyName;
    readonly questionToken?: QuestionToken;      // Present for use with reporting a grammar error for auto-accessors (see `isGrammarError` in utilities.ts)
    readonly exclamationToken?: ExclamationToken;
    readonly type?: TypeNode;
    readonly initializer?: Expression;           // Optional initializer
}

interface ObjectLiteralElement extends NamedDeclaration {
    _objectLiteralBrand: any;
    readonly name?: PropertyName;
}

/** Unlike ObjectLiteralElement, excludes JSXAttribute and JSXSpreadAttribute. */
export type ObjectLiteralElementLike =
    | PropertyAssignment
    | ShorthandPropertyAssignment
    | SpreadAssignment
    | MethodDeclaration
    | AccessorDeclaration;

export interface PropertyAssignment extends ObjectLiteralElement, JSDocContainer {
    readonly kind: SyntaxKind.PropertyAssignment;
    readonly parent: ObjectLiteralExpression;
    readonly name: PropertyName;
    readonly initializer: Expression;

    // The following properties are used only to report grammar errors (see `isGrammarError` in utilities.ts)
    /** @internal */ readonly modifiers?: NodeArray<ModifierLike> | undefined; // property assignment cannot have decorators or modifiers
    /** @internal */ readonly questionToken?: QuestionToken | undefined; // property assignment cannot have a question token
    /** @internal */ readonly exclamationToken?: ExclamationToken | undefined; // property assignment cannot have an exclamation token
}

export interface ShorthandPropertyAssignment extends ObjectLiteralElement, JSDocContainer {
    readonly kind: SyntaxKind.ShorthandPropertyAssignment;
    readonly parent: ObjectLiteralExpression;
    readonly name: Identifier;
    // used when ObjectLiteralExpression is used in ObjectAssignmentPattern
    // it is a grammar error to appear in actual object initializer (see `isGrammarError` in utilities.ts):
    readonly equalsToken?: EqualsToken;
    readonly objectAssignmentInitializer?: Expression;

    // The following properties are used only to report grammar errors (see `isGrammarError` in utilities.ts)
    /** @internal */ readonly modifiers?: NodeArray<ModifierLike> | undefined; // shorthand property assignment cannot have decorators or modifiers
    /** @internal */ readonly questionToken?: QuestionToken | undefined; // shorthand property assignment cannot have a question token
    /** @internal */ readonly exclamationToken?: ExclamationToken | undefined; // shorthand property assignment cannot have an exclamation token
}

export interface SpreadAssignment extends ObjectLiteralElement, JSDocContainer {
    readonly kind: SyntaxKind.SpreadAssignment;
    readonly parent: ObjectLiteralExpression;
    readonly expression: Expression;
}

export type VariableLikeDeclaration =
    | VariableDeclaration
    | ParameterDeclaration
    | BindingElement
    | PropertyDeclaration
    | PropertyAssignment
    | PropertySignature
    | JsxAttribute
    | ShorthandPropertyAssignment
    | EnumMember
    | JSDocPropertyTag
    | JSDocParameterTag;

export interface ObjectBindingPattern extends Node {
    readonly kind: SyntaxKind.ObjectBindingPattern;
    readonly parent: VariableDeclaration | ParameterDeclaration | BindingElement;
    readonly elements: NodeArray<BindingElement>;
}

export interface ArrayBindingPattern extends Node {
    readonly kind: SyntaxKind.ArrayBindingPattern;
    readonly parent: VariableDeclaration | ParameterDeclaration | BindingElement;
    readonly elements: NodeArray<ArrayBindingElement>;
}

export type BindingPattern = ObjectBindingPattern | ArrayBindingPattern;

export type ArrayBindingElement = BindingElement | OmittedExpression;

/**
 * Several node kinds share function-like features such as a signature,
 * a name, and a body. These nodes should extend FunctionLikeDeclarationBase.
 * Examples:
 * - FunctionDeclaration
 * - MethodDeclaration
 * - AccessorDeclaration
 */
interface FunctionLikeDeclarationBase extends SignatureDeclarationBase {
    _functionLikeDeclarationBrand: any;

    readonly asteriskToken?: AsteriskToken | undefined;
    readonly questionToken?: QuestionToken | undefined;
    readonly exclamationToken?: ExclamationToken | undefined;
    readonly body?: Block | Expression | undefined;
}

export type FunctionLikeDeclaration =
    | FunctionDeclaration
    | MethodDeclaration
    | GetAccessorDeclaration
    | SetAccessorDeclaration
    | ConstructorDeclaration
    | FunctionExpression
    | ArrowFunction;

export interface FunctionDeclaration extends FunctionLikeDeclarationBase, DeclarationStatement, LocalsContainer {
    readonly kind: SyntaxKind.FunctionDeclaration;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly name?: Identifier;
    readonly body?: FunctionBody;
}

export interface MethodSignature extends SignatureDeclarationBase, TypeElement, LocalsContainer {
    readonly kind: SyntaxKind.MethodSignature;
    readonly parent: TypeLiteralNode | InterfaceDeclaration;
    readonly modifiers?: NodeArray<Modifier>;
    readonly name: PropertyName;
}

// Note that a MethodDeclaration is considered both a ClassElement and an ObjectLiteralElement.
// Both the grammars for ClassDeclaration and ObjectLiteralExpression allow for MethodDeclarations
// as child elements, and so a MethodDeclaration satisfies both interfaces.  This avoids the
// alternative where we would need separate kinds/types for ClassMethodDeclaration and
// ObjectLiteralMethodDeclaration, which would look identical.
//
// Because of this, it may be necessary to determine what sort of MethodDeclaration you have
// at later stages of the compiler pipeline.  In that case, you can either check the parent kind
// of the method, or use helpers like isObjectLiteralMethodDeclaration
export interface MethodDeclaration extends FunctionLikeDeclarationBase, ClassElement, ObjectLiteralElement, JSDocContainer, LocalsContainer, FlowContainer {
    readonly kind: SyntaxKind.MethodDeclaration;
    readonly parent: ClassLikeDeclaration | ObjectLiteralExpression;
    readonly modifiers?: NodeArray<ModifierLike> | undefined;
    readonly name: PropertyName;
    readonly body?: FunctionBody | undefined;

    // The following properties are used only to report grammar errors (see `isGrammarError` in utilities.ts)
    /** @internal */ readonly exclamationToken?: ExclamationToken | undefined; // A method cannot have an exclamation token
}

export interface ConstructorDeclaration extends FunctionLikeDeclarationBase, ClassElement, JSDocContainer, LocalsContainer {
    readonly kind: SyntaxKind.Constructor;
    readonly parent: ClassLikeDeclaration;
    readonly modifiers?: NodeArray<ModifierLike> | undefined;
    readonly body?: FunctionBody | undefined;

    // The following properties are used only to report grammar errors (see `isGrammarError` in utilities.ts)
    /** @internal */ readonly typeParameters?: NodeArray<TypeParameterDeclaration>; // A constructor cannot have type parameters
    /** @internal */ readonly type?: TypeNode; // A constructor cannot have a return type annotation
}

/** For when we encounter a semicolon in a class declaration. ES6 allows these as class elements. */
export interface SemicolonClassElement extends ClassElement, JSDocContainer {
    readonly kind: SyntaxKind.SemicolonClassElement;
    readonly parent: ClassLikeDeclaration;
}

// See the comment on MethodDeclaration for the intuition behind GetAccessorDeclaration being a
// ClassElement and an ObjectLiteralElement.
export interface GetAccessorDeclaration extends FunctionLikeDeclarationBase, ClassElement, TypeElement, ObjectLiteralElement, JSDocContainer, LocalsContainer, FlowContainer {
    readonly kind: SyntaxKind.GetAccessor;
    readonly parent: ClassLikeDeclaration | ObjectLiteralExpression | TypeLiteralNode | InterfaceDeclaration;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly name: PropertyName;
    readonly body?: FunctionBody;

    // The following properties are used only to report grammar errors (see `isGrammarError` in utilities.ts)
    /** @internal */ readonly typeParameters?: NodeArray<TypeParameterDeclaration> | undefined; // A get accessor cannot have type parameters
}

// See the comment on MethodDeclaration for the intuition behind SetAccessorDeclaration being a
// ClassElement and an ObjectLiteralElement.
export interface SetAccessorDeclaration extends FunctionLikeDeclarationBase, ClassElement, TypeElement, ObjectLiteralElement, JSDocContainer, LocalsContainer, FlowContainer {
    readonly kind: SyntaxKind.SetAccessor;
    readonly parent: ClassLikeDeclaration | ObjectLiteralExpression | TypeLiteralNode | InterfaceDeclaration;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly name: PropertyName;
    readonly body?: FunctionBody;

    // The following properties are used only to report grammar errors (see `isGrammarError` in utilities.ts)
    /** @internal */ readonly typeParameters?: NodeArray<TypeParameterDeclaration> | undefined; // A set accessor cannot have type parameters
    /** @internal */ readonly type?: TypeNode | undefined; // A set accessor cannot have a return type
}

export type AccessorDeclaration = GetAccessorDeclaration | SetAccessorDeclaration;

export interface IndexSignatureDeclaration extends SignatureDeclarationBase, ClassElement, TypeElement, LocalsContainer {
    readonly kind: SyntaxKind.IndexSignature;
    readonly parent: ObjectTypeDeclaration;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly type: TypeNode;
}

export interface ClassStaticBlockDeclaration extends ClassElement, JSDocContainer, LocalsContainer {
    readonly kind: SyntaxKind.ClassStaticBlockDeclaration;
    readonly parent: ClassDeclaration | ClassExpression;
    readonly body: Block;

    // The following properties are used only to report grammar errors (see `isGrammarError` in utilities.ts)
    /** @internal */ readonly modifiers?: NodeArray<ModifierLike> | undefined;
}

export interface TypeNode extends Node {
    _typeNodeBrand: any;
}

/** @internal */
export interface TypeNode extends Node {
    readonly kind: TypeNodeSyntaxKind;
}

export interface KeywordTypeNode<TKind extends KeywordTypeSyntaxKind = KeywordTypeSyntaxKind> extends KeywordToken<TKind>, TypeNode {
    readonly kind: TKind;
}

/** @deprecated */
export interface ImportTypeAssertionContainer extends Node {
    readonly kind: SyntaxKind.ImportTypeAssertionContainer;
    readonly parent: ImportTypeNode;
    /** @deprecated */ readonly assertClause: AssertClause;
    readonly multiLine?: boolean;
}

export interface ImportTypeNode extends NodeWithTypeArguments {
    readonly kind: SyntaxKind.ImportType;
    readonly isTypeOf: boolean;
    readonly argument: TypeNode;
    /** @deprecated */ readonly assertions?: ImportTypeAssertionContainer;
    readonly attributes?: ImportAttributes;
    readonly qualifier?: EntityName;
}

export interface ThisTypeNode extends TypeNode {
    readonly kind: SyntaxKind.ThisType;
}

export type FunctionOrConstructorTypeNode = FunctionTypeNode | ConstructorTypeNode;

interface FunctionOrConstructorTypeNodeBase extends TypeNode, SignatureDeclarationBase {
    readonly kind: SyntaxKind.FunctionType | SyntaxKind.ConstructorType;
    readonly type: TypeNode;
}

export interface FunctionTypeNode extends FunctionOrConstructorTypeNodeBase, LocalsContainer {
    readonly kind: SyntaxKind.FunctionType;

    // A function type cannot have modifiers
    /** @internal */ readonly modifiers?: undefined;
}

export interface ConstructorTypeNode extends FunctionOrConstructorTypeNodeBase, LocalsContainer {
    readonly kind: SyntaxKind.ConstructorType;
    readonly modifiers?: NodeArray<Modifier>;
}

interface NodeWithTypeArguments extends TypeNode {
    readonly typeArguments?: NodeArray<TypeNode>;
}

export interface TypeReferenceNode extends NodeWithTypeArguments {
    readonly kind: SyntaxKind.TypeReference;
    readonly typeName: EntityName;
}

export interface TypePredicateNode extends TypeNode {
    readonly kind: SyntaxKind.TypePredicate;
    readonly parent: SignatureDeclaration | JSDocTypeExpression;
    readonly assertsModifier?: AssertsKeyword;
    readonly parameterName: Identifier | ThisTypeNode;
    readonly type?: TypeNode;
}

export interface TypeQueryNode extends NodeWithTypeArguments {
    readonly kind: SyntaxKind.TypeQuery;
    readonly exprName: EntityName;
}

// A TypeLiteral is the declaration node for an anonymous symbol.
export interface TypeLiteralNode extends TypeNode, Declaration {
    readonly kind: SyntaxKind.TypeLiteral;
    readonly members: NodeArray<TypeElement>;
}

export interface ArrayTypeNode extends TypeNode {
    readonly kind: SyntaxKind.ArrayType;
    readonly elementType: TypeNode;
}

export interface TupleTypeNode extends TypeNode {
    readonly kind: SyntaxKind.TupleType;
    readonly elements: NodeArray<TypeNode | NamedTupleMember>;
}

export interface NamedTupleMember extends TypeNode, Declaration, JSDocContainer {
    readonly kind: SyntaxKind.NamedTupleMember;
    readonly dotDotDotToken?: Token<SyntaxKind.DotDotDotToken>;
    readonly name: Identifier;
    readonly questionToken?: Token<SyntaxKind.QuestionToken>;
    readonly type: TypeNode;
}

export interface OptionalTypeNode extends TypeNode {
    readonly kind: SyntaxKind.OptionalType;
    readonly type: TypeNode;
}

export interface RestTypeNode extends TypeNode {
    readonly kind: SyntaxKind.RestType;
    readonly type: TypeNode;
}

export type UnionOrIntersectionTypeNode = UnionTypeNode | IntersectionTypeNode;

export interface UnionTypeNode extends TypeNode {
    readonly kind: SyntaxKind.UnionType;
    readonly types: NodeArray<TypeNode>;
}

export interface IntersectionTypeNode extends TypeNode {
    readonly kind: SyntaxKind.IntersectionType;
    readonly types: NodeArray<TypeNode>;
}

export interface ConditionalTypeNode extends TypeNode, LocalsContainer {
    readonly kind: SyntaxKind.ConditionalType;
    readonly checkType: TypeNode;
    readonly extendsType: TypeNode;
    readonly trueType: TypeNode;
    readonly falseType: TypeNode;
}

export interface InferTypeNode extends TypeNode {
    readonly kind: SyntaxKind.InferType;
    readonly typeParameter: TypeParameterDeclaration;
}

export interface ParenthesizedTypeNode extends TypeNode {
    readonly kind: SyntaxKind.ParenthesizedType;
    readonly type: TypeNode;
}

export interface TypeOperatorNode extends TypeNode {
    readonly kind: SyntaxKind.TypeOperator;
    readonly operator: SyntaxKind.KeyOfKeyword | SyntaxKind.UniqueKeyword | SyntaxKind.ReadonlyKeyword;
    readonly type: TypeNode;
}

export interface IndexedAccessTypeNode extends TypeNode {
    readonly kind: SyntaxKind.IndexedAccessType;
    readonly objectType: TypeNode;
    readonly indexType: TypeNode;
}

export interface MappedTypeNode extends TypeNode, Declaration, LocalsContainer {
    readonly kind: SyntaxKind.MappedType;
    readonly readonlyToken?: ReadonlyKeyword | PlusToken | MinusToken;
    readonly typeParameter: TypeParameterDeclaration;
    readonly nameType?: TypeNode;
    readonly questionToken?: QuestionToken | PlusToken | MinusToken;
    readonly type?: TypeNode;
    /** Used only to produce grammar errors */
    readonly members?: NodeArray<TypeElement>;
}

export interface LiteralTypeNode extends TypeNode {
    readonly kind: SyntaxKind.LiteralType;
    readonly literal: NullLiteral | BooleanLiteral | LiteralExpression | PrefixUnaryExpression;
}

export interface StringLiteral extends LiteralExpression, Declaration {
    readonly kind: SyntaxKind.StringLiteral;
    /** @internal */ readonly textSourceNode?: Identifier | StringLiteralLike | NumericLiteral | PrivateIdentifier | JsxNamespacedName; // Allows a StringLiteral to get its text from another node (used by transforms).
    /**
     * Note: this is only set when synthesizing a node, not during parsing.
     *
     * @internal
     */
    readonly singleQuote?: boolean;
}

export type StringLiteralLike = StringLiteral | NoSubstitutionTemplateLiteral;
export type PropertyNameLiteral = Identifier | StringLiteralLike | NumericLiteral | JsxNamespacedName;

export interface TemplateLiteralTypeNode extends TypeNode {
    kind: SyntaxKind.TemplateLiteralType;
    readonly head: TemplateHead;
    readonly templateSpans: NodeArray<TemplateLiteralTypeSpan>;
}

export interface TemplateLiteralTypeSpan extends TypeNode {
    readonly kind: SyntaxKind.TemplateLiteralTypeSpan;
    readonly parent: TemplateLiteralTypeNode;
    readonly type: TypeNode;
    readonly literal: TemplateMiddle | TemplateTail;
}

// Note: 'brands' in our syntax nodes serve to give us a small amount of nominal typing.
// Consider 'Expression'.  Without the brand, 'Expression' is actually no different
// (structurally) than 'Node'.  Because of this you can pass any Node to a function that
// takes an Expression without any error.  By using the 'brands' we ensure that the type
// checker actually thinks you have something of the right type.  Note: the brands are
// never actually given values.  At runtime they have zero cost.

export interface Expression extends Node {
    _expressionBrand: any;
}

export interface OmittedExpression extends Expression {
    readonly kind: SyntaxKind.OmittedExpression;
}

// Represents an expression that is elided as part of a transformation to emit comments on a
// not-emitted node. The 'expression' property of a PartiallyEmittedExpression should be emitted.
export interface PartiallyEmittedExpression extends LeftHandSideExpression {
    readonly kind: SyntaxKind.PartiallyEmittedExpression;
    readonly expression: Expression;
}

export interface UnaryExpression extends Expression {
    _unaryExpressionBrand: any;
}

export interface UpdateExpression extends UnaryExpression {
    _updateExpressionBrand: any;
}

// see: https://tc39.github.io/ecma262/#prod-UpdateExpression
// see: https://tc39.github.io/ecma262/#prod-UnaryExpression
export type PrefixUnaryOperator =
    | SyntaxKind.PlusPlusToken
    | SyntaxKind.MinusMinusToken
    | SyntaxKind.PlusToken
    | SyntaxKind.MinusToken
    | SyntaxKind.TildeToken
    | SyntaxKind.ExclamationToken;

export interface PrefixUnaryExpression extends UpdateExpression {
    readonly kind: SyntaxKind.PrefixUnaryExpression;
    readonly operator: PrefixUnaryOperator;
    readonly operand: UnaryExpression;
}

// see: https://tc39.github.io/ecma262/#prod-UpdateExpression
export type PostfixUnaryOperator =
    | SyntaxKind.PlusPlusToken
    | SyntaxKind.MinusMinusToken;

export interface PostfixUnaryExpression extends UpdateExpression {
    readonly kind: SyntaxKind.PostfixUnaryExpression;
    readonly operand: LeftHandSideExpression;
    readonly operator: PostfixUnaryOperator;
}

export interface LeftHandSideExpression extends UpdateExpression {
    _leftHandSideExpressionBrand: any;
}

export interface MemberExpression extends LeftHandSideExpression {
    _memberExpressionBrand: any;
}

export interface PrimaryExpression extends MemberExpression {
    _primaryExpressionBrand: any;
}

export interface NullLiteral extends PrimaryExpression {
    readonly kind: SyntaxKind.NullKeyword;
}

export interface TrueLiteral extends PrimaryExpression {
    readonly kind: SyntaxKind.TrueKeyword;
}

export interface FalseLiteral extends PrimaryExpression {
    readonly kind: SyntaxKind.FalseKeyword;
}

export type BooleanLiteral = TrueLiteral | FalseLiteral;

export interface ThisExpression extends PrimaryExpression, FlowContainer {
    readonly kind: SyntaxKind.ThisKeyword;
}

export interface SuperExpression extends PrimaryExpression, FlowContainer {
    readonly kind: SyntaxKind.SuperKeyword;
}

export interface ImportExpression extends PrimaryExpression {
    readonly kind: SyntaxKind.ImportKeyword;
}

export interface DeleteExpression extends UnaryExpression {
    readonly kind: SyntaxKind.DeleteExpression;
    readonly expression: UnaryExpression;
}

export interface TypeOfExpression extends UnaryExpression {
    readonly kind: SyntaxKind.TypeOfExpression;
    readonly expression: UnaryExpression;
}

export interface VoidExpression extends UnaryExpression {
    readonly kind: SyntaxKind.VoidExpression;
    readonly expression: UnaryExpression;
}

export interface AwaitExpression extends UnaryExpression {
    readonly kind: SyntaxKind.AwaitExpression;
    readonly expression: UnaryExpression;
}

export interface YieldExpression extends Expression {
    readonly kind: SyntaxKind.YieldExpression;
    readonly asteriskToken?: AsteriskToken;
    readonly expression?: Expression;
}

// see: https://tc39.github.io/ecma262/#prod-ExponentiationExpression
type ExponentiationOperator = SyntaxKind.AsteriskAsteriskToken;

// see: https://tc39.github.io/ecma262/#prod-MultiplicativeOperator
type MultiplicativeOperator =
    | SyntaxKind.AsteriskToken
    | SyntaxKind.SlashToken
    | SyntaxKind.PercentToken;

// see: https://tc39.github.io/ecma262/#prod-MultiplicativeExpression
type MultiplicativeOperatorOrHigher =
    | ExponentiationOperator
    | MultiplicativeOperator;

// see: https://tc39.github.io/ecma262/#prod-AdditiveExpression
type AdditiveOperator =
    | SyntaxKind.PlusToken
    | SyntaxKind.MinusToken;

// see: https://tc39.github.io/ecma262/#prod-AdditiveExpression
type AdditiveOperatorOrHigher =
    | MultiplicativeOperatorOrHigher
    | AdditiveOperator;

// see: https://tc39.github.io/ecma262/#prod-ShiftExpression
type ShiftOperator =
    | SyntaxKind.LessThanLessThanToken
    | SyntaxKind.GreaterThanGreaterThanToken
    | SyntaxKind.GreaterThanGreaterThanGreaterThanToken;

// see: https://tc39.github.io/ecma262/#prod-ShiftExpression
type ShiftOperatorOrHigher =
    | AdditiveOperatorOrHigher
    | ShiftOperator;

// see: https://tc39.github.io/ecma262/#prod-RelationalExpression
type RelationalOperator =
    | SyntaxKind.LessThanToken
    | SyntaxKind.LessThanEqualsToken
    | SyntaxKind.GreaterThanToken
    | SyntaxKind.GreaterThanEqualsToken
    | SyntaxKind.InstanceOfKeyword
    | SyntaxKind.InKeyword;

// see: https://tc39.github.io/ecma262/#prod-RelationalExpression
type RelationalOperatorOrHigher =
    | ShiftOperatorOrHigher
    | RelationalOperator;

// see: https://tc39.github.io/ecma262/#prod-EqualityExpression
type EqualityOperator =
    | SyntaxKind.EqualsEqualsToken
    | SyntaxKind.EqualsEqualsEqualsToken
    | SyntaxKind.ExclamationEqualsEqualsToken
    | SyntaxKind.ExclamationEqualsToken;

// see: https://tc39.github.io/ecma262/#prod-EqualityExpression
type EqualityOperatorOrHigher =
    | RelationalOperatorOrHigher
    | EqualityOperator;

// see: https://tc39.github.io/ecma262/#prod-BitwiseANDExpression
// see: https://tc39.github.io/ecma262/#prod-BitwiseXORExpression
// see: https://tc39.github.io/ecma262/#prod-BitwiseORExpression
type BitwiseOperator =
    | SyntaxKind.AmpersandToken
    | SyntaxKind.BarToken
    | SyntaxKind.CaretToken;

// see: https://tc39.github.io/ecma262/#prod-BitwiseANDExpression
// see: https://tc39.github.io/ecma262/#prod-BitwiseXORExpression
// see: https://tc39.github.io/ecma262/#prod-BitwiseORExpression
type BitwiseOperatorOrHigher =
    | EqualityOperatorOrHigher
    | BitwiseOperator;

// see: https://tc39.github.io/ecma262/#prod-LogicalANDExpression
// see: https://tc39.github.io/ecma262/#prod-LogicalORExpression
type LogicalOperator =
    | SyntaxKind.AmpersandAmpersandToken
    | SyntaxKind.BarBarToken;

// see: https://tc39.github.io/ecma262/#prod-LogicalANDExpression
// see: https://tc39.github.io/ecma262/#prod-LogicalORExpression
type LogicalOperatorOrHigher =
    | BitwiseOperatorOrHigher
    | LogicalOperator;

// see: https://tc39.github.io/ecma262/#prod-AssignmentOperator
type CompoundAssignmentOperator =
    | SyntaxKind.PlusEqualsToken
    | SyntaxKind.MinusEqualsToken
    | SyntaxKind.AsteriskAsteriskEqualsToken
    | SyntaxKind.AsteriskEqualsToken
    | SyntaxKind.SlashEqualsToken
    | SyntaxKind.PercentEqualsToken
    | SyntaxKind.AmpersandEqualsToken
    | SyntaxKind.BarEqualsToken
    | SyntaxKind.CaretEqualsToken
    | SyntaxKind.LessThanLessThanEqualsToken
    | SyntaxKind.GreaterThanGreaterThanGreaterThanEqualsToken
    | SyntaxKind.GreaterThanGreaterThanEqualsToken
    | SyntaxKind.BarBarEqualsToken
    | SyntaxKind.AmpersandAmpersandEqualsToken
    | SyntaxKind.QuestionQuestionEqualsToken;

// see: https://tc39.github.io/ecma262/#prod-AssignmentExpression
type AssignmentOperator =
    | SyntaxKind.EqualsToken
    | CompoundAssignmentOperator;

// see: https://tc39.github.io/ecma262/#prod-AssignmentExpression
type AssignmentOperatorOrHigher =
    | SyntaxKind.QuestionQuestionToken
    | LogicalOperatorOrHigher
    | AssignmentOperator;

// see: https://tc39.github.io/ecma262/#prod-Expression
export type BinaryOperator =
    | AssignmentOperatorOrHigher
    | SyntaxKind.CommaToken;

export type LogicalOrCoalescingAssignmentOperator =
    | SyntaxKind.AmpersandAmpersandEqualsToken
    | SyntaxKind.BarBarEqualsToken
    | SyntaxKind.QuestionQuestionEqualsToken;

export type BinaryOperatorToken = Token<BinaryOperator>;

export interface BinaryExpression extends Expression, Declaration, JSDocContainer {
    readonly kind: SyntaxKind.BinaryExpression;
    readonly left: Expression;
    readonly operatorToken: BinaryOperatorToken;
    readonly right: Expression;
}

export type AssignmentOperatorToken = Token<AssignmentOperator>;

export interface AssignmentExpression<TOperator extends AssignmentOperatorToken> extends BinaryExpression {
    readonly left: LeftHandSideExpression;
    readonly operatorToken: TOperator;
}

interface ObjectDestructuringAssignment extends AssignmentExpression<EqualsToken> {
    readonly left: ObjectLiteralExpression;
}

interface ArrayDestructuringAssignment extends AssignmentExpression<EqualsToken> {
    readonly left: ArrayLiteralExpression;
}

type DestructuringAssignment =
    | ObjectDestructuringAssignment
    | ArrayDestructuringAssignment;

export type BindingOrAssignmentElement =
    | VariableDeclaration
    | ParameterDeclaration
    | ObjectBindingOrAssignmentElement
    | ArrayBindingOrAssignmentElement;

type ObjectBindingOrAssignmentElement =
    | BindingElement
    | PropertyAssignment // AssignmentProperty
    | ShorthandPropertyAssignment // AssignmentProperty
    | SpreadAssignment // AssignmentRestProperty
;

type ArrayBindingOrAssignmentElement =
    | BindingElement
    | OmittedExpression // Elision
    | SpreadElement // AssignmentRestElement
    | ArrayLiteralExpression // ArrayAssignmentPattern
    | ObjectLiteralExpression // ObjectAssignmentPattern
    | AssignmentExpression<EqualsToken> // AssignmentElement
    | Identifier // DestructuringAssignmentTarget
    | PropertyAccessExpression // DestructuringAssignmentTarget
    | ElementAccessExpression // DestructuringAssignmentTarget
;

export type BindingOrAssignmentElementTarget =
    | BindingOrAssignmentPattern
    | Identifier
    | PropertyAccessExpression
    | ElementAccessExpression
    | OmittedExpression;

type ObjectBindingOrAssignmentPattern =
    | ObjectBindingPattern
    | ObjectLiteralExpression // ObjectAssignmentPattern
;

type ArrayBindingOrAssignmentPattern =
    | ArrayBindingPattern
    | ArrayLiteralExpression // ArrayAssignmentPattern
;

export type AssignmentPattern = ObjectLiteralExpression | ArrayLiteralExpression;

export type BindingOrAssignmentPattern = ObjectBindingOrAssignmentPattern | ArrayBindingOrAssignmentPattern;

export interface ConditionalExpression extends Expression {
    readonly kind: SyntaxKind.ConditionalExpression;
    readonly condition: Expression;
    readonly questionToken: QuestionToken;
    readonly whenTrue: Expression;
    readonly colonToken: ColonToken;
    readonly whenFalse: Expression;
}

type FunctionBody = Block;
export type ConciseBody = FunctionBody | Expression;

export interface FunctionExpression extends PrimaryExpression, FunctionLikeDeclarationBase, JSDocContainer, LocalsContainer, FlowContainer {
    readonly kind: SyntaxKind.FunctionExpression;
    readonly modifiers?: NodeArray<Modifier>;
    readonly name?: Identifier;
    readonly body: FunctionBody; // Required, whereas the member inherited from FunctionDeclaration is optional
}

export interface ArrowFunction extends Expression, FunctionLikeDeclarationBase, JSDocContainer, LocalsContainer, FlowContainer {
    readonly kind: SyntaxKind.ArrowFunction;
    readonly modifiers?: NodeArray<Modifier>;
    readonly equalsGreaterThanToken: EqualsGreaterThanToken;
    readonly body: ConciseBody;
    readonly name: never;
}

// The text property of a LiteralExpression stores the interpreted value of the literal in text form. For a StringLiteral,
// or any literal of a template, this means quotes have been removed and escapes have been converted to actual characters.
// For a NumericLiteral, the stored value is the toString() representation of the number. For example 1, 1.00, and 1e0 are all stored as just "1".
export interface LiteralLikeNode extends Node {
    text: string;
    isUnterminated?: boolean;
    hasExtendedUnicodeEscape?: boolean;
}

export interface TemplateLiteralLikeNode extends LiteralLikeNode {
    rawText?: string;
    /** @internal */
    templateFlags?: TokenFlags;
}

// The text property of a LiteralExpression stores the interpreted value of the literal in text form. For a StringLiteral,
// or any literal of a template, this means quotes have been removed and escapes have been converted to actual characters.
// For a NumericLiteral, the stored value is the toString() representation of the number. For example 1, 1.00, and 1e0 are all stored as just "1".
export interface LiteralExpression extends LiteralLikeNode, PrimaryExpression {
    _literalExpressionBrand: any;
}

export interface RegularExpressionLiteral extends LiteralExpression {
    readonly kind: SyntaxKind.RegularExpressionLiteral;
}

export interface NoSubstitutionTemplateLiteral extends LiteralExpression, TemplateLiteralLikeNode, Declaration {
    readonly kind: SyntaxKind.NoSubstitutionTemplateLiteral;
    /** @internal */
    templateFlags?: TokenFlags;
}

// dprint-ignore
export const enum TokenFlags {
    None = 0,
    /** @internal */
    PrecedingLineBreak = 1 << 0,
    /** @internal */
    PrecedingJSDocComment = 1 << 1,
    /** @internal */
    Unterminated = 1 << 2,
    /** @internal */
    ExtendedUnicodeEscape = 1 << 3,     // e.g. `\u{10ffff}`
    Scientific = 1 << 4,                // e.g. `10e2`
    Octal = 1 << 5,                     // e.g. `0777`
    HexSpecifier = 1 << 6,              // e.g. `0x00000000`
    BinarySpecifier = 1 << 7,           // e.g. `0b0110010000000000`
    OctalSpecifier = 1 << 8,            // e.g. `0o777`
    /** @internal */
    ContainsSeparator = 1 << 9,         // e.g. `0b1100_0101`
    /** @internal */
    UnicodeEscape = 1 << 10,            // e.g. `\u00a0`
    /** @internal */
    ContainsInvalidEscape = 1 << 11,    // e.g. `\uhello`
    /** @internal */
    HexEscape = 1 << 12,                // e.g. `\xa0`
    /** @internal */
    ContainsLeadingZero = 1 << 13,      // e.g. `0888`
    /** @internal */
    ContainsInvalidSeparator = 1 << 14, // e.g. `0_1`
    /** @internal */
    BinaryOrOctalSpecifier = BinarySpecifier | OctalSpecifier,
    /** @internal */
    WithSpecifier = HexSpecifier | BinaryOrOctalSpecifier,
    /** @internal */
    StringLiteralFlags = HexEscape | UnicodeEscape | ExtendedUnicodeEscape | ContainsInvalidEscape,
    /** @internal */
    NumericLiteralFlags = Scientific | Octal | ContainsLeadingZero | WithSpecifier | ContainsSeparator | ContainsInvalidSeparator,
    /** @internal */
    TemplateLiteralLikeFlags = HexEscape | UnicodeEscape | ExtendedUnicodeEscape | ContainsInvalidEscape,
    /** @internal */
    IsInvalid = Octal | ContainsLeadingZero | ContainsInvalidSeparator | ContainsInvalidEscape,
}

export interface NumericLiteral extends LiteralExpression, Declaration {
    readonly kind: SyntaxKind.NumericLiteral;
    /** @internal */
    readonly numericLiteralFlags: TokenFlags;
}

export interface BigIntLiteral extends LiteralExpression {
    readonly kind: SyntaxKind.BigIntLiteral;
}

export type LiteralToken =
    | NumericLiteral
    | BigIntLiteral
    | StringLiteral
    | JsxText
    | RegularExpressionLiteral
    | NoSubstitutionTemplateLiteral;

export interface TemplateHead extends TemplateLiteralLikeNode {
    readonly kind: SyntaxKind.TemplateHead;
    readonly parent: TemplateExpression | TemplateLiteralTypeNode;
    /** @internal */
    templateFlags?: TokenFlags;
}

export interface TemplateMiddle extends TemplateLiteralLikeNode {
    readonly kind: SyntaxKind.TemplateMiddle;
    readonly parent: TemplateSpan | TemplateLiteralTypeSpan;
    /** @internal */
    templateFlags?: TokenFlags;
}

export interface TemplateTail extends TemplateLiteralLikeNode {
    readonly kind: SyntaxKind.TemplateTail;
    readonly parent: TemplateSpan | TemplateLiteralTypeSpan;
    /** @internal */
    templateFlags?: TokenFlags;
}

type PseudoLiteralToken =
    | TemplateHead
    | TemplateMiddle
    | TemplateTail;

export type TemplateLiteralToken =
    | NoSubstitutionTemplateLiteral
    | PseudoLiteralToken;

export interface TemplateExpression extends PrimaryExpression {
    readonly kind: SyntaxKind.TemplateExpression;
    readonly head: TemplateHead;
    readonly templateSpans: NodeArray<TemplateSpan>;
}

export type TemplateLiteral =
    | TemplateExpression
    | NoSubstitutionTemplateLiteral;

// Each of these corresponds to a substitution expression and a template literal, in that order.
// The template literal must have kind TemplateMiddleLiteral or TemplateTailLiteral.
export interface TemplateSpan extends Node {
    readonly kind: SyntaxKind.TemplateSpan;
    readonly parent: TemplateExpression;
    readonly expression: Expression;
    readonly literal: TemplateMiddle | TemplateTail;
}

export interface ParenthesizedExpression extends PrimaryExpression, JSDocContainer {
    readonly kind: SyntaxKind.ParenthesizedExpression;
    readonly expression: Expression;
}

/** @internal */
export interface JSDocTypeAssertion extends ParenthesizedExpression {
    readonly _jsDocTypeAssertionBrand: never;
}

export interface ArrayLiteralExpression extends PrimaryExpression {
    readonly kind: SyntaxKind.ArrayLiteralExpression;
    readonly elements: NodeArray<Expression>;
    /** @internal */
    multiLine?: boolean;
}

export interface SpreadElement extends Expression {
    readonly kind: SyntaxKind.SpreadElement;
    readonly parent: ArrayLiteralExpression | CallExpression | NewExpression;
    readonly expression: Expression;
}

/**
 * This interface is a base interface for ObjectLiteralExpression and JSXAttributes to extend from. JSXAttributes is similar to
 * ObjectLiteralExpression in that it contains array of properties; however, JSXAttributes' properties can only be
 * JSXAttribute or JSXSpreadAttribute. ObjectLiteralExpression, on the other hand, can only have properties of type
 * ObjectLiteralElement (e.g. PropertyAssignment, ShorthandPropertyAssignment etc.)
 */
interface ObjectLiteralExpressionBase<T extends ObjectLiteralElement> extends PrimaryExpression, Declaration {
    readonly properties: NodeArray<T>;
}

// An ObjectLiteralExpression is the declaration node for an anonymous symbol.
export interface ObjectLiteralExpression extends ObjectLiteralExpressionBase<ObjectLiteralElementLike>, JSDocContainer {
    readonly kind: SyntaxKind.ObjectLiteralExpression;
    /** @internal */
    multiLine?: boolean;
}

export type EntityNameExpression = Identifier | PropertyAccessEntityNameExpression;
export type AccessExpression = PropertyAccessExpression | ElementAccessExpression;

export interface PropertyAccessExpression extends MemberExpression, NamedDeclaration, JSDocContainer, FlowContainer {
    readonly kind: SyntaxKind.PropertyAccessExpression;
    readonly expression: LeftHandSideExpression;
    readonly questionDotToken?: QuestionDotToken;
    readonly name: MemberName;
}

export interface PropertyAccessChain extends PropertyAccessExpression {
    _optionalChainBrand: any;
    readonly name: MemberName;
}

interface SuperPropertyAccessExpression extends PropertyAccessExpression {
    readonly expression: SuperExpression;
}

/** Brand for a PropertyAccessExpression which, like a QualifiedName, consists of a sequence of identifiers separated by dots. */
export interface PropertyAccessEntityNameExpression extends PropertyAccessExpression {
    _propertyAccessExpressionLikeQualifiedNameBrand?: any;
    readonly expression: EntityNameExpression;
    readonly name: Identifier;
}

export interface ElementAccessExpression extends MemberExpression, Declaration, JSDocContainer, FlowContainer {
    readonly kind: SyntaxKind.ElementAccessExpression;
    readonly expression: LeftHandSideExpression;
    readonly questionDotToken?: QuestionDotToken;
    readonly argumentExpression: Expression;
}

export interface ElementAccessChain extends ElementAccessExpression {
    _optionalChainBrand: any;
}

interface SuperElementAccessExpression extends ElementAccessExpression {
    readonly expression: SuperExpression;
}

// see: https://tc39.github.io/ecma262/#prod-SuperProperty
export type SuperProperty = SuperPropertyAccessExpression | SuperElementAccessExpression;

export interface CallExpression extends LeftHandSideExpression, Declaration {
    readonly kind: SyntaxKind.CallExpression;
    readonly expression: LeftHandSideExpression;
    readonly questionDotToken?: QuestionDotToken;
    readonly typeArguments?: NodeArray<TypeNode>;
    readonly arguments: NodeArray<Expression>;
}

export interface CallChain extends CallExpression {
    _optionalChainBrand: any;
}

/** @internal */
export type BindableObjectDefinePropertyCall = CallExpression & {
    readonly arguments: readonly [BindableStaticNameExpression, StringLiteralLike | NumericLiteral, ObjectLiteralExpression] & Readonly<TextRange>;
};

/** @internal */
export type BindableStaticNameExpression =
    | EntityNameExpression
    | BindableStaticElementAccessExpression;

/** @internal */
export type LiteralLikeElementAccessExpression = ElementAccessExpression & Declaration & {
    readonly argumentExpression: StringLiteralLike | NumericLiteral;
};

/** @internal */
export type BindableStaticElementAccessExpression = LiteralLikeElementAccessExpression & {
    readonly expression: BindableStaticNameExpression;
};

/** @internal */
export type BindableStaticAccessExpression =
    | PropertyAccessEntityNameExpression
    | BindableStaticElementAccessExpression;

export interface ExpressionWithTypeArguments extends MemberExpression, NodeWithTypeArguments {
    readonly kind: SyntaxKind.ExpressionWithTypeArguments;
    readonly expression: LeftHandSideExpression;
}

export interface NewExpression extends PrimaryExpression, Declaration {
    readonly kind: SyntaxKind.NewExpression;
    readonly expression: LeftHandSideExpression;
    readonly typeArguments?: NodeArray<TypeNode>;
    readonly arguments?: NodeArray<Expression>;
}

export interface TaggedTemplateExpression extends MemberExpression {
    readonly kind: SyntaxKind.TaggedTemplateExpression;
    readonly tag: LeftHandSideExpression;
    readonly typeArguments?: NodeArray<TypeNode>;
    readonly template: TemplateLiteral;
    /** @internal */ questionDotToken?: QuestionDotToken; // NOTE: Invalid syntax, only used to report a grammar error.
}

export interface AsExpression extends Expression {
    readonly kind: SyntaxKind.AsExpression;
    readonly expression: Expression;
    readonly type: TypeNode;
}

export interface TypeAssertion extends UnaryExpression {
    readonly kind: SyntaxKind.TypeAssertionExpression;
    readonly type: TypeNode;
    readonly expression: UnaryExpression;
}

export interface SatisfiesExpression extends Expression {
    readonly kind: SyntaxKind.SatisfiesExpression;
    readonly expression: Expression;
    readonly type: TypeNode;
}

export type AssertionExpression =
    | TypeAssertion
    | AsExpression;

export interface NonNullExpression extends LeftHandSideExpression {
    readonly kind: SyntaxKind.NonNullExpression;
    readonly expression: Expression;
}

export interface NonNullChain extends NonNullExpression {
    _optionalChainBrand: any;
}

// NOTE: MetaProperty is really a MemberExpression, but we consider it a PrimaryExpression
//       for the same reasons we treat NewExpression as a PrimaryExpression.
export interface MetaProperty extends PrimaryExpression, FlowContainer {
    readonly kind: SyntaxKind.MetaProperty;
    readonly keywordToken: SyntaxKind.NewKeyword | SyntaxKind.ImportKeyword;
    readonly name: Identifier;
}

/// A JSX expression of the form <TagName attrs>...</TagName>
export interface JsxElement extends PrimaryExpression {
    readonly kind: SyntaxKind.JsxElement;
    readonly openingElement: JsxOpeningElement;
    readonly children: NodeArray<JsxChild>;
    readonly closingElement: JsxClosingElement;
}

/// Either the opening tag in a <Tag>...</Tag> pair or the lone <Tag /> in a self-closing form
export type JsxOpeningLikeElement =
    | JsxSelfClosingElement
    | JsxOpeningElement;

export type JsxAttributeLike =
    | JsxAttribute
    | JsxSpreadAttribute;

export type JsxAttributeName =
    | Identifier
    | JsxNamespacedName;

export type JsxTagNameExpression =
    | Identifier
    | ThisExpression
    | JsxTagNamePropertyAccess
    | JsxNamespacedName;

interface JsxTagNamePropertyAccess extends PropertyAccessExpression {
    readonly expression: Identifier | ThisExpression | JsxTagNamePropertyAccess;
}

export interface JsxAttributes extends PrimaryExpression, Declaration {
    readonly properties: NodeArray<JsxAttributeLike>;
    readonly kind: SyntaxKind.JsxAttributes;
    readonly parent: JsxOpeningLikeElement;
}

export interface JsxNamespacedName extends Node {
    readonly kind: SyntaxKind.JsxNamespacedName;
    readonly name: Identifier;
    readonly namespace: Identifier;
}

/// The opening element of a <Tag>...</Tag> JsxElement
export interface JsxOpeningElement extends Expression {
    readonly kind: SyntaxKind.JsxOpeningElement;
    readonly parent: JsxElement;
    readonly tagName: JsxTagNameExpression;
    readonly typeArguments?: NodeArray<TypeNode>;
    readonly attributes: JsxAttributes;
}

/// A JSX expression of the form <TagName attrs />
export interface JsxSelfClosingElement extends PrimaryExpression {
    readonly kind: SyntaxKind.JsxSelfClosingElement;
    readonly tagName: JsxTagNameExpression;
    readonly typeArguments?: NodeArray<TypeNode>;
    readonly attributes: JsxAttributes;
}

/// A JSX expression of the form <>...</>
export interface JsxFragment extends PrimaryExpression {
    readonly kind: SyntaxKind.JsxFragment;
    readonly openingFragment: JsxOpeningFragment;
    readonly children: NodeArray<JsxChild>;
    readonly closingFragment: JsxClosingFragment;
}

/// The opening element of a <>...</> JsxFragment
export interface JsxOpeningFragment extends Expression {
    readonly kind: SyntaxKind.JsxOpeningFragment;
    readonly parent: JsxFragment;
}

/// The closing element of a <>...</> JsxFragment
export interface JsxClosingFragment extends Expression {
    readonly kind: SyntaxKind.JsxClosingFragment;
    readonly parent: JsxFragment;
}

export interface JsxAttribute extends Declaration {
    readonly kind: SyntaxKind.JsxAttribute;
    readonly parent: JsxAttributes;
    readonly name: JsxAttributeName;
    /// JSX attribute initializers are optional; <X y /> is sugar for <X y={true} />
    readonly initializer?: JsxAttributeValue;
}

export type JsxAttributeValue =
    | StringLiteral
    | JsxExpression
    | JsxElement
    | JsxSelfClosingElement
    | JsxFragment;

export interface JsxSpreadAttribute extends ObjectLiteralElement {
    readonly kind: SyntaxKind.JsxSpreadAttribute;
    readonly parent: JsxAttributes;
    readonly expression: Expression;
}

export interface JsxClosingElement extends Node {
    readonly kind: SyntaxKind.JsxClosingElement;
    readonly parent: JsxElement;
    readonly tagName: JsxTagNameExpression;
}

export interface JsxExpression extends Expression {
    readonly kind: SyntaxKind.JsxExpression;
    readonly parent: JsxElement | JsxFragment | JsxAttributeLike;
    readonly dotDotDotToken?: Token<SyntaxKind.DotDotDotToken>;
    readonly expression?: Expression;
}

export interface JsxText extends LiteralLikeNode {
    readonly kind: SyntaxKind.JsxText;
    readonly parent: JsxElement | JsxFragment;
    readonly containsOnlyTriviaWhiteSpaces: boolean;
}

export type JsxChild =
    | JsxText
    | JsxExpression
    | JsxElement
    | JsxSelfClosingElement
    | JsxFragment;

export interface Statement extends Node, JSDocContainer {
    _statementBrand: any;
}

// Represents a statement that is elided as part of a transformation to emit comments on a
// not-emitted node.
export interface NotEmittedStatement extends Statement {
    readonly kind: SyntaxKind.NotEmittedStatement;
}

/**
 * A list of comma-separated expressions. This node is only created by transformations.
 */
export interface CommaListExpression extends Expression {
    readonly kind: SyntaxKind.CommaListExpression;
    readonly elements: NodeArray<Expression>;
}

/** @internal */
export interface SyntheticReferenceExpression extends LeftHandSideExpression {
    readonly kind: SyntaxKind.SyntheticReferenceExpression;
    readonly expression: Expression;
    readonly thisArg: Expression;
}

export interface EmptyStatement extends Statement {
    readonly kind: SyntaxKind.EmptyStatement;
}

export interface DebuggerStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.DebuggerStatement;
}

export interface MissingDeclaration extends DeclarationStatement, PrimaryExpression {
    readonly kind: SyntaxKind.MissingDeclaration;
    readonly name?: Identifier;

    // The following properties are used only to report grammar errors
    /** @internal */ readonly modifiers?: NodeArray<ModifierLike> | undefined;
}

export interface Block extends Statement, LocalsContainer {
    readonly kind: SyntaxKind.Block;
    readonly statements: NodeArray<Statement>;
    /** @internal */ multiLine?: boolean;
}

export interface VariableStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.VariableStatement;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly declarationList: VariableDeclarationList;
}

export interface ExpressionStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.ExpressionStatement;
    readonly expression: Expression;
}

/** @internal */
export interface PrologueDirective extends ExpressionStatement {
    readonly expression: StringLiteral;
}

export interface IfStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.IfStatement;
    readonly expression: Expression;
    readonly thenStatement: Statement;
    readonly elseStatement?: Statement;
}

export interface IterationStatement extends Statement {
    readonly statement: Statement;
}

export interface DoStatement extends IterationStatement, FlowContainer {
    readonly kind: SyntaxKind.DoStatement;
    readonly expression: Expression;
}

export interface WhileStatement extends IterationStatement, FlowContainer {
    readonly kind: SyntaxKind.WhileStatement;
    readonly expression: Expression;
}

export type ForInitializer =
    | VariableDeclarationList
    | Expression;

export interface ForStatement extends IterationStatement, LocalsContainer, FlowContainer {
    readonly kind: SyntaxKind.ForStatement;
    readonly initializer?: ForInitializer;
    readonly condition?: Expression;
    readonly incrementor?: Expression;
}

export type ForInOrOfStatement =
    | ForInStatement
    | ForOfStatement;

export interface ForInStatement extends IterationStatement, LocalsContainer, FlowContainer {
    readonly kind: SyntaxKind.ForInStatement;
    readonly initializer: ForInitializer;
    readonly expression: Expression;
}

export interface ForOfStatement extends IterationStatement, LocalsContainer, FlowContainer {
    readonly kind: SyntaxKind.ForOfStatement;
    readonly awaitModifier?: AwaitKeyword;
    readonly initializer: ForInitializer;
    readonly expression: Expression;
}

export interface BreakStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.BreakStatement;
    readonly label?: Identifier;
}

export interface ContinueStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.ContinueStatement;
    readonly label?: Identifier;
}

export type BreakOrContinueStatement =
    | BreakStatement
    | ContinueStatement;

export interface ReturnStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.ReturnStatement;
    readonly expression?: Expression;
}

export interface WithStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.WithStatement;
    readonly expression: Expression;
    readonly statement: Statement;
}

export interface SwitchStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.SwitchStatement;
    readonly expression: Expression;
    readonly caseBlock: CaseBlock;
    possiblyExhaustive?: boolean; // initialized by binding
}

export interface CaseBlock extends Node, LocalsContainer {
    readonly kind: SyntaxKind.CaseBlock;
    readonly parent: SwitchStatement;
    readonly clauses: NodeArray<CaseOrDefaultClause>;
}

export interface CaseClause extends Node, JSDocContainer {
    readonly kind: SyntaxKind.CaseClause;
    readonly parent: CaseBlock;
    readonly expression: Expression;
    readonly statements: NodeArray<Statement>;
}

export interface DefaultClause extends Node {
    readonly kind: SyntaxKind.DefaultClause;
    readonly parent: CaseBlock;
    readonly statements: NodeArray<Statement>;
}

export type CaseOrDefaultClause =
    | CaseClause
    | DefaultClause;

export interface LabeledStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.LabeledStatement;
    readonly label: Identifier;
    readonly statement: Statement;
}

export interface ThrowStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.ThrowStatement;
    readonly expression: Expression;
}

export interface TryStatement extends Statement, FlowContainer {
    readonly kind: SyntaxKind.TryStatement;
    readonly tryBlock: Block;
    readonly catchClause?: CatchClause;
    readonly finallyBlock?: Block;
}

export interface CatchClause extends Node, LocalsContainer {
    readonly kind: SyntaxKind.CatchClause;
    readonly parent: TryStatement;
    readonly variableDeclaration?: VariableDeclaration;
    readonly block: Block;
}

type ObjectTypeDeclaration =
    | ClassLikeDeclaration
    | InterfaceDeclaration
    | TypeLiteralNode;

type DeclarationWithTypeParameterChildren =
    | SignatureDeclaration
    | ClassLikeDeclaration
    | InterfaceDeclaration
    | TypeAliasDeclaration
    | JSDocTemplateTag;

interface ClassLikeDeclarationBase extends NamedDeclaration, JSDocContainer {
    readonly kind: SyntaxKind.ClassDeclaration | SyntaxKind.ClassExpression;
    readonly name?: Identifier;
    readonly typeParameters?: NodeArray<TypeParameterDeclaration>;
    readonly heritageClauses?: NodeArray<HeritageClause>;
    readonly members: NodeArray<ClassElement>;
}

export interface ClassDeclaration extends ClassLikeDeclarationBase, DeclarationStatement {
    readonly kind: SyntaxKind.ClassDeclaration;
    readonly modifiers?: NodeArray<ModifierLike>;
    /** May be undefined in `export default class { ... }`. */
    readonly name?: Identifier;
}

export interface ClassExpression extends ClassLikeDeclarationBase, PrimaryExpression {
    readonly kind: SyntaxKind.ClassExpression;
    readonly modifiers?: NodeArray<ModifierLike>;
}

export type ClassLikeDeclaration =
    | ClassDeclaration
    | ClassExpression;

export interface ClassElement extends NamedDeclaration {
    _classElementBrand: any;
    readonly name?: PropertyName;
}

export interface TypeElement extends NamedDeclaration {
    _typeElementBrand: any;
    readonly name?: PropertyName;
    readonly questionToken?: QuestionToken | undefined;
}

export interface InterfaceDeclaration extends DeclarationStatement, JSDocContainer {
    readonly kind: SyntaxKind.InterfaceDeclaration;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly name: Identifier;
    readonly typeParameters?: NodeArray<TypeParameterDeclaration>;
    readonly heritageClauses?: NodeArray<HeritageClause>;
    readonly members: NodeArray<TypeElement>;
}

export interface HeritageClause extends Node {
    readonly kind: SyntaxKind.HeritageClause;
    readonly parent: InterfaceDeclaration | ClassLikeDeclaration;
    readonly token: SyntaxKind.ExtendsKeyword | SyntaxKind.ImplementsKeyword;
    readonly types: NodeArray<ExpressionWithTypeArguments>;
}

export interface TypeAliasDeclaration extends DeclarationStatement, JSDocContainer, LocalsContainer {
    readonly kind: SyntaxKind.TypeAliasDeclaration;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly name: Identifier;
    readonly typeParameters?: NodeArray<TypeParameterDeclaration>;
    readonly type: TypeNode;
}

export interface EnumMember extends NamedDeclaration, JSDocContainer {
    readonly kind: SyntaxKind.EnumMember;
    readonly parent: EnumDeclaration;
    // This does include ComputedPropertyName, but the parser will give an error
    // if it parses a ComputedPropertyName in an EnumMember
    readonly name: PropertyName;
    readonly initializer?: Expression;
}

export interface EnumDeclaration extends DeclarationStatement, JSDocContainer {
    readonly kind: SyntaxKind.EnumDeclaration;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly name: Identifier;
    readonly members: NodeArray<EnumMember>;
}

export type ModuleName =
    | Identifier
    | StringLiteral;

export type ModuleBody =
    | NamespaceBody
    | JSDocNamespaceBody;

export interface ModuleDeclaration extends DeclarationStatement, JSDocContainer, LocalsContainer {
    readonly kind: SyntaxKind.ModuleDeclaration;
    readonly parent: ModuleBody | SourceFile;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly name: ModuleName;
    readonly body?: ModuleBody | JSDocNamespaceDeclaration;
}

type NamespaceBody =
    | ModuleBlock
    | NamespaceDeclaration;

export interface NamespaceDeclaration extends ModuleDeclaration {
    readonly name: Identifier;
    readonly body: NamespaceBody;
}

export type JSDocNamespaceBody =
    | Identifier
    | JSDocNamespaceDeclaration;

export interface JSDocNamespaceDeclaration extends ModuleDeclaration {
    readonly name: Identifier;
    readonly body?: JSDocNamespaceBody;
}

export interface ModuleBlock extends Node, Statement {
    readonly kind: SyntaxKind.ModuleBlock;
    readonly parent: ModuleDeclaration;
    readonly statements: NodeArray<Statement>;
}

export type ModuleReference =
    | EntityName
    | ExternalModuleReference;

/**
 * One of:
 * - import x = require("mod");
 * - import x = M.x;
 */
export interface ImportEqualsDeclaration extends DeclarationStatement, JSDocContainer {
    readonly kind: SyntaxKind.ImportEqualsDeclaration;
    readonly parent: SourceFile | ModuleBlock;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly name: Identifier;
    readonly isTypeOnly: boolean;

    // 'EntityName' for an internal module reference, 'ExternalModuleReference' for an external
    // module reference.
    readonly moduleReference: ModuleReference;
}

export interface ExternalModuleReference extends Node {
    readonly kind: SyntaxKind.ExternalModuleReference;
    readonly parent: ImportEqualsDeclaration;
    readonly expression: Expression;
}

// In case of:
// import "mod"  => importClause = undefined, moduleSpecifier = "mod"
// In rest of the cases, module specifier is string literal corresponding to module
// ImportClause information is shown at its declaration below.
export interface ImportDeclaration extends Statement {
    readonly kind: SyntaxKind.ImportDeclaration;
    readonly parent: SourceFile | ModuleBlock;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly importClause?: ImportClause;
    /** If this is not a StringLiteral it will be a grammar error. */
    readonly moduleSpecifier: Expression;
    /** @deprecated */ readonly assertClause?: AssertClause;
    readonly attributes?: ImportAttributes;
}

export type NamedImportBindings =
    | NamespaceImport
    | NamedImports;

export type NamedExportBindings =
    | NamespaceExport
    | NamedExports;

// In case of:
// import d from "mod" => name = d, namedBinding = undefined
// import * as ns from "mod" => name = undefined, namedBinding: NamespaceImport = { name: ns }
// import d, * as ns from "mod" => name = d, namedBinding: NamespaceImport = { name: ns }
// import { a, b as x } from "mod" => name = undefined, namedBinding: NamedImports = { elements: [{ name: a }, { name: x, propertyName: b}]}
// import d, { a, b as x } from "mod" => name = d, namedBinding: NamedImports = { elements: [{ name: a }, { name: x, propertyName: b}]}
export interface ImportClause extends NamedDeclaration {
    readonly kind: SyntaxKind.ImportClause;
    readonly parent: ImportDeclaration;
    readonly isTypeOnly: boolean;
    readonly name?: Identifier; // Default binding
    readonly namedBindings?: NamedImportBindings;
}

/** @deprecated */
export type AssertionKey = ImportAttributeName;

/** @deprecated */
export interface AssertEntry extends ImportAttribute {}

/** @deprecated */
export interface AssertClause extends ImportAttributes {}

export type ImportAttributeName = Identifier | StringLiteral;

export interface ImportAttribute extends Node {
    readonly kind: SyntaxKind.ImportAttribute;
    readonly parent: ImportAttributes;
    readonly name: ImportAttributeName;
    readonly value: Expression;
}

export interface ImportAttributes extends Node {
    readonly token: SyntaxKind.WithKeyword | SyntaxKind.AssertKeyword;
    readonly kind: SyntaxKind.ImportAttributes;
    readonly parent: ImportDeclaration | ExportDeclaration;
    readonly elements: NodeArray<ImportAttribute>;
    readonly multiLine?: boolean;
}

export interface NamespaceImport extends NamedDeclaration {
    readonly kind: SyntaxKind.NamespaceImport;
    readonly parent: ImportClause;
    readonly name: Identifier;
}

export interface NamespaceExport extends NamedDeclaration {
    readonly kind: SyntaxKind.NamespaceExport;
    readonly parent: ExportDeclaration;
    readonly name: Identifier;
}

export interface NamespaceExportDeclaration extends DeclarationStatement, JSDocContainer {
    readonly kind: SyntaxKind.NamespaceExportDeclaration;
    readonly name: Identifier;

    // The following properties are used only to report grammar errors (see `isGrammarError` in utilities.ts)
    /** @internal */ readonly modifiers?: NodeArray<ModifierLike> | undefined;
}

export interface ExportDeclaration extends DeclarationStatement, JSDocContainer {
    readonly kind: SyntaxKind.ExportDeclaration;
    readonly parent: SourceFile | ModuleBlock;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly isTypeOnly: boolean;
    /** Will not be assigned in the case of `export * from "foo";` */
    readonly exportClause?: NamedExportBindings;
    /** If this is not a StringLiteral it will be a grammar error. */
    readonly moduleSpecifier?: Expression;
    /** @deprecated */ readonly assertClause?: AssertClause;
    readonly attributes?: ImportAttributes;
}

export interface NamedImports extends Node {
    readonly kind: SyntaxKind.NamedImports;
    readonly parent: ImportClause;
    readonly elements: NodeArray<ImportSpecifier>;
}

export interface NamedExports extends Node {
    readonly kind: SyntaxKind.NamedExports;
    readonly parent: ExportDeclaration;
    readonly elements: NodeArray<ExportSpecifier>;
}

export type NamedImportsOrExports = NamedImports | NamedExports;

export interface ImportSpecifier extends NamedDeclaration {
    readonly kind: SyntaxKind.ImportSpecifier;
    readonly parent: NamedImports;
    readonly propertyName?: Identifier; // Name preceding "as" keyword (or undefined when "as" is absent)
    readonly name: Identifier; // Declared name
    readonly isTypeOnly: boolean;
}

export interface ExportSpecifier extends NamedDeclaration, JSDocContainer {
    readonly kind: SyntaxKind.ExportSpecifier;
    readonly parent: NamedExports;
    readonly isTypeOnly: boolean;
    readonly propertyName?: Identifier; // Name preceding "as" keyword (or undefined when "as" is absent)
    readonly name: Identifier; // Declared name
}

export type ImportOrExportSpecifier =
    | ImportSpecifier
    | ExportSpecifier;

/**
 * This is either an `export =` or an `export default` declaration.
 * Unless `isExportEquals` is set, this node was parsed as an `export default`.
 */
export interface ExportAssignment extends DeclarationStatement, JSDocContainer {
    readonly kind: SyntaxKind.ExportAssignment;
    readonly parent: SourceFile;
    readonly modifiers?: NodeArray<ModifierLike>;
    readonly isExportEquals?: boolean;
    readonly expression: Expression;
}

export interface FileReference extends TextRange {
    fileName: string;
    resolutionMode?: ResolutionMode;
}

interface CheckJsDirective extends TextRange {
    enabled: boolean;
}

export type CommentKind = SyntaxKind.SingleLineCommentTrivia | SyntaxKind.MultiLineCommentTrivia;

export interface CommentRange extends TextRange {
    hasTrailingNewLine?: boolean;
    kind: CommentKind;
}

export interface SynthesizedComment extends CommentRange {
    text: string;
    pos: -1;
    end: -1;
    hasLeadingNewline?: boolean;
}

// represents a top level: { type } expression in a JSDoc comment.
export interface JSDocTypeExpression extends TypeNode {
    readonly kind: SyntaxKind.JSDocTypeExpression;
    readonly type: TypeNode;
}

export interface JSDocNameReference extends Node {
    readonly kind: SyntaxKind.JSDocNameReference;
    readonly name: EntityName | JSDocMemberName;
}

/** Class#method reference in JSDoc */
export interface JSDocMemberName extends Node {
    readonly kind: SyntaxKind.JSDocMemberName;
    readonly left: EntityName | JSDocMemberName;
    readonly right: Identifier;
}

export interface JSDocType extends TypeNode {
    _jsDocTypeBrand: any;
}

export interface JSDocAllType extends JSDocType {
    readonly kind: SyntaxKind.JSDocAllType;
}

export interface JSDocUnknownType extends JSDocType {
    readonly kind: SyntaxKind.JSDocUnknownType;
}

export interface JSDocNonNullableType extends JSDocType {
    readonly kind: SyntaxKind.JSDocNonNullableType;
    readonly type: TypeNode;
    readonly postfix: boolean;
}

export interface JSDocNullableType extends JSDocType {
    readonly kind: SyntaxKind.JSDocNullableType;
    readonly type: TypeNode;
    readonly postfix: boolean;
}

export interface JSDocOptionalType extends JSDocType {
    readonly kind: SyntaxKind.JSDocOptionalType;
    readonly type: TypeNode;
}

export interface JSDocFunctionType extends JSDocType, SignatureDeclarationBase, LocalsContainer {
    readonly kind: SyntaxKind.JSDocFunctionType;
}

export interface JSDocVariadicType extends JSDocType {
    readonly kind: SyntaxKind.JSDocVariadicType;
    readonly type: TypeNode;
}

export interface JSDocNamepathType extends JSDocType {
    readonly kind: SyntaxKind.JSDocNamepathType;
    readonly type: TypeNode;
}

export interface JSDoc extends Node {
    readonly kind: SyntaxKind.JSDoc;
    readonly parent: HasJSDoc;
    readonly tags?: NodeArray<JSDocTag>;
    readonly comment?: string | NodeArray<JSDocComment>;
}

export interface JSDocTag extends Node {
    readonly parent: JSDoc | JSDocTypeLiteral;
    readonly tagName: Identifier;
    readonly comment?: string | NodeArray<JSDocComment>;
}

export interface JSDocLink extends Node {
    readonly kind: SyntaxKind.JSDocLink;
    readonly name?: EntityName | JSDocMemberName;
    text: string;
}

export interface JSDocLinkCode extends Node {
    readonly kind: SyntaxKind.JSDocLinkCode;
    readonly name?: EntityName | JSDocMemberName;
    text: string;
}

export interface JSDocLinkPlain extends Node {
    readonly kind: SyntaxKind.JSDocLinkPlain;
    readonly name?: EntityName | JSDocMemberName;
    text: string;
}

export type JSDocComment = JSDocText | JSDocLink | JSDocLinkCode | JSDocLinkPlain;

export interface JSDocText extends Node {
    readonly kind: SyntaxKind.JSDocText;
    text: string;
}

export interface JSDocUnknownTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocTag;
}

/**
 * Note that `@extends` is a synonym of `@augments`.
 * Both tags are represented by this interface.
 */
export interface JSDocAugmentsTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocAugmentsTag;
    readonly class: ExpressionWithTypeArguments & { readonly expression: Identifier | PropertyAccessEntityNameExpression; };
}

export interface JSDocImplementsTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocImplementsTag;
    readonly class: ExpressionWithTypeArguments & { readonly expression: Identifier | PropertyAccessEntityNameExpression; };
}

export interface JSDocAuthorTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocAuthorTag;
}

export interface JSDocDeprecatedTag extends JSDocTag {
    kind: SyntaxKind.JSDocDeprecatedTag;
}

export interface JSDocClassTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocClassTag;
}

export interface JSDocPublicTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocPublicTag;
}

export interface JSDocPrivateTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocPrivateTag;
}

export interface JSDocProtectedTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocProtectedTag;
}

export interface JSDocReadonlyTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocReadonlyTag;
}

export interface JSDocOverrideTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocOverrideTag;
}

export interface JSDocEnumTag extends JSDocTag, Declaration, LocalsContainer {
    readonly kind: SyntaxKind.JSDocEnumTag;
    readonly parent: JSDoc;
    readonly typeExpression: JSDocTypeExpression;
}

export interface JSDocThisTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocThisTag;
    readonly typeExpression: JSDocTypeExpression;
}

export interface JSDocTemplateTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocTemplateTag;
    readonly constraint: JSDocTypeExpression | undefined;
    readonly typeParameters: NodeArray<TypeParameterDeclaration>;
}

export interface JSDocSeeTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocSeeTag;
    readonly name?: JSDocNameReference;
}

export interface JSDocReturnTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocReturnTag;
    readonly typeExpression?: JSDocTypeExpression;
}

export interface JSDocTypeTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocTypeTag;
    readonly typeExpression: JSDocTypeExpression;
}

export interface JSDocTypedefTag extends JSDocTag, NamedDeclaration, LocalsContainer {
    readonly kind: SyntaxKind.JSDocTypedefTag;
    readonly parent: JSDoc;
    readonly fullName?: JSDocNamespaceDeclaration | Identifier;
    readonly name?: Identifier;
    readonly typeExpression?: JSDocTypeExpression | JSDocTypeLiteral;
}

export interface JSDocCallbackTag extends JSDocTag, NamedDeclaration, LocalsContainer {
    readonly kind: SyntaxKind.JSDocCallbackTag;
    readonly parent: JSDoc;
    readonly fullName?: JSDocNamespaceDeclaration | Identifier;
    readonly name?: Identifier;
    readonly typeExpression: JSDocSignature;
}

export interface JSDocOverloadTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocOverloadTag;
    readonly parent: JSDoc;
    readonly typeExpression: JSDocSignature;
}

export interface JSDocThrowsTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocThrowsTag;
    readonly typeExpression?: JSDocTypeExpression;
}

export interface JSDocSignature extends JSDocType, Declaration, JSDocContainer, LocalsContainer {
    readonly kind: SyntaxKind.JSDocSignature;
    readonly typeParameters?: readonly JSDocTemplateTag[];
    readonly parameters: readonly JSDocParameterTag[];
    readonly type: JSDocReturnTag | undefined;
}

export interface JSDocPropertyLikeTag extends JSDocTag, Declaration {
    readonly parent: JSDoc;
    readonly name: EntityName;
    readonly typeExpression?: JSDocTypeExpression;
    /** Whether the property name came before the type -- non-standard for JSDoc, but Typescript-like */
    readonly isNameFirst: boolean;
    readonly isBracketed: boolean;
}

export interface JSDocPropertyTag extends JSDocPropertyLikeTag {
    readonly kind: SyntaxKind.JSDocPropertyTag;
}

export interface JSDocParameterTag extends JSDocPropertyLikeTag {
    readonly kind: SyntaxKind.JSDocParameterTag;
}

export interface JSDocTypeLiteral extends JSDocType, Declaration {
    readonly kind: SyntaxKind.JSDocTypeLiteral;
    readonly jsDocPropertyTags?: readonly JSDocPropertyLikeTag[];
    /** If true, then this type literal represents an *array* of its type. */
    readonly isArrayType: boolean;
}

export interface JSDocSatisfiesTag extends JSDocTag {
    readonly kind: SyntaxKind.JSDocSatisfiesTag;
    readonly typeExpression: JSDocTypeExpression;
}

interface AmdDependency {
    path: string;
    name?: string;
}

/**
 * Subset of properties from SourceFile that are used in multiple utility functions
 */
export interface SourceFileLike {
    readonly text: string;
    /** @internal */
    lineMap?: readonly number[];
    /** @internal */
    getPositionOfLineAndCharacter?(line: number, character: number, allowEdits?: true): number;
}

export type ResolutionMode = ModuleKind.ESNext | ModuleKind.CommonJS | undefined;

// Source files are declarations when they are external modules.
export interface SourceFile extends Declaration, LocalsContainer {
    readonly kind: SyntaxKind.SourceFile;
    readonly statements: NodeArray<Statement>;
    readonly endOfFileToken: Token<SyntaxKind.EndOfFileToken>;

    fileName: string;
    /** @internal */ path: Path;
    text: string;
    /** Resolved path can be different from path property,
     * when file is included through project reference is mapped to its output instead of source
     * in that case resolvedPath = path to output file
     * path = input file's path
     *
     * @internal
     */
    resolvedPath: Path;
    /** Original file name that can be different from fileName,
     * when file is included through project reference is mapped to its output instead of source
     * in that case originalFileName = name of input file
     * fileName = output file's name
     *
     * @internal
     */
    originalFileName: string;

    amdDependencies: readonly AmdDependency[];
    moduleName?: string;
    referencedFiles: readonly FileReference[];
    typeReferenceDirectives: readonly FileReference[];
    libReferenceDirectives: readonly FileReference[];
    languageVariant: LanguageVariant;
    isDeclarationFile: boolean;

    // this map is used by transpiler to supply alternative names for dependencies (i.e. in case of bundling)
    /** @internal */
    renamedDependencies?: ReadonlyMap<string, string>;

    /**
     * lib.d.ts should have a reference comment like
     *
     *  /// <reference no-default-lib="true"/>
     *
     * If any other file has this comment, it signals not to include lib.d.ts
     * because this containing file is intended to act as a default library.
     */
    hasNoDefaultLib: boolean;

    languageVersion: ScriptTarget;

    /**
     * When `module` is `Node16` or `NodeNext`, this field controls whether the
     * source file in question is an ESNext-output-format file, or a CommonJS-output-format
     * module. This is derived by the module resolver as it looks up the file, since
     * it is derived from either the file extension of the module, or the containing
     * `package.json` context, and affects both checking and emit.
     *
     * It is _public_ so that (pre)transformers can set this field,
     * since it switches the builtin `node` module transform. Generally speaking, if unset,
     * the field is treated as though it is `ModuleKind.CommonJS`.
     *
     * Note that this field is only set by the module resolution process when
     * `moduleResolution` is `Node16` or `NodeNext`, which is implied by the `module` setting
     * of `Node16` or `NodeNext`, respectively, but may be overriden (eg, by a `moduleResolution`
     * of `node`). If so, this field will be unset and source files will be considered to be
     * CommonJS-output-format by the node module transformer and type checker, regardless of extension or context.
     */
    impliedNodeFormat?: ResolutionMode;

    /** @internal */ scriptKind: ScriptKind;

    /**
     * The first "most obvious" node that makes a file an external module.
     * This is intended to be the first top-level import/export,
     * but could be arbitrarily nested (e.g. `import.meta`).
     *
     * @internal
     */
    externalModuleIndicator?: Node | true;
    /**
     * The callback used to set the external module indicator - this is saved to
     * be later reused during incremental reparsing, which otherwise lacks the information
     * to set this field
     *
     * @internal
     */
    setExternalModuleIndicator?: (file: SourceFile) => void;
    // The first node that causes this file to be a CommonJS module
    /** @internal */ commonJsModuleIndicator?: Node;
    // JS identifier-declarations that are intended to merge with globals
    /** @internal */ jsGlobalAugmentations?: SymbolTable;

    /** @internal */ identifiers: ReadonlyMap<string, string>; // Map from a string to an interned string
    /** @internal */ nodeCount: number;
    /** @internal */ identifierCount: number;
    /** @internal */ symbolCount: number;

    // File-level diagnostics reported by the parser (includes diagnostics about /// references
    // as well as code diagnostics).
    /** @internal */ parseDiagnostics: DiagnosticWithLocation[];

    // File-level diagnostics reported by the binder.
    /** @internal */ bindDiagnostics: DiagnosticWithLocation[];
    /** @internal */ bindSuggestionDiagnostics?: DiagnosticWithLocation[];

    // File-level JSDoc diagnostics reported by the JSDoc parser
    /** @internal */ jsDocDiagnostics?: DiagnosticWithLocation[];

    // Stores additional file-level diagnostics reported by the program
    /** @internal */ additionalSyntacticDiagnostics?: readonly DiagnosticWithLocation[];

    // Stores a line map for the file.
    // This field should never be used directly to obtain line map, use getLineMap function instead.
    /** @internal */ lineMap: readonly number[];
    /** @internal */ classifiableNames?: ReadonlySet<__String>;
    // Comments containing @ts-* directives, in order.
    /** @internal */ commentDirectives?: CommentDirective[];
    /** @internal */ imports: readonly StringLiteralLike[];
    // Identifier only if `declare global`
    /** @internal */ moduleAugmentations: readonly (StringLiteral | Identifier)[];
    /** @internal */ patternAmbientModules?: PatternAmbientModule[];
    /** @internal */ ambientModuleNames: readonly string[];
    /** @internal */ checkJsDirective?: CheckJsDirective;
    /** @internal */ version: string;
    /** @internal */ pragmas: ReadonlyPragmaMap;
    /** @internal */ localJsxNamespace?: __String;
    /** @internal */ localJsxFragmentNamespace?: __String;
    /** @internal */ localJsxFactory?: EntityName;
    /** @internal */ localJsxFragmentFactory?: EntityName;

    /** @internal */ exportedModulesFromDeclarationEmit?: ExportedModulesFromDeclarationEmit;

    /** @internal */ jsDocParsingMode?: JSDocParsingMode;
}

/** @internal */
interface ReadonlyPragmaContext {
    languageVersion: ScriptTarget;
    pragmas?: ReadonlyPragmaMap;
    checkJsDirective?: CheckJsDirective;
    referencedFiles: readonly FileReference[];
    typeReferenceDirectives: readonly FileReference[];
    libReferenceDirectives: readonly FileReference[];
    amdDependencies: readonly AmdDependency[];
    hasNoDefaultLib?: boolean;
    moduleName?: string;
}

/** @internal */
export interface PragmaContext extends ReadonlyPragmaContext {
    pragmas?: PragmaMap;
    referencedFiles: FileReference[];
    typeReferenceDirectives: FileReference[];
    libReferenceDirectives: FileReference[];
    amdDependencies: AmdDependency[];
}

/** @internal */
export interface SourceFile extends ReadonlyPragmaContext {}

/** @internal */
export interface CommentDirective {
    range: TextRange;
    type: CommentDirectiveType;
}

/** @internal */
export const enum CommentDirectiveType {
    ExpectError,
    Ignore,
}

/** @internal */
type ExportedModulesFromDeclarationEmit = readonly Symbol[];

/** @deprecated */
interface UnparsedSource extends Node {
    readonly kind: SyntaxKind.UnparsedSource;
    fileName: string;
    text: string;
    readonly prologues: readonly UnparsedPrologue[];
    helpers: readonly UnscopedEmitHelper[] | undefined;

    // References and noDefaultLibAre Dts only
    referencedFiles: readonly FileReference[];
    typeReferenceDirectives: readonly FileReference[] | undefined;
    libReferenceDirectives: readonly FileReference[];
    hasNoDefaultLib?: boolean;

    sourceMapPath?: string;
    sourceMapText?: string;
    readonly syntheticReferences?: readonly UnparsedSyntheticReference[];
    readonly texts: readonly UnparsedSourceText[];
    /** @internal */ oldFileOfCurrentEmit?: boolean;
    /** @internal */ parsedSourceMap?: RawSourceMap | false | undefined;
    // Adding this to satisfy services, fix later
    /** @internal */
    getLineAndCharacterOfPosition(pos: number): LineAndCharacter;
}

/** @deprecated */
type UnparsedSourceText =
    | UnparsedPrepend
    | UnparsedTextLike;

/** @deprecated */
interface UnparsedSection extends Node {
    readonly kind: SyntaxKind;
    readonly parent: UnparsedSource;
    readonly data?: string;
}

/** @deprecated */
interface UnparsedPrologue extends UnparsedSection {
    readonly kind: SyntaxKind.UnparsedPrologue;
    readonly parent: UnparsedSource;
    readonly data: string;
}

/** @deprecated */
interface UnparsedPrepend extends UnparsedSection {
    readonly kind: SyntaxKind.UnparsedPrepend;
    readonly parent: UnparsedSource;
    readonly data: string;
    readonly texts: readonly UnparsedTextLike[];
}

/** @deprecated */
interface UnparsedTextLike extends UnparsedSection {
    readonly kind: SyntaxKind.UnparsedText | SyntaxKind.UnparsedInternalText;
    readonly parent: UnparsedSource;
}

/** @deprecated */
interface UnparsedSyntheticReference extends UnparsedSection {
    readonly kind: SyntaxKind.UnparsedSyntheticReference;
    readonly parent: UnparsedSource;
    /** @internal */ readonly section: BundleFileHasNoDefaultLib | BundleFileReference;
}

export interface JsonSourceFile extends SourceFile {
    readonly statements: NodeArray<JsonObjectExpressionStatement>;
}

interface TsConfigSourceFile extends JsonSourceFile {
    extendedSourceFiles?: string[];
    /** @internal */ configFileSpecs?: ConfigFileSpecs;
}

export interface JsonMinusNumericLiteral extends PrefixUnaryExpression {
    readonly kind: SyntaxKind.PrefixUnaryExpression;
    readonly operator: SyntaxKind.MinusToken;
    readonly operand: NumericLiteral;
}

type JsonObjectExpression =
    | ObjectLiteralExpression
    | ArrayLiteralExpression
    | JsonMinusNumericLiteral
    | NumericLiteral
    | StringLiteral
    | BooleanLiteral
    | NullLiteral;

export interface JsonObjectExpressionStatement extends ExpressionStatement {
    readonly expression: JsonObjectExpression;
}

const enum InternalSymbolName {
    Call = "__call", // Call signatures
    Constructor = "__constructor", // Constructor implementations
    New = "__new", // Constructor signatures
    Index = "__index", // Index signatures
    ExportStar = "__export", // Module export * declarations
    Global = "__global", // Global self-reference
    Missing = "__missing", // Indicates missing symbol
    Type = "__type", // Anonymous type literal symbol
    Object = "__object", // Anonymous object literal declaration
    JSXAttributes = "__jsxAttributes", // Anonymous JSX attributes object literal declaration
    Class = "__class", // Unnamed class expression
    Function = "__function", // Unnamed function expression
    Computed = "__computed", // Computed property name declaration with dynamic name
    Resolving = "__resolving__", // Indicator symbol used to mark partially resolved type aliases
    ExportEquals = "export=", // Export assignment symbol
    Default = "default", // Default export symbol (technically not wholly internal, but included here for usability)
    This = "this",
    InstantiationExpression = "__instantiationExpression", // Instantiation expressions
    ImportAttributes = "__importAttributes",
}

/**
 * This represents a string whose leading underscore have been escaped by adding extra leading underscores.
 * The shape of this brand is rather unique compared to others we've used.
 * Instead of just an intersection of a string and an object, it is that union-ed
 * with an intersection of void and an object. This makes it wholly incompatible
 * with a normal string (which is good, it cannot be misused on assignment or on usage),
 * while still being comparable with a normal string via === (also good) and castable from a string.
 */
export type __String = (string & { __escapedIdentifier: void; }) | (void & { __escapedIdentifier: void; }) | InternalSymbolName;

/** SymbolTable based on ES6 Map interface. */
type SymbolTable = Map<__String, Symbol>;

/**
 * Used to track a `declare module "foo*"`-like declaration.
 *
 * @internal
 */
interface PatternAmbientModule {
    pattern: Pattern;
    symbol: Symbol;
}

/** @internal */
export const enum AssignmentDeclarationKind {
    None,
    /// exports.name = expr
    /// module.exports.name = expr
    ExportsProperty,
    /// module.exports = expr
    ModuleExports,
    /// className.prototype.name = expr
    PrototypeProperty,
    /// this.name = expr
    ThisProperty,
    // F.name = expr
    Property,
    // F.prototype = { ... }
    Prototype,
    // Object.defineProperty(x, 'name', { value: any, writable?: boolean (false by default) });
    // Object.defineProperty(x, 'name', { get: Function, set: Function });
    // Object.defineProperty(x, 'name', { get: Function });
    // Object.defineProperty(x, 'name', { set: Function });
    ObjectDefinePropertyValue,
    // Object.defineProperty(exports || module.exports, 'name', ...);
    ObjectDefinePropertyExports,
    // Object.defineProperty(Foo.prototype, 'name', ...);
    ObjectDefinePrototypeProperty,
}

export interface DiagnosticMessage {
    key: string;
    category: DiagnosticCategory;
    code: number;
    message: string;
    reportsUnnecessary?: {};
    reportsDeprecated?: {};
    /** @internal */
    elidedInCompatabilityPyramid?: boolean;
}

/** @internal */
interface RepopulateModuleNotFoundDiagnosticChain {
    moduleReference: string;
    mode: ResolutionMode;
    packageName: string | undefined;
}

/** @internal */
type RepopulateDiagnosticChainInfo = RepopulateModuleNotFoundDiagnosticChain;

/**
 * A linked list of formatted diagnostic messages to be used as part of a multiline message.
 * It is built from the bottom up, leaving the head to be the "main" diagnostic.
 * While it seems that DiagnosticMessageChain is structurally similar to DiagnosticMessage,
 * the difference is that messages are all preformatted in DMC.
 */
interface DiagnosticMessageChain {
    messageText: string;
    category: DiagnosticCategory;
    code: number;
    next?: DiagnosticMessageChain[];
    /** @internal */
    repopulateInfo?: () => RepopulateDiagnosticChainInfo;
}

export interface Diagnostic extends DiagnosticRelatedInformation {
    /** May store more in future. For now, this will simply be `true` to indicate when a diagnostic is an unused-identifier diagnostic. */
    reportsUnnecessary?: {};

    reportsDeprecated?: {};
    source?: string;
    relatedInformation?: DiagnosticRelatedInformation[];
    /** @internal */ skippedOn?: keyof CompilerOptions;
}

/** @internal */
export type DiagnosticArguments = (string | number)[];

export interface DiagnosticRelatedInformation {
    category: DiagnosticCategory;
    code: number;
    file: SourceFile | undefined;
    start: number | undefined;
    length: number | undefined;
    messageText: string | DiagnosticMessageChain;
}

export interface DiagnosticWithLocation extends Diagnostic {
    file: SourceFile;
    start: number;
    length: number;
}

/** @internal */
export interface DiagnosticWithDetachedLocation extends Diagnostic {
    file: undefined;
    fileName: string;
    start: number;
    length: number;
}

export enum DiagnosticCategory {
    Warning,
    Error,
    Suggestion,
    Message,
}

enum ModuleResolutionKind {
    Classic = 1,
    /**
     * @deprecated
     * `NodeJs` was renamed to `Node10` to better reflect the version of Node that it targets.
     * Use the new name or consider switching to a modern module resolution target.
     */
    NodeJs = 2,
    Node10 = 2,
    // Starting with node12, node's module resolver has significant departures from traditional cjs resolution
    // to better support ECMAScript modules and their use within node - however more features are still being added.
    // TypeScript's Node ESM support was introduced after Node 12 went end-of-life, and Node 14 is the earliest stable
    // version that supports both pattern trailers - *but*, Node 16 is the first version that also supports ECMAScript 2022.
    // In turn, we offer both a `NodeNext` moving resolution target, and a `Node16` version-anchored resolution target
    Node16 = 3,
    NodeNext = 99, // Not simply `Node16` so that compiled code linked against TS can use the `Next` value reliably (same as with `ModuleKind`)
    Bundler = 100,
}

enum ModuleDetectionKind {
    /**
     * Files with imports, exports and/or import.meta are considered modules
     */
    Legacy = 1,
    /**
     * Legacy, but also files with jsx under react-jsx or react-jsxdev and esm mode files under moduleResolution: node16+
     */
    Auto = 2,
    /**
     * Consider all non-declaration files modules, regardless of present syntax
     */
    Force = 3,
}

interface PluginImport {
    name: string;
}

interface ProjectReference {
    /** A normalized path on disk */
    path: string;
    /** The path as the user originally wrote it */
    originalPath?: string;
    /** True if the output of this reference should be prepended to the output of this project. Only valid for --outFile compilations */
    prepend?: boolean;
    /** True if it is intended that this reference form a circularity */
    circular?: boolean;
}

type CompilerOptionsValue = string | number | boolean | (string | number)[] | string[] | MapLike<string[]> | PluginImport[] | ProjectReference[] | null | undefined;

interface CompilerOptions {
    /** @internal */ all?: boolean;
    allowImportingTsExtensions?: boolean;
    allowJs?: boolean;
    /** @internal */ allowNonTsExtensions?: boolean;
    allowArbitraryExtensions?: boolean;
    allowSyntheticDefaultImports?: boolean;
    allowUmdGlobalAccess?: boolean;
    allowUnreachableCode?: boolean;
    allowUnusedLabels?: boolean;
    alwaysStrict?: boolean; // Always combine with strict property
    baseUrl?: string;
    /**
     * An error if set - this should only go through the -b pipeline and not actually be observed
     *
     * @internal
     */
    build?: boolean;
    charset?: string;
    checkJs?: boolean;
    /** @internal */ configFilePath?: string;
    /**
     * configFile is set as non enumerable property so as to avoid checking of json source files
     *
     * @internal
     */
    readonly configFile?: TsConfigSourceFile;
    customConditions?: string[];
    declaration?: boolean;
    declarationMap?: boolean;
    emitDeclarationOnly?: boolean;
    declarationDir?: string;
    /** @internal */ diagnostics?: boolean;
    /** @internal */ extendedDiagnostics?: boolean;
    disableSizeLimit?: boolean;
    disableSourceOfProjectReferenceRedirect?: boolean;
    disableSolutionSearching?: boolean;
    disableReferencedProjectLoad?: boolean;
    downlevelIteration?: boolean;
    emitBOM?: boolean;
    emitDecoratorMetadata?: boolean;
    exactOptionalPropertyTypes?: boolean;
    experimentalDecorators?: boolean;
    forceConsistentCasingInFileNames?: boolean;
    /** @internal */ generateCpuProfile?: string;
    /** @internal */ generateTrace?: string;
    /** @internal */ help?: boolean;
    ignoreDeprecations?: string;
    importHelpers?: boolean;
    importsNotUsedAsValues?: ImportsNotUsedAsValues;
    /** @internal */ init?: boolean;
    inlineSourceMap?: boolean;
    inlineSources?: boolean;
    isolatedModules?: boolean;
    jsx?: JsxEmit;
    keyofStringsOnly?: boolean;
    lib?: string[];
    /** @internal */ listEmittedFiles?: boolean;
    /** @internal */ listFiles?: boolean;
    /** @internal */ explainFiles?: boolean;
    /** @internal */ listFilesOnly?: boolean;
    locale?: string;
    mapRoot?: string;
    maxNodeModuleJsDepth?: number;
    module?: ModuleKind;
    moduleResolution?: ModuleResolutionKind;
    moduleSuffixes?: string[];
    moduleDetection?: ModuleDetectionKind;
    newLine?: NewLineKind;
    noEmit?: boolean;
    /** @internal */ noEmitForJsFiles?: boolean;
    noEmitHelpers?: boolean;
    noEmitOnError?: boolean;
    noErrorTruncation?: boolean;
    noFallthroughCasesInSwitch?: boolean;
    noImplicitAny?: boolean; // Always combine with strict property
    noImplicitReturns?: boolean;
    noImplicitThis?: boolean; // Always combine with strict property
    noStrictGenericChecks?: boolean;
    noUnusedLocals?: boolean;
    noUnusedParameters?: boolean;
    noImplicitUseStrict?: boolean;
    noPropertyAccessFromIndexSignature?: boolean;
    assumeChangesOnlyAffectDirectDependencies?: boolean;
    noLib?: boolean;
    noResolve?: boolean;
    /** @internal */
    noDtsResolution?: boolean;
    noUncheckedIndexedAccess?: boolean;
    out?: string;
    outDir?: string;
    outFile?: string;
    paths?: MapLike<string[]>;
    /**
     * The directory of the config file that specified 'paths'. Used to resolve relative paths when 'baseUrl' is absent.
     *
     * @internal
     */
    pathsBasePath?: string;
    /** @internal */ plugins?: PluginImport[];
    preserveConstEnums?: boolean;
    noImplicitOverride?: boolean;
    preserveSymlinks?: boolean;
    preserveValueImports?: boolean;
    /** @internal */ preserveWatchOutput?: boolean;
    project?: string;
    /** @internal */ pretty?: boolean;
    reactNamespace?: string;
    jsxFactory?: string;
    jsxFragmentFactory?: string;
    jsxImportSource?: string;
    composite?: boolean;
    incremental?: boolean;
    tsBuildInfoFile?: string;
    removeComments?: boolean;
    resolvePackageJsonExports?: boolean;
    resolvePackageJsonImports?: boolean;
    rootDir?: string;
    rootDirs?: string[];
    skipLibCheck?: boolean;
    skipDefaultLibCheck?: boolean;
    sourceMap?: boolean;
    sourceRoot?: string;
    strict?: boolean;
    strictFunctionTypes?: boolean; // Always combine with strict property
    strictBindCallApply?: boolean; // Always combine with strict property
    strictNullChecks?: boolean; // Always combine with strict property
    strictPropertyInitialization?: boolean; // Always combine with strict property
    stripInternal?: boolean;
    suppressExcessPropertyErrors?: boolean;
    suppressImplicitAnyIndexErrors?: boolean;
    /** @internal */ suppressOutputPathCheck?: boolean;
    target?: ScriptTarget;
    traceResolution?: boolean;
    useUnknownInCatchVariables?: boolean;
    resolveJsonModule?: boolean;
    types?: string[];
    /** Paths used to compute primary types search locations */
    typeRoots?: string[];
    verbatimModuleSyntax?: boolean;
    /** @internal */ version?: boolean;
    /** @internal */ watch?: boolean;
    esModuleInterop?: boolean;
    /** @internal */ showConfig?: boolean;
    useDefineForClassFields?: boolean;

    [option: string]: CompilerOptionsValue | TsConfigSourceFile | undefined;
}

export enum ModuleKind {
    None = 0,
    CommonJS = 1,
    AMD = 2,
    UMD = 3,
    System = 4,

    // NOTE: ES module kinds should be contiguous to more easily check whether a module kind is *any* ES module kind.
    //       Non-ES module kinds should not come between ES2015 (the earliest ES module kind) and ESNext (the last ES
    //       module kind).
    ES2015 = 5,
    ES2020 = 6,
    ES2022 = 7,
    ESNext = 99,

    // Node16+ is an amalgam of commonjs (albeit updated) and es2022+, and represents a distinct module system from es2020/esnext
    Node16 = 100,
    NodeNext = 199,

    // Emit as written
    Preserve = 200,
}

const enum JsxEmit {
    None = 0,
    Preserve = 1,
    React = 2,
    ReactNative = 3,
    ReactJSX = 4,
    ReactJSXDev = 5,
}

const enum ImportsNotUsedAsValues {
    Remove,
    Preserve,
    Error,
}

const enum NewLineKind {
    CarriageReturnLineFeed = 0,
    LineFeed = 1,
}

export interface LineAndCharacter {
    /** 0-based. */
    line: number;
    /*
     * 0-based. This value denotes the character position in line and is different from the 'column' because of tab characters.
     */
    character: number;
}

export const enum ScriptKind {
    Unknown = 0,
    JS = 1,
    JSX = 2,
    TS = 3,
    TSX = 4,
    External = 5,
    JSON = 6,
    /**
     * Used on extensions that doesn't define the ScriptKind but the content defines it.
     * Deferred extensions are going to be included in all project contexts.
     */
    Deferred = 7,
}

export const enum ScriptTarget {
    ES3 = 0,
    ES5 = 1,
    ES2015 = 2,
    ES2016 = 3,
    ES2017 = 4,
    ES2018 = 5,
    ES2019 = 6,
    ES2020 = 7,
    ES2021 = 8,
    ES2022 = 9,
    ESNext = 99,
    JSON = 100,
    Latest = ESNext,
}

export const enum LanguageVariant {
    Standard,
    JSX,
}

/** @internal */
interface ConfigFileSpecs {
    filesSpecs: readonly string[] | undefined;
    /**
     * Present to report errors (user specified specs), validatedIncludeSpecs are used for file name matching
     */
    includeSpecs: readonly string[] | undefined;
    /**
     * Present to report errors (user specified specs), validatedExcludeSpecs are used for file name matching
     */
    excludeSpecs: readonly string[] | undefined;
    validatedFilesSpec: readonly string[] | undefined;
    validatedIncludeSpecs: readonly string[] | undefined;
    validatedExcludeSpecs: readonly string[] | undefined;
    pathPatterns: readonly (string | Pattern)[] | undefined;
    isDefaultIncludeSpec: boolean;
}

// dprint-ignore
/** @internal */
interface CommandLineOptionBase {
    name: string;
    type: "string" | "number" | "boolean" | "object" | "list" | "listOrElement" | Map<string, number | string>;    // a value of a primitive type, or an object literal mapping named values to actual values
    isFilePath?: boolean;                                   // True if option value is a path or fileName
    shortName?: string;                                     // A short mnemonic for convenience - for instance, 'h' can be used in place of 'help'
    description?: DiagnosticMessage;                        // The message describing what the command line switch does.
    defaultValueDescription?: string | number | boolean | DiagnosticMessage | undefined;   // The message describing what the dafault value is. string type is prepared for fixed chosen like "false" which do not need I18n.
    paramType?: DiagnosticMessage;                          // The name to be used for a non-boolean option's parameter
    isTSConfigOnly?: boolean;                               // True if option can only be specified via tsconfig.json file
    isCommandLineOnly?: boolean;
    showInSimplifiedHelpView?: boolean;
    category?: DiagnosticMessage;
    strictFlag?: true;                                      // true if the option is one of the flag under strict
    allowJsFlag?: true;
    affectsSourceFile?: true;                               // true if we should recreate SourceFiles after this option changes
    affectsModuleResolution?: true;                         // currently same effect as `affectsSourceFile`
    affectsBindDiagnostics?: true;                          // true if this affects binding (currently same effect as `affectsSourceFile`)
    affectsSemanticDiagnostics?: true;                      // true if option affects semantic diagnostics
    affectsEmit?: true;                                     // true if the options affects emit
    affectsProgramStructure?: true;                         // true if program should be reconstructed from root files if option changes and does not affect module resolution as affectsModuleResolution indirectly means program needs to reconstructed
    affectsDeclarationPath?: true;                          // true if the options affects declaration file path computed
    affectsBuildInfo?: true;                                // true if this options should be emitted in buildInfo
    transpileOptionValue?: boolean | undefined;             // If set this means that the option should be set to this value when transpiling
    extraValidation?: (value: CompilerOptionsValue) => [DiagnosticMessage, ...string[]] | undefined; // Additional validation to be performed for the value to be valid
    disallowNullOrUndefined?: true;                                    // If set option does not allow setting null
}

/** @internal */
interface CommandLineOptionOfStringType extends CommandLineOptionBase {
    type: "string";
    defaultValueDescription?: string | undefined | DiagnosticMessage;
}

/** @internal */
interface CommandLineOptionOfNumberType extends CommandLineOptionBase {
    type: "number";
    defaultValueDescription: number | undefined | DiagnosticMessage;
}

/** @internal */
interface CommandLineOptionOfBooleanType extends CommandLineOptionBase {
    type: "boolean";
    defaultValueDescription: boolean | undefined | DiagnosticMessage;
}

/** @internal */
interface CommandLineOptionOfCustomType extends CommandLineOptionBase {
    type: Map<string, number | string>; // an object literal mapping named values to actual values
    defaultValueDescription: number | string | undefined | DiagnosticMessage;
    deprecatedKeys?: Set<string>;
}

/** @internal */
interface AlternateModeDiagnostics {
    diagnostic: DiagnosticMessage;
    getOptionsNameMap: () => OptionsNameMap;
}

/** @internal */
interface DidYouMeanOptionsDiagnostics {
    alternateMode?: AlternateModeDiagnostics;
    optionDeclarations: CommandLineOption[];
    unknownOptionDiagnostic: DiagnosticMessage;
    unknownDidYouMeanDiagnostic: DiagnosticMessage;
}

/** @internal */
export interface TsConfigOnlyOption extends CommandLineOptionBase {
    type: "object";
    elementOptions?: Map<string, CommandLineOption>;
    extraKeyDiagnostics?: DidYouMeanOptionsDiagnostics;
}

/** @internal */
export interface CommandLineOptionOfListType extends CommandLineOptionBase {
    type: "list" | "listOrElement";
    element: CommandLineOptionOfCustomType | CommandLineOptionOfStringType | CommandLineOptionOfNumberType | CommandLineOptionOfBooleanType | TsConfigOnlyOption;
    listPreserveFalsyValues?: boolean;
}

/** @internal */
export type CommandLineOption = CommandLineOptionOfCustomType | CommandLineOptionOfStringType | CommandLineOptionOfNumberType | CommandLineOptionOfBooleanType | TsConfigOnlyOption | CommandLineOptionOfListType;

// dprint-ignore
/** @internal */
export const enum CharacterCodes {
    nullCharacter = 0,
    maxAsciiCharacter = 0x7F,

    lineFeed = 0x0A,              // \n
    carriageReturn = 0x0D,        // \r
    lineSeparator = 0x2028,
    paragraphSeparator = 0x2029,
    nextLine = 0x0085,

    // Unicode 3.0 space characters
    space = 0x0020,   // " "
    nonBreakingSpace = 0x00A0,   //
    enQuad = 0x2000,
    emQuad = 0x2001,
    enSpace = 0x2002,
    emSpace = 0x2003,
    threePerEmSpace = 0x2004,
    fourPerEmSpace = 0x2005,
    sixPerEmSpace = 0x2006,
    figureSpace = 0x2007,
    punctuationSpace = 0x2008,
    thinSpace = 0x2009,
    hairSpace = 0x200A,
    zeroWidthSpace = 0x200B,
    narrowNoBreakSpace = 0x202F,
    ideographicSpace = 0x3000,
    mathematicalSpace = 0x205F,
    ogham = 0x1680,

    _ = 0x5F,
    $ = 0x24,

    _0 = 0x30,
    _1 = 0x31,
    _2 = 0x32,
    _3 = 0x33,
    _4 = 0x34,
    _5 = 0x35,
    _6 = 0x36,
    _7 = 0x37,
    _8 = 0x38,
    _9 = 0x39,

    a = 0x61,
    b = 0x62,
    c = 0x63,
    d = 0x64,
    e = 0x65,
    f = 0x66,
    g = 0x67,
    h = 0x68,
    i = 0x69,
    j = 0x6A,
    k = 0x6B,
    l = 0x6C,
    m = 0x6D,
    n = 0x6E,
    o = 0x6F,
    p = 0x70,
    q = 0x71,
    r = 0x72,
    s = 0x73,
    t = 0x74,
    u = 0x75,
    v = 0x76,
    w = 0x77,
    x = 0x78,
    y = 0x79,
    z = 0x7A,

    A = 0x41,
    B = 0x42,
    C = 0x43,
    D = 0x44,
    E = 0x45,
    F = 0x46,
    G = 0x47,
    H = 0x48,
    I = 0x49,
    J = 0x4A,
    K = 0x4B,
    L = 0x4C,
    M = 0x4D,
    N = 0x4E,
    O = 0x4F,
    P = 0x50,
    Q = 0x51,
    R = 0x52,
    S = 0x53,
    T = 0x54,
    U = 0x55,
    V = 0x56,
    W = 0x57,
    X = 0x58,
    Y = 0x59,
    Z = 0x5a,

    ampersand = 0x26,             // &
    asterisk = 0x2A,              // *
    at = 0x40,                    // @
    backslash = 0x5C,             // \
    backtick = 0x60,              // `
    bar = 0x7C,                   // |
    caret = 0x5E,                 // ^
    closeBrace = 0x7D,            // }
    closeBracket = 0x5D,          // ]
    closeParen = 0x29,            // )
    colon = 0x3A,                 // :
    comma = 0x2C,                 // ,
    dot = 0x2E,                   // .
    doubleQuote = 0x22,           // "
    equals = 0x3D,                // =
    exclamation = 0x21,           // !
    greaterThan = 0x3E,           // >
    hash = 0x23,                  // #
    lessThan = 0x3C,              // <
    minus = 0x2D,                 // -
    openBrace = 0x7B,             // {
    openBracket = 0x5B,           // [
    openParen = 0x28,             // (
    percent = 0x25,               // %
    plus = 0x2B,                  // +
    question = 0x3F,              // ?
    semicolon = 0x3B,             // ;
    singleQuote = 0x27,           // '
    slash = 0x2F,                 // /
    tilde = 0x7E,                 // ~

    backspace = 0x08,             // \b
    formFeed = 0x0C,              // \f
    byteOrderMark = 0xFEFF,
    tab = 0x09,                   // \t
    verticalTab = 0x0B,           // \v
}

export const enum Extension {
    Ts = ".ts",
    Tsx = ".tsx",
    Dts = ".d.ts",
    Js = ".js",
    Jsx = ".jsx",
    Json = ".json",
    TsBuildInfo = ".tsbuildinfo",
    Mjs = ".mjs",
    Mts = ".mts",
    Dmts = ".d.mts",
    Cjs = ".cjs",
    Cts = ".cts",
    Dcts = ".d.cts",
}

/** @internal */
export const enum TransformFlags {
    None = 0,

    // Facts
    // - Flags used to indicate that a node or subtree contains syntax that requires transformation.
    ContainsTypeScript = 1 << 0,
    ContainsJsx = 1 << 1,
    ContainsESNext = 1 << 2,
    ContainsES2022 = 1 << 3,
    ContainsES2021 = 1 << 4,
    ContainsES2020 = 1 << 5,
    ContainsES2019 = 1 << 6,
    ContainsES2018 = 1 << 7,
    ContainsES2017 = 1 << 8,
    ContainsES2016 = 1 << 9,
    ContainsES2015 = 1 << 10,
    ContainsGenerator = 1 << 11,
    ContainsDestructuringAssignment = 1 << 12,

    // Markers
    // - Flags used to indicate that a subtree contains a specific transformation.
    ContainsTypeScriptClassSyntax = 1 << 13, // Property Initializers, Parameter Property Initializers
    ContainsLexicalThis = 1 << 14,
    ContainsRestOrSpread = 1 << 15,
    ContainsObjectRestOrSpread = 1 << 16,
    ContainsComputedPropertyName = 1 << 17,
    ContainsBlockScopedBinding = 1 << 18,
    ContainsBindingPattern = 1 << 19,
    ContainsYield = 1 << 20,
    ContainsAwait = 1 << 21,
    ContainsHoistedDeclarationOrCompletion = 1 << 22,
    ContainsDynamicImport = 1 << 23,
    ContainsClassFields = 1 << 24,
    ContainsDecorators = 1 << 25,
    ContainsPossibleTopLevelAwait = 1 << 26,
    ContainsLexicalSuper = 1 << 27,
    ContainsUpdateExpressionForIdentifier = 1 << 28,
    ContainsPrivateIdentifierInExpression = 1 << 29,

    HasComputedFlags = 1 << 31, // Transform flags have been computed.

    // Assertions
    // - Bitmasks that are used to assert facts about the syntax of a node and its subtree.
    AssertTypeScript = ContainsTypeScript,
    AssertJsx = ContainsJsx,
    AssertESNext = ContainsESNext,
    AssertES2022 = ContainsES2022,
    AssertES2021 = ContainsES2021,
    AssertES2020 = ContainsES2020,
    AssertES2019 = ContainsES2019,
    AssertES2018 = ContainsES2018,
    AssertES2017 = ContainsES2017,
    AssertES2016 = ContainsES2016,
    AssertES2015 = ContainsES2015,
    AssertGenerator = ContainsGenerator,
    AssertDestructuringAssignment = ContainsDestructuringAssignment,

    // Scope Exclusions
    // - Bitmasks that exclude flags from propagating out of a specific context
    //   into the subtree flags of their container.
    OuterExpressionExcludes = HasComputedFlags,
    PropertyAccessExcludes = OuterExpressionExcludes,
    NodeExcludes = PropertyAccessExcludes,
    ArrowFunctionExcludes = NodeExcludes | ContainsTypeScriptClassSyntax | ContainsBlockScopedBinding | ContainsYield | ContainsAwait | ContainsHoistedDeclarationOrCompletion | ContainsBindingPattern | ContainsObjectRestOrSpread | ContainsPossibleTopLevelAwait,
    FunctionExcludes = NodeExcludes | ContainsTypeScriptClassSyntax | ContainsLexicalThis | ContainsLexicalSuper | ContainsBlockScopedBinding | ContainsYield | ContainsAwait | ContainsHoistedDeclarationOrCompletion | ContainsBindingPattern | ContainsObjectRestOrSpread | ContainsPossibleTopLevelAwait,
    ConstructorExcludes = NodeExcludes | ContainsLexicalThis | ContainsLexicalSuper | ContainsBlockScopedBinding | ContainsYield | ContainsAwait | ContainsHoistedDeclarationOrCompletion | ContainsBindingPattern | ContainsObjectRestOrSpread | ContainsPossibleTopLevelAwait,
    MethodOrAccessorExcludes = NodeExcludes | ContainsLexicalThis | ContainsLexicalSuper | ContainsBlockScopedBinding | ContainsYield | ContainsAwait | ContainsHoistedDeclarationOrCompletion | ContainsBindingPattern | ContainsObjectRestOrSpread,
    PropertyExcludes = NodeExcludes | ContainsLexicalThis | ContainsLexicalSuper,
    ClassExcludes = NodeExcludes | ContainsTypeScriptClassSyntax | ContainsComputedPropertyName,
    ModuleExcludes = NodeExcludes | ContainsTypeScriptClassSyntax | ContainsLexicalThis | ContainsLexicalSuper | ContainsBlockScopedBinding | ContainsHoistedDeclarationOrCompletion | ContainsPossibleTopLevelAwait,
    TypeExcludes = ~ContainsTypeScript,
    ObjectLiteralExcludes = NodeExcludes | ContainsTypeScriptClassSyntax | ContainsComputedPropertyName | ContainsObjectRestOrSpread,
    ArrayLiteralOrCallOrNewExcludes = NodeExcludes | ContainsRestOrSpread,
    VariableDeclarationListExcludes = NodeExcludes | ContainsBindingPattern | ContainsObjectRestOrSpread,
    ParameterExcludes = NodeExcludes,
    CatchClauseExcludes = NodeExcludes | ContainsObjectRestOrSpread,
    BindingPatternExcludes = NodeExcludes | ContainsRestOrSpread,
    ContainsLexicalThisOrSuper = ContainsLexicalThis | ContainsLexicalSuper,

    // Propagating flags
    // - Bitmasks for flags that should propagate from a child
    PropertyNamePropagatingFlags = ContainsLexicalThis | ContainsLexicalSuper,
    // Masks
    // - Additional bitmasks
}

export interface SourceMapRange extends TextRange {
    source?: SourceMapSource;
}

export interface SourceMapSource {
    fileName: string;
    text: string;
    /** @internal */ lineMap: readonly number[];
    skipTrivia?: (pos: number) => number;
}

/** @internal */
// NOTE: Any new properties should be accounted for in `mergeEmitNode` in factory/nodeFactory.ts
// dprint-ignore
export interface EmitNode {
    flags: EmitFlags;                        // Flags that customize emit
    internalFlags: InternalEmitFlags;        // Internal flags that customize emit
    annotatedNodes?: Node[];                 // Tracks Parse-tree nodes with EmitNodes for eventual cleanup.
    leadingComments?: SynthesizedComment[];  // Synthesized leading comments
    trailingComments?: SynthesizedComment[]; // Synthesized trailing comments
    commentRange?: TextRange;                // The text range to use when emitting leading or trailing comments
    sourceMapRange?: SourceMapRange;         // The text range to use when emitting leading or trailing source mappings
    tokenSourceMapRanges?: (SourceMapRange | undefined)[]; // The text range to use when emitting source mappings for tokens
    constantValue?: string | number;         // The constant value of an expression
    externalHelpersModuleName?: Identifier;  // The local name for an imported helpers module
    externalHelpers?: boolean;
    helpers?: EmitHelper[];                  // Emit helpers for the node
    startsOnNewLine?: boolean;               // If the node should begin on a new line
    snippetElement?: SnippetElement;         // Snippet element of the node
    typeNode?: TypeNode;                     // VariableDeclaration type
    classThis?: Identifier;                  // Identifier that points to a captured static `this` for a class which may be updated after decorators are applied
    assignedName?: Expression;               // Expression used as the assigned name of a class or function
    identifierTypeArguments?: NodeArray<TypeNode | TypeParameterDeclaration>; // Only defined on synthesized identifiers. Though not syntactically valid, used in emitting diagnostics, quickinfo, and signature help.
    autoGenerate: AutoGenerateInfo | undefined; // Used for auto-generated identifiers and private identifiers.
    generatedImportReference?: ImportSpecifier; // Reference to the generated import specifier this identifier refers to
}

/** @internal */
type SnippetElement = TabStop | Placeholder;

/** @internal */
interface TabStop {
    kind: SnippetKind.TabStop;
    order: number;
}

/** @internal */
interface Placeholder {
    kind: SnippetKind.Placeholder;
    order: number;
}

// Reference: https://code.visualstudio.com/docs/editor/userdefinedsnippets#_snippet-syntax
// dprint-ignore
/** @internal */
const enum SnippetKind {
    TabStop,                                // `$1`, `$2`
    Placeholder,                            // `${1:foo}`
    Choice,                                 // `${1|one,two,three|}`
    Variable,                               // `$name`, `${name:default}`
}

// dprint-ignore
export const enum EmitFlags {
    None = 0,
    SingleLine = 1 << 0,                    // The contents of this node should be emitted on a single line.
    MultiLine = 1 << 1,
    AdviseOnEmitNode = 1 << 2,              // The printer should invoke the onEmitNode callback when printing this node.
    NoSubstitution = 1 << 3,                // Disables further substitution of an expression.
    CapturesThis = 1 << 4,                  // The function captures a lexical `this`
    NoLeadingSourceMap = 1 << 5,            // Do not emit a leading source map location for this node.
    NoTrailingSourceMap = 1 << 6,           // Do not emit a trailing source map location for this node.
    NoSourceMap = NoLeadingSourceMap | NoTrailingSourceMap, // Do not emit a source map location for this node.
    NoNestedSourceMaps = 1 << 7,            // Do not emit source map locations for children of this node.
    NoTokenLeadingSourceMaps = 1 << 8,      // Do not emit leading source map location for token nodes.
    NoTokenTrailingSourceMaps = 1 << 9,     // Do not emit trailing source map location for token nodes.
    NoTokenSourceMaps = NoTokenLeadingSourceMaps | NoTokenTrailingSourceMaps, // Do not emit source map locations for tokens of this node.
    NoLeadingComments = 1 << 10,            // Do not emit leading comments for this node.
    NoTrailingComments = 1 << 11,           // Do not emit trailing comments for this node.
    NoComments = NoLeadingComments | NoTrailingComments, // Do not emit comments for this node.
    NoNestedComments = 1 << 12,
    HelperName = 1 << 13,                   // The Identifier refers to an *unscoped* emit helper (one that is emitted at the top of the file)
    ExportName = 1 << 14,                   // Ensure an export prefix is added for an identifier that points to an exported declaration with a local name (see SymbolFlags.ExportHasLocal).
    LocalName = 1 << 15,                    // Ensure an export prefix is not added for an identifier that points to an exported declaration.
    InternalName = 1 << 16,                 // The name is internal to an ES5 class body function.
    Indented = 1 << 17,                     // Adds an explicit extra indentation level for class and function bodies when printing (used to match old emitter).
    NoIndentation = 1 << 18,                // Do not indent the node.
    AsyncFunctionBody = 1 << 19,
    ReuseTempVariableScope = 1 << 20,       // Reuse the existing temp variable scope during emit.
    CustomPrologue = 1 << 21,               // Treat the statement as if it were a prologue directive (NOTE: Prologue directives are *not* transformed).
    NoHoisting = 1 << 22,                   // Do not hoist this declaration in --module system
    Iterator = 1 << 23,                     // The expression to a `yield*` should be treated as an Iterator when down-leveling, not an Iterable.
    NoAsciiEscaping = 1 << 24,              // When synthesizing nodes that lack an original node or textSourceNode, we want to write the text on the node with ASCII escaping substitutions.
}

// dprint-ignore
/** @internal */
export const enum InternalEmitFlags {
    None = 0,
    TypeScriptClassWrapper = 1 << 0, // The node is an IIFE class wrapper created by the ts transform.
    NeverApplyImportHelper = 1 << 1, // Indicates the node should never be wrapped with an import star helper (because, for example, it imports tslib itself)
    IgnoreSourceNewlines = 1 << 2,   // Overrides `printerOptions.preserveSourceNewlines` to print this node (and all descendants) with default whitespace.
    Immutable = 1 << 3,              // Indicates a node is a singleton intended to be reused in multiple locations. Any attempt to make further changes to the node will result in an error.
    IndirectCall = 1 << 4,           // Emit CallExpression as an indirect call: `(0, f)()`
    TransformPrivateStaticElements = 1 << 5, // Indicates static private elements in a file or class should be transformed regardless of --target (used by esDecorators transform)
}

// dprint-ignore
interface EmitHelperBase {
    readonly name: string;                                          // A unique name for this helper.
    readonly scoped: boolean;                                       // Indicates whether the helper MUST be emitted in the current scope.
    readonly text: string | ((node: EmitHelperUniqueNameCallback) => string);  // ES3-compatible raw script text, or a function yielding such a string
    readonly priority?: number;                                     // Helpers with a higher priority are emitted earlier than other helpers on the node.
    readonly dependencies?: EmitHelper[]
}

interface ScopedEmitHelper extends EmitHelperBase {
    readonly scoped: true;
}

// dprint-ignore
interface UnscopedEmitHelper extends EmitHelperBase {
    readonly scoped: false;                                         // Indicates whether the helper MUST be emitted in the current scope.
    /** @internal */
    readonly importName?: string;                                   // The name of the helper to use when importing via `--importHelpers`.
    readonly text: string;                                          // ES3-compatible raw script text, or a function yielding such a string
}

type EmitHelper = ScopedEmitHelper | UnscopedEmitHelper;

type EmitHelperUniqueNameCallback = (name: string) => string;

/** @internal */
export interface PropertyDescriptorAttributes {
    enumerable?: boolean | Expression;
    configurable?: boolean | Expression;
    writable?: boolean | Expression;
    value?: Expression;
    get?: Expression;
    set?: Expression;
}

export const enum OuterExpressionKinds {
    Parentheses = 1 << 0,
    TypeAssertions = 1 << 1,
    NonNullAssertions = 1 << 2,
    PartiallyEmittedExpressions = 1 << 3,

    Assertions = TypeAssertions | NonNullAssertions,
    All = Parentheses | Assertions | PartiallyEmittedExpressions,

    ExcludeJSDocTypeAssertion = 1 << 4,
}

/** @internal */
export type OuterExpression =
    | ParenthesizedExpression
    | TypeAssertion
    | SatisfiesExpression
    | AsExpression
    | NonNullExpression
    | PartiallyEmittedExpression;

/** @internal */
export type WrappedExpression<T extends Expression> =
    | OuterExpression & { readonly expression: WrappedExpression<T>; }
    | T;

/** @internal */
export type TypeOfTag = "null" | "undefined" | "number" | "bigint" | "boolean" | "string" | "symbol" | "object" | "function";

/** @internal */
export interface CallBinding {
    target: LeftHandSideExpression;
    thisArg: Expression;
}

/** @internal */
export interface ParenthesizerRules {
    getParenthesizeLeftSideOfBinaryForOperator(binaryOperator: SyntaxKind): (leftSide: Expression) => Expression;
    getParenthesizeRightSideOfBinaryForOperator(binaryOperator: SyntaxKind): (rightSide: Expression) => Expression;
    parenthesizeLeftSideOfBinary(binaryOperator: SyntaxKind, leftSide: Expression): Expression;
    parenthesizeRightSideOfBinary(binaryOperator: SyntaxKind, leftSide: Expression | undefined, rightSide: Expression): Expression;
    parenthesizeExpressionOfComputedPropertyName(expression: Expression): Expression;
    parenthesizeConditionOfConditionalExpression(condition: Expression): Expression;
    parenthesizeBranchOfConditionalExpression(branch: Expression): Expression;
    parenthesizeExpressionOfExportDefault(expression: Expression): Expression;
    parenthesizeExpressionOfNew(expression: Expression): LeftHandSideExpression;
    parenthesizeLeftSideOfAccess(expression: Expression, optionalChain?: boolean): LeftHandSideExpression;
    parenthesizeOperandOfPostfixUnary(operand: Expression): LeftHandSideExpression;
    parenthesizeOperandOfPrefixUnary(operand: Expression): UnaryExpression;
    parenthesizeExpressionsOfCommaDelimitedList(elements: readonly Expression[]): NodeArray<Expression>;
    parenthesizeExpressionForDisallowedComma(expression: Expression): Expression;
    parenthesizeExpressionOfExpressionStatement(expression: Expression): Expression;
    parenthesizeConciseBodyOfArrowFunction(body: Expression): Expression;
    parenthesizeConciseBodyOfArrowFunction(body: ConciseBody): ConciseBody;
    parenthesizeCheckTypeOfConditionalType(type: TypeNode): TypeNode;
    parenthesizeExtendsTypeOfConditionalType(type: TypeNode): TypeNode;
    parenthesizeOperandOfTypeOperator(type: TypeNode): TypeNode;
    parenthesizeOperandOfReadonlyTypeOperator(type: TypeNode): TypeNode;
    parenthesizeNonArrayTypeOfPostfixType(type: TypeNode): TypeNode;
    parenthesizeElementTypesOfTupleType(types: readonly (TypeNode | NamedTupleMember)[]): NodeArray<TypeNode>;
    parenthesizeElementTypeOfTupleType(type: TypeNode | NamedTupleMember): TypeNode | NamedTupleMember;
    parenthesizeTypeOfOptionalType(type: TypeNode): TypeNode;
    parenthesizeConstituentTypeOfUnionType(type: TypeNode): TypeNode;
    parenthesizeConstituentTypesOfUnionType(constituents: readonly TypeNode[]): NodeArray<TypeNode>;
    parenthesizeConstituentTypeOfIntersectionType(type: TypeNode): TypeNode;
    parenthesizeConstituentTypesOfIntersectionType(constituents: readonly TypeNode[]): NodeArray<TypeNode>;
    parenthesizeLeadingTypeArgument(typeNode: TypeNode): TypeNode;
    parenthesizeTypeArguments(typeParameters: readonly TypeNode[] | undefined): NodeArray<TypeNode> | undefined;
}

/** @internal */
export interface GeneratedNamePart {
    /** an additional prefix to insert before the text sourced from `node` */
    prefix?: string;
    node: Identifier | PrivateIdentifier;
    /** an additional suffix to insert after the text sourced from `node` */
    suffix?: string;
}

export type ImmediatelyInvokedFunctionExpression = CallExpression & { readonly expression: FunctionExpression; };
export type ImmediatelyInvokedArrowFunction = CallExpression & { readonly expression: ParenthesizedExpression & { readonly expression: ArrowFunction; }; };

export interface NodeFactory {
    createNodeArray<T extends Node>(elements?: readonly T[], hasTrailingComma?: boolean): NodeArray<T>;

    //
    // Literals
    //

    createNumericLiteral(value: string | number, numericLiteralFlags?: TokenFlags): NumericLiteral;
    createBigIntLiteral(value: string | PseudoBigInt): BigIntLiteral;
    createStringLiteral(text: string, isSingleQuote?: boolean): StringLiteral;
    /** @internal */ createStringLiteral(text: string, isSingleQuote?: boolean, hasExtendedUnicodeEscape?: boolean): StringLiteral; // eslint-disable-line @typescript-eslint/unified-signatures
    createStringLiteralFromNode(sourceNode: PropertyNameLiteral | PrivateIdentifier, isSingleQuote?: boolean): StringLiteral;
    createRegularExpressionLiteral(text: string): RegularExpressionLiteral;

    //
    // Identifiers
    //

    createIdentifier(text: string): Identifier;
    /** @internal */ createIdentifier(text: string, originalKeywordKind?: SyntaxKind, hasExtendedUnicodeEscape?: boolean): Identifier; // eslint-disable-line @typescript-eslint/unified-signatures

    /**
     * Create a unique temporary variable.
     * @param recordTempVariable An optional callback used to record the temporary variable name. This
     * should usually be a reference to `hoistVariableDeclaration` from a `TransformationContext`, but
     * can be `undefined` if you plan to record the temporary variable manually.
     * @param reservedInNestedScopes When `true`, reserves the temporary variable name in all nested scopes
     * during emit so that the variable can be referenced in a nested function body. This is an alternative to
     * setting `EmitFlags.ReuseTempVariableScope` on the nested function itself.
     */
    createTempVariable(recordTempVariable: ((node: Identifier) => void) | undefined, reservedInNestedScopes?: boolean): Identifier;
    /** @internal */ createTempVariable(recordTempVariable: ((node: Identifier) => void) | undefined, reservedInNestedScopes?: boolean, prefix?: string | GeneratedNamePart, suffix?: string): Identifier; // eslint-disable-line @typescript-eslint/unified-signatures

    /**
     * Create a unique temporary variable for use in a loop.
     * @param reservedInNestedScopes When `true`, reserves the temporary variable name in all nested scopes
     * during emit so that the variable can be referenced in a nested function body. This is an alternative to
     * setting `EmitFlags.ReuseTempVariableScope` on the nested function itself.
     */
    createLoopVariable(reservedInNestedScopes?: boolean): Identifier;

    /** Create a unique name based on the supplied text. */
    createUniqueName(text: string, flags?: GeneratedIdentifierFlags): Identifier;
    /** @internal */ createUniqueName(text: string, flags?: GeneratedIdentifierFlags, prefix?: string | GeneratedNamePart, suffix?: string): Identifier; // eslint-disable-line @typescript-eslint/unified-signatures

    /** Create a unique name generated for a node. */
    getGeneratedNameForNode(node: Node | undefined, flags?: GeneratedIdentifierFlags): Identifier;
    /** @internal */ getGeneratedNameForNode(node: Node | undefined, flags?: GeneratedIdentifierFlags, prefix?: string | GeneratedNamePart, suffix?: string): Identifier; // eslint-disable-line @typescript-eslint/unified-signatures

    createPrivateIdentifier(text: string): PrivateIdentifier;
    createUniquePrivateName(text?: string): PrivateIdentifier;
    /** @internal */ createUniquePrivateName(text?: string, prefix?: string | GeneratedNamePart, suffix?: string): PrivateIdentifier; // eslint-disable-line @typescript-eslint/unified-signatures
    getGeneratedPrivateNameForNode(node: Node): PrivateIdentifier;
    /** @internal */ getGeneratedPrivateNameForNode(node: Node, prefix?: string | GeneratedNamePart, suffix?: string): PrivateIdentifier; // eslint-disable-line @typescript-eslint/unified-signatures

    //
    // Punctuation
    //

    createToken(token: SyntaxKind.SuperKeyword): SuperExpression;
    createToken(token: SyntaxKind.ThisKeyword): ThisExpression;
    createToken(token: SyntaxKind.NullKeyword): NullLiteral;
    createToken(token: SyntaxKind.TrueKeyword): TrueLiteral;
    createToken(token: SyntaxKind.FalseKeyword): FalseLiteral;
    createToken(token: SyntaxKind.EndOfFileToken): EndOfFileToken;
    createToken(token: SyntaxKind.Unknown): Token<SyntaxKind.Unknown>;
    createToken<TKind extends PunctuationSyntaxKind>(token: TKind): PunctuationToken<TKind>;
    createToken<TKind extends KeywordTypeSyntaxKind>(token: TKind): KeywordTypeNode<TKind>;
    createToken<TKind extends ModifierSyntaxKind>(token: TKind): ModifierToken<TKind>;
    createToken<TKind extends KeywordSyntaxKind>(token: TKind): KeywordToken<TKind>;
    /** @internal */ createToken<TKind extends SyntaxKind>(token: TKind): Token<TKind>;

    //
    // Reserved words
    //

    createSuper(): SuperExpression;
    createThis(): ThisExpression;
    createNull(): NullLiteral;
    createTrue(): TrueLiteral;
    createFalse(): FalseLiteral;

    //
    // Modifiers
    //

    createModifier<T extends ModifierSyntaxKind>(kind: T): ModifierToken<T>;
    createModifiersFromModifierFlags(flags: ModifierFlags): Modifier[] | undefined;

    //
    // Names
    //

    createQualifiedName(left: EntityName, right: string | Identifier): QualifiedName;
    updateQualifiedName(node: QualifiedName, left: EntityName, right: Identifier): QualifiedName;
    createComputedPropertyName(expression: Expression): ComputedPropertyName;
    updateComputedPropertyName(node: ComputedPropertyName, expression: Expression): ComputedPropertyName;

    //
    // Signature elements
    //

    createTypeParameterDeclaration(modifiers: readonly Modifier[] | undefined, name: string | Identifier, constraint?: TypeNode, defaultType?: TypeNode): TypeParameterDeclaration;
    updateTypeParameterDeclaration(node: TypeParameterDeclaration, modifiers: readonly Modifier[] | undefined, name: Identifier, constraint: TypeNode | undefined, defaultType: TypeNode | undefined): TypeParameterDeclaration;
    createParameterDeclaration(modifiers: readonly ModifierLike[] | undefined, dotDotDotToken: DotDotDotToken | undefined, name: string | BindingName, questionToken?: QuestionToken, type?: TypeNode, initializer?: Expression): ParameterDeclaration;
    updateParameterDeclaration(node: ParameterDeclaration, modifiers: readonly ModifierLike[] | undefined, dotDotDotToken: DotDotDotToken | undefined, name: string | BindingName, questionToken: QuestionToken | undefined, type: TypeNode | undefined, initializer: Expression | undefined): ParameterDeclaration;
    createDecorator(expression: Expression): Decorator;
    updateDecorator(node: Decorator, expression: Expression): Decorator;

    //
    // Type Elements
    //

    createPropertySignature(modifiers: readonly Modifier[] | undefined, name: PropertyName | string, questionToken: QuestionToken | undefined, type: TypeNode | undefined): PropertySignature;
    updatePropertySignature(node: PropertySignature, modifiers: readonly Modifier[] | undefined, name: PropertyName, questionToken: QuestionToken | undefined, type: TypeNode | undefined): PropertySignature;
    createPropertyDeclaration(modifiers: readonly ModifierLike[] | undefined, name: string | PropertyName, questionOrExclamationToken: QuestionToken | ExclamationToken | undefined, type: TypeNode | undefined, initializer: Expression | undefined): PropertyDeclaration;
    updatePropertyDeclaration(node: PropertyDeclaration, modifiers: readonly ModifierLike[] | undefined, name: string | PropertyName, questionOrExclamationToken: QuestionToken | ExclamationToken | undefined, type: TypeNode | undefined, initializer: Expression | undefined): PropertyDeclaration;
    createMethodSignature(modifiers: readonly Modifier[] | undefined, name: string | PropertyName, questionToken: QuestionToken | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined): MethodSignature;
    updateMethodSignature(node: MethodSignature, modifiers: readonly Modifier[] | undefined, name: PropertyName, questionToken: QuestionToken | undefined, typeParameters: NodeArray<TypeParameterDeclaration> | undefined, parameters: NodeArray<ParameterDeclaration>, type: TypeNode | undefined): MethodSignature;
    createMethodDeclaration(modifiers: readonly ModifierLike[] | undefined, asteriskToken: AsteriskToken | undefined, name: string | PropertyName, questionToken: QuestionToken | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, body: Block | undefined): MethodDeclaration;
    updateMethodDeclaration(node: MethodDeclaration, modifiers: readonly ModifierLike[] | undefined, asteriskToken: AsteriskToken | undefined, name: PropertyName, questionToken: QuestionToken | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, body: Block | undefined): MethodDeclaration;
    createConstructorDeclaration(modifiers: readonly ModifierLike[] | undefined, parameters: readonly ParameterDeclaration[], body: Block | undefined): ConstructorDeclaration;
    updateConstructorDeclaration(node: ConstructorDeclaration, modifiers: readonly ModifierLike[] | undefined, parameters: readonly ParameterDeclaration[], body: Block | undefined): ConstructorDeclaration;
    createGetAccessorDeclaration(modifiers: readonly ModifierLike[] | undefined, name: string | PropertyName, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, body: Block | undefined): GetAccessorDeclaration;
    updateGetAccessorDeclaration(node: GetAccessorDeclaration, modifiers: readonly ModifierLike[] | undefined, name: PropertyName, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, body: Block | undefined): GetAccessorDeclaration;
    createSetAccessorDeclaration(modifiers: readonly ModifierLike[] | undefined, name: string | PropertyName, parameters: readonly ParameterDeclaration[], body: Block | undefined): SetAccessorDeclaration;
    updateSetAccessorDeclaration(node: SetAccessorDeclaration, modifiers: readonly ModifierLike[] | undefined, name: PropertyName, parameters: readonly ParameterDeclaration[], body: Block | undefined): SetAccessorDeclaration;
    createCallSignature(typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined): CallSignatureDeclaration;
    updateCallSignature(node: CallSignatureDeclaration, typeParameters: NodeArray<TypeParameterDeclaration> | undefined, parameters: NodeArray<ParameterDeclaration>, type: TypeNode | undefined): CallSignatureDeclaration;
    createConstructSignature(typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined): ConstructSignatureDeclaration;
    updateConstructSignature(node: ConstructSignatureDeclaration, typeParameters: NodeArray<TypeParameterDeclaration> | undefined, parameters: NodeArray<ParameterDeclaration>, type: TypeNode | undefined): ConstructSignatureDeclaration;
    createIndexSignature(modifiers: readonly ModifierLike[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode): IndexSignatureDeclaration;
    /** @internal */ createIndexSignature(modifiers: readonly ModifierLike[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined): IndexSignatureDeclaration; // eslint-disable-line @typescript-eslint/unified-signatures
    updateIndexSignature(node: IndexSignatureDeclaration, modifiers: readonly ModifierLike[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode): IndexSignatureDeclaration;
    createTemplateLiteralTypeSpan(type: TypeNode, literal: TemplateMiddle | TemplateTail): TemplateLiteralTypeSpan;
    updateTemplateLiteralTypeSpan(node: TemplateLiteralTypeSpan, type: TypeNode, literal: TemplateMiddle | TemplateTail): TemplateLiteralTypeSpan;
    createClassStaticBlockDeclaration(body: Block): ClassStaticBlockDeclaration;
    updateClassStaticBlockDeclaration(node: ClassStaticBlockDeclaration, body: Block): ClassStaticBlockDeclaration;

    //
    // Types
    //

    createKeywordTypeNode<TKind extends KeywordTypeSyntaxKind>(kind: TKind): KeywordTypeNode<TKind>;
    createTypePredicateNode(assertsModifier: AssertsKeyword | undefined, parameterName: Identifier | ThisTypeNode | string, type: TypeNode | undefined): TypePredicateNode;
    updateTypePredicateNode(node: TypePredicateNode, assertsModifier: AssertsKeyword | undefined, parameterName: Identifier | ThisTypeNode, type: TypeNode | undefined): TypePredicateNode;
    createTypeReferenceNode(typeName: string | EntityName, typeArguments?: readonly TypeNode[]): TypeReferenceNode;
    updateTypeReferenceNode(node: TypeReferenceNode, typeName: EntityName, typeArguments: NodeArray<TypeNode> | undefined): TypeReferenceNode;
    createFunctionTypeNode(typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode): FunctionTypeNode;
    updateFunctionTypeNode(node: FunctionTypeNode, typeParameters: NodeArray<TypeParameterDeclaration> | undefined, parameters: NodeArray<ParameterDeclaration>, type: TypeNode): FunctionTypeNode;
    createConstructorTypeNode(modifiers: readonly Modifier[] | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode): ConstructorTypeNode;
    updateConstructorTypeNode(node: ConstructorTypeNode, modifiers: readonly Modifier[] | undefined, typeParameters: NodeArray<TypeParameterDeclaration> | undefined, parameters: NodeArray<ParameterDeclaration>, type: TypeNode): ConstructorTypeNode;
    createTypeQueryNode(exprName: EntityName, typeArguments?: readonly TypeNode[]): TypeQueryNode;
    updateTypeQueryNode(node: TypeQueryNode, exprName: EntityName, typeArguments?: readonly TypeNode[]): TypeQueryNode;
    createTypeLiteralNode(members: readonly TypeElement[] | undefined): TypeLiteralNode;
    updateTypeLiteralNode(node: TypeLiteralNode, members: NodeArray<TypeElement>): TypeLiteralNode;
    createArrayTypeNode(elementType: TypeNode): ArrayTypeNode;
    updateArrayTypeNode(node: ArrayTypeNode, elementType: TypeNode): ArrayTypeNode;
    createTupleTypeNode(elements: readonly (TypeNode | NamedTupleMember)[]): TupleTypeNode;
    updateTupleTypeNode(node: TupleTypeNode, elements: readonly (TypeNode | NamedTupleMember)[]): TupleTypeNode;
    createNamedTupleMember(dotDotDotToken: DotDotDotToken | undefined, name: Identifier, questionToken: QuestionToken | undefined, type: TypeNode): NamedTupleMember;
    updateNamedTupleMember(node: NamedTupleMember, dotDotDotToken: DotDotDotToken | undefined, name: Identifier, questionToken: QuestionToken | undefined, type: TypeNode): NamedTupleMember;
    createOptionalTypeNode(type: TypeNode): OptionalTypeNode;
    updateOptionalTypeNode(node: OptionalTypeNode, type: TypeNode): OptionalTypeNode;
    createRestTypeNode(type: TypeNode): RestTypeNode;
    updateRestTypeNode(node: RestTypeNode, type: TypeNode): RestTypeNode;
    createUnionTypeNode(types: readonly TypeNode[]): UnionTypeNode;
    updateUnionTypeNode(node: UnionTypeNode, types: NodeArray<TypeNode>): UnionTypeNode;
    createIntersectionTypeNode(types: readonly TypeNode[]): IntersectionTypeNode;
    updateIntersectionTypeNode(node: IntersectionTypeNode, types: NodeArray<TypeNode>): IntersectionTypeNode;
    createConditionalTypeNode(checkType: TypeNode, extendsType: TypeNode, trueType: TypeNode, falseType: TypeNode): ConditionalTypeNode;
    updateConditionalTypeNode(node: ConditionalTypeNode, checkType: TypeNode, extendsType: TypeNode, trueType: TypeNode, falseType: TypeNode): ConditionalTypeNode;
    createInferTypeNode(typeParameter: TypeParameterDeclaration): InferTypeNode;
    updateInferTypeNode(node: InferTypeNode, typeParameter: TypeParameterDeclaration): InferTypeNode;
    createImportTypeNode(argument: TypeNode, attributes?: ImportAttributes, qualifier?: EntityName, typeArguments?: readonly TypeNode[], isTypeOf?: boolean): ImportTypeNode;
    updateImportTypeNode(node: ImportTypeNode, argument: TypeNode, attributes: ImportAttributes | undefined, qualifier: EntityName | undefined, typeArguments: readonly TypeNode[] | undefined, isTypeOf?: boolean): ImportTypeNode;
    createParenthesizedType(type: TypeNode): ParenthesizedTypeNode;
    updateParenthesizedType(node: ParenthesizedTypeNode, type: TypeNode): ParenthesizedTypeNode;
    createThisTypeNode(): ThisTypeNode;
    createTypeOperatorNode(operator: SyntaxKind.KeyOfKeyword | SyntaxKind.UniqueKeyword | SyntaxKind.ReadonlyKeyword, type: TypeNode): TypeOperatorNode;
    updateTypeOperatorNode(node: TypeOperatorNode, type: TypeNode): TypeOperatorNode;
    createIndexedAccessTypeNode(objectType: TypeNode, indexType: TypeNode): IndexedAccessTypeNode;
    updateIndexedAccessTypeNode(node: IndexedAccessTypeNode, objectType: TypeNode, indexType: TypeNode): IndexedAccessTypeNode;
    createMappedTypeNode(readonlyToken: ReadonlyKeyword | PlusToken | MinusToken | undefined, typeParameter: TypeParameterDeclaration, nameType: TypeNode | undefined, questionToken: QuestionToken | PlusToken | MinusToken | undefined, type: TypeNode | undefined, members: NodeArray<TypeElement> | undefined): MappedTypeNode;
    updateMappedTypeNode(node: MappedTypeNode, readonlyToken: ReadonlyKeyword | PlusToken | MinusToken | undefined, typeParameter: TypeParameterDeclaration, nameType: TypeNode | undefined, questionToken: QuestionToken | PlusToken | MinusToken | undefined, type: TypeNode | undefined, members: NodeArray<TypeElement> | undefined): MappedTypeNode;
    createLiteralTypeNode(literal: LiteralTypeNode["literal"]): LiteralTypeNode;
    updateLiteralTypeNode(node: LiteralTypeNode, literal: LiteralTypeNode["literal"]): LiteralTypeNode;
    createTemplateLiteralType(head: TemplateHead, templateSpans: readonly TemplateLiteralTypeSpan[]): TemplateLiteralTypeNode;
    updateTemplateLiteralType(node: TemplateLiteralTypeNode, head: TemplateHead, templateSpans: readonly TemplateLiteralTypeSpan[]): TemplateLiteralTypeNode;

    //
    // Binding Patterns
    //

    createObjectBindingPattern(elements: readonly BindingElement[]): ObjectBindingPattern;
    updateObjectBindingPattern(node: ObjectBindingPattern, elements: readonly BindingElement[]): ObjectBindingPattern;
    createArrayBindingPattern(elements: readonly ArrayBindingElement[]): ArrayBindingPattern;
    updateArrayBindingPattern(node: ArrayBindingPattern, elements: readonly ArrayBindingElement[]): ArrayBindingPattern;
    createBindingElement(dotDotDotToken: DotDotDotToken | undefined, propertyName: string | PropertyName | undefined, name: string | BindingName, initializer?: Expression): BindingElement;
    updateBindingElement(node: BindingElement, dotDotDotToken: DotDotDotToken | undefined, propertyName: PropertyName | undefined, name: BindingName, initializer: Expression | undefined): BindingElement;

    //
    // Expression
    //

    createArrayLiteralExpression(elements?: readonly Expression[], multiLine?: boolean): ArrayLiteralExpression;
    updateArrayLiteralExpression(node: ArrayLiteralExpression, elements: readonly Expression[]): ArrayLiteralExpression;
    createObjectLiteralExpression(properties?: readonly ObjectLiteralElementLike[], multiLine?: boolean): ObjectLiteralExpression;
    updateObjectLiteralExpression(node: ObjectLiteralExpression, properties: readonly ObjectLiteralElementLike[]): ObjectLiteralExpression;
    createPropertyAccessExpression(expression: Expression, name: string | MemberName): PropertyAccessExpression;
    updatePropertyAccessExpression(node: PropertyAccessExpression, expression: Expression, name: MemberName): PropertyAccessExpression;
    createPropertyAccessChain(expression: Expression, questionDotToken: QuestionDotToken | undefined, name: string | MemberName): PropertyAccessChain;
    updatePropertyAccessChain(node: PropertyAccessChain, expression: Expression, questionDotToken: QuestionDotToken | undefined, name: MemberName): PropertyAccessChain;
    createElementAccessExpression(expression: Expression, index: number | Expression): ElementAccessExpression;
    updateElementAccessExpression(node: ElementAccessExpression, expression: Expression, argumentExpression: Expression): ElementAccessExpression;
    createElementAccessChain(expression: Expression, questionDotToken: QuestionDotToken | undefined, index: number | Expression): ElementAccessChain;
    updateElementAccessChain(node: ElementAccessChain, expression: Expression, questionDotToken: QuestionDotToken | undefined, argumentExpression: Expression): ElementAccessChain;
    createCallExpression(expression: Expression, typeArguments: readonly TypeNode[] | undefined, argumentsArray: readonly Expression[] | undefined): CallExpression;
    updateCallExpression(node: CallExpression, expression: Expression, typeArguments: readonly TypeNode[] | undefined, argumentsArray: readonly Expression[]): CallExpression;
    createCallChain(expression: Expression, questionDotToken: QuestionDotToken | undefined, typeArguments: readonly TypeNode[] | undefined, argumentsArray: readonly Expression[] | undefined): CallChain;
    updateCallChain(node: CallChain, expression: Expression, questionDotToken: QuestionDotToken | undefined, typeArguments: readonly TypeNode[] | undefined, argumentsArray: readonly Expression[]): CallChain;
    createNewExpression(expression: Expression, typeArguments: readonly TypeNode[] | undefined, argumentsArray: readonly Expression[] | undefined): NewExpression;
    updateNewExpression(node: NewExpression, expression: Expression, typeArguments: readonly TypeNode[] | undefined, argumentsArray: readonly Expression[] | undefined): NewExpression;
    createTaggedTemplateExpression(tag: Expression, typeArguments: readonly TypeNode[] | undefined, template: TemplateLiteral): TaggedTemplateExpression;
    updateTaggedTemplateExpression(node: TaggedTemplateExpression, tag: Expression, typeArguments: readonly TypeNode[] | undefined, template: TemplateLiteral): TaggedTemplateExpression;
    createTypeAssertion(type: TypeNode, expression: Expression): TypeAssertion;
    updateTypeAssertion(node: TypeAssertion, type: TypeNode, expression: Expression): TypeAssertion;
    createParenthesizedExpression(expression: Expression): ParenthesizedExpression;
    updateParenthesizedExpression(node: ParenthesizedExpression, expression: Expression): ParenthesizedExpression;
    createFunctionExpression(modifiers: readonly Modifier[] | undefined, asteriskToken: AsteriskToken | undefined, name: string | Identifier | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[] | undefined, type: TypeNode | undefined, body: Block): FunctionExpression;
    updateFunctionExpression(node: FunctionExpression, modifiers: readonly Modifier[] | undefined, asteriskToken: AsteriskToken | undefined, name: Identifier | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, body: Block): FunctionExpression;
    createArrowFunction(modifiers: readonly Modifier[] | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, equalsGreaterThanToken: EqualsGreaterThanToken | undefined, body: ConciseBody): ArrowFunction;
    updateArrowFunction(node: ArrowFunction, modifiers: readonly Modifier[] | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, equalsGreaterThanToken: EqualsGreaterThanToken, body: ConciseBody): ArrowFunction;
    createDeleteExpression(expression: Expression): DeleteExpression;
    updateDeleteExpression(node: DeleteExpression, expression: Expression): DeleteExpression;
    createTypeOfExpression(expression: Expression): TypeOfExpression;
    updateTypeOfExpression(node: TypeOfExpression, expression: Expression): TypeOfExpression;
    createVoidExpression(expression: Expression): VoidExpression;
    updateVoidExpression(node: VoidExpression, expression: Expression): VoidExpression;
    createAwaitExpression(expression: Expression): AwaitExpression;
    updateAwaitExpression(node: AwaitExpression, expression: Expression): AwaitExpression;
    createPrefixUnaryExpression(operator: PrefixUnaryOperator, operand: Expression): PrefixUnaryExpression;
    updatePrefixUnaryExpression(node: PrefixUnaryExpression, operand: Expression): PrefixUnaryExpression;
    createPostfixUnaryExpression(operand: Expression, operator: PostfixUnaryOperator): PostfixUnaryExpression;
    updatePostfixUnaryExpression(node: PostfixUnaryExpression, operand: Expression): PostfixUnaryExpression;
    createBinaryExpression(left: Expression, operator: BinaryOperator | BinaryOperatorToken, right: Expression): BinaryExpression;
    updateBinaryExpression(node: BinaryExpression, left: Expression, operator: BinaryOperator | BinaryOperatorToken, right: Expression): BinaryExpression;
    createConditionalExpression(condition: Expression, questionToken: QuestionToken | undefined, whenTrue: Expression, colonToken: ColonToken | undefined, whenFalse: Expression): ConditionalExpression;
    updateConditionalExpression(node: ConditionalExpression, condition: Expression, questionToken: QuestionToken, whenTrue: Expression, colonToken: ColonToken, whenFalse: Expression): ConditionalExpression;
    createTemplateExpression(head: TemplateHead, templateSpans: readonly TemplateSpan[]): TemplateExpression;
    updateTemplateExpression(node: TemplateExpression, head: TemplateHead, templateSpans: readonly TemplateSpan[]): TemplateExpression;
    createTemplateHead(text: string, rawText?: string, templateFlags?: TokenFlags): TemplateHead;
    createTemplateHead(text: string | undefined, rawText: string, templateFlags?: TokenFlags): TemplateHead;
    createTemplateMiddle(text: string, rawText?: string, templateFlags?: TokenFlags): TemplateMiddle;
    createTemplateMiddle(text: string | undefined, rawText: string, templateFlags?: TokenFlags): TemplateMiddle;
    createTemplateTail(text: string, rawText?: string, templateFlags?: TokenFlags): TemplateTail;
    createTemplateTail(text: string | undefined, rawText: string, templateFlags?: TokenFlags): TemplateTail;
    createNoSubstitutionTemplateLiteral(text: string, rawText?: string): NoSubstitutionTemplateLiteral;
    createNoSubstitutionTemplateLiteral(text: string | undefined, rawText: string): NoSubstitutionTemplateLiteral;
    /** @internal */ createLiteralLikeNode(kind: LiteralToken["kind"] | SyntaxKind.JsxTextAllWhiteSpaces, text: string): LiteralToken;
    /** @internal */ createTemplateLiteralLikeNode(kind: TemplateLiteralToken["kind"], text: string, rawText: string, templateFlags: TokenFlags | undefined): TemplateLiteralLikeNode;
    createYieldExpression(asteriskToken: AsteriskToken, expression: Expression): YieldExpression;
    createYieldExpression(asteriskToken: undefined, expression: Expression | undefined): YieldExpression;
    /** @internal */ createYieldExpression(asteriskToken: AsteriskToken | undefined, expression: Expression | undefined): YieldExpression; // eslint-disable-line @typescript-eslint/unified-signatures
    updateYieldExpression(node: YieldExpression, asteriskToken: AsteriskToken | undefined, expression: Expression | undefined): YieldExpression;
    createSpreadElement(expression: Expression): SpreadElement;
    updateSpreadElement(node: SpreadElement, expression: Expression): SpreadElement;
    createClassExpression(modifiers: readonly ModifierLike[] | undefined, name: string | Identifier | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, heritageClauses: readonly HeritageClause[] | undefined, members: readonly ClassElement[]): ClassExpression;
    updateClassExpression(node: ClassExpression, modifiers: readonly ModifierLike[] | undefined, name: Identifier | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, heritageClauses: readonly HeritageClause[] | undefined, members: readonly ClassElement[]): ClassExpression;
    createOmittedExpression(): OmittedExpression;
    createExpressionWithTypeArguments(expression: Expression, typeArguments: readonly TypeNode[] | undefined): ExpressionWithTypeArguments;
    updateExpressionWithTypeArguments(node: ExpressionWithTypeArguments, expression: Expression, typeArguments: readonly TypeNode[] | undefined): ExpressionWithTypeArguments;
    createAsExpression(expression: Expression, type: TypeNode): AsExpression;
    updateAsExpression(node: AsExpression, expression: Expression, type: TypeNode): AsExpression;
    createNonNullExpression(expression: Expression): NonNullExpression;
    updateNonNullExpression(node: NonNullExpression, expression: Expression): NonNullExpression;
    createNonNullChain(expression: Expression): NonNullChain;
    updateNonNullChain(node: NonNullChain, expression: Expression): NonNullChain;
    createMetaProperty(keywordToken: MetaProperty["keywordToken"], name: Identifier): MetaProperty;
    updateMetaProperty(node: MetaProperty, name: Identifier): MetaProperty;
    createSatisfiesExpression(expression: Expression, type: TypeNode): SatisfiesExpression;
    updateSatisfiesExpression(node: SatisfiesExpression, expression: Expression, type: TypeNode): SatisfiesExpression;

    //
    // Misc
    //

    createTemplateSpan(expression: Expression, literal: TemplateMiddle | TemplateTail): TemplateSpan;
    updateTemplateSpan(node: TemplateSpan, expression: Expression, literal: TemplateMiddle | TemplateTail): TemplateSpan;
    createSemicolonClassElement(): SemicolonClassElement;

    //
    // Element
    //

    createBlock(statements: readonly Statement[], multiLine?: boolean): Block;
    updateBlock(node: Block, statements: readonly Statement[]): Block;
    createVariableStatement(modifiers: readonly ModifierLike[] | undefined, declarationList: VariableDeclarationList | readonly VariableDeclaration[]): VariableStatement;
    updateVariableStatement(node: VariableStatement, modifiers: readonly ModifierLike[] | undefined, declarationList: VariableDeclarationList): VariableStatement;
    createEmptyStatement(): EmptyStatement;
    createExpressionStatement(expression: Expression): ExpressionStatement;
    updateExpressionStatement(node: ExpressionStatement, expression: Expression): ExpressionStatement;
    createIfStatement(expression: Expression, thenStatement: Statement, elseStatement?: Statement): IfStatement;
    updateIfStatement(node: IfStatement, expression: Expression, thenStatement: Statement, elseStatement: Statement | undefined): IfStatement;
    createDoStatement(statement: Statement, expression: Expression): DoStatement;
    updateDoStatement(node: DoStatement, statement: Statement, expression: Expression): DoStatement;
    createWhileStatement(expression: Expression, statement: Statement): WhileStatement;
    updateWhileStatement(node: WhileStatement, expression: Expression, statement: Statement): WhileStatement;
    createForStatement(initializer: ForInitializer | undefined, condition: Expression | undefined, incrementor: Expression | undefined, statement: Statement): ForStatement;
    updateForStatement(node: ForStatement, initializer: ForInitializer | undefined, condition: Expression | undefined, incrementor: Expression | undefined, statement: Statement): ForStatement;
    createForInStatement(initializer: ForInitializer, expression: Expression, statement: Statement): ForInStatement;
    updateForInStatement(node: ForInStatement, initializer: ForInitializer, expression: Expression, statement: Statement): ForInStatement;
    createForOfStatement(awaitModifier: AwaitKeyword | undefined, initializer: ForInitializer, expression: Expression, statement: Statement): ForOfStatement;
    updateForOfStatement(node: ForOfStatement, awaitModifier: AwaitKeyword | undefined, initializer: ForInitializer, expression: Expression, statement: Statement): ForOfStatement;
    createContinueStatement(label?: string | Identifier): ContinueStatement;
    updateContinueStatement(node: ContinueStatement, label: Identifier | undefined): ContinueStatement;
    createBreakStatement(label?: string | Identifier): BreakStatement;
    updateBreakStatement(node: BreakStatement, label: Identifier | undefined): BreakStatement;
    createReturnStatement(expression?: Expression): ReturnStatement;
    updateReturnStatement(node: ReturnStatement, expression: Expression | undefined): ReturnStatement;
    createWithStatement(expression: Expression, statement: Statement): WithStatement;
    updateWithStatement(node: WithStatement, expression: Expression, statement: Statement): WithStatement;
    createSwitchStatement(expression: Expression, caseBlock: CaseBlock): SwitchStatement;
    updateSwitchStatement(node: SwitchStatement, expression: Expression, caseBlock: CaseBlock): SwitchStatement;
    createLabeledStatement(label: string | Identifier, statement: Statement): LabeledStatement;
    updateLabeledStatement(node: LabeledStatement, label: Identifier, statement: Statement): LabeledStatement;
    createThrowStatement(expression: Expression): ThrowStatement;
    updateThrowStatement(node: ThrowStatement, expression: Expression): ThrowStatement;
    createTryStatement(tryBlock: Block, catchClause: CatchClause | undefined, finallyBlock: Block | undefined): TryStatement;
    updateTryStatement(node: TryStatement, tryBlock: Block, catchClause: CatchClause | undefined, finallyBlock: Block | undefined): TryStatement;
    createDebuggerStatement(): DebuggerStatement;
    createVariableDeclaration(name: string | BindingName, exclamationToken?: ExclamationToken, type?: TypeNode, initializer?: Expression): VariableDeclaration;
    updateVariableDeclaration(node: VariableDeclaration, name: BindingName, exclamationToken: ExclamationToken | undefined, type: TypeNode | undefined, initializer: Expression | undefined): VariableDeclaration;
    createVariableDeclarationList(declarations: readonly VariableDeclaration[], flags?: NodeFlags): VariableDeclarationList;
    updateVariableDeclarationList(node: VariableDeclarationList, declarations: readonly VariableDeclaration[]): VariableDeclarationList;
    createFunctionDeclaration(modifiers: readonly ModifierLike[] | undefined, asteriskToken: AsteriskToken | undefined, name: string | Identifier | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, body: Block | undefined): FunctionDeclaration;
    updateFunctionDeclaration(node: FunctionDeclaration, modifiers: readonly ModifierLike[] | undefined, asteriskToken: AsteriskToken | undefined, name: Identifier | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, body: Block | undefined): FunctionDeclaration;
    createClassDeclaration(modifiers: readonly ModifierLike[] | undefined, name: string | Identifier | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, heritageClauses: readonly HeritageClause[] | undefined, members: readonly ClassElement[]): ClassDeclaration;
    updateClassDeclaration(node: ClassDeclaration, modifiers: readonly ModifierLike[] | undefined, name: Identifier | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, heritageClauses: readonly HeritageClause[] | undefined, members: readonly ClassElement[]): ClassDeclaration;
    createInterfaceDeclaration(modifiers: readonly ModifierLike[] | undefined, name: string | Identifier, typeParameters: readonly TypeParameterDeclaration[] | undefined, heritageClauses: readonly HeritageClause[] | undefined, members: readonly TypeElement[]): InterfaceDeclaration;
    updateInterfaceDeclaration(node: InterfaceDeclaration, modifiers: readonly ModifierLike[] | undefined, name: Identifier, typeParameters: readonly TypeParameterDeclaration[] | undefined, heritageClauses: readonly HeritageClause[] | undefined, members: readonly TypeElement[]): InterfaceDeclaration;
    createTypeAliasDeclaration(modifiers: readonly ModifierLike[] | undefined, name: string | Identifier, typeParameters: readonly TypeParameterDeclaration[] | undefined, type: TypeNode): TypeAliasDeclaration;
    updateTypeAliasDeclaration(node: TypeAliasDeclaration, modifiers: readonly ModifierLike[] | undefined, name: Identifier, typeParameters: readonly TypeParameterDeclaration[] | undefined, type: TypeNode): TypeAliasDeclaration;
    createEnumDeclaration(modifiers: readonly ModifierLike[] | undefined, name: string | Identifier, members: readonly EnumMember[]): EnumDeclaration;
    updateEnumDeclaration(node: EnumDeclaration, modifiers: readonly ModifierLike[] | undefined, name: Identifier, members: readonly EnumMember[]): EnumDeclaration;
    createModuleDeclaration(modifiers: readonly ModifierLike[] | undefined, name: ModuleName, body: ModuleBody | undefined, flags?: NodeFlags): ModuleDeclaration;
    updateModuleDeclaration(node: ModuleDeclaration, modifiers: readonly ModifierLike[] | undefined, name: ModuleName, body: ModuleBody | undefined): ModuleDeclaration;
    createModuleBlock(statements: readonly Statement[]): ModuleBlock;
    updateModuleBlock(node: ModuleBlock, statements: readonly Statement[]): ModuleBlock;
    createCaseBlock(clauses: readonly CaseOrDefaultClause[]): CaseBlock;
    updateCaseBlock(node: CaseBlock, clauses: readonly CaseOrDefaultClause[]): CaseBlock;
    createNamespaceExportDeclaration(name: string | Identifier): NamespaceExportDeclaration;
    updateNamespaceExportDeclaration(node: NamespaceExportDeclaration, name: Identifier): NamespaceExportDeclaration;
    createImportEqualsDeclaration(modifiers: readonly ModifierLike[] | undefined, isTypeOnly: boolean, name: string | Identifier, moduleReference: ModuleReference): ImportEqualsDeclaration;
    updateImportEqualsDeclaration(node: ImportEqualsDeclaration, modifiers: readonly ModifierLike[] | undefined, isTypeOnly: boolean, name: Identifier, moduleReference: ModuleReference): ImportEqualsDeclaration;
    createImportDeclaration(modifiers: readonly ModifierLike[] | undefined, importClause: ImportClause | undefined, moduleSpecifier: Expression, attributes?: ImportAttributes): ImportDeclaration;
    updateImportDeclaration(node: ImportDeclaration, modifiers: readonly ModifierLike[] | undefined, importClause: ImportClause | undefined, moduleSpecifier: Expression, attributes: ImportAttributes | undefined): ImportDeclaration;
    createImportClause(isTypeOnly: boolean, name: Identifier | undefined, namedBindings: NamedImportBindings | undefined): ImportClause;
    updateImportClause(node: ImportClause, isTypeOnly: boolean, name: Identifier | undefined, namedBindings: NamedImportBindings | undefined): ImportClause;
    /** @deprecated */ createAssertClause(elements: NodeArray<AssertEntry>, multiLine?: boolean): AssertClause;
    /** @deprecated */ updateAssertClause(node: AssertClause, elements: NodeArray<AssertEntry>, multiLine?: boolean): AssertClause;
    /** @deprecated */ createAssertEntry(name: AssertionKey, value: Expression): AssertEntry;
    /** @deprecated */ updateAssertEntry(node: AssertEntry, name: AssertionKey, value: Expression): AssertEntry;
    /** @deprecated */ createImportTypeAssertionContainer(clause: AssertClause, multiLine?: boolean): ImportTypeAssertionContainer;
    /** @deprecated */ updateImportTypeAssertionContainer(node: ImportTypeAssertionContainer, clause: AssertClause, multiLine?: boolean): ImportTypeAssertionContainer;
    createImportAttributes(elements: NodeArray<ImportAttribute>, multiLine?: boolean): ImportAttributes;
    /** @internal */ createImportAttributes(elements: NodeArray<ImportAttribute>, multiLine?: boolean, token?: ImportAttributes["token"]): ImportAttributes;
    updateImportAttributes(node: ImportAttributes, elements: NodeArray<ImportAttribute>, multiLine?: boolean): ImportAttributes;
    createImportAttribute(name: ImportAttributeName, value: Expression): ImportAttribute;
    updateImportAttribute(node: ImportAttribute, name: ImportAttributeName, value: Expression): ImportAttribute;
    createNamespaceImport(name: Identifier): NamespaceImport;
    updateNamespaceImport(node: NamespaceImport, name: Identifier): NamespaceImport;
    createNamespaceExport(name: Identifier): NamespaceExport;
    updateNamespaceExport(node: NamespaceExport, name: Identifier): NamespaceExport;
    createNamedImports(elements: readonly ImportSpecifier[]): NamedImports;
    updateNamedImports(node: NamedImports, elements: readonly ImportSpecifier[]): NamedImports;
    createImportSpecifier(isTypeOnly: boolean, propertyName: Identifier | undefined, name: Identifier): ImportSpecifier;
    updateImportSpecifier(node: ImportSpecifier, isTypeOnly: boolean, propertyName: Identifier | undefined, name: Identifier): ImportSpecifier;
    createExportAssignment(modifiers: readonly ModifierLike[] | undefined, isExportEquals: boolean | undefined, expression: Expression): ExportAssignment;
    updateExportAssignment(node: ExportAssignment, modifiers: readonly ModifierLike[] | undefined, expression: Expression): ExportAssignment;
    createExportDeclaration(modifiers: readonly ModifierLike[] | undefined, isTypeOnly: boolean, exportClause: NamedExportBindings | undefined, moduleSpecifier?: Expression, attributes?: ImportAttributes): ExportDeclaration;
    updateExportDeclaration(node: ExportDeclaration, modifiers: readonly ModifierLike[] | undefined, isTypeOnly: boolean, exportClause: NamedExportBindings | undefined, moduleSpecifier: Expression | undefined, attributes: ImportAttributes | undefined): ExportDeclaration;
    createNamedExports(elements: readonly ExportSpecifier[]): NamedExports;
    updateNamedExports(node: NamedExports, elements: readonly ExportSpecifier[]): NamedExports;
    createExportSpecifier(isTypeOnly: boolean, propertyName: string | Identifier | undefined, name: string | Identifier): ExportSpecifier;
    updateExportSpecifier(node: ExportSpecifier, isTypeOnly: boolean, propertyName: Identifier | undefined, name: Identifier): ExportSpecifier;
    /** @internal */ createMissingDeclaration(): MissingDeclaration;

    //
    // Module references
    //

    createExternalModuleReference(expression: Expression): ExternalModuleReference;
    updateExternalModuleReference(node: ExternalModuleReference, expression: Expression): ExternalModuleReference;

    //
    // JSDoc
    //

    createJSDocAllType(): JSDocAllType;
    createJSDocUnknownType(): JSDocUnknownType;
    createJSDocNonNullableType(type: TypeNode, postfix?: boolean): JSDocNonNullableType;
    updateJSDocNonNullableType(node: JSDocNonNullableType, type: TypeNode): JSDocNonNullableType;
    createJSDocNullableType(type: TypeNode, postfix?: boolean): JSDocNullableType;
    updateJSDocNullableType(node: JSDocNullableType, type: TypeNode): JSDocNullableType;
    createJSDocOptionalType(type: TypeNode): JSDocOptionalType;
    updateJSDocOptionalType(node: JSDocOptionalType, type: TypeNode): JSDocOptionalType;
    createJSDocFunctionType(parameters: readonly ParameterDeclaration[], type: TypeNode | undefined): JSDocFunctionType;
    updateJSDocFunctionType(node: JSDocFunctionType, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined): JSDocFunctionType;
    createJSDocVariadicType(type: TypeNode): JSDocVariadicType;
    updateJSDocVariadicType(node: JSDocVariadicType, type: TypeNode): JSDocVariadicType;
    createJSDocNamepathType(type: TypeNode): JSDocNamepathType;
    updateJSDocNamepathType(node: JSDocNamepathType, type: TypeNode): JSDocNamepathType;
    createJSDocTypeExpression(type: TypeNode): JSDocTypeExpression;
    updateJSDocTypeExpression(node: JSDocTypeExpression, type: TypeNode): JSDocTypeExpression;
    createJSDocNameReference(name: EntityName | JSDocMemberName): JSDocNameReference;
    updateJSDocNameReference(node: JSDocNameReference, name: EntityName | JSDocMemberName): JSDocNameReference;
    createJSDocMemberName(left: EntityName | JSDocMemberName, right: Identifier): JSDocMemberName;
    updateJSDocMemberName(node: JSDocMemberName, left: EntityName | JSDocMemberName, right: Identifier): JSDocMemberName;
    createJSDocLink(name: EntityName | JSDocMemberName | undefined, text: string): JSDocLink;
    updateJSDocLink(node: JSDocLink, name: EntityName | JSDocMemberName | undefined, text: string): JSDocLink;
    createJSDocLinkCode(name: EntityName | JSDocMemberName | undefined, text: string): JSDocLinkCode;
    updateJSDocLinkCode(node: JSDocLinkCode, name: EntityName | JSDocMemberName | undefined, text: string): JSDocLinkCode;
    createJSDocLinkPlain(name: EntityName | JSDocMemberName | undefined, text: string): JSDocLinkPlain;
    updateJSDocLinkPlain(node: JSDocLinkPlain, name: EntityName | JSDocMemberName | undefined, text: string): JSDocLinkPlain;
    createJSDocTypeLiteral(jsDocPropertyTags?: readonly JSDocPropertyLikeTag[], isArrayType?: boolean): JSDocTypeLiteral;
    updateJSDocTypeLiteral(node: JSDocTypeLiteral, jsDocPropertyTags: readonly JSDocPropertyLikeTag[] | undefined, isArrayType: boolean | undefined): JSDocTypeLiteral;
    createJSDocSignature(typeParameters: readonly JSDocTemplateTag[] | undefined, parameters: readonly JSDocParameterTag[], type?: JSDocReturnTag): JSDocSignature;
    updateJSDocSignature(node: JSDocSignature, typeParameters: readonly JSDocTemplateTag[] | undefined, parameters: readonly JSDocParameterTag[], type: JSDocReturnTag | undefined): JSDocSignature;
    createJSDocTemplateTag(tagName: Identifier | undefined, constraint: JSDocTypeExpression | undefined, typeParameters: readonly TypeParameterDeclaration[], comment?: string | NodeArray<JSDocComment>): JSDocTemplateTag;
    updateJSDocTemplateTag(node: JSDocTemplateTag, tagName: Identifier | undefined, constraint: JSDocTypeExpression | undefined, typeParameters: readonly TypeParameterDeclaration[], comment: string | NodeArray<JSDocComment> | undefined): JSDocTemplateTag;
    createJSDocTypedefTag(tagName: Identifier | undefined, typeExpression?: JSDocTypeExpression | JSDocTypeLiteral, fullName?: Identifier | JSDocNamespaceDeclaration, comment?: string | NodeArray<JSDocComment>): JSDocTypedefTag;
    updateJSDocTypedefTag(node: JSDocTypedefTag, tagName: Identifier | undefined, typeExpression: JSDocTypeExpression | JSDocTypeLiteral | undefined, fullName: Identifier | JSDocNamespaceDeclaration | undefined, comment: string | NodeArray<JSDocComment> | undefined): JSDocTypedefTag;
    createJSDocParameterTag(tagName: Identifier | undefined, name: EntityName, isBracketed: boolean, typeExpression?: JSDocTypeExpression, isNameFirst?: boolean, comment?: string | NodeArray<JSDocComment>): JSDocParameterTag;
    updateJSDocParameterTag(node: JSDocParameterTag, tagName: Identifier | undefined, name: EntityName, isBracketed: boolean, typeExpression: JSDocTypeExpression | undefined, isNameFirst: boolean, comment: string | NodeArray<JSDocComment> | undefined): JSDocParameterTag;
    createJSDocPropertyTag(tagName: Identifier | undefined, name: EntityName, isBracketed: boolean, typeExpression?: JSDocTypeExpression, isNameFirst?: boolean, comment?: string | NodeArray<JSDocComment>): JSDocPropertyTag;
    updateJSDocPropertyTag(node: JSDocPropertyTag, tagName: Identifier | undefined, name: EntityName, isBracketed: boolean, typeExpression: JSDocTypeExpression | undefined, isNameFirst: boolean, comment: string | NodeArray<JSDocComment> | undefined): JSDocPropertyTag;
    createJSDocTypeTag(tagName: Identifier | undefined, typeExpression: JSDocTypeExpression, comment?: string | NodeArray<JSDocComment>): JSDocTypeTag;
    updateJSDocTypeTag(node: JSDocTypeTag, tagName: Identifier | undefined, typeExpression: JSDocTypeExpression, comment: string | NodeArray<JSDocComment> | undefined): JSDocTypeTag;
    createJSDocSeeTag(tagName: Identifier | undefined, nameExpression: JSDocNameReference | undefined, comment?: string | NodeArray<JSDocComment>): JSDocSeeTag;
    updateJSDocSeeTag(node: JSDocSeeTag, tagName: Identifier | undefined, nameExpression: JSDocNameReference | undefined, comment?: string | NodeArray<JSDocComment>): JSDocSeeTag;
    createJSDocReturnTag(tagName: Identifier | undefined, typeExpression?: JSDocTypeExpression, comment?: string | NodeArray<JSDocComment>): JSDocReturnTag;
    updateJSDocReturnTag(node: JSDocReturnTag, tagName: Identifier | undefined, typeExpression: JSDocTypeExpression | undefined, comment: string | NodeArray<JSDocComment> | undefined): JSDocReturnTag;
    createJSDocThisTag(tagName: Identifier | undefined, typeExpression: JSDocTypeExpression, comment?: string | NodeArray<JSDocComment>): JSDocThisTag;
    updateJSDocThisTag(node: JSDocThisTag, tagName: Identifier | undefined, typeExpression: JSDocTypeExpression | undefined, comment: string | NodeArray<JSDocComment> | undefined): JSDocThisTag;
    createJSDocEnumTag(tagName: Identifier | undefined, typeExpression: JSDocTypeExpression, comment?: string | NodeArray<JSDocComment>): JSDocEnumTag;
    updateJSDocEnumTag(node: JSDocEnumTag, tagName: Identifier | undefined, typeExpression: JSDocTypeExpression, comment: string | NodeArray<JSDocComment> | undefined): JSDocEnumTag;
    createJSDocCallbackTag(tagName: Identifier | undefined, typeExpression: JSDocSignature, fullName?: Identifier | JSDocNamespaceDeclaration, comment?: string | NodeArray<JSDocComment>): JSDocCallbackTag;
    updateJSDocCallbackTag(node: JSDocCallbackTag, tagName: Identifier | undefined, typeExpression: JSDocSignature, fullName: Identifier | JSDocNamespaceDeclaration | undefined, comment: string | NodeArray<JSDocComment> | undefined): JSDocCallbackTag;
    createJSDocOverloadTag(tagName: Identifier | undefined, typeExpression: JSDocSignature, comment?: string | NodeArray<JSDocComment>): JSDocOverloadTag;
    updateJSDocOverloadTag(node: JSDocOverloadTag, tagName: Identifier | undefined, typeExpression: JSDocSignature, comment: string | NodeArray<JSDocComment> | undefined): JSDocOverloadTag;
    createJSDocAugmentsTag(tagName: Identifier | undefined, className: JSDocAugmentsTag["class"], comment?: string | NodeArray<JSDocComment>): JSDocAugmentsTag;
    updateJSDocAugmentsTag(node: JSDocAugmentsTag, tagName: Identifier | undefined, className: JSDocAugmentsTag["class"], comment: string | NodeArray<JSDocComment> | undefined): JSDocAugmentsTag;
    createJSDocImplementsTag(tagName: Identifier | undefined, className: JSDocImplementsTag["class"], comment?: string | NodeArray<JSDocComment>): JSDocImplementsTag;
    updateJSDocImplementsTag(node: JSDocImplementsTag, tagName: Identifier | undefined, className: JSDocImplementsTag["class"], comment: string | NodeArray<JSDocComment> | undefined): JSDocImplementsTag;
    createJSDocAuthorTag(tagName: Identifier | undefined, comment?: string | NodeArray<JSDocComment>): JSDocAuthorTag;
    updateJSDocAuthorTag(node: JSDocAuthorTag, tagName: Identifier | undefined, comment: string | NodeArray<JSDocComment> | undefined): JSDocAuthorTag;
    createJSDocClassTag(tagName: Identifier | undefined, comment?: string | NodeArray<JSDocComment>): JSDocClassTag;
    updateJSDocClassTag(node: JSDocClassTag, tagName: Identifier | undefined, comment: string | NodeArray<JSDocComment> | undefined): JSDocClassTag;
    createJSDocPublicTag(tagName: Identifier | undefined, comment?: string | NodeArray<JSDocComment>): JSDocPublicTag;
    updateJSDocPublicTag(node: JSDocPublicTag, tagName: Identifier | undefined, comment: string | NodeArray<JSDocComment> | undefined): JSDocPublicTag;
    createJSDocPrivateTag(tagName: Identifier | undefined, comment?: string | NodeArray<JSDocComment>): JSDocPrivateTag;
    updateJSDocPrivateTag(node: JSDocPrivateTag, tagName: Identifier | undefined, comment: string | NodeArray<JSDocComment> | undefined): JSDocPrivateTag;
    createJSDocProtectedTag(tagName: Identifier | undefined, comment?: string | NodeArray<JSDocComment>): JSDocProtectedTag;
    updateJSDocProtectedTag(node: JSDocProtectedTag, tagName: Identifier | undefined, comment: string | NodeArray<JSDocComment> | undefined): JSDocProtectedTag;
    createJSDocReadonlyTag(tagName: Identifier | undefined, comment?: string | NodeArray<JSDocComment>): JSDocReadonlyTag;
    updateJSDocReadonlyTag(node: JSDocReadonlyTag, tagName: Identifier | undefined, comment: string | NodeArray<JSDocComment> | undefined): JSDocReadonlyTag;
    createJSDocUnknownTag(tagName: Identifier, comment?: string | NodeArray<JSDocComment>): JSDocUnknownTag;
    updateJSDocUnknownTag(node: JSDocUnknownTag, tagName: Identifier, comment: string | NodeArray<JSDocComment> | undefined): JSDocUnknownTag;
    createJSDocDeprecatedTag(tagName: Identifier | undefined, comment?: string | NodeArray<JSDocComment>): JSDocDeprecatedTag;
    updateJSDocDeprecatedTag(node: JSDocDeprecatedTag, tagName: Identifier | undefined, comment?: string | NodeArray<JSDocComment>): JSDocDeprecatedTag;
    createJSDocOverrideTag(tagName: Identifier | undefined, comment?: string | NodeArray<JSDocComment>): JSDocOverrideTag;
    updateJSDocOverrideTag(node: JSDocOverrideTag, tagName: Identifier | undefined, comment?: string | NodeArray<JSDocComment>): JSDocOverrideTag;
    createJSDocThrowsTag(tagName: Identifier, typeExpression: JSDocTypeExpression | undefined, comment?: string | NodeArray<JSDocComment>): JSDocThrowsTag;
    updateJSDocThrowsTag(node: JSDocThrowsTag, tagName: Identifier | undefined, typeExpression: JSDocTypeExpression | undefined, comment?: string | NodeArray<JSDocComment> | undefined): JSDocThrowsTag;
    createJSDocSatisfiesTag(tagName: Identifier | undefined, typeExpression: JSDocTypeExpression, comment?: string | NodeArray<JSDocComment>): JSDocSatisfiesTag;
    updateJSDocSatisfiesTag(node: JSDocSatisfiesTag, tagName: Identifier | undefined, typeExpression: JSDocTypeExpression, comment: string | NodeArray<JSDocComment> | undefined): JSDocSatisfiesTag;
    createJSDocText(text: string): JSDocText;
    updateJSDocText(node: JSDocText, text: string): JSDocText;
    createJSDocComment(comment?: string | NodeArray<JSDocComment> | undefined, tags?: readonly JSDocTag[] | undefined): JSDoc;
    updateJSDocComment(node: JSDoc, comment: string | NodeArray<JSDocComment> | undefined, tags: readonly JSDocTag[] | undefined): JSDoc;

    //
    // JSX
    //

    createJsxElement(openingElement: JsxOpeningElement, children: readonly JsxChild[], closingElement: JsxClosingElement): JsxElement;
    updateJsxElement(node: JsxElement, openingElement: JsxOpeningElement, children: readonly JsxChild[], closingElement: JsxClosingElement): JsxElement;
    createJsxSelfClosingElement(tagName: JsxTagNameExpression, typeArguments: readonly TypeNode[] | undefined, attributes: JsxAttributes): JsxSelfClosingElement;
    updateJsxSelfClosingElement(node: JsxSelfClosingElement, tagName: JsxTagNameExpression, typeArguments: readonly TypeNode[] | undefined, attributes: JsxAttributes): JsxSelfClosingElement;
    createJsxOpeningElement(tagName: JsxTagNameExpression, typeArguments: readonly TypeNode[] | undefined, attributes: JsxAttributes): JsxOpeningElement;
    updateJsxOpeningElement(node: JsxOpeningElement, tagName: JsxTagNameExpression, typeArguments: readonly TypeNode[] | undefined, attributes: JsxAttributes): JsxOpeningElement;
    createJsxClosingElement(tagName: JsxTagNameExpression): JsxClosingElement;
    updateJsxClosingElement(node: JsxClosingElement, tagName: JsxTagNameExpression): JsxClosingElement;
    createJsxFragment(openingFragment: JsxOpeningFragment, children: readonly JsxChild[], closingFragment: JsxClosingFragment): JsxFragment;
    createJsxText(text: string, containsOnlyTriviaWhiteSpaces?: boolean): JsxText;
    updateJsxText(node: JsxText, text: string, containsOnlyTriviaWhiteSpaces?: boolean): JsxText;
    createJsxOpeningFragment(): JsxOpeningFragment;
    createJsxJsxClosingFragment(): JsxClosingFragment;
    updateJsxFragment(node: JsxFragment, openingFragment: JsxOpeningFragment, children: readonly JsxChild[], closingFragment: JsxClosingFragment): JsxFragment;
    createJsxAttribute(name: JsxAttributeName, initializer: JsxAttributeValue | undefined): JsxAttribute;
    updateJsxAttribute(node: JsxAttribute, name: JsxAttributeName, initializer: JsxAttributeValue | undefined): JsxAttribute;
    createJsxAttributes(properties: readonly JsxAttributeLike[]): JsxAttributes;
    updateJsxAttributes(node: JsxAttributes, properties: readonly JsxAttributeLike[]): JsxAttributes;
    createJsxSpreadAttribute(expression: Expression): JsxSpreadAttribute;
    updateJsxSpreadAttribute(node: JsxSpreadAttribute, expression: Expression): JsxSpreadAttribute;
    createJsxExpression(dotDotDotToken: DotDotDotToken | undefined, expression: Expression | undefined): JsxExpression;
    updateJsxExpression(node: JsxExpression, expression: Expression | undefined): JsxExpression;
    createJsxNamespacedName(namespace: Identifier, name: Identifier): JsxNamespacedName;
    updateJsxNamespacedName(node: JsxNamespacedName, namespace: Identifier, name: Identifier): JsxNamespacedName;

    //
    // Clauses
    //

    createCaseClause(expression: Expression, statements: readonly Statement[]): CaseClause;
    updateCaseClause(node: CaseClause, expression: Expression, statements: readonly Statement[]): CaseClause;
    createDefaultClause(statements: readonly Statement[]): DefaultClause;
    updateDefaultClause(node: DefaultClause, statements: readonly Statement[]): DefaultClause;
    createHeritageClause(token: HeritageClause["token"], types: readonly ExpressionWithTypeArguments[]): HeritageClause;
    updateHeritageClause(node: HeritageClause, types: readonly ExpressionWithTypeArguments[]): HeritageClause;
    createCatchClause(variableDeclaration: string | BindingName | VariableDeclaration | undefined, block: Block): CatchClause;
    updateCatchClause(node: CatchClause, variableDeclaration: VariableDeclaration | undefined, block: Block): CatchClause;

    //
    // Property assignments
    //

    createPropertyAssignment(name: string | PropertyName, initializer: Expression): PropertyAssignment;
    updatePropertyAssignment(node: PropertyAssignment, name: PropertyName, initializer: Expression): PropertyAssignment;
    createShorthandPropertyAssignment(name: string | Identifier, objectAssignmentInitializer?: Expression): ShorthandPropertyAssignment;
    updateShorthandPropertyAssignment(node: ShorthandPropertyAssignment, name: Identifier, objectAssignmentInitializer: Expression | undefined): ShorthandPropertyAssignment;
    createSpreadAssignment(expression: Expression): SpreadAssignment;
    updateSpreadAssignment(node: SpreadAssignment, expression: Expression): SpreadAssignment;

    //
    // Enum
    //

    createEnumMember(name: string | PropertyName, initializer?: Expression): EnumMember;
    updateEnumMember(node: EnumMember, name: PropertyName, initializer: Expression | undefined): EnumMember;

    //
    // Top-level nodes
    //

    createSourceFile(statements: readonly Statement[], endOfFileToken: EndOfFileToken, flags: NodeFlags): SourceFile;
    updateSourceFile(node: SourceFile, statements: readonly Statement[], isDeclarationFile?: boolean, referencedFiles?: readonly FileReference[], typeReferences?: readonly FileReference[], hasNoDefaultLib?: boolean, libReferences?: readonly FileReference[]): SourceFile;

    //
    // Synthetic Nodes
    //
    /** @internal */ createSyntaxList(children: Node[]): SyntaxList;

    //
    // Transformation nodes
    //

    createNotEmittedStatement(original: Node): NotEmittedStatement;
    createPartiallyEmittedExpression(expression: Expression, original?: Node): PartiallyEmittedExpression;
    updatePartiallyEmittedExpression(node: PartiallyEmittedExpression, expression: Expression): PartiallyEmittedExpression;
    /** @internal */ createSyntheticReferenceExpression(expression: Expression, thisArg: Expression): SyntheticReferenceExpression;
    /** @internal */ updateSyntheticReferenceExpression(node: SyntheticReferenceExpression, expression: Expression, thisArg: Expression): SyntheticReferenceExpression;
    createCommaListExpression(elements: readonly Expression[]): CommaListExpression;
    updateCommaListExpression(node: CommaListExpression, elements: readonly Expression[]): CommaListExpression;

    //
    // Common operators
    //

    createComma(left: Expression, right: Expression): BinaryExpression;
    createAssignment(left: ObjectLiteralExpression | ArrayLiteralExpression, right: Expression): DestructuringAssignment;
    createAssignment(left: Expression, right: Expression): AssignmentExpression<EqualsToken>;
    createLogicalOr(left: Expression, right: Expression): BinaryExpression;
    createLogicalAnd(left: Expression, right: Expression): BinaryExpression;
    createBitwiseOr(left: Expression, right: Expression): BinaryExpression;
    createBitwiseXor(left: Expression, right: Expression): BinaryExpression;
    createBitwiseAnd(left: Expression, right: Expression): BinaryExpression;
    createStrictEquality(left: Expression, right: Expression): BinaryExpression;
    createStrictInequality(left: Expression, right: Expression): BinaryExpression;
    createEquality(left: Expression, right: Expression): BinaryExpression;
    createInequality(left: Expression, right: Expression): BinaryExpression;
    createLessThan(left: Expression, right: Expression): BinaryExpression;
    createLessThanEquals(left: Expression, right: Expression): BinaryExpression;
    createGreaterThan(left: Expression, right: Expression): BinaryExpression;
    createGreaterThanEquals(left: Expression, right: Expression): BinaryExpression;
    createLeftShift(left: Expression, right: Expression): BinaryExpression;
    createRightShift(left: Expression, right: Expression): BinaryExpression;
    createUnsignedRightShift(left: Expression, right: Expression): BinaryExpression;
    createAdd(left: Expression, right: Expression): BinaryExpression;
    createSubtract(left: Expression, right: Expression): BinaryExpression;
    createMultiply(left: Expression, right: Expression): BinaryExpression;
    createDivide(left: Expression, right: Expression): BinaryExpression;
    createModulo(left: Expression, right: Expression): BinaryExpression;
    createExponent(left: Expression, right: Expression): BinaryExpression;
    createPrefixPlus(operand: Expression): PrefixUnaryExpression;
    createPrefixMinus(operand: Expression): PrefixUnaryExpression;
    createPrefixIncrement(operand: Expression): PrefixUnaryExpression;
    createPrefixDecrement(operand: Expression): PrefixUnaryExpression;
    createBitwiseNot(operand: Expression): PrefixUnaryExpression;
    createLogicalNot(operand: Expression): PrefixUnaryExpression;
    createPostfixIncrement(operand: Expression): PostfixUnaryExpression;
    createPostfixDecrement(operand: Expression): PostfixUnaryExpression;

    //
    // Compound Nodes
    //

    createImmediatelyInvokedFunctionExpression(statements: readonly Statement[]): CallExpression;
    createImmediatelyInvokedFunctionExpression(statements: readonly Statement[], param: ParameterDeclaration, paramValue: Expression): CallExpression;
    createImmediatelyInvokedArrowFunction(statements: readonly Statement[]): ImmediatelyInvokedArrowFunction;
    createImmediatelyInvokedArrowFunction(statements: readonly Statement[], param: ParameterDeclaration, paramValue: Expression): ImmediatelyInvokedArrowFunction;

    createVoidZero(): VoidExpression;
    createExportDefault(expression: Expression): ExportAssignment;
    createExternalModuleExport(exportName: Identifier): ExportDeclaration;

    /** @internal */ createTypeCheck(value: Expression, tag: TypeOfTag): Expression;
    /** @internal */ createIsNotTypeCheck(value: Expression, tag: TypeOfTag): Expression;
    /** @internal */ createMethodCall(object: Expression, methodName: string | Identifier, argumentsList: readonly Expression[]): CallExpression;
    /** @internal */ createGlobalMethodCall(globalObjectName: string, globalMethodName: string, argumentsList: readonly Expression[]): CallExpression;
    /** @internal */ createFunctionBindCall(target: Expression, thisArg: Expression, argumentsList: readonly Expression[]): CallExpression;
    /** @internal */ createFunctionCallCall(target: Expression, thisArg: Expression, argumentsList: readonly Expression[]): CallExpression;
    /** @internal */ createFunctionApplyCall(target: Expression, thisArg: Expression, argumentsExpression: Expression): CallExpression;
    /** @internal */ createObjectDefinePropertyCall(target: Expression, propertyName: string | Expression, attributes: Expression): CallExpression;
    /** @internal */ createObjectGetOwnPropertyDescriptorCall(target: Expression, propertyName: string | Expression): CallExpression;
    /** @internal */ createReflectGetCall(target: Expression, propertyKey: Expression, receiver?: Expression): CallExpression;
    /** @internal */ createReflectSetCall(target: Expression, propertyKey: Expression, value: Expression, receiver?: Expression): CallExpression;
    /** @internal */ createPropertyDescriptor(attributes: PropertyDescriptorAttributes, singleLine?: boolean): ObjectLiteralExpression;
    /** @internal */ createArraySliceCall(array: Expression, start?: number | Expression): CallExpression;
    /** @internal */ createArrayConcatCall(array: Expression, values: readonly Expression[]): CallExpression;
    /** @internal */ createCallBinding(expression: Expression, recordTempVariable: (temp: Identifier) => void, languageVersion?: ScriptTarget, cacheIdentifiers?: boolean): CallBinding;
    /**
     * Wraps an expression that cannot be an assignment target in an expression that can be.
     *
     * Given a `paramName` of `_a`:
     * ```
     * Reflect.set(obj, "x", _a)
     * ```
     * Becomes
     * ```ts
     * ({ set value(_a) { Reflect.set(obj, "x", _a); } }).value
     * ```
     *
     * @param paramName
     * @param expression
     *
     * @internal
     */
    createAssignmentTargetWrapper(paramName: Identifier, expression: Expression): PropertyAccessExpression;
    /** @internal */ inlineExpressions(expressions: readonly Expression[]): Expression;
    /**
     * Gets the internal name of a declaration. This is primarily used for declarations that can be
     * referred to by name in the body of an ES5 class function body. An internal name will *never*
     * be prefixed with an module or namespace export modifier like "exports." when emitted as an
     * expression. An internal name will also *never* be renamed due to a collision with a block
     * scoped variable.
     *
     * @param node The declaration.
     * @param allowComments A value indicating whether comments may be emitted for the name.
     * @param allowSourceMaps A value indicating whether source maps may be emitted for the name.
     *
     * @internal
     */
    getInternalName(node: Declaration, allowComments?: boolean, allowSourceMaps?: boolean): Identifier;
    /**
     * Gets the local name of a declaration. This is primarily used for declarations that can be
     * referred to by name in the declaration's immediate scope (classes, enums, namespaces). A
     * local name will *never* be prefixed with an module or namespace export modifier like
     * "exports." when emitted as an expression.
     *
     * @param node The declaration.
     * @param allowComments A value indicating whether comments may be emitted for the name.
     * @param allowSourceMaps A value indicating whether source maps may be emitted for the name.
     * @param ignoreAssignedName Indicates that the assigned name of a declaration shouldn't be considered.
     *
     * @internal
     */
    getLocalName(node: Declaration, allowComments?: boolean, allowSourceMaps?: boolean, ignoreAssignedName?: boolean): Identifier;
    /**
     * Gets the export name of a declaration. This is primarily used for declarations that can be
     * referred to by name in the declaration's immediate scope (classes, enums, namespaces). An
     * export name will *always* be prefixed with a module or namespace export modifier like
     * `"exports."` when emitted as an expression if the name points to an exported symbol.
     *
     * @param node The declaration.
     * @param allowComments A value indicating whether comments may be emitted for the name.
     * @param allowSourceMaps A value indicating whether source maps may be emitted for the name.
     *
     * @internal
     */
    getExportName(node: Declaration, allowComments?: boolean, allowSourceMaps?: boolean): Identifier;
    /**
     * Gets the name of a declaration for use in declarations.
     *
     * @param node The declaration.
     * @param allowComments A value indicating whether comments may be emitted for the name.
     * @param allowSourceMaps A value indicating whether source maps may be emitted for the name.
     *
     * @internal
     */
    getDeclarationName(node: Declaration | undefined, allowComments?: boolean, allowSourceMaps?: boolean): Identifier;
    /**
     * Gets a namespace-qualified name for use in expressions.
     *
     * @param ns The namespace identifier.
     * @param name The name.
     * @param allowComments A value indicating whether comments may be emitted for the name.
     * @param allowSourceMaps A value indicating whether source maps may be emitted for the name.
     *
     * @internal
     */
    getNamespaceMemberName(ns: Identifier, name: Identifier, allowComments?: boolean, allowSourceMaps?: boolean): PropertyAccessExpression;
    /**
     * Gets the exported name of a declaration for use in expressions.
     *
     * An exported name will *always* be prefixed with an module or namespace export modifier like
     * "exports." if the name points to an exported symbol.
     *
     * @param ns The namespace identifier.
     * @param node The declaration.
     * @param allowComments A value indicating whether comments may be emitted for the name.
     * @param allowSourceMaps A value indicating whether source maps may be emitted for the name.
     *
     * @internal
     */
    getExternalModuleOrNamespaceExportName(ns: Identifier | undefined, node: Declaration, allowComments?: boolean, allowSourceMaps?: boolean): Identifier | PropertyAccessExpression;

    //
    // Utilities
    //

    restoreOuterExpressions(outerExpression: Expression | undefined, innerExpression: Expression, kinds?: OuterExpressionKinds): Expression;
    /** @internal */ restoreEnclosingLabel(node: Statement, outermostLabeledStatement: LabeledStatement | undefined, afterRestoreLabelCallback?: (node: LabeledStatement) => void): Statement;
    /** @internal */ createUseStrictPrologue(): PrologueDirective;
    /**
     * Copies any necessary standard and custom prologue-directives into target array.
     * @param source origin statements array
     * @param target result statements array
     * @param ensureUseStrict boolean determining whether the function need to add prologue-directives
     * @param visitor Optional callback used to visit any custom prologue directives.
     *
     * @internal
     */
    copyPrologue(source: readonly Statement[], target: Statement[], ensureUseStrict?: boolean, visitor?: (node: Node) => VisitResult<Node | undefined>): number;
    /**
     * Copies only the standard (string-expression) prologue-directives into the target statement-array.
     * @param source origin statements array
     * @param target result statements array
     * @param statementOffset The offset at which to begin the copy.
     * @param ensureUseStrict boolean determining whether the function need to add prologue-directives
     *
     * @internal
     */
    copyStandardPrologue(source: readonly Statement[], target: Statement[], statementOffset: number | undefined, ensureUseStrict?: boolean): number;
    /**
     * Copies only the custom prologue-directives into target statement-array.
     * @param source origin statements array
     * @param target result statements array
     * @param statementOffset The offset at which to begin the copy.
     * @param visitor Optional callback used to visit any custom prologue directives.
     *
     * @internal
     */
    copyCustomPrologue(source: readonly Statement[], target: Statement[], statementOffset: number, visitor?: (node: Node) => VisitResult<Node | undefined>, filter?: (node: Statement) => boolean): number;
    /** @internal */ copyCustomPrologue(source: readonly Statement[], target: Statement[], statementOffset: number | undefined, visitor?: (node: Node) => VisitResult<Node | undefined>, filter?: (node: Statement) => boolean): number | undefined;
    /** @internal */ ensureUseStrict(statements: NodeArray<Statement>): NodeArray<Statement>;
    /** @internal */ liftToBlock(nodes: readonly Node[]): Statement;
    /**
     * Merges generated lexical declarations into a new statement list.
     *
     * @internal
     */
    mergeLexicalEnvironment(statements: NodeArray<Statement>, declarations: readonly Statement[] | undefined): NodeArray<Statement>;
    /**
     * Appends generated lexical declarations to an array of statements.
     *
     * @internal
     */
    mergeLexicalEnvironment(statements: Statement[], declarations: readonly Statement[] | undefined): Statement[];
    /**
     * Creates a shallow, memberwise clone of a node.
     * - The result will have its `original` pointer set to `node`.
     * - The result will have its `pos` and `end` set to `-1`.
     * - *DO NOT USE THIS* if a more appropriate function is available.
     *
     * @internal
     */
    cloneNode<T extends Node | undefined>(node: T): T;
    /**
     * Updates a node that may contain modifiers, replacing only the modifiers of the node.
     */
    replaceModifiers<T extends HasModifiers>(node: T, modifiers: readonly Modifier[] | ModifierFlags | undefined): T;
    /**
     * Updates a node that may contain decorators or modifiers, replacing only the decorators and modifiers of the node.
     */
    replaceDecoratorsAndModifiers<T extends HasModifiers & HasDecorators>(node: T, modifiers: readonly ModifierLike[] | undefined): T;
    /**
     * Updates a node that contains a property name, replacing only the name of the node.
     */
    replacePropertyName<T extends AccessorDeclaration | MethodDeclaration | MethodSignature | PropertyDeclaration | PropertySignature | PropertyAssignment>(node: T, name: T["name"]): T;
}

/**
 * A function that accepts and possibly transforms a node.
 */
export type Visitor<TIn extends Node = Node, TOut extends Node | undefined = TIn | undefined> = (node: TIn) => VisitResult<TOut>;

export type VisitResult<T extends Node | undefined> = T | readonly Node[];

/** @deprecated @internal */
const enum BundleFileSectionKind {
    Prologue = "prologue",
    EmitHelpers = "emitHelpers",
    NoDefaultLib = "no-default-lib",
    Reference = "reference",
    Type = "type",
    TypeResolutionModeRequire = "type-require",
    TypeResolutionModeImport = "type-import",
    Lib = "lib",
    Prepend = "prepend",
    Text = "text",
    Internal = "internal",
    // comments?
}

/** @deprecated @internal */
interface BundleFileSectionBase extends TextRange {
    kind: BundleFileSectionKind;
    data?: string;
}

/** @deprecated @internal */
interface BundleFileHasNoDefaultLib extends BundleFileSectionBase {
    kind: BundleFileSectionKind.NoDefaultLib;
}

/** @deprecated @internal */
interface BundleFileReference extends BundleFileSectionBase {
    kind: BundleFileSectionKind.Reference | BundleFileSectionKind.Type | BundleFileSectionKind.Lib | BundleFileSectionKind.TypeResolutionModeImport | BundleFileSectionKind.TypeResolutionModeRequire;
    data: string;
}

/** @internal */
interface RawSourceMap {
    version: 3;
    file: string;
    sourceRoot?: string | null;
    sources: string[];
    sourcesContent?: (string | null)[] | null;
    mappings: string;
    names?: string[] | null;
}

export interface TextSpan {
    start: number;
    length: number;
}

export interface TextChangeRange {
    span: TextSpan;
    newLength: number;
}

// SyntaxKind.SyntaxList
export interface SyntaxList extends Node {
    kind: SyntaxKind.SyntaxList;
    _children: Node[];
}

/** @internal */
export const enum PragmaKindFlags {
    None = 0,
    /**
     * Triple slash comment of the form
     * /// <pragma-name argname="value" />
     */
    TripleSlashXML = 1 << 0,
    /**
     * Single line comment of the form
     * // @pragma-name argval1 argval2
     * or
     * /// @pragma-name argval1 argval2
     */
    SingleLine = 1 << 1,
    /**
     * Multiline non-jsdoc pragma of the form
     * /* @pragma-name argval1 argval2 * /
     */
    MultiLine = 1 << 2,
    All = TripleSlashXML | SingleLine | MultiLine,
    Default = All,
}

/** @internal */
interface PragmaArgumentSpecification<TName extends string> {
    name: TName; // Determines the name of the key in the resulting parsed type, type parameter to cause literal type inference
    optional?: boolean;
    captureSpan?: boolean;
}

/** @internal */
export interface PragmaDefinition<T1 extends string = string, T2 extends string = string, T3 extends string = string, T4 extends string = string> {
    args?:
        | readonly [PragmaArgumentSpecification<T1>]
        | readonly [PragmaArgumentSpecification<T1>, PragmaArgumentSpecification<T2>]
        | readonly [PragmaArgumentSpecification<T1>, PragmaArgumentSpecification<T2>, PragmaArgumentSpecification<T3>]
        | readonly [PragmaArgumentSpecification<T1>, PragmaArgumentSpecification<T2>, PragmaArgumentSpecification<T3>, PragmaArgumentSpecification<T4>];
    // If not present, defaults to PragmaKindFlags.Default
    kind?: PragmaKindFlags;
}

// While not strictly a type, this is here because `PragmaMap` needs to be here to be used with `SourceFile`, and we don't
//  fancy effectively defining it twice, once in value-space and once in type-space
/** @internal */
export const commentPragmas = {
    "reference": {
        args: [
            { name: "types", optional: true, captureSpan: true },
            { name: "lib", optional: true, captureSpan: true },
            { name: "path", optional: true, captureSpan: true },
            { name: "no-default-lib", optional: true },
            { name: "resolution-mode", optional: true },
        ],
        kind: PragmaKindFlags.TripleSlashXML,
    },
    "amd-dependency": {
        args: [{ name: "path" }, { name: "name", optional: true }],
        kind: PragmaKindFlags.TripleSlashXML,
    },
    "amd-module": {
        args: [{ name: "name" }],
        kind: PragmaKindFlags.TripleSlashXML,
    },
    "ts-check": {
        kind: PragmaKindFlags.SingleLine,
    },
    "ts-nocheck": {
        kind: PragmaKindFlags.SingleLine,
    },
    "jsx": {
        args: [{ name: "factory" }],
        kind: PragmaKindFlags.MultiLine,
    },
    "jsxfrag": {
        args: [{ name: "factory" }],
        kind: PragmaKindFlags.MultiLine,
    },
    "jsximportsource": {
        args: [{ name: "factory" }],
        kind: PragmaKindFlags.MultiLine,
    },
    "jsxruntime": {
        args: [{ name: "factory" }],
        kind: PragmaKindFlags.MultiLine,
    },
} as const;

export const enum JSDocParsingMode {
    /**
     * Always parse JSDoc comments and include them in the AST.
     *
     * This is the default if no mode is provided.
     */
    ParseAll,
    /**
     * Never parse JSDoc comments, mo matter the file type.
     */
    ParseNone,
    /**
     * Parse only JSDoc comments which are needed to provide correct type errors.
     *
     * This will always parse JSDoc in non-TS files, but only parse JSDoc comments
     * containing `@see` and `@link` in TS files.
     */
    ParseForTypeErrors,
    /**
     * Parse only JSDoc comments which are needed to provide correct type info.
     *
     * This will always parse JSDoc in non-TS files, but never in TS files.
     *
     * Note: Do not use this mode if you require accurate type errors; use {@link ParseForTypeErrors} instead.
     */
    ParseForTypeInfo,
}

/** @internal */
type PragmaArgTypeMaybeCapture<TDesc> = TDesc extends { captureSpan: true; } ? { value: string; pos: number; end: number; } : string;

/** @internal */
type PragmaArgTypeOptional<TDesc, TName extends string> = TDesc extends { optional: true; } ? { [K in TName]?: PragmaArgTypeMaybeCapture<TDesc>; }
    : { [K in TName]: PragmaArgTypeMaybeCapture<TDesc>; };

/** @internal */
type UnionToIntersection<U> = (U extends any ? (k: U) => void : never) extends ((k: infer I) => void) ? I : never;

/** @internal */
type ArgumentDefinitionToFieldUnion<T extends readonly PragmaArgumentSpecification<any>[]> = {
    [K in keyof T]: PragmaArgTypeOptional<T[K], T[K] extends { name: infer TName; } ? TName extends string ? TName : never : never>;
}[Extract<keyof T, number>]; // The mapped type maps over only the tuple members, but this reindex gets _all_ members - by extracting only `number` keys, we get only the tuple members

/**
 * Maps a pragma definition into the desired shape for its arguments object
 *
 * @internal
 */
type PragmaArgumentType<KPrag extends keyof ConcretePragmaSpecs> = ConcretePragmaSpecs[KPrag] extends { args: readonly PragmaArgumentSpecification<any>[]; } ? UnionToIntersection<ArgumentDefinitionToFieldUnion<ConcretePragmaSpecs[KPrag]["args"]>>
    : never;

/** @internal */
type ConcretePragmaSpecs = typeof commentPragmas;

/** @internal */
export type PragmaPseudoMap = { [K in keyof ConcretePragmaSpecs]: { arguments: PragmaArgumentType<K>; range: CommentRange; }; };

/** @internal */
export type PragmaPseudoMapEntry = { [K in keyof PragmaPseudoMap]: { name: K; args: PragmaPseudoMap[K]; }; }[keyof PragmaPseudoMap];

/** @internal */
export interface ReadonlyPragmaMap extends ReadonlyMap<string, PragmaPseudoMap[keyof PragmaPseudoMap] | PragmaPseudoMap[keyof PragmaPseudoMap][]> {
    get<TKey extends keyof PragmaPseudoMap>(key: TKey): PragmaPseudoMap[TKey] | PragmaPseudoMap[TKey][];
    forEach(action: <TKey extends keyof PragmaPseudoMap>(value: PragmaPseudoMap[TKey] | PragmaPseudoMap[TKey][], key: TKey, map: ReadonlyPragmaMap) => void): void;
}

/**
 * A strongly-typed es6 map of pragma entries, the values of which are either a single argument
 * value (if only one was found), or an array of multiple argument values if the pragma is present
 * in multiple places
 *
 * @internal
 */
export interface PragmaMap extends Map<string, PragmaPseudoMap[keyof PragmaPseudoMap] | PragmaPseudoMap[keyof PragmaPseudoMap][]>, ReadonlyPragmaMap {
    set<TKey extends keyof PragmaPseudoMap>(key: TKey, value: PragmaPseudoMap[TKey] | PragmaPseudoMap[TKey][]): this;
    get<TKey extends keyof PragmaPseudoMap>(key: TKey): PragmaPseudoMap[TKey] | PragmaPseudoMap[TKey][];
    forEach(action: <TKey extends keyof PragmaPseudoMap>(value: PragmaPseudoMap[TKey] | PragmaPseudoMap[TKey][], key: TKey, map: PragmaMap) => void): void;
}

/** Represents a bigint literal value without requiring bigint support */
export interface PseudoBigInt {
    negative: boolean;
    base10Value: string;
}
