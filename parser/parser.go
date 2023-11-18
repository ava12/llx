// Package parser defines generic LL(*) parser.
package parser

import (
	"bytes"
	"github.com/ava12/llx/internal/bmap"
	"math/bits"
	"regexp"
	"sort"
	"strings"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/internal/queue"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/source"
)

type Token = lexer.Token

// TokenHook allows to perform additional actions when token is fetched from lexer, but before it is queued,
// e.g. emit external $indent/$dedent tokens when text indentation changes, fetch complex lexemes (e.g. heredoc strings).
// emit flag set to true means that incoming token (even aside one) must be put to the beginning of token queue,
// false means that it must be skipped.
type TokenHook = func(token *Token, pc *ParseContext) (emit bool, e error)

// NodeHookInstance receives notifications for node being processed by parser.
type NodeHookInstance interface {
	// NewNode is called before a child node is pushed on stack.
	// Receives child node name and its initial token.
	NewNode(node string, token *Token) error

	// HandleNode is called when nested node is finalized and dropped.
	// Receives child node name.
	// Receives result of closest nested hook EndNode() call or nil if none of nested nodes was hooked.
	HandleNode(node string, result any) error

	// HandleToken is called when a token belonging to current node is received.
	HandleToken(token *Token) error

	// EndNode is called when current node is finalized.
	// result is passed to parent node hook or returned as a parse result if current node is the root.
	EndNode() (result any, e error)
}

// NodeHook allows to perform actions on nodes emitted by parser.
// Receives node name and initial token.
type NodeHook = func(node string, token *Token, pc *ParseContext) (NodeHookInstance, error)

type defaultHookInstance struct {
	result any
}

func (dhi *defaultHookInstance) NewNode(node string, token *Token) error {
	return nil
}

func (dhi *defaultHookInstance) HandleNode(node string, result any) error {
	dhi.result = result
	return nil
}

func (dhi *defaultHookInstance) HandleToken(token *Token) error {
	return nil
}

func (dhi *defaultHookInstance) EndNode() (result any, e error) {
	return dhi.result, nil
}

// Special token type names used by token hooks.
const (
	AnyToken = ""                 // any token type
	EofToken = lexer.EofTokenName // end-of-file token
	EoiToken = lexer.EoiTokenName // end-of-input token
)

// AnyNode denotes any node, used by node hooks.
const AnyNode = ""

const anyOffset = -1

type TokenHooks map[string]TokenHook
type NodeHooks map[string]NodeHook

// Hooks contains all token and node hooks used in parsing process.
// Default action when no suitable token hook found is to drop aside token and use non-aside token as is.
type Hooks struct {
	// Tokens contains hooks for different token types. Key is either token type name or AnyToken constant.
	// AnyToken hook is a fallback.
	Tokens TokenHooks

	// Literals contains hooks for tokens with specific content. Key is token content.
	// These hooks have top priority (if token type allows matching against literals).
	Literals TokenHooks

	// Nodes contains hooks for nodes. Key is either node name or AnyNode constant.
	// AnyNode hook is a fallback.
	Nodes NodeHooks
}

// Parser holds prepared data for some grammar.
// Parser is immutable and reusable.
type Parser struct {
	grammar  *grammar.Grammar
	names    map[string]int
	literals *bmap.BMap[int]
	lexers   []*lexer.Lexer
}

// New constructs new parser for specific grammar.
// Grammar must not be changed after this function is called.
func New(g *grammar.Grammar) (*Parser, error) {
	maxGroup := 0
	literalsCnt := 0
	for _, t := range g.Tokens {
		mg := bits.Len(uint(t.Groups)) - 1
		if mg > maxGroup {
			maxGroup = mg
		}
		if t.Flags&grammar.LiteralToken != 0 {
			literalsCnt++
		}
	}

	type lexerRec struct {
		patterns []string
		types    []lexer.TokenType
	}
	lrs := make([]lexerRec, maxGroup+1)

	names := make(map[string]int)
	literals := bmap.New[int](literalsCnt)

	names[tokenKey(AnyToken)] = grammar.AnyToken
	names[nodeKey(AnyNode)] = -1
	names[tokenKey(EofToken)] = lexer.EofTokenType
	names[tokenKey(EoiToken)] = lexer.EoiTokenType

	for i, t := range g.Tokens {
		if (t.Flags & grammar.LiteralToken) != 0 {
			literals.Set([]byte(t.Name), i)
		} else if (t.Flags & grammar.ErrorToken) == 0 {
			names[tokenKey(t.Name)] = i
		}
		if t.Re == "" {
			continue
		}

		group := -1
		gs := t.Groups
		pattern := "(" + t.Re + ")"
		for ; gs != 0; gs >>= 1 {
			group++
			if (gs & 1) == 0 {
				continue
			}

			lrs[group].types = append(lrs[group].types, lexer.TokenType{i, t.Name})
			lrs[group].patterns = append(lrs[group].patterns, pattern)
		}
	}

	ls := make([]*lexer.Lexer, len(lrs))
	for i := range ls {
		re, e := regexp.Compile("^(?s:" + strings.Join(lrs[i].patterns, "|") + ")")
		if e != nil {
			return nil, e
		}

		ls[i] = lexer.New(re, lrs[i].types)
	}

	for i, nt := range g.Nodes {
		names[nodeKey(nt.Name)] = i
	}

	return &Parser{g, names, literals, ls}, nil
}

func tokenKey(name string) string {
	return "$" + name
}

func nodeKey(text string) string {
	return ":" + text
}

// Parse launches new parsing process with new ParseContext.
// result is the value returned by root node hook or nil if no node hooks used.
func (p *Parser) Parse(q *source.Queue, hs *Hooks) (result any, e error) {
	if hs == nil {
		hs = &Hooks{}
	}
	pc, e := newParseContext(p, q, hs)
	if e != nil {
		return nil, e
	}

	return pc.parse()
}

// ParseString is same as Parse, except it creates source queue containing single source having
// provided content with provided name (name may be empty).
func (p *Parser) ParseString(name, content string, hs *Hooks) (result any, e error) {
	q := source.NewQueue().Append(source.New(name, []byte(content)))
	return p.Parse(q, hs)
}

type nodeRec struct {
	prev   *nodeRec
	hook   NodeHookInstance
	asides []*Token
	group  int
	index  int
	state  int
}

// ParseContext contains all context used in parsing process.
type ParseContext struct {
	parser       *Parser
	sources      *source.Queue
	tokenHooks   []TokenHook
	nodeHooks    []NodeHook
	tokens       *queue.Queue[*Token]
	appliedRules *queue.Queue[grammar.Rule]
	tokenError   error
	lastResult   any
	node         *nodeRec
}

const (
	tokenHooksOffset = -lexer.LowestTokenType
	nodeHooksOffset  = -grammar.AnyToken
)

func newParseContext(p *Parser, q *source.Queue, hs *Hooks) (*ParseContext, error) {
	result := &ParseContext{
		parser:       p,
		sources:      q,
		tokenHooks:   make([]TokenHook, len(p.grammar.Tokens)+tokenHooksOffset),
		nodeHooks:    make([]NodeHook, len(p.grammar.Nodes)+nodeHooksOffset),
		tokens:       queue.New[*Token](),
		appliedRules: queue.New[grammar.Rule](),
	}

	for k, th := range hs.Tokens {
		i, f := p.names[tokenKey(k)]
		if !f {
			return nil, unknownTokenTypeError(k)
		}

		result.tokenHooks[i+tokenHooksOffset] = th
	}

	for k, th := range hs.Literals {
		i, f := p.literals.Get([]byte(k))
		if !f {
			return nil, unknownTokenLiteralError(k)
		}

		result.tokenHooks[i+tokenHooksOffset] = th
	}

	for k, nth := range hs.Nodes {
		i, f := p.names[nodeKey(k)]
		if !f {
			return nil, unknownNodeError(k)
		}

		result.nodeHooks[i+nodeHooksOffset] = nth
	}

	e := result.pushNode(grammar.RootNode, lexer.NewToken(grammar.AnyToken, "", nil, q.SourcePos()))
	return result, e
}

// EmitToken adds new element to the end of token queue.
// Token's type must be defined in grammar, and it must not be a literal or an error token.
func (pc *ParseContext) EmitToken(t *Token) error {
	tt := t.Type()
	if tt < 0 || tt >= len(pc.parser.grammar.Tokens) {
		return emitWrongTokenError(t)
	}

	flags := pc.parser.grammar.Tokens[tt].Flags
	if flags&(grammar.LiteralToken|grammar.ErrorToken) != 0 {
		return emitWrongTokenError(t)
	}

	pc.tokens.Append(t)
	return nil
}

func (pc *ParseContext) pushNode(index int, tok *Token) error {
	e := pc.ntHandleAsides()
	if e != nil {
		return e
	}

	gr := pc.parser.grammar
	nt := gr.Nodes[index]
	if pc.node != nil {
		e = pc.node.hook.NewNode(nt.Name, tok)
		if e != nil {
			return e
		}
	}

	hook, e := pc.getNodeHook(index, tok)
	if e != nil {
		return e
	}

	pc.node = &nodeRec{pc.node, hook, nil, gr.States[nt.FirstState].Group, index, nt.FirstState}
	return nil
}

func (pc *ParseContext) popNode() error {
	var (
		e   error
		res any
	)
	nts := pc.parser.grammar.Nodes

	asides := pc.node.asides
	pc.node.asides = nil

	for e == nil && pc.node != nil && pc.node.state == grammar.FinalState {
		nt := pc.node
		if nt.prev == nil {
			for _, t := range asides {
				e = nt.hook.HandleToken(t)
			}
			if e != nil {
				return e
			}
		}

		pc.node = nt.prev
		res, e = nt.hook.EndNode()
		pc.lastResult = res
		if pc.node == nil {
			break
		}

		if e == nil {
			e = pc.node.hook.HandleNode(nts[nt.index].Name, res)
		}
	}

	if e == nil && pc.node != nil {
		pc.node.asides = asides
	}

	return e
}

const repeatState = -128

func (pc *ParseContext) parse() (any, error) {
	var (
		tok           *Token
		e             error
		tokenConsumed bool
	)
	gr := pc.parser.grammar

	for pc.node != nil {
		tok, e = pc.nextToken(pc.node.group)
		tokenConsumed = false
		if e != nil {
			return nil, e
		}

		for !tokenConsumed && pc.node != nil {
			nt := pc.node
			rule, found := pc.nextRule(tok, gr.States[nt.state])
			if !found {
				shrunk, e := pc.shrinkToken(tok, nt.group)
				if e != nil {
					return nil, e
				}

				if shrunk {
					break
				}

				expected := pc.getExpectedToken(gr.States[nt.state])
				if tok.Type() == lexer.EoiTokenType {
					e = unexpectedEofError(tok, expected)
				} else {
					e = unexpectedTokenError(tok, expected)
				}
				return nil, e
			}

			sameNode := (rule.Node == grammar.SameNode)
			tokenConsumed = (sameNode && rule.Token != grammar.AnyToken)
			if rule.State != repeatState {
				pc.node.state = rule.State
				if rule.State != grammar.FinalState {
					pc.node.group = gr.States[rule.State].Group
				}
			}

			if !sameNode {
				e = pc.pushNode(rule.Node, tok)
			} else if tokenConsumed {
				e = pc.ntHandleToken(tok)
			}

			if e == nil && pc.node.state == grammar.FinalState {
				e = pc.popNode()
			}

			if e != nil {
				return nil, e
			}
		}
	}

	if !tokenConsumed && tok.Type() != lexer.EoiTokenType {
		s := tok.Source()
		if s != nil {
			if s != pc.sources.Source() {
				pc.sources.Prepend(s)
			}
			pc.sources.Seek(s.Pos(tok.Line(), tok.Col()))
		}
	}

	return pc.lastResult, nil
}

func (pc *ParseContext) shrinkToken(tok *Token, group int) (bool, error) {
	if tok.Type() < 0 || pc.parser.grammar.Tokens[tok.Type()].Flags&grammar.ShrinkableToken == 0 {
		return false, nil
	}

	if !pc.tokens.IsEmpty() {
		return false, nil
	}

	res := pc.parser.lexers[group].Shrink(pc.sources, tok)
	var e error
	if res != nil {
		e = pc.handleToken(res)
	}
	return (res != nil), e
}

func (pc *ParseContext) resolve(tok *Token, ars []grammar.Rule) ([]*Token, []grammar.Rule) {
	liveBranch := createBranches(pc, pc.node, ars)
	tokens := make([]*Token, 0)
	pc.tokens.Prepend(tok)
	for {
		var parentBranch, deadBranch, lastDead *branch
		currentBranch := liveBranch
		deadBranch = nil
		lastDead = nil
		survivors := 0
		tok, e := pc.nextToken(liveBranch.nextGroup())
		if e != nil || tok == nil {
			return tokens, liveBranch.applied
		}

		tokens = append(tokens, tok)

		for currentBranch != nil {
			if currentBranch.applyToken(tok) {
				survivors++
				parentBranch = currentBranch
				currentBranch = currentBranch.next
			} else {
				if parentBranch == nil {
					liveBranch = currentBranch.next
				} else {
					parentBranch.next = currentBranch.next
				}

				if lastDead == nil {
					deadBranch = currentBranch
				} else {
					lastDead.next = currentBranch
				}
				nextBranch := currentBranch.next
				lastDead = currentBranch
				lastDead.next = nil
				currentBranch = nextBranch
			}
		}

		if survivors < 2 {
			if survivors == 0 {
				liveBranch = deadBranch
			}

			return tokens, liveBranch.applied
		}
	}
}

func (pc *ParseContext) getExpectedToken(s grammar.State) string {
	g := pc.parser.grammar
	index := 0
	if s.HighRule > s.LowRule {
		index = g.Rules[s.LowRule].Token
		if index < 0 && s.HighRule > s.LowRule+1 {
			index = g.Rules[s.LowRule+1].Token
		}
	}
	if s.HighMultiRule > s.LowMultiRule {
		i := g.MultiRules[s.LowMultiRule].Token
		if i < index {
			index = i
		}
	}

	token := g.Tokens[index]
	if (token.Flags & grammar.LiteralToken) != 0 {
		return token.Name
	} else {
		return "$" + token.Name
	}
}

func (pc *ParseContext) findRules(t *Token, s grammar.State) []grammar.Rule {
	if pc.isAsideToken(t) {
		return []grammar.Rule{{t.Type(), repeatState, grammar.SameNode}}
	}

	keys := pc.possibleRuleKeys(t)
	g := pc.parser.grammar
	rules := g.Rules[s.LowRule:s.HighRule]
	multiRules := g.MultiRules[s.LowMultiRule:s.HighMultiRule]
	rlen := len(rules)
	mrlen := len(multiRules)

	for _, key := range keys {
		if key == grammar.AnyToken && rlen > 0 && rules[0].Token == key {
			return rules[0:1]
		}

		index := sort.Search(rlen, func(i int) bool {
			return rules[i].Token >= key
		})
		if index < rlen && rules[index].Token == key {
			return rules[index : index+1]
		}

		index = sort.Search(mrlen, func(i int) bool {
			return multiRules[i].Token >= key
		})
		if index < mrlen && multiRules[index].Token == key {
			mr := multiRules[index]
			return g.Rules[mr.LowRule:mr.HighRule]
		}
	}

	return nil
}

func (pc *ParseContext) possibleRuleKeys(t *Token) []int {
	keys := make([]int, 0, 3)
	tt := t.Type()
	var tf grammar.TokenFlags
	tokens := pc.parser.grammar.Tokens
	if tt >= 0 {
		tf = tokens[tt].Flags
	} else {
		tf = grammar.NoLiteralsToken
	}

	literalFound := false
	literalIndex := 0
	if (tf & grammar.NoLiteralsToken) == 0 {
		literal := t.Content()
		if tf&grammar.CaselessToken != 0 {
			literal = bytes.ToUpper(literal)
		}
		literalIndex, literalFound = pc.parser.literals.Get(literal)
		literalFound = literalFound && (literalIndex >= 0)
		if literalFound {
			keys = append(keys, literalIndex)
		}
	}

	if !literalFound || literalIndex < 0 || (tokens[literalIndex].Flags&grammar.ReservedToken) == 0 {
		keys = append(keys, tt)
	}
	keys = append(keys, grammar.AnyToken)

	return keys
}

func (pc *ParseContext) getNodeHook(ntIndex int, tok *Token) (res NodeHookInstance, e error) {
	h := pc.nodeHooks[ntIndex+nodeHooksOffset]
	if h == nil {
		h = pc.nodeHooks[anyOffset+nodeHooksOffset]
	}
	if h != nil {
		res, e = h(pc.parser.grammar.Nodes[ntIndex].Name, tok, pc)
	} else {
		e = nil
	}
	if res == nil {
		res = &defaultHookInstance{}
	}
	return
}

func (pc *ParseContext) nextToken(group int) (result *Token, e error) {
	var fetched bool
	result, fetched = pc.tokens.First()
	if !fetched {
		if pc.tokenError != nil {
			e = pc.tokenError
		} else {
			result, e = pc.fetchToken(group)
		}
	}

	return
}

func (pc *ParseContext) nextRule(t *Token, s grammar.State) (r grammar.Rule, found bool) {
	r, found = pc.appliedRules.First()
	if found {
		return
	}

	rules := pc.findRules(t, s)
	if len(rules) == 0 {
		return
	}

	found = true
	if len(rules) == 1 {
		r = rules[0]
	} else {
		tokens, rules := pc.resolve(t, rules)
		r = rules[0]
		for i := len(tokens) - 1; i >= 1; i-- {
			pc.tokens.Prepend(tokens[i])
		}
		pc.appliedRules.Fill(rules[1:])
	}

	return
}

func (pc *ParseContext) handleToken(tok *Token) error {
	tts := make([]int, 0, 3)
	tt := tok.Type()

	if tt < 0 {
		tts = append(tts, tt)
	} else {
		if pc.parser.grammar.Tokens[tt].Flags&grammar.NoLiteralsToken == 0 {
			i, f := pc.parser.literals.Get(tok.Content())
			if f {
				tts = append(tts, i)
			}
		}
		tts = append(tts, tt, anyOffset)
	}

	var h TokenHook
	for _, i := range tts {
		h = pc.tokenHooks[i+tokenHooksOffset]
		if h != nil {
			break
		}
	}

	if h == nil {
		if pc.isAsideToken(tok) || tt == lexer.EofTokenType {
			return nil
		}

		pc.tokens.Append(tok)
		return nil
	}

	emit, e := h(tok, pc)
	if tt == lexer.EofTokenType {
		emit = false
	}
	if emit || tt < 0 {
		pc.tokens.Append(tok)
	}
	if e != nil {
		if !pc.tokens.IsEmpty() {
			pc.tokenError = e
		} else {
			return e
		}
	}

	return nil
}

func (pc *ParseContext) fetchToken(group int) (*Token, error) {
	for pc.tokens.IsEmpty() {
		result, e := pc.parser.lexers[group].Next(pc.sources)
		if e == nil {
			e = pc.handleToken(result)
		}

		if e != nil {
			return nil, e
		}
	}

	result, _ := pc.tokens.First()
	return result, nil
}

func (pc *ParseContext) isAsideToken(t *Token) bool {
	tokens := pc.parser.grammar.Tokens
	i := t.Type()
	return (i >= 0 && i < len(tokens) && tokens[i].Flags&grammar.AsideToken != 0)
}

func (pc *ParseContext) ntHandleAsides() (res error) {
	ntr := pc.node
	if ntr == nil || ntr.asides == nil {
		return nil
	}

	for _, t := range ntr.asides {
		res = ntr.hook.HandleToken(t)
		if res != nil {
			break
		}
	}
	if res == nil {
		ntr.asides = nil
	}
	return
}

func (pc *ParseContext) ntHandleToken(tok *Token) (res error) {
	ntr := pc.node
	if pc.isAsideToken(tok) {
		ntr.asides = append(ntr.asides, tok)
	} else {
		res = pc.ntHandleAsides()
		if res == nil {
			res = ntr.hook.HandleToken(tok)
		}
	}
	return
}
