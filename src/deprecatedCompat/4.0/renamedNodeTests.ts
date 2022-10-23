import { Debug, Node, SyntaxKind, TypeAssertion } from "../_namespaces/ts";

// DEPRECATION: Renamed node tests
// DEPRECATION PLAN:
//     - soft: 4.0
//     - warn: 4.1
//     - error: TBD
/** @deprecated Use `isTypeAssertionExpression` instead. */
export const isTypeAssertion = Debug.deprecate(function isTypeAssertion(node: Node): node is TypeAssertion {
    return node.kind === SyntaxKind.TypeAssertionExpression;
}, {
    since: "4.0",
    warnAfter: "4.1",
    message: "Use `isTypeAssertionExpression` instead."
});