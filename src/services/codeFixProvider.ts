import {
    arrayFrom,
    cast,
    CodeActionCommand,
    CodeFixAction,
    CodeFixAllContext,
    CodeFixContext,
    CodeFixContextBase,
    CodeFixRegistration,
    CombinedCodeActions,
    computeSuggestionDiagnostics,
    contains,
    createMultiMap,
    Debug,
    Diagnostic,
    DiagnosticOrDiagnosticAndArguments,
    diagnosticToString,
    DiagnosticWithLocation,
    FileTextChanges,
    flatMap,
    getEmitDeclarations,
    isString,
    map,
    TextChange,
    textChanges,
} from "./_namespaces/ts.js";
import addConvertToUnknownForNonOverlappingTypes from "./codefixes/addConvertToUnknownForNonOverlappingTypes.js";
import addEmptyExportDeclaration from "./codefixes/addEmptyExportDeclaration.js";
import addMissingAsync from "./codefixes/addMissingAsync.js";
import addMissingAwait from "./codefixes/addMissingAwait.js";
import addMissingConst from "./codefixes/addMissingConst.js";
import addMissingDeclareProperty from "./codefixes/addMissingDeclareProperty.js";
import addMissingInvocationForDecorator from "./codefixes/addMissingInvocationForDecorator.js";
import addNameToNamelessParameter from "./codefixes/addNameToNamelessParameter.js";
import addOptionalPropertyUndefined from "./codefixes/addOptionalPropertyUndefined.js";
import annotateWithTypeFromJSDoc from "./codefixes/annotateWithTypeFromJSDoc.js";
import convertConstToLet from "./codefixes/convertConstToLet.js";
import convertFunctionToEs6Class from "./codefixes/convertFunctionToEs6Class.js";
import convertLiteralTypeToMappedType from "./codefixes/convertLiteralTypeToMappedType.js";
import convertToAsyncFunction from "./codefixes/convertToAsyncFunction.js";
import convertToEsModule from "./codefixes/convertToEsModule.js";
import convertToMappedObjectType from "./codefixes/convertToMappedObjectType.js";
import convertToTypeOnlyExport from "./codefixes/convertToTypeOnlyExport.js";
import convertToTypeOnlyImport from "./codefixes/convertToTypeOnlyImport.js";
import convertTypedefToType from "./codefixes/convertTypedefToType.js";
import correctQualifiedNameToIndexedAccessType from "./codefixes/correctQualifiedNameToIndexedAccessType.js";
import disableJsDiagnostics from "./codefixes/disableJsDiagnostics.js";
import fixAddMissingConstraint from "./codefixes/fixAddMissingConstraint.js";
import fixAddMissingMember from "./codefixes/fixAddMissingMember.js";
import fixAddMissingNewOperator from "./codefixes/fixAddMissingNewOperator.js";
import fixAddMissingParam from "./codefixes/fixAddMissingParam.js";
import fixAddModuleReferTypeMissingTypeof from "./codefixes/fixAddModuleReferTypeMissingTypeof.js";
import fixAddVoidToPromise from "./codefixes/fixAddVoidToPromise.js";
import fixAwaitInSyncFunction from "./codefixes/fixAwaitInSyncFunction.js";
import fixCannotFindModule from "./codefixes/fixCannotFindModule.js";
import fixClassDoesntImplementInheritedAbstractMember from "./codefixes/fixClassDoesntImplementInheritedAbstractMember.js";
import fixClassIncorrectlyImplementsInterface from "./codefixes/fixClassIncorrectlyImplementsInterface.js";
import fixClassSuperMustPrecedeThisAccess from "./codefixes/fixClassSuperMustPrecedeThisAccess.js";
import fixConstructorForDerivedNeedSuperCall from "./codefixes/fixConstructorForDerivedNeedSuperCall.js";
import fixEnableJsxFlag from "./codefixes/fixEnableJsxFlag.js";
import fixExpectedComma from "./codefixes/fixExpectedComma.js";
import fixExtendsInterfaceBecomesImplements from "./codefixes/fixExtendsInterfaceBecomesImplements.js";
import fixForgottenThisPropertyAccess from "./codefixes/fixForgottenThisPropertyAccess.js";
import fixImplicitThis from "./codefixes/fixImplicitThis.js";
import fixImportNonExportedMember from "./codefixes/fixImportNonExportedMember.js";
import fixIncorrectNamedTupleSyntax from "./codefixes/fixIncorrectNamedTupleSyntax.js";
import fixInvalidImportSyntax from "./codefixes/fixInvalidImportSyntax.js";
import fixInvalidJsxCharacters from "./codefixes/fixInvalidJsxCharacters.js";
import fixJSDocTypes from "./codefixes/fixJSDocTypes.js";
import fixMissingCallParentheses from "./codefixes/fixMissingCallParentheses.js";
import fixMissingTypeAnnotationOnExports from "./codefixes/fixMissingTypeAnnotationOnExports.js";
import fixModuleAndTargetOptions from "./codefixes/fixModuleAndTargetOptions.js";
import fixNaNEquality from "./codefixes/fixNaNEquality.js";
import fixNoPropertyAccessFromIndexSignature from "./codefixes/fixNoPropertyAccessFromIndexSignature.js";
import fixOverrideModifier from "./codefixes/fixOverrideModifier.js";
import fixPropertyAssignment from "./codefixes/fixPropertyAssignment.js";
import fixPropertyOverrideAccessor from "./codefixes/fixPropertyOverrideAccessor.js";
import fixReturnTypeInAsyncFunction from "./codefixes/fixReturnTypeInAsyncFunction.js";
import fixSpelling from "./codefixes/fixSpelling.js";
import fixStrictClassInitialization from "./codefixes/fixStrictClassInitialization.js";
import fixUnmatchedParameter from "./codefixes/fixUnmatchedParameter.js";
import fixUnreachableCode from "./codefixes/fixUnreachableCode.js";
import fixUnreferenceableDecoratorMetadata from "./codefixes/fixUnreferenceableDecoratorMetadata.js";
import fixUnusedIdentifier from "./codefixes/fixUnusedIdentifier.js";
import fixUnusedLabel from "./codefixes/fixUnusedLabel.js";
import importFixes from "./codefixes/importFixes.js";
import inferFromUsage from "./codefixes/inferFromUsage.js";
import removeAccidentalCallParentheses from "./codefixes/removeAccidentalCallParentheses.js";
import removeUnnecessaryAwait from "./codefixes/removeUnnecessaryAwait.js";
import requireInTs from "./codefixes/requireInTs.js";
import returnValueCorrect from "./codefixes/returnValueCorrect.js";
import splitTypeOnlyImport from "./codefixes/splitTypeOnlyImport.js";
import useBigintLiteral from "./codefixes/useBigintLiteral.js";
import useDefaultImport from "./codefixes/useDefaultImport.js";
import wrapDecoratorInParentheses from "./codefixes/wrapDecoratorInParentheses.js";
import wrapJsxInFragment from "./codefixes/wrapJsxInFragment.js";

const errorCodeToFixes = createMultiMap<string, CodeFixRegistration>();
const fixIdToRegistration = new Map<string, CodeFixRegistration>();

/** @internal */
export function createCodeFixActionWithoutFixAll(fixName: string, changes: FileTextChanges[], description: DiagnosticOrDiagnosticAndArguments) {
    return createCodeFixActionWorker(fixName, diagnosticToString(description), changes, /*fixId*/ undefined, /*fixAllDescription*/ undefined);
}

/** @internal */
export function createCodeFixAction(fixName: string, changes: FileTextChanges[], description: DiagnosticOrDiagnosticAndArguments, fixId: {}, fixAllDescription: DiagnosticOrDiagnosticAndArguments, command?: CodeActionCommand): CodeFixAction {
    return createCodeFixActionWorker(fixName, diagnosticToString(description), changes, fixId, diagnosticToString(fixAllDescription), command);
}

/** @internal */
export function createCodeFixActionMaybeFixAll(fixName: string, changes: FileTextChanges[], description: DiagnosticOrDiagnosticAndArguments, fixId?: {}, fixAllDescription?: DiagnosticOrDiagnosticAndArguments, command?: CodeActionCommand) {
    return createCodeFixActionWorker(fixName, diagnosticToString(description), changes, fixId, fixAllDescription && diagnosticToString(fixAllDescription), command);
}

function createCodeFixActionWorker(fixName: string, description: string, changes: FileTextChanges[], fixId?: {}, fixAllDescription?: string, command?: CodeActionCommand): CodeFixAction {
    return { fixName, description, changes, fixId, fixAllDescription, commands: command ? [command] : undefined };
}

/**
 * @internal
 * @deprecated Import codefixes into codeFixProvider.ts instead.
 */
export function registerCodeFix(reg: CodeFixRegistration) {
    for (const error of reg.errorCodes) {
        errorCodeToFixesArray = undefined;
        errorCodeToFixes.add(String(error), reg);
    }
    if (reg.fixIds) {
        for (const fixId of reg.fixIds) {
            Debug.assert(!fixIdToRegistration.has(fixId));
            fixIdToRegistration.set(fixId, reg);
        }
    }
}

const codefixes: CodeFixRegistration[] = [
    addConvertToUnknownForNonOverlappingTypes,
    addEmptyExportDeclaration,
    addMissingAsync,
    addMissingAwait,
    addMissingConst,
    addMissingDeclareProperty,
    addMissingInvocationForDecorator,
    addNameToNamelessParameter,
    addOptionalPropertyUndefined,
    annotateWithTypeFromJSDoc,
    convertFunctionToEs6Class,
    convertToAsyncFunction,
    convertToEsModule,
    correctQualifiedNameToIndexedAccessType,
    convertToTypeOnlyExport,
    convertToTypeOnlyImport,
    convertTypedefToType,
    convertLiteralTypeToMappedType,
    fixClassIncorrectlyImplementsInterface,
    importFixes,
    fixAddMissingConstraint,
    fixOverrideModifier,
    fixNoPropertyAccessFromIndexSignature,
    fixImplicitThis,
    fixImportNonExportedMember,
    fixIncorrectNamedTupleSyntax,
    fixSpelling,
    returnValueCorrect,
    fixAddMissingMember,
    fixAddMissingNewOperator,
    fixAddMissingParam,
    fixCannotFindModule,
    fixClassDoesntImplementInheritedAbstractMember,
    fixClassSuperMustPrecedeThisAccess,
    fixConstructorForDerivedNeedSuperCall,
    fixEnableJsxFlag,
    fixNaNEquality,
    fixModuleAndTargetOptions,
    fixPropertyAssignment,
    fixExtendsInterfaceBecomesImplements,
    fixForgottenThisPropertyAccess,
    fixInvalidJsxCharacters,
    fixUnmatchedParameter,
    fixUnreferenceableDecoratorMetadata,
    fixUnusedIdentifier,
    fixUnreachableCode,
    fixUnusedLabel,
    fixJSDocTypes,
    fixMissingCallParentheses,
    fixMissingTypeAnnotationOnExports,
    fixAwaitInSyncFunction,
    fixPropertyOverrideAccessor,
    inferFromUsage,
    fixReturnTypeInAsyncFunction,
    disableJsDiagnostics,
    ...fixInvalidImportSyntax,
    fixStrictClassInitialization,
    requireInTs,
    useDefaultImport,
    useBigintLiteral,
    fixAddModuleReferTypeMissingTypeof,
    wrapJsxInFragment,
    wrapDecoratorInParentheses,
    convertToMappedObjectType,
    removeAccidentalCallParentheses,
    removeUnnecessaryAwait,
    splitTypeOnlyImport,
    convertConstToLet,
    fixExpectedComma,
    fixAddVoidToPromise,
];

for (const codefix of codefixes) {
    registerCodeFix(codefix);
}

let errorCodeToFixesArray: readonly string[] | undefined;
/** @internal */
export function getSupportedErrorCodes(): readonly string[] {
    return errorCodeToFixesArray ??= arrayFrom(errorCodeToFixes.keys());
}

function removeFixIdIfFixAllUnavailable(registration: CodeFixRegistration, diagnostics: Diagnostic[]) {
    const { errorCodes } = registration;
    let maybeFixableDiagnostics = 0;
    for (const diag of diagnostics) {
        if (contains(errorCodes, diag.code)) maybeFixableDiagnostics++;
        if (maybeFixableDiagnostics > 1) break;
    }

    const fixAllUnavailable = maybeFixableDiagnostics < 2;
    return ({ fixId, fixAllDescription, ...action }: CodeFixAction): CodeFixAction => {
        return fixAllUnavailable ? action : { ...action, fixId, fixAllDescription };
    };
}

/** @internal */
export function getFixes(context: CodeFixContext): readonly CodeFixAction[] {
    const diagnostics = getDiagnostics(context);
    const registrations = errorCodeToFixes.get(String(context.errorCode));
    return flatMap(registrations, f => map(f.getCodeActions(context), removeFixIdIfFixAllUnavailable(f, diagnostics)));
}

/** @internal */
export function getAllFixes(context: CodeFixAllContext): CombinedCodeActions {
    // Currently fixId is always a string.
    return fixIdToRegistration.get(cast(context.fixId, isString))!.getAllCodeActions!(context);
}

/** @internal */
export function createCombinedCodeActions(changes: FileTextChanges[], commands?: CodeActionCommand[]): CombinedCodeActions {
    return { changes, commands };
}

/** @internal */
export function createFileTextChanges(fileName: string, textChanges: TextChange[]): FileTextChanges {
    return { fileName, textChanges };
}

/** @internal */
export function codeFixAll(
    context: CodeFixAllContext,
    errorCodes: number[],
    use: (changes: textChanges.ChangeTracker, error: DiagnosticWithLocation, commands: CodeActionCommand[]) => void,
): CombinedCodeActions {
    const commands: CodeActionCommand[] = [];
    const changes = textChanges.ChangeTracker.with(context, t => eachDiagnostic(context, errorCodes, diag => use(t, diag, commands)));
    return createCombinedCodeActions(changes, commands.length === 0 ? undefined : commands);
}

/** @internal */
export function eachDiagnostic(context: CodeFixAllContext, errorCodes: readonly number[], cb: (diag: DiagnosticWithLocation) => void): void {
    for (const diag of getDiagnostics(context)) {
        if (contains(errorCodes, diag.code)) {
            cb(diag as DiagnosticWithLocation);
        }
    }
}

function getDiagnostics({ program, sourceFile, cancellationToken }: CodeFixContextBase) {
    const diagnostics = [
        ...program.getSemanticDiagnostics(sourceFile, cancellationToken),
        ...program.getSyntacticDiagnostics(sourceFile, cancellationToken),
        ...computeSuggestionDiagnostics(sourceFile, program, cancellationToken),
    ];
    if (getEmitDeclarations(program.getCompilerOptions())) {
        diagnostics.push(
            ...program.getDeclarationDiagnostics(sourceFile, cancellationToken),
        );
    }
    return diagnostics;
}
