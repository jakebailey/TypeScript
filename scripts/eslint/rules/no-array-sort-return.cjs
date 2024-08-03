const { ESLintUtils } = require("@typescript-eslint/utils");
const { createRule } = require("./utils.cjs");
const { getConstrainedTypeAtLocation, isTypeArrayTypeOrUnionOfArrayTypes } = require("@typescript-eslint/type-utils");

/**
 * @import { TSESTree } from "@typescript-eslint/utils"
 * @typedef {TSESTree.CallExpression | TSESTree.NewExpression} CallOrNewExpression
 */
void 0;

module.exports = createRule({
    name: "no-array-sort-return",
    meta: {
        docs: {
            description: ``,
        },
        messages: {
            noArraySortUse: `This call to sort appears to be unintentional as it appears in an expression position. Sort the array in a separate statement, or use slice/toSorted to ensure a copy is made.`,
        },
        schema: [],
        type: "problem",
    },
    defaultOptions: [],

    create(context) {
        const services = ESLintUtils.getParserServices(context, /*allowWithoutFullTypeInformation*/ true);
        if (!services.program) {
            return {};
        }

        const checker = services.program.getTypeChecker();

        /** @type {(node: TSESTree.Expression) => boolean} */
        const isReadPosition = node => {
            // TODO(jakebailey): use a loop instead
            const parent = node.parent;
            switch (parent.type) {
                case "VariableDeclarator":
                case "AssignmentExpression":
                case "ForOfStatement":
                case "ForInStatement":
                case "ReturnStatement":
                    return true;
                case "ExpressionStatement":
                    return false;

                case "CallExpression":
                case "NewExpression":
                    if (parent.callee === node) {
                        return isReadPosition(parent);
                    }
                    return true;
                case "UnaryExpression":
                    return parent.operator !== "void";

                case "MemberExpression":
                case "ConditionalExpression":
                    return isReadPosition(parent);

                default:
                    throw new Error(`Unexpected parent type: ${parent.type}`);
            }
        };

        /** @type {(callee: TSESTree.MemberExpression) => boolean} */
        const isAllowedSortCall = callee => {
            const object = callee.object;

            if (object.type === "ArrayExpression") {
                return true;
            }

            if (object.type !== "CallExpression") {
                return false;
            }

            if (object.callee.type === "Identifier") {
                // TypeScript codebase specific helpers.
                // TODO(jakebailey): handle ts.
                switch (object.callee.name) {
                    case "arrayFrom":
                    case "getOwnKeys":
                        return true;
                }
                return false;
            }

            if (object.callee.type === "MemberExpression" && object.callee.property.type === "Identifier") {
                switch (object.callee.property.name) {
                    case "concat":
                    case "filter":
                    case "map":
                    case "slice":
                        return true;
                }

                if (object.callee.object.type === "Identifier") {
                    if (object.callee.object.name === "Array") {
                        switch (object.callee.property.name) {
                            case "from":
                            case "of":
                                return true;
                        }
                        return false;
                    }

                    if (object.callee.object.name === "Object") {
                        switch (object.callee.property.name) {
                            case "values":
                            case "keys":
                            case "entries":
                                return true;
                        }
                        return false;
                    }
                }
            }

            return false;
        };

        /** @type {(callee: TSESTree.MemberExpression) => void} */
        const checkSortArgument = callee => {
            if (isAllowedSortCall(callee)) return;

            const calleeObjType = getConstrainedTypeAtLocation(services, callee.object);

            if (!isTypeArrayTypeOrUnionOfArrayTypes(calleeObjType, checker)) return;

            if (!isReadPosition(callee)) return;

            context.report({ node: callee.property, messageId: "noArraySortUse" });
        };

        return {
            "CallExpression > MemberExpression[property.name='sort'][computed=false]": checkSortArgument,
        };
    },
});
