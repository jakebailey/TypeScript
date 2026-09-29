package transformers

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
)

type Transformer struct {
	AfterSourceElement func(*ast.Node, []*ast.Node) []*ast.Node
	emitContext        *printer.EmitContext
	factory            *printer.NodeFactory
	visitor            *ast.NodeVisitor
	components         []*Transformer
	sourceStatements   func(*ast.StatementList, *ast.NodeVisitor) *ast.StatementList
}

func (tx *Transformer) InSourcePipeline() bool {
	return tx.sourceStatements != nil
}

func (tx *Transformer) NewTransformer(visit func(node *ast.Node) *ast.Node, emitContext *printer.EmitContext) *Transformer {
	if tx.emitContext != nil {
		panic("Transformer already initialized")
	}
	if emitContext == nil {
		emitContext = printer.NewEmitContext()
	}
	tx.emitContext = emitContext
	tx.factory = emitContext.Factory
	tx.visitor = emitContext.NewNodeVisitor(visit)
	tx.visitor.Hooks.VisitTopLevelStatements = tx.VisitSourceFileStatements
	return tx
}

func (tx *Transformer) VisitSourceFileStatements(nodes *ast.StatementList, visitor *ast.NodeVisitor) *ast.StatementList {
	if tx.sourceStatements != nil {
		return tx.sourceStatements(nodes, visitor)
	}
	return tx.emitContext.VisitVariableEnvironment(nodes, visitor)
}

func (tx *Transformer) EmitContext() *printer.EmitContext {
	return tx.emitContext
}

func (tx *Transformer) Visitor() *ast.NodeVisitor {
	return tx.visitor
}

func (tx *Transformer) Factory() *printer.NodeFactory {
	return tx.factory
}

func (tx *Transformer) TransformSourceFile(file *ast.SourceFile) *ast.SourceFile {
	return tx.visitor.VisitSourceFile(file)
}
