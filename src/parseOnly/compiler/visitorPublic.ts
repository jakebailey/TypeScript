import {
    isArray,
    singleOrUndefined,
} from "./core";
import {
    Debug,
} from "./debug";
import {
    Node,
    Visitor,
} from "./types";

/**
 * Visits a Node using the supplied visitor, possibly returning a new Node in its place.
 *
 * - If the input node is undefined, then the output is undefined.
 * - If the visitor returns undefined, then the output is undefined.
 * - If the output node is not undefined, then it will satisfy the test function.
 * - In order to obtain a return type that is more specific than `Node`, a test
 *   function _must_ be provided, and that function must be a type predicate.
 *
 * @param node The Node to visit.
 * @param visitor The callback used to visit the Node.
 * @param test A callback to execute to verify the Node is valid.
 * @param lift An optional callback to execute to lift a NodeArray into a valid Node.
 */
export function visitNode<TIn extends Node | undefined, TVisited extends Node | undefined, TOut extends Node>(
    node: TIn,
    visitor: Visitor<NonNullable<TIn>, TVisited>,
    test: (node: Node) => node is TOut,
    lift?: (node: readonly Node[]) => Node,
): TOut | (TIn & undefined) | (TVisited & undefined);
/**
 * Visits a Node using the supplied visitor, possibly returning a new Node in its place.
 *
 * - If the input node is undefined, then the output is undefined.
 * - If the visitor returns undefined, then the output is undefined.
 * - If the output node is not undefined, then it will satisfy the test function.
 * - In order to obtain a return type that is more specific than `Node`, a test
 *   function _must_ be provided, and that function must be a type predicate.
 *
 * @param node The Node to visit.
 * @param visitor The callback used to visit the Node.
 * @param test A callback to execute to verify the Node is valid.
 * @param lift An optional callback to execute to lift a NodeArray into a valid Node.
 */
export function visitNode<TIn extends Node | undefined, TVisited extends Node | undefined>(
    node: TIn,
    visitor: Visitor<NonNullable<TIn>, TVisited>,
    test?: (node: Node) => boolean,
    lift?: (node: readonly Node[]) => Node,
): Node | (TIn & undefined) | (TVisited & undefined);
export function visitNode(
    node: Node,
    visitor: Visitor,
    test?: (node: Node) => boolean,
    lift?: (node: readonly Node[]) => Node,
): Node | undefined {
    if (node === undefined) {
        // If the input type is undefined, then the output type can be undefined.
        return node;
    }

    const visited = visitor(node);

    let visitedNode: Node | undefined;
    if (visited === undefined) {
        // If the visited node is undefined, then the visitor must have returned undefined,
        // so the visitor must have been declared as able to return undefined, so TOut must be
        // potentially undefined.
        return undefined;
    }
    else if (isArray(visited)) {
        visitedNode = (lift || extractSingleNode)(visited);
    }
    else {
        visitedNode = visited;
    }

    Debug.assertNode(visitedNode, test);
    return visitedNode;
}

// NOTE: Before you can add a new method to `visitEachChildTable`, you must first ensure the `Node` subtype you
//       wish to add is defined in the `HasChildren` union in types.ts.

/**
 * Extracts the single node from a NodeArray.
 *
 * @param nodes The NodeArray.
 */
function extractSingleNode(nodes: readonly Node[]): Node | undefined {
    Debug.assert(nodes.length <= 1, "Too many nodes written to output.");
    return singleOrUndefined(nodes);
}
