import {
    Debug,
} from "../debug";
import {
    AssignmentPattern,
    BinaryExpression,
    BindingOrAssignmentElement,
    BindingOrAssignmentElementTarget,
    BindingOrAssignmentPattern,
    CharacterCodes,
    CommaListExpression,
    EmitFlags,
    Expression,
    ExpressionStatement,
    GeneratedIdentifier,
    GeneratedNamePart,
    GeneratedPrivateIdentifier,
    Identifier,
    JSDocNamespaceBody,
    JSDocTypeAssertion,
    Node,
    OuterExpression,
    OuterExpressionKinds,
    PrivateIdentifier,
    Statement,
    SyntaxKind,
    Token,
    TransformFlags,
    WrappedExpression,
} from "../types";
import {
    getEmitFlags,
    isAssignmentExpression,
    isInJSFile,
    isPrologueDirective,
} from "../utilities";
import {
    getJSDocTypeTag,
    idText,
    isAssignmentPattern,
    isDeclarationBindingElement,
    isGeneratedIdentifier,
    isGeneratedPrivateIdentifier,
    isObjectLiteralElementLike,
} from "../utilitiesPublic";
import {
    setStartsOnNewLine,
} from "./emitNode";
import {
    isCommaListExpression,
    isIdentifier,
    isParenthesizedExpression,
    isPrivateIdentifier,
    isSpreadElement,
    isStringLiteral,
} from "./nodeTests";

// Compound nodes

// Utilities

/**
 * Gets whether an identifier should only be referred to by its local name.
 *
 * @internal
 */
export function isLocalName(node: Identifier) {
    return (getEmitFlags(node) & EmitFlags.LocalName) !== 0;
}

function isUseStrictPrologue(node: ExpressionStatement): boolean {
    return isStringLiteral(node.expression) && node.expression.text === "use strict";
}

/** @internal */
export function findUseStrictPrologue(statements: readonly Statement[]): Statement | undefined {
    for (const statement of statements) {
        if (isPrologueDirective(statement)) {
            if (isUseStrictPrologue(statement)) {
                return statement;
            }
        }
        else {
            break;
        }
    }
    return undefined;
}

/** @internal */
function isCommaExpression(node: Expression): node is BinaryExpression & { operatorToken: Token<SyntaxKind.CommaToken>; } {
    return node.kind === SyntaxKind.BinaryExpression && (node as BinaryExpression).operatorToken.kind === SyntaxKind.CommaToken;
}

/** @internal */
export function isCommaSequence(node: Expression): node is BinaryExpression & { operatorToken: Token<SyntaxKind.CommaToken>; } | CommaListExpression {
    return isCommaExpression(node) || isCommaListExpression(node);
}

/** @internal */
function isJSDocTypeAssertion(node: Node): node is JSDocTypeAssertion {
    return isParenthesizedExpression(node)
        && isInJSFile(node)
        && !!getJSDocTypeTag(node);
}

/** @internal */
export function isOuterExpression(node: Node, kinds = OuterExpressionKinds.All): node is OuterExpression {
    switch (node.kind) {
        case SyntaxKind.ParenthesizedExpression:
            if (kinds & OuterExpressionKinds.ExcludeJSDocTypeAssertion && isJSDocTypeAssertion(node)) {
                return false;
            }
            return (kinds & OuterExpressionKinds.Parentheses) !== 0;
        case SyntaxKind.TypeAssertionExpression:
        case SyntaxKind.AsExpression:
        case SyntaxKind.ExpressionWithTypeArguments:
        case SyntaxKind.SatisfiesExpression:
            return (kinds & OuterExpressionKinds.TypeAssertions) !== 0;
        case SyntaxKind.NonNullExpression:
            return (kinds & OuterExpressionKinds.NonNullAssertions) !== 0;
        case SyntaxKind.PartiallyEmittedExpression:
            return (kinds & OuterExpressionKinds.PartiallyEmittedExpressions) !== 0;
    }
    return false;
}

/** @internal */
export function skipOuterExpressions<T extends Expression>(node: WrappedExpression<T>): T;
/** @internal */
export function skipOuterExpressions(node: Expression, kinds?: OuterExpressionKinds): Expression;
/** @internal */
export function skipOuterExpressions(node: Node, kinds?: OuterExpressionKinds): Node;
/** @internal */
export function skipOuterExpressions(node: Node, kinds = OuterExpressionKinds.All) {
    while (isOuterExpression(node, kinds)) {
        node = node.expression;
    }
    return node;
}

/** @internal */
export function startOnNewLine<T extends Node>(node: T): T {
    return setStartsOnNewLine(node, /*newLine*/ true);
}

/**
 * Gets the name of an BindingOrAssignmentElement.
 *
 * @internal
 */
function getTargetOfBindingOrAssignmentElement(bindingElement: BindingOrAssignmentElement): BindingOrAssignmentElementTarget | undefined {
    if (isDeclarationBindingElement(bindingElement)) {
        // `a` in `let { a } = ...`
        // `a` in `let { a = 1 } = ...`
        // `b` in `let { a: b } = ...`
        // `b` in `let { a: b = 1 } = ...`
        // `a` in `let { ...a } = ...`
        // `{b}` in `let { a: {b} } = ...`
        // `{b}` in `let { a: {b} = 1 } = ...`
        // `[b]` in `let { a: [b] } = ...`
        // `[b]` in `let { a: [b] = 1 } = ...`
        // `a` in `let [a] = ...`
        // `a` in `let [a = 1] = ...`
        // `a` in `let [...a] = ...`
        // `{a}` in `let [{a}] = ...`
        // `{a}` in `let [{a} = 1] = ...`
        // `[a]` in `let [[a]] = ...`
        // `[a]` in `let [[a] = 1] = ...`
        return bindingElement.name;
    }

    if (isObjectLiteralElementLike(bindingElement)) {
        switch (bindingElement.kind) {
            case SyntaxKind.PropertyAssignment:
                // `b` in `({ a: b } = ...)`
                // `b` in `({ a: b = 1 } = ...)`
                // `{b}` in `({ a: {b} } = ...)`
                // `{b}` in `({ a: {b} = 1 } = ...)`
                // `[b]` in `({ a: [b] } = ...)`
                // `[b]` in `({ a: [b] = 1 } = ...)`
                // `b.c` in `({ a: b.c } = ...)`
                // `b.c` in `({ a: b.c = 1 } = ...)`
                // `b[0]` in `({ a: b[0] } = ...)`
                // `b[0]` in `({ a: b[0] = 1 } = ...)`
                return getTargetOfBindingOrAssignmentElement(bindingElement.initializer as BindingOrAssignmentElement);

            case SyntaxKind.ShorthandPropertyAssignment:
                // `a` in `({ a } = ...)`
                // `a` in `({ a = 1 } = ...)`
                return bindingElement.name;

            case SyntaxKind.SpreadAssignment:
                // `a` in `({ ...a } = ...)`
                return getTargetOfBindingOrAssignmentElement(bindingElement.expression as BindingOrAssignmentElement);
        }

        // no target
        return undefined;
    }

    if (isAssignmentExpression(bindingElement, /*excludeCompoundAssignment*/ true)) {
        // `a` in `[a = 1] = ...`
        // `{a}` in `[{a} = 1] = ...`
        // `[a]` in `[[a] = 1] = ...`
        // `a.b` in `[a.b = 1] = ...`
        // `a[0]` in `[a[0] = 1] = ...`
        return getTargetOfBindingOrAssignmentElement(bindingElement.left as BindingOrAssignmentElement);
    }

    if (isSpreadElement(bindingElement)) {
        // `a` in `[...a] = ...`
        return getTargetOfBindingOrAssignmentElement(bindingElement.expression as BindingOrAssignmentElement);
    }

    // `a` in `[a] = ...`
    // `{a}` in `[{a}] = ...`
    // `[a]` in `[[a]] = ...`
    // `a.b` in `[a.b] = ...`
    // `a[0]` in `[a[0]] = ...`
    return bindingElement;
}

/**
 * Gets the elements of a BindingOrAssignmentPattern
 *
 * @internal
 */
function getElementsOfBindingOrAssignmentPattern(name: BindingOrAssignmentPattern): readonly BindingOrAssignmentElement[] {
    switch (name.kind) {
        case SyntaxKind.ObjectBindingPattern:
        case SyntaxKind.ArrayBindingPattern:
        case SyntaxKind.ArrayLiteralExpression:
            // `a` in `{a}`
            // `a` in `[a]`
            return name.elements as readonly BindingOrAssignmentElement[];

        case SyntaxKind.ObjectLiteralExpression:
            // `a` in `{a}`
            return name.properties as readonly BindingOrAssignmentElement[];
    }
}

/** @internal */
export function getJSDocTypeAliasName(fullName: JSDocNamespaceBody | undefined) {
    if (fullName) {
        let rightNode = fullName;
        while (true) {
            if (isIdentifier(rightNode) || !rightNode.body) {
                return isIdentifier(rightNode) ? rightNode : rightNode.name;
            }
            rightNode = rightNode.body;
        }
    }
}

// NOTE: The version in utilities includes ExclamationToken, which is not a binary operator.

/**
 * Formats a prefix or suffix of a generated name.
 *
 * @internal
 */
function formatGeneratedNamePart(part: string | undefined): string;
/**
 * Formats a prefix or suffix of a generated name. If the part is a {@link GeneratedNamePart}, calls {@link generateName} to format the source node.
 *
 * @internal
 */
function formatGeneratedNamePart(part: string | GeneratedNamePart | undefined, generateName: (name: GeneratedIdentifier | GeneratedPrivateIdentifier) => string): string;
/** @internal */
function formatGeneratedNamePart(part: string | GeneratedNamePart | undefined, generateName?: (name: GeneratedIdentifier | GeneratedPrivateIdentifier) => string): string {
    return typeof part === "object" ? formatGeneratedName(/*privateName*/ false, part.prefix, part.node, part.suffix, generateName!) :
        typeof part === "string" ? part.length > 0 && part.charCodeAt(0) === CharacterCodes.hash ? part.slice(1) : part :
        "";
}

function formatIdentifier(name: string | Identifier | PrivateIdentifier, generateName?: (name: GeneratedIdentifier | GeneratedPrivateIdentifier) => string) {
    return typeof name === "string" ? name :
        formatIdentifierWorker(name, Debug.checkDefined(generateName));
}

function formatIdentifierWorker(node: Identifier | PrivateIdentifier, generateName: (name: GeneratedIdentifier | GeneratedPrivateIdentifier) => string) {
    return isGeneratedPrivateIdentifier(node) ? generateName(node).slice(1) :
        isGeneratedIdentifier(node) ? generateName(node) :
        isPrivateIdentifier(node) ? (node.escapedText as string).slice(1) :
        idText(node);
}

/**
 * Formats a generated name.
 * @param privateName When `true`, inserts a `#` character at the start of the result.
 * @param prefix The prefix (if any) to include before the base name.
 * @param baseName The base name for the generated name.
 * @param suffix The suffix (if any) to include after the base name.
 *
 * @internal
 */
export function formatGeneratedName(privateName: boolean, prefix: string | undefined, baseName: string, suffix: string | undefined): string;
/**
 * Formats a generated name.
 * @param privateName When `true`, inserts a `#` character at the start of the result.
 * @param prefix The prefix (if any) to include before the base name.
 * @param baseName The base name for the generated name.
 * @param suffix The suffix (if any) to include after the base name.
 * @param generateName Called to format the source node of {@link prefix} when it is a {@link GeneratedNamePart}.
 *
 * @internal
 */
export function formatGeneratedName(privateName: boolean, prefix: string | GeneratedNamePart | undefined, baseName: string | Identifier | PrivateIdentifier, suffix: string | GeneratedNamePart | undefined, generateName: (name: GeneratedIdentifier | GeneratedPrivateIdentifier) => string): string;
/** @internal */
export function formatGeneratedName(privateName: boolean, prefix: string | GeneratedNamePart | undefined, baseName: string | Identifier | PrivateIdentifier, suffix: string | GeneratedNamePart | undefined, generateName?: (name: GeneratedIdentifier | GeneratedPrivateIdentifier) => string) {
    prefix = formatGeneratedNamePart(prefix, generateName!);
    suffix = formatGeneratedNamePart(suffix, generateName!);
    baseName = formatIdentifier(baseName, generateName);
    return `${privateName ? "#" : ""}${prefix}${baseName}${suffix}`;
}

/**
 * Walk an AssignmentPattern to determine if it contains object rest (`...`) syntax. We cannot rely on
 * propagation of `TransformFlags.ContainsObjectRestOrSpread` since it isn't propagated by default in
 * ObjectLiteralExpression and ArrayLiteralExpression since we do not know whether they belong to an
 * AssignmentPattern at the time the nodes are parsed.
 *
 * @internal
 */
export function containsObjectRestOrSpread(node: AssignmentPattern): boolean {
    if (node.transformFlags & TransformFlags.ContainsObjectRestOrSpread) return true;
    if (node.transformFlags & TransformFlags.ContainsES2018) {
        // check for nested spread assignments, otherwise '{ x: { a, ...b } = foo } = c'
        // will not be correctly interpreted by the ES2018 transformer
        for (const element of getElementsOfBindingOrAssignmentPattern(node)) {
            const target = getTargetOfBindingOrAssignmentElement(element);
            if (target && isAssignmentPattern(target)) {
                if (target.transformFlags & TransformFlags.ContainsObjectRestOrSpread) {
                    return true;
                }
                if (target.transformFlags & TransformFlags.ContainsES2018) {
                    if (containsObjectRestOrSpread(target)) return true;
                }
            }
        }
    }
    return false;
}
