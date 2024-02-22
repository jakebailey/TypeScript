import {
    addRange,
    concatenate,
    contains,
    emptyArray,
    every,
    filter,
    firstOrUndefined,
    flatMap,
    last,
    lastOrUndefined,
    length,
    some,
} from "./core";
import {
    MapLike,
} from "./corePublic";
import {
    Debug,
} from "./debug";
import {
    isBinaryExpression,
    isCallExpression,
    isClassDeclaration,
    isElementAccessExpression,
    isExpressionStatement,
    isExpressionWithTypeArguments,
    isFunctionDeclaration,
    isHeritageClause,
    isIdentifier,
    isJSDoc,
    isJSDocAugmentsTag,
    isJSDocImplementsTag,
    isJSDocMemberName,
    isJSDocNameReference,
    isJSDocOverloadTag,
    isJSDocSatisfiesTag,
    isJSDocSignature,
    isJSDocTypeExpression,
    isJSDocTypeTag,
    isJsxNamespacedName,
    isJsxText,
    isModuleDeclaration,
    isNoSubstitutionTemplateLiteral,
    isNumericLiteral,
    isObjectLiteralExpression,
    isParameter,
    isParenthesizedExpression,
    isPrefixUnaryExpression,
    isPrivateIdentifier,
    isPropertyAccessExpression,
    isVariableStatement,
    isVoidExpression,
} from "./factory/nodeTests";
import {
    skipOuterExpressions,
} from "./factory/utilities";
import {
    canHaveModifiers,
} from "./factory/utilitiesPublic";
import {
    forEachChild,
    forEachChildRecursively,
} from "./parser";
import {
    createScanner,
    getLeadingCommentRanges,
    getLineAndCharacterOfPosition,
    getLineStarts,
    getTrailingCommentRanges,
    isLineBreak,
    skipTrivia,
} from "./scanner";

import {
    __String,
    AccessExpression,
    ArrowFunction,
    AsExpression,
    AssertionExpression,
    AssignmentDeclarationKind,
    AssignmentExpression,
    AssignmentOperatorToken,
    BinaryExpression,
    BindableObjectDefinePropertyCall,
    BindableStaticAccessExpression,
    BindableStaticElementAccessExpression,
    BindableStaticNameExpression,
    CallExpression,
    CaseOrDefaultClause,
    CharacterCodes,
    ClassDeclaration,
    ClassElement,
    ClassExpression,
    ClassLikeDeclaration,
    ComputedPropertyName,
    ConditionalExpression,
    DeclarationName,
    Diagnostic,
    DiagnosticArguments,
    DiagnosticMessage,
    DiagnosticRelatedInformation,
    DiagnosticWithDetachedLocation,
    DiagnosticWithLocation,
    ElementAccessExpression,
    EmitFlags,
    EntityNameExpression,
    EqualsToken,
    Expression,
    ExpressionStatement,
    ExpressionWithTypeArguments,
    Extension,
    ForInOrOfStatement,
    ForStatement,
    FunctionLikeDeclaration,
    HasInitializer,
    HasJSDoc,
    HasType,
    Identifier,
    ImportTypeNode,
    JSDoc,
    JSDocArray,
    JSDocSatisfiesTag,
    JSDocSignature,
    JSDocTag,
    JSDocTemplateTag,
    JsxAttributeName,
    JsxNamespacedName,
    JsxOpeningLikeElement,
    KeywordSyntaxKind,
    LanguageVariant,
    LiteralLikeElementAccessExpression,
    LogicalOrCoalescingAssignmentOperator,
    ModifierFlags,
    ModifierLike,
    NamedDeclaration,
    NewExpression,
    Node,
    NodeFlags,
    NonNullExpression,
    NoSubstitutionTemplateLiteral,
    NumericLiteral,
    OuterExpressionKinds,
    ParameterDeclaration,
    PartiallyEmittedExpression,
    PostfixUnaryExpression,
    PrefixUnaryExpression,
    PrivateIdentifier,
    PrologueDirective,
    PropertyAccessEntityNameExpression,
    PropertyAccessExpression,
    PropertyAssignment,
    PropertyDeclaration,
    PropertyName,
    PropertyNameLiteral,
    PseudoBigInt,
    PunctuationOrKeywordSyntaxKind,
    PunctuationSyntaxKind,
    QualifiedName,
    ReadonlyTextRange,
    ReturnStatement,
    SatisfiesExpression,
    ScriptKind,
    ShorthandPropertyAssignment,
    SignatureDeclaration,
    SourceFile,
    SourceFileLike,
    Statement,
    StringLiteralLike,
    SuperProperty,
    SyntaxKind,
    TaggedTemplateExpression,
    TemplateLiteral,
    TemplateLiteralToken,
    TemplateSpan,
    TextRange,
    TextSpan,
    Token,
    TokenFlags,
    TransformFlags,
    TypeAssertion,
    TypeParameterDeclaration,
    VariableDeclaration,
    VariableLikeDeclaration,
    YieldExpression,
} from "./types";
import {
    createTextSpan,
    createTextSpanFromBounds,
    escapeLeadingUnderscores,
    findAncestor,
    getJSDocDeprecatedTagNoCache,
    getJSDocOverrideTagNoCache,
    getJSDocParameterTags,
    getJSDocParameterTagsNoCache,
    getJSDocPrivateTagNoCache,
    getJSDocProtectedTagNoCache,
    getJSDocPublicTagNoCache,
    getJSDocReadonlyTagNoCache,
    getJSDocTypeParameterTags,
    getJSDocTypeParameterTagsNoCache,
    hasInitializer,
    hasJSDocNodes,
    idText,
    isClassLike,
    isFunctionLike,
    isJSDocLinkLike,
    isJSDocNode,
    isLeftHandSideExpression,
    isMemberName,
    isNamedDeclaration,
    isStringLiteralLike,
} from "./utilitiesPublic";

/** @internal */
export function getFullWidth(node: Node) {
    return node.end - node.pos;
}

// Returns true if this node contains a parse error anywhere underneath it.
/** @internal */
export function containsParseError(node: Node): boolean {
    aggregateChildData(node);
    return (node.flags & NodeFlags.ThisNodeOrAnySubNodesHasError) !== 0;
}

function aggregateChildData(node: Node): void {
    if (!(node.flags & NodeFlags.HasAggregatedChildData)) {
        // A node is considered to contain a parse error if:
        //  a) the parser explicitly marked that it had an error
        //  b) any of it's children reported that it had an error.
        const thisNodeOrAnySubNodesHasError = ((node.flags & NodeFlags.ThisNodeHasError) !== 0) ||
            forEachChild(node, containsParseError);

        // If so, mark ourselves accordingly.
        if (thisNodeOrAnySubNodesHasError) {
            (node as Mutable<Node>).flags |= NodeFlags.ThisNodeOrAnySubNodesHasError;
        }

        // Also mark that we've propagated the child information to this node.  This way we can
        // always consult the bit directly on this node without needing to check its children
        // again.
        (node as Mutable<Node>).flags |= NodeFlags.HasAggregatedChildData;
    }
}

/** @internal */
export function getSourceFileOfNode(node: Node): SourceFile;
/** @internal */
export function getSourceFileOfNode(node: Node | undefined): SourceFile | undefined;
/** @internal */
export function getSourceFileOfNode(node: Node | undefined): SourceFile | undefined {
    while (node && node.kind !== SyntaxKind.SourceFile) {
        node = node.parent;
    }
    return node as SourceFile;
}

// This is a useful function for debugging purposes.

/** @internal */
function getEndLinePosition(line: number, sourceFile: SourceFileLike): number {
    Debug.assert(line >= 0);
    const lineStarts = getLineStarts(sourceFile);

    const lineIndex = line;
    const sourceText = sourceFile.text;
    if (lineIndex + 1 === lineStarts.length) {
        // last line - return EOF
        return sourceText.length - 1;
    }
    else {
        // current line start
        const start = lineStarts[lineIndex];
        // take the start position of the next line - 1 = it should be some line break
        let pos = lineStarts[lineIndex + 1] - 1;
        Debug.assert(isLineBreak(sourceText.charCodeAt(pos)));
        // walk backwards skipping line breaks, stop the the beginning of current line.
        // i.e:
        // <some text>
        // $ <- end of line for this position should match the start position
        while (start <= pos && isLineBreak(sourceText.charCodeAt(pos))) {
            pos--;
        }
        return pos;
    }
}

// Returns true if this node is missing from the actual source code. A 'missing' node is different
// from 'undefined/defined'. When a node is undefined (which can happen for optional nodes
// in the tree), it is definitely missing. However, a node may be defined, but still be
// missing.  This happens whenever the parser knows it needs to parse something, but can't
// get anything in the source code that it expects at that location. For example:
//
//          let a: ;
//
// Here, the Type in the Type-Annotation is not-optional (as there is a colon in the source
// code). So the parser will attempt to parse out a type, and will create an actual node.
// However, this node will be 'missing' in the sense that no actual source-code/tokens are
// contained within it.
/** @internal */
export function nodeIsMissing(node: Node | undefined): boolean {
    if (node === undefined) {
        return true;
    }

    return node.pos === node.end && node.pos >= 0 && node.kind !== SyntaxKind.EndOfFileToken;
}

/** @internal */
export function nodeIsPresent(node: Node | undefined): boolean {
    return !nodeIsMissing(node);
}

/** @internal */
function getSourceTextOfNodeFromSourceFile(sourceFile: SourceFile, node: Node, includeTrivia = false): string {
    return getTextOfNodeFromSourceText(sourceFile.text, node, includeTrivia);
}

function isJSDocTypeExpressionOrChild(node: Node): boolean {
    return !!findAncestor(node, isJSDocTypeExpression);
}

/** @internal */
export function getTextOfNodeFromSourceText(sourceText: string, node: Node, includeTrivia = false): string {
    if (nodeIsMissing(node)) {
        return "";
    }

    let text = sourceText.substring(includeTrivia ? node.pos : skipTrivia(sourceText, node.pos), node.end);

    if (isJSDocTypeExpressionOrChild(node)) {
        // strip space + asterisk at line start
        text = text.split(/\r\n|\n|\r/).map(line => line.replace(/^\s*\*/, "").trimStart()).join("\n");
    }

    return text;
}

/**
 * Gets flags that control emit behavior of a node.
 *
 * @internal
 */
export function getEmitFlags(node: Node): EmitFlags {
    const emitNode = node.emitNode;
    return emitNode && emitNode.flags || 0;
}

// Make an identifier from an external module name by extracting the string after the last "/" and replacing
// all non-alphanumeric characters with underscores

// Gets the nearest enclosing block scope container that has the provided node
// as a descendant, that is not the provided node.

// Return display name of an identifier
// Computed property names will just be emitted as "[<expr>]", where <expr> is the source
// text of the expression in the computed property.

/** @internal */
export function isComputedNonLiteralName(name: PropertyName): boolean {
    return name.kind === SyntaxKind.ComputedPropertyName && !isStringOrNumericLiteralLike(name.expression);
}

/** @internal */
function tryGetTextOfPropertyName(name: PropertyName | NoSubstitutionTemplateLiteral | JsxAttributeName): __String | undefined {
    switch (name.kind) {
        case SyntaxKind.Identifier:
        case SyntaxKind.PrivateIdentifier:
            return name.emitNode?.autoGenerate ? undefined : name.escapedText;
        case SyntaxKind.StringLiteral:
        case SyntaxKind.NumericLiteral:
        case SyntaxKind.NoSubstitutionTemplateLiteral:
            return escapeLeadingUnderscores(name.text);
        case SyntaxKind.ComputedPropertyName:
            if (isStringOrNumericLiteralLike(name.expression)) return escapeLeadingUnderscores(name.expression.text);
            return undefined;
        case SyntaxKind.JsxNamespacedName:
            return getEscapedTextOfJsxNamespacedName(name);
        default:
            return Debug.assertNever(name);
    }
}

/** @internal */
export function getTextOfPropertyName(name: PropertyName | NoSubstitutionTemplateLiteral | JsxAttributeName): __String {
    return Debug.checkDefined(tryGetTextOfPropertyName(name));
}

/** @internal */
export function createDiagnosticForNodeInSourceFile(sourceFile: SourceFile, node: Node, message: DiagnosticMessage, ...args: DiagnosticArguments): DiagnosticWithLocation {
    const span = getErrorSpanForNode(sourceFile, node);
    return createFileDiagnostic(sourceFile, span.start, span.length, message, ...args);
}

function assertDiagnosticLocation(sourceText: string, start: number, length: number) {
    Debug.assertGreaterThanOrEqual(start, 0);
    Debug.assertGreaterThanOrEqual(length, 0);
    Debug.assertLessThanOrEqual(start, sourceText.length);
    Debug.assertLessThanOrEqual(start + length, sourceText.length);
}

/** @internal */
function getSpanOfTokenAtPosition(sourceFile: SourceFile, pos: number): TextSpan {
    const scanner = createScanner(sourceFile.languageVersion, /*skipTrivia*/ true, sourceFile.languageVariant, sourceFile.text, /*onError*/ undefined, pos);
    scanner.scan();
    const start = scanner.getTokenStart();
    return createTextSpanFromBounds(start, scanner.getTokenEnd());
}

function getErrorSpanForArrowFunction(sourceFile: SourceFile, node: ArrowFunction): TextSpan {
    const pos = skipTrivia(sourceFile.text, node.pos);
    if (node.body && node.body.kind === SyntaxKind.Block) {
        const { line: startLine } = getLineAndCharacterOfPosition(sourceFile, node.body.pos);
        const { line: endLine } = getLineAndCharacterOfPosition(sourceFile, node.body.end);
        if (startLine < endLine) {
            // The arrow function spans multiple lines,
            // make the error span be the first line, inclusive.
            return createTextSpan(pos, getEndLinePosition(startLine, sourceFile) - pos + 1);
        }
    }
    return createTextSpanFromBounds(pos, node.end);
}

/** @internal */
function getErrorSpanForNode(sourceFile: SourceFile, node: Node): TextSpan {
    let errorNode: Node | undefined = node;
    switch (node.kind) {
        case SyntaxKind.SourceFile: {
            const pos = skipTrivia(sourceFile.text, 0, /*stopAfterLineBreak*/ false);
            if (pos === sourceFile.text.length) {
                // file is empty - return span for the beginning of the file
                return createTextSpan(0, 0);
            }
            return getSpanOfTokenAtPosition(sourceFile, pos);
        }
        // This list is a work in progress. Add missing node kinds to improve their error
        // spans.
        case SyntaxKind.VariableDeclaration:
        case SyntaxKind.BindingElement:
        case SyntaxKind.ClassDeclaration:
        case SyntaxKind.ClassExpression:
        case SyntaxKind.InterfaceDeclaration:
        case SyntaxKind.ModuleDeclaration:
        case SyntaxKind.EnumDeclaration:
        case SyntaxKind.EnumMember:
        case SyntaxKind.FunctionDeclaration:
        case SyntaxKind.FunctionExpression:
        case SyntaxKind.MethodDeclaration:
        case SyntaxKind.GetAccessor:
        case SyntaxKind.SetAccessor:
        case SyntaxKind.TypeAliasDeclaration:
        case SyntaxKind.PropertyDeclaration:
        case SyntaxKind.PropertySignature:
        case SyntaxKind.NamespaceImport:
            errorNode = (node as NamedDeclaration).name;
            break;
        case SyntaxKind.ArrowFunction:
            return getErrorSpanForArrowFunction(sourceFile, node as ArrowFunction);
        case SyntaxKind.CaseClause:
        case SyntaxKind.DefaultClause: {
            const start = skipTrivia(sourceFile.text, (node as CaseOrDefaultClause).pos);
            const end = (node as CaseOrDefaultClause).statements.length > 0 ? (node as CaseOrDefaultClause).statements[0].pos : (node as CaseOrDefaultClause).end;
            return createTextSpanFromBounds(start, end);
        }
        case SyntaxKind.ReturnStatement:
        case SyntaxKind.YieldExpression: {
            const pos = skipTrivia(sourceFile.text, (node as ReturnStatement | YieldExpression).pos);
            return getSpanOfTokenAtPosition(sourceFile, pos);
        }
        case SyntaxKind.SatisfiesExpression: {
            const pos = skipTrivia(sourceFile.text, (node as SatisfiesExpression).expression.end);
            return getSpanOfTokenAtPosition(sourceFile, pos);
        }
        case SyntaxKind.JSDocSatisfiesTag: {
            const pos = skipTrivia(sourceFile.text, (node as JSDocSatisfiesTag).tagName.pos);
            return getSpanOfTokenAtPosition(sourceFile, pos);
        }
    }

    if (errorNode === undefined) {
        // If we don't have a better node, then just set the error on the first token of
        // construct.
        return getSpanOfTokenAtPosition(sourceFile, node.pos);
    }

    Debug.assert(!isJSDoc(errorNode));

    const isMissing = nodeIsMissing(errorNode);
    const pos = isMissing || isJsxText(node)
        ? errorNode.pos
        : skipTrivia(sourceFile.text, errorNode.pos);

    // These asserts should all be satisfied for a properly constructed `errorNode`.
    if (isMissing) {
        Debug.assert(pos === errorNode.pos, "This failure could trigger https://github.com/Microsoft/TypeScript/issues/20809");
        Debug.assert(pos === errorNode.end, "This failure could trigger https://github.com/Microsoft/TypeScript/issues/20809");
    }
    else {
        Debug.assert(pos >= errorNode.pos, "This failure could trigger https://github.com/Microsoft/TypeScript/issues/20809");
        Debug.assert(pos <= errorNode.end, "This failure could trigger https://github.com/Microsoft/TypeScript/issues/20809");
    }

    return createTextSpanFromBounds(pos, errorNode.end);
}

/** @internal */
export function isPrologueDirective(node: Node): node is PrologueDirective {
    return node.kind === SyntaxKind.ExpressionStatement
        && (node as ExpressionStatement).expression.kind === SyntaxKind.StringLiteral;
}

/** @internal */
export function isCustomPrologue(node: Statement) {
    return !!(getEmitFlags(node) & EmitFlags.CustomPrologue);
}

/** @internal */
export function isHoistedFunction(node: Statement) {
    return isCustomPrologue(node)
        && isFunctionDeclaration(node);
}

function isHoistedVariable(node: VariableDeclaration) {
    return isIdentifier(node.name)
        && !node.initializer;
}

/** @internal */
export function isHoistedVariableStatement(node: Statement) {
    return isCustomPrologue(node)
        && isVariableStatement(node)
        && every(node.declarationList.declarations, isHoistedVariable);
}

/** @internal */
export function getJSDocCommentRanges(node: Node, text: string) {
    const commentRanges = (node.kind === SyntaxKind.Parameter ||
            node.kind === SyntaxKind.TypeParameter ||
            node.kind === SyntaxKind.FunctionExpression ||
            node.kind === SyntaxKind.ArrowFunction ||
            node.kind === SyntaxKind.ParenthesizedExpression ||
            node.kind === SyntaxKind.VariableDeclaration ||
            node.kind === SyntaxKind.ExportSpecifier) ?
        concatenate(getTrailingCommentRanges(text, node.pos), getLeadingCommentRanges(text, node.pos)) :
        getLeadingCommentRanges(text, node.pos);
    // True if the comment starts with '/**' but not if it is '/**/'
    return filter(commentRanges, comment =>
        text.charCodeAt(comment.pos + 1) === CharacterCodes.asterisk &&
        text.charCodeAt(comment.pos + 2) === CharacterCodes.asterisk &&
        text.charCodeAt(comment.pos + 3) !== CharacterCodes.slash);
}

function isPartOfTypeNode(node: Node): boolean {
    if (SyntaxKind.FirstTypeNode <= node.kind && node.kind <= SyntaxKind.LastTypeNode) {
        return true;
    }

    switch (node.kind) {
        case SyntaxKind.AnyKeyword:
        case SyntaxKind.UnknownKeyword:
        case SyntaxKind.NumberKeyword:
        case SyntaxKind.BigIntKeyword:
        case SyntaxKind.StringKeyword:
        case SyntaxKind.BooleanKeyword:
        case SyntaxKind.SymbolKeyword:
        case SyntaxKind.ObjectKeyword:
        case SyntaxKind.UndefinedKeyword:
        case SyntaxKind.NullKeyword:
        case SyntaxKind.NeverKeyword:
            return true;
        case SyntaxKind.VoidKeyword:
            return node.parent.kind !== SyntaxKind.VoidExpression;
        case SyntaxKind.ExpressionWithTypeArguments:
            return isPartOfTypeExpressionWithTypeArguments(node);
        case SyntaxKind.TypeParameter:
            return node.parent.kind === SyntaxKind.MappedType || node.parent.kind === SyntaxKind.InferType;

        // Identifiers and qualified names may be type nodes, depending on their context. Climb
        // above them to find the lowest container
        case SyntaxKind.Identifier:
            // If the identifier is the RHS of a qualified name, then it's a type iff its parent is.
            if (node.parent.kind === SyntaxKind.QualifiedName && (node.parent as QualifiedName).right === node) {
                node = node.parent;
            }
            else if (node.parent.kind === SyntaxKind.PropertyAccessExpression && (node.parent as PropertyAccessExpression).name === node) {
                node = node.parent;
            }
            // At this point, node is either a qualified name or an identifier
            Debug.assert(node.kind === SyntaxKind.Identifier || node.kind === SyntaxKind.QualifiedName || node.kind === SyntaxKind.PropertyAccessExpression, "'node' was expected to be a qualified name, identifier or property access in 'isPartOfTypeNode'.");
            // falls through

        case SyntaxKind.QualifiedName:
        case SyntaxKind.PropertyAccessExpression:
        case SyntaxKind.ThisKeyword: {
            const { parent } = node;
            if (parent.kind === SyntaxKind.TypeQuery) {
                return false;
            }
            if (parent.kind === SyntaxKind.ImportType) {
                return !(parent as ImportTypeNode).isTypeOf;
            }
            // Do not recursively call isPartOfTypeNode on the parent. In the example:
            //
            //     let a: A.B.C;
            //
            // Calling isPartOfTypeNode would consider the qualified name A.B a type node.
            // Only C and A.B.C are type nodes.
            if (SyntaxKind.FirstTypeNode <= parent.kind && parent.kind <= SyntaxKind.LastTypeNode) {
                return true;
            }
            switch (parent.kind) {
                case SyntaxKind.ExpressionWithTypeArguments:
                    return isPartOfTypeExpressionWithTypeArguments(parent);
                case SyntaxKind.TypeParameter:
                    return node === (parent as TypeParameterDeclaration).constraint;
                case SyntaxKind.JSDocTemplateTag:
                    return node === (parent as JSDocTemplateTag).constraint;
                case SyntaxKind.PropertyDeclaration:
                case SyntaxKind.PropertySignature:
                case SyntaxKind.Parameter:
                case SyntaxKind.VariableDeclaration:
                    return node === (parent as HasType).type;
                case SyntaxKind.FunctionDeclaration:
                case SyntaxKind.FunctionExpression:
                case SyntaxKind.ArrowFunction:
                case SyntaxKind.Constructor:
                case SyntaxKind.MethodDeclaration:
                case SyntaxKind.MethodSignature:
                case SyntaxKind.GetAccessor:
                case SyntaxKind.SetAccessor:
                    return node === (parent as FunctionLikeDeclaration).type;
                case SyntaxKind.CallSignature:
                case SyntaxKind.ConstructSignature:
                case SyntaxKind.IndexSignature:
                    return node === (parent as SignatureDeclaration).type;
                case SyntaxKind.TypeAssertionExpression:
                    return node === (parent as TypeAssertion).type;
                case SyntaxKind.CallExpression:
                case SyntaxKind.NewExpression:
                case SyntaxKind.TaggedTemplateExpression:
                    return contains((parent as CallExpression | TaggedTemplateExpression).typeArguments, node);
            }
        }
    }

    return false;
}

function isPartOfTypeExpressionWithTypeArguments(node: Node) {
    return isJSDocImplementsTag(node.parent)
        || isJSDocAugmentsTag(node.parent)
        || isHeritageClause(node.parent) && !isExpressionWithTypeArgumentsInClassExtendsClause(node);
}

// Warning: This has the same semantics as the forEach family of functions,
//          in that traversal terminates in the event that 'visitor' supplies a truthy value.

/** @internal */
function isVariableLike(node: Node): node is VariableLikeDeclaration {
    if (node) {
        switch (node.kind) {
            case SyntaxKind.BindingElement:
            case SyntaxKind.EnumMember:
            case SyntaxKind.Parameter:
            case SyntaxKind.PropertyAssignment:
            case SyntaxKind.PropertyDeclaration:
            case SyntaxKind.PropertySignature:
            case SyntaxKind.ShorthandPropertyAssignment:
            case SyntaxKind.VariableDeclaration:
                return true;
        }
    }
    return false;
}

/** @internal */
export function isFunctionBlock(node: Node): boolean {
    return node && node.kind === SyntaxKind.Block && isFunctionLike(node.parent);
}

/**
 * Determines whether a node is a property or element access expression for `super`.
 *
 * @internal
 */
export function isSuperProperty(node: Node): node is SuperProperty {
    const kind = node.kind;
    return (kind === SyntaxKind.PropertyAccessExpression || kind === SyntaxKind.ElementAccessExpression)
        && (node as PropertyAccessExpression | ElementAccessExpression).expression.kind === SyntaxKind.SuperKeyword;
}

/** @internal */
function nodeCanBeDecorated(useLegacyDecorators: boolean, node: ClassDeclaration): true;
/** @internal */
function nodeCanBeDecorated(useLegacyDecorators: boolean, node: ClassExpression): boolean;
/** @internal */
function nodeCanBeDecorated(useLegacyDecorators: boolean, node: ClassElement, parent: Node): boolean;
/** @internal */
function nodeCanBeDecorated(useLegacyDecorators: boolean, node: Node, parent: Node, grandparent: Node): boolean;
/** @internal */
function nodeCanBeDecorated(useLegacyDecorators: boolean, node: Node, parent?: Node, grandparent?: Node): boolean {
    // private names cannot be used with decorators yet
    if (useLegacyDecorators && isNamedDeclaration(node) && isPrivateIdentifier(node.name)) {
        return false;
    }

    switch (node.kind) {
        case SyntaxKind.ClassDeclaration:
            // class declarations are valid targets
            return true;

        case SyntaxKind.ClassExpression:
            // class expressions are valid targets for native decorators
            return !useLegacyDecorators;

        case SyntaxKind.PropertyDeclaration:
            // property declarations are valid if their parent is a class declaration.
            return parent !== undefined
                && (useLegacyDecorators ? isClassDeclaration(parent) : isClassLike(parent) && !hasAbstractModifier(node) && !hasAmbientModifier(node));

        case SyntaxKind.GetAccessor:
        case SyntaxKind.SetAccessor:
        case SyntaxKind.MethodDeclaration:
            // if this method has a body and its parent is a class declaration, this is a valid target.
            return (node as FunctionLikeDeclaration).body !== undefined
                && parent !== undefined
                && (useLegacyDecorators ? isClassDeclaration(parent) : isClassLike(parent));

        case SyntaxKind.Parameter:
            // TODO(rbuckton): Parameter decorator support for ES decorators must wait until it is standardized
            if (!useLegacyDecorators) return false;
            // if the parameter's parent has a body and its grandparent is a class declaration, this is a valid target.
            return parent !== undefined
                && (parent as FunctionLikeDeclaration).body !== undefined
                && (parent.kind === SyntaxKind.Constructor
                    || parent.kind === SyntaxKind.MethodDeclaration
                    || parent.kind === SyntaxKind.SetAccessor)
                && getThisParameter(parent as FunctionLikeDeclaration) !== node
                && grandparent !== undefined
                && grandparent.kind === SyntaxKind.ClassDeclaration;
    }

    return false;
}

/** @internal */
function nodeIsDecorated(useLegacyDecorators: boolean, node: ClassDeclaration | ClassExpression): boolean;
/** @internal */
function nodeIsDecorated(useLegacyDecorators: boolean, node: ClassElement, parent: Node): boolean;
/** @internal */
function nodeIsDecorated(useLegacyDecorators: boolean, node: Node, parent: Node, grandparent: Node): boolean;
/** @internal */
function nodeIsDecorated(useLegacyDecorators: boolean, node: Node, parent?: Node, grandparent?: Node): boolean {
    return hasDecorators(node)
        && nodeCanBeDecorated(useLegacyDecorators, node, parent!, grandparent!);
}

/** @internal */
function nodeOrChildIsDecorated(useLegacyDecorators: boolean, node: ClassDeclaration | ClassExpression): boolean;
/** @internal */
function nodeOrChildIsDecorated(useLegacyDecorators: boolean, node: ClassElement, parent: Node): boolean;
/** @internal */
function nodeOrChildIsDecorated(useLegacyDecorators: boolean, node: Node, parent: Node, grandparent: Node): boolean;
/** @internal */
function nodeOrChildIsDecorated(useLegacyDecorators: boolean, node: Node, parent?: Node, grandparent?: Node): boolean {
    return nodeIsDecorated(useLegacyDecorators, node, parent!, grandparent!)
        || childIsDecorated(useLegacyDecorators, node, parent!);
}

/** @internal */
function childIsDecorated(useLegacyDecorators: boolean, node: ClassDeclaration | ClassExpression): boolean;
/** @internal */
function childIsDecorated(useLegacyDecorators: boolean, node: Node, parent: Node): boolean;
/** @internal */
function childIsDecorated(useLegacyDecorators: boolean, node: Node, parent?: Node): boolean {
    switch (node.kind) {
        case SyntaxKind.ClassDeclaration:
            return some((node as ClassDeclaration).members, m => nodeOrChildIsDecorated(useLegacyDecorators, m, node, parent!));
        case SyntaxKind.ClassExpression:
            return !useLegacyDecorators && some((node as ClassExpression).members, m => nodeOrChildIsDecorated(useLegacyDecorators, m, node, parent!));
        case SyntaxKind.MethodDeclaration:
        case SyntaxKind.SetAccessor:
        case SyntaxKind.Constructor:
            return some((node as FunctionLikeDeclaration).parameters, p => nodeIsDecorated(useLegacyDecorators, p, node, parent!));
        default:
            return false;
    }
}

/** @internal */
function isJSXTagName(node: Node) {
    const { parent } = node;
    if (
        parent.kind === SyntaxKind.JsxOpeningElement ||
        parent.kind === SyntaxKind.JsxSelfClosingElement ||
        parent.kind === SyntaxKind.JsxClosingElement
    ) {
        return (parent as JsxOpeningLikeElement).tagName === node;
    }
    return false;
}

/** @internal */
function isExpressionNode(node: Node): boolean {
    switch (node.kind) {
        case SyntaxKind.SuperKeyword:
        case SyntaxKind.NullKeyword:
        case SyntaxKind.TrueKeyword:
        case SyntaxKind.FalseKeyword:
        case SyntaxKind.RegularExpressionLiteral:
        case SyntaxKind.ArrayLiteralExpression:
        case SyntaxKind.ObjectLiteralExpression:
        case SyntaxKind.PropertyAccessExpression:
        case SyntaxKind.ElementAccessExpression:
        case SyntaxKind.CallExpression:
        case SyntaxKind.NewExpression:
        case SyntaxKind.TaggedTemplateExpression:
        case SyntaxKind.AsExpression:
        case SyntaxKind.TypeAssertionExpression:
        case SyntaxKind.SatisfiesExpression:
        case SyntaxKind.NonNullExpression:
        case SyntaxKind.ParenthesizedExpression:
        case SyntaxKind.FunctionExpression:
        case SyntaxKind.ClassExpression:
        case SyntaxKind.ArrowFunction:
        case SyntaxKind.VoidExpression:
        case SyntaxKind.DeleteExpression:
        case SyntaxKind.TypeOfExpression:
        case SyntaxKind.PrefixUnaryExpression:
        case SyntaxKind.PostfixUnaryExpression:
        case SyntaxKind.BinaryExpression:
        case SyntaxKind.ConditionalExpression:
        case SyntaxKind.SpreadElement:
        case SyntaxKind.TemplateExpression:
        case SyntaxKind.OmittedExpression:
        case SyntaxKind.JsxElement:
        case SyntaxKind.JsxSelfClosingElement:
        case SyntaxKind.JsxFragment:
        case SyntaxKind.YieldExpression:
        case SyntaxKind.AwaitExpression:
        case SyntaxKind.MetaProperty:
            return true;
        case SyntaxKind.ExpressionWithTypeArguments:
            return !isHeritageClause(node.parent) && !isJSDocAugmentsTag(node.parent);
        case SyntaxKind.QualifiedName:
            while (node.parent.kind === SyntaxKind.QualifiedName) {
                node = node.parent;
            }
            return node.parent.kind === SyntaxKind.TypeQuery || isJSDocLinkLike(node.parent) || isJSDocNameReference(node.parent) || isJSDocMemberName(node.parent) || isJSXTagName(node);
        case SyntaxKind.JSDocMemberName:
            while (isJSDocMemberName(node.parent)) {
                node = node.parent;
            }
            return node.parent.kind === SyntaxKind.TypeQuery || isJSDocLinkLike(node.parent) || isJSDocNameReference(node.parent) || isJSDocMemberName(node.parent) || isJSXTagName(node);
        case SyntaxKind.PrivateIdentifier:
            return isBinaryExpression(node.parent) && node.parent.left === node && node.parent.operatorToken.kind === SyntaxKind.InKeyword;
        case SyntaxKind.Identifier:
            if (node.parent.kind === SyntaxKind.TypeQuery || isJSDocLinkLike(node.parent) || isJSDocNameReference(node.parent) || isJSDocMemberName(node.parent) || isJSXTagName(node)) {
                return true;
            }
            // falls through

        case SyntaxKind.NumericLiteral:
        case SyntaxKind.BigIntLiteral:
        case SyntaxKind.StringLiteral:
        case SyntaxKind.NoSubstitutionTemplateLiteral:
        case SyntaxKind.ThisKeyword:
            return isInExpressionContext(node);
        default:
            return false;
    }
}

/** @internal */
function isInExpressionContext(node: Node): boolean {
    const { parent } = node;
    switch (parent.kind) {
        case SyntaxKind.VariableDeclaration:
        case SyntaxKind.Parameter:
        case SyntaxKind.PropertyDeclaration:
        case SyntaxKind.PropertySignature:
        case SyntaxKind.EnumMember:
        case SyntaxKind.PropertyAssignment:
        case SyntaxKind.BindingElement:
            return (parent as HasInitializer).initializer === node;
        case SyntaxKind.ExpressionStatement:
        case SyntaxKind.IfStatement:
        case SyntaxKind.DoStatement:
        case SyntaxKind.WhileStatement:
        case SyntaxKind.ReturnStatement:
        case SyntaxKind.WithStatement:
        case SyntaxKind.SwitchStatement:
        case SyntaxKind.CaseClause:
        case SyntaxKind.ThrowStatement:
            return (parent as ExpressionStatement).expression === node;
        case SyntaxKind.ForStatement:
            const forStatement = parent as ForStatement;
            return (forStatement.initializer === node && forStatement.initializer.kind !== SyntaxKind.VariableDeclarationList) ||
                forStatement.condition === node ||
                forStatement.incrementor === node;
        case SyntaxKind.ForInStatement:
        case SyntaxKind.ForOfStatement:
            const forInOrOfStatement = parent as ForInOrOfStatement;
            return (forInOrOfStatement.initializer === node && forInOrOfStatement.initializer.kind !== SyntaxKind.VariableDeclarationList) ||
                forInOrOfStatement.expression === node;
        case SyntaxKind.TypeAssertionExpression:
        case SyntaxKind.AsExpression:
            return node === (parent as AssertionExpression).expression;
        case SyntaxKind.TemplateSpan:
            return node === (parent as TemplateSpan).expression;
        case SyntaxKind.ComputedPropertyName:
            return node === (parent as ComputedPropertyName).expression;
        case SyntaxKind.Decorator:
        case SyntaxKind.JsxExpression:
        case SyntaxKind.JsxSpreadAttribute:
        case SyntaxKind.SpreadAssignment:
            return true;
        case SyntaxKind.ExpressionWithTypeArguments:
            return (parent as ExpressionWithTypeArguments).expression === node && !isPartOfTypeNode(parent);
        case SyntaxKind.ShorthandPropertyAssignment:
            return (parent as ShorthandPropertyAssignment).objectAssignmentInitializer === node;
        case SyntaxKind.SatisfiesExpression:
            return node === (parent as SatisfiesExpression).expression;
        default:
            return isExpressionNode(parent);
    }
}

/** @internal */
export function isInJSFile(node: Node | undefined): boolean {
    return !!node && !!(node.flags & NodeFlags.JavaScriptFile);
}

/** @internal */
export function isStringDoubleQuoted(str: StringLiteralLike, sourceFile: SourceFile): boolean {
    return getSourceTextOfNodeFromSourceFile(sourceFile, str).charCodeAt(0) === CharacterCodes.doubleQuote;
}

/** @internal */
function getRightMostAssignedExpression(node: Expression): Expression {
    while (isAssignmentExpression(node, /*excludeCompoundAssignment*/ true)) {
        node = node.right;
    }
    return node;
}

/** @internal */
function isExportsIdentifier(node: Node) {
    return isIdentifier(node) && node.escapedText === "exports";
}

/** @internal */
function isModuleIdentifier(node: Node) {
    return isIdentifier(node) && node.escapedText === "module";
}

/** @internal */
function isModuleExportsAccessExpression(node: Node): node is LiteralLikeElementAccessExpression & { expression: Identifier; } {
    return (isPropertyAccessExpression(node) || isLiteralLikeElementAccess(node))
        && isModuleIdentifier(node.expression)
        && getElementOrPropertyAccessName(node) === "exports";
}

/// Given a BinaryExpression, returns SpecialPropertyAssignmentKind for the various kinds of property
/// assignments we treat as special in the binder
/** @internal */
export function getAssignmentDeclarationKind(expr: BinaryExpression | CallExpression): AssignmentDeclarationKind {
    const special = getAssignmentDeclarationKindWorker(expr);
    return special === AssignmentDeclarationKind.Property || isInJSFile(expr) ? special : AssignmentDeclarationKind.None;
}

/** @internal */
function isBindableObjectDefinePropertyCall(expr: CallExpression): expr is BindableObjectDefinePropertyCall {
    return length(expr.arguments) === 3 &&
        isPropertyAccessExpression(expr.expression) &&
        isIdentifier(expr.expression.expression) &&
        idText(expr.expression.expression) === "Object" &&
        idText(expr.expression.name) === "defineProperty" &&
        isStringOrNumericLiteralLike(expr.arguments[1]) &&
        isBindableStaticNameExpression(expr.arguments[0], /*excludeThisKeyword*/ true);
}

/**
 * x[0] OR x['a'] OR x[Symbol.y]
 *
 * @internal
 */
function isLiteralLikeElementAccess(node: Node): node is LiteralLikeElementAccessExpression {
    return isElementAccessExpression(node) && isStringOrNumericLiteralLike(node.argumentExpression);
}

/**
 * Any series of property and element accesses.
 *
 * @internal
 */
function isBindableStaticAccessExpression(node: Node, excludeThisKeyword?: boolean): node is BindableStaticAccessExpression {
    return isPropertyAccessExpression(node) && (!excludeThisKeyword && node.expression.kind === SyntaxKind.ThisKeyword || isIdentifier(node.name) && isBindableStaticNameExpression(node.expression, /*excludeThisKeyword*/ true))
        || isBindableStaticElementAccessExpression(node, excludeThisKeyword);
}

/**
 * Any series of property and element accesses, ending in a literal element access
 *
 * @internal
 */
export function isBindableStaticElementAccessExpression(node: Node, excludeThisKeyword?: boolean): node is BindableStaticElementAccessExpression {
    return isLiteralLikeElementAccess(node)
        && ((!excludeThisKeyword && node.expression.kind === SyntaxKind.ThisKeyword) ||
            isEntityNameExpression(node.expression) ||
            isBindableStaticAccessExpression(node.expression, /*excludeThisKeyword*/ true));
}

/** @internal */
function isBindableStaticNameExpression(node: Node, excludeThisKeyword?: boolean): node is BindableStaticNameExpression {
    return isEntityNameExpression(node) || isBindableStaticAccessExpression(node, excludeThisKeyword);
}

function getAssignmentDeclarationKindWorker(expr: BinaryExpression | CallExpression): AssignmentDeclarationKind {
    if (isCallExpression(expr)) {
        if (!isBindableObjectDefinePropertyCall(expr)) {
            return AssignmentDeclarationKind.None;
        }
        const entityName = expr.arguments[0];
        if (isExportsIdentifier(entityName) || isModuleExportsAccessExpression(entityName)) {
            return AssignmentDeclarationKind.ObjectDefinePropertyExports;
        }
        if (isBindableStaticAccessExpression(entityName) && getElementOrPropertyAccessName(entityName) === "prototype") {
            return AssignmentDeclarationKind.ObjectDefinePrototypeProperty;
        }
        return AssignmentDeclarationKind.ObjectDefinePropertyValue;
    }
    if (expr.operatorToken.kind !== SyntaxKind.EqualsToken || !isAccessExpression(expr.left) || isVoidZero(getRightMostAssignedExpression(expr))) {
        return AssignmentDeclarationKind.None;
    }
    if (isBindableStaticNameExpression(expr.left.expression, /*excludeThisKeyword*/ true) && getElementOrPropertyAccessName(expr.left) === "prototype" && isObjectLiteralExpression(getInitializerOfBinaryExpression(expr))) {
        // F.prototype = { ... }
        return AssignmentDeclarationKind.Prototype;
    }
    return getAssignmentDeclarationPropertyAccessKind(expr.left);
}

function isVoidZero(node: Node) {
    return isVoidExpression(node) && isNumericLiteral(node.expression) && node.expression.text === "0";
}

/**
 * Does not handle signed numeric names like `a[+0]` - handling those would require handling prefix unary expressions
 * throughout late binding handling as well, which is awkward (but ultimately probably doable if there is demand)
 *
 * @internal
 */
export function getElementOrPropertyAccessArgumentExpressionOrName(node: AccessExpression): Identifier | PrivateIdentifier | StringLiteralLike | NumericLiteral | ElementAccessExpression | undefined {
    if (isPropertyAccessExpression(node)) {
        return node.name;
    }
    const arg = skipParentheses(node.argumentExpression);
    if (isNumericLiteral(arg) || isStringLiteralLike(arg)) {
        return arg;
    }
    return node;
}

/** @internal */
function getElementOrPropertyAccessName(node: LiteralLikeElementAccessExpression | PropertyAccessExpression): __String;
/** @internal */
function getElementOrPropertyAccessName(node: AccessExpression): __String | undefined;
/** @internal */
function getElementOrPropertyAccessName(node: AccessExpression): __String | undefined {
    const name = getElementOrPropertyAccessArgumentExpressionOrName(node);
    if (name) {
        if (isIdentifier(name)) {
            return name.escapedText;
        }
        if (isStringLiteralLike(name) || isNumericLiteral(name)) {
            return escapeLeadingUnderscores(name.text);
        }
    }
    return undefined;
}

/** @internal */
function getAssignmentDeclarationPropertyAccessKind(lhs: AccessExpression): AssignmentDeclarationKind {
    if (lhs.expression.kind === SyntaxKind.ThisKeyword) {
        return AssignmentDeclarationKind.ThisProperty;
    }
    else if (isModuleExportsAccessExpression(lhs)) {
        // module.exports = expr
        return AssignmentDeclarationKind.ModuleExports;
    }
    else if (isBindableStaticNameExpression(lhs.expression, /*excludeThisKeyword*/ true)) {
        if (isPrototypeAccess(lhs.expression)) {
            // F.G....prototype.x = expr
            return AssignmentDeclarationKind.PrototypeProperty;
        }

        let nextToLast = lhs;
        while (!isIdentifier(nextToLast.expression)) {
            nextToLast = nextToLast.expression as Exclude<BindableStaticNameExpression, Identifier>;
        }
        const id = nextToLast.expression;
        if (
            (id.escapedText === "exports" ||
                id.escapedText === "module" && getElementOrPropertyAccessName(nextToLast) === "exports") &&
            // ExportsProperty does not support binding with computed names
            isBindableStaticAccessExpression(lhs)
        ) {
            // exports.name = expr OR module.exports.name = expr OR exports["name"] = expr ...
            return AssignmentDeclarationKind.ExportsProperty;
        }
        if (isBindableStaticNameExpression(lhs, /*excludeThisKeyword*/ true) || (isElementAccessExpression(lhs) && isDynamicName(lhs))) {
            // F.G...x = expr
            return AssignmentDeclarationKind.Property;
        }
    }

    return AssignmentDeclarationKind.None;
}

/** @internal */
function getInitializerOfBinaryExpression(expr: BinaryExpression) {
    while (isBinaryExpression(expr.right)) {
        expr = expr.right;
    }
    return expr.right;
}

function getSourceOfAssignment(node: Node): Node | undefined {
    return isExpressionStatement(node) &&
            isBinaryExpression(node.expression) &&
            node.expression.operatorToken.kind === SyntaxKind.EqualsToken
        ? getRightMostAssignedExpression(node.expression)
        : undefined;
}

function getSourceOfDefaultedAssignment(node: Node): Node | undefined {
    return isExpressionStatement(node) &&
            isBinaryExpression(node.expression) &&
            getAssignmentDeclarationKind(node.expression) !== AssignmentDeclarationKind.None &&
            isBinaryExpression(node.expression.right) &&
            (node.expression.right.operatorToken.kind === SyntaxKind.BarBarToken || node.expression.right.operatorToken.kind === SyntaxKind.QuestionQuestionToken)
        ? node.expression.right.right
        : undefined;
}

/** @internal */
function getSingleInitializerOfVariableStatementOrPropertyDeclaration(node: Node): Expression | undefined {
    switch (node.kind) {
        case SyntaxKind.VariableStatement:
            const v = getSingleVariableOfVariableStatement(node);
            return v && v.initializer;
        case SyntaxKind.PropertyDeclaration:
            return (node as PropertyDeclaration).initializer;
        case SyntaxKind.PropertyAssignment:
            return (node as PropertyAssignment).initializer;
    }
}

/** @internal */
function getSingleVariableOfVariableStatement(node: Node): VariableDeclaration | undefined {
    return isVariableStatement(node) ? firstOrUndefined(node.declarationList.declarations) : undefined;
}

function getNestedModuleDeclaration(node: Node): Node | undefined {
    return isModuleDeclaration(node) &&
            node.body &&
            node.body.kind === SyntaxKind.ModuleDeclaration
        ? node.body
        : undefined;
}

/** @internal */
export function canHaveJSDoc(node: Node): node is HasJSDoc {
    switch (node.kind) {
        case SyntaxKind.ArrowFunction:
        case SyntaxKind.BinaryExpression:
        case SyntaxKind.Block:
        case SyntaxKind.BreakStatement:
        case SyntaxKind.CallSignature:
        case SyntaxKind.CaseClause:
        case SyntaxKind.ClassDeclaration:
        case SyntaxKind.ClassExpression:
        case SyntaxKind.ClassStaticBlockDeclaration:
        case SyntaxKind.Constructor:
        case SyntaxKind.ConstructorType:
        case SyntaxKind.ConstructSignature:
        case SyntaxKind.ContinueStatement:
        case SyntaxKind.DebuggerStatement:
        case SyntaxKind.DoStatement:
        case SyntaxKind.ElementAccessExpression:
        case SyntaxKind.EmptyStatement:
        case SyntaxKind.EndOfFileToken:
        case SyntaxKind.EnumDeclaration:
        case SyntaxKind.EnumMember:
        case SyntaxKind.ExportAssignment:
        case SyntaxKind.ExportDeclaration:
        case SyntaxKind.ExportSpecifier:
        case SyntaxKind.ExpressionStatement:
        case SyntaxKind.ForInStatement:
        case SyntaxKind.ForOfStatement:
        case SyntaxKind.ForStatement:
        case SyntaxKind.FunctionDeclaration:
        case SyntaxKind.FunctionExpression:
        case SyntaxKind.FunctionType:
        case SyntaxKind.GetAccessor:
        case SyntaxKind.Identifier:
        case SyntaxKind.IfStatement:
        case SyntaxKind.ImportDeclaration:
        case SyntaxKind.ImportEqualsDeclaration:
        case SyntaxKind.IndexSignature:
        case SyntaxKind.InterfaceDeclaration:
        case SyntaxKind.JSDocFunctionType:
        case SyntaxKind.JSDocSignature:
        case SyntaxKind.LabeledStatement:
        case SyntaxKind.MethodDeclaration:
        case SyntaxKind.MethodSignature:
        case SyntaxKind.ModuleDeclaration:
        case SyntaxKind.NamedTupleMember:
        case SyntaxKind.NamespaceExportDeclaration:
        case SyntaxKind.ObjectLiteralExpression:
        case SyntaxKind.Parameter:
        case SyntaxKind.ParenthesizedExpression:
        case SyntaxKind.PropertyAccessExpression:
        case SyntaxKind.PropertyAssignment:
        case SyntaxKind.PropertyDeclaration:
        case SyntaxKind.PropertySignature:
        case SyntaxKind.ReturnStatement:
        case SyntaxKind.SemicolonClassElement:
        case SyntaxKind.SetAccessor:
        case SyntaxKind.ShorthandPropertyAssignment:
        case SyntaxKind.SpreadAssignment:
        case SyntaxKind.SwitchStatement:
        case SyntaxKind.ThrowStatement:
        case SyntaxKind.TryStatement:
        case SyntaxKind.TypeAliasDeclaration:
        case SyntaxKind.TypeParameter:
        case SyntaxKind.VariableDeclaration:
        case SyntaxKind.VariableStatement:
        case SyntaxKind.WhileStatement:
        case SyntaxKind.WithStatement:
            return true;
        default:
            return false;
    }
}

/**
 * This function checks multiple locations for JSDoc comments that apply to a host node.
 * At each location, the whole comment may apply to the node, or only a specific tag in
 * the comment. In the first case, location adds the entire {@link JSDoc} object. In the
 * second case, it adds the applicable {@link JSDocTag}.
 *
 * For example, a JSDoc comment before a parameter adds the entire {@link JSDoc}. But a
 * `@param` tag on the parent function only adds the {@link JSDocTag} for the `@param`.
 *
 * ```ts
 * /** JSDoc will be returned for `a` *\/
 * const a = 0
 * /**
 *  * Entire JSDoc will be returned for `b`
 *  * @param c JSDocTag will be returned for `c`
 *  *\/
 * function b(/** JSDoc will be returned for `c` *\/ c) {}
 * ```
 */
export function getJSDocCommentsAndTags(hostNode: Node): readonly (JSDoc | JSDocTag)[];
/** @internal separate signature so that stripInternal can remove noCache from the public API */
// eslint-disable-next-line @typescript-eslint/unified-signatures
export function getJSDocCommentsAndTags(hostNode: Node, noCache?: boolean): readonly (JSDoc | JSDocTag)[];
export function getJSDocCommentsAndTags(hostNode: Node, noCache?: boolean): readonly (JSDoc | JSDocTag)[] {
    let result: (JSDoc | JSDocTag)[] | undefined;
    // Pull parameter comments from declaring function as well
    if (isVariableLike(hostNode) && hasInitializer(hostNode) && hasJSDocNodes(hostNode.initializer!)) {
        result = addRange(result, filterOwnedJSDocTags(hostNode, hostNode.initializer.jsDoc!));
    }

    let node: Node | undefined = hostNode;
    while (node && node.parent) {
        if (hasJSDocNodes(node)) {
            result = addRange(result, filterOwnedJSDocTags(hostNode, node.jsDoc!));
        }

        if (node.kind === SyntaxKind.Parameter) {
            result = addRange(result, (noCache ? getJSDocParameterTagsNoCache : getJSDocParameterTags)(node as ParameterDeclaration));
            break;
        }
        if (node.kind === SyntaxKind.TypeParameter) {
            result = addRange(result, (noCache ? getJSDocTypeParameterTagsNoCache : getJSDocTypeParameterTags)(node as TypeParameterDeclaration));
            break;
        }
        node = getNextJSDocCommentLocation(node);
    }
    return result || emptyArray;
}

function filterOwnedJSDocTags(hostNode: Node, comments: JSDocArray) {
    const lastJsDoc = last(comments);
    return flatMap<JSDoc, JSDoc | JSDocTag>(comments, jsDoc => {
        if (jsDoc === lastJsDoc) {
            const ownedTags = filter(jsDoc.tags, tag => ownsJSDocTag(hostNode, tag));
            return jsDoc.tags === ownedTags ? [jsDoc] : ownedTags;
        }
        else {
            return filter(jsDoc.tags, isJSDocOverloadTag);
        }
    });
}

/**
 * Determines whether a host node owns a jsDoc tag. A `@type`/`@satisfies` tag attached to a
 * a ParenthesizedExpression belongs only to the ParenthesizedExpression.
 */
function ownsJSDocTag(hostNode: Node, tag: JSDocTag) {
    return !(isJSDocTypeTag(tag) || isJSDocSatisfiesTag(tag))
        || !tag.parent
        || !isJSDoc(tag.parent)
        || !isParenthesizedExpression(tag.parent.parent)
        || tag.parent.parent === hostNode;
}

/** @internal */
function getNextJSDocCommentLocation(node: Node) {
    const parent = node.parent;
    if (
        parent.kind === SyntaxKind.PropertyAssignment ||
        parent.kind === SyntaxKind.ExportAssignment ||
        parent.kind === SyntaxKind.PropertyDeclaration ||
        parent.kind === SyntaxKind.ExpressionStatement && node.kind === SyntaxKind.PropertyAccessExpression ||
        parent.kind === SyntaxKind.ReturnStatement ||
        getNestedModuleDeclaration(parent) ||
        isAssignmentExpression(node)
    ) {
        return parent;
    }
    // Try to recognize this pattern when node is initializer of variable declaration and JSDoc comments are on containing variable statement.
    // /**
    //   * @param {number} name
    //   * @returns {number}
    //   */
    // var x = function(name) { return name.length; }
    else if (
        parent.parent &&
        (getSingleVariableOfVariableStatement(parent.parent) === node || isAssignmentExpression(parent))
    ) {
        return parent.parent;
    }
    else if (
        parent.parent && parent.parent.parent &&
        (getSingleVariableOfVariableStatement(parent.parent.parent) ||
            getSingleInitializerOfVariableStatementOrPropertyDeclaration(parent.parent.parent) === node ||
            getSourceOfDefaultedAssignment(parent.parent.parent))
    ) {
        return parent.parent.parent;
    }
}

/** @internal */
function getEffectiveJSDocHost(node: Node): Node | undefined {
    const host = getJSDocHost(node);
    if (host) {
        return getSourceOfDefaultedAssignment(host)
            || getSourceOfAssignment(host)
            || getSingleInitializerOfVariableStatementOrPropertyDeclaration(host)
            || getSingleVariableOfVariableStatement(host)
            || getNestedModuleDeclaration(host)
            || host;
    }
}

/**
 * Use getEffectiveJSDocHost if you additionally need to look for jsdoc on parent nodes, like assignments.
 *
 * @internal
 */
function getJSDocHost(node: Node): HasJSDoc | undefined {
    const jsDoc = getJSDocRoot(node);
    if (!jsDoc) {
        return undefined;
    }

    const host = jsDoc.parent;
    if (host && host.jsDoc && jsDoc === lastOrUndefined(host.jsDoc)) {
        return host;
    }
}

/** @internal */
function getJSDocRoot(node: Node): JSDoc | undefined {
    return findAncestor(node.parent, isJSDoc);
}

/** @internal */
export function skipParentheses(node: Expression, excludeJSDocTypeAssertions?: boolean): Expression;
/** @internal */
export function skipParentheses(node: Node, excludeJSDocTypeAssertions?: boolean): Node;
/** @internal */
export function skipParentheses(node: Node, excludeJSDocTypeAssertions?: boolean): Node {
    const flags = excludeJSDocTypeAssertions ?
        OuterExpressionKinds.Parentheses | OuterExpressionKinds.ExcludeJSDocTypeAssertion :
        OuterExpressionKinds.Parentheses;
    return skipOuterExpressions(node, flags);
}

// a node is delete target iff. it is PropertyAccessExpression/ElementAccessExpression with parentheses skipped

// True if `name` is the name of a declaration node

// See GH#16030

// Return true if the given identifier is classified as an IdentifierName

// An alias symbol is created by one of the following declarations:
// import <symbol> = ...
// import <symbol> from ...
// import * as <symbol> from ...
// import { x as <symbol> } from ...
// export { x as <symbol> } from ...
// export * as ns <symbol> from ...
// export = <EntityNameExpression>
// export default <EntityNameExpression>
// module.exports = <EntityNameExpression>
// module.exports.x = <EntityNameExpression>
// const x = require("...")
// const { x } = require("...")
// const x = require("...").y
// const { x } = require("...").y

/** @internal */
export function isKeyword(token: SyntaxKind): token is KeywordSyntaxKind {
    return SyntaxKind.FirstKeyword <= token && token <= SyntaxKind.LastKeyword;
}

/** @internal */
function isPunctuation(token: SyntaxKind): token is PunctuationSyntaxKind {
    return SyntaxKind.FirstPunctuation <= token && token <= SyntaxKind.LastPunctuation;
}

/** @internal */
export function isKeywordOrPunctuation(token: SyntaxKind): token is PunctuationOrKeywordSyntaxKind {
    return isKeyword(token) || isPunctuation(token);
}

/** @internal */
export function isStringOrNumericLiteralLike(node: Node): node is StringLiteralLike | NumericLiteral {
    return isStringLiteralLike(node) || isNumericLiteral(node);
}

/** @internal */
function isSignedNumericLiteral(node: Node): node is PrefixUnaryExpression & { operand: NumericLiteral; } {
    return isPrefixUnaryExpression(node) && (node.operator === SyntaxKind.PlusToken || node.operator === SyntaxKind.MinusToken) && isNumericLiteral(node.operand);
}

/** @internal */
function isDynamicName(name: DeclarationName): boolean {
    if (!(name.kind === SyntaxKind.ComputedPropertyName || name.kind === SyntaxKind.ElementAccessExpression)) {
        return false;
    }
    const expr = isElementAccessExpression(name) ? skipParentheses(name.argumentExpression) : name.expression;
    return !isStringOrNumericLiteralLike(expr) &&
        !isSignedNumericLiteral(expr);
}

/** @internal */
export function getTextOfIdentifierOrLiteral(node: PropertyNameLiteral | PrivateIdentifier): string {
    return isMemberName(node) ? idText(node) : isJsxNamespacedName(node) ? getTextOfJsxNamespacedName(node) : node.text;
}

// TODO(jakebailey): this function should not be named this. While it does technically
// return true if the argument is a ParameterDeclaration, it also returns true for nodes
// that are children of ParameterDeclarations inside binding elements.
// Probably, this should be called `rootDeclarationIsParameter`.

/** @internal */
export function nodeIsSynthesized(range: TextRange): boolean {
    return positionIsSynthesized(range.pos)
        || positionIsSynthesized(range.end);
}

/** @internal */
export const enum Associativity {
    Left,
    Right,
}

/** @internal */
export function getExpressionAssociativity(expression: Expression) {
    const operator = getOperator(expression);
    const hasArguments = expression.kind === SyntaxKind.NewExpression && (expression as NewExpression).arguments !== undefined;
    return getOperatorAssociativity(expression.kind, operator, hasArguments);
}

/** @internal */
export function getOperatorAssociativity(kind: SyntaxKind, operator: SyntaxKind, hasArguments?: boolean) {
    switch (kind) {
        case SyntaxKind.NewExpression:
            return hasArguments ? Associativity.Left : Associativity.Right;

        case SyntaxKind.PrefixUnaryExpression:
        case SyntaxKind.TypeOfExpression:
        case SyntaxKind.VoidExpression:
        case SyntaxKind.DeleteExpression:
        case SyntaxKind.AwaitExpression:
        case SyntaxKind.ConditionalExpression:
        case SyntaxKind.YieldExpression:
            return Associativity.Right;

        case SyntaxKind.BinaryExpression:
            switch (operator) {
                case SyntaxKind.AsteriskAsteriskToken:
                case SyntaxKind.EqualsToken:
                case SyntaxKind.PlusEqualsToken:
                case SyntaxKind.MinusEqualsToken:
                case SyntaxKind.AsteriskAsteriskEqualsToken:
                case SyntaxKind.AsteriskEqualsToken:
                case SyntaxKind.SlashEqualsToken:
                case SyntaxKind.PercentEqualsToken:
                case SyntaxKind.LessThanLessThanEqualsToken:
                case SyntaxKind.GreaterThanGreaterThanEqualsToken:
                case SyntaxKind.GreaterThanGreaterThanGreaterThanEqualsToken:
                case SyntaxKind.AmpersandEqualsToken:
                case SyntaxKind.CaretEqualsToken:
                case SyntaxKind.BarEqualsToken:
                case SyntaxKind.BarBarEqualsToken:
                case SyntaxKind.AmpersandAmpersandEqualsToken:
                case SyntaxKind.QuestionQuestionEqualsToken:
                    return Associativity.Right;
            }
    }
    return Associativity.Left;
}

/** @internal */
export function getExpressionPrecedence(expression: Expression) {
    const operator = getOperator(expression);
    const hasArguments = expression.kind === SyntaxKind.NewExpression && (expression as NewExpression).arguments !== undefined;
    return getOperatorPrecedence(expression.kind, operator, hasArguments);
}

/** @internal */
function getOperator(expression: Expression): SyntaxKind {
    if (expression.kind === SyntaxKind.BinaryExpression) {
        return (expression as BinaryExpression).operatorToken.kind;
    }
    else if (expression.kind === SyntaxKind.PrefixUnaryExpression || expression.kind === SyntaxKind.PostfixUnaryExpression) {
        return (expression as PrefixUnaryExpression | PostfixUnaryExpression).operator;
    }
    else {
        return expression.kind;
    }
}

/** @internal */
export const enum OperatorPrecedence {
    // Expression:
    //     AssignmentExpression
    //     Expression `,` AssignmentExpression
    Comma,

    // NOTE: `Spread` is higher than `Comma` due to how it is parsed in |ElementList|
    // SpreadElement:
    //     `...` AssignmentExpression
    Spread,

    // AssignmentExpression:
    //     ConditionalExpression
    //     YieldExpression
    //     ArrowFunction
    //     AsyncArrowFunction
    //     LeftHandSideExpression `=` AssignmentExpression
    //     LeftHandSideExpression AssignmentOperator AssignmentExpression
    //
    // NOTE: AssignmentExpression is broken down into several precedences due to the requirements
    //       of the parenthesizer rules.

    // AssignmentExpression: YieldExpression
    // YieldExpression:
    //     `yield`
    //     `yield` AssignmentExpression
    //     `yield` `*` AssignmentExpression
    Yield,

    // AssignmentExpression: LeftHandSideExpression `=` AssignmentExpression
    // AssignmentExpression: LeftHandSideExpression AssignmentOperator AssignmentExpression
    // AssignmentOperator: one of
    //     `*=` `/=` `%=` `+=` `-=` `<<=` `>>=` `>>>=` `&=` `^=` `|=` `**=`
    Assignment,

    // NOTE: `Conditional` is considered higher than `Assignment` here, but in reality they have
    //       the same precedence.
    // AssignmentExpression: ConditionalExpression
    // ConditionalExpression:
    //     ShortCircuitExpression
    //     ShortCircuitExpression `?` AssignmentExpression `:` AssignmentExpression
    // ShortCircuitExpression:
    //     LogicalORExpression
    //     CoalesceExpression
    Conditional,

    // CoalesceExpression:
    //     CoalesceExpressionHead `??` BitwiseORExpression
    // CoalesceExpressionHead:
    //     CoalesceExpression
    //     BitwiseORExpression
    Coalesce = Conditional, // NOTE: This is wrong

    // LogicalORExpression:
    //     LogicalANDExpression
    //     LogicalORExpression `||` LogicalANDExpression
    LogicalOR,

    // LogicalANDExpression:
    //     BitwiseORExpression
    //     LogicalANDExprerssion `&&` BitwiseORExpression
    LogicalAND,

    // BitwiseORExpression:
    //     BitwiseXORExpression
    //     BitwiseORExpression `^` BitwiseXORExpression
    BitwiseOR,

    // BitwiseXORExpression:
    //     BitwiseANDExpression
    //     BitwiseXORExpression `^` BitwiseANDExpression
    BitwiseXOR,

    // BitwiseANDExpression:
    //     EqualityExpression
    //     BitwiseANDExpression `^` EqualityExpression
    BitwiseAND,

    // EqualityExpression:
    //     RelationalExpression
    //     EqualityExpression `==` RelationalExpression
    //     EqualityExpression `!=` RelationalExpression
    //     EqualityExpression `===` RelationalExpression
    //     EqualityExpression `!==` RelationalExpression
    Equality,

    // RelationalExpression:
    //     ShiftExpression
    //     RelationalExpression `<` ShiftExpression
    //     RelationalExpression `>` ShiftExpression
    //     RelationalExpression `<=` ShiftExpression
    //     RelationalExpression `>=` ShiftExpression
    //     RelationalExpression `instanceof` ShiftExpression
    //     RelationalExpression `in` ShiftExpression
    //     [+TypeScript] RelationalExpression `as` Type
    Relational,

    // ShiftExpression:
    //     AdditiveExpression
    //     ShiftExpression `<<` AdditiveExpression
    //     ShiftExpression `>>` AdditiveExpression
    //     ShiftExpression `>>>` AdditiveExpression
    Shift,

    // AdditiveExpression:
    //     MultiplicativeExpression
    //     AdditiveExpression `+` MultiplicativeExpression
    //     AdditiveExpression `-` MultiplicativeExpression
    Additive,

    // MultiplicativeExpression:
    //     ExponentiationExpression
    //     MultiplicativeExpression MultiplicativeOperator ExponentiationExpression
    // MultiplicativeOperator: one of `*`, `/`, `%`
    Multiplicative,

    // ExponentiationExpression:
    //     UnaryExpression
    //     UpdateExpression `**` ExponentiationExpression
    Exponentiation,

    // UnaryExpression:
    //     UpdateExpression
    //     `delete` UnaryExpression
    //     `void` UnaryExpression
    //     `typeof` UnaryExpression
    //     `+` UnaryExpression
    //     `-` UnaryExpression
    //     `~` UnaryExpression
    //     `!` UnaryExpression
    //     AwaitExpression
    // UpdateExpression:            // TODO: Do we need to investigate the precedence here?
    //     `++` UnaryExpression
    //     `--` UnaryExpression
    Unary,

    // UpdateExpression:
    //     LeftHandSideExpression
    //     LeftHandSideExpression `++`
    //     LeftHandSideExpression `--`
    Update,

    // LeftHandSideExpression:
    //     NewExpression
    //     CallExpression
    // NewExpression:
    //     MemberExpression
    //     `new` NewExpression
    LeftHandSide,

    // CallExpression:
    //     CoverCallExpressionAndAsyncArrowHead
    //     SuperCall
    //     ImportCall
    //     CallExpression Arguments
    //     CallExpression `[` Expression `]`
    //     CallExpression `.` IdentifierName
    //     CallExpression TemplateLiteral
    // MemberExpression:
    //     PrimaryExpression
    //     MemberExpression `[` Expression `]`
    //     MemberExpression `.` IdentifierName
    //     MemberExpression TemplateLiteral
    //     SuperProperty
    //     MetaProperty
    //     `new` MemberExpression Arguments
    Member,

    // TODO: JSXElement?
    // PrimaryExpression:
    //     `this`
    //     IdentifierReference
    //     Literal
    //     ArrayLiteral
    //     ObjectLiteral
    //     FunctionExpression
    //     ClassExpression
    //     GeneratorExpression
    //     AsyncFunctionExpression
    //     AsyncGeneratorExpression
    //     RegularExpressionLiteral
    //     TemplateLiteral
    //     CoverParenthesizedExpressionAndArrowParameterList
    Primary,

    Highest = Primary,
    Lowest = Comma,
    // -1 is lower than all other precedences. Returning it will cause binary expression
    // parsing to stop.
    Invalid = -1,
}

/** @internal */
export function getOperatorPrecedence(nodeKind: SyntaxKind, operatorKind: SyntaxKind, hasArguments?: boolean) {
    switch (nodeKind) {
        case SyntaxKind.CommaListExpression:
            return OperatorPrecedence.Comma;

        case SyntaxKind.SpreadElement:
            return OperatorPrecedence.Spread;

        case SyntaxKind.YieldExpression:
            return OperatorPrecedence.Yield;

        case SyntaxKind.ConditionalExpression:
            return OperatorPrecedence.Conditional;

        case SyntaxKind.BinaryExpression:
            switch (operatorKind) {
                case SyntaxKind.CommaToken:
                    return OperatorPrecedence.Comma;

                case SyntaxKind.EqualsToken:
                case SyntaxKind.PlusEqualsToken:
                case SyntaxKind.MinusEqualsToken:
                case SyntaxKind.AsteriskAsteriskEqualsToken:
                case SyntaxKind.AsteriskEqualsToken:
                case SyntaxKind.SlashEqualsToken:
                case SyntaxKind.PercentEqualsToken:
                case SyntaxKind.LessThanLessThanEqualsToken:
                case SyntaxKind.GreaterThanGreaterThanEqualsToken:
                case SyntaxKind.GreaterThanGreaterThanGreaterThanEqualsToken:
                case SyntaxKind.AmpersandEqualsToken:
                case SyntaxKind.CaretEqualsToken:
                case SyntaxKind.BarEqualsToken:
                case SyntaxKind.BarBarEqualsToken:
                case SyntaxKind.AmpersandAmpersandEqualsToken:
                case SyntaxKind.QuestionQuestionEqualsToken:
                    return OperatorPrecedence.Assignment;

                default:
                    return getBinaryOperatorPrecedence(operatorKind);
            }

        // TODO: Should prefix `++` and `--` be moved to the `Update` precedence?
        case SyntaxKind.TypeAssertionExpression:
        case SyntaxKind.NonNullExpression:
        case SyntaxKind.PrefixUnaryExpression:
        case SyntaxKind.TypeOfExpression:
        case SyntaxKind.VoidExpression:
        case SyntaxKind.DeleteExpression:
        case SyntaxKind.AwaitExpression:
            return OperatorPrecedence.Unary;

        case SyntaxKind.PostfixUnaryExpression:
            return OperatorPrecedence.Update;

        case SyntaxKind.CallExpression:
            return OperatorPrecedence.LeftHandSide;

        case SyntaxKind.NewExpression:
            return hasArguments ? OperatorPrecedence.Member : OperatorPrecedence.LeftHandSide;

        case SyntaxKind.TaggedTemplateExpression:
        case SyntaxKind.PropertyAccessExpression:
        case SyntaxKind.ElementAccessExpression:
        case SyntaxKind.MetaProperty:
            return OperatorPrecedence.Member;

        case SyntaxKind.AsExpression:
        case SyntaxKind.SatisfiesExpression:
            return OperatorPrecedence.Relational;

        case SyntaxKind.ThisKeyword:
        case SyntaxKind.SuperKeyword:
        case SyntaxKind.Identifier:
        case SyntaxKind.PrivateIdentifier:
        case SyntaxKind.NullKeyword:
        case SyntaxKind.TrueKeyword:
        case SyntaxKind.FalseKeyword:
        case SyntaxKind.NumericLiteral:
        case SyntaxKind.BigIntLiteral:
        case SyntaxKind.StringLiteral:
        case SyntaxKind.ArrayLiteralExpression:
        case SyntaxKind.ObjectLiteralExpression:
        case SyntaxKind.FunctionExpression:
        case SyntaxKind.ArrowFunction:
        case SyntaxKind.ClassExpression:
        case SyntaxKind.RegularExpressionLiteral:
        case SyntaxKind.NoSubstitutionTemplateLiteral:
        case SyntaxKind.TemplateExpression:
        case SyntaxKind.ParenthesizedExpression:
        case SyntaxKind.OmittedExpression:
        case SyntaxKind.JsxElement:
        case SyntaxKind.JsxSelfClosingElement:
        case SyntaxKind.JsxFragment:
            return OperatorPrecedence.Primary;

        default:
            return OperatorPrecedence.Invalid;
    }
}

/** @internal */
export function getBinaryOperatorPrecedence(kind: SyntaxKind): OperatorPrecedence {
    switch (kind) {
        case SyntaxKind.QuestionQuestionToken:
            return OperatorPrecedence.Coalesce;
        case SyntaxKind.BarBarToken:
            return OperatorPrecedence.LogicalOR;
        case SyntaxKind.AmpersandAmpersandToken:
            return OperatorPrecedence.LogicalAND;
        case SyntaxKind.BarToken:
            return OperatorPrecedence.BitwiseOR;
        case SyntaxKind.CaretToken:
            return OperatorPrecedence.BitwiseXOR;
        case SyntaxKind.AmpersandToken:
            return OperatorPrecedence.BitwiseAND;
        case SyntaxKind.EqualsEqualsToken:
        case SyntaxKind.ExclamationEqualsToken:
        case SyntaxKind.EqualsEqualsEqualsToken:
        case SyntaxKind.ExclamationEqualsEqualsToken:
            return OperatorPrecedence.Equality;
        case SyntaxKind.LessThanToken:
        case SyntaxKind.GreaterThanToken:
        case SyntaxKind.LessThanEqualsToken:
        case SyntaxKind.GreaterThanEqualsToken:
        case SyntaxKind.InstanceOfKeyword:
        case SyntaxKind.InKeyword:
        case SyntaxKind.AsKeyword:
        case SyntaxKind.SatisfiesKeyword:
            return OperatorPrecedence.Relational;
        case SyntaxKind.LessThanLessThanToken:
        case SyntaxKind.GreaterThanGreaterThanToken:
        case SyntaxKind.GreaterThanGreaterThanGreaterThanToken:
            return OperatorPrecedence.Shift;
        case SyntaxKind.PlusToken:
        case SyntaxKind.MinusToken:
            return OperatorPrecedence.Additive;
        case SyntaxKind.AsteriskToken:
        case SyntaxKind.SlashToken:
        case SyntaxKind.PercentToken:
            return OperatorPrecedence.Multiplicative;
        case SyntaxKind.AsteriskAsteriskToken:
            return OperatorPrecedence.Exponentiation;
    }

    // -1 is lower than all other precedences.  Returning it will cause binary expression
    // parsing to stop.
    return -1;
}

function containsInvalidEscapeFlag(node: TemplateLiteralToken): boolean {
    return !!((node.templateFlags || 0) & TokenFlags.ContainsInvalidEscape);
}

/** @internal */
export function hasInvalidEscape(template: TemplateLiteral): boolean {
    return template && !!(isNoSubstitutionTemplateLiteral(template)
        ? containsInvalidEscapeFlag(template)
        : (containsInvalidEscapeFlag(template.head) || some(template.templateSpans, span => containsInvalidEscapeFlag(span.literal))));
}

// This consists of the first 19 unprintable ASCII characters, canonical escapes, lineSeparator,
// paragraphSeparator, and nextLine. The latter three are just desirable to suppress new lines in
// the language service. These characters should be escaped when printing, and if any characters are added,
// the map below must be updated. Note that this regexp *does not* include the 'delete' character.
// There is no reason for this other than that JSON.stringify does not handle it either.
// Template strings preserve simple LF newlines, still encode CRLF (or CR)

// This consists of the first 19 unprintable ASCII characters, JSX canonical escapes, lineSeparator,
// paragraphSeparator, and nextLine. The latter three are just desirable to suppress new lines in
// the language service. These characters should be escaped when printing, and if any characters are added,
// the map below must be updated.

/** @internal */
function getThisParameter(signature: SignatureDeclaration | JSDocSignature): ParameterDeclaration | undefined {
    // callback tags do not currently support this parameters
    if (signature.parameters.length && !isJSDocSignature(signature)) {
        const thisParameter = signature.parameters[0];
        if (parameterIsThisKeyword(thisParameter)) {
            return thisParameter;
        }
    }
}

/** @internal */
function parameterIsThisKeyword(parameter: ParameterDeclaration): boolean {
    return isThisIdentifier(parameter.name);
}

/** @internal */
export function isThisIdentifier(node: Node | undefined): boolean {
    return !!node && node.kind === SyntaxKind.Identifier && identifierIsThisKeyword(node as Identifier);
}

/** @internal */
function identifierIsThisKeyword(id: Identifier): boolean {
    return id.escapedText === "this";
}

/** @internal */
export function hasSyntacticModifier(node: Node, flags: ModifierFlags): boolean {
    return !!getSelectedSyntacticModifierFlags(node, flags);
}

/** @internal */
function hasAbstractModifier(node: Node): boolean {
    return hasSyntacticModifier(node, ModifierFlags.Abstract);
}

/** @internal */
function hasAmbientModifier(node: Node): boolean {
    return hasSyntacticModifier(node, ModifierFlags.Ambient);
}

/** @internal */
function hasDecorators(node: Node): boolean {
    return hasSyntacticModifier(node, ModifierFlags.Decorator);
}

/** @internal */
function getSelectedSyntacticModifierFlags(node: Node, flags: ModifierFlags): ModifierFlags {
    return getSyntacticModifierFlags(node) & flags;
}

function getModifierFlagsWorker(node: Node, includeJSDoc: boolean, alwaysIncludeJSDoc?: boolean): ModifierFlags {
    if (node.kind >= SyntaxKind.FirstToken && node.kind <= SyntaxKind.LastToken) {
        return ModifierFlags.None;
    }

    if (!(node.modifierFlagsCache & ModifierFlags.HasComputedFlags)) {
        node.modifierFlagsCache = getSyntacticModifierFlagsNoCache(node) | ModifierFlags.HasComputedFlags;
    }

    if (alwaysIncludeJSDoc || includeJSDoc && isInJSFile(node)) {
        if (!(node.modifierFlagsCache & ModifierFlags.HasComputedJSDocModifiers) && node.parent) {
            node.modifierFlagsCache |= getRawJSDocModifierFlagsNoCache(node) | ModifierFlags.HasComputedJSDocModifiers;
        }
        return selectEffectiveModifierFlags(node.modifierFlagsCache);
    }

    return selectSyntacticModifierFlags(node.modifierFlagsCache);
}

/**
 * Gets the ModifierFlags for syntactic modifiers on the provided node. The modifiers will be cached on the node to improve performance.
 *
 * NOTE: This function does not use `parent` pointers and will not include modifiers from JSDoc.
 *
 * @internal
 */
function getSyntacticModifierFlags(node: Node): ModifierFlags {
    return getModifierFlagsWorker(node, /*includeJSDoc*/ false);
}

function getRawJSDocModifierFlagsNoCache(node: Node): ModifierFlags {
    let flags = ModifierFlags.None;
    if (!!node.parent && !isParameter(node)) {
        if (isInJSFile(node)) {
            if (getJSDocPublicTagNoCache(node)) flags |= ModifierFlags.JSDocPublic;
            if (getJSDocPrivateTagNoCache(node)) flags |= ModifierFlags.JSDocPrivate;
            if (getJSDocProtectedTagNoCache(node)) flags |= ModifierFlags.JSDocProtected;
            if (getJSDocReadonlyTagNoCache(node)) flags |= ModifierFlags.JSDocReadonly;
            if (getJSDocOverrideTagNoCache(node)) flags |= ModifierFlags.JSDocOverride;
        }
        if (getJSDocDeprecatedTagNoCache(node)) flags |= ModifierFlags.Deprecated;
    }

    return flags;
}

function selectSyntacticModifierFlags(flags: ModifierFlags) {
    return flags & ModifierFlags.SyntacticModifiers;
}

function selectEffectiveModifierFlags(flags: ModifierFlags) {
    return (flags & ModifierFlags.NonCacheOnlyModifiers) |
        ((flags & ModifierFlags.JSDocCacheOnlyModifiers) >>> 23); // shift ModifierFlags.JSDoc* to match ModifierFlags.*
}

/**
 * Gets the ModifierFlags for syntactic modifiers on the provided node. The modifier flags cache on the node is ignored.
 *
 * NOTE: This function does not use `parent` pointers and will not include modifiers from JSDoc.
 *
 * @internal
 */
function getSyntacticModifierFlagsNoCache(node: Node): ModifierFlags {
    let flags = canHaveModifiers(node) ? modifiersToFlags(node.modifiers) : ModifierFlags.None;
    if (node.flags & NodeFlags.NestedNamespace || node.kind === SyntaxKind.Identifier && node.flags & NodeFlags.IdentifierIsInJSDocNamespace) {
        flags |= ModifierFlags.Export;
    }
    return flags;
}

/** @internal */
export function modifiersToFlags(modifiers: readonly ModifierLike[] | undefined) {
    let flags = ModifierFlags.None;
    if (modifiers) {
        for (const modifier of modifiers) {
            flags |= modifierToFlag(modifier.kind);
        }
    }
    return flags;
}

/** @internal */
export function modifierToFlag(token: SyntaxKind): ModifierFlags {
    switch (token) {
        case SyntaxKind.StaticKeyword:
            return ModifierFlags.Static;
        case SyntaxKind.PublicKeyword:
            return ModifierFlags.Public;
        case SyntaxKind.ProtectedKeyword:
            return ModifierFlags.Protected;
        case SyntaxKind.PrivateKeyword:
            return ModifierFlags.Private;
        case SyntaxKind.AbstractKeyword:
            return ModifierFlags.Abstract;
        case SyntaxKind.AccessorKeyword:
            return ModifierFlags.Accessor;
        case SyntaxKind.ExportKeyword:
            return ModifierFlags.Export;
        case SyntaxKind.DeclareKeyword:
            return ModifierFlags.Ambient;
        case SyntaxKind.ConstKeyword:
            return ModifierFlags.Const;
        case SyntaxKind.DefaultKeyword:
            return ModifierFlags.Default;
        case SyntaxKind.AsyncKeyword:
            return ModifierFlags.Async;
        case SyntaxKind.ReadonlyKeyword:
            return ModifierFlags.Readonly;
        case SyntaxKind.OverrideKeyword:
            return ModifierFlags.Override;
        case SyntaxKind.InKeyword:
            return ModifierFlags.In;
        case SyntaxKind.OutKeyword:
            return ModifierFlags.Out;
        case SyntaxKind.Decorator:
            return ModifierFlags.Decorator;
    }
    return ModifierFlags.None;
}

/** @internal */
export function isLogicalOrCoalescingAssignmentOperator(token: SyntaxKind): token is LogicalOrCoalescingAssignmentOperator {
    return token === SyntaxKind.BarBarEqualsToken
        || token === SyntaxKind.AmpersandAmpersandEqualsToken
        || token === SyntaxKind.QuestionQuestionEqualsToken;
}

/** @internal */
export function isAssignmentOperator(token: SyntaxKind): boolean {
    return token >= SyntaxKind.FirstAssignment && token <= SyntaxKind.LastAssignment;
}

/**
 * Get `C` given `N` if `N` is in the position `class C extends N` where `N` is an ExpressionWithTypeArguments.
 *
 * @internal
 */
function tryGetClassExtendingExpressionWithTypeArguments(node: Node): ClassLikeDeclaration | undefined {
    const cls = tryGetClassImplementingOrExtendingExpressionWithTypeArguments(node);
    return cls && !cls.isImplements ? cls.class : undefined;
}

/** @internal */
interface ClassImplementingOrExtendingExpressionWithTypeArguments {
    readonly class: ClassLikeDeclaration;
    readonly isImplements: boolean;
}
/** @internal */
function tryGetClassImplementingOrExtendingExpressionWithTypeArguments(node: Node): ClassImplementingOrExtendingExpressionWithTypeArguments | undefined {
    if (isExpressionWithTypeArguments(node)) {
        if (isHeritageClause(node.parent) && isClassLike(node.parent.parent)) {
            return { class: node.parent.parent, isImplements: node.parent.token === SyntaxKind.ImplementsKeyword };
        }
        if (isJSDocAugmentsTag(node.parent)) {
            const host = getEffectiveJSDocHost(node.parent);
            if (host && isClassLike(host)) {
                return { class: host, isImplements: false };
            }
        }
    }
    return undefined;
}

/** @internal */
export function isAssignmentExpression(node: Node, excludeCompoundAssignment: true): node is AssignmentExpression<EqualsToken>;
/** @internal */
export function isAssignmentExpression(node: Node, excludeCompoundAssignment?: false): node is AssignmentExpression<AssignmentOperatorToken>;
/** @internal */
export function isAssignmentExpression(node: Node, excludeCompoundAssignment?: boolean): node is AssignmentExpression<AssignmentOperatorToken> {
    return isBinaryExpression(node)
        && (excludeCompoundAssignment
            ? node.operatorToken.kind === SyntaxKind.EqualsToken
            : isAssignmentOperator(node.operatorToken.kind))
        && isLeftHandSideExpression(node.left);
}

/** @internal */
function isExpressionWithTypeArgumentsInClassExtendsClause(node: Node): node is ExpressionWithTypeArguments {
    return tryGetClassExtendingExpressionWithTypeArguments(node) !== undefined;
}

/** @internal */
function isEntityNameExpression(node: Node): node is EntityNameExpression {
    return node.kind === SyntaxKind.Identifier || isPropertyAccessEntityNameExpression(node);
}

/** @internal */
function isPropertyAccessEntityNameExpression(node: Node): node is PropertyAccessEntityNameExpression {
    return isPropertyAccessExpression(node) && isIdentifier(node.name) && isEntityNameExpression(node.expression);
}

/** @internal */
function isPrototypeAccess(node: Node): node is BindableStaticAccessExpression {
    return isBindableStaticAccessExpression(node) && getElementOrPropertyAccessName(node) === "prototype";
}

/** @internal */
export function getLastChild(node: Node): Node | undefined {
    let lastChild: Node | undefined;
    forEachChild(node, child => {
        if (nodeIsPresent(child)) lastChild = child;
    }, children => {
        // As an optimization, jump straight to the end of the list.
        for (let i = children.length - 1; i >= 0; i--) {
            if (nodeIsPresent(children[i])) {
                lastChild = children[i];
                break;
            }
        }
    });
    return lastChild;
}

/** @internal */
export function isAccessExpression(node: Node): node is AccessExpression {
    return node.kind === SyntaxKind.PropertyAccessExpression || node.kind === SyntaxKind.ElementAccessExpression;
}

/** @internal */
export function getLeftmostExpression(node: Expression, stopAtCallExpressions: boolean) {
    while (true) {
        switch (node.kind) {
            case SyntaxKind.PostfixUnaryExpression:
                node = (node as PostfixUnaryExpression).operand;
                continue;

            case SyntaxKind.BinaryExpression:
                node = (node as BinaryExpression).left;
                continue;

            case SyntaxKind.ConditionalExpression:
                node = (node as ConditionalExpression).condition;
                continue;

            case SyntaxKind.TaggedTemplateExpression:
                node = (node as TaggedTemplateExpression).tag;
                continue;

            case SyntaxKind.CallExpression:
                if (stopAtCallExpressions) {
                    return node;
                }
                // falls through
            case SyntaxKind.AsExpression:
            case SyntaxKind.ElementAccessExpression:
            case SyntaxKind.PropertyAccessExpression:
            case SyntaxKind.NonNullExpression:
            case SyntaxKind.PartiallyEmittedExpression:
            case SyntaxKind.SatisfiesExpression:
                node = (node as CallExpression | PropertyAccessExpression | ElementAccessExpression | AsExpression | NonNullExpression | PartiallyEmittedExpression | SatisfiesExpression).expression;
                continue;
        }

        return node;
    }
}

/** @internal */
interface ObjectAllocator {
    getNodeConstructor(): new (kind: SyntaxKind, pos: number, end: number) => Node;
    getTokenConstructor(): new <TKind extends SyntaxKind>(kind: TKind, pos: number, end: number) => Token<TKind>;
    getIdentifierConstructor(): new (kind: SyntaxKind.Identifier, pos: number, end: number) => Identifier;
    getPrivateIdentifierConstructor(): new (kind: SyntaxKind.PrivateIdentifier, pos: number, end: number) => PrivateIdentifier;
    getSourceFileConstructor(): new (kind: SyntaxKind.SourceFile, pos: number, end: number) => SourceFile;
}

function Node(this: Mutable<Node>, kind: SyntaxKind, pos: number, end: number) {
    this.pos = pos;
    this.end = end;
    this.kind = kind;
    this.id = 0;
    this.flags = NodeFlags.None;
    this.modifierFlagsCache = ModifierFlags.None;
    this.transformFlags = TransformFlags.None;
    this.parent = undefined!;
    this.original = undefined;
    this.emitNode = undefined;
}

function Token(this: Mutable<Node>, kind: SyntaxKind, pos: number, end: number) {
    this.pos = pos;
    this.end = end;
    this.kind = kind;
    this.id = 0;
    this.flags = NodeFlags.None;
    this.transformFlags = TransformFlags.None;
    this.parent = undefined!;
    this.emitNode = undefined;
}

function Identifier(this: Mutable<Node>, kind: SyntaxKind, pos: number, end: number) {
    this.pos = pos;
    this.end = end;
    this.kind = kind;
    this.id = 0;
    this.flags = NodeFlags.None;
    this.transformFlags = TransformFlags.None;
    this.parent = undefined!;
    this.original = undefined;
    this.emitNode = undefined;
}

/** @internal */
export const objectAllocator: ObjectAllocator = {
    getNodeConstructor: () => Node as any,
    getTokenConstructor: () => Token as any,
    getIdentifierConstructor: () => Identifier as any,
    getPrivateIdentifierConstructor: () => Node as any,
    getSourceFileConstructor: () => Node as any,
};

/** @internal */
function formatStringFromArgs(text: string, args: DiagnosticArguments): string {
    return text.replace(/{(\d+)}/g, (_match, index: string) => "" + Debug.checkDefined(args[+index]));
}

let localizedDiagnosticMessages: MapLike<string> | undefined;

/** @internal */
function getLocaleSpecificMessage(message: DiagnosticMessage) {
    return localizedDiagnosticMessages && localizedDiagnosticMessages[message.key] || message.message;
}

/** @internal */
export function createDetachedDiagnostic(fileName: string, sourceText: string, start: number, length: number, message: DiagnosticMessage, ...args: DiagnosticArguments): DiagnosticWithDetachedLocation {
    if ((start + length) > sourceText.length) {
        length = sourceText.length - start;
    }

    assertDiagnosticLocation(sourceText, start, length);
    let text = getLocaleSpecificMessage(message);

    if (some(args)) {
        text = formatStringFromArgs(text, args);
    }

    return {
        file: undefined,
        start,
        length,

        messageText: text,
        category: message.category,
        code: message.code,
        reportsUnnecessary: message.reportsUnnecessary,
        fileName,
    };
}

function isDiagnosticWithDetachedLocation(diagnostic: DiagnosticRelatedInformation | DiagnosticWithDetachedLocation): diagnostic is DiagnosticWithDetachedLocation {
    return diagnostic.file === undefined
        && diagnostic.start !== undefined
        && diagnostic.length !== undefined
        && typeof (diagnostic as DiagnosticWithDetachedLocation).fileName === "string";
}

function attachFileToDiagnostic(diagnostic: DiagnosticWithDetachedLocation, file: SourceFile): DiagnosticWithLocation {
    const fileName = file.fileName || "";
    const length = file.text.length;
    Debug.assertEqual(diagnostic.fileName, fileName);
    Debug.assertLessThanOrEqual(diagnostic.start, length);
    Debug.assertLessThanOrEqual(diagnostic.start + diagnostic.length, length);
    const diagnosticWithLocation: DiagnosticWithLocation = {
        file,
        start: diagnostic.start,
        length: diagnostic.length,
        messageText: diagnostic.messageText,
        category: diagnostic.category,
        code: diagnostic.code,
        reportsUnnecessary: diagnostic.reportsUnnecessary,
    };
    if (diagnostic.relatedInformation) {
        diagnosticWithLocation.relatedInformation = [];
        for (const related of diagnostic.relatedInformation) {
            if (isDiagnosticWithDetachedLocation(related) && related.fileName === fileName) {
                Debug.assertLessThanOrEqual(related.start, length);
                Debug.assertLessThanOrEqual(related.start + related.length, length);
                diagnosticWithLocation.relatedInformation.push(attachFileToDiagnostic(related, file));
            }
            else {
                diagnosticWithLocation.relatedInformation.push(related);
            }
        }
    }
    return diagnosticWithLocation;
}

/** @internal */
export function attachFileToDiagnostics(diagnostics: DiagnosticWithDetachedLocation[], file: SourceFile): DiagnosticWithLocation[] {
    const diagnosticsWithLocation: DiagnosticWithLocation[] = [];
    for (const diagnostic of diagnostics) {
        diagnosticsWithLocation.push(attachFileToDiagnostic(diagnostic, file));
    }
    return diagnosticsWithLocation;
}

/** @internal */
function createFileDiagnostic(file: SourceFile, start: number, length: number, message: DiagnosticMessage, ...args: DiagnosticArguments): DiagnosticWithLocation {
    assertDiagnosticLocation(file.text, start, length);

    let text = getLocaleSpecificMessage(message);

    if (some(args)) {
        text = formatStringFromArgs(text, args);
    }

    return {
        file,
        start,
        length,

        messageText: text,
        category: message.category,
        code: message.code,
        reportsUnnecessary: message.reportsUnnecessary,
        reportsDeprecated: message.reportsDeprecated,
    };
}

/** @internal */
export function getLanguageVariant(scriptKind: ScriptKind) {
    // .tsx and .jsx files are treated as jsx language variant.
    return scriptKind === ScriptKind.TSX || scriptKind === ScriptKind.JSX || scriptKind === ScriptKind.JS || scriptKind === ScriptKind.JSON ? LanguageVariant.JSX : LanguageVariant.Standard;
}

// KLUDGE: Don't assume one 'node_modules' links to another. More likely a single directory inside the node_modules is the symlink.
// ALso, don't assume that an `@foo` directory is linked. More likely the contents of that are linked.

// Reserved characters, forces escaping of any non-word (or digit), non-whitespace character.
// It may be inefficient (we could just match (/[-[\]{}()*+?.,\\^$|#\s]/g), but this is future
// proof.

/** @internal */
export function ensureScriptKind(fileName: string, scriptKind: ScriptKind | undefined): ScriptKind {
    // Using scriptKind as a condition handles both:
    // - 'scriptKind' is unspecified and thus it is `undefined`
    // - 'scriptKind' is set and it is `Unknown` (0)
    // If the 'scriptKind' is 'undefined' or 'Unknown' then we attempt
    // to get the ScriptKind from the file name. If it cannot be resolved
    // from the file name then the default 'TS' script kind is returned.
    return scriptKind || getScriptKindFromFileName(fileName) || ScriptKind.TS;
}

/** @internal */
function getScriptKindFromFileName(fileName: string): ScriptKind {
    const ext = fileName.substr(fileName.lastIndexOf("."));
    switch (ext.toLowerCase()) {
        case Extension.Js:
        case Extension.Cjs:
        case Extension.Mjs:
            return ScriptKind.JS;
        case Extension.Jsx:
            return ScriptKind.JSX;
        case Extension.Ts:
        case Extension.Cts:
        case Extension.Mts:
            return ScriptKind.TS;
        case Extension.Tsx:
            return ScriptKind.TSX;
        case Extension.Json:
            return ScriptKind.JSON;
        default:
            return ScriptKind.Unknown;
    }
}

/** @internal */
export const supportedDeclarationExtensions: readonly Extension[] = [Extension.Dts, Extension.Dcts, Extension.Dmts];

/** @internal */
export function positionIsSynthesized(pos: number): boolean {
    // This is a fast way of testing the following conditions:
    //  pos === undefined || pos === null || isNaN(pos) || pos < 0;
    return !(pos >= 0);
}

/** @internal */
export type Mutable<T extends object> = { -readonly [K in keyof T]: T[K]; };

/** @internal */
export function addRelatedInfo<T extends Diagnostic>(diagnostic: T, ...relatedInformation: DiagnosticRelatedInformation[]): T {
    if (!relatedInformation.length) {
        return diagnostic;
    }
    if (!diagnostic.relatedInformation) {
        diagnostic.relatedInformation = [];
    }
    Debug.assert(diagnostic.relatedInformation !== emptyArray, "Diagnostic had empty array singleton for related info, but is still being constructed!");
    diagnostic.relatedInformation.push(...relatedInformation);
    return diagnostic;
}

/**
 * Converts a bigint literal string, e.g. `0x1234n`,
 * to its decimal string representation, e.g. `4660`.
 *
 * @internal
 */
export function parsePseudoBigInt(stringValue: string): string {
    let log2Base: number;
    switch (stringValue.charCodeAt(1)) { // "x" in "0x123"
        case CharacterCodes.b:
        case CharacterCodes.B: // 0b or 0B
            log2Base = 1;
            break;
        case CharacterCodes.o:
        case CharacterCodes.O: // 0o or 0O
            log2Base = 3;
            break;
        case CharacterCodes.x:
        case CharacterCodes.X: // 0x or 0X
            log2Base = 4;
            break;
        default: // already in decimal; omit trailing "n"
            const nIndex = stringValue.length - 1;
            // Skip leading 0s
            let nonZeroStart = 0;
            while (stringValue.charCodeAt(nonZeroStart) === CharacterCodes._0) {
                nonZeroStart++;
            }
            return stringValue.slice(nonZeroStart, nIndex) || "0";
    }

    // Omit leading "0b", "0o", or "0x", and trailing "n"
    const startIndex = 2, endIndex = stringValue.length - 1;
    const bitsNeeded = (endIndex - startIndex) * log2Base;
    // Stores the value specified by the string as a LE array of 16-bit integers
    // using Uint16 instead of Uint32 so combining steps can use bitwise operators
    const segments = new Uint16Array((bitsNeeded >>> 4) + (bitsNeeded & 15 ? 1 : 0));
    // Add the digits, one at a time
    for (let i = endIndex - 1, bitOffset = 0; i >= startIndex; i--, bitOffset += log2Base) {
        const segment = bitOffset >>> 4;
        const digitChar = stringValue.charCodeAt(i);
        // Find character range: 0-9 < A-F < a-f
        const digit = digitChar <= CharacterCodes._9
            ? digitChar - CharacterCodes._0
            : 10 + digitChar -
                (digitChar <= CharacterCodes.F ? CharacterCodes.A : CharacterCodes.a);
        const shiftedDigit = digit << (bitOffset & 15);
        segments[segment] |= shiftedDigit;
        const residual = shiftedDigit >>> 16;
        if (residual) segments[segment + 1] |= residual; // overflows segment
    }
    // Repeatedly divide segments by 10 and add remainder to base10Value
    let base10Value = "";
    let firstNonzeroSegment = segments.length - 1;
    let segmentsRemaining = true;
    while (segmentsRemaining) {
        let mod10 = 0;
        segmentsRemaining = false;
        for (let segment = firstNonzeroSegment; segment >= 0; segment--) {
            const newSegment = mod10 << 16 | segments[segment];
            const segmentValue = (newSegment / 10) | 0;
            segments[segment] = segmentValue;
            mod10 = newSegment - segmentValue * 10;
            if (segmentValue && !segmentsRemaining) {
                firstNonzeroSegment = segment;
                segmentsRemaining = true;
            }
        }
        base10Value = mod10 + base10Value;
    }
    return base10Value;
}

/** @internal */
export function pseudoBigIntToString({ negative, base10Value }: PseudoBigInt): string {
    return (negative && base10Value !== "0" ? "-" : "") + base10Value;
}

/**
 * Bypasses immutability and directly sets the `pos` property of a `TextRange` or `Node`.
 *
 * @internal
 */
export function setTextRangePos<T extends ReadonlyTextRange>(range: T, pos: number) {
    (range as TextRange).pos = pos;
    return range;
}

/**
 * Bypasses immutability and directly sets the `end` property of a `TextRange` or `Node`.
 *
 * @internal
 */
function setTextRangeEnd<T extends ReadonlyTextRange>(range: T, end: number) {
    (range as TextRange).end = end;
    return range;
}

/**
 * Bypasses immutability and directly sets the `pos` and `end` properties of a `TextRange` or `Node`.
 *
 * @internal
 */
export function setTextRangePosEnd<T extends ReadonlyTextRange>(range: T, pos: number, end: number) {
    return setTextRangeEnd(setTextRangePos(range, pos), end);
}

/**
 * Bypasses immutability and directly sets the `pos` and `end` properties of a `TextRange` or `Node` from the
 * provided position and width.
 *
 * @internal
 */
export function setTextRangePosWidth<T extends ReadonlyTextRange>(range: T, pos: number, width: number) {
    return setTextRangePosEnd(range, pos, pos + width);
}

/**
 * Bypasses immutability and directly sets the `parent` property of a `Node`.
 *
 * @internal
 */
export function setParent<T extends Node>(child: T, parent: T["parent"] | undefined): T;
/** @internal */
export function setParent<T extends Node>(child: T | undefined, parent: T["parent"] | undefined): T | undefined;
/** @internal */
export function setParent<T extends Node>(child: T | undefined, parent: T["parent"] | undefined): T | undefined {
    if (child && parent) {
        (child as Mutable<T>).parent = parent;
    }
    return child;
}

/**
 * Bypasses immutability and directly sets the `parent` property of each `Node` recursively.
 * @param rootNode The root node from which to start the recursion.
 * @param incremental When `true`, only recursively descends through nodes whose `parent` pointers are incorrect.
 * This allows us to quickly bail out of setting `parent` for subtrees during incremental parsing.
 *
 * @internal
 */
export function setParentRecursive<T extends Node>(rootNode: T, incremental: boolean): T;
/** @internal */
export function setParentRecursive<T extends Node>(rootNode: T | undefined, incremental: boolean): T | undefined;
/** @internal */
export function setParentRecursive<T extends Node>(rootNode: T | undefined, incremental: boolean): T | undefined {
    if (!rootNode) return rootNode;
    forEachChildRecursively(rootNode, isJSDocNode(rootNode) ? bindParentToChildIgnoringJSDoc : bindParentToChild);
    return rootNode;

    function bindParentToChildIgnoringJSDoc(child: Node, parent: Node): void | "skip" {
        if (incremental && child.parent === parent) {
            return "skip";
        }
        setParent(child, parent);
    }

    function bindJSDoc(child: Node) {
        if (hasJSDocNodes(child)) {
            for (const doc of child.jsDoc!) {
                bindParentToChildIgnoringJSDoc(doc, child);
                forEachChildRecursively(doc, bindParentToChildIgnoringJSDoc);
            }
        }
    }

    function bindParentToChild(child: Node, parent: Node) {
        return bindParentToChildIgnoringJSDoc(child, parent) || bindJSDoc(child);
    }
}

/** @internal */
function getEscapedTextOfJsxNamespacedName(node: JsxNamespacedName): __String {
    return `${node.namespace.escapedText}:${idText(node.name)}` as __String;
}

/** @internal */
function getTextOfJsxNamespacedName(node: JsxNamespacedName) {
    return `${idText(node.namespace)}:${idText(node.name)}`;
}
