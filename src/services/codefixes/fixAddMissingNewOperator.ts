import {
    codeFixAll,
    createCodeFixAction,
} from "../_namespaces/ts.codefix.js";
import {
    cast,
    CodeFixRegistration,
    Diagnostics,
    factory,
    getTokenAtPosition,
    isCallExpression,
    Node,
    SourceFile,
    textChanges,
    TextSpan,
    textSpanEnd,
} from "../_namespaces/ts.js";

const fixId = "addMissingNewOperator";
const errorCodes = [Diagnostics.Value_of_type_0_is_not_callable_Did_you_mean_to_include_new.code];
const codefix: CodeFixRegistration = {
    errorCodes,
    getCodeActions(context) {
        const { sourceFile, span } = context;
        const changes = textChanges.ChangeTracker.with(context, t => addMissingNewOperator(t, sourceFile, span));
        return [createCodeFixAction(fixId, changes, Diagnostics.Add_missing_new_operator_to_call, fixId, Diagnostics.Add_missing_new_operator_to_all_calls)];
    },
    fixIds: [fixId],
    getAllCodeActions: context => codeFixAll(context, errorCodes, (changes, diag) => addMissingNewOperator(changes, context.sourceFile, diag)),
};
export default codefix;

function addMissingNewOperator(changes: textChanges.ChangeTracker, sourceFile: SourceFile, span: TextSpan): void {
    const call = cast(findAncestorMatchingSpan(sourceFile, span), isCallExpression);
    const newExpression = factory.createNewExpression(call.expression, call.typeArguments, call.arguments);

    changes.replaceNode(sourceFile, call, newExpression);
}

function findAncestorMatchingSpan(sourceFile: SourceFile, span: TextSpan): Node {
    let token = getTokenAtPosition(sourceFile, span.start);
    const end = textSpanEnd(span);
    while (token.end < end) {
        token = token.parent;
    }
    return token;
}
