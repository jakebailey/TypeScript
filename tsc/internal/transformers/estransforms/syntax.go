package estransforms

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/transformers"
)

// Syntax lowerings share a visitor and a lexical environment. In particular, a
// lowering must visit its operands through this visitor, not through a separate
// walk for its own language version.
type syntaxTransformer struct {
	transformers.Transformer
	facts            ast.SubtreeFacts
	objectRestSpread *objectRestSpreadTransformer
}

func newSyntaxTransformer(opts *transformers.TransformOptions) *transformers.Transformer {
	target := opts.CompilerOptions.GetEmitScriptTarget()
	var facts ast.SubtreeFacts
	if target < core.ScriptTargetES2021 {
		facts |= ast.SubtreeContainsLogicalAssignments
	}
	if target < core.ScriptTargetES2020 {
		facts |= ast.SubtreeContainsNullishCoalescing | ast.SubtreeContainsOptionalChaining
	}
	if target < core.ScriptTargetES2019 {
		facts |= ast.SubtreeContainsMissingCatchClauseVariable
	}
	if target < core.ScriptTargetES2016 {
		facts |= ast.SubtreeContainsExponentiationOperator
	}
	if target < core.ScriptTargetES2018 {
		facts |= ast.SubtreeContainsESObjectRestOrSpread
	}
	if facts == 0 {
		return nil
	}
	tx := &syntaxTransformer{facts: facts}
	result := tx.NewTransformer(tx.visit, opts.Context)
	if target < core.ScriptTargetES2018 {
		tx.objectRestSpread = &objectRestSpreadTransformer{Transformer: result, compilerOptions: opts.CompilerOptions}
	}
	return result
}

func (tx *syntaxTransformer) visit(node *ast.Node) *ast.Node {
	if node.Kind == ast.KindSourceFile {
		// Decorators can introduce ??, and JSX can introduce object spread.
		sourceFacts := tx.facts | ast.SubtreeContainsDecorators
		if tx.objectRestSpread != nil {
			sourceFacts |= ast.SubtreeContainsJsx
		}
		if node.SubtreeFacts()&sourceFacts == 0 {
			return node
		}
	}
	rest := tx.objectRestSpread
	activeParameters := rest != nil && rest.parametersWithPrecedingObjectRestOrSpread != nil
	if node.Kind != ast.KindSourceFile && node.SubtreeFacts()&tx.facts == 0 && !activeParameters {
		return node
	}
	var result *ast.Node
	if rest != nil {
		unused := rest.expressionResultIsUnused
		rest.expressionResultIsUnused = false
		result = tx.visitLocalSyntax(node)
		rest.expressionResultIsUnused = unused
	} else {
		result = tx.visitLocalSyntax(node)
	}
	if result != nil {
		return result
	}
	if rest != nil && (node.Kind == ast.KindSourceFile || node.SubtreeFacts()&ast.SubtreeContainsESObjectRestOrSpread != 0 || activeParameters) {
		return rest.visit(node)
	}
	return tx.Visitor().VisitEachChild(node)
}

func (tx *syntaxTransformer) visitLocalSyntax(node *ast.Node) *ast.Node {
	switch node.Kind {
	case ast.KindBinaryExpression:
		switch node.AsBinaryExpression().OperatorToken.Kind {
		case ast.KindBarBarEqualsToken, ast.KindAmpersandAmpersandEqualsToken, ast.KindQuestionQuestionEqualsToken:
			if tx.facts&ast.SubtreeContainsLogicalAssignments != 0 {
				result := tx.visitLogicalAssignment(node.AsBinaryExpression())
				if result.AsBinaryExpression().OperatorToken.Kind == ast.KindQuestionQuestionToken &&
					tx.facts&ast.SubtreeContainsNullishCoalescing != 0 {
					// The operands have already been visited. Lower the newly generated
					// ?? without walking either operand again.
					return tx.lowerNullishCoalescing(result.AsBinaryExpression().Left, result.AsBinaryExpression().Right)
				}
				return result
			}
		case ast.KindQuestionQuestionToken:
			if tx.facts&ast.SubtreeContainsNullishCoalescing != 0 {
				n := node.AsBinaryExpression()
				return tx.lowerNullishCoalescing(tx.Visitor().VisitNode(n.Left), tx.Visitor().VisitNode(n.Right))
			}
		case ast.KindAsteriskAsteriskToken:
			if tx.facts&ast.SubtreeContainsExponentiationOperator != 0 {
				return tx.visitExponentiationExpression(node.AsBinaryExpression())
			}
		case ast.KindAsteriskAsteriskEqualsToken:
			if tx.facts&ast.SubtreeContainsExponentiationOperator != 0 {
				return tx.visitExponentiationAssignmentExpression(node.AsBinaryExpression())
			}
		}
	case ast.KindCallExpression:
		if tx.facts&ast.SubtreeContainsOptionalChaining != 0 {
			return tx.visitCallExpression(node.AsCallExpression(), false)
		}
	case ast.KindPropertyAccessExpression, ast.KindElementAccessExpression:
		if tx.facts&ast.SubtreeContainsOptionalChaining != 0 && node.Flags&ast.NodeFlagsOptionalChain != 0 {
			return tx.visitOptionalExpression(node, false, false)
		}
	case ast.KindDeleteExpression:
		if tx.facts&ast.SubtreeContainsOptionalChaining != 0 {
			return tx.visitDeleteExpression(node.AsDeleteExpression())
		}
	case ast.KindCatchClause:
		if tx.facts&ast.SubtreeContainsMissingCatchClauseVariable != 0 && node.AsCatchClause().VariableDeclaration == nil {
			return tx.visitCatchClause(node.AsCatchClause())
		}
	}
	return nil
}
