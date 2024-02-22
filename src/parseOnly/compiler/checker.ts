import {
    __String,
    Node,
} from "./types";

let nextNodeId = 1;

/** @internal */
export function getNodeId(node: Node): number {
    if (!node.id) {
        node.id = nextNodeId;
        nextNodeId++;
    }
    return node.id;
}
