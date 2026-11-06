package checker

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
)

func (c *Checker) GetStringType() *Type {
	return c.stringType
}

func (c *Checker) GetNumberType() *Type {
	return c.numberType
}

func (c *Checker) GetBooleanType() *Type {
	return c.booleanType
}

func (c *Checker) GetVoidType() *Type {
	return c.voidType
}

func (c *Checker) GetUndefinedType() *Type {
	return c.undefinedType
}

func (c *Checker) GetNullType() *Type {
	return c.nullType
}

func (c *Checker) GetAnyType() *Type {
	return c.anyType
}

func (c *Checker) GetErrorType() *Type {
	return c.errorType
}

func (c *Checker) GetNeverType() *Type {
	return c.neverType
}

func (c *Checker) GetUnknownType() *Type {
	return c.unknownType
}

func (c *Checker) GetBigIntType() *Type {
	return c.bigintType
}

func (c *Checker) GetESSymbolType() *Type {
	return c.esSymbolType
}

func (c *Checker) GetNonPrimitiveType() *Type {
	return c.nonPrimitiveType
}

func (c *Checker) GetBaseTypeOfLiteralType(t *Type) *Type {
	return c.getBaseTypeOfLiteralType(t)
}

func (c *Checker) getUnknownSymbol() ast.SymbolId {
	return c.unknownSymbol
}

func (c *Checker) GetUndefinedSymbol() ast.Symbol {
	return c.symbolOf(c.getUndefinedSymbol())
}

func (c *Checker) getUndefinedSymbol() ast.SymbolId {
	return c.undefinedSymbol
}

func (c *Checker) GetArgumentsSymbol() ast.Symbol {
	return c.symbolOf(c.getArgumentsSymbol())
}

func (c *Checker) getArgumentsSymbol() ast.SymbolId {
	return c.argumentsSymbol
}

func (c *Checker) GetUnknownSignature() *Signature {
	return c.unknownSignature
}

func (c *Checker) GetUnionType(types []*Type) *Type {
	return c.getUnionType(typeIdsOf(types))
}

func (c *Checker) getNameTypeOfSymbol(symbol ast.SymbolId) *Type {
	if links := c.valueSymbolLinks.TryGet(symbol); links != nil {
		return links.getNameType(c)
	}
	return nil
}

func IsTypeUsableAsPropertyName(t *Type) bool {
	return isTypeUsableAsPropertyName(t)
}

func GetPropertyNameFromType(t *Type) ast.SymbolName {
	return getPropertyNameFromType(t)
}

func (c *Checker) getGlobalSymbolId(name ast.SymbolName, meaning ast.SymbolFlags, diagnostic *diagnostics.Message) ast.SymbolId {
	return c.getGlobalSymbol(name, meaning, diagnostic)
}

func (c *Checker) getMergedSymbolId(symbol ast.SymbolId) ast.SymbolId {
	return c.getMergedSymbol(symbol)
}

func (c *Checker) GetSymbolOfNode(node ast.Node) ast.Symbol {
	return c.symbolOf(c.getSymbolOfNodeId(node))
}

func (c *Checker) getSymbolOfNodeId(node ast.Node) ast.SymbolId {
	return c.getSymbolOfNode(node)
}

func (c *Checker) GetSymbolOfDeclaration(node ast.Node) ast.Symbol {
	return c.symbolOf(c.getSymbolOfDeclarationId(node))
}

func (c *Checker) getSymbolOfDeclarationId(node ast.Node) ast.SymbolId {
	return c.getSymbolOfDeclaration(node)
}

func (c *Checker) GetParentOfSymbol(symbol ast.Symbol) ast.Symbol {
	return c.symbolOf(c.getParentOfSymbolId(c.symbolId(symbol)))
}

func (c *Checker) getParentOfSymbolId(symbol ast.SymbolId) ast.SymbolId {
	return c.getParentOfSymbol(symbol)
}

func (c *Checker) tryFindAmbientModuleId(moduleName string) ast.SymbolId {
	return c.tryFindAmbientModule(moduleName, true /* withAugmentations */)
}

func (c *Checker) getImmediateAliasedSymbolId(symbol ast.SymbolId) ast.SymbolId {
	return c.getImmediateAliasedSymbol(symbol)
}

func (c *Checker) GetTargetSymbol(symbol ast.Symbol) ast.Symbol {
	return c.symbolOf(c.getTargetSymbolId(c.symbolId(symbol)))
}

func (c *Checker) getTargetSymbolId(symbol ast.SymbolId) ast.SymbolId {
	return c.getTargetSymbol(symbol)
}

func (c *Checker) getTypeOnlyAliasDeclarationId(symbol ast.SymbolId) ast.Node {
	return c.getTypeOnlyAliasDeclaration(symbol)
}

func (c *Checker) resolveExternalModuleNameId(moduleSpecifier ast.Node, importAttributesType *Type) ast.SymbolId {
	return c.resolveExternalModuleName(moduleSpecifier, moduleSpecifier, true /*ignoreErrors*/, importAttributesType)
}

func (c *Checker) ResolveExternalModuleSymbol(moduleSymbol ast.Symbol) ast.Symbol {
	return c.symbolOf(c.resolveExternalModuleSymbolId(c.symbolId(moduleSymbol)))
}

func (c *Checker) resolveExternalModuleSymbolId(moduleSymbol ast.SymbolId) ast.SymbolId {
	return c.resolveExternalModuleSymbol(moduleSymbol, false /*dontResolveAlias*/)
}

func (c *Checker) GetTypeFromTypeNode(node ast.Node) *Type {
	return c.getTypeFromTypeNode(node)
}

func (c *Checker) IsArrayLikeType(t *Type) bool {
	return c.isArrayLikeType(t)
}

func (c *Checker) getPropertiesOfTypeId(t *Type) []ast.SymbolId {
	return c.getPropertiesOfType(t)
}

func (c *Checker) getPropertyOfTypeId(t *Type, name ast.SymbolName) ast.SymbolId {
	return c.getPropertyOfType(t, name)
}

func (c *Checker) TypeHasCallOrConstructSignatures(t *Type) bool {
	return c.typeHasCallOrConstructSignatures(t)
}

// Checks if a property can be accessed in a location.
// The location is given by the `node` parameter.
// The node does not need to be a property access.
// @param node location where to check property accessibility
// @param isSuper whether to consider this a `super` property access, e.g. `super.foo`.
// @param isWrite whether this is a write access, e.g. `++foo.x`.
// @param containingType type where the property comes from.
// @param property property symbol.
func (c *Checker) isPropertyAccessibleId(node ast.Node, isSuper bool, isWrite bool, containingType *Type, property ast.SymbolId) bool {
	return c.isPropertyAccessible(node, isSuper, isWrite, containingType, property)
}

func (c *Checker) GetTypeOfPropertyOfContextualType(t *Type, name ast.SymbolName) *Type {
	return c.getTypeOfPropertyOfContextualType(t, name)
}

func (c *Checker) WasCanceled() bool {
	return c.wasCanceled
}

func (c *Checker) GetSignaturesOfType(t *Type, kind SignatureKind) []*Signature {
	return c.getSignaturesOfType(t, kind)
}

func (c *Checker) getDeclaredTypeOfSymbolId(symbol ast.SymbolId) *Type {
	return c.getDeclaredTypeOfSymbol(symbol)
}

func (c *Checker) getTypeOfSymbolId(symbol ast.SymbolId) *Type {
	return c.getTypeOfSymbol(symbol)
}

func (c *Checker) GetNonMissingTypeOfSymbol(symbol ast.Symbol) *Type {
	return c.getNonMissingTypeOfSymbolId(c.symbolId(symbol))
}

func (c *Checker) getNonMissingTypeOfSymbolId(symbol ast.SymbolId) *Type {
	return c.getNonMissingTypeOfSymbol(symbol)
}

func (c *Checker) GetConstraintOfTypeParameter(typeParameter *Type) *Type {
	return c.getConstraintOfTypeParameter(typeParameter)
}

func (c *Checker) GetTrueTypeOfConditionalType(t *Type) *Type {
	return c.getTrueTypeFromConditionalType(t)
}

func (c *Checker) GetFalseTypeOfConditionalType(t *Type) *Type {
	return c.getFalseTypeFromConditionalType(t)
}

func (c *Checker) GetDefaultFromTypeParameter(typeParameter *Type) *Type {
	return c.getDefaultFromTypeParameter(typeParameter)
}

func (c *Checker) GetEffectiveDeclarationFlags(n ast.Node, flagsToCheck ast.ModifierFlags) ast.ModifierFlags {
	return c.getEffectiveDeclarationFlags(n, flagsToCheck)
}

func (c *Checker) GetBaseConstraintOfType(t *Type) *Type {
	return c.getBaseConstraintOfType(t)
}

func (c *Checker) GetTypePredicateOfSignature(sig *Signature) *TypePredicate {
	return c.getTypePredicateOfSignature(sig)
}

func IsTupleType(t *Type) bool {
	return isTupleType(t)
}

func GetEndElementCount(t *TupleType, flags ElementFlags) int {
	return getEndElementCount(t, flags)
}

func IsTupleTypeTarget(t *Type) bool {
	return isTupleType(t) && t.Target() == t
}

func (c *Checker) IsArrayType(t *Type) bool {
	return c.isArrayType(t)
}

func (c *Checker) IsReadonlySymbol(symbol ast.Symbol) bool {
	return c.isReadonlySymbolId(c.symbolId(symbol))
}

func (c *Checker) isReadonlySymbolId(symbol ast.SymbolId) bool {
	return c.isReadonlySymbol(symbol)
}

func (c *Checker) GetReturnTypeOfSignature(sig *Signature) *Type {
	return c.getReturnTypeOfSignature(sig)
}

func (c *Checker) HasEffectiveRestParameter(signature *Signature) bool {
	return c.hasEffectiveRestParameter(signature)
}

func (c *Checker) getLocalTypeParametersOfClassOrInterfaceOrTypeAliasId(symbol ast.SymbolId) []TypeId {
	return c.getLocalTypeParametersOfClassOrInterfaceOrTypeAlias(symbol)
}

func (c *Checker) GetContextualTypeForObjectLiteralElement(element ast.Node, contextFlags ContextFlags) *Type {
	return c.getContextualTypeForObjectLiteralElement(element, contextFlags)
}

func (c *Checker) TypePredicateToString(t *TypePredicate) string {
	return c.typePredicateToString(t)
}

func (c *Checker) GetExpandedParameters(signature *Signature, skipUnionExpanding bool) [][]ast.Symbol {
	lists := c.getExpandedParameters(signature, skipUnionExpanding)
	result := make([][]ast.Symbol, len(lists))
	for i, list := range lists {
		result[i] = c.symbolsOf(list)
	}
	return result
}

func (c *Checker) GetResolvedSignature(node ast.Node) *Signature {
	return c.getResolvedSignature(node, nil, CheckModeNormal)
}

// Return the type of the given property in the given type, or nil if no such property exists
func (c *Checker) GetTypeOfPropertyOfType(t *Type, name ast.SymbolName) *Type {
	return c.getTypeOfPropertyOfType(t, name)
}

func (c *Checker) GetContextualTypeForArgumentAtIndex(node ast.Node, argIndex int) *Type {
	return c.getContextualTypeForArgumentAtIndex(node, argIndex)
}

func (c *Checker) GetAwaitedType(t *Type) *Type {
	return c.getAwaitedType(t)
}

func (c *Checker) GetIndexSignaturesAtLocation(node ast.Node) []ast.Node {
	return c.getIndexSignaturesAtLocation(node)
}

func (c *Checker) getResolvedSymbolId(node ast.Node) ast.SymbolId {
	return c.getResolvedSymbol(node)
}

func (c *Checker) GetJsxNamespace(location ast.Node) string {
	return c.getJsxNamespace(location)
}

func (c *Checker) GetJsxFragmentFactory(location ast.Node) string {
	entity := c.getJsxFragmentFactoryEntity(location)
	if !entity.IsNil() {
		return ast.GetFirstIdentifier(entity).Text()
	}
	return ""
}

func (c *Checker) resolveNameId(name ast.SymbolName, location ast.Node, meaning ast.SymbolFlags, excludeGlobals bool) ast.SymbolId {
	return c.resolveName(location, name, meaning, nil, true, excludeGlobals)
}

func (c *Checker) getSymbolFlagsId(symbol ast.SymbolId) ast.SymbolFlags {
	return c.getSymbolFlags(symbol)
}

func (c *Checker) GetBaseTypes(t *Type) []*Type {
	return c.typesOf(c.getBaseTypes(t))
}

func (c *Checker) GetApparentType(t *Type) *Type {
	return c.getApparentType(t)
}

func (c *Checker) GetReducedType(t *Type) *Type {
	return c.getReducedType(t)
}

// GetFullyQualifiedName returns the fully qualified name of a symbol, walking up
// its parent chain (e.g. `"/path/to/module".Namespace.Name`).
func (c *Checker) GetFullyQualifiedName(symbol ast.Symbol) string {
	return c.getFullyQualifiedNameId(c.symbolId(symbol))
}

func (c *Checker) getFullyQualifiedNameId(symbol ast.SymbolId) string {
	return c.getFullyQualifiedName(symbol, ast.Node{} /*containingLocation*/)
}

func (c *Checker) GetBaseConstructorTypeOfClass(t *Type) *Type {
	return c.getBaseConstructorTypeOfClass(t)
}

func (c *Checker) getMemberOverrideModifierStatusId(node ast.Node, member ast.Node, memberSymbol ast.SymbolId) MemberOverrideStatus {
	return c.getMemberOverrideModifierStatus(node, member, memberSymbol)
}

func (c *Checker) GetRestTypeOfSignature(sig *Signature) *Type {
	return c.getRestTypeOfSignature(sig)
}

func (c *Checker) GetTypeArguments(t *Type) []*Type {
	return c.typesOf(c.getTypeArguments(t))
}

func (c *Checker) GetIndexInfoOfType(t *Type, keyType *Type) *IndexInfo {
	return c.getIndexInfoOfType(t, keyType)
}

func (c *Checker) GetIndexTypeOfType(t *Type, keyType *Type) *Type {
	return c.getIndexTypeOfType(t, keyType)
}

func (c *Checker) GetIndexInfosOfType(t *Type) []*IndexInfo {
	return c.getIndexInfosOfType(t)
}

func (c *Checker) getIndexInfosOfIndexSymbolId(symbol ast.SymbolId) []*IndexInfo {
	var siblingSymbols []ast.SymbolId
	if c.symbolParent(symbol) != 0 {
		siblingSymbols = slices.Collect(c.getMembersOfSymbol(c.symbolParent(symbol)).ValueIds(c.symbols))
	}
	return c.getIndexInfosOfIndexSymbol(symbol, siblingSymbols)
}

func (c *Checker) IsContextSensitive(node ast.Node) bool {
	return c.isContextSensitive(node)
}

func (c *Checker) FillMissingTypeArguments(typeArguments []*Type, typeParameters []*Type, minTypeArgumentCount int, isJavaScriptImplicitAny bool) []*Type {
	return c.typesOf(c.fillMissingTypeArguments(typeIdsOf(typeArguments), typeIdsOf(typeParameters), minTypeArgumentCount, isJavaScriptImplicitAny))
}

func (c *Checker) GetMinTypeArgumentCount(typeParameters []*Type) int {
	return c.getMinTypeArgumentCount(typeIdsOf(typeParameters))
}

func (c *Checker) GetWidenedLiteralType(t *Type) *Type {
	return c.getWidenedLiteralType(t)
}

func (c *Checker) IsTypeAssignableTo(source *Type, target *Type) bool {
	return c.isTypeAssignableTo(source, target)
}

func (c *Checker) GetUnionTypeEx(types []*Type, unionReduction UnionReduction) *Type {
	return c.getUnionTypeEx(typeIdsOf(types), unionReduction, nil, nil)
}

func (c *Checker) RequiresAddingImplicitUndefined(node ast.Node) bool {
	enclosingDeclaration := ast.FindAncestor(node, ast.IsDeclaration)
	if enclosingDeclaration.IsNil() {
		enclosingDeclaration = ast.GetSourceFileOfNode(node).AsNode()
	}
	symbol := c.nodeSymbol(node)
	if symbol == 0 {
		return false
	}
	return c.requiresAddingImplicitUndefined(node, symbol, enclosingDeclaration)
}

func (c *Checker) RemoveMissingOrUndefinedType(t *Type) *Type {
	return c.removeMissingOrUndefinedType(t)
}

func (c *Checker) GetWidenedType(t *Type) *Type {
	return c.getWidenedType(t)
}

func (c *Checker) CompareSymbols(s1 ast.Symbol, s2 ast.Symbol) int {
	return c.compareSymbolIds(c.symbolId(s1), c.symbolId(s2))
}

func (c *Checker) compareSymbolIds(s1, s2 ast.SymbolId) int {
	return c.compareSymbols(s1, s2)
}

func IsDistributedTypeParameter(t *Type) bool {
	return t.flags&TypeFlagsTypeParameter != 0 && t.AsTypeParameter().isDistributed()
}

func (c *Checker) GetSignatureDeclaration(sig *Signature) ast.Node {
	return c.nodeOf(sig.declaration)
}

// SignatureDeclaration returns sig's declaration, for callers that hold the
// program's file table but no checker.
func SignatureDeclaration(files *ast.FileTable, sig *Signature) ast.Node {
	return files.Node(sig.declaration)
}

func GetDeclarationModifierFlagsFromSymbol(s ast.Symbol) ast.ModifierFlags {
	return declarationModifierFlagsOfView(s, false /*isWrite*/)
}

// declarationModifierFlagsOfView is getDeclarationModifierFlagsFromSymbolEx
// for callers outside the checker, which hold views.
func declarationModifierFlagsOfView(s ast.Symbol, isWrite bool) ast.ModifierFlags {
	if s.CheckFlags()&ast.CheckFlagsSynthetic != 0 {
		var accessModifier ast.ModifierFlags
		switch {
		case !isWrite && s.CheckFlags()&ast.CheckFlagsContainsPublic != 0 || isWrite && s.CheckFlags()&ast.CheckFlagsContainsWritePublic != 0:
			accessModifier = ast.ModifierFlagsPublic
		case !isWrite && s.CheckFlags()&ast.CheckFlagsContainsProtected != 0 || isWrite && s.CheckFlags()&ast.CheckFlagsContainsWriteProtected != 0:
			accessModifier = ast.ModifierFlagsProtected
		case !isWrite && s.CheckFlags()&ast.CheckFlagsContainsPrivate != 0 || isWrite && s.CheckFlags()&ast.CheckFlagsContainsWritePrivate != 0:
			accessModifier = ast.ModifierFlagsPrivate
		}
		if s.CheckFlags()&ast.CheckFlagsContainsStatic != 0 {
			return accessModifier | ast.ModifierFlagsStatic
		}
		return accessModifier
	}
	if !s.ValueDeclaration().IsNil() {
		var declaration ast.Node
		if isWrite {
			declaration = s.Declarations().Find(ast.IsSetAccessorDeclaration)
		}
		if declaration.IsNil() && s.Flags()&ast.SymbolFlagsGetAccessor != 0 {
			declaration = s.Declarations().Find(ast.IsGetAccessorDeclaration)
		}
		if declaration.IsNil() {
			declaration = s.ValueDeclaration()
		}
		flags := ast.GetCombinedModifierFlags(declaration)
		if !s.Parent().IsNil() && s.Parent().Flags()&ast.SymbolFlagsClass != 0 {
			return flags
		}
		return flags & ^ast.ModifierFlagsAccessibilityModifier
	}
	if s.Flags()&ast.SymbolFlagsPrototype != 0 {
		return ast.ModifierFlagsPublic | ast.ModifierFlagsStatic
	}
	return ast.ModifierFlagsNone
}

func (c *Checker) GetUnknownSymbol() ast.Symbol {
	return c.symbolOf(c.getUnknownSymbol())
}

func (c *Checker) GetNameTypeOfSymbol(symbol ast.Symbol) *Type {
	return c.getNameTypeOfSymbol(c.symbolId(symbol))
}

func (c *Checker) GetGlobalSymbol(name ast.SymbolName, meaning ast.SymbolFlags, diagnostic *diagnostics.Message) ast.Symbol {
	return c.symbolOf(c.getGlobalSymbolId(name, meaning, diagnostic))
}

func (c *Checker) GetMergedSymbol(symbol ast.Symbol) ast.Symbol {
	return c.symbolOf(c.getMergedSymbolId(c.symbolId(symbol)))
}

func (c *Checker) TryFindAmbientModule(moduleName string) ast.Symbol {
	return c.symbolOf(c.tryFindAmbientModuleId(moduleName))
}

func (c *Checker) GetImmediateAliasedSymbol(symbol ast.Symbol) ast.Symbol {
	return c.symbolOf(c.getImmediateAliasedSymbolId(c.symbolId(symbol)))
}

func (c *Checker) GetTypeOnlyAliasDeclaration(symbol ast.Symbol) ast.Node {
	return c.getTypeOnlyAliasDeclarationId(c.symbolId(symbol))
}

func (c *Checker) ResolveExternalModuleName(moduleSpecifier ast.Node, importAttributesType *Type) ast.Symbol {
	return c.symbolOf(c.resolveExternalModuleNameId(moduleSpecifier, importAttributesType))
}

func (c *Checker) GetPropertiesOfType(t *Type) []ast.Symbol {
	return c.symbolsOf(c.getPropertiesOfTypeId(t))
}

func (c *Checker) GetPropertyOfType(t *Type, name ast.SymbolName) ast.Symbol {
	return c.symbolOf(c.getPropertyOfTypeId(t, name))
}

func (c *Checker) IsPropertyAccessible(node ast.Node, isSuper bool, isWrite bool, containingType *Type, property ast.Symbol) bool {
	return c.isPropertyAccessibleId(node, isSuper, isWrite, containingType, c.symbolId(property))
}

func (c *Checker) GetDeclaredTypeOfSymbol(symbol ast.Symbol) *Type {
	return c.getDeclaredTypeOfSymbolId(c.symbolId(symbol))
}

func (c *Checker) GetTypeOfSymbol(symbol ast.Symbol) *Type {
	return c.getTypeOfSymbolId(c.symbolId(symbol))
}

func (c *Checker) GetLocalTypeParametersOfClassOrInterfaceOrTypeAlias(symbol ast.Symbol) []*Type {
	return c.typesOf(c.getLocalTypeParametersOfClassOrInterfaceOrTypeAliasId(c.symbolId(symbol)))
}

func (c *Checker) GetResolvedSymbol(node ast.Node) ast.Symbol {
	return c.symbolOf(c.getResolvedSymbolId(node))
}

func (c *Checker) ResolveName(name ast.SymbolName, location ast.Node, meaning ast.SymbolFlags, excludeGlobals bool) ast.Symbol {
	return c.symbolOf(c.resolveNameId(name, location, meaning, excludeGlobals))
}

func (c *Checker) GetSymbolFlags(symbol ast.Symbol) ast.SymbolFlags {
	return c.getSymbolFlagsId(c.symbolId(symbol))
}

func (c *Checker) GetMemberOverrideModifierStatus(node ast.Node, member ast.Node, memberSymbol ast.Symbol) MemberOverrideStatus {
	return c.getMemberOverrideModifierStatusId(node, member, c.symbolId(memberSymbol))
}

func (c *Checker) GetIndexInfosOfIndexSymbol(symbol ast.Symbol) []*IndexInfo {
	return c.getIndexInfosOfIndexSymbolId(c.symbolId(symbol))
}
