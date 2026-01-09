package parser

import "github.com/ava12/llx/grammar"

type nodeRec struct {
	hooks []NodeHookInstance
	sides []*Token
	types grammar.BitSet
	index int
	state int
}

type nodeStack struct {
	nodes []nodeRec
}

func newNodeStack() *nodeStack {
	return &nodeStack{}
}

func (s *nodeStack) IsEmpty() bool {
	return len(s.nodes) == 0
}

func (s *nodeStack) Len() int {
	return len(s.nodes)
}

func (s *nodeStack) Push(n nodeRec) {
	s.nodes = append(s.nodes, n)
}

func (s *nodeStack) Drop() {
	if len(s.nodes) != 0 {
		s.nodes = s.nodes[:len(s.nodes)-1]
	}
}

func (s *nodeStack) Top() *nodeRec {
	if len(s.nodes) == 0 {
		return nil
	}

	return &s.nodes[len(s.nodes)-1]
}

func (s *nodeStack) Copy() *nodeStack {
	result := &nodeStack{
		nodes: make([]nodeRec, len(s.nodes)),
	}
	copy(result.nodes, s.nodes)
	return result
}

func (s *nodeStack) Indexes() []int {
	result := make([]int, len(s.nodes))
	l := len(s.nodes) - 1
	for i, n := range s.nodes {
		result[l-i] = n.index
	}
	return result
}

type parsingBranch struct {
	NodeStack    *nodeStack
	AppliedRules []grammar.Rule
}

func (pc *ParseContext) newParsingBranch() *parsingBranch {
	result := &parsingBranch{
		NodeStack: newNodeStack(),
	}
	node := pc.nodeStack.Top()
	result.NodeStack.Push(nodeRec{
		types: node.types,
		index: node.index,
		state: node.state,
	})
	return result
}

func (b *parsingBranch) Split(count int) []*parsingBranch {
	result := make([]*parsingBranch, count)
	result[0] = b
	for i := 1; i < count; i++ {
		nb := &parsingBranch{
			NodeStack:    b.NodeStack.Copy(),
			AppliedRules: make([]grammar.Rule, len(b.AppliedRules)),
		}
		copy(nb.AppliedRules, b.AppliedRules)
		result[i] = nb
	}
	return result
}

func (b *parsingBranch) AddRule(rule grammar.Rule) {
	b.AppliedRules = append(b.AppliedRules, rule)
}
