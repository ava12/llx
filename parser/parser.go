// Package parser defines generic LL(*) parser.
package parser

import (
	"context"
	"maps"
	"regexp"
	"sort"
	"strings"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/internal/queue"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/source"
)

type Token = lexer.Token

type parserSettings struct {
	layerTemplates map[string]HookLayerTemplate
}

// Option is an optional argument for New
type Option func(*parserSettings)

// WithLayerTemplates defines hook layer templates that override registered ones.
func WithLayerTemplates(templates map[string]HookLayerTemplate) Option {
	return func(ps *parserSettings) {
		ps.layerTemplates = templates
	}
}

// Parser holds prepared data for some grammar.
// Parser is immutable and reusable.
type Parser struct {
	grammar    *grammar.Grammar
	tokenNames map[string]int
	nodeNames  map[string]int
	literals   map[string]int
	lexers     []*lexer.Lexer
	layers     []HookLayer
}

// New constructs new parser for specific grammar.
// Grammar must not be changed after this function is called.
func New(g *grammar.Grammar, opts ...Option) (*Parser, error) {
	var settings parserSettings
	for _, opt := range opts {
		opt(&settings)
	}

	maxGroup := 0
	literalsCnt := 0
	for _, t := range g.Tokens {
		if t.Flags&grammar.LiteralToken != 0 {
			literalsCnt++
		} else if t.Group > maxGroup {
			maxGroup = t.Group
		}
	}

	type lexerRec struct {
		patterns []string
		types    []lexer.TokenType
	}
	lrs := make([]lexerRec, maxGroup+1)

	tokenNames := make(map[string]int, len(g.Tokens)+3)
	nodeNames := make(map[string]int, len(g.Nodes)+1)
	literals := make(map[string]int, literalsCnt)

	tokenNames[AnyToken] = grammar.AnyToken
	nodeNames[AnyNode] = -1
	tokenNames[EofToken] = lexer.EofTokenType
	tokenNames[EoiToken] = lexer.EoiTokenType

	for i, t := range g.Tokens {
		if (t.Flags & grammar.LiteralToken) != 0 {
			literals[t.Name] = i
		} else if (t.Flags & grammar.ErrorToken) == 0 {
			tokenNames[t.Name] = i
		}
		if t.Re == "" {
			continue
		}

		lr := &lrs[t.Group]
		pattern := "(" + t.Re + ")"
		lr.types = append(lr.types, lexer.TokenType{i, t.Name})
		lr.patterns = append(lr.patterns, pattern)
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
		nodeNames[nt.Name] = i
	}

	result := &Parser{g, tokenNames, nodeNames, literals, ls, nil}

	if len(g.Layers) != 0 {
		knownHookLayerLock.RLock()
		defer knownHookLayerLock.RUnlock()

		result.layers = make([]HookLayer, 0, len(g.Layers))
		for _, layerDef := range g.Layers {
			tpl, has := settings.layerTemplates[layerDef.Type]
			if !has {
				tpl, has = knownHookLayers[layerDef.Type]
			}
			if !has {
				return nil, unknownLayerError(layerDef.Type)
			}

			config, e := tpl.Setup(layerDef.Commands, result)
			if e != nil {
				return nil, e
			}

			result.layers = append(result.layers, config)
		}
	}

	return result, nil
}

// ParseOption is an optional argument for Parser.Parse*
type ParseOption func(*ParseContext)

// WithSides instructs parser to pass all tokens (include side ones) to node hooks.
// By default only non-side tokens are passed to node hooks.
func WithSides() ParseOption {
	return func(pc *ParseContext) {
		pc.passSides = true
	}
}

// WithFullSource instructs parser to check whether there are any non-side tokens left
// in source file after parsing is done.
// By default parser stops and returns result as soon as root node is finalized,
// no matter if end of source is reached or not.
// With this option parser returns error if there are any non-side tokens left.
func WithFullSource() ParseOption {
	return func(pc *ParseContext) {
		pc.fullSource = true
	}
}

// Parse launches new parsing process with new ParseContext.
// result is the value returned by root node hook or nil if no node hooks used.
func (p *Parser) Parse(ctx context.Context, q *source.Queue, hs Hooks, opts ...ParseOption) (result any, e error) {
	pc, e := newParseContext(ctx, p, q, hs)
	if e != nil {
		return nil, e
	}

	for _, opt := range opts {
		opt(pc)
	}

	return pc.parse(ctx)
}

// ParseString is same as Parse, except it creates source queue containing single source having
// provided content with provided name (name may be empty).
func (p *Parser) ParseString(ctx context.Context, name, content string, hs Hooks, opts ...ParseOption) (result any, e error) {
	q := source.NewQueue().Append(source.New(name, []byte(content)))
	return p.Parse(ctx, q, hs, opts...)
}

// Tokens returns all tokens defined in grammar.
func (p *Parser) Tokens() []grammar.Token {
	return p.grammar.Tokens
}

// IsSideType returns true if given argument is a valid side token type.
func (p *Parser) IsSideType(tt int) bool {
	return tt >= 0 && tt < len(p.grammar.Tokens) && (p.grammar.Tokens[tt].Flags&grammar.SideToken != 0)
}

// IsSpecialType returns true if given argument is a valid special token type (EoF, EoI).
func (p *Parser) IsSpecialType(tt int) bool {
	return tt < 0 && tt >= lexer.LowestTokenType && tt != grammar.AnyToken
}

// IsValidType returns true if given argument is a valid token type that can be used by parser.
func (p *Parser) IsValidType(tt int) bool {
	return tt >= lexer.LowestTokenType && tt < len(p.grammar.Tokens) && tt != grammar.AnyToken
}

// TokenType returns token type for given type name and true if given argument is a valid token type name.
// Returns false if type name is unknown to parser.
func (p *Parser) TokenType(typeName string) (tokenType int, valid bool) {
	tokenType, valid = p.tokenNames[typeName]
	return
}

// MakeToken generates artificial token.
// Returns error if given typeName is not defined in grammar.
func (p *Parser) MakeToken(typeName string, content []byte) (*Token, error) {
	tt, valid := p.tokenNames[typeName]
	if !valid {
		return nil, unknownTokenTypeError(typeName)
	}

	return lexer.NewToken(tt, typeName, content, source.NewPos(nil, 0)), nil
}

type tokenHookRec struct {
	hooks  []TokenHook
	tokens *queue.Queue[*Token]
}

// ParseContext contains data used in parsing process.
type ParseContext struct {
	parser       *Parser
	sources      *source.Queue
	literals     map[string]int
	tokens       *queue.Queue[*Token]
	tokenHooks   []tokenHookRec
	nodeHooks    [][]NodeHook
	appliedRules *queue.Queue[grammar.Rule]
	lastResult   any
	nodeStack    *nodeStack
	tc           TokenContext
	passSides    bool
	watchNodes   bool
	fullSource   bool
}

const (
	tokenHooksOffset = -lexer.LowestTokenType
	nodeHooksOffset  = -grammar.AnyToken
)

func newParseContext(ctx context.Context, p *Parser, q *source.Queue, hs Hooks) (*ParseContext, error) {
	result := &ParseContext{
		parser:       p,
		sources:      q,
		literals:     make(map[string]int, len(p.literals)),
		tokens:       queue.New[*Token](),
		tokenHooks:   make([]tokenHookRec, 0, len(p.layers)+1),
		nodeHooks:    make([][]NodeHook, 0, len(p.layers)+1),
		appliedRules: queue.New[grammar.Rule](),
		nodeStack:    newNodeStack(),
		watchNodes:   len(hs.Nodes) != 0,
	}
	result.tc = TokenContext{pc: result}

	maps.Copy(result.literals, p.literals)

	e := newHookLayer(p, result, hs)
	if e != nil {
		return nil, e
	}

	if result.watchNodes && result.nodeHooks[0][0] == nil {
		result.nodeHooks[0][0] = defaultNodeHook
	}

	for i := len(p.layers) - 1; i >= 0; i-- {
		hooks := p.layers[i].Init(ctx, result)
		newHookLayer(p, result, hooks)
	}

	firstToken := lexer.NewToken(grammar.AnyToken, "", nil, q.SourcePos())
	e = result.pushNode(ctx, result.nodeStack, grammar.RootNode, firstToken, true)
	return result, e
}

func newHookLayer(p *Parser, pc *ParseContext, hs Hooks) error {
	if len(hs.Tokens)+len(hs.Literals) > 0 {
		maxTokenType := 0

		for k := range hs.Tokens {
			i, f := p.tokenNames[k]
			if !f {
				return unknownTokenTypeError(k)
			}

			if i > maxTokenType {
				maxTokenType = i
			}
		}

		for k := range hs.Literals {
			i, f := pc.literals[k]
			if !f {
				i = len(p.grammar.Tokens) + len(pc.literals) - len(p.literals)
				pc.literals[k] = i
			}
			if i > maxTokenType {
				maxTokenType = i
			}
		}

		tokenLayer := tokenHookRec{
			hooks:  make([]TokenHook, maxTokenType+tokenHooksOffset+1),
			tokens: queue.New[*Token](),
		}

		for k, th := range hs.Tokens {
			tokenLayer.hooks[p.tokenNames[k]+tokenHooksOffset] = th
		}

		for k, th := range hs.Literals {
			i, f := p.literals[k]
			if !f {
				i, f = pc.literals[k]
			}
			tokenLayer.hooks[i+tokenHooksOffset] = th
		}
		pc.tokenHooks = append(pc.tokenHooks, tokenLayer)
	}

	if len(hs.Nodes) > 0 {
		nodeHooks := make([]NodeHook, len(p.grammar.Nodes)+nodeHooksOffset)
		for k, nth := range hs.Nodes {
			i, f := p.nodeNames[k]
			if !f {
				return unknownNodeError(k)
			}

			nodeHooks[i+nodeHooksOffset] = nth
		}
		pc.nodeHooks = append(pc.nodeHooks, nodeHooks)
	}

	return nil
}

// Parser returns parser used by this context.
func (pc *ParseContext) Parser() *Parser {
	return pc.parser
}

// Sources returns source queue used by this context.
func (pc *ParseContext) Sources() *source.Queue {
	return pc.sources
}

func (pc *ParseContext) pushNode(ctx context.Context, stack *nodeStack, index int, tok *Token, useHooks bool) error {
	var e error

	node := stack.Top()
	if useHooks {
		e = pc.handleNodeSides(node)
		if e != nil {
			return e
		}
	}

	gr := pc.parser.grammar
	nt := gr.Nodes[index]
	if useHooks && !stack.IsEmpty() {
		for _, hook := range node.hooks {
			if hook != nil {
				e = hook.NewNode(nt.Name, tok)
				if e != nil {
					return e
				}
			}
		}
	}

	var newHooks []NodeHookInstance
	if useHooks {
		newHooks = make([]NodeHookInstance, len(pc.nodeHooks))
		for layer := range pc.nodeHooks {
			hook, e := pc.getNodeHook(ctx, layer, index, tok)
			if e != nil {
				return e
			}

			newHooks[layer] = hook
		}
	}

	newNode := nodeRec{
		types: gr.States[nt.FirstState].TokenTypes,
		index: index,
		state: nt.FirstState,
		hooks: newHooks,
	}
	stack.Push(newNode)
	return nil
}

func (pc *ParseContext) popNode(stack *nodeStack, useHooks bool) error {
	var (
		e   error
		res any
	)
	nts := pc.parser.grammar.Nodes

	node := stack.Top()
	sides := node.sides
	node.sides = nil

	for e == nil && node != nil && node.state == grammar.FinalState {
		if stack.Len() <= 1 {
			if useHooks {
				for _, t := range sides {
					for _, hook := range node.hooks {
						if hook == nil {
							continue
						}

						e = hook.HandleToken(t)
						if e != nil {
							return e
						}
					}
				}
			}
		}

		results := make([]any, len(node.hooks))
		if useHooks {
			for i, hook := range node.hooks {
				if hook == nil {
					continue
				}

				res, e = hook.EndNode()
				if e != nil {
					return e
				}

				results[i] = res
				if i == 0 && pc.watchNodes {
					pc.lastResult = res
				}
			}
		}

		nodeIndex := node.index
		stack.Drop()
		node = stack.Top()
		if node == nil {
			break
		}

		if useHooks {
			for i, hook := range node.hooks {
				if hook == nil {
					continue
				}

				e = hook.HandleNode(nts[nodeIndex].Name, results[i])
				if e != nil {
					return e
				}
			}
		}
	}

	if node != nil {
		node.sides = sides
	}

	return e
}

const repeatState = -128

func (pc *ParseContext) parse(ctx context.Context) (any, error) {
	var (
		tok           *Token
		e             error
		tokenConsumed bool
	)
	gr := pc.parser.grammar

	node := pc.nodeStack.Top()
	for node != nil {
		e = ctx.Err()
		if e != nil {
			return nil, e
		}

		tok, e = pc.nextToken(ctx, node.types)
		tokenConsumed = false
		if e != nil {
			return nil, e
		}

		for !tokenConsumed && node != nil {
			rule, found := pc.nextRule(ctx, tok, gr.States[node.state])
			if !found {
				if tok == nil {
					tok, e = pc.nextToken(ctx, lexer.AllTokenTypes)
					if e != nil {
						return nil, e
					}
				}
				expected := pc.getExpectedToken(gr.States[node.state])
				if tok.Type() == lexer.EoiTokenType {
					e = unexpectedEofError(tok, expected)
				} else {
					e = unexpectedTokenError(tok, expected)
				}
				return nil, e
			}

			tokenConsumed, e = pc.applyRule(ctx, tok, rule, pc.nodeStack, true)
			node = pc.nodeStack.Top()

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
			pc.sources.Seek(tok.Pos().Pos())
		}
	}

	if pc.fullSource && (tok == nil || tok.Type() != lexer.EoiTokenType) {
		for {
			tok, e = pc.nextToken(ctx, lexer.AllTokenTypes)
			if e != nil {
				return nil, e
			}

			tt := tok.Type()
			if tt == lexer.EoiTokenType {
				break
			}

			if !pc.parser.IsSideType(tt) {
				return nil, remainingSourceError(tok)
			}
		}
	}

	return pc.lastResult, nil
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
	if pc.isSideToken(t) {
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
	if t == nil {
		return []int{grammar.AnyToken}
	}

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
		literal := t.Text()
		if tf&grammar.CaselessToken != 0 {
			literal = strings.ToUpper(literal)
		}
		literalIndex, literalFound = pc.parser.literals[literal]
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

func (pc *ParseContext) getNodeHook(ctx context.Context, layerIndex, ntIndex int, tok *Token) (res NodeHookInstance, e error) {
	hook := pc.nodeHooks[layerIndex][ntIndex+nodeHooksOffset]
	if hook == nil {
		hook = pc.nodeHooks[layerIndex][anyOffset+nodeHooksOffset]
	}
	if hook != nil {
		nc := &NodeContext{pc}
		res, e = hook(ctx, pc.parser.grammar.Nodes[ntIndex].Name, tok, nc)
	} else {
		e = nil
	}
	return
}

func (pc *ParseContext) nextToken(ctx context.Context, types grammar.BitSet) (*Token, error) {
	if !pc.tokens.IsEmpty() {
		result, _ := pc.tokens.First()
		return result, nil
	}

	for {
		tok, e := pc.pullToken(ctx, types, 0)
		if e != nil {
			return nil, e
		}

		isSide := tok != nil && pc.parser.IsSideType(tok.Type())
		isEof := tok != nil && tok.Type() == lexer.EofTokenType
		if !isEof && (!isSide || pc.passSides) {
			return tok, nil
		}
	}
}

func (pc *ParseContext) pullToken(ctx context.Context, types grammar.BitSet, layer int) (*Token, error) {
	if layer >= len(pc.tokenHooks) {
		tok, e := pc.fetchToken(types)
		return tok, e
	}

	queue := pc.tokenHooks[layer].tokens
	result, fetched := queue.First()
	if fetched {
		return result, nil
	}

	for {
		result, e := pc.pullToken(ctx, types, layer+1)
		if e != nil {
			return nil, e
		}

		emit, extra, e := pc.handleToken(ctx, result, layer)
		if e != nil {
			return nil, e
		}

		for _, tok := range extra {
			if !pc.parser.IsValidType(tok.Type()) {
				return nil, emitWrongTokenError(tok)
			}
			queue.Append(tok)
		}

		if emit {
			return result, nil
		}

		result, fetched = queue.First()
		if fetched {
			return result, nil
		}
	}
}

func (pc *ParseContext) nextRule(ctx context.Context, t *Token, s grammar.State) (r grammar.Rule, found bool) {
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
		//		tokens, rules := pc.resolve(ctx, t, rules)
		tokens, rules := pc.resolveConflict(ctx, t)
		r = rules[0]
		for i := len(tokens) - 1; i >= 1; i-- {
			pc.tokens.Prepend(tokens[i])
		}
		pc.appliedRules.Fill(rules[1:])
	}

	return
}

func (pc *ParseContext) handleToken(ctx context.Context, tok *Token, layer int) (emit bool, extra []*Token, e error) {
	if tok == nil {
		return true, nil, nil
	}

	e = ctx.Err()
	if e != nil {
		return false, nil, e
	}

	hooks := pc.tokenHooks[layer].hooks
	maxTokenType := len(hooks) - tokenHooksOffset - 1
	tts := make([]int, 0, 3)
	tt := tok.Type()

	if tt < 0 {
		tts = append(tts, tt)
	} else {
		if pc.parser.grammar.Tokens[tt].Flags&grammar.NoLiteralsToken == 0 {
			i, f := pc.literals[tok.Text()]
			if f && i <= maxTokenType {
				tts = append(tts, i)
			}
		}
		if tt <= maxTokenType {
			tts = append(tts, tt)
		}
		tts = append(tts, anyOffset)
	}

	var hook TokenHook
	for _, i := range tts {
		hook = hooks[i+tokenHooksOffset]
		if hook != nil {
			break
		}
	}

	if hook == nil {
		return true, nil, nil
	}

	pc.tc.layer = layer
	emit, extra, e = hook(ctx, tok, &pc.tc)
	pc.tc.restore()

	if !emit && tt < 0 {
		if len(extra) == 0 {
			emit = true
		} else {
			extra = append(extra, tok)
		}
	}
	return emit, extra, e
}

func (pc *ParseContext) fetchToken(types grammar.BitSet) (*Token, error) {
	var firstError error
	var result *Token
	var e error

	for _, l := range pc.parser.lexers {
		result, e = l.NextOf(pc.sources, types)
		if e == nil && result != nil {
			return result, nil
		}

		if e != nil && firstError == nil {
			firstError = e
		}
	}

	if types == lexer.AllTokenTypes {
		return nil, firstError
	}

	return nil, nil
}

func (pc *ParseContext) isSideToken(t *Token) bool {
	if t == nil {
		return false
	}

	tokens := pc.parser.grammar.Tokens
	i := t.Type()
	return (i >= 0 && i < len(tokens) && tokens[i].Flags&grammar.SideToken != 0)
}

func (pc *ParseContext) handleNodeSides(node *nodeRec) (e error) {
	if node == nil || node.sides == nil {
		return nil
	}

	for _, t := range node.sides {
		for _, hook := range node.hooks {
			if hook == nil {
				continue
			}

			e = hook.HandleToken(t)
			if e != nil {
				return
			}
		}
	}

	node.sides = nil
	return
}

func (pc *ParseContext) handleNodeToken(tok *Token, node *nodeRec) (e error) {
	if tok == nil {
		return nil
	}

	if pc.isSideToken(tok) {
		node.sides = append(node.sides, tok)
	} else {
		e = pc.handleNodeSides(node)
		if e == nil {
			for _, hook := range node.hooks {
				if hook == nil {
					continue
				}

				e = hook.HandleToken(tok)
				if e != nil {
					break
				}
			}
		}
	}
	return
}

func (pc *ParseContext) applyRule(ctx context.Context, tok *lexer.Token, rule grammar.Rule, stack *nodeStack, useHooks bool) (bool, error) {
	node := stack.Top()
	gr := pc.parser.grammar
	sameNode := (rule.Node == grammar.SameNode)
	tokenConsumed := ((sameNode && rule.Token != grammar.AnyToken) || tok == nil)
	if rule.State != repeatState {
		node.state = rule.State
		if rule.State != grammar.FinalState {
			node.types = gr.States[rule.State].TokenTypes
		}
	}

	var e error
	if !sameNode {
		e = pc.pushNode(ctx, stack, rule.Node, tok, useHooks)
		node = stack.Top()
	} else if tokenConsumed && useHooks {
		e = pc.handleNodeToken(tok, node)
	}

	if e == nil && node.state == grammar.FinalState {
		e = pc.popNode(stack, useHooks)
		node = stack.Top()
	}

	return tokenConsumed, e
}

func (pc *ParseContext) resolveConflict(ctx context.Context, tok *Token) ([]*Token, []grammar.Rule) {
	var tokens []*lexer.Token
	nextBranches := []*parsingBranch{pc.newParsingBranch()}
	var branches []*parsingBranch
	q := queue.New[*parsingBranch]()
	gr := pc.parser.grammar
	var lastSuccess, lastError, b *parsingBranch

	applyRule := func(b *parsingBranch, rule grammar.Rule) {
		b.AddRule(rule)
		consumed, _ := pc.applyRule(ctx, tok, rule, b.NodeStack, false)
		if b.NodeStack.Len() == 0 || consumed {
			if consumed {
				nextBranches = append(nextBranches, b)
			} else {
				lastSuccess = b
			}
		} else {
			q.Append(b)
		}
	}

	for {
		branches = nextBranches
		nextBranches = nextBranches[:0]
		tokens = append(tokens, tok)
		lastSuccess = nil

		for _, b = range branches {
			q.Append(b)

			for !q.IsEmpty() {
				b, _ = q.First()
				if b.NodeStack.IsEmpty() {
					lastSuccess = b
					continue
				}

				rules := pc.findRules(tok, gr.States[b.NodeStack.Top().state])

				switch len(rules) {
				case 0:
					lastError = b
				case 1:
					applyRule(b, rules[0])
				default:
					for i, bb := range b.Split(len(rules)) {
						applyRule(bb, rules[i])
					}
				}
			}
		}

		if len(nextBranches) <= 1 {
			break
		}

		var types grammar.BitSet
		for _, b = range nextBranches {
			if !b.NodeStack.IsEmpty() {
				types |= b.NodeStack.Top().types
			}
		}

		if types == 0 {
			break
		}

		var e error
		tok, e = pc.nextToken(ctx, types)
		if e != nil {
			break
		}
	}

	if len(nextBranches) != 0 {
		b = nextBranches[0]
	} else if lastSuccess != nil {
		b = lastSuccess
	} else {
		b = lastError
	}

	return tokens, b.AppliedRules
}
