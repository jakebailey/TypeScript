import {
    Debug,
} from "../debug";
import {
    AutoGenerateInfo,
    EmitFlags,
    EmitNode,
    Identifier,
    InternalEmitFlags,
    Node,
    NodeArray,
    PrivateIdentifier,
    SourceMapRange,
    SyntaxKind,
    SynthesizedComment,
    TextRange,
    TypeNode,
    TypeParameterDeclaration,
} from "../types";
import {
    getSourceFileOfNode,
} from "../utilities";
import {
    getParseTreeNode,
    isParseTreeNode,
} from "../utilitiesPublic";

/**
 * Associates a node with the current transformation, initializing
 * various transient transformation properties.
 * @internal
 */
function getOrCreateEmitNode(node: Node): EmitNode {
    if (!node.emitNode) {
        if (isParseTreeNode(node)) {
            // To avoid holding onto transformation artifacts, we keep track of any
            // parse tree node we are annotating. This allows us to clean them up after
            // all transformations have completed.
            if (node.kind === SyntaxKind.SourceFile) {
                return node.emitNode = { annotatedNodes: [node] } as EmitNode;
            }

            const sourceFile = getSourceFileOfNode(getParseTreeNode(getSourceFileOfNode(node))) ?? Debug.fail("Could not determine parsed source file.");
            getOrCreateEmitNode(sourceFile).annotatedNodes!.push(node);
        }

        node.emitNode = {} as EmitNode;
    }
    else {
        Debug.assert(!(node.emitNode.internalFlags & InternalEmitFlags.Immutable), "Invalid attempt to mutate an immutable node.");
    }
    return node.emitNode;
}

/**
 * Sets flags that control emit behavior of a node.
 */
export function setEmitFlags<T extends Node>(node: T, emitFlags: EmitFlags) {
    getOrCreateEmitNode(node).flags = emitFlags;
    return node;
}

/**
 * Gets a custom text range to use when emitting source maps.
 */
export function getSourceMapRange(node: Node): SourceMapRange {
    return node.emitNode?.sourceMapRange ?? node;
}

/**
 * Sets a custom text range to use when emitting comments.
 *
 * @internal
 */
export function setStartsOnNewLine<T extends Node>(node: T, newLine: boolean) {
    getOrCreateEmitNode(node).startsOnNewLine = newLine;
    return node;
}

/**
 * Gets a custom text range to use when emitting comments.
 */
export function getCommentRange(node: Node): TextRange {
    return node.emitNode?.commentRange ?? node;
}

export function getSyntheticLeadingComments(node: Node): SynthesizedComment[] | undefined {
    return node.emitNode?.leadingComments;
}

export function getSyntheticTrailingComments(node: Node): SynthesizedComment[] | undefined {
    return node.emitNode?.trailingComments;
}

/** @internal */
export function setIdentifierTypeArguments<T extends Identifier>(node: T, typeArguments: NodeArray<TypeNode | TypeParameterDeclaration> | undefined) {
    getOrCreateEmitNode(node).identifierTypeArguments = typeArguments;
    return node;
}

/** @internal */
export function getIdentifierTypeArguments(node: Identifier): NodeArray<TypeNode | TypeParameterDeclaration> | undefined {
    return node.emitNode?.identifierTypeArguments;
}

/** @internal */
export function setIdentifierAutoGenerate<T extends Identifier | PrivateIdentifier>(node: T, autoGenerate: AutoGenerateInfo | undefined) {
    getOrCreateEmitNode(node).autoGenerate = autoGenerate;
    return node;
}
