import {
    filter,
    isString,
} from "./core";
import {
    Diagnostics,
} from "./diagnosticInformationMap.generated";
import {
    isStringLiteral,
} from "./factory/nodeTests";
import {
    ArrayLiteralExpression,
    CommandLineOption,
    CommandLineOptionOfListType,
    Diagnostic,
    Expression,
    JsonSourceFile,
    Node,
    NodeArray,
    NumericLiteral,
    ObjectLiteralExpression,
    PrefixUnaryExpression,
    PropertyAssignment,
    StringLiteral,
    SyntaxKind,
    TsConfigOnlyOption,
} from "./types";
import {
    createDiagnosticForNodeInSourceFile,
    getTextOfPropertyName,
    isComputedNonLiteralName,
    isStringDoubleQuoted,
} from "./utilities";
import {
    unescapeLeadingUnderscores,
} from "./utilitiesPublic";

// NOTE: The order here is important to default lib ordering as entries will have the same
//       order in the generated program (see `getDefaultLibPriority` in program.ts). This
//       order also affects overload resolution when a type declared in one lib is
//       augmented in another lib.

// Watch related options

// Build related options

/** @internal */
export interface OptionsNameMap {
    optionsNameMap: Map<string, CommandLineOption>;
    shortOptionNames: Map<string, string>;
}

/** @internal */
interface JsonConversionNotifier {
    rootOptions: TsConfigOnlyOption;
    onPropertySet(
        keyText: string,
        value: any,
        propertyAssignment: PropertyAssignment,
        parentOption: TsConfigOnlyOption | undefined,
        option: CommandLineOption | undefined,
    ): void;
}

/**
 * Convert the json syntax tree into the json value and report errors
 * This returns the json value (apart from checking errors) only if returnValue provided is true.
 * Otherwise it just checks the errors and returns undefined
 *
 * @internal
 */
export function convertToJson(
    sourceFile: JsonSourceFile,
    rootExpression: Expression | undefined,
    errors: Diagnostic[],
    returnValue: boolean,
    jsonConversionNotifier: JsonConversionNotifier | undefined,
): any {
    if (!rootExpression) {
        return returnValue ? {} : undefined;
    }

    return convertPropertyValueToJson(rootExpression, jsonConversionNotifier?.rootOptions);

    function convertObjectLiteralExpressionToJson(
        node: ObjectLiteralExpression,
        objectOption: TsConfigOnlyOption | undefined,
    ): any {
        const result: any = returnValue ? {} : undefined;
        for (const element of node.properties) {
            if (element.kind !== SyntaxKind.PropertyAssignment) {
                errors.push(createDiagnosticForNodeInSourceFile(sourceFile, element, Diagnostics.Property_assignment_expected));
                continue;
            }

            if (element.questionToken) {
                errors.push(createDiagnosticForNodeInSourceFile(sourceFile, element.questionToken, Diagnostics.The_0_modifier_can_only_be_used_in_TypeScript_files, "?"));
            }
            if (!isDoubleQuotedString(element.name)) {
                errors.push(createDiagnosticForNodeInSourceFile(sourceFile, element.name, Diagnostics.String_literal_with_double_quotes_expected));
            }

            const textOfKey = isComputedNonLiteralName(element.name) ? undefined : getTextOfPropertyName(element.name);
            const keyText = textOfKey && unescapeLeadingUnderscores(textOfKey);
            const option = keyText ? objectOption?.elementOptions?.get(keyText) : undefined;
            const value = convertPropertyValueToJson(element.initializer, option);
            if (typeof keyText !== "undefined") {
                if (returnValue) {
                    result[keyText] = value;
                }

                // Notify key value set, if user asked for it
                jsonConversionNotifier?.onPropertySet(keyText, value, element, objectOption, option);
            }
        }
        return result;
    }

    function convertArrayLiteralExpressionToJson(
        elements: NodeArray<Expression>,
        elementOption: CommandLineOption | undefined,
    ) {
        if (!returnValue) {
            elements.forEach(element => convertPropertyValueToJson(element, elementOption));
            return undefined;
        }

        // Filter out invalid values
        return filter(elements.map(element => convertPropertyValueToJson(element, elementOption)), v => v !== undefined);
    }

    function convertPropertyValueToJson(valueExpression: Expression, option: CommandLineOption | undefined): any {
        switch (valueExpression.kind) {
            case SyntaxKind.TrueKeyword:
                return true;

            case SyntaxKind.FalseKeyword:
                return false;

            case SyntaxKind.NullKeyword:
                return null; // eslint-disable-line no-null/no-null

            case SyntaxKind.StringLiteral:
                if (!isDoubleQuotedString(valueExpression)) {
                    errors.push(createDiagnosticForNodeInSourceFile(sourceFile, valueExpression, Diagnostics.String_literal_with_double_quotes_expected));
                }
                return (valueExpression as StringLiteral).text;

            case SyntaxKind.NumericLiteral:
                return Number((valueExpression as NumericLiteral).text);

            case SyntaxKind.PrefixUnaryExpression:
                if ((valueExpression as PrefixUnaryExpression).operator !== SyntaxKind.MinusToken || (valueExpression as PrefixUnaryExpression).operand.kind !== SyntaxKind.NumericLiteral) {
                    break; // not valid JSON syntax
                }
                return -Number(((valueExpression as PrefixUnaryExpression).operand as NumericLiteral).text);

            case SyntaxKind.ObjectLiteralExpression:
                const objectLiteralExpression = valueExpression as ObjectLiteralExpression;

                // Currently having element option declaration in the tsconfig with type "object"
                // determines if it needs onSetValidOptionKeyValueInParent callback or not
                // At moment there are only "compilerOptions", "typeAcquisition" and "typingOptions"
                // that satisfies it and need it to modify options set in them (for normalizing file paths)
                // vs what we set in the json
                // If need arises, we can modify this interface and callbacks as needed
                return convertObjectLiteralExpressionToJson(objectLiteralExpression, option as TsConfigOnlyOption);

            case SyntaxKind.ArrayLiteralExpression:
                return convertArrayLiteralExpressionToJson(
                    (valueExpression as ArrayLiteralExpression).elements,
                    option && (option as CommandLineOptionOfListType).element,
                );
        }

        // Not in expected format
        if (option) {
            errors.push(createDiagnosticForNodeInSourceFile(sourceFile, valueExpression, Diagnostics.Compiler_option_0_requires_a_value_of_type_1, option.name, getCompilerOptionValueTypeString(option)));
        }
        else {
            errors.push(createDiagnosticForNodeInSourceFile(sourceFile, valueExpression, Diagnostics.Property_value_can_only_be_string_literal_numeric_literal_true_false_null_object_literal_or_array_literal));
        }

        return undefined;
    }

    function isDoubleQuotedString(node: Node): boolean {
        return isStringLiteral(node) && isStringDoubleQuoted(node, sourceFile);
    }
}

function getCompilerOptionValueTypeString(option: CommandLineOption): string {
    return (option.type === "listOrElement") ?
        `${getCompilerOptionValueTypeString(option.element)} or Array` :
        option.type === "list" ?
        "Array" :
        isString(option.type) ? option.type : "string";
}
