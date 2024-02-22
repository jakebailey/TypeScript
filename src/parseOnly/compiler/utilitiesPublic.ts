import {
    emptyArray,
    every,
    find,
    flatMap,
    hasProperty,
} from "./core";
import {
    Debug,
} from "./debug";
import {
    isArrowFunction,
    isBinaryExpression,
    isBindingElement,
    isCallExpression,
    isClassExpression,
    isElementAccessExpression,
    isFunctionExpression,
    isIdentifier,
    isJSDoc,
    isJSDocDeprecatedTag,
    isJSDocOverrideTag,
    isJSDocParameterTag,
    isJSDocPrivateTag,
    isJSDocProtectedTag,
    isJSDocPublicTag,
    isJSDocReadonlyTag,
    isJSDocTemplateTag,
    isJSDocTypeTag,
    isNonNullExpression,
    isOmittedExpression,
    isPrivateIdentifier,
    isPropertyAccessExpression,
    isPropertyAssignment,
    isVariableDeclaration,
} from "./factory/nodeTests";
import {
    skipOuterExpressions,
} from "./factory/utilities";
import {
    __String,
    AccessExpression,
    ArrayBindingElement,
    AssignmentDeclarationKind,
    AssignmentPattern,
    BinaryExpression,
    BindableObjectDefinePropertyCall,
    BindingElement,
    BindingName,
    BindingOrAssignmentElement,
    BindingPattern,
    Block,
    CallChain,
    CallExpression,
    CharacterCodes,
    ClassLikeDeclaration,
    ConstructorTypeNode,
    Declaration,
    DeclarationName,
    ElementAccessChain,
    ElementAccessExpression,
    ExportAssignment,
    Expression,
    FileReference,
    FunctionTypeNode,
    GeneratedIdentifier,
    GeneratedPrivateIdentifier,
    HasInitializer,
    HasJSDoc,
    Identifier,
    JSDocContainer,
    JSDocDeprecatedTag,
    JSDocEnumTag,
    JSDocLink,
    JSDocLinkCode,
    JSDocLinkPlain,
    JSDocOverrideTag,
    JSDocParameterTag,
    JSDocPrivateTag,
    JSDocPropertyLikeTag,
    JSDocProtectedTag,
    JSDocPublicTag,
    JSDocReadonlyTag,
    JSDocTag,
    JSDocTemplateTag,
    JSDocTypedefTag,
    JSDocTypeTag,
    LeftHandSideExpression,
    LiteralToken,
    MemberName,
    Modifier,
    ModifierFlags,
    NamedDeclaration,
    Node,
    NodeArray,
    NodeFlags,
    NonNullChain,
    ObjectLiteralElementLike,
    OuterExpressionKinds,
    ParameterDeclaration,
    PrivateIdentifier,
    PropertyAccessChain,
    PropertyAccessExpression,
    PropertyName,
    SignatureDeclaration,
    Statement,
    StringLiteralLike,
    SyntaxKind,
    TemplateLiteralToken,
    TextChangeRange,
    TextSpan,
    TypeParameterDeclaration,
    UnaryExpression,
    VariableDeclaration,
} from "./types";
import {
    canHaveJSDoc,
    getAssignmentDeclarationKind,
    getElementOrPropertyAccessArgumentExpressionOrName,
    getJSDocCommentsAndTags,
    isAccessExpression,
    isBindableStaticElementAccessExpression,
    isFunctionBlock,
    isInJSFile,
    modifierToFlag,
} from "./utilities";

export function textSpanEnd(span: TextSpan) {
    return span.start + span.length;
}

function textSpanIsEmpty(span: TextSpan) {
    return span.length === 0;
}

// Returns true if 'span' contains 'other'.

export function createTextSpan(start: number, length: number): TextSpan {
    if (start < 0) {
        throw new Error("start < 0");
    }
    if (length < 0) {
        throw new Error("length < 0");
    }

    return { start, length };
}

export function createTextSpanFromBounds(start: number, end: number) {
    return createTextSpan(start, end - start);
}

export function textChangeRangeNewSpan(range: TextChangeRange) {
    return createTextSpan(range.span.start, range.newLength);
}

export function textChangeRangeIsUnchanged(range: TextChangeRange) {
    return textSpanIsEmpty(range.span) && range.newLength === 0;
}

export function createTextChangeRange(span: TextSpan, newLength: number): TextChangeRange {
    if (newLength < 0) {
        throw new Error("newLength < 0");
    }

    return { span, newLength };
}

function isEmptyBindingPattern(node: BindingName): node is BindingPattern {
    if (isBindingPattern(node)) {
        return every(node.elements, isEmptyBindingElement);
    }
    return false;
}

// TODO(jakebailey): It is very weird that we have BindingElement and ArrayBindingElement;
// we should have ObjectBindingElement and ArrayBindingElement, which are both BindingElement,
// just like BindingPattern is a ObjectBindingPattern or a ArrayBindingPattern.
function isEmptyBindingElement(node: BindingElement | ArrayBindingElement): boolean {
    if (isOmittedExpression(node)) {
        return true;
    }
    return isEmptyBindingPattern(node.name);
}

// Returns the node flags for this node and all relevant parent nodes.  This is done so that
// nodes like variable declarations and binding elements can returned a view of their flags
// that includes the modifiers from their container.  i.e. flags like export/declare aren't
// stored on the variable declaration directly, but on the containing variable statement
// (if it has one).  Similarly, flags for let/const are stored on the variable declaration
// list.  By calling this function, all those flags are combined so that the client can treat
// the node as if it actually had those flags.

/**
 * Iterates through the parent chain of a node and performs the callback on each parent until the callback
 * returns a truthy value, then returns that value.
 * If no such value is found, it applies the callback until the parent pointer is undefined or the callback returns "quit"
 * At that point findAncestor returns undefined.
 */
export function findAncestor<T extends Node>(node: Node | undefined, callback: (element: Node) => element is T): T | undefined;
export function findAncestor(node: Node | undefined, callback: (element: Node) => boolean | "quit"): Node | undefined;
export function findAncestor(node: Node | undefined, callback: (element: Node) => boolean | "quit"): Node | undefined {
    while (node) {
        const result = callback(node);
        if (result === "quit") {
            return undefined;
        }
        else if (result) {
            return node;
        }
        node = node.parent;
    }
    return undefined;
}

/**
 * Gets a value indicating whether a node originated in the parse tree.
 *
 * @param node The node to test.
 */
export function isParseTreeNode(node: Node): boolean {
    return (node.flags & NodeFlags.Synthesized) === 0;
}

/**
 * Gets the original parse tree node for a node.
 *
 * @param node The original node.
 * @returns The original parse tree node if found; otherwise, undefined.
 */
export function getParseTreeNode(node: Node | undefined): Node | undefined;

/**
 * Gets the original parse tree node for a node.
 *
 * @param node The original node.
 * @param nodeTest A callback used to ensure the correct type of parse tree node is returned.
 * @returns The original parse tree node if found; otherwise, undefined.
 */
export function getParseTreeNode<T extends Node>(node: T | undefined, nodeTest?: (node: Node) => node is T): T | undefined;
export function getParseTreeNode(node: Node | undefined, nodeTest?: (node: Node) => boolean): Node | undefined {
    if (node === undefined || isParseTreeNode(node)) {
        return node;
    }

    node = node.original;
    while (node) {
        if (isParseTreeNode(node)) {
            return !nodeTest || nodeTest(node) ? node : undefined;
        }
        node = node.original;
    }
}

/** Add an extra underscore to identifiers that start with two underscores to avoid issues with magic names like '__proto__' */
export function escapeLeadingUnderscores(identifier: string): __String {
    return (identifier.length >= 2 && identifier.charCodeAt(0) === CharacterCodes._ && identifier.charCodeAt(1) === CharacterCodes._ ? "_" + identifier : identifier) as __String;
}

/**
 * Remove extra underscore from escaped identifier text content.
 *
 * @param identifier The escaped identifier text.
 * @returns The unescaped identifier text.
 */
export function unescapeLeadingUnderscores(identifier: __String): string {
    const id = identifier as string;
    return id.length >= 3 && id.charCodeAt(0) === CharacterCodes._ && id.charCodeAt(1) === CharacterCodes._ && id.charCodeAt(2) === CharacterCodes._ ? id.substr(1) : id;
}

export function idText(identifierOrPrivateName: Identifier | PrivateIdentifier): string {
    return unescapeLeadingUnderscores(identifierOrPrivateName.escapedText);
}

/**
 * A JSDocTypedef tag has an _optional_ name field - if a name is not directly present, we should
 * attempt to draw the name from the node the declaration is on (as that declaration is what its' symbol
 * will be merged with)
 */
function nameForNamelessJSDocTypedef(declaration: JSDocTypedefTag | JSDocEnumTag): Identifier | PrivateIdentifier | undefined {
    const hostNode = declaration.parent.parent;
    if (!hostNode) {
        return undefined;
    }
    // Covers classes, functions - any named declaration host node
    if (isDeclaration(hostNode)) {
        return getDeclarationIdentifier(hostNode);
    }
    // Covers remaining cases (returning undefined if none match).
    switch (hostNode.kind) {
        case SyntaxKind.VariableStatement:
            if (hostNode.declarationList && hostNode.declarationList.declarations[0]) {
                return getDeclarationIdentifier(hostNode.declarationList.declarations[0]);
            }
            break;
        case SyntaxKind.ExpressionStatement:
            let expr = hostNode.expression;
            if (expr.kind === SyntaxKind.BinaryExpression && (expr as BinaryExpression).operatorToken.kind === SyntaxKind.EqualsToken) {
                expr = (expr as BinaryExpression).left;
            }
            switch (expr.kind) {
                case SyntaxKind.PropertyAccessExpression:
                    return (expr as PropertyAccessExpression).name;
                case SyntaxKind.ElementAccessExpression:
                    const arg = (expr as ElementAccessExpression).argumentExpression;
                    if (isIdentifier(arg)) {
                        return arg;
                    }
            }
            break;
        case SyntaxKind.ParenthesizedExpression: {
            return getDeclarationIdentifier(hostNode.expression);
        }
        case SyntaxKind.LabeledStatement: {
            if (isDeclaration(hostNode.statement) || isExpression(hostNode.statement)) {
                return getDeclarationIdentifier(hostNode.statement);
            }
            break;
        }
    }
}

function getDeclarationIdentifier(node: Declaration | Expression): Identifier | undefined {
    const name = getNameOfDeclaration(node);
    return name && isIdentifier(name) ? name : undefined;
}

function getNameOfJSDocTypedef(declaration: JSDocTypedefTag): Identifier | PrivateIdentifier | undefined {
    return declaration.name || nameForNamelessJSDocTypedef(declaration);
}

/** @internal */
export function isNamedDeclaration(node: Node): node is NamedDeclaration & { name: DeclarationName; } {
    return !!(node as NamedDeclaration).name; // A 'name' property should always be a DeclarationName.
}

/** @internal */
export function getNonAssignedNameOfDeclaration(declaration: Declaration | Expression): DeclarationName | undefined {
    switch (declaration.kind) {
        case SyntaxKind.Identifier:
            return declaration as Identifier;
        case SyntaxKind.JSDocPropertyTag:
        case SyntaxKind.JSDocParameterTag: {
            const { name } = declaration as JSDocPropertyLikeTag;
            if (name.kind === SyntaxKind.QualifiedName) {
                return name.right;
            }
            break;
        }
        case SyntaxKind.CallExpression:
        case SyntaxKind.BinaryExpression: {
            const expr = declaration as BinaryExpression | CallExpression;
            switch (getAssignmentDeclarationKind(expr)) {
                case AssignmentDeclarationKind.ExportsProperty:
                case AssignmentDeclarationKind.ThisProperty:
                case AssignmentDeclarationKind.Property:
                case AssignmentDeclarationKind.PrototypeProperty:
                    return getElementOrPropertyAccessArgumentExpressionOrName((expr as BinaryExpression).left as AccessExpression);
                case AssignmentDeclarationKind.ObjectDefinePropertyValue:
                case AssignmentDeclarationKind.ObjectDefinePropertyExports:
                case AssignmentDeclarationKind.ObjectDefinePrototypeProperty:
                    return (expr as BindableObjectDefinePropertyCall).arguments[1];
                default:
                    return undefined;
            }
        }
        case SyntaxKind.JSDocTypedefTag:
            return getNameOfJSDocTypedef(declaration as JSDocTypedefTag);
        case SyntaxKind.JSDocEnumTag:
            return nameForNamelessJSDocTypedef(declaration as JSDocEnumTag);
        case SyntaxKind.ExportAssignment: {
            const { expression } = declaration as ExportAssignment;
            return isIdentifier(expression) ? expression : undefined;
        }
        case SyntaxKind.ElementAccessExpression:
            const expr = declaration as ElementAccessExpression;
            if (isBindableStaticElementAccessExpression(expr)) {
                return expr.argumentExpression;
            }
    }
    return (declaration as NamedDeclaration).name;
}

export function getNameOfDeclaration(declaration: Declaration | Expression | undefined): DeclarationName | undefined {
    if (declaration === undefined) return undefined;
    return getNonAssignedNameOfDeclaration(declaration) ||
        (isFunctionExpression(declaration) || isArrowFunction(declaration) || isClassExpression(declaration) ? getAssignedName(declaration) : undefined);
}

/** @internal */
function getAssignedName(node: Node): DeclarationName | undefined {
    if (!node.parent) {
        return undefined;
    }
    else if (isPropertyAssignment(node.parent) || isBindingElement(node.parent)) {
        return node.parent.name;
    }
    else if (isBinaryExpression(node.parent) && node === node.parent.right) {
        if (isIdentifier(node.parent.left)) {
            return node.parent.left;
        }
        else if (isAccessExpression(node.parent.left)) {
            return getElementOrPropertyAccessArgumentExpressionOrName(node.parent.left);
        }
    }
    else if (isVariableDeclaration(node.parent) && isIdentifier(node.parent.name)) {
        return node.parent.name;
    }
}

function getJSDocParameterTagsWorker(param: ParameterDeclaration, noCache?: boolean): readonly JSDocParameterTag[] {
    if (param.name) {
        if (isIdentifier(param.name)) {
            const name = param.name.escapedText;
            return getJSDocTagsWorker(param.parent, noCache).filter((tag): tag is JSDocParameterTag => isJSDocParameterTag(tag) && isIdentifier(tag.name) && tag.name.escapedText === name);
        }
        else {
            const i = param.parent.parameters.indexOf(param);
            Debug.assert(i > -1, "Parameters should always be in their parents' parameter list");
            const paramTags = getJSDocTagsWorker(param.parent, noCache).filter(isJSDocParameterTag);
            if (i < paramTags.length) {
                return [paramTags[i]];
            }
        }
    }
    // return empty array for: out-of-order binding patterns and JSDoc function syntax, which has un-named parameters
    return emptyArray;
}

/**
 * Gets the JSDoc parameter tags for the node if present.
 *
 * @remarks Returns any JSDoc param tag whose name matches the provided
 * parameter, whether a param tag on a containing function
 * expression, or a param tag on a variable declaration whose
 * initializer is the containing function. The tags closest to the
 * node are returned first, so in the previous example, the param
 * tag on the containing function expression would be first.
 *
 * For binding patterns, parameter tags are matched by position.
 */
export function getJSDocParameterTags(param: ParameterDeclaration): readonly JSDocParameterTag[] {
    return getJSDocParameterTagsWorker(param, /*noCache*/ false);
}

/** @internal */
export function getJSDocParameterTagsNoCache(param: ParameterDeclaration): readonly JSDocParameterTag[] {
    return getJSDocParameterTagsWorker(param, /*noCache*/ true);
}

function getJSDocTypeParameterTagsWorker(param: TypeParameterDeclaration, noCache?: boolean): readonly JSDocTemplateTag[] {
    const name = param.name.escapedText;
    return getJSDocTagsWorker(param.parent, noCache).filter((tag): tag is JSDocTemplateTag => isJSDocTemplateTag(tag) && tag.typeParameters.some(tp => tp.name.escapedText === name));
}

/**
 * Gets the JSDoc type parameter tags for the node if present.
 *
 * @remarks Returns any JSDoc template tag whose names match the provided
 * parameter, whether a template tag on a containing function
 * expression, or a template tag on a variable declaration whose
 * initializer is the containing function. The tags closest to the
 * node are returned first, so in the previous example, the template
 * tag on the containing function expression would be first.
 */
export function getJSDocTypeParameterTags(param: TypeParameterDeclaration): readonly JSDocTemplateTag[] {
    return getJSDocTypeParameterTagsWorker(param, /*noCache*/ false);
}

/** @internal */
export function getJSDocTypeParameterTagsNoCache(param: TypeParameterDeclaration): readonly JSDocTemplateTag[] {
    return getJSDocTypeParameterTagsWorker(param, /*noCache*/ true);
}

/** @internal */
export function getJSDocPublicTagNoCache(node: Node): JSDocPublicTag | undefined {
    return getFirstJSDocTag(node, isJSDocPublicTag, /*noCache*/ true);
}

/** @internal */
export function getJSDocPrivateTagNoCache(node: Node): JSDocPrivateTag | undefined {
    return getFirstJSDocTag(node, isJSDocPrivateTag, /*noCache*/ true);
}

/** @internal */
export function getJSDocProtectedTagNoCache(node: Node): JSDocProtectedTag | undefined {
    return getFirstJSDocTag(node, isJSDocProtectedTag, /*noCache*/ true);
}

/** @internal */
export function getJSDocReadonlyTagNoCache(node: Node): JSDocReadonlyTag | undefined {
    return getFirstJSDocTag(node, isJSDocReadonlyTag, /*noCache*/ true);
}

export function getJSDocOverrideTagNoCache(node: Node): JSDocOverrideTag | undefined {
    return getFirstJSDocTag(node, isJSDocOverrideTag, /*noCache*/ true);
}

/** @internal */
export function getJSDocDeprecatedTagNoCache(node: Node): JSDocDeprecatedTag | undefined {
    return getFirstJSDocTag(node, isJSDocDeprecatedTag, /*noCache*/ true);
}

/** Gets the JSDoc type tag for the node if present and valid */
export function getJSDocTypeTag(node: Node): JSDocTypeTag | undefined {
    // We should have already issued an error if there were multiple type jsdocs, so just use the first one.
    const tag = getFirstJSDocTag(node, isJSDocTypeTag);
    if (tag && tag.typeExpression && tag.typeExpression.type) {
        return tag;
    }
    return undefined;
}

function getJSDocTagsWorker(node: Node, noCache?: boolean): readonly JSDocTag[] {
    if (!canHaveJSDoc(node)) return emptyArray;
    let tags = node.jsDoc?.jsDocCache;
    // If cache is 'null', that means we did the work of searching for JSDoc tags and came up with nothing.
    if (tags === undefined || noCache) {
        const comments = getJSDocCommentsAndTags(node, noCache);
        Debug.assert(comments.length < 2 || comments[0] !== comments[1]);
        tags = flatMap(comments, j => isJSDoc(j) ? j.tags : j);
        if (!noCache) {
            node.jsDoc ??= [];
            node.jsDoc.jsDocCache = tags;
        }
    }
    return tags;
}

/** Get the first JSDoc tag of a specified kind, or undefined if not present. */
function getFirstJSDocTag<T extends JSDocTag>(node: Node, predicate: (tag: JSDocTag) => tag is T, noCache?: boolean): T | undefined {
    return find(getJSDocTagsWorker(node, noCache), predicate);
}

// #region

export function isMemberName(node: Node): node is MemberName {
    return node.kind === SyntaxKind.Identifier || node.kind === SyntaxKind.PrivateIdentifier;
}

export function isPropertyAccessChain(node: Node): node is PropertyAccessChain {
    return isPropertyAccessExpression(node) && !!(node.flags & NodeFlags.OptionalChain);
}

export function isElementAccessChain(node: Node): node is ElementAccessChain {
    return isElementAccessExpression(node) && !!(node.flags & NodeFlags.OptionalChain);
}

export function isCallChain(node: Node): node is CallChain {
    return isCallExpression(node) && !!(node.flags & NodeFlags.OptionalChain);
}

export function isOptionalChain(node: Node): node is PropertyAccessChain | ElementAccessChain | CallChain | NonNullChain {
    const kind = node.kind;
    return !!(node.flags & NodeFlags.OptionalChain) &&
        (kind === SyntaxKind.PropertyAccessExpression
            || kind === SyntaxKind.ElementAccessExpression
            || kind === SyntaxKind.CallExpression
            || kind === SyntaxKind.NonNullExpression);
}

export function skipPartiallyEmittedExpressions(node: Expression): Expression;
export function skipPartiallyEmittedExpressions(node: Node): Node;
export function skipPartiallyEmittedExpressions(node: Node) {
    return skipOuterExpressions(node, OuterExpressionKinds.PartiallyEmittedExpressions);
}

export function isNonNullChain(node: Node): node is NonNullChain {
    return isNonNullExpression(node) && !!(node.flags & NodeFlags.OptionalChain);
}

// #endregion

// #region
// Node tests
//
// All node tests in the following list should *not* reference parent pointers so that
// they may be used with transformations.

/** @internal */
export function isNodeKind(kind: SyntaxKind) {
    return kind >= SyntaxKind.FirstNode;
}

// Node Arrays

/** @internal */
export function isNodeArray<T extends Node>(array: readonly T[]): array is NodeArray<T> {
    return hasProperty(array, "pos") && hasProperty(array, "end");
}

// Literals

/** @internal */
export function isLiteralKind(kind: SyntaxKind): kind is LiteralToken["kind"] {
    return SyntaxKind.FirstLiteralToken <= kind && kind <= SyntaxKind.LastLiteralToken;
}

// Pseudo-literals

/** @internal */
export function isTemplateLiteralKind(kind: SyntaxKind): kind is TemplateLiteralToken["kind"] {
    return SyntaxKind.FirstTemplateToken <= kind && kind <= SyntaxKind.LastTemplateToken;
}

// Identifiers

/** @internal */
export function isGeneratedIdentifier(node: Node): node is GeneratedIdentifier {
    return isIdentifier(node) && node.emitNode?.autoGenerate !== undefined;
}

/** @internal */
export function isGeneratedPrivateIdentifier(node: Node): node is GeneratedPrivateIdentifier {
    return isPrivateIdentifier(node) && node.emitNode?.autoGenerate !== undefined;
}

// Private Identifiers

// Keywords

/** @internal */
export function isModifierKind(token: SyntaxKind): token is Modifier["kind"] {
    switch (token) {
        case SyntaxKind.AbstractKeyword:
        case SyntaxKind.AccessorKeyword:
        case SyntaxKind.AsyncKeyword:
        case SyntaxKind.ConstKeyword:
        case SyntaxKind.DeclareKeyword:
        case SyntaxKind.DefaultKeyword:
        case SyntaxKind.ExportKeyword:
        case SyntaxKind.InKeyword:
        case SyntaxKind.PublicKeyword:
        case SyntaxKind.PrivateKeyword:
        case SyntaxKind.ProtectedKeyword:
        case SyntaxKind.ReadonlyKeyword:
        case SyntaxKind.StaticKeyword:
        case SyntaxKind.OutKeyword:
        case SyntaxKind.OverrideKeyword:
            return true;
    }
    return false;
}

/** @internal */
function isParameterPropertyModifier(kind: SyntaxKind): boolean {
    return !!(modifierToFlag(kind) & ModifierFlags.ParameterPropertyModifier);
}

/** @internal */
export function isClassMemberModifier(idToken: SyntaxKind): boolean {
    return isParameterPropertyModifier(idToken) ||
        idToken === SyntaxKind.StaticKeyword ||
        idToken === SyntaxKind.OverrideKeyword ||
        idToken === SyntaxKind.AccessorKeyword;
}

export function isPropertyName(node: Node): node is PropertyName {
    const kind = node.kind;
    return kind === SyntaxKind.Identifier
        || kind === SyntaxKind.PrivateIdentifier
        || kind === SyntaxKind.StringLiteral
        || kind === SyntaxKind.NumericLiteral
        || kind === SyntaxKind.ComputedPropertyName;
}

// Functions

export function isFunctionLike(node: Node | undefined): node is SignatureDeclaration {
    return !!node && isFunctionLikeKind(node.kind);
}

function isFunctionLikeDeclarationKind(kind: SyntaxKind): boolean {
    switch (kind) {
        case SyntaxKind.FunctionDeclaration:
        case SyntaxKind.MethodDeclaration:
        case SyntaxKind.Constructor:
        case SyntaxKind.GetAccessor:
        case SyntaxKind.SetAccessor:
        case SyntaxKind.FunctionExpression:
        case SyntaxKind.ArrowFunction:
            return true;
        default:
            return false;
    }
}

/** @internal */
function isFunctionLikeKind(kind: SyntaxKind): boolean {
    switch (kind) {
        case SyntaxKind.MethodSignature:
        case SyntaxKind.CallSignature:
        case SyntaxKind.JSDocSignature:
        case SyntaxKind.ConstructSignature:
        case SyntaxKind.IndexSignature:
        case SyntaxKind.FunctionType:
        case SyntaxKind.JSDocFunctionType:
        case SyntaxKind.ConstructorType:
            return true;
        default:
            return isFunctionLikeDeclarationKind(kind);
    }
}

// Classes

export function isClassLike(node: Node): node is ClassLikeDeclaration {
    return node && (node.kind === SyntaxKind.ClassDeclaration || node.kind === SyntaxKind.ClassExpression);
}

// Type members

export function isObjectLiteralElementLike(node: Node): node is ObjectLiteralElementLike {
    const kind = node.kind;
    return kind === SyntaxKind.PropertyAssignment
        || kind === SyntaxKind.ShorthandPropertyAssignment
        || kind === SyntaxKind.SpreadAssignment
        || kind === SyntaxKind.MethodDeclaration
        || kind === SyntaxKind.GetAccessor
        || kind === SyntaxKind.SetAccessor;
}

// Type

export function isFunctionOrConstructorTypeNode(node: Node): node is FunctionTypeNode | ConstructorTypeNode {
    switch (node.kind) {
        case SyntaxKind.FunctionType:
        case SyntaxKind.ConstructorType:
            return true;
    }

    return false;
}

// Binding patterns

/** @internal */
function isBindingPattern(node: Node | undefined): node is BindingPattern {
    if (node) {
        const kind = node.kind;
        return kind === SyntaxKind.ArrayBindingPattern
            || kind === SyntaxKind.ObjectBindingPattern;
    }

    return false;
}

/** @internal */
export function isAssignmentPattern(node: Node): node is AssignmentPattern {
    const kind = node.kind;
    return kind === SyntaxKind.ArrayLiteralExpression
        || kind === SyntaxKind.ObjectLiteralExpression;
}

/**
 * Determines whether the BindingOrAssignmentElement is a BindingElement-like declaration
 *
 * @internal
 */
export function isDeclarationBindingElement(bindingElement: BindingOrAssignmentElement): bindingElement is VariableDeclaration | ParameterDeclaration | BindingElement {
    switch (bindingElement.kind) {
        case SyntaxKind.VariableDeclaration:
        case SyntaxKind.Parameter:
        case SyntaxKind.BindingElement:
            return true;
    }

    return false;
}

// Expression

export function isLeftHandSideExpression(node: Node): node is LeftHandSideExpression {
    return isLeftHandSideExpressionKind(skipPartiallyEmittedExpressions(node).kind);
}

function isLeftHandSideExpressionKind(kind: SyntaxKind): boolean {
    switch (kind) {
        case SyntaxKind.PropertyAccessExpression:
        case SyntaxKind.ElementAccessExpression:
        case SyntaxKind.NewExpression:
        case SyntaxKind.CallExpression:
        case SyntaxKind.JsxElement:
        case SyntaxKind.JsxSelfClosingElement:
        case SyntaxKind.JsxFragment:
        case SyntaxKind.TaggedTemplateExpression:
        case SyntaxKind.ArrayLiteralExpression:
        case SyntaxKind.ParenthesizedExpression:
        case SyntaxKind.ObjectLiteralExpression:
        case SyntaxKind.ClassExpression:
        case SyntaxKind.FunctionExpression:
        case SyntaxKind.Identifier:
        case SyntaxKind.PrivateIdentifier: // technically this is only an Expression if it's in a `#field in expr` BinaryExpression
        case SyntaxKind.RegularExpressionLiteral:
        case SyntaxKind.NumericLiteral:
        case SyntaxKind.BigIntLiteral:
        case SyntaxKind.StringLiteral:
        case SyntaxKind.NoSubstitutionTemplateLiteral:
        case SyntaxKind.TemplateExpression:
        case SyntaxKind.FalseKeyword:
        case SyntaxKind.NullKeyword:
        case SyntaxKind.ThisKeyword:
        case SyntaxKind.TrueKeyword:
        case SyntaxKind.SuperKeyword:
        case SyntaxKind.NonNullExpression:
        case SyntaxKind.ExpressionWithTypeArguments:
        case SyntaxKind.MetaProperty:
        case SyntaxKind.ImportKeyword: // technically this is only an Expression if it's in a CallExpression
        case SyntaxKind.MissingDeclaration:
            return true;
        default:
            return false;
    }
}

/** @internal */
export function isUnaryExpression(node: Node): node is UnaryExpression {
    return isUnaryExpressionKind(skipPartiallyEmittedExpressions(node).kind);
}

function isUnaryExpressionKind(kind: SyntaxKind): boolean {
    switch (kind) {
        case SyntaxKind.PrefixUnaryExpression:
        case SyntaxKind.PostfixUnaryExpression:
        case SyntaxKind.DeleteExpression:
        case SyntaxKind.TypeOfExpression:
        case SyntaxKind.VoidExpression:
        case SyntaxKind.AwaitExpression:
        case SyntaxKind.TypeAssertionExpression:
            return true;
        default:
            return isLeftHandSideExpressionKind(kind);
    }
}

/**
 * Determines whether a node is an expression based only on its kind.
 */
function isExpression(node: Node): node is Expression {
    return isExpressionKind(skipPartiallyEmittedExpressions(node).kind);
}

function isExpressionKind(kind: SyntaxKind): boolean {
    switch (kind) {
        case SyntaxKind.ConditionalExpression:
        case SyntaxKind.YieldExpression:
        case SyntaxKind.ArrowFunction:
        case SyntaxKind.BinaryExpression:
        case SyntaxKind.SpreadElement:
        case SyntaxKind.AsExpression:
        case SyntaxKind.OmittedExpression:
        case SyntaxKind.CommaListExpression:
        case SyntaxKind.PartiallyEmittedExpression:
        case SyntaxKind.SatisfiesExpression:
            return true;
        default:
            return isUnaryExpressionKind(kind);
    }
}

// Statement

// Element

function isDeclarationKind(kind: SyntaxKind) {
    return kind === SyntaxKind.ArrowFunction
        || kind === SyntaxKind.BindingElement
        || kind === SyntaxKind.ClassDeclaration
        || kind === SyntaxKind.ClassExpression
        || kind === SyntaxKind.ClassStaticBlockDeclaration
        || kind === SyntaxKind.Constructor
        || kind === SyntaxKind.EnumDeclaration
        || kind === SyntaxKind.EnumMember
        || kind === SyntaxKind.ExportSpecifier
        || kind === SyntaxKind.FunctionDeclaration
        || kind === SyntaxKind.FunctionExpression
        || kind === SyntaxKind.GetAccessor
        || kind === SyntaxKind.ImportClause
        || kind === SyntaxKind.ImportEqualsDeclaration
        || kind === SyntaxKind.ImportSpecifier
        || kind === SyntaxKind.InterfaceDeclaration
        || kind === SyntaxKind.JsxAttribute
        || kind === SyntaxKind.MethodDeclaration
        || kind === SyntaxKind.MethodSignature
        || kind === SyntaxKind.ModuleDeclaration
        || kind === SyntaxKind.NamespaceExportDeclaration
        || kind === SyntaxKind.NamespaceImport
        || kind === SyntaxKind.NamespaceExport
        || kind === SyntaxKind.Parameter
        || kind === SyntaxKind.PropertyAssignment
        || kind === SyntaxKind.PropertyDeclaration
        || kind === SyntaxKind.PropertySignature
        || kind === SyntaxKind.SetAccessor
        || kind === SyntaxKind.ShorthandPropertyAssignment
        || kind === SyntaxKind.TypeAliasDeclaration
        || kind === SyntaxKind.TypeParameter
        || kind === SyntaxKind.VariableDeclaration
        || kind === SyntaxKind.JSDocTypedefTag
        || kind === SyntaxKind.JSDocCallbackTag
        || kind === SyntaxKind.JSDocPropertyTag;
}

function isDeclarationStatementKind(kind: SyntaxKind) {
    return kind === SyntaxKind.FunctionDeclaration
        || kind === SyntaxKind.MissingDeclaration
        || kind === SyntaxKind.ClassDeclaration
        || kind === SyntaxKind.InterfaceDeclaration
        || kind === SyntaxKind.TypeAliasDeclaration
        || kind === SyntaxKind.EnumDeclaration
        || kind === SyntaxKind.ModuleDeclaration
        || kind === SyntaxKind.ImportDeclaration
        || kind === SyntaxKind.ImportEqualsDeclaration
        || kind === SyntaxKind.ExportDeclaration
        || kind === SyntaxKind.ExportAssignment
        || kind === SyntaxKind.NamespaceExportDeclaration;
}

function isStatementKindButNotDeclarationKind(kind: SyntaxKind) {
    return kind === SyntaxKind.BreakStatement
        || kind === SyntaxKind.ContinueStatement
        || kind === SyntaxKind.DebuggerStatement
        || kind === SyntaxKind.DoStatement
        || kind === SyntaxKind.ExpressionStatement
        || kind === SyntaxKind.EmptyStatement
        || kind === SyntaxKind.ForInStatement
        || kind === SyntaxKind.ForOfStatement
        || kind === SyntaxKind.ForStatement
        || kind === SyntaxKind.IfStatement
        || kind === SyntaxKind.LabeledStatement
        || kind === SyntaxKind.ReturnStatement
        || kind === SyntaxKind.SwitchStatement
        || kind === SyntaxKind.ThrowStatement
        || kind === SyntaxKind.TryStatement
        || kind === SyntaxKind.VariableStatement
        || kind === SyntaxKind.WhileStatement
        || kind === SyntaxKind.WithStatement
        || kind === SyntaxKind.NotEmittedStatement;
}

/** @internal */
function isDeclaration(node: Node): node is NamedDeclaration {
    if (node.kind === SyntaxKind.TypeParameter) {
        return (node.parent && node.parent.kind !== SyntaxKind.JSDocTemplateTag) || isInJSFile(node);
    }

    return isDeclarationKind(node.kind);
}

export function isStatement(node: Node): node is Statement {
    const kind = node.kind;
    return isStatementKindButNotDeclarationKind(kind)
        || isDeclarationStatementKind(kind)
        || isBlockStatement(node);
}

function isBlockStatement(node: Node): node is Block {
    if (node.kind !== SyntaxKind.Block) return false;
    if (node.parent !== undefined) {
        if (node.parent.kind === SyntaxKind.TryStatement || node.parent.kind === SyntaxKind.CatchClause) {
            return false;
        }
    }
    return !isFunctionBlock(node);
}

// TODO(jakebailey): should we be exporting this function and not isStatement?
/**
 * NOTE: This is similar to `isStatement` but does not access parent pointers.
 *
 * @internal
 */
export function isStatementOrBlock(node: Node): node is Statement | Block {
    const kind = node.kind;
    return isStatementKindButNotDeclarationKind(kind)
        || isDeclarationStatementKind(kind)
        || kind === SyntaxKind.Block;
}

// Module references

// JSX

// Clauses

// JSDoc

/**
 * True if node is of some JSDoc syntax kind.
 *
 * @internal
 */
export function isJSDocNode(node: Node): boolean {
    return node.kind >= SyntaxKind.FirstJSDocNode && node.kind <= SyntaxKind.LastJSDocNode;
}

// TODO: determine what this does before making it public.

/**
 * True if has jsdoc nodes attached to it.
 *
 * @internal
 */
// TODO: GH#19856 Would like to return `node is Node & { jsDoc: JSDoc[] }` but it causes long compile times
export function hasJSDocNodes(node: Node): node is HasJSDoc {
    if (!canHaveJSDoc(node)) return false;

    const { jsDoc } = node as JSDocContainer;
    return !!jsDoc && jsDoc.length > 0;
}

/**
 * True if has initializer node attached to it.
 *
 * @internal
 */
export function hasInitializer(node: Node): node is HasInitializer {
    return !!(node as HasInitializer).initializer;
}

export function isStringLiteralLike(node: Node | FileReference): node is StringLiteralLike {
    return (node as Node).kind === SyntaxKind.StringLiteral || (node as Node).kind === SyntaxKind.NoSubstitutionTemplateLiteral;
}

export function isJSDocLinkLike(node: Node): node is JSDocLink | JSDocLinkCode | JSDocLinkPlain {
    return node.kind === SyntaxKind.JSDocLink || node.kind === SyntaxKind.JSDocLinkCode || node.kind === SyntaxKind.JSDocLinkPlain;
}
