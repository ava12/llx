package parser

import (
	"math/bits"
	"regexp"
	"sort"
	"strings"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/source"
)

type Token = lexer.Token

type TokenHook = func (token *Token, pc *ParseContext) (emit bool, e error)

type NonTermHookInstance interface {
	NewNonTerm (nonTerm string, token *Token) error
	HandleNonTerm (nonTerm string, result interface{}) error
	HandleToken (token *Token) error
	EndNonTerm () (result interface{}, e error)
}

type NonTermHook = func (nonTerm string, token *Token, pc *ParseContext) (NonTermHookInstance, error)

type defaultHookInstance struct {
	result interface{}
}

func (dhi *defaultHookInstance) NewNonTerm (nonTerm string, token *Token) error {
	return nil
}

func (dhi *defaultHookInstance) HandleNonTerm (nonTerm string, result interface{}) error {
	dhi.result = result
	return nil
}

func (dhi *defaultHookInstance) HandleToken (token *Token) error {
	return nil
}

func (dhi *defaultHookInstance) EndNonTerm () (result interface{}, e error) {
	return dhi.result, nil
}

const (
	AnyToken   = ""
	EofToken   = lexer.EofTokenName
	EoiToken   = lexer.EoiTokenName
	AnyNonTerm = ""
)

const any = -1

type TokenHooks map[string]TokenHook
type NonTermHooks map[string]NonTermHook

type Hooks struct {
	Tokens   TokenHooks
	Literals TokenHooks
	NonTerms NonTermHooks
}

type lexerRec struct {
	re    *regexp.Regexp
	types []lexer.TokenType
}

type Parser struct {
	grammar *grammar.Grammar
	names   map[string]int
	lexers  []lexerRec
}

func New (g *grammar.Grammar) *Parser {
	maxGroup := 0
	for _, t := range g.Tokens {
		mg := bits.Len(uint(t.Groups)) - 1
		if mg > maxGroup {
			maxGroup = mg
		}
	}
	lrs := make([]lexerRec, maxGroup + 1)
	names := make(map[string]int)
	names[tokenKey(AnyToken)] = grammar.AnyToken
	names[literalKey(AnyToken)] = grammar.AnyToken
	names[AnyNonTerm] = -1
	names[tokenKey(EofToken)] = lexer.EofTokenType
	names[tokenKey(EoiToken)] = lexer.EoiTokenType
	ms := make([][]string, maxGroup + 1)

	for i, t := range g.Tokens {
		if (t.Flags & grammar.LiteralToken) != 0 {
			names[literalKey(t.Name)] = i
		} else if (t.Flags & grammar.ErrorToken) == 0 {
			names[tokenKey(t.Name)] = i
		}
		if t.Re == "" {
			continue
		}

		group := -1
		gs := t.Groups
		mask := "(" + t.Re + ")"
		for ; gs != 0; gs >>= 1 {
			group++
			if (gs & 1) == 0 {
				continue
			}

			lrs[group].types = append(lrs[group].types, lexer.TokenType{i, t.Name})
			ms[group] = append(ms[group], mask)
		}
	}

	for i := range lrs {
		lrs[i].re = regexp.MustCompile("(?s:" + strings.Join(ms[i], "|") + ")")
	}

	for i, nt := range g.NonTerms {
		names[nt.Name] = i
	}

	return &Parser{g, names, lrs}
}

func tokenKey (name string) string {
	return "$" + name
}

func literalKey (text string) string {
	return ":" + text
}

func (p *Parser) Parse (q *source.Queue, hs *Hooks) (result interface{}, e error) {
	if hs == nil {
		hs = &Hooks{}
	}
	pc, e := newParseContext(p, q, hs)
	if e != nil {
		return nil, e
	}

	return pc.parse()
}

func (p *Parser) ParseString (name, content string, hs *Hooks) (result interface{}, e error) {
	q := source.NewQueue().Append(source.New(name, []byte(content)))
	return p.Parse(q, hs)
}

type nonTermRec struct {
	prev   *nonTermRec
	hook   NonTermHookInstance
	asides []*Token
	group  int
	index  int
	state  int
}

type ParseContext struct {
	parser       *Parser
	lexers       []*lexer.Lexer
	queue        *source.Queue
	tokenHooks   []TokenHook
	nonTermHooks []NonTermHook
	tokens       []*Token
	appliedRules []grammar.Rule
	tokenError   error
	lastResult   interface{}
	nonTerm      *nonTermRec
}

const (
	tokenHooksOffset   = -lexer.LowestTokenType
	nonTermHooksOffset = -grammar.AnyToken
)

func newParseContext (p *Parser, q *source.Queue, hs *Hooks) (*ParseContext, error) {
	result := &ParseContext{
		parser:       p,
		lexers:       make([]*lexer.Lexer, len(p.lexers)),
		queue:        q,
		tokenHooks:   make([]TokenHook, len(p.grammar.Tokens) + tokenHooksOffset),
		nonTermHooks: make([]NonTermHook, len(p.grammar.NonTerms) + nonTermHooksOffset),
		tokens:       make([]*Token, 0),
		appliedRules: make([]grammar.Rule, 0),
	}

	for i, lr := range p.lexers {
		result.lexers[i] = lexer.New(lr.re, lr.types, q)
	}

	for k, th := range hs.Tokens {
		i, f := p.names[tokenKey(k)]
		if !f {
			return nil, unknownTokenTypeError(k)
		}

		result.tokenHooks[i + tokenHooksOffset] = th
	}

	for k, th := range hs.Literals {
		i, f := p.names[literalKey(k)]
		if !f {
			return nil, unknownTokenLiteralError(k)
		}

		result.tokenHooks[i + tokenHooksOffset] = th
	}

	for k, nth := range hs.NonTerms {
		i, f := p.names[k]
		if !f {
			return nil, unknownNonTermError(k)
		}

		result.nonTermHooks[i + nonTermHooksOffset] = nth
	}

	e := result.pushNonTerm(grammar.RootNonTerm, lexer.NewToken(grammar.AnyToken, "", "", q.SourcePos()))
	return result, e
}

func (pc *ParseContext) TokenType (typeName string) (typ int, valid bool) {
	typ, valid = pc.parser.names[tokenKey(typeName)]
	return
}

func (pc *ParseContext) LiteralType (text string) (typ int, valid bool) {
	typ, valid = pc.parser.names[literalKey(text)]
	return
}

func (pc *ParseContext) NonTerminalIndex (name string) (index int, valid bool) {
	index, valid = pc.parser.names[name]
	return
}

func (pc *ParseContext) EmitToken (t *Token) error {
	if t.Type() >= len(pc.parser.grammar.Tokens) {
		return emitWrongTokenError(t)
	}

	pc.tokens = append(pc.tokens, t)
	return nil
}

func (pc *ParseContext) IncludeSource (s *source.Source) error {
	if len(pc.appliedRules) > 0 {
		var ntName string
		if pc.nonTerm != nil {
			ntName = pc.parser.grammar.NonTerms[pc.nonTerm.index].Name
		}
		return includeUnresolvedError(ntName, s.Name())
	}

	pc.queue.Prepend(s)
	return nil
}


func (pc *ParseContext) pushNonTerm (index int, tok *Token) error {
	e := pc.ntHandleAsides()
	if e != nil {
		return e
	}

	if pc.nonTerm != nil {
		e = pc.nonTerm.hook.NewNonTerm(pc.parser.grammar.NonTerms[index].Name, tok)
		if e != nil {
			return e
		}
	}

	gr := pc.parser.grammar
	nt := gr.NonTerms[index]
	hook, e := pc.getNonTermHook(index, tok)
	if e != nil {
		return e
	}

	pc.nonTerm = &nonTermRec{pc.nonTerm, hook, nil, gr.States[nt.FirstState].Group, index, nt.FirstState}
	return nil
}

func (pc *ParseContext) popNonTerm () error {
	var (
		e error
		res interface{}
	)
	nts := pc.parser.grammar.NonTerms

	asides := pc.nonTerm.asides
	pc.nonTerm.asides = nil

	for e == nil && pc.nonTerm != nil && pc.nonTerm.state == grammar.FinalState {
		nt := pc.nonTerm
		if nt.prev == nil {
			for _, t := range asides {
				e = nt.hook.HandleToken(t)
			}
			if e != nil {
				return e
			}
		}

		pc.nonTerm = nt.prev
		res, e = nt.hook.EndNonTerm()
		pc.lastResult = res
		if pc.nonTerm == nil {
			break
		}

		if e == nil {
			e = pc.nonTerm.hook.HandleNonTerm(nts[nt.index].Name, res)
		}
	}

	if e == nil && pc.nonTerm != nil {
		pc.nonTerm.asides = asides
	}

	return e
}

const repeatState = -128

func (pc *ParseContext) parse () (interface{}, error) {
	var (
		tok *Token
		e error
		tokenConsumed bool
	)
	gr := pc.parser.grammar

	for pc.nonTerm != nil {
		tok, e = pc.nextToken(pc.nonTerm.group)
		tokenConsumed = false
		if e != nil {
			return nil, e
		}

		for !tokenConsumed && pc.nonTerm != nil {
			nt := pc.nonTerm
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

			sameNonTerm := (rule.NonTerm == grammar.SameNonTerm)
			tokenConsumed = (sameNonTerm && rule.Token != grammar.AnyToken)
			if rule.State != repeatState {
				pc.nonTerm.state = rule.State
				if rule.State != grammar.FinalState {
					pc.nonTerm.group = gr.States[rule.State].Group
				}
			}

			if !sameNonTerm {
				e = pc.pushNonTerm(rule.NonTerm, tok)
			} else if tokenConsumed {
				e = pc.ntHandleToken(tok)
			}

			if e == nil && pc.nonTerm.state == grammar.FinalState {
				e = pc.popNonTerm()
			}

			if e != nil {
				return nil, e
			}
		}
	}

	if !tokenConsumed && tok.Type() != lexer.EoiTokenType {
		s := tok.Source()
		if s != nil {
			if s != pc.queue.Source() {
				pc.queue.Prepend(s)
			}
			pc.queue.Seek(s.Pos(tok.Line(), tok.Col()))
		}
	}

	return pc.lastResult, nil
}

func (pc *ParseContext) shrinkToken (tok *Token, group int) (bool, error) {
	if tok.Type() < 0 || pc.parser.grammar.Tokens[tok.Type()].Flags & grammar.ShrinkableToken == 0 {
		return false, nil
	}

	if len(pc.tokens) > 0 {
		return false, nil
	}

	res, e := pc.lexers[group].Shrink(tok)
	if res != nil && e == nil {
		e = pc.handleToken(res)
	}
	return (res != nil), e
}

func (pc *ParseContext) resolve (tok *Token, ars []grammar.Rule) ([]*Token, []grammar.Rule) {
	liveBranch := createBranches(pc, pc.nonTerm, ars)
	tokens := make([]*Token, 0)
	pc.tokens = append([]*Token{tok}, pc.tokens...)
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
				shrunk, _ := pc.shrinkToken(tok, deadBranch.nextGroup())
				if shrunk {
					tokens = tokens[: len(tokens) - 1]
					continue
				}
			}

			return tokens, liveBranch.applied
		}
	}
}

func (pc *ParseContext) getExpectedToken (s grammar.State) string {
	g := pc.parser.grammar
	index := 0
	if s.HighRule > s.LowRule {
		index = g.Rules[s.LowRule].Token
		if index < 0 && s.HighRule > s.LowRule + 1 {
			index = g.Rules[s.LowRule + 1].Token
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

func (pc *ParseContext) findRules (t *Token, s grammar.State) []grammar.Rule {
	if pc.isAsideToken(t) {
		return []grammar.Rule{{t.Type(), repeatState, grammar.SameNonTerm}}
	}

	keys := pc.possibleRuleKeys(t)
	g := pc.parser.grammar
	rules := g.Rules[s.LowRule : s.HighRule]
	multiRules := g.MultiRules[s.LowMultiRule : s.HighMultiRule]
	rlen := len(rules)
	mrlen := len(multiRules)

	for _, key := range keys {
		if key == grammar.AnyToken && rlen > 0 && rules[0].Token == key {
			return rules[0 : 1]
		}

		index := sort.Search(rlen, func (i int) bool {
			return rules[i].Token >= key
		})
		if index < rlen && rules[index].Token == key {
			return rules[index : index + 1]
		}

		index = sort.Search(mrlen, func (i int) bool {
			return multiRules[i].Token >= key
		})
		if index < mrlen && multiRules[index].Token == key {
			mr := multiRules[index]
			return g.Rules[mr.LowRule : mr.HighRule]
		}
	}

	return nil
}

func (pc *ParseContext) possibleRuleKeys (t *Token) []int {
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
		if tf & grammar.CaselessToken != 0 {
			literal = strings.ToUpper(literal)
		}
		literalIndex, literalFound = pc.parser.names[literalKey(literal)]
		literalFound = literalFound && (literalIndex >= 0)
		if literalFound {
			keys = append(keys, literalIndex)
		}
	}

	if !literalFound || literalIndex < 0 || (tokens[literalIndex].Flags & grammar.ReservedToken) == 0 {
		keys = append(keys, tt)
	}
	keys = append(keys, grammar.AnyToken)

	return keys
}

func (pc *ParseContext) getNonTermHook (ntIndex int, tok *Token) (res NonTermHookInstance, e error) {
	h := pc.nonTermHooks[ntIndex + nonTermHooksOffset]
	if h == nil {
		h = pc.nonTermHooks[any + nonTermHooksOffset]
	}
	if h != nil {
		res, e = h(pc.parser.grammar.NonTerms[ntIndex].Name, tok, pc)
	} else {
		e = nil
	}
	if res == nil {
		res = &defaultHookInstance{}
	}
	return
}

func (pc *ParseContext) nextToken (group int) (result *Token, e error) {
	if len(pc.tokens) > 0 {
		result = pc.tokens[0]
		pc.tokens = pc.tokens[1 :]
		if result.Type() >= 0 {
			groups := pc.parser.grammar.Tokens[result.Type()].Groups
			if groups & (1 << group) == 0 {
				e = unexpectedGroupError(result, group)
				result = nil
			}
		}
	} else if pc.tokenError != nil {
		e = pc.tokenError
	} else {
		result, e = pc.fetchToken(group)
	}

	return
}

func (pc *ParseContext) nextRule (t *Token, s grammar.State) (r grammar.Rule, found bool) {
	if len(pc.appliedRules) > 0 {
		r = pc.appliedRules[0]
		found = true
		pc.appliedRules = pc.appliedRules[1:]
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
		pc.tokens = append(tokens[1 :], pc.tokens...)
		pc.appliedRules = rules[1 :]
	}

	return
}

func (pc *ParseContext) handleToken (tok *Token) error {
	tts := make([]int, 0, 3)
	tt := tok.Type()

	if tt < 0 {
		tts = append(tts, tt)
	} else {
		i, f := pc.parser.names[literalKey(tok.Text())]
		if f {
			tts = append(tts, i)
		}
		tts = append(tts, tok.Type(), any)
	}

	var h TokenHook
	for _, i := range tts {
		h = pc.tokenHooks[i + tokenHooksOffset]
		if h != nil {
			break
		}
	}

	if h == nil {
		if pc.isAsideToken(tok) || tt == lexer.EofTokenType {
			return nil
		}

		pc.tokens = append(pc.tokens, tok)
		return nil
	}

	emit, e := h(tok, pc)
	if tt == lexer.EofTokenType {
		emit = false
	}
	if emit || tt < 0 {
		pc.tokens = append(pc.tokens, tok)
	}
	if e != nil {
		if len(pc.tokens) > 0 {
			pc.tokenError = e
		} else {
			return e
		}
	}

	return nil
}

func (pc *ParseContext) fetchToken (group int) (*Token, error) {
	for len(pc.tokens) == 0 {
		result, e := pc.lexers[group].Next()
		if result == nil || e != nil {
			return nil, e
		}

		e = pc.handleToken(result)
		if e != nil {
			return nil, e
		}
	}

	result := pc.tokens[0]
	pc.tokens = pc.tokens[1 :]
	return result, nil
}

func (pc *ParseContext) isAsideToken (t *Token) bool {
	tokens := pc.parser.grammar.Tokens
	i := t.Type()
	return (i >= 0 && i < len(tokens) && tokens[i].Flags & grammar.AsideToken != 0)
}

func (pc *ParseContext) ntHandleAsides () (res error) {
	ntr := pc.nonTerm
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

func (pc *ParseContext) ntHandleToken (tok *Token) (res error) {
	ntr := pc.nonTerm
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
