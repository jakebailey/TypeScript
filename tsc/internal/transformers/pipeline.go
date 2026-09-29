package transformers

import (
	"slices"

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
	return runSourcePipeline(file, stages)
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
	stages           []pipelineStage
	context          *printer.EmitContext
	completed        map[*ast.Node]struct{}
	capturesElements bool
}

func (p *sourcePipeline) visit(node *ast.Node, start int, output []*ast.Node) []*ast.Node {
	for i := start; i < len(p.stages); i++ {
		stage := &p.stages[i]
		if stage.visitor == nil || stage.visitor.Visit == nil {
			continue
		}
		saved := p.context.SwapEnvironment(&stage.environment)
		node = stage.visitor.Visit(node)
		p.context.SwapEnvironment(saved)
		if node == nil {
			return output
		}
		if node.Kind == ast.KindSyntaxList {
			for _, child := range node.AsSyntaxList().Children {
				output = p.visitOutput(child, i, output)
			}
			return output
		}
		if after := stage.transformer.AfterSourceElement; after != nil {
			return p.visitOutput(node, i, output)
		}
	}
	p.completed[node] = struct{}{}
	return append(output, node)
}

func (p *sourcePipeline) visitOutput(node *ast.Node, index int, output []*ast.Node) []*ast.Node {
	if after := p.stages[index].transformer.AfterSourceElement; after != nil {
		p.capturesElements = true
		return append(output, after(node, p.visit(node, index+1, nil))...)
	}
	return p.visit(node, index+1, output)
}

func (p *sourcePipeline) transform(file *ast.SourceFile, index int) *ast.SourceFile {
	if index < 0 {
		p.completed = make(map[*ast.Node]struct{}, len(file.Statements.Nodes))
		statements := make([]*ast.Node, 0, len(file.Statements.Nodes))
		for _, node := range file.Statements.Nodes {
			statements = p.visit(node, 0, statements)
		}
		list := p.context.Factory.NewNodeList(statements)
		list.Loc = file.Statements.Loc
		return p.context.Factory.UpdateSourceFile(file, list, file.EndOfFileToken).AsSourceFile()
	}
	stage := &p.stages[index]
	tx := stage.transformer
	saved := p.context.SwapEnvironment(&stage.environment)
	defer func() {
		p.context.SwapEnvironment(saved)
		tx.sourceStatements = nil
	}()
	var inner *ast.SourceFile
	tx.sourceStatements = func(nodes *ast.StatementList, visitor *ast.NodeVisitor) *ast.StatementList {
		p.context.StartVariableEnvironment()
		stage.visitor = visitor
		inner = p.transform(file, index-1)
		// Earlier source-file epilogues may have introduced imports or hoisted
		// statements after the main stream was consumed.
		var statements []*ast.Node
		changed := false
		for i, node := range inner.Statements.Nodes {
			if _, done := p.completed[node]; !done {
				if !changed {
					statements = slices.Clone(inner.Statements.Nodes[:i])
					changed = true
					if p.capturesElements {
						restore := p.context.SkipSubtrees(p.completed)
						defer restore()
					}
				}
				statements = p.visit(node, index, statements)
			} else if changed {
				statements = append(statements, node)
			}
		}
		list := inner.Statements
		if changed {
			list = p.context.Factory.NewNodeList(statements)
			list.Loc = inner.Statements.Loc
		}
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
