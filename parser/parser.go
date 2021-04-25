package parser

import (
	"math/bits"
	"regexp"
	"strings"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/source"
)

type TokenHook = func (token *lexer.Token, pc *ParseContext) (emit bool, e error)

type NonTermHookInstance interface {
	HandleNonTerm (nonTerm string, result interface{}) error
	HandleToken (token *lexer.Token) error
	EndNonTerm () (result interface{}, e error)
}

type NonTermHook = func (nonTerm string, pc *ParseContext) (NonTermHookInstance, error)

type defaultHookInstance struct {
	result interface{}
}

func (dhi *defaultHookInstance) HandleNonTerm (nonTerm string, result interface{}) error {
	dhi.result = result
	return nil
}

func (dhi *defaultHookInstance) HandleToken (token *lexer.Token) error {
	return nil
}

func (dhi *defaultHookInstance) EndNonTerm () (result interface{}, e error) {
	return dhi.result, nil
}

const AnyTokenType = -128
const AnyNonTerm = ""

type TokenHooks map[int]TokenHook
type NonTermHooks map[string]NonTermHook

type Hooks struct {
	Tokens   TokenHooks
	NonTerms NonTermHooks
}

type lexerRec struct {
	re    *regexp.Regexp
	types []lexer.TokenType
}

type Parser struct {
	grammar  *grammar.Grammar
	literals map[string]int
	lexers   []lexerRec
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
	ls := make(map[string]int)
	ms := make([][]string, maxGroup + 1)
	for i, t := range g.Tokens {
		if (t.Flags & grammar.LiteralToken) != 0 {
			ls[t.Name] = i
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

	return &Parser{g, ls, lrs}
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

type nonTermRec struct {
	prev  *nonTermRec
	hook  NonTermHookInstance
	group int
	index int
	state int
}

type ParseContext struct {
	parser       *Parser
	lexers       []*lexer.Lexer
	queue        *source.Queue
	tokenHooks   TokenHooks
	nonTermHooks NonTermHooks
	tokens       []*lexer.Token
	lastResult   interface{}
	nonTerm      *nonTermRec
}

func newParseContext (p *Parser, q *source.Queue, hs *Hooks) (*ParseContext, error) {
	result := &ParseContext{
		parser:       p,
		lexers:       make([]*lexer.Lexer, len(p.lexers)),
		queue:        q,
		tokenHooks:   hs.Tokens,
		nonTermHooks: hs.NonTerms,
		tokens:       make([]*lexer.Token, 0),
	}

	for i, lr := range p.lexers {
		result.lexers[i] = lexer.New(lr.re, lr.types, q)
	}

	e := result.pushNonTerm(grammar.RootNonTerm)
	return result, e
}


func (pc *ParseContext) EmitToken (t *lexer.Token) error {
	if t.Type() >= len(pc.parser.grammar.Tokens) {
		return emitWrongTokenError(t)
	}

	pc.tokens = append(pc.tokens, t)
	return nil
}


func (pc *ParseContext) pushNonTerm (index int) error {
	gr := pc.parser.grammar
	nt := gr.NonTerms[index]
	hook, e := pc.getNonTermHook(nt.Name)
	if e != nil {
		return e
	}

	pc.nonTerm = &nonTermRec{pc.nonTerm, hook, gr.States[nt.FirstState].Group, index, nt.FirstState}
	return nil
}

func (pc *ParseContext) popNonTerm () error {
	var (
		e error
		res interface{}
	)

	for e == nil && pc.nonTerm != nil && pc.nonTerm.state == grammar.FinalState {
		nt := pc.nonTerm
		pc.nonTerm = nt.prev
		res, e = nt.hook.EndNonTerm()
		pc.lastResult = res
		if pc.nonTerm == nil {
			break
		}

		if e == nil {
			e = pc.nonTerm.hook.HandleNonTerm(pc.parser.grammar.NonTerms[nt.index].Name, res)
		}
	}

	return e
}

type appliedRule struct {
	token, state, nonTerm int
}

const repeatState = -128


func (pc *ParseContext) parse () (interface{}, error) {
	var (
		tok *lexer.Token
		e error
	)
	gr := pc.parser.grammar
	tokenConsumed := true
	for pc.nonTerm != nil {
		if tokenConsumed {
			tok, e = pc.nextToken(pc.nonTerm.group)
			tokenConsumed = false
			if e != nil {
				return nil, e
			}
		}

		nt := pc.nonTerm
		rules := pc.findRules(tok, gr.States[nt.state])
		if rules == nil {
			shrunk, e := pc.shrinkToken(tok, nt.group)
			if e != nil {
				return nil, e
			}

			if shrunk {
				tokenConsumed = true
				continue
			}

			expected := pc.getExpectedToken(gr.States[nt.state])
			if tok.Type() == lexer.EofTokenType {
				e = unexpectedEofError(tok, expected)
			} else {
				e = unexpectedTokenError(tok, expected)
			}
			return nil, e
		}

		if len(rules) > 1 {
			rules = pc.resolve(tok, rules)
			tokenConsumed = true
		} else {
			tokenConsumed = false
		}

		for _, rule := range rules {
			if tokenConsumed {
				tok, e = pc.nextToken(pc.nonTerm.group)
				if e != nil {
					return nil, e
				}

				if tok.Type() == lexer.EofTokenType {
					tokenConsumed = false
				}
			}

			sameNonTerm := (rule.nonTerm == grammar.SameNonTerm)
			tokenConsumed = (sameNonTerm && rule.token != grammar.AnyToken)
			if rule.state != repeatState {
				pc.nonTerm.state = rule.state
				if rule.state != grammar.FinalState {
					pc.nonTerm.group = gr.States[rule.state].Group
				}
			}

			if !sameNonTerm {
				e = pc.pushNonTerm(rule.nonTerm)
			} else if tokenConsumed {
				e = pc.nonTerm.hook.HandleToken(tok)
			}

			if e == nil && pc.nonTerm.state == grammar.FinalState {
				e = pc.popNonTerm()
			}

			if e != nil {
				return nil, e
			}
		}
	}

	if !tokenConsumed {
		s := tok.Source()
		if s != pc.queue.Source() {
			pc.queue.Prepend(s)
		}
		pc.queue.Seek(s.Pos(tok.Line(), tok.Col()))
	}

	return pc.lastResult, nil
}

func (pc *ParseContext) shrinkToken (tok *lexer.Token, group int) (bool, error) {
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

func (pc *ParseContext) resolve (tok *lexer.Token, ars []appliedRule) []appliedRule {
	liveBranch := createBranches(pc, pc.nonTerm, ars)
	tokens := make([]*lexer.Token, 0)
	pc.tokens = append([]*lexer.Token{tok}, pc.tokens...)
	for {
		var parentBranch, deadBranch, lastDead *branch
		currentBranch := liveBranch
		deadBranch = nil
		lastDead = nil
		survivors := 0
		tok, e := pc.nextToken(liveBranch.nextGroup())
		if e != nil || tok == nil {
			pc.tokens = append(pc.tokens, tokens...)
			return liveBranch.applied
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

			pc.tokens = append(tokens, pc.tokens...)
			return liveBranch.applied
		}
	}
}

func (pc *ParseContext) getExpectedToken (s grammar.State) string {
	var i int
	for i = 0; i < len(pc.parser.grammar.Tokens); i++ {
		_, f := s.Rules[i]
		if !f {
			_, f = s.MultiRules[i]
		}
		if f {
			break
		}
	}
	token := pc.parser.grammar.Tokens[i]
	if (token.Flags & grammar.LiteralToken) != 0 {
		return token.Name
	} else {
		return "$" + token.Name
	}
}

func (pc *ParseContext) findRules (t *lexer.Token, s grammar.State) []appliedRule {
	if pc.isAsideToken(t) {
		return []appliedRule{{t.Type(), repeatState, grammar.SameNonTerm}}
	}

	indexes := make([]int, 0, 3)
	index, f := pc.parser.literals[t.Text()]
	if f {
		indexes = append(indexes, index)
	}
	indexes = append(indexes, t.Type(), grammar.AnyToken)
	for _, index = range indexes {
		r, f := s.Rules[index]
		if f {
			return []appliedRule{{index, r.State, r.NonTerm}}
		}

		rs := s.MultiRules[index]
		if len(rs) != 0 {
			result := make([]appliedRule, len(rs))
			for i, r := range rs {
				result[i] = appliedRule{index, r.State, r.NonTerm}
			}
			return result
		}
	}

	return nil
}

func (pc *ParseContext) getNonTermHook (nonTerm string) (res NonTermHookInstance, e error) {
	h, f := pc.nonTermHooks[nonTerm]
	if !f {
		h, f = pc.nonTermHooks[AnyNonTerm]
	}
	if f {
		res, e = h(nonTerm, pc)
	} else {
		e = nil
	}
	if res == nil {
		res = &defaultHookInstance{}
	}
	return
}

func (pc *ParseContext) nextToken (group int) (result *lexer.Token, e error) {
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
	} else {
		result, e = pc.fetchToken(group)
	}

	return
}

func (pc *ParseContext) handleToken (tok *lexer.Token) error {
	tts := make([]int, 0, 3)
	tt := tok.Type()

	if tt < 0 {
		tts = append(tts, tt)
	} else {
		i, f := pc.parser.literals[tok.Text()]
		if f {
			tts = append(tts, i)
		}
		tts = append(tts, tok.Type(), AnyTokenType)
	}

	var h TokenHook
	for _, i := range tts {
		h = pc.tokenHooks[i]
		if h != nil {
			break
		}
	}

	if h == nil {
		if pc.isAsideToken(tok) {
			return nil
		}

		pc.tokens = append(pc.tokens, tok)
		return nil
	}

	emit, e := h(tok, pc)
	if e != nil {
		return e
	}

	if emit || tt < 0 {
		pc.tokens = append(pc.tokens, tok)
	}
	return nil
}

func (pc *ParseContext) fetchToken (group int) (*lexer.Token, error) {
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

func (pc *ParseContext) isAsideToken (t *lexer.Token) bool {
	tokens := pc.parser.grammar.Tokens
	i := t.Type()
	return (i >= 0 && i < len(tokens) && tokens[i].Flags & grammar.AsideToken != 0)
}
