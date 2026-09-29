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

const sourceElementBatchSize = 16

// Pipeline completes the ordered transforms for a bounded window of source
// elements before advancing. This keeps the working set local without switching
// between large visitors for every statement. Source-file visitors retain their
// file-level state and complete in forward phase order.
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

func (p *sourcePipeline) visitBatch(nodes []*ast.Node) []*ast.Node {
	for i := range p.stages {
		stage := &p.stages[i]
		if stage.visitor == nil || stage.visitor.Visit == nil {
			continue
		}
		saved := p.context.SwapEnvironment(&stage.environment)
		nodes, _ = stage.visitor.VisitSlice(nodes)
		p.context.SwapEnvironment(saved)
		if stage.transformer.AfterSourceElement != nil {
			return p.visitCapturedBatch(nodes, i)
		}
	}
	for _, node := range nodes {
		p.completed[node] = struct{}{}
	}
	return nodes
}

type pipelineElement struct {
	node   *ast.Node
	origin int
}

func (p *sourcePipeline) visitCapturedBatch(roots []*ast.Node, index int) []*ast.Node {
	// Preserve ownership across elision and expansion so each callback receives
	// exactly its source element's fully transformed output.
	p.capturesElements = true
	elements := make([]pipelineElement, len(roots))
	for i, node := range roots {
		elements[i] = pipelineElement{node, i}
	}
	spare := make([]pipelineElement, 0, len(roots))
	for i := index + 1; i < len(p.stages); i++ {
		stage := &p.stages[i]
		if stage.visitor == nil || stage.visitor.Visit == nil {
			continue
		}
		saved := p.context.SwapEnvironment(&stage.environment)
		for _, element := range elements {
			result := stage.visitor.Visit(element.node)
			if result == nil {
				continue
			}
			var nodes []*ast.Node
			if result.Kind == ast.KindSyntaxList {
				nodes = result.AsSyntaxList().Children
			} else {
				nodes = []*ast.Node{result}
			}
			for _, node := range nodes {
				if stage.transformer.AfterSourceElement != nil {
					for _, output := range p.visitOutput(node, i, nil) {
						spare = append(spare, pipelineElement{output, element.origin})
					}
				} else {
					spare = append(spare, pipelineElement{node, element.origin})
				}
			}
		}
		p.context.SwapEnvironment(saved)
		elements, spare = spare, elements[:0]
		if stage.transformer.AfterSourceElement != nil {
			break
		}
	}
	completed := make([]*ast.Node, len(elements))
	for i, element := range elements {
		completed[i] = element.node
		p.completed[element.node] = struct{}{}
	}
	var output []*ast.Node
	pos := 0
	after := p.stages[index].transformer.AfterSourceElement
	for i, root := range roots {
		start := pos
		for pos < len(elements) && elements[pos].origin == i {
			pos++
		}
		output = append(output, after(root, completed[start:pos])...)
	}
	return output
}

func (p *sourcePipeline) transform(file *ast.SourceFile, index int) *ast.SourceFile {
	if index < 0 {
		p.completed = make(map[*ast.Node]struct{}, len(file.Statements.Nodes))
		statements := make([]*ast.Node, 0, len(file.Statements.Nodes))
		for i := 0; i < len(file.Statements.Nodes); i += sourceElementBatchSize {
			statements = append(statements, p.visitBatch(file.Statements.Nodes[i:min(i+sourceElementBatchSize, len(file.Statements.Nodes))])...)
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
