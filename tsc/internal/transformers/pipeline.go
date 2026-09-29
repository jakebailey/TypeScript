package transformers

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
)

type pipelineStage struct {
	transformer *Transformer
	visitor     *ast.NodeVisitor
	environment printer.EmitEnvironment
}

// Pipeline completes the ordered transforms for each source element before
// advancing to the next element. Source-file visitors are entered in reverse
// order and completed in forward order, retaining their file-level state.
func Pipeline(file *ast.SourceFile, transforms []*Transformer) *ast.SourceFile {
	var stages []pipelineStage
	var appendStages func([]*Transformer)
	appendStages = func(transforms []*Transformer) {
		for _, tx := range transforms {
			if tx == nil {
				continue
			}
			if tx.components != nil {
				appendStages(tx.components)
			} else {
				stages = append(stages, pipelineStage{transformer: tx})
			}
		}
	}
	appendStages(transforms)
	for _, stage := range stages {
		if stage.transformer.EmitContext() != stages[0].transformer.EmitContext() {
			panic("Pipeline transforms must share an emit context")
		}
	}
	start := 0
	for i := range stages {
		if barrier := stages[i].transformer.SourceFileBarrier; barrier != nil && barrier(file) {
			file = runSourcePipeline(file, stages[start:i])
			file = stages[i].transformer.TransformSourceFile(file)
			start = i + 1
		}
	}
	return runSourcePipeline(file, stages[start:])
}

func runSourcePipeline(file *ast.SourceFile, stages []pipelineStage) *ast.SourceFile {
	if len(stages) == 0 {
		return file
	}
	if len(stages) == 1 {
		return stages[0].transformer.TransformSourceFile(file)
	}
	p := sourcePipeline{stages: stages, context: stages[0].transformer.EmitContext()}
	return p.transform(file, len(stages)-1)
}

type sourcePipeline struct {
	stages    []pipelineStage
	context   *printer.EmitContext
	completed map[*ast.Node]struct{}
}

func (p *sourcePipeline) visit(node *ast.Node, start int) []*ast.Node {
	nodes := []*ast.Node{node}
	for i := start; i < len(p.stages); i++ {
		stage := &p.stages[i]
		if stage.visitor == nil {
			continue
		}
		saved := p.context.SwapEnvironment(stage.environment)
		nodes, _ = stage.visitor.VisitSlice(nodes)
		stage.environment = p.context.SwapEnvironment(saved)
	}
	for _, node := range nodes {
		p.completed[node] = struct{}{}
	}
	return nodes
}

func (p *sourcePipeline) transform(file *ast.SourceFile, index int) *ast.SourceFile {
	if index < 0 {
		p.completed = make(map[*ast.Node]struct{}, len(file.Statements.Nodes))
		var statements []*ast.Node
		for _, node := range file.Statements.Nodes {
			statements = append(statements, p.visit(node, 0)...)
		}
		list := p.context.Factory.NewNodeList(statements)
		list.Loc = file.Statements.Loc
		return p.context.Factory.UpdateSourceFile(file, list, file.EndOfFileToken).AsSourceFile()
	}
	stage := &p.stages[index]
	tx := stage.transformer
	saved := p.context.SwapEnvironment(printer.EmitEnvironment{})
	defer func() {
		p.context.SwapEnvironment(saved)
		tx.sourceStatements = nil
	}()
	var inner *ast.SourceFile
	tx.sourceStatements = func(nodes *ast.StatementList, visitor *ast.NodeVisitor) *ast.StatementList {
		p.context.StartVariableEnvironment()
		stage.visitor = visitor
		stage.environment = p.context.SwapEnvironment(printer.EmitEnvironment{})
		inner = p.transform(file, index-1)
		p.context.SwapEnvironment(stage.environment)
		// Earlier source-file epilogues may have introduced imports or hoisted
		// statements after the main stream was consumed.
		var statements []*ast.Node
		for _, node := range inner.Statements.Nodes {
			if _, done := p.completed[node]; done {
				statements = append(statements, node)
			} else {
				stage.environment = p.context.SwapEnvironment(printer.EmitEnvironment{})
				statements = append(statements, p.visit(node, index)...)
				p.context.SwapEnvironment(stage.environment)
			}
		}
		list := p.context.Factory.NewNodeList(statements)
		list.Loc = inner.Statements.Loc
		list = p.context.EndAndMergeVariableEnvironmentList(list)
		// Propagate source metadata before this stage's epilogue consumes it.
		p.context.AddEmitHelper(file.AsNode(), p.context.GetEmitHelpers(inner.AsNode())...)
		p.context.AddEmitFlags(file.AsNode(), p.context.EmitFlags(inner.AsNode()))
		return list
	}
	result := tx.TransformSourceFile(file)
	if inner == nil {
		if result != file {
			panic("Source-file rewrites must participate in the pipeline")
		}
		// An inactive transform (for example, module lowering in a script) can
		// skip the source visitor entirely.
		result = p.transform(file, index-1)
	}
	return result
}
