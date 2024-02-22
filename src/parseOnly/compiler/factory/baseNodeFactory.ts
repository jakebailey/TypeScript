import {
    Node,
    SyntaxKind,
} from "../types";

/**
 * A `BaseNodeFactory` is an abstraction over an `ObjectAllocator` that handles caching `Node` constructors
 * and allocating `Node` instances based on a set of predefined types.
 *
 * @internal
 */
export interface BaseNodeFactory {
    createBaseSourceFileNode(kind: SyntaxKind.SourceFile): Node;
    createBaseIdentifierNode(kind: SyntaxKind.Identifier): Node;
    createBasePrivateIdentifierNode(kind: SyntaxKind.PrivateIdentifier): Node;
    createBaseTokenNode(kind: SyntaxKind): Node;
    createBaseNode(kind: SyntaxKind): Node;
}
