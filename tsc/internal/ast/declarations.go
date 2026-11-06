package ast

import (
	"iter"
	"slices"
)

// Declarations is a read-only snapshot of a symbol's declarations.
//
// Nearly all symbols have their declarations in one store, so the compact form keeps the store once
// and the declarations as node indices: the first inline, any others in rest.idx. A symbol with no
// other declarations allocates nothing. Declarations that span stores use the general form, with
// s == nil and the nodes in rest.nodes.
//
// A snapshot never changes: adding a declaration copies the list, which keeps snapshots shared
// between symbols safe and leaves each list exactly sized.
type Declarations struct {
	s     *NodeStore
	rest  *declRest
	first uint32
}

type declRest struct {
	idx   []uint32 // Compact form: the declarations after the first, all in s.
	nodes []Node   // General form: all declarations.
	value Node     // General form: the value declaration.
}

// NewDeclarations returns declarations holding nodes, in the compact form when they share a store.
func NewDeclarations(nodes ...Node) Declarations {
	if len(nodes) == 0 {
		return Declarations{}
	}
	s := nodes[0].s
	for _, n := range nodes[1:] {
		if n.s != s {
			return Declarations{rest: &declRest{nodes: nodes}}
		}
	}
	d := Declarations{s: s, first: nodes[0].node}
	if len(nodes) > 1 {
		idx := make([]uint32, len(nodes)-1)
		for i, n := range nodes[1:] {
			idx[i] = n.node
		}
		d.rest = &declRest{idx: idx}
	}
	return d
}

func singleDeclaration(n Node) Declarations {
	if n.IsNil() {
		return Declarations{}
	}
	return Declarations{s: n.s, first: n.node}
}

func (d Declarations) Len() int {
	if d.s != nil {
		if d.first == 0 {
			return 0
		}
		if d.rest == nil {
			return 1
		}
		return 1 + len(d.rest.idx)
	}
	if d.rest != nil {
		return len(d.rest.nodes)
	}
	return 0
}

func (d Declarations) At(i int) Node {
	if d.s != nil {
		if i == 0 && d.first != 0 {
			return Node{d.s, d.first}
		}
		return Node{d.s, d.rest.idx[i-1]}
	}
	return d.rest.nodes[i]
}

// All iterates over the declarations with their positions.
func (d Declarations) All() iter.Seq2[int, Node] {
	return func(yield func(int, Node) bool) {
		n := d.Len()
		for i := range n {
			if !yield(i, d.At(i)) {
				return
			}
		}
	}
}

// Values iterates over the declarations.
func (d Declarations) Values() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		n := d.Len()
		for i := range n {
			if !yield(d.At(i)) {
				return
			}
		}
	}
}

// Nodes returns the declarations as a new slice, or nil when there are none.
func (d Declarations) Nodes() []Node {
	n := d.Len()
	if n == 0 {
		return nil
	}
	if d.s == nil {
		return append([]Node(nil), d.rest.nodes...)
	}
	result := make([]Node, n)
	for i := range n {
		result[i] = d.At(i)
	}
	return result
}

func (d Declarations) FirstOrNil() Node {
	if d.Len() == 0 {
		return Node{}
	}
	return d.At(0)
}

func (d Declarations) LastOrNil() Node {
	n := d.Len()
	if n == 0 {
		return Node{}
	}
	return d.At(n - 1)
}

func (d Declarations) Some(f func(Node) bool) bool {
	return d.FindIndex(f) >= 0
}

func (d Declarations) Every(f func(Node) bool) bool {
	n := d.Len()
	for i := range n {
		if !f(d.At(i)) {
			return false
		}
	}
	return true
}

func (d Declarations) Find(f func(Node) bool) Node {
	if i := d.FindIndex(f); i >= 0 {
		return d.At(i)
	}
	return Node{}
}

func (d Declarations) FindLast(f func(Node) bool) Node {
	for i := d.Len() - 1; i >= 0; i-- {
		if n := d.At(i); f(n) {
			return n
		}
	}
	return Node{}
}

func (d Declarations) FindIndex(f func(Node) bool) int {
	n := d.Len()
	for i := range n {
		if f(d.At(i)) {
			return i
		}
	}
	return -1
}

func (d Declarations) CountWhere(f func(Node) bool) int {
	count := 0
	n := d.Len()
	for i := range n {
		if f(d.At(i)) {
			count++
		}
	}
	return count
}

// Filter returns the declarations for which f is true, or nil when there are none.
func (d Declarations) Filter(f func(Node) bool) []Node {
	var result []Node
	n := d.Len()
	for i := range n {
		if node := d.At(i); f(node) {
			result = append(result, node)
		}
	}
	return result
}

func (d Declarations) Index(node Node) int {
	if d.s != nil {
		if node.s != d.s || node.node == 0 {
			return -1
		}
		if d.first == node.node {
			return 0
		}
		if d.rest != nil {
			for i, idx := range d.rest.idx {
				if idx == node.node {
					return i + 1
				}
			}
		}
		return -1
	}
	if d.rest != nil {
		for i, n := range d.rest.nodes {
			if n == node {
				return i
			}
		}
	}
	return -1
}

func (d Declarations) Contains(node Node) bool {
	return d.Index(node) >= 0
}

// Append returns a new list holding the declarations of d followed by node.
func (d Declarations) Append(node Node) Declarations {
	if d.s != nil && node.s == d.s {
		if d.first == 0 {
			return Declarations{s: d.s, first: node.node}
		}
		var idx []uint32
		if d.rest != nil {
			idx = d.rest.idx
		}
		return Declarations{s: d.s, first: d.first, rest: &declRest{idx: append(slices.Clip(idx), node.node)}}
	}
	if d.s == nil && d.rest == nil {
		return singleDeclaration(node)
	}
	return Declarations{rest: &declRest{nodes: append(d.Nodes(), node)}}
}

// AppendIfUnique returns d with node added at the end if it is not already present.
func (d Declarations) AppendIfUnique(node Node) Declarations {
	if d.Contains(node) {
		return d
	}
	return d.Append(node)
}

// Concat returns a new list holding the declarations of d followed by those of other.
func (d Declarations) Concat(other Declarations) Declarations {
	if other.Len() == 0 {
		return d
	}
	if d.Len() == 0 {
		return other
	}
	return NewDeclarations(slices.AppendSeq(d.Nodes(), other.Values())...)
}
