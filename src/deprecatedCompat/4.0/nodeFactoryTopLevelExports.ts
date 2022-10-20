import {
    ArrowFunction, AsteriskToken, BinaryExpression, BinaryOperator, BinaryOperatorToken, BindingName, BooleanLiteral,
    ClassElement, ClassExpression, ColonToken, ConciseBody, ConditionalExpression, ConstructorTypeNode, Debug,
    Decorator, DeprecationOptions, EntityName, EqualsGreaterThanToken, ExclamationToken, ExportDeclaration, Expression,
    ExpressionWithTypeArguments, factory, GeneratedIdentifierFlags, HeritageClause, Identifier, ImportClause,
    IndexSignatureDeclaration, isNodeKind, JSDocParameterTag, JSDocTypeExpression, MethodSignature, Modifier, Mutable,
    NamedExportBindings, NamedImportBindings, Node, NodeArray, NoSubstitutionTemplateLiteral, NumericLiteral,
    ParameterDeclaration, parseBaseNodeFactory, PostfixUnaryExpression, PrefixUnaryExpression, PrimaryExpression,
    PropertyName, PropertySignature, PseudoBigInt, QuestionToken, setParent, setTextRange, setTextRangePosEnd,
    StringLiteral, SyntaxKind, TaggedTemplateExpression, TemplateLiteral, ThisTypeNode, Token, TypeNode,
    TypeOperatorNode, TypeParameterDeclaration, TypePredicateNode, VariableDeclaration, YieldExpression,
} from "../_namespaces/ts";

// DEPRECATION: Node factory top-level exports
// DEPRECATION PLAN:
//     - soft: 4.0
//     - warn: 4.1
//     - error: 5.0
// NOTE: These exports are deprecated in favor of using a `NodeFactory` instance and exist here purely for backwards compatibility reasons.
const factoryDeprecation: DeprecationOptions = { since: "4.0", warnAfter: "4.1", message: "Use the appropriate method on 'ts.factory' or the 'factory' supplied by your transformation context instead." };

/** @deprecated Use `factory.createNodeArray` or the factory supplied by your transformation context instead. */
export const createNodeArray: typeof factory.createNodeArray = Debug.deprecate(factory.createNodeArray, factoryDeprecation);

/** @deprecated Use `factory.createNumericLiteral` or the factory supplied by your transformation context instead. */
export const createNumericLiteral: typeof factory.createNumericLiteral = Debug.deprecate(factory.createNumericLiteral, factoryDeprecation);

/** @deprecated Use `factory.createBigIntLiteral` or the factory supplied by your transformation context instead. */
export const createBigIntLiteral: typeof factory.createBigIntLiteral = Debug.deprecate(factory.createBigIntLiteral, factoryDeprecation);

/** @deprecated Use `factory.createStringLiteral` or the factory supplied by your transformation context instead. */
export const createStringLiteral: typeof factory.createStringLiteral = Debug.deprecate(factory.createStringLiteral, factoryDeprecation);

/** @deprecated Use `factory.createStringLiteralFromNode` or the factory supplied by your transformation context instead. */
export const createStringLiteralFromNode: typeof factory.createStringLiteralFromNode = Debug.deprecate(factory.createStringLiteralFromNode, factoryDeprecation);

/** @deprecated Use `factory.createRegularExpressionLiteral` or the factory supplied by your transformation context instead. */
export const createRegularExpressionLiteral: typeof factory.createRegularExpressionLiteral = Debug.deprecate(factory.createRegularExpressionLiteral, factoryDeprecation);

/** @deprecated Use `factory.createLoopVariable` or the factory supplied by your transformation context instead. */
export const createLoopVariable: typeof factory.createLoopVariable = Debug.deprecate(factory.createLoopVariable, factoryDeprecation);

/** @deprecated Use `factory.createUniqueName` or the factory supplied by your transformation context instead. */
export const createUniqueName: typeof factory.createUniqueName = Debug.deprecate(factory.createUniqueName, factoryDeprecation);

/** @deprecated Use `factory.createPrivateIdentifier` or the factory supplied by your transformation context instead. */
export const createPrivateIdentifier: typeof factory.createPrivateIdentifier = Debug.deprecate(factory.createPrivateIdentifier, factoryDeprecation);

/** @deprecated Use `factory.createSuper` or the factory supplied by your transformation context instead. */
export const createSuper: typeof factory.createSuper = Debug.deprecate(factory.createSuper, factoryDeprecation);

/** @deprecated Use `factory.createThis` or the factory supplied by your transformation context instead. */
export const createThis: typeof factory.createThis = Debug.deprecate(factory.createThis, factoryDeprecation);

/** @deprecated Use `factory.createNull` or the factory supplied by your transformation context instead. */
export const createNull: typeof factory.createNull = Debug.deprecate(factory.createNull, factoryDeprecation);

/** @deprecated Use `factory.createTrue` or the factory supplied by your transformation context instead. */
export const createTrue: typeof factory.createTrue = Debug.deprecate(factory.createTrue, factoryDeprecation);

/** @deprecated Use `factory.createFalse` or the factory supplied by your transformation context instead. */
export const createFalse: typeof factory.createFalse = Debug.deprecate(factory.createFalse, factoryDeprecation);

/** @deprecated Use `factory.createModifier` or the factory supplied by your transformation context instead. */
export const createModifier: typeof factory.createModifier = Debug.deprecate(factory.createModifier, factoryDeprecation);

/** @deprecated Use `factory.createModifiersFromModifierFlags` or the factory supplied by your transformation context instead. */
export const createModifiersFromModifierFlags: typeof factory.createModifiersFromModifierFlags = Debug.deprecate(factory.createModifiersFromModifierFlags, factoryDeprecation);

/** @deprecated Use `factory.createQualifiedName` or the factory supplied by your transformation context instead. */
export const createQualifiedName: typeof factory.createQualifiedName = Debug.deprecate(factory.createQualifiedName, factoryDeprecation);

/** @deprecated Use `factory.updateQualifiedName` or the factory supplied by your transformation context instead. */
export const updateQualifiedName: typeof factory.updateQualifiedName = Debug.deprecate(factory.updateQualifiedName, factoryDeprecation);

/** @deprecated Use `factory.createComputedPropertyName` or the factory supplied by your transformation context instead. */
export const createComputedPropertyName: typeof factory.createComputedPropertyName = Debug.deprecate(factory.createComputedPropertyName, factoryDeprecation);

/** @deprecated Use `factory.updateComputedPropertyName` or the factory supplied by your transformation context instead. */
export const updateComputedPropertyName: typeof factory.updateComputedPropertyName = Debug.deprecate(factory.updateComputedPropertyName, factoryDeprecation);

/** @deprecated Use `factory.createTypeParameterDeclaration` or the factory supplied by your transformation context instead. */
export const createTypeParameterDeclaration: typeof factory.createTypeParameterDeclaration = Debug.deprecate(factory.createTypeParameterDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateTypeParameterDeclaration` or the factory supplied by your transformation context instead. */
export const updateTypeParameterDeclaration: typeof factory.updateTypeParameterDeclaration = Debug.deprecate(factory.updateTypeParameterDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createParameterDeclaration` or the factory supplied by your transformation context instead. */
export const createParameter: typeof factory.createParameterDeclaration = Debug.deprecate(factory.createParameterDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateParameterDeclaration` or the factory supplied by your transformation context instead. */
export const updateParameter: typeof factory.updateParameterDeclaration = Debug.deprecate(factory.updateParameterDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createDecorator` or the factory supplied by your transformation context instead. */
export const createDecorator: typeof factory.createDecorator = Debug.deprecate(factory.createDecorator, factoryDeprecation);

/** @deprecated Use `factory.updateDecorator` or the factory supplied by your transformation context instead. */
export const updateDecorator: typeof factory.updateDecorator = Debug.deprecate(factory.updateDecorator, factoryDeprecation);

/** @deprecated Use `factory.createPropertyDeclaration` or the factory supplied by your transformation context instead. */
export const createProperty: typeof factory.createPropertyDeclaration = Debug.deprecate(factory.createPropertyDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updatePropertyDeclaration` or the factory supplied by your transformation context instead. */
export const updateProperty: typeof factory.updatePropertyDeclaration = Debug.deprecate(factory.updatePropertyDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createMethodDeclaration` or the factory supplied by your transformation context instead. */
export const createMethod: typeof factory.createMethodDeclaration = Debug.deprecate(factory.createMethodDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateMethodDeclaration` or the factory supplied by your transformation context instead. */
export const updateMethod: typeof factory.updateMethodDeclaration = Debug.deprecate(factory.updateMethodDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createConstructorDeclaration` or the factory supplied by your transformation context instead. */
export const createConstructor: typeof factory.createConstructorDeclaration = Debug.deprecate(factory.createConstructorDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateConstructorDeclaration` or the factory supplied by your transformation context instead. */
export const updateConstructor: typeof factory.updateConstructorDeclaration = Debug.deprecate(factory.updateConstructorDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createGetAccessorDeclaration` or the factory supplied by your transformation context instead. */
export const createGetAccessor: typeof factory.createGetAccessorDeclaration = Debug.deprecate(factory.createGetAccessorDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateGetAccessorDeclaration` or the factory supplied by your transformation context instead. */
export const updateGetAccessor: typeof factory.updateGetAccessorDeclaration = Debug.deprecate(factory.updateGetAccessorDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createSetAccessorDeclaration` or the factory supplied by your transformation context instead. */
export const createSetAccessor: typeof factory.createSetAccessorDeclaration = Debug.deprecate(factory.createSetAccessorDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateSetAccessorDeclaration` or the factory supplied by your transformation context instead. */
export const updateSetAccessor: typeof factory.updateSetAccessorDeclaration = Debug.deprecate(factory.updateSetAccessorDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createCallSignature` or the factory supplied by your transformation context instead. */
export const createCallSignature: typeof factory.createCallSignature = Debug.deprecate(factory.createCallSignature, factoryDeprecation);

/** @deprecated Use `factory.updateCallSignature` or the factory supplied by your transformation context instead. */
export const updateCallSignature: typeof factory.updateCallSignature = Debug.deprecate(factory.updateCallSignature, factoryDeprecation);

/** @deprecated Use `factory.createConstructSignature` or the factory supplied by your transformation context instead. */
export const createConstructSignature: typeof factory.createConstructSignature = Debug.deprecate(factory.createConstructSignature, factoryDeprecation);

/** @deprecated Use `factory.updateConstructSignature` or the factory supplied by your transformation context instead. */
export const updateConstructSignature: typeof factory.updateConstructSignature = Debug.deprecate(factory.updateConstructSignature, factoryDeprecation);

/** @deprecated Use `factory.updateIndexSignature` or the factory supplied by your transformation context instead. */
export const updateIndexSignature: typeof factory.updateIndexSignature = Debug.deprecate(factory.updateIndexSignature, factoryDeprecation);

/** @deprecated Use `factory.createKeywordTypeNode` or the factory supplied by your transformation context instead. */
export const createKeywordTypeNode: typeof factory.createKeywordTypeNode = Debug.deprecate(factory.createKeywordTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createTypePredicateNode` or the factory supplied by your transformation context instead. */
export const createTypePredicateNodeWithModifier: typeof factory.createTypePredicateNode = Debug.deprecate(factory.createTypePredicateNode, factoryDeprecation);

/** @deprecated Use `factory.updateTypePredicateNode` or the factory supplied by your transformation context instead. */
export const updateTypePredicateNodeWithModifier: typeof factory.updateTypePredicateNode = Debug.deprecate(factory.updateTypePredicateNode, factoryDeprecation);

/** @deprecated Use `factory.createTypeReferenceNode` or the factory supplied by your transformation context instead. */
export const createTypeReferenceNode: typeof factory.createTypeReferenceNode = Debug.deprecate(factory.createTypeReferenceNode, factoryDeprecation);

/** @deprecated Use `factory.updateTypeReferenceNode` or the factory supplied by your transformation context instead. */
export const updateTypeReferenceNode: typeof factory.updateTypeReferenceNode = Debug.deprecate(factory.updateTypeReferenceNode, factoryDeprecation);

/** @deprecated Use `factory.createFunctionTypeNode` or the factory supplied by your transformation context instead. */
export const createFunctionTypeNode: typeof factory.createFunctionTypeNode = Debug.deprecate(factory.createFunctionTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateFunctionTypeNode` or the factory supplied by your transformation context instead. */
export const updateFunctionTypeNode: typeof factory.updateFunctionTypeNode = Debug.deprecate(factory.updateFunctionTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createConstructorTypeNode` or the factory supplied by your transformation context instead. */
export const createConstructorTypeNode = Debug.deprecate((
    typeParameters: readonly TypeParameterDeclaration[] | undefined,
    parameters: readonly ParameterDeclaration[],
    type: TypeNode
) => {
    return factory.createConstructorTypeNode(/*modifiers*/ undefined, typeParameters, parameters, type);
}, factoryDeprecation);

/** @deprecated Use `factory.updateConstructorTypeNode` or the factory supplied by your transformation context instead. */
export const updateConstructorTypeNode = Debug.deprecate((
    node: ConstructorTypeNode,
    typeParameters: NodeArray<TypeParameterDeclaration> | undefined,
    parameters: NodeArray<ParameterDeclaration>,
    type: TypeNode
) => {
    return factory.updateConstructorTypeNode(node, node.modifiers, typeParameters, parameters, type);
}, factoryDeprecation);

/** @deprecated Use `factory.createTypeQueryNode` or the factory supplied by your transformation context instead. */
export const createTypeQueryNode: typeof factory.createTypeQueryNode = Debug.deprecate(factory.createTypeQueryNode, factoryDeprecation);

/** @deprecated Use `factory.updateTypeQueryNode` or the factory supplied by your transformation context instead. */
export const updateTypeQueryNode: typeof factory.updateTypeQueryNode = Debug.deprecate(factory.updateTypeQueryNode, factoryDeprecation);

/** @deprecated Use `factory.createTypeLiteralNode` or the factory supplied by your transformation context instead. */
export const createTypeLiteralNode: typeof factory.createTypeLiteralNode = Debug.deprecate(factory.createTypeLiteralNode, factoryDeprecation);

/** @deprecated Use `factory.updateTypeLiteralNode` or the factory supplied by your transformation context instead. */
export const updateTypeLiteralNode: typeof factory.updateTypeLiteralNode = Debug.deprecate(factory.updateTypeLiteralNode, factoryDeprecation);

/** @deprecated Use `factory.createArrayTypeNode` or the factory supplied by your transformation context instead. */
export const createArrayTypeNode: typeof factory.createArrayTypeNode = Debug.deprecate(factory.createArrayTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateArrayTypeNode` or the factory supplied by your transformation context instead. */
export const updateArrayTypeNode: typeof factory.updateArrayTypeNode = Debug.deprecate(factory.updateArrayTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createTupleTypeNode` or the factory supplied by your transformation context instead. */
export const createTupleTypeNode: typeof factory.createTupleTypeNode = Debug.deprecate(factory.createTupleTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateTupleTypeNode` or the factory supplied by your transformation context instead. */
export const updateTupleTypeNode: typeof factory.updateTupleTypeNode = Debug.deprecate(factory.updateTupleTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createOptionalTypeNode` or the factory supplied by your transformation context instead. */
export const createOptionalTypeNode: typeof factory.createOptionalTypeNode = Debug.deprecate(factory.createOptionalTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateOptionalTypeNode` or the factory supplied by your transformation context instead. */
export const updateOptionalTypeNode: typeof factory.updateOptionalTypeNode = Debug.deprecate(factory.updateOptionalTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createRestTypeNode` or the factory supplied by your transformation context instead. */
export const createRestTypeNode: typeof factory.createRestTypeNode = Debug.deprecate(factory.createRestTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateRestTypeNode` or the factory supplied by your transformation context instead. */
export const updateRestTypeNode: typeof factory.updateRestTypeNode = Debug.deprecate(factory.updateRestTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createUnionTypeNode` or the factory supplied by your transformation context instead. */
export const createUnionTypeNode: typeof factory.createUnionTypeNode = Debug.deprecate(factory.createUnionTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateUnionTypeNode` or the factory supplied by your transformation context instead. */
export const updateUnionTypeNode: typeof factory.updateUnionTypeNode = Debug.deprecate(factory.updateUnionTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createIntersectionTypeNode` or the factory supplied by your transformation context instead. */
export const createIntersectionTypeNode: typeof factory.createIntersectionTypeNode = Debug.deprecate(factory.createIntersectionTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateIntersectionTypeNode` or the factory supplied by your transformation context instead. */
export const updateIntersectionTypeNode: typeof factory.updateIntersectionTypeNode = Debug.deprecate(factory.updateIntersectionTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createConditionalTypeNode` or the factory supplied by your transformation context instead. */
export const createConditionalTypeNode: typeof factory.createConditionalTypeNode = Debug.deprecate(factory.createConditionalTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateConditionalTypeNode` or the factory supplied by your transformation context instead. */
export const updateConditionalTypeNode: typeof factory.updateConditionalTypeNode = Debug.deprecate(factory.updateConditionalTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createInferTypeNode` or the factory supplied by your transformation context instead. */
export const createInferTypeNode: typeof factory.createInferTypeNode = Debug.deprecate(factory.createInferTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateInferTypeNode` or the factory supplied by your transformation context instead. */
export const updateInferTypeNode: typeof factory.updateInferTypeNode = Debug.deprecate(factory.updateInferTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createImportTypeNode` or the factory supplied by your transformation context instead. */
export const createImportTypeNode: typeof factory.createImportTypeNode = Debug.deprecate(factory.createImportTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateImportTypeNode` or the factory supplied by your transformation context instead. */
export const updateImportTypeNode: typeof factory.updateImportTypeNode = Debug.deprecate(factory.updateImportTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createParenthesizedType` or the factory supplied by your transformation context instead. */
export const createParenthesizedType: typeof factory.createParenthesizedType = Debug.deprecate(factory.createParenthesizedType, factoryDeprecation);

/** @deprecated Use `factory.updateParenthesizedType` or the factory supplied by your transformation context instead. */
export const updateParenthesizedType: typeof factory.updateParenthesizedType = Debug.deprecate(factory.updateParenthesizedType, factoryDeprecation);

/** @deprecated Use `factory.createThisTypeNode` or the factory supplied by your transformation context instead. */
export const createThisTypeNode: typeof factory.createThisTypeNode = Debug.deprecate(factory.createThisTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateTypeOperatorNode` or the factory supplied by your transformation context instead. */
export const updateTypeOperatorNode: typeof factory.updateTypeOperatorNode = Debug.deprecate(factory.updateTypeOperatorNode, factoryDeprecation);

/** @deprecated Use `factory.createIndexedAccessTypeNode` or the factory supplied by your transformation context instead. */
export const createIndexedAccessTypeNode: typeof factory.createIndexedAccessTypeNode = Debug.deprecate(factory.createIndexedAccessTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateIndexedAccessTypeNode` or the factory supplied by your transformation context instead. */
export const updateIndexedAccessTypeNode: typeof factory.updateIndexedAccessTypeNode = Debug.deprecate(factory.updateIndexedAccessTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createMappedTypeNode` or the factory supplied by your transformation context instead. */
export const createMappedTypeNode: typeof factory.createMappedTypeNode = Debug.deprecate(factory.createMappedTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateMappedTypeNode` or the factory supplied by your transformation context instead. */
export const updateMappedTypeNode: typeof factory.updateMappedTypeNode = Debug.deprecate(factory.updateMappedTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createLiteralTypeNode` or the factory supplied by your transformation context instead. */
export const createLiteralTypeNode: typeof factory.createLiteralTypeNode = Debug.deprecate(factory.createLiteralTypeNode, factoryDeprecation);

/** @deprecated Use `factory.updateLiteralTypeNode` or the factory supplied by your transformation context instead. */
export const updateLiteralTypeNode: typeof factory.updateLiteralTypeNode = Debug.deprecate(factory.updateLiteralTypeNode, factoryDeprecation);

/** @deprecated Use `factory.createObjectBindingPattern` or the factory supplied by your transformation context instead. */
export const createObjectBindingPattern: typeof factory.createObjectBindingPattern = Debug.deprecate(factory.createObjectBindingPattern, factoryDeprecation);

/** @deprecated Use `factory.updateObjectBindingPattern` or the factory supplied by your transformation context instead. */
export const updateObjectBindingPattern: typeof factory.updateObjectBindingPattern = Debug.deprecate(factory.updateObjectBindingPattern, factoryDeprecation);

/** @deprecated Use `factory.createArrayBindingPattern` or the factory supplied by your transformation context instead. */
export const createArrayBindingPattern: typeof factory.createArrayBindingPattern = Debug.deprecate(factory.createArrayBindingPattern, factoryDeprecation);

/** @deprecated Use `factory.updateArrayBindingPattern` or the factory supplied by your transformation context instead. */
export const updateArrayBindingPattern: typeof factory.updateArrayBindingPattern = Debug.deprecate(factory.updateArrayBindingPattern, factoryDeprecation);

/** @deprecated Use `factory.createBindingElement` or the factory supplied by your transformation context instead. */
export const createBindingElement: typeof factory.createBindingElement = Debug.deprecate(factory.createBindingElement, factoryDeprecation);

/** @deprecated Use `factory.updateBindingElement` or the factory supplied by your transformation context instead. */
export const updateBindingElement: typeof factory.updateBindingElement = Debug.deprecate(factory.updateBindingElement, factoryDeprecation);

/** @deprecated Use `factory.createArrayLiteralExpression` or the factory supplied by your transformation context instead. */
export const createArrayLiteral: typeof factory.createArrayLiteralExpression = Debug.deprecate(factory.createArrayLiteralExpression, factoryDeprecation);

/** @deprecated Use `factory.updateArrayLiteralExpression` or the factory supplied by your transformation context instead. */
export const updateArrayLiteral: typeof factory.updateArrayLiteralExpression = Debug.deprecate(factory.updateArrayLiteralExpression, factoryDeprecation);

/** @deprecated Use `factory.createObjectLiteralExpression` or the factory supplied by your transformation context instead. */
export const createObjectLiteral: typeof factory.createObjectLiteralExpression = Debug.deprecate(factory.createObjectLiteralExpression, factoryDeprecation);

/** @deprecated Use `factory.updateObjectLiteralExpression` or the factory supplied by your transformation context instead. */
export const updateObjectLiteral: typeof factory.updateObjectLiteralExpression = Debug.deprecate(factory.updateObjectLiteralExpression, factoryDeprecation);

/** @deprecated Use `factory.createPropertyAccessExpression` or the factory supplied by your transformation context instead. */
export const createPropertyAccess: typeof factory.createPropertyAccessExpression = Debug.deprecate(factory.createPropertyAccessExpression, factoryDeprecation);

/** @deprecated Use `factory.updatePropertyAccessExpression` or the factory supplied by your transformation context instead. */
export const updatePropertyAccess: typeof factory.updatePropertyAccessExpression = Debug.deprecate(factory.updatePropertyAccessExpression, factoryDeprecation);

/** @deprecated Use `factory.createPropertyAccessChain` or the factory supplied by your transformation context instead. */
export const createPropertyAccessChain: typeof factory.createPropertyAccessChain = Debug.deprecate(factory.createPropertyAccessChain, factoryDeprecation);

/** @deprecated Use `factory.updatePropertyAccessChain` or the factory supplied by your transformation context instead. */
export const updatePropertyAccessChain: typeof factory.updatePropertyAccessChain = Debug.deprecate(factory.updatePropertyAccessChain, factoryDeprecation);

/** @deprecated Use `factory.createElementAccessExpression` or the factory supplied by your transformation context instead. */
export const createElementAccess: typeof factory.createElementAccessExpression = Debug.deprecate(factory.createElementAccessExpression, factoryDeprecation);

/** @deprecated Use `factory.updateElementAccessExpression` or the factory supplied by your transformation context instead. */
export const updateElementAccess: typeof factory.updateElementAccessExpression = Debug.deprecate(factory.updateElementAccessExpression, factoryDeprecation);

/** @deprecated Use `factory.createElementAccessChain` or the factory supplied by your transformation context instead. */
export const createElementAccessChain: typeof factory.createElementAccessChain = Debug.deprecate(factory.createElementAccessChain, factoryDeprecation);

/** @deprecated Use `factory.updateElementAccessChain` or the factory supplied by your transformation context instead. */
export const updateElementAccessChain: typeof factory.updateElementAccessChain = Debug.deprecate(factory.updateElementAccessChain, factoryDeprecation);

/** @deprecated Use `factory.createCallExpression` or the factory supplied by your transformation context instead. */
export const createCall: typeof factory.createCallExpression = Debug.deprecate(factory.createCallExpression, factoryDeprecation);

/** @deprecated Use `factory.updateCallExpression` or the factory supplied by your transformation context instead. */
export const updateCall: typeof factory.updateCallExpression = Debug.deprecate(factory.updateCallExpression, factoryDeprecation);

/** @deprecated Use `factory.createCallChain` or the factory supplied by your transformation context instead. */
export const createCallChain: typeof factory.createCallChain = Debug.deprecate(factory.createCallChain, factoryDeprecation);

/** @deprecated Use `factory.updateCallChain` or the factory supplied by your transformation context instead. */
export const updateCallChain: typeof factory.updateCallChain = Debug.deprecate(factory.updateCallChain, factoryDeprecation);

/** @deprecated Use `factory.createNewExpression` or the factory supplied by your transformation context instead. */
export const createNew: typeof factory.createNewExpression = Debug.deprecate(factory.createNewExpression, factoryDeprecation);

/** @deprecated Use `factory.updateNewExpression` or the factory supplied by your transformation context instead. */
export const updateNew: typeof factory.updateNewExpression = Debug.deprecate(factory.updateNewExpression, factoryDeprecation);

/** @deprecated Use `factory.createTypeAssertion` or the factory supplied by your transformation context instead. */
export const createTypeAssertion: typeof factory.createTypeAssertion = Debug.deprecate(factory.createTypeAssertion, factoryDeprecation);

/** @deprecated Use `factory.updateTypeAssertion` or the factory supplied by your transformation context instead. */
export const updateTypeAssertion: typeof factory.updateTypeAssertion = Debug.deprecate(factory.updateTypeAssertion, factoryDeprecation);

/** @deprecated Use `factory.createParenthesizedExpression` or the factory supplied by your transformation context instead. */
export const createParen: typeof factory.createParenthesizedExpression = Debug.deprecate(factory.createParenthesizedExpression, factoryDeprecation);

/** @deprecated Use `factory.updateParenthesizedExpression` or the factory supplied by your transformation context instead. */
export const updateParen: typeof factory.updateParenthesizedExpression = Debug.deprecate(factory.updateParenthesizedExpression, factoryDeprecation);

/** @deprecated Use `factory.createFunctionExpression` or the factory supplied by your transformation context instead. */
export const createFunctionExpression: typeof factory.createFunctionExpression = Debug.deprecate(factory.createFunctionExpression, factoryDeprecation);

/** @deprecated Use `factory.updateFunctionExpression` or the factory supplied by your transformation context instead. */
export const updateFunctionExpression: typeof factory.updateFunctionExpression = Debug.deprecate(factory.updateFunctionExpression, factoryDeprecation);

/** @deprecated Use `factory.createDeleteExpression` or the factory supplied by your transformation context instead. */
export const createDelete: typeof factory.createDeleteExpression = Debug.deprecate(factory.createDeleteExpression, factoryDeprecation);

/** @deprecated Use `factory.updateDeleteExpression` or the factory supplied by your transformation context instead. */
export const updateDelete: typeof factory.updateDeleteExpression = Debug.deprecate(factory.updateDeleteExpression, factoryDeprecation);

/** @deprecated Use `factory.createTypeOfExpression` or the factory supplied by your transformation context instead. */
export const createTypeOf: typeof factory.createTypeOfExpression = Debug.deprecate(factory.createTypeOfExpression, factoryDeprecation);

/** @deprecated Use `factory.updateTypeOfExpression` or the factory supplied by your transformation context instead. */
export const updateTypeOf: typeof factory.updateTypeOfExpression = Debug.deprecate(factory.updateTypeOfExpression, factoryDeprecation);

/** @deprecated Use `factory.createVoidExpression` or the factory supplied by your transformation context instead. */
export const createVoid: typeof factory.createVoidExpression = Debug.deprecate(factory.createVoidExpression, factoryDeprecation);

/** @deprecated Use `factory.updateVoidExpression` or the factory supplied by your transformation context instead. */
export const updateVoid: typeof factory.updateVoidExpression = Debug.deprecate(factory.updateVoidExpression, factoryDeprecation);

/** @deprecated Use `factory.createAwaitExpression` or the factory supplied by your transformation context instead. */
export const createAwait: typeof factory.createAwaitExpression = Debug.deprecate(factory.createAwaitExpression, factoryDeprecation);

/** @deprecated Use `factory.updateAwaitExpression` or the factory supplied by your transformation context instead. */
export const updateAwait: typeof factory.updateAwaitExpression = Debug.deprecate(factory.updateAwaitExpression, factoryDeprecation);

/** @deprecated Use `factory.createPrefixExpression` or the factory supplied by your transformation context instead. */
export const createPrefix: typeof factory.createPrefixUnaryExpression = Debug.deprecate(factory.createPrefixUnaryExpression, factoryDeprecation);

/** @deprecated Use `factory.updatePrefixExpression` or the factory supplied by your transformation context instead. */
export const updatePrefix: typeof factory.updatePrefixUnaryExpression = Debug.deprecate(factory.updatePrefixUnaryExpression, factoryDeprecation);

/** @deprecated Use `factory.createPostfixUnaryExpression` or the factory supplied by your transformation context instead. */
export const createPostfix: typeof factory.createPostfixUnaryExpression = Debug.deprecate(factory.createPostfixUnaryExpression, factoryDeprecation);

/** @deprecated Use `factory.updatePostfixUnaryExpression` or the factory supplied by your transformation context instead. */
export const updatePostfix: typeof factory.updatePostfixUnaryExpression = Debug.deprecate(factory.updatePostfixUnaryExpression, factoryDeprecation);

/** @deprecated Use `factory.createBinaryExpression` or the factory supplied by your transformation context instead. */
export const createBinary: typeof factory.createBinaryExpression = Debug.deprecate(factory.createBinaryExpression, factoryDeprecation);

/** @deprecated Use `factory.updateConditionalExpression` or the factory supplied by your transformation context instead. */
export const updateConditional: typeof factory.updateConditionalExpression = Debug.deprecate(factory.updateConditionalExpression, factoryDeprecation);

/** @deprecated Use `factory.createTemplateExpression` or the factory supplied by your transformation context instead. */
export const createTemplateExpression: typeof factory.createTemplateExpression = Debug.deprecate(factory.createTemplateExpression, factoryDeprecation);

/** @deprecated Use `factory.updateTemplateExpression` or the factory supplied by your transformation context instead. */
export const updateTemplateExpression: typeof factory.updateTemplateExpression = Debug.deprecate(factory.updateTemplateExpression, factoryDeprecation);

/** @deprecated Use `factory.createTemplateHead` or the factory supplied by your transformation context instead. */
export const createTemplateHead: typeof factory.createTemplateHead = Debug.deprecate(factory.createTemplateHead, factoryDeprecation);

/** @deprecated Use `factory.createTemplateMiddle` or the factory supplied by your transformation context instead. */
export const createTemplateMiddle: typeof factory.createTemplateMiddle = Debug.deprecate(factory.createTemplateMiddle, factoryDeprecation);

/** @deprecated Use `factory.createTemplateTail` or the factory supplied by your transformation context instead. */
export const createTemplateTail: typeof factory.createTemplateTail = Debug.deprecate(factory.createTemplateTail, factoryDeprecation);

/** @deprecated Use `factory.createNoSubstitutionTemplateLiteral` or the factory supplied by your transformation context instead. */
export const createNoSubstitutionTemplateLiteral: typeof factory.createNoSubstitutionTemplateLiteral = Debug.deprecate(factory.createNoSubstitutionTemplateLiteral, factoryDeprecation);

/** @deprecated Use `factory.updateYieldExpression` or the factory supplied by your transformation context instead. */
export const updateYield: typeof factory.updateYieldExpression = Debug.deprecate(factory.updateYieldExpression, factoryDeprecation);

/** @deprecated Use `factory.createSpreadExpression` or the factory supplied by your transformation context instead. */
export const createSpread: typeof factory.createSpreadElement = Debug.deprecate(factory.createSpreadElement, factoryDeprecation);

/** @deprecated Use `factory.updateSpreadExpression` or the factory supplied by your transformation context instead. */
export const updateSpread: typeof factory.updateSpreadElement = Debug.deprecate(factory.updateSpreadElement, factoryDeprecation);

/** @deprecated Use `factory.createOmittedExpression` or the factory supplied by your transformation context instead. */
export const createOmittedExpression: typeof factory.createOmittedExpression = Debug.deprecate(factory.createOmittedExpression, factoryDeprecation);

/** @deprecated Use `factory.createAsExpression` or the factory supplied by your transformation context instead. */
export const createAsExpression: typeof factory.createAsExpression = Debug.deprecate(factory.createAsExpression, factoryDeprecation);

/** @deprecated Use `factory.updateAsExpression` or the factory supplied by your transformation context instead. */
export const updateAsExpression: typeof factory.updateAsExpression = Debug.deprecate(factory.updateAsExpression, factoryDeprecation);

/** @deprecated Use `factory.createNonNullExpression` or the factory supplied by your transformation context instead. */
export const createNonNullExpression: typeof factory.createNonNullExpression = Debug.deprecate(factory.createNonNullExpression, factoryDeprecation);

/** @deprecated Use `factory.updateNonNullExpression` or the factory supplied by your transformation context instead. */
export const updateNonNullExpression: typeof factory.updateNonNullExpression = Debug.deprecate(factory.updateNonNullExpression, factoryDeprecation);

/** @deprecated Use `factory.createNonNullChain` or the factory supplied by your transformation context instead. */
export const createNonNullChain: typeof factory.createNonNullChain = Debug.deprecate(factory.createNonNullChain, factoryDeprecation);

/** @deprecated Use `factory.updateNonNullChain` or the factory supplied by your transformation context instead. */
export const updateNonNullChain: typeof factory.updateNonNullChain = Debug.deprecate(factory.updateNonNullChain, factoryDeprecation);

/** @deprecated Use `factory.createMetaProperty` or the factory supplied by your transformation context instead. */
export const createMetaProperty: typeof factory.createMetaProperty = Debug.deprecate(factory.createMetaProperty, factoryDeprecation);

/** @deprecated Use `factory.updateMetaProperty` or the factory supplied by your transformation context instead. */
export const updateMetaProperty: typeof factory.updateMetaProperty = Debug.deprecate(factory.updateMetaProperty, factoryDeprecation);

/** @deprecated Use `factory.createTemplateSpan` or the factory supplied by your transformation context instead. */
export const createTemplateSpan: typeof factory.createTemplateSpan = Debug.deprecate(factory.createTemplateSpan, factoryDeprecation);

/** @deprecated Use `factory.updateTemplateSpan` or the factory supplied by your transformation context instead. */
export const updateTemplateSpan: typeof factory.updateTemplateSpan = Debug.deprecate(factory.updateTemplateSpan, factoryDeprecation);

/** @deprecated Use `factory.createSemicolonClassElement` or the factory supplied by your transformation context instead. */
export const createSemicolonClassElement: typeof factory.createSemicolonClassElement = Debug.deprecate(factory.createSemicolonClassElement, factoryDeprecation);

/** @deprecated Use `factory.createBlock` or the factory supplied by your transformation context instead. */
export const createBlock: typeof factory.createBlock = Debug.deprecate(factory.createBlock, factoryDeprecation);

/** @deprecated Use `factory.updateBlock` or the factory supplied by your transformation context instead. */
export const updateBlock: typeof factory.updateBlock = Debug.deprecate(factory.updateBlock, factoryDeprecation);

/** @deprecated Use `factory.createVariableStatement` or the factory supplied by your transformation context instead. */
export const createVariableStatement: typeof factory.createVariableStatement = Debug.deprecate(factory.createVariableStatement, factoryDeprecation);

/** @deprecated Use `factory.updateVariableStatement` or the factory supplied by your transformation context instead. */
export const updateVariableStatement: typeof factory.updateVariableStatement = Debug.deprecate(factory.updateVariableStatement, factoryDeprecation);

/** @deprecated Use `factory.createEmptyStatement` or the factory supplied by your transformation context instead. */
export const createEmptyStatement: typeof factory.createEmptyStatement = Debug.deprecate(factory.createEmptyStatement, factoryDeprecation);

/** @deprecated Use `factory.createExpressionStatement` or the factory supplied by your transformation context instead. */
export const createExpressionStatement: typeof factory.createExpressionStatement = Debug.deprecate(factory.createExpressionStatement, factoryDeprecation);

/** @deprecated Use `factory.updateExpressionStatement` or the factory supplied by your transformation context instead. */
export const updateExpressionStatement: typeof factory.updateExpressionStatement = Debug.deprecate(factory.updateExpressionStatement, factoryDeprecation);

/** @deprecated Use `factory.createExpressionStatement` or the factory supplied by your transformation context instead. */
export const createStatement: typeof factory.createExpressionStatement = Debug.deprecate(factory.createExpressionStatement, factoryDeprecation);

/** @deprecated Use `factory.updateExpressionStatement` or the factory supplied by your transformation context instead. */
export const updateStatement: typeof factory.updateExpressionStatement = Debug.deprecate(factory.updateExpressionStatement, factoryDeprecation);

/** @deprecated Use `factory.createIfStatement` or the factory supplied by your transformation context instead. */
export const createIf: typeof factory.createIfStatement = Debug.deprecate(factory.createIfStatement, factoryDeprecation);

/** @deprecated Use `factory.updateIfStatement` or the factory supplied by your transformation context instead. */
export const updateIf: typeof factory.updateIfStatement = Debug.deprecate(factory.updateIfStatement, factoryDeprecation);

/** @deprecated Use `factory.createDoStatement` or the factory supplied by your transformation context instead. */
export const createDo: typeof factory.createDoStatement = Debug.deprecate(factory.createDoStatement, factoryDeprecation);

/** @deprecated Use `factory.updateDoStatement` or the factory supplied by your transformation context instead. */
export const updateDo: typeof factory.updateDoStatement = Debug.deprecate(factory.updateDoStatement, factoryDeprecation);

/** @deprecated Use `factory.createWhileStatement` or the factory supplied by your transformation context instead. */
export const createWhile: typeof factory.createWhileStatement = Debug.deprecate(factory.createWhileStatement, factoryDeprecation);

/** @deprecated Use `factory.updateWhileStatement` or the factory supplied by your transformation context instead. */
export const updateWhile: typeof factory.updateWhileStatement = Debug.deprecate(factory.updateWhileStatement, factoryDeprecation);

/** @deprecated Use `factory.createForStatement` or the factory supplied by your transformation context instead. */
export const createFor: typeof factory.createForStatement = Debug.deprecate(factory.createForStatement, factoryDeprecation);

/** @deprecated Use `factory.updateForStatement` or the factory supplied by your transformation context instead. */
export const updateFor: typeof factory.updateForStatement = Debug.deprecate(factory.updateForStatement, factoryDeprecation);

/** @deprecated Use `factory.createForInStatement` or the factory supplied by your transformation context instead. */
export const createForIn: typeof factory.createForInStatement = Debug.deprecate(factory.createForInStatement, factoryDeprecation);

/** @deprecated Use `factory.updateForInStatement` or the factory supplied by your transformation context instead. */
export const updateForIn: typeof factory.updateForInStatement = Debug.deprecate(factory.updateForInStatement, factoryDeprecation);

/** @deprecated Use `factory.createForOfStatement` or the factory supplied by your transformation context instead. */
export const createForOf: typeof factory.createForOfStatement = Debug.deprecate(factory.createForOfStatement, factoryDeprecation);

/** @deprecated Use `factory.updateForOfStatement` or the factory supplied by your transformation context instead. */
export const updateForOf: typeof factory.updateForOfStatement = Debug.deprecate(factory.updateForOfStatement, factoryDeprecation);

/** @deprecated Use `factory.createContinueStatement` or the factory supplied by your transformation context instead. */
export const createContinue: typeof factory.createContinueStatement = Debug.deprecate(factory.createContinueStatement, factoryDeprecation);

/** @deprecated Use `factory.updateContinueStatement` or the factory supplied by your transformation context instead. */
export const updateContinue: typeof factory.updateContinueStatement = Debug.deprecate(factory.updateContinueStatement, factoryDeprecation);

/** @deprecated Use `factory.createBreakStatement` or the factory supplied by your transformation context instead. */
export const createBreak: typeof factory.createBreakStatement = Debug.deprecate(factory.createBreakStatement, factoryDeprecation);

/** @deprecated Use `factory.updateBreakStatement` or the factory supplied by your transformation context instead. */
export const updateBreak: typeof factory.updateBreakStatement = Debug.deprecate(factory.updateBreakStatement, factoryDeprecation);

/** @deprecated Use `factory.createReturnStatement` or the factory supplied by your transformation context instead. */
export const createReturn: typeof factory.createReturnStatement = Debug.deprecate(factory.createReturnStatement, factoryDeprecation);

/** @deprecated Use `factory.updateReturnStatement` or the factory supplied by your transformation context instead. */
export const updateReturn: typeof factory.updateReturnStatement = Debug.deprecate(factory.updateReturnStatement, factoryDeprecation);

/** @deprecated Use `factory.createWithStatement` or the factory supplied by your transformation context instead. */
export const createWith: typeof factory.createWithStatement = Debug.deprecate(factory.createWithStatement, factoryDeprecation);

/** @deprecated Use `factory.updateWithStatement` or the factory supplied by your transformation context instead. */
export const updateWith: typeof factory.updateWithStatement = Debug.deprecate(factory.updateWithStatement, factoryDeprecation);

/** @deprecated Use `factory.createSwitchStatement` or the factory supplied by your transformation context instead. */
export const createSwitch: typeof factory.createSwitchStatement = Debug.deprecate(factory.createSwitchStatement, factoryDeprecation);

/** @deprecated Use `factory.updateSwitchStatement` or the factory supplied by your transformation context instead. */
export const updateSwitch: typeof factory.updateSwitchStatement = Debug.deprecate(factory.updateSwitchStatement, factoryDeprecation);

/** @deprecated Use `factory.createLabelStatement` or the factory supplied by your transformation context instead. */
export const createLabel: typeof factory.createLabeledStatement = Debug.deprecate(factory.createLabeledStatement, factoryDeprecation);

/** @deprecated Use `factory.updateLabelStatement` or the factory supplied by your transformation context instead. */
export const updateLabel: typeof factory.updateLabeledStatement = Debug.deprecate(factory.updateLabeledStatement, factoryDeprecation);

/** @deprecated Use `factory.createThrowStatement` or the factory supplied by your transformation context instead. */
export const createThrow: typeof factory.createThrowStatement = Debug.deprecate(factory.createThrowStatement, factoryDeprecation);

/** @deprecated Use `factory.updateThrowStatement` or the factory supplied by your transformation context instead. */
export const updateThrow: typeof factory.updateThrowStatement = Debug.deprecate(factory.updateThrowStatement, factoryDeprecation);

/** @deprecated Use `factory.createTryStatement` or the factory supplied by your transformation context instead. */
export const createTry: typeof factory.createTryStatement = Debug.deprecate(factory.createTryStatement, factoryDeprecation);

/** @deprecated Use `factory.updateTryStatement` or the factory supplied by your transformation context instead. */
export const updateTry: typeof factory.updateTryStatement = Debug.deprecate(factory.updateTryStatement, factoryDeprecation);

/** @deprecated Use `factory.createDebuggerStatement` or the factory supplied by your transformation context instead. */
export const createDebuggerStatement: typeof factory.createDebuggerStatement = Debug.deprecate(factory.createDebuggerStatement, factoryDeprecation);

/** @deprecated Use `factory.createVariableDeclarationList` or the factory supplied by your transformation context instead. */
export const createVariableDeclarationList: typeof factory.createVariableDeclarationList = Debug.deprecate(factory.createVariableDeclarationList, factoryDeprecation);

/** @deprecated Use `factory.updateVariableDeclarationList` or the factory supplied by your transformation context instead. */
export const updateVariableDeclarationList: typeof factory.updateVariableDeclarationList = Debug.deprecate(factory.updateVariableDeclarationList, factoryDeprecation);

/** @deprecated Use `factory.createFunctionDeclaration` or the factory supplied by your transformation context instead. */
export const createFunctionDeclaration: typeof factory.createFunctionDeclaration = Debug.deprecate(factory.createFunctionDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateFunctionDeclaration` or the factory supplied by your transformation context instead. */
export const updateFunctionDeclaration: typeof factory.updateFunctionDeclaration = Debug.deprecate(factory.updateFunctionDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createClassDeclaration` or the factory supplied by your transformation context instead. */
export const createClassDeclaration: typeof factory.createClassDeclaration = Debug.deprecate(factory.createClassDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateClassDeclaration` or the factory supplied by your transformation context instead. */
export const updateClassDeclaration: typeof factory.updateClassDeclaration = Debug.deprecate(factory.updateClassDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createInterfaceDeclaration` or the factory supplied by your transformation context instead. */
export const createInterfaceDeclaration: typeof factory.createInterfaceDeclaration = Debug.deprecate(factory.createInterfaceDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateInterfaceDeclaration` or the factory supplied by your transformation context instead. */
export const updateInterfaceDeclaration: typeof factory.updateInterfaceDeclaration = Debug.deprecate(factory.updateInterfaceDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createTypeAliasDeclaration` or the factory supplied by your transformation context instead. */
export const createTypeAliasDeclaration: typeof factory.createTypeAliasDeclaration = Debug.deprecate(factory.createTypeAliasDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateTypeAliasDeclaration` or the factory supplied by your transformation context instead. */
export const updateTypeAliasDeclaration: typeof factory.updateTypeAliasDeclaration = Debug.deprecate(factory.updateTypeAliasDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createEnumDeclaration` or the factory supplied by your transformation context instead. */
export const createEnumDeclaration: typeof factory.createEnumDeclaration = Debug.deprecate(factory.createEnumDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateEnumDeclaration` or the factory supplied by your transformation context instead. */
export const updateEnumDeclaration: typeof factory.updateEnumDeclaration = Debug.deprecate(factory.updateEnumDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createModuleDeclaration` or the factory supplied by your transformation context instead. */
export const createModuleDeclaration: typeof factory.createModuleDeclaration = Debug.deprecate(factory.createModuleDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateModuleDeclaration` or the factory supplied by your transformation context instead. */
export const updateModuleDeclaration: typeof factory.updateModuleDeclaration = Debug.deprecate(factory.updateModuleDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createModuleBlock` or the factory supplied by your transformation context instead. */
export const createModuleBlock: typeof factory.createModuleBlock = Debug.deprecate(factory.createModuleBlock, factoryDeprecation);

/** @deprecated Use `factory.updateModuleBlock` or the factory supplied by your transformation context instead. */
export const updateModuleBlock: typeof factory.updateModuleBlock = Debug.deprecate(factory.updateModuleBlock, factoryDeprecation);

/** @deprecated Use `factory.createCaseBlock` or the factory supplied by your transformation context instead. */
export const createCaseBlock: typeof factory.createCaseBlock = Debug.deprecate(factory.createCaseBlock, factoryDeprecation);

/** @deprecated Use `factory.updateCaseBlock` or the factory supplied by your transformation context instead. */
export const updateCaseBlock: typeof factory.updateCaseBlock = Debug.deprecate(factory.updateCaseBlock, factoryDeprecation);

/** @deprecated Use `factory.createNamespaceExportDeclaration` or the factory supplied by your transformation context instead. */
export const createNamespaceExportDeclaration: typeof factory.createNamespaceExportDeclaration = Debug.deprecate(factory.createNamespaceExportDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateNamespaceExportDeclaration` or the factory supplied by your transformation context instead. */
export const updateNamespaceExportDeclaration: typeof factory.updateNamespaceExportDeclaration = Debug.deprecate(factory.updateNamespaceExportDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createImportEqualsDeclaration` or the factory supplied by your transformation context instead. */
export const createImportEqualsDeclaration: typeof factory.createImportEqualsDeclaration = Debug.deprecate(factory.createImportEqualsDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateImportEqualsDeclaration` or the factory supplied by your transformation context instead. */
export const updateImportEqualsDeclaration: typeof factory.updateImportEqualsDeclaration = Debug.deprecate(factory.updateImportEqualsDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createImportDeclaration` or the factory supplied by your transformation context instead. */
export const createImportDeclaration: typeof factory.createImportDeclaration = Debug.deprecate(factory.createImportDeclaration, factoryDeprecation);

/** @deprecated Use `factory.updateImportDeclaration` or the factory supplied by your transformation context instead. */
export const updateImportDeclaration: typeof factory.updateImportDeclaration = Debug.deprecate(factory.updateImportDeclaration, factoryDeprecation);

/** @deprecated Use `factory.createNamespaceImport` or the factory supplied by your transformation context instead. */
export const createNamespaceImport: typeof factory.createNamespaceImport = Debug.deprecate(factory.createNamespaceImport, factoryDeprecation);

/** @deprecated Use `factory.updateNamespaceImport` or the factory supplied by your transformation context instead. */
export const updateNamespaceImport: typeof factory.updateNamespaceImport = Debug.deprecate(factory.updateNamespaceImport, factoryDeprecation);

/** @deprecated Use `factory.createNamedImports` or the factory supplied by your transformation context instead. */
export const createNamedImports: typeof factory.createNamedImports = Debug.deprecate(factory.createNamedImports, factoryDeprecation);

/** @deprecated Use `factory.updateNamedImports` or the factory supplied by your transformation context instead. */
export const updateNamedImports: typeof factory.updateNamedImports = Debug.deprecate(factory.updateNamedImports, factoryDeprecation);

/** @deprecated Use `factory.createImportSpecifier` or the factory supplied by your transformation context instead. */
export const createImportSpecifier: typeof factory.createImportSpecifier = Debug.deprecate(factory.createImportSpecifier, factoryDeprecation);

/** @deprecated Use `factory.updateImportSpecifier` or the factory supplied by your transformation context instead. */
export const updateImportSpecifier: typeof factory.updateImportSpecifier = Debug.deprecate(factory.updateImportSpecifier, factoryDeprecation);

/** @deprecated Use `factory.createExportAssignment` or the factory supplied by your transformation context instead. */
export const createExportAssignment: typeof factory.createExportAssignment = Debug.deprecate(factory.createExportAssignment, factoryDeprecation);

/** @deprecated Use `factory.updateExportAssignment` or the factory supplied by your transformation context instead. */
export const updateExportAssignment: typeof factory.updateExportAssignment = Debug.deprecate(factory.updateExportAssignment, factoryDeprecation);

/** @deprecated Use `factory.createNamedExports` or the factory supplied by your transformation context instead. */
export const createNamedExports: typeof factory.createNamedExports = Debug.deprecate(factory.createNamedExports, factoryDeprecation);

/** @deprecated Use `factory.updateNamedExports` or the factory supplied by your transformation context instead. */
export const updateNamedExports: typeof factory.updateNamedExports = Debug.deprecate(factory.updateNamedExports, factoryDeprecation);

/** @deprecated Use `factory.createExportSpecifier` or the factory supplied by your transformation context instead. */
export const createExportSpecifier: typeof factory.createExportSpecifier = Debug.deprecate(factory.createExportSpecifier, factoryDeprecation);

/** @deprecated Use `factory.updateExportSpecifier` or the factory supplied by your transformation context instead. */
export const updateExportSpecifier: typeof factory.updateExportSpecifier = Debug.deprecate(factory.updateExportSpecifier, factoryDeprecation);

/** @deprecated Use `factory.createExternalModuleReference` or the factory supplied by your transformation context instead. */
export const createExternalModuleReference: typeof factory.createExternalModuleReference = Debug.deprecate(factory.createExternalModuleReference, factoryDeprecation);

/** @deprecated Use `factory.updateExternalModuleReference` or the factory supplied by your transformation context instead. */
export const updateExternalModuleReference: typeof factory.updateExternalModuleReference = Debug.deprecate(factory.updateExternalModuleReference, factoryDeprecation);

/** @deprecated Use `factory.createJSDocTypeExpression` or the factory supplied by your transformation context instead. */
export const createJSDocTypeExpression: typeof factory.createJSDocTypeExpression = Debug.deprecate(factory.createJSDocTypeExpression, factoryDeprecation);

/** @deprecated Use `factory.createJSDocTypeTag` or the factory supplied by your transformation context instead. */
export const createJSDocTypeTag: typeof factory.createJSDocTypeTag = Debug.deprecate(factory.createJSDocTypeTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocReturnTag` or the factory supplied by your transformation context instead. */
export const createJSDocReturnTag: typeof factory.createJSDocReturnTag = Debug.deprecate(factory.createJSDocReturnTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocThisTag` or the factory supplied by your transformation context instead. */
export const createJSDocThisTag: typeof factory.createJSDocThisTag = Debug.deprecate(factory.createJSDocThisTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocComment` or the factory supplied by your transformation context instead. */
export const createJSDocComment: typeof factory.createJSDocComment = Debug.deprecate(factory.createJSDocComment, factoryDeprecation);

/** @deprecated Use `factory.createJSDocParameterTag` or the factory supplied by your transformation context instead. */
export const createJSDocParameterTag: typeof factory.createJSDocParameterTag = Debug.deprecate(factory.createJSDocParameterTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocClassTag` or the factory supplied by your transformation context instead. */
export const createJSDocClassTag: typeof factory.createJSDocClassTag = Debug.deprecate(factory.createJSDocClassTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocAugmentsTag` or the factory supplied by your transformation context instead. */
export const createJSDocAugmentsTag: typeof factory.createJSDocAugmentsTag = Debug.deprecate(factory.createJSDocAugmentsTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocEnumTag` or the factory supplied by your transformation context instead. */
export const createJSDocEnumTag: typeof factory.createJSDocEnumTag = Debug.deprecate(factory.createJSDocEnumTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocTemplateTag` or the factory supplied by your transformation context instead. */
export const createJSDocTemplateTag: typeof factory.createJSDocTemplateTag = Debug.deprecate(factory.createJSDocTemplateTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocTypedefTag` or the factory supplied by your transformation context instead. */
export const createJSDocTypedefTag: typeof factory.createJSDocTypedefTag = Debug.deprecate(factory.createJSDocTypedefTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocCallbackTag` or the factory supplied by your transformation context instead. */
export const createJSDocCallbackTag: typeof factory.createJSDocCallbackTag = Debug.deprecate(factory.createJSDocCallbackTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocSignature` or the factory supplied by your transformation context instead. */
export const createJSDocSignature: typeof factory.createJSDocSignature = Debug.deprecate(factory.createJSDocSignature, factoryDeprecation);

/** @deprecated Use `factory.createJSDocPropertyTag` or the factory supplied by your transformation context instead. */
export const createJSDocPropertyTag: typeof factory.createJSDocPropertyTag = Debug.deprecate(factory.createJSDocPropertyTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocTypeLiteral` or the factory supplied by your transformation context instead. */
export const createJSDocTypeLiteral: typeof factory.createJSDocTypeLiteral = Debug.deprecate(factory.createJSDocTypeLiteral, factoryDeprecation);

/** @deprecated Use `factory.createJSDocImplementsTag` or the factory supplied by your transformation context instead. */
export const createJSDocImplementsTag: typeof factory.createJSDocImplementsTag = Debug.deprecate(factory.createJSDocImplementsTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocAuthorTag` or the factory supplied by your transformation context instead. */
export const createJSDocAuthorTag: typeof factory.createJSDocAuthorTag = Debug.deprecate(factory.createJSDocAuthorTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocPublicTag` or the factory supplied by your transformation context instead. */
export const createJSDocPublicTag: typeof factory.createJSDocPublicTag = Debug.deprecate(factory.createJSDocPublicTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocPrivateTag` or the factory supplied by your transformation context instead. */
export const createJSDocPrivateTag: typeof factory.createJSDocPrivateTag = Debug.deprecate(factory.createJSDocPrivateTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocProtectedTag` or the factory supplied by your transformation context instead. */
export const createJSDocProtectedTag: typeof factory.createJSDocProtectedTag = Debug.deprecate(factory.createJSDocProtectedTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocReadonlyTag` or the factory supplied by your transformation context instead. */
export const createJSDocReadonlyTag: typeof factory.createJSDocReadonlyTag = Debug.deprecate(factory.createJSDocReadonlyTag, factoryDeprecation);

/** @deprecated Use `factory.createJSDocUnknownTag` or the factory supplied by your transformation context instead. */
export const createJSDocTag: typeof factory.createJSDocUnknownTag = Debug.deprecate(factory.createJSDocUnknownTag, factoryDeprecation);

/** @deprecated Use `factory.createJsxElement` or the factory supplied by your transformation context instead. */
export const createJsxElement: typeof factory.createJsxElement = Debug.deprecate(factory.createJsxElement, factoryDeprecation);

/** @deprecated Use `factory.updateJsxElement` or the factory supplied by your transformation context instead. */
export const updateJsxElement: typeof factory.updateJsxElement = Debug.deprecate(factory.updateJsxElement, factoryDeprecation);

/** @deprecated Use `factory.createJsxSelfClosingElement` or the factory supplied by your transformation context instead. */
export const createJsxSelfClosingElement: typeof factory.createJsxSelfClosingElement = Debug.deprecate(factory.createJsxSelfClosingElement, factoryDeprecation);

/** @deprecated Use `factory.updateJsxSelfClosingElement` or the factory supplied by your transformation context instead. */
export const updateJsxSelfClosingElement: typeof factory.updateJsxSelfClosingElement = Debug.deprecate(factory.updateJsxSelfClosingElement, factoryDeprecation);

/** @deprecated Use `factory.createJsxOpeningElement` or the factory supplied by your transformation context instead. */
export const createJsxOpeningElement: typeof factory.createJsxOpeningElement = Debug.deprecate(factory.createJsxOpeningElement, factoryDeprecation);

/** @deprecated Use `factory.updateJsxOpeningElement` or the factory supplied by your transformation context instead. */
export const updateJsxOpeningElement: typeof factory.updateJsxOpeningElement = Debug.deprecate(factory.updateJsxOpeningElement, factoryDeprecation);

/** @deprecated Use `factory.createJsxClosingElement` or the factory supplied by your transformation context instead. */
export const createJsxClosingElement: typeof factory.createJsxClosingElement = Debug.deprecate(factory.createJsxClosingElement, factoryDeprecation);

/** @deprecated Use `factory.updateJsxClosingElement` or the factory supplied by your transformation context instead. */
export const updateJsxClosingElement: typeof factory.updateJsxClosingElement = Debug.deprecate(factory.updateJsxClosingElement, factoryDeprecation);

/** @deprecated Use `factory.createJsxFragment` or the factory supplied by your transformation context instead. */
export const createJsxFragment: typeof factory.createJsxFragment = Debug.deprecate(factory.createJsxFragment, factoryDeprecation);

/** @deprecated Use `factory.createJsxText` or the factory supplied by your transformation context instead. */
export const createJsxText: typeof factory.createJsxText = Debug.deprecate(factory.createJsxText, factoryDeprecation);

/** @deprecated Use `factory.updateJsxText` or the factory supplied by your transformation context instead. */
export const updateJsxText: typeof factory.updateJsxText = Debug.deprecate(factory.updateJsxText, factoryDeprecation);

/** @deprecated Use `factory.createJsxOpeningFragment` or the factory supplied by your transformation context instead. */
export const createJsxOpeningFragment: typeof factory.createJsxOpeningFragment = Debug.deprecate(factory.createJsxOpeningFragment, factoryDeprecation);

/** @deprecated Use `factory.createJsxJsxClosingFragment` or the factory supplied by your transformation context instead. */
export const createJsxJsxClosingFragment: typeof factory.createJsxJsxClosingFragment = Debug.deprecate(factory.createJsxJsxClosingFragment, factoryDeprecation);

/** @deprecated Use `factory.updateJsxFragment` or the factory supplied by your transformation context instead. */
export const updateJsxFragment: typeof factory.updateJsxFragment = Debug.deprecate(factory.updateJsxFragment, factoryDeprecation);

/** @deprecated Use `factory.createJsxAttribute` or the factory supplied by your transformation context instead. */
export const createJsxAttribute: typeof factory.createJsxAttribute = Debug.deprecate(factory.createJsxAttribute, factoryDeprecation);

/** @deprecated Use `factory.updateJsxAttribute` or the factory supplied by your transformation context instead. */
export const updateJsxAttribute: typeof factory.updateJsxAttribute = Debug.deprecate(factory.updateJsxAttribute, factoryDeprecation);

/** @deprecated Use `factory.createJsxAttributes` or the factory supplied by your transformation context instead. */
export const createJsxAttributes: typeof factory.createJsxAttributes = Debug.deprecate(factory.createJsxAttributes, factoryDeprecation);

/** @deprecated Use `factory.updateJsxAttributes` or the factory supplied by your transformation context instead. */
export const updateJsxAttributes: typeof factory.updateJsxAttributes = Debug.deprecate(factory.updateJsxAttributes, factoryDeprecation);

/** @deprecated Use `factory.createJsxSpreadAttribute` or the factory supplied by your transformation context instead. */
export const createJsxSpreadAttribute: typeof factory.createJsxSpreadAttribute = Debug.deprecate(factory.createJsxSpreadAttribute, factoryDeprecation);

/** @deprecated Use `factory.updateJsxSpreadAttribute` or the factory supplied by your transformation context instead. */
export const updateJsxSpreadAttribute: typeof factory.updateJsxSpreadAttribute = Debug.deprecate(factory.updateJsxSpreadAttribute, factoryDeprecation);

/** @deprecated Use `factory.createJsxExpression` or the factory supplied by your transformation context instead. */
export const createJsxExpression: typeof factory.createJsxExpression = Debug.deprecate(factory.createJsxExpression, factoryDeprecation);

/** @deprecated Use `factory.updateJsxExpression` or the factory supplied by your transformation context instead. */
export const updateJsxExpression: typeof factory.updateJsxExpression = Debug.deprecate(factory.updateJsxExpression, factoryDeprecation);

/** @deprecated Use `factory.createCaseClause` or the factory supplied by your transformation context instead. */
export const createCaseClause: typeof factory.createCaseClause = Debug.deprecate(factory.createCaseClause, factoryDeprecation);

/** @deprecated Use `factory.updateCaseClause` or the factory supplied by your transformation context instead. */
export const updateCaseClause: typeof factory.updateCaseClause = Debug.deprecate(factory.updateCaseClause, factoryDeprecation);

/** @deprecated Use `factory.createDefaultClause` or the factory supplied by your transformation context instead. */
export const createDefaultClause: typeof factory.createDefaultClause = Debug.deprecate(factory.createDefaultClause, factoryDeprecation);

/** @deprecated Use `factory.updateDefaultClause` or the factory supplied by your transformation context instead. */
export const updateDefaultClause: typeof factory.updateDefaultClause = Debug.deprecate(factory.updateDefaultClause, factoryDeprecation);

/** @deprecated Use `factory.createHeritageClause` or the factory supplied by your transformation context instead. */
export const createHeritageClause: typeof factory.createHeritageClause = Debug.deprecate(factory.createHeritageClause, factoryDeprecation);

/** @deprecated Use `factory.updateHeritageClause` or the factory supplied by your transformation context instead. */
export const updateHeritageClause: typeof factory.updateHeritageClause = Debug.deprecate(factory.updateHeritageClause, factoryDeprecation);

/** @deprecated Use `factory.createCatchClause` or the factory supplied by your transformation context instead. */
export const createCatchClause: typeof factory.createCatchClause = Debug.deprecate(factory.createCatchClause, factoryDeprecation);

/** @deprecated Use `factory.updateCatchClause` or the factory supplied by your transformation context instead. */
export const updateCatchClause: typeof factory.updateCatchClause = Debug.deprecate(factory.updateCatchClause, factoryDeprecation);

/** @deprecated Use `factory.createPropertyAssignment` or the factory supplied by your transformation context instead. */
export const createPropertyAssignment: typeof factory.createPropertyAssignment = Debug.deprecate(factory.createPropertyAssignment, factoryDeprecation);

/** @deprecated Use `factory.updatePropertyAssignment` or the factory supplied by your transformation context instead. */
export const updatePropertyAssignment: typeof factory.updatePropertyAssignment = Debug.deprecate(factory.updatePropertyAssignment, factoryDeprecation);

/** @deprecated Use `factory.createShorthandPropertyAssignment` or the factory supplied by your transformation context instead. */
export const createShorthandPropertyAssignment: typeof factory.createShorthandPropertyAssignment = Debug.deprecate(factory.createShorthandPropertyAssignment, factoryDeprecation);

/** @deprecated Use `factory.updateShorthandPropertyAssignment` or the factory supplied by your transformation context instead. */
export const updateShorthandPropertyAssignment: typeof factory.updateShorthandPropertyAssignment = Debug.deprecate(factory.updateShorthandPropertyAssignment, factoryDeprecation);

/** @deprecated Use `factory.createSpreadAssignment` or the factory supplied by your transformation context instead. */
export const createSpreadAssignment: typeof factory.createSpreadAssignment = Debug.deprecate(factory.createSpreadAssignment, factoryDeprecation);

/** @deprecated Use `factory.updateSpreadAssignment` or the factory supplied by your transformation context instead. */
export const updateSpreadAssignment: typeof factory.updateSpreadAssignment = Debug.deprecate(factory.updateSpreadAssignment, factoryDeprecation);

/** @deprecated Use `factory.createEnumMember` or the factory supplied by your transformation context instead. */
export const createEnumMember: typeof factory.createEnumMember = Debug.deprecate(factory.createEnumMember, factoryDeprecation);

/** @deprecated Use `factory.updateEnumMember` or the factory supplied by your transformation context instead. */
export const updateEnumMember: typeof factory.updateEnumMember = Debug.deprecate(factory.updateEnumMember, factoryDeprecation);

/** @deprecated Use `factory.updateSourceFile` or the factory supplied by your transformation context instead. */
export const updateSourceFileNode: typeof factory.updateSourceFile = Debug.deprecate(factory.updateSourceFile, factoryDeprecation);

/** @deprecated Use `factory.createNotEmittedStatement` or the factory supplied by your transformation context instead. */
export const createNotEmittedStatement: typeof factory.createNotEmittedStatement = Debug.deprecate(factory.createNotEmittedStatement, factoryDeprecation);

/** @deprecated Use `factory.createPartiallyEmittedExpression` or the factory supplied by your transformation context instead. */
export const createPartiallyEmittedExpression: typeof factory.createPartiallyEmittedExpression = Debug.deprecate(factory.createPartiallyEmittedExpression, factoryDeprecation);

/** @deprecated Use `factory.updatePartiallyEmittedExpression` or the factory supplied by your transformation context instead. */
export const updatePartiallyEmittedExpression: typeof factory.updatePartiallyEmittedExpression = Debug.deprecate(factory.updatePartiallyEmittedExpression, factoryDeprecation);

/** @deprecated Use `factory.createCommaListExpression` or the factory supplied by your transformation context instead. */
export const createCommaList: typeof factory.createCommaListExpression = Debug.deprecate(factory.createCommaListExpression, factoryDeprecation);

/** @deprecated Use `factory.updateCommaListExpression` or the factory supplied by your transformation context instead. */
export const updateCommaList: typeof factory.updateCommaListExpression = Debug.deprecate(factory.updateCommaListExpression, factoryDeprecation);

/** @deprecated Use `factory.createBundle` or the factory supplied by your transformation context instead. */
export const createBundle: typeof factory.createBundle = Debug.deprecate(factory.createBundle, factoryDeprecation);

/** @deprecated Use `factory.updateBundle` or the factory supplied by your transformation context instead. */
export const updateBundle: typeof factory.updateBundle = Debug.deprecate(factory.updateBundle, factoryDeprecation);

/** @deprecated Use `factory.createImmediatelyInvokedFunctionExpression` or the factory supplied by your transformation context instead. */
export const createImmediatelyInvokedFunctionExpression: typeof factory.createImmediatelyInvokedFunctionExpression = Debug.deprecate(factory.createImmediatelyInvokedFunctionExpression, factoryDeprecation);

/** @deprecated Use `factory.createImmediatelyInvokedArrowFunction` or the factory supplied by your transformation context instead. */
export const createImmediatelyInvokedArrowFunction: typeof factory.createImmediatelyInvokedArrowFunction = Debug.deprecate(factory.createImmediatelyInvokedArrowFunction, factoryDeprecation);

/** @deprecated Use `factory.createVoidZero` or the factory supplied by your transformation context instead. */
export const createVoidZero: typeof factory.createVoidZero = Debug.deprecate(factory.createVoidZero, factoryDeprecation);

/** @deprecated Use `factory.createExportDefault` or the factory supplied by your transformation context instead. */
export const createExportDefault: typeof factory.createExportDefault = Debug.deprecate(factory.createExportDefault, factoryDeprecation);

/** @deprecated Use `factory.createExternalModuleExport` or the factory supplied by your transformation context instead. */
export const createExternalModuleExport: typeof factory.createExternalModuleExport = Debug.deprecate(factory.createExternalModuleExport, factoryDeprecation);

/** @deprecated Use `factory.createNamespaceExport` or the factory supplied by your transformation context instead. */
export const createNamespaceExport: typeof factory.createNamespaceExport = Debug.deprecate(factory.createNamespaceExport, factoryDeprecation);

/** @deprecated Use `factory.updateNamespaceExport` or the factory supplied by your transformation context instead. */
export const updateNamespaceExport: typeof factory.updateNamespaceExport = Debug.deprecate(factory.updateNamespaceExport, factoryDeprecation);

/** @deprecated Use `factory.createToken` or the factory supplied by your transformation context instead. */
export const createToken = Debug.deprecate(function createToken<TKind extends SyntaxKind>(kind: TKind): Token<TKind> {
    return factory.createToken(kind);
}, factoryDeprecation);

/** @deprecated Use `factory.createIdentifier` or the factory supplied by your transformation context instead. */
export const createIdentifier = Debug.deprecate(function createIdentifier(text: string) {
    return factory.createIdentifier(text, /*typeArguments*/ undefined, /*originalKeywordKind*/ undefined);
}, factoryDeprecation);

/** @deprecated Use `factory.createTempVariable` or the factory supplied by your transformation context instead. */
export const createTempVariable = Debug.deprecate(function createTempVariable(recordTempVariable: ((node: Identifier) => void) | undefined): Identifier {
    return factory.createTempVariable(recordTempVariable, /*reserveInNestedScopes*/ undefined);
}, factoryDeprecation);

/** @deprecated Use `factory.getGeneratedNameForNode` or the factory supplied by your transformation context instead. */
export const getGeneratedNameForNode = Debug.deprecate(function getGeneratedNameForNode(node: Node | undefined): Identifier {
    return factory.getGeneratedNameForNode(node, /*flags*/ undefined);
}, factoryDeprecation);

/** @deprecated Use `factory.createUniqueName(text, GeneratedIdentifierFlags.Optimistic)` or the factory supplied by your transformation context instead. */
export const createOptimisticUniqueName = Debug.deprecate(function createOptimisticUniqueName(text: string): Identifier {
    return factory.createUniqueName(text, GeneratedIdentifierFlags.Optimistic);
}, factoryDeprecation);

/** @deprecated Use `factory.createUniqueName(text, GeneratedIdentifierFlags.Optimistic | GeneratedIdentifierFlags.FileLevel)` or the factory supplied by your transformation context instead. */
export const createFileLevelUniqueName = Debug.deprecate(function createFileLevelUniqueName(text: string): Identifier {
    return factory.createUniqueName(text, GeneratedIdentifierFlags.Optimistic | GeneratedIdentifierFlags.FileLevel);
}, factoryDeprecation);

/** @deprecated Use `factory.createIndexSignature` or the factory supplied by your transformation context instead. */
export const createIndexSignature = Debug.deprecate(function createIndexSignature(decorators: readonly Decorator[] | undefined, modifiers: readonly Modifier[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode): IndexSignatureDeclaration {
    return factory.createIndexSignature(decorators, modifiers, parameters, type);
}, factoryDeprecation);

/** @deprecated Use `factory.createTypePredicateNode` or the factory supplied by your transformation context instead. */
export const createTypePredicateNode = Debug.deprecate(function createTypePredicateNode(parameterName: Identifier | ThisTypeNode | string, type: TypeNode): TypePredicateNode {
    return factory.createTypePredicateNode(/*assertsModifier*/ undefined, parameterName, type);
}, factoryDeprecation);

/** @deprecated Use `factory.updateTypePredicateNode` or the factory supplied by your transformation context instead. */
export const updateTypePredicateNode = Debug.deprecate(function updateTypePredicateNode(node: TypePredicateNode, parameterName: Identifier | ThisTypeNode, type: TypeNode): TypePredicateNode {
    return factory.updateTypePredicateNode(node, /*assertsModifier*/ undefined, parameterName, type);
}, factoryDeprecation);

/** @deprecated Use `factory.createStringLiteral`, `factory.createStringLiteralFromNode`, `factory.createNumericLiteral`, `factory.createBigIntLiteral`, `factory.createTrue`, `factory.createFalse`, or the factory supplied by your transformation context instead. */
export const createLiteral = Debug.deprecate(function createLiteral(value: string | number | PseudoBigInt | boolean | StringLiteral | NoSubstitutionTemplateLiteral | NumericLiteral | Identifier): PrimaryExpression {
    if (typeof value === "number") {
        return factory.createNumericLiteral(value);
    }
    // eslint-disable-next-line local/no-in-operator
    if (typeof value === "object" && "base10Value" in value) { // PseudoBigInt
        return factory.createBigIntLiteral(value);
    }
    if (typeof value === "boolean") {
        return value ? factory.createTrue() : factory.createFalse();
    }
    if (typeof value === "string") {
        return factory.createStringLiteral(value, /*isSingleQuote*/ undefined);
    }
    return factory.createStringLiteralFromNode(value);
} as {
    (value: string | StringLiteral | NoSubstitutionTemplateLiteral | NumericLiteral | Identifier): StringLiteral;
    (value: number | PseudoBigInt): NumericLiteral;
    (value: boolean): BooleanLiteral;
    (value: string | number | PseudoBigInt | boolean): PrimaryExpression;
}, { since: "4.0", warnAfter: "4.1", message: "Use `factory.createStringLiteral`, `factory.createStringLiteralFromNode`, `factory.createNumericLiteral`, `factory.createBigIntLiteral`, `factory.createTrue`, `factory.createFalse`, or the factory supplied by your transformation context instead." });

/** @deprecated Use `factory.createMethodSignature` or the factory supplied by your transformation context instead. */
export const createMethodSignature = Debug.deprecate(function createMethodSignature(
    typeParameters: readonly TypeParameterDeclaration[] | undefined,
    parameters: readonly ParameterDeclaration[],
    type: TypeNode | undefined,
    name: string | PropertyName,
    questionToken: QuestionToken | undefined
) {
    return factory.createMethodSignature(/*modifiers*/ undefined, name, questionToken, typeParameters, parameters, type);
}, factoryDeprecation);

/** @deprecated Use `factory.updateMethodSignature` or the factory supplied by your transformation context instead. */
export const updateMethodSignature = Debug.deprecate(function updateMethodSignature(
    node: MethodSignature,
    typeParameters: NodeArray<TypeParameterDeclaration> | undefined,
    parameters: NodeArray<ParameterDeclaration>,
    type: TypeNode | undefined,
    name: PropertyName,
    questionToken: QuestionToken | undefined
) {
    return factory.updateMethodSignature(node, node.modifiers, name, questionToken, typeParameters, parameters, type);
}, factoryDeprecation);

/** @deprecated Use `factory.createTypeOperatorNode` or the factory supplied by your transformation context instead. */
export const createTypeOperatorNode = Debug.deprecate(function createTypeOperatorNode(operatorOrType: SyntaxKind.KeyOfKeyword | SyntaxKind.UniqueKeyword | SyntaxKind.ReadonlyKeyword | TypeNode, type?: TypeNode) {
    let operator: TypeOperatorNode["operator"];
    if (type) {
        operator = operatorOrType as TypeOperatorNode["operator"];
    }
    else {
        type = operatorOrType as TypeNode;
        operator = SyntaxKind.KeyOfKeyword;
    }
    return factory.createTypeOperatorNode(operator, type);
} as {
    (type: TypeNode): TypeOperatorNode;
    (operator: SyntaxKind.KeyOfKeyword | SyntaxKind.UniqueKeyword | SyntaxKind.ReadonlyKeyword, type: TypeNode): TypeOperatorNode;
}, factoryDeprecation);

/** @deprecated Use `factory.createTaggedTemplate` or the factory supplied by your transformation context instead. */
export const createTaggedTemplate = Debug.deprecate(function createTaggedTemplate(tag: Expression, typeArgumentsOrTemplate: readonly TypeNode[] | TemplateLiteral | undefined, template?: TemplateLiteral) {
    let typeArguments: readonly TypeNode[] | undefined;
    if (template) {
        typeArguments = typeArgumentsOrTemplate as readonly TypeNode[] | undefined;
    }
    else {
        template = typeArgumentsOrTemplate as TemplateLiteral;
    }
    return factory.createTaggedTemplateExpression(tag, typeArguments, template);
} as {
    (tag: Expression, template: TemplateLiteral): TaggedTemplateExpression;
    (tag: Expression, typeArguments: readonly TypeNode[] | undefined, template: TemplateLiteral): TaggedTemplateExpression;
}, factoryDeprecation);

/** @deprecated Use `factory.updateTaggedTemplate` or the factory supplied by your transformation context instead. */
export const updateTaggedTemplate = Debug.deprecate(function updateTaggedTemplate(node: TaggedTemplateExpression, tag: Expression, typeArgumentsOrTemplate: readonly TypeNode[] | TemplateLiteral | undefined, template?: TemplateLiteral) {
    let typeArguments: readonly TypeNode[] | undefined;
    if (template) {
        typeArguments = typeArgumentsOrTemplate as readonly TypeNode[] | undefined;
    }
    else {
        template = typeArgumentsOrTemplate as TemplateLiteral;
    }
    return factory.updateTaggedTemplateExpression(node, tag, typeArguments, template);
} as {
    (node: TaggedTemplateExpression, tag: Expression, template: TemplateLiteral): TaggedTemplateExpression;
    (node: TaggedTemplateExpression, tag: Expression, typeArguments: readonly TypeNode[] | undefined, template: TemplateLiteral): TaggedTemplateExpression;
}, factoryDeprecation);

/** @deprecated Use `factory.updateBinary` or the factory supplied by your transformation context instead. */
export const updateBinary = Debug.deprecate(function updateBinary(node: BinaryExpression, left: Expression, right: Expression, operator: BinaryOperator | BinaryOperatorToken = node.operatorToken) {
    if (typeof operator === "number") {
        operator = operator === node.operatorToken.kind ? node.operatorToken : factory.createToken(operator);
    }
    return factory.updateBinaryExpression(node, left, operator, right);
}, factoryDeprecation);

/** @deprecated Use `factory.createConditional` or the factory supplied by your transformation context instead. */
export const createConditional = Debug.deprecate(function createConditional(condition: Expression, questionTokenOrWhenTrue: QuestionToken | Expression, whenTrueOrWhenFalse: Expression, colonToken?: ColonToken, whenFalse?: Expression) {
    return arguments.length === 5 ? factory.createConditionalExpression(condition, questionTokenOrWhenTrue as QuestionToken, whenTrueOrWhenFalse, colonToken, whenFalse!) :
        arguments.length === 3 ? factory.createConditionalExpression(condition, factory.createToken(SyntaxKind.QuestionToken), questionTokenOrWhenTrue as Expression, factory.createToken(SyntaxKind.ColonToken), whenTrueOrWhenFalse) :
        Debug.fail("Argument count mismatch");
} as {
    (condition: Expression, whenTrue: Expression, whenFalse: Expression): ConditionalExpression;
    (condition: Expression, questionToken: QuestionToken, whenTrue: Expression, colonToken: ColonToken, whenFalse: Expression): ConditionalExpression;
}, factoryDeprecation);

/** @deprecated Use `factory.createYield` or the factory supplied by your transformation context instead. */
export const createYield = Debug.deprecate(function createYield(asteriskTokenOrExpression?: AsteriskToken | Expression | undefined, expression?: Expression) {
    let asteriskToken: AsteriskToken | undefined;
    if (expression) {
        asteriskToken = asteriskTokenOrExpression as AsteriskToken;
    }
    else {
        expression = asteriskTokenOrExpression as Expression;
    }
    return factory.createYieldExpression(asteriskToken, expression);
} as {
    (expression?: Expression | undefined): YieldExpression;
    (asteriskToken: AsteriskToken | undefined, expression: Expression): YieldExpression;
}, factoryDeprecation);

/** @deprecated Use `factory.createClassExpression` or the factory supplied by your transformation context instead. */
export const createClassExpression = Debug.deprecate(function createClassExpression(
    modifiers: readonly Modifier[] | undefined,
    name: string | Identifier | undefined,
    typeParameters: readonly TypeParameterDeclaration[] | undefined,
    heritageClauses: readonly HeritageClause[] | undefined,
    members: readonly ClassElement[]
) {
    return factory.createClassExpression(/*decorators*/ undefined, modifiers, name, typeParameters, heritageClauses, members);
}, factoryDeprecation);

/** @deprecated Use `factory.updateClassExpression` or the factory supplied by your transformation context instead. */
export const updateClassExpression = Debug.deprecate(function updateClassExpression(
    node: ClassExpression,
    modifiers: readonly Modifier[] | undefined,
    name: Identifier | undefined,
    typeParameters: readonly TypeParameterDeclaration[] | undefined,
    heritageClauses: readonly HeritageClause[] | undefined,
    members: readonly ClassElement[]
) {
    return factory.updateClassExpression(node, /*decorators*/ undefined, modifiers, name, typeParameters, heritageClauses, members);
}, factoryDeprecation);

/** @deprecated Use `factory.createPropertySignature` or the factory supplied by your transformation context instead. */
export const createPropertySignature = Debug.deprecate(function createPropertySignature(
    modifiers: readonly Modifier[] | undefined,
    name: PropertyName | string,
    questionToken: QuestionToken | undefined,
    type: TypeNode | undefined,
    initializer?: Expression | undefined
): PropertySignature {
    const node = factory.createPropertySignature(modifiers, name, questionToken, type);
    (node as Mutable<PropertySignature>).initializer = initializer;
    return node;
}, factoryDeprecation);

/** @deprecated Use `factory.updatePropertySignature` or the factory supplied by your transformation context instead. */
export const updatePropertySignature = Debug.deprecate(function updatePropertySignature(
    node: PropertySignature,
    modifiers: readonly Modifier[] | undefined,
    name: PropertyName,
    questionToken: QuestionToken | undefined,
    type: TypeNode | undefined,
    initializer: Expression | undefined
) {
    let updated = factory.updatePropertySignature(node, modifiers, name, questionToken, type);
    if (node.initializer !== initializer) {
        if (updated === node) {
            updated = factory.cloneNode(node);
        }
        (updated as Mutable<PropertySignature>).initializer = initializer;
    }
    return updated;
}, factoryDeprecation);

/** @deprecated Use `factory.createExpressionWithTypeArguments` or the factory supplied by your transformation context instead. */
export const createExpressionWithTypeArguments = Debug.deprecate(function createExpressionWithTypeArguments(typeArguments: readonly TypeNode[] | undefined, expression: Expression) {
    return factory.createExpressionWithTypeArguments(expression, typeArguments);
}, factoryDeprecation);

/** @deprecated Use `factory.updateExpressionWithTypeArguments` or the factory supplied by your transformation context instead. */
export const updateExpressionWithTypeArguments = Debug.deprecate(function updateExpressionWithTypeArguments(node: ExpressionWithTypeArguments, typeArguments: readonly TypeNode[] | undefined, expression: Expression) {
    return factory.updateExpressionWithTypeArguments(node, expression, typeArguments);
}, factoryDeprecation);

/** @deprecated Use `factory.createArrowFunction` or the factory supplied by your transformation context instead. */
export const createArrowFunction = Debug.deprecate(function createArrowFunction(modifiers: readonly Modifier[] | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, equalsGreaterThanTokenOrBody: ConciseBody | EqualsGreaterThanToken | undefined, body?: ConciseBody) {
    return arguments.length === 6 ? factory.createArrowFunction(modifiers, typeParameters, parameters, type, equalsGreaterThanTokenOrBody as EqualsGreaterThanToken | undefined, body!) :
        arguments.length === 5 ? factory.createArrowFunction(modifiers, typeParameters, parameters, type, /*equalsGreaterThanToken*/ undefined, equalsGreaterThanTokenOrBody as ConciseBody) :
        Debug.fail("Argument count mismatch");
} as {
    (modifiers: readonly Modifier[] | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, equalsGreaterThanToken: EqualsGreaterThanToken | undefined, body: ConciseBody): ArrowFunction;
    (modifiers: readonly Modifier[] | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, body: ConciseBody): ArrowFunction;
}, factoryDeprecation);

/** @deprecated Use `factory.updateArrowFunction` or the factory supplied by your transformation context instead. */
export const updateArrowFunction = Debug.deprecate(function updateArrowFunction(node: ArrowFunction, modifiers: readonly Modifier[] | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, equalsGreaterThanTokenOrBody: EqualsGreaterThanToken | ConciseBody, body?: ConciseBody) {
    return arguments.length === 7 ? factory.updateArrowFunction(node, modifiers, typeParameters, parameters, type, equalsGreaterThanTokenOrBody as EqualsGreaterThanToken, body!) :
        arguments.length === 6 ? factory.updateArrowFunction(node, modifiers, typeParameters, parameters, type, node.equalsGreaterThanToken, equalsGreaterThanTokenOrBody as ConciseBody) :
        Debug.fail("Argument count mismatch");
} as {
    (node: ArrowFunction, modifiers: readonly Modifier[] | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, equalsGreaterThanToken: EqualsGreaterThanToken, body: ConciseBody): ArrowFunction;
    (node: ArrowFunction, modifiers: readonly Modifier[] | undefined, typeParameters: readonly TypeParameterDeclaration[] | undefined, parameters: readonly ParameterDeclaration[], type: TypeNode | undefined, body: ConciseBody): ArrowFunction;
}, factoryDeprecation);

/** @deprecated Use `factory.createVariableDeclaration` or the factory supplied by your transformation context instead. */
export const createVariableDeclaration = Debug.deprecate(function createVariableDeclaration(name: string | BindingName, exclamationTokenOrType?: ExclamationToken | TypeNode, typeOrInitializer?: TypeNode | Expression, initializer?: Expression) {
    return arguments.length === 4 ? factory.createVariableDeclaration(name, exclamationTokenOrType as ExclamationToken | undefined, typeOrInitializer as TypeNode | undefined, initializer) :
        arguments.length >= 1 && arguments.length <= 3 ? factory.createVariableDeclaration(name, /*exclamationToken*/ undefined, exclamationTokenOrType as TypeNode | undefined, typeOrInitializer as Expression | undefined) :
        Debug.fail("Argument count mismatch");
} as {
    (name: string | BindingName, type?: TypeNode | undefined, initializer?: Expression | undefined): VariableDeclaration;
    (name: string | BindingName, exclamationToken: ExclamationToken | undefined, type: TypeNode | undefined, initializer: Expression | undefined): VariableDeclaration;
}, factoryDeprecation);

/** @deprecated Use `factory.updateVariableDeclaration` or the factory supplied by your transformation context instead. */
export const updateVariableDeclaration = Debug.deprecate(function updateVariableDeclaration(node: VariableDeclaration, name: BindingName, exclamationTokenOrType: ExclamationToken | TypeNode | undefined, typeOrInitializer: TypeNode | Expression | undefined, initializer?: Expression | undefined) {
    return arguments.length === 5 ? factory.updateVariableDeclaration(node, name, exclamationTokenOrType as ExclamationToken | undefined, typeOrInitializer as TypeNode | undefined, initializer) :
        arguments.length === 4 ? factory.updateVariableDeclaration(node, name, node.exclamationToken, exclamationTokenOrType as TypeNode | undefined, typeOrInitializer as Expression | undefined) :
        Debug.fail("Argument count mismatch");
} as {
    (node: VariableDeclaration, name: BindingName, type: TypeNode | undefined, initializer: Expression | undefined): VariableDeclaration;
    (node: VariableDeclaration, name: BindingName, exclamationToken: ExclamationToken | undefined, type: TypeNode | undefined, initializer: Expression | undefined): VariableDeclaration;
}, factoryDeprecation);

/** @deprecated Use `factory.createImportClause` or the factory supplied by your transformation context instead. */
export const createImportClause = Debug.deprecate(function createImportClause(name: Identifier | undefined, namedBindings: NamedImportBindings | undefined, isTypeOnly = false): ImportClause {
    return factory.createImportClause(isTypeOnly, name, namedBindings);
}, factoryDeprecation);

/** @deprecated Use `factory.updateImportClause` or the factory supplied by your transformation context instead. */
export const updateImportClause = Debug.deprecate(function updateImportClause(node: ImportClause, name: Identifier | undefined, namedBindings: NamedImportBindings | undefined, isTypeOnly: boolean) {
    return factory.updateImportClause(node, isTypeOnly, name, namedBindings);
}, factoryDeprecation);

/** @deprecated Use `factory.createExportDeclaration` or the factory supplied by your transformation context instead. */
export const createExportDeclaration = Debug.deprecate(function createExportDeclaration(decorators: readonly Decorator[] | undefined, modifiers: readonly Modifier[] | undefined, exportClause: NamedExportBindings | undefined, moduleSpecifier?: Expression | undefined, isTypeOnly = false) {
    return factory.createExportDeclaration(decorators, modifiers, isTypeOnly, exportClause, moduleSpecifier);
}, factoryDeprecation);

/** @deprecated Use `factory.updateExportDeclaration` or the factory supplied by your transformation context instead. */
export const updateExportDeclaration = Debug.deprecate(function updateExportDeclaration(
    node: ExportDeclaration,
    decorators: readonly Decorator[] | undefined,
    modifiers: readonly Modifier[] | undefined,
    exportClause: NamedExportBindings | undefined,
    moduleSpecifier: Expression | undefined,
    isTypeOnly: boolean) {
    return factory.updateExportDeclaration(node, decorators, modifiers, isTypeOnly, exportClause, moduleSpecifier, node.assertClause);
}, factoryDeprecation);

/** @deprecated Use `factory.createJSDocParameterTag` or the factory supplied by your transformation context instead. */
export const createJSDocParamTag = Debug.deprecate(function createJSDocParamTag(name: EntityName, isBracketed: boolean, typeExpression?: JSDocTypeExpression | undefined, comment?: string | undefined): JSDocParameterTag {
    return factory.createJSDocParameterTag(/*tagName*/ undefined, name, isBracketed, typeExpression, /*isNameFirst*/ false, comment ? factory.createNodeArray([factory.createJSDocText(comment)]) : undefined);
}, factoryDeprecation);

/** @deprecated Use `factory.createComma` or the factory supplied by your transformation context instead. */
export const createComma = Debug.deprecate(function createComma(left: Expression, right: Expression): Expression {
    return factory.createComma(left, right);
}, factoryDeprecation);

/** @deprecated Use `factory.createLessThan` or the factory supplied by your transformation context instead. */
export const createLessThan = Debug.deprecate(function createLessThan(left: Expression, right: Expression): Expression {
    return factory.createLessThan(left, right);
}, factoryDeprecation);

/** @deprecated Use `factory.createAssignment` or the factory supplied by your transformation context instead. */
export const createAssignment = Debug.deprecate(function createAssignment(left: Expression, right: Expression): BinaryExpression {
    return factory.createAssignment(left, right);
}, factoryDeprecation);

/** @deprecated Use `factory.createStrictEquality` or the factory supplied by your transformation context instead. */
export const createStrictEquality = Debug.deprecate(function createStrictEquality(left: Expression, right: Expression): BinaryExpression {
    return factory.createStrictEquality(left, right);
}, factoryDeprecation);

/** @deprecated Use `factory.createStrictInequality` or the factory supplied by your transformation context instead. */
export const createStrictInequality = Debug.deprecate(function createStrictInequality(left: Expression, right: Expression): BinaryExpression {
    return factory.createStrictInequality(left, right);
}, factoryDeprecation);

/** @deprecated Use `factory.createAdd` or the factory supplied by your transformation context instead. */
export const createAdd = Debug.deprecate(function createAdd(left: Expression, right: Expression): BinaryExpression {
    return factory.createAdd(left, right);
}, factoryDeprecation);

/** @deprecated Use `factory.createSubtract` or the factory supplied by your transformation context instead. */
export const createSubtract = Debug.deprecate(function createSubtract(left: Expression, right: Expression): BinaryExpression {
    return factory.createSubtract(left, right);
}, factoryDeprecation);

/** @deprecated Use `factory.createLogicalAnd` or the factory supplied by your transformation context instead. */
export const createLogicalAnd = Debug.deprecate(function createLogicalAnd(left: Expression, right: Expression): BinaryExpression {
    return factory.createLogicalAnd(left, right);
}, factoryDeprecation);

/** @deprecated Use `factory.createLogicalOr` or the factory supplied by your transformation context instead. */
export const createLogicalOr = Debug.deprecate(function createLogicalOr(left: Expression, right: Expression): BinaryExpression {
    return factory.createLogicalOr(left, right);
}, factoryDeprecation);

/** @deprecated Use `factory.createPostfixIncrement` or the factory supplied by your transformation context instead. */
export const createPostfixIncrement = Debug.deprecate(function createPostfixIncrement(operand: Expression): PostfixUnaryExpression {
    return factory.createPostfixIncrement(operand);
}, factoryDeprecation);

/** @deprecated Use `factory.createLogicalNot` or the factory supplied by your transformation context instead. */
export const createLogicalNot = Debug.deprecate(function createLogicalNot(operand: Expression): PrefixUnaryExpression {
    return factory.createLogicalNot(operand);
}, factoryDeprecation);

/** @deprecated Use an appropriate `factory` method instead. */
export const createNode = Debug.deprecate(function createNode(kind: SyntaxKind, pos = 0, end = 0): Node {
    return setTextRangePosEnd(
        kind === SyntaxKind.SourceFile ? parseBaseNodeFactory.createBaseSourceFileNode(kind) :
        kind === SyntaxKind.Identifier ? parseBaseNodeFactory.createBaseIdentifierNode(kind) :
        kind === SyntaxKind.PrivateIdentifier ? parseBaseNodeFactory.createBasePrivateIdentifierNode(kind) :
        !isNodeKind(kind) ? parseBaseNodeFactory.createBaseTokenNode(kind) :
        parseBaseNodeFactory.createBaseNode(kind),
        pos,
        end
    );
}, { since: "4.0", warnAfter: "4.1", message: "Use an appropriate `factory` method instead." });

/**
 * Creates a shallow, memberwise clone of a node ~for mutation~ with its `pos`, `end`, and `parent` set.
 *
 * NOTE: It is unsafe to change any properties of a `Node` that relate to its AST children, as those changes won't be
 * captured with respect to transformations.
 *
 * @deprecated Use an appropriate `factory.update...` method instead, use `setCommentRange` or `setSourceMapRange`, and avoid setting `parent`.
 */
export const getMutableClone = Debug.deprecate(function getMutableClone<T extends Node>(node: T): T {
    const clone = factory.cloneNode(node);
    setTextRange(clone, node);
    setParent(clone, node.parent);
    return clone;
}, { since: "4.0", warnAfter: "4.1", message: "Use an appropriate `factory.update...` method instead, use `setCommentRange` or `setSourceMapRange`, and avoid setting `parent`." });
