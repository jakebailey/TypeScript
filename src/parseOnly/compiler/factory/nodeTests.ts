import {
    ArrayLiteralExpression,
    ArrowFunction,
    AsyncKeyword,
    BinaryExpression,
    BindingElement,
    Block,
    CallExpression,
    ClassDeclaration,
    ClassExpression,
    CommaListExpression,
    ComputedPropertyName,
    ConditionalTypeNode,
    ConstructorDeclaration,
    ConstructorTypeNode,
    ElementAccessExpression,
    EnumDeclaration,
    ExclamationToken,
    ExportAssignment,
    ExportDeclaration,
    ExportKeyword,
    ExpressionStatement,
    ExpressionWithTypeArguments,
    ExternalModuleReference,
    FunctionDeclaration,
    FunctionExpression,
    FunctionTypeNode,
    GetAccessorDeclaration,
    HeritageClause,
    Identifier,
    ImportDeclaration,
    ImportEqualsDeclaration,
    ImportExpression,
    IndexSignatureDeclaration,
    InferTypeNode,
    InterfaceDeclaration,
    IntersectionTypeNode,
    JSDoc,
    JSDocAugmentsTag,
    JSDocDeprecatedTag,
    JSDocFunctionType,
    JSDocImplementsTag,
    JSDocMemberName,
    JSDocNameReference,
    JSDocNullableType,
    JSDocOverloadTag,
    JSDocOverrideTag,
    JSDocParameterTag,
    JSDocPrivateTag,
    JSDocProtectedTag,
    JSDocPublicTag,
    JSDocReadonlyTag,
    JSDocReturnTag,
    JSDocSatisfiesTag,
    JSDocSignature,
    JSDocTemplateTag,
    JSDocTypeExpression,
    JSDocTypeTag,
    JsxNamespacedName,
    JsxOpeningElement,
    JsxOpeningFragment,
    JsxText,
    LabeledStatement,
    MetaProperty,
    MethodDeclaration,
    MethodSignature,
    ModuleDeclaration,
    NamedTupleMember,
    Node,
    NonNullExpression,
    NoSubstitutionTemplateLiteral,
    NotEmittedStatement,
    NumericLiteral,
    ObjectLiteralExpression,
    OmittedExpression,
    ParameterDeclaration,
    ParenthesizedExpression,
    PrefixUnaryExpression,
    PrivateIdentifier,
    PropertyAccessExpression,
    PropertyAssignment,
    PropertyDeclaration,
    PropertySignature,
    QuestionToken,
    SetAccessorDeclaration,
    SourceFile,
    SpreadElement,
    StringLiteral,
    SuperExpression,
    SyntaxKind,
    TaggedTemplateExpression,
    Token,
    TypeAliasDeclaration,
    TypeOperatorNode,
    TypeParameterDeclaration,
    TypeReferenceNode,
    UnionTypeNode,
    VariableDeclaration,
    VariableStatement,
    VoidExpression,
} from "../types";
// Literals

export function isNumericLiteral(node: Node): node is NumericLiteral {
    return node.kind === SyntaxKind.NumericLiteral;
}

export function isStringLiteral(node: Node): node is StringLiteral {
    return node.kind === SyntaxKind.StringLiteral;
}

export function isJsxText(node: Node): node is JsxText {
    return node.kind === SyntaxKind.JsxText;
}

export function isNoSubstitutionTemplateLiteral(node: Node): node is NoSubstitutionTemplateLiteral {
    return node.kind === SyntaxKind.NoSubstitutionTemplateLiteral;
}

// Pseudo-literals

// Punctuation

/** @internal */
export function isCommaToken(node: Node): node is Token<SyntaxKind.CommaToken> {
    return node.kind === SyntaxKind.CommaToken;
}

export function isExclamationToken(node: Node): node is ExclamationToken {
    return node.kind === SyntaxKind.ExclamationToken;
}

export function isQuestionToken(node: Node): node is QuestionToken {
    return node.kind === SyntaxKind.QuestionToken;
}

// Identifiers

export function isIdentifier(node: Node): node is Identifier {
    return node.kind === SyntaxKind.Identifier;
}

export function isPrivateIdentifier(node: Node): node is PrivateIdentifier {
    return node.kind === SyntaxKind.PrivateIdentifier;
}

// Reserved Words

/** @internal */
export function isExportModifier(node: Node): node is ExportKeyword {
    return node.kind === SyntaxKind.ExportKeyword;
}

/** @internal */
export function isAsyncModifier(node: Node): node is AsyncKeyword {
    return node.kind === SyntaxKind.AsyncKeyword;
}

/** @internal */
export function isSuperKeyword(node: Node): node is SuperExpression {
    return node.kind === SyntaxKind.SuperKeyword;
}

/** @internal */
export function isImportKeyword(node: Node): node is ImportExpression {
    return node.kind === SyntaxKind.ImportKeyword;
}

// Names

export function isComputedPropertyName(node: Node): node is ComputedPropertyName {
    return node.kind === SyntaxKind.ComputedPropertyName;
}

// Signature elements

export function isTypeParameterDeclaration(node: Node): node is TypeParameterDeclaration {
    return node.kind === SyntaxKind.TypeParameter;
}

// TODO(rbuckton): Rename to 'isParameterDeclaration'
export function isParameter(node: Node): node is ParameterDeclaration {
    return node.kind === SyntaxKind.Parameter;
}

// TypeMember

export function isPropertySignature(node: Node): node is PropertySignature {
    return node.kind === SyntaxKind.PropertySignature;
}

export function isPropertyDeclaration(node: Node): node is PropertyDeclaration {
    return node.kind === SyntaxKind.PropertyDeclaration;
}

export function isMethodSignature(node: Node): node is MethodSignature {
    return node.kind === SyntaxKind.MethodSignature;
}

export function isMethodDeclaration(node: Node): node is MethodDeclaration {
    return node.kind === SyntaxKind.MethodDeclaration;
}

export function isConstructorDeclaration(node: Node): node is ConstructorDeclaration {
    return node.kind === SyntaxKind.Constructor;
}

export function isGetAccessorDeclaration(node: Node): node is GetAccessorDeclaration {
    return node.kind === SyntaxKind.GetAccessor;
}

export function isSetAccessorDeclaration(node: Node): node is SetAccessorDeclaration {
    return node.kind === SyntaxKind.SetAccessor;
}

export function isIndexSignatureDeclaration(node: Node): node is IndexSignatureDeclaration {
    return node.kind === SyntaxKind.IndexSignature;
}

// Type

export function isTypeReferenceNode(node: Node): node is TypeReferenceNode {
    return node.kind === SyntaxKind.TypeReference;
}

export function isFunctionTypeNode(node: Node): node is FunctionTypeNode {
    return node.kind === SyntaxKind.FunctionType;
}

export function isConstructorTypeNode(node: Node): node is ConstructorTypeNode {
    return node.kind === SyntaxKind.ConstructorType;
}

export function isNamedTupleMember(node: Node): node is NamedTupleMember {
    return node.kind === SyntaxKind.NamedTupleMember;
}

export function isUnionTypeNode(node: Node): node is UnionTypeNode {
    return node.kind === SyntaxKind.UnionType;
}

export function isIntersectionTypeNode(node: Node): node is IntersectionTypeNode {
    return node.kind === SyntaxKind.IntersectionType;
}

export function isConditionalTypeNode(node: Node): node is ConditionalTypeNode {
    return node.kind === SyntaxKind.ConditionalType;
}

export function isInferTypeNode(node: Node): node is InferTypeNode {
    return node.kind === SyntaxKind.InferType;
}

export function isTypeOperatorNode(node: Node): node is TypeOperatorNode {
    return node.kind === SyntaxKind.TypeOperator;
}

// Binding patterns

export function isBindingElement(node: Node): node is BindingElement {
    return node.kind === SyntaxKind.BindingElement;
}

// Expression

export function isArrayLiteralExpression(node: Node): node is ArrayLiteralExpression {
    return node.kind === SyntaxKind.ArrayLiteralExpression;
}

export function isObjectLiteralExpression(node: Node): node is ObjectLiteralExpression {
    return node.kind === SyntaxKind.ObjectLiteralExpression;
}

export function isPropertyAccessExpression(node: Node): node is PropertyAccessExpression {
    return node.kind === SyntaxKind.PropertyAccessExpression;
}

export function isElementAccessExpression(node: Node): node is ElementAccessExpression {
    return node.kind === SyntaxKind.ElementAccessExpression;
}

export function isCallExpression(node: Node): node is CallExpression {
    return node.kind === SyntaxKind.CallExpression;
}

export function isTaggedTemplateExpression(node: Node): node is TaggedTemplateExpression {
    return node.kind === SyntaxKind.TaggedTemplateExpression;
}

export function isParenthesizedExpression(node: Node): node is ParenthesizedExpression {
    return node.kind === SyntaxKind.ParenthesizedExpression;
}

export function isFunctionExpression(node: Node): node is FunctionExpression {
    return node.kind === SyntaxKind.FunctionExpression;
}

export function isArrowFunction(node: Node): node is ArrowFunction {
    return node.kind === SyntaxKind.ArrowFunction;
}

export function isVoidExpression(node: Node): node is VoidExpression {
    return node.kind === SyntaxKind.VoidExpression;
}

export function isPrefixUnaryExpression(node: Node): node is PrefixUnaryExpression {
    return node.kind === SyntaxKind.PrefixUnaryExpression;
}

export function isBinaryExpression(node: Node): node is BinaryExpression {
    return node.kind === SyntaxKind.BinaryExpression;
}

export function isSpreadElement(node: Node): node is SpreadElement {
    return node.kind === SyntaxKind.SpreadElement;
}

export function isClassExpression(node: Node): node is ClassExpression {
    return node.kind === SyntaxKind.ClassExpression;
}

export function isOmittedExpression(node: Node): node is OmittedExpression {
    return node.kind === SyntaxKind.OmittedExpression;
}

export function isExpressionWithTypeArguments(node: Node): node is ExpressionWithTypeArguments {
    return node.kind === SyntaxKind.ExpressionWithTypeArguments;
}

export function isNonNullExpression(node: Node): node is NonNullExpression {
    return node.kind === SyntaxKind.NonNullExpression;
}

export function isMetaProperty(node: Node): node is MetaProperty {
    return node.kind === SyntaxKind.MetaProperty;
}

export function isCommaListExpression(node: Node): node is CommaListExpression {
    return node.kind === SyntaxKind.CommaListExpression;
}

// Misc

// Elements

export function isBlock(node: Node): node is Block {
    return node.kind === SyntaxKind.Block;
}

export function isVariableStatement(node: Node): node is VariableStatement {
    return node.kind === SyntaxKind.VariableStatement;
}

export function isExpressionStatement(node: Node): node is ExpressionStatement {
    return node.kind === SyntaxKind.ExpressionStatement;
}

export function isLabeledStatement(node: Node): node is LabeledStatement {
    return node.kind === SyntaxKind.LabeledStatement;
}

export function isVariableDeclaration(node: Node): node is VariableDeclaration {
    return node.kind === SyntaxKind.VariableDeclaration;
}

export function isFunctionDeclaration(node: Node): node is FunctionDeclaration {
    return node.kind === SyntaxKind.FunctionDeclaration;
}

export function isClassDeclaration(node: Node): node is ClassDeclaration {
    return node.kind === SyntaxKind.ClassDeclaration;
}

export function isInterfaceDeclaration(node: Node): node is InterfaceDeclaration {
    return node.kind === SyntaxKind.InterfaceDeclaration;
}

export function isTypeAliasDeclaration(node: Node): node is TypeAliasDeclaration {
    return node.kind === SyntaxKind.TypeAliasDeclaration;
}

export function isEnumDeclaration(node: Node): node is EnumDeclaration {
    return node.kind === SyntaxKind.EnumDeclaration;
}

export function isModuleDeclaration(node: Node): node is ModuleDeclaration {
    return node.kind === SyntaxKind.ModuleDeclaration;
}

export function isImportEqualsDeclaration(node: Node): node is ImportEqualsDeclaration {
    return node.kind === SyntaxKind.ImportEqualsDeclaration;
}

export function isImportDeclaration(node: Node): node is ImportDeclaration {
    return node.kind === SyntaxKind.ImportDeclaration;
}

export function isExportAssignment(node: Node): node is ExportAssignment {
    return node.kind === SyntaxKind.ExportAssignment;
}

export function isExportDeclaration(node: Node): node is ExportDeclaration {
    return node.kind === SyntaxKind.ExportDeclaration;
}

export function isNotEmittedStatement(node: Node): node is NotEmittedStatement {
    return node.kind === SyntaxKind.NotEmittedStatement;
}

// Module References

export function isExternalModuleReference(node: Node): node is ExternalModuleReference {
    return node.kind === SyntaxKind.ExternalModuleReference;
}

// JSX

export function isJsxOpeningElement(node: Node): node is JsxOpeningElement {
    return node.kind === SyntaxKind.JsxOpeningElement;
}

export function isJsxOpeningFragment(node: Node): node is JsxOpeningFragment {
    return node.kind === SyntaxKind.JsxOpeningFragment;
}

export function isJsxNamespacedName(node: Node): node is JsxNamespacedName {
    return node.kind === SyntaxKind.JsxNamespacedName;
}

// Clauses

export function isHeritageClause(node: Node): node is HeritageClause {
    return node.kind === SyntaxKind.HeritageClause;
}

// Property assignments

export function isPropertyAssignment(node: Node): node is PropertyAssignment {
    return node.kind === SyntaxKind.PropertyAssignment;
}

// Enum

// Unparsed

// TODO(rbuckton): isUnparsedPrologue

// TODO(rbuckton): isUnparsedText
// TODO(rbuckton): isUnparsedInternalText
// TODO(rbuckton): isUnparsedSyntheticReference

// Top-level nodes
export function isSourceFile(node: Node): node is SourceFile {
    return node.kind === SyntaxKind.SourceFile;
}

// TODO(rbuckton): isInputFiles

// JSDoc Elements

export function isJSDocTypeExpression(node: Node): node is JSDocTypeExpression {
    return node.kind === SyntaxKind.JSDocTypeExpression;
}

export function isJSDocNameReference(node: Node): node is JSDocNameReference {
    return node.kind === SyntaxKind.JSDocNameReference;
}

export function isJSDocMemberName(node: Node): node is JSDocMemberName {
    return node.kind === SyntaxKind.JSDocMemberName;
}

export function isJSDocNullableType(node: Node): node is JSDocNullableType {
    return node.kind === SyntaxKind.JSDocNullableType;
}

export function isJSDocFunctionType(node: Node): node is JSDocFunctionType {
    return node.kind === SyntaxKind.JSDocFunctionType;
}

export function isJSDoc(node: Node): node is JSDoc {
    return node.kind === SyntaxKind.JSDoc;
}

export function isJSDocSignature(node: Node): node is JSDocSignature {
    return node.kind === SyntaxKind.JSDocSignature;
}

// JSDoc Tags

export function isJSDocAugmentsTag(node: Node): node is JSDocAugmentsTag {
    return node.kind === SyntaxKind.JSDocAugmentsTag;
}

export function isJSDocPublicTag(node: Node): node is JSDocPublicTag {
    return node.kind === SyntaxKind.JSDocPublicTag;
}

export function isJSDocPrivateTag(node: Node): node is JSDocPrivateTag {
    return node.kind === SyntaxKind.JSDocPrivateTag;
}

export function isJSDocProtectedTag(node: Node): node is JSDocProtectedTag {
    return node.kind === SyntaxKind.JSDocProtectedTag;
}

export function isJSDocReadonlyTag(node: Node): node is JSDocReadonlyTag {
    return node.kind === SyntaxKind.JSDocReadonlyTag;
}

export function isJSDocOverrideTag(node: Node): node is JSDocOverrideTag {
    return node.kind === SyntaxKind.JSDocOverrideTag;
}

export function isJSDocOverloadTag(node: Node): node is JSDocOverloadTag {
    return node.kind === SyntaxKind.JSDocOverloadTag;
}

export function isJSDocDeprecatedTag(node: Node): node is JSDocDeprecatedTag {
    return node.kind === SyntaxKind.JSDocDeprecatedTag;
}

export function isJSDocParameterTag(node: Node): node is JSDocParameterTag {
    return node.kind === SyntaxKind.JSDocParameterTag;
}

export function isJSDocReturnTag(node: Node): node is JSDocReturnTag {
    return node.kind === SyntaxKind.JSDocReturnTag;
}

export function isJSDocTypeTag(node: Node): node is JSDocTypeTag {
    return node.kind === SyntaxKind.JSDocTypeTag;
}

export function isJSDocTemplateTag(node: Node): node is JSDocTemplateTag {
    return node.kind === SyntaxKind.JSDocTemplateTag;
}

export function isJSDocImplementsTag(node: Node): node is JSDocImplementsTag {
    return node.kind === SyntaxKind.JSDocImplementsTag;
}

export function isJSDocSatisfiesTag(node: Node): node is JSDocSatisfiesTag {
    return node.kind === SyntaxKind.JSDocSatisfiesTag;
}

// Synthesized list
