package parser

import (
	"math/bits"
	"regexp"
	"strings"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/source"
)

type TokenHook interface {
	HandleToken (token *lexer.Token, pc *ParseContext) (emit bool, e error)
}

type NontermHookInstance interface {
	HandleNonterm (nonterm string, result interface{}) error
	HandleToken (token *lexer.Token) error
	EndNonterm () (result interface{}, e error)
}

type NontermHook interface {
	BeginNonterm (nonterm string, pc *ParseContext) (NontermHookInstance, error)
}

type defaultHookInstance struct {
	result interface{}
}

func (dhi *defaultHookInstance) HandleNonterm (nonterm string, result interface{}) error {
	dhi.result = result
	return nil
}

func (dhi *defaultHookInstance) HandleToken (token *lexer.Token) error {
	return nil
}

func (dhi *defaultHookInstance) EndNonterm () (result interface{}, e error) {
	return dhi.result, nil
}

const AnyTokenType = -128
const AnyNonterm = ""

type TokenHooks map[int]TokenHook
type NontermHooks map[string]NontermHook

type Hooks struct {
	Tokens TokenHooks
	Nonterms NontermHooks
}

type lexerRec struct {
	re *regexp.Regexp
	types []lexer.TokenType
}

type Parser struct {
	grammar *grammar.Grammar
	literals map[string]int
	lexers []lexerRec
}

func New (g *grammar.Grammar) *Parser {
	maxGroup := 0
	for _, t := range g.Terms {
		mg := 1 << (bits.Len(uint(t.Groups)) - 1)
		if mg > maxGroup {
			maxGroup = mg
		}
	}
	lrs := make([]lexerRec, maxGroup + 1)
	ls := make(map[string]int)
	ms := make([][]string, maxGroup + 1)
	for i, t := range g.Terms {
		if (t.Flags & grammar.LiteralTerm) != 0 {
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

type nontermRec struct {
	prev *nontermRec
	lexer *lexer.Lexer
	states []grammar.State
	hook NontermHookInstance
	index, state int
}

type ParseContext struct {
	parser *Parser
	lexers []*lexer.Lexer
	queue *source.Queue
	tokenHooks TokenHooks
	nontermHooks NontermHooks
	tokens []*lexer.Token
	lastResult interface{}

	nonterm *nontermRec
}

func newParseContext (p *Parser, q *source.Queue, hs *Hooks) (*ParseContext, error) {
	result := &ParseContext{
		parser: p,
		lexers: make([]*lexer.Lexer, len(p.lexers)),
		queue: q,
		tokenHooks: hs.Tokens,
		nontermHooks: hs.Nonterms,
		tokens: make([]*lexer.Token, 0),
	}

	for i, lr := range p.lexers {
		result.lexers[i] = lexer.New(lr.re, lr.types, q)
	}

	e := result.pushNonterm(grammar.RootNonterm)
	return result, e
}


func (pc *ParseContext) EmitToken (t *lexer.Token) {
	pc.tokens = append(pc.tokens, t)
}


func (pc *ParseContext) pushNonterm (index int) error {
	nt := pc.parser.grammar.Nonterms[index]
	hook, e := pc.getNontermHook(nt.Name)
	if e != nil {
		return e
	}

	pc.nonterm = &nontermRec{pc.nonterm, pc.lexers[nt.States[0].Group], nt.States, hook, index, grammar.InitialState}
	return nil
}

func (pc *ParseContext) popNonterm () error {
	var (
		e error
		res interface{}
	)

	for e == nil && pc.nonterm != nil && pc.nonterm.state == grammar.FinalState {
		nt := pc.nonterm
		pc.nonterm = nt.prev
		res, e = nt.hook.EndNonterm()
		pc.lastResult = res
		if pc.nonterm == nil {
			break
		}

		if e == nil {
			e = pc.nonterm.hook.HandleNonterm(pc.parser.grammar.Nonterms[nt.index].Name, res)
		}
	}

	return e
}

type appliedRule struct {
	term, state, nonterm int
}

const repeatState = -128


func (pc *ParseContext) parse () (interface{}, error) {
	var (
		tok *lexer.Token
		e error
	)
	tokenConsumed := true
	for pc.nonterm != nil {
		if tokenConsumed {
			tok, e = pc.nextToken()
			if e != nil {
				return nil, e
			}
		}

		nt := pc.nonterm
		rules := pc.findRules(tok, nt.states[nt.state])
		if rules == nil {
			expected := pc.getExpectedTerm(nt.states[nt.state])
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
				tok, e = pc.nextToken()
				if e != nil {
					return nil, e
				}

				if tok.Type() == lexer.EofTokenType {
					tokenConsumed = false
				}
			}

			sameNonterm := (rule.nonterm == grammar.SameNonterm)
			tokenConsumed = (sameNonterm && rule.term != grammar.AnyTerm)
			if rule.state != repeatState {
				pc.nonterm.state = rule.state
				if rule.state != grammar.FinalState {
					pc.nonterm.lexer = pc.lexers[pc.nonterm.states[rule.state].Group]
				}
			}

			if !sameNonterm {
				e = pc.pushNonterm(rule.nonterm)
			} else if tokenConsumed {
				e = pc.nonterm.hook.HandleToken(tok)
			}

			if e == nil && pc.nonterm.state == grammar.FinalState {
				e = pc.popNonterm()
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

func (pc *ParseContext) resolve (tok *lexer.Token, ars []appliedRule) []appliedRule {
	firstBranch := createBranches(pc, pc.nonterm, ars)
	tokens := make([]*lexer.Token, 0)
	pc.tokens = append([]*lexer.Token{tok}, pc.tokens...)
	for {
		var parentBranch *branch
		parentBranch = nil
		defaultBranch := firstBranch
		currentBranch := firstBranch
		liveCnt := 0
		tok, e := pc.nextToken()
		if e != nil || tok == nil {
			pc.tokens = append(pc.tokens, tokens...)
			return firstBranch.applied
		}

		tokens = append(tokens, tok)

		for currentBranch != nil {
			if currentBranch.applyToken(tok) {
				liveCnt++
				parentBranch = currentBranch
			} else {
				if parentBranch == nil {
					firstBranch = currentBranch.next
				} else {
					parentBranch.next = currentBranch.next
				}
			}
			currentBranch = currentBranch.next
		}

		if liveCnt < 2 {
			if liveCnt == 0 {
				firstBranch = defaultBranch
			}
			pc.tokens = append(tokens, pc.tokens...)
			return firstBranch.applied
		}
	}
}

func (pc *ParseContext) getExpectedTerm (s grammar.State) string {
	var i int
	for i = 0; i < len(pc.parser.grammar.Terms); i++ {
		_, f := s.Rules[i]
		if !f {
			_, f = s.MultiRules[i]
		}
		if f {
			break
		}
	}
	term := pc.parser.grammar.Terms[i]
	if (term.Flags & grammar.LiteralTerm) != 0 {
		return term.Name
	} else {
		return "$" + term.Name
	}
}

func (pc *ParseContext) findRules (t *lexer.Token, s grammar.State) []appliedRule {
	if pc.isAsideToken(t) {
		return []appliedRule{{t.Type(), repeatState, grammar.SameNonterm}}
	}

	indexes := make([]int, 0, 3)
	index, f := pc.parser.literals[t.Text()]
	if f {
		indexes = append(indexes, index)
	}
	indexes = append(indexes, t.Type(), grammar.AnyTerm)
	for _, index = range indexes {
		r, f := s.Rules[index]
		if f {
			return []appliedRule{{index, r.State, r.Nonterm}}
		}

		rs := s.MultiRules[index]
		if rs != nil {
			result := make([]appliedRule, len(rs))
			for i, r := range rs {
				result[i] = appliedRule{index, r.State, r.Nonterm}
			}
			return result
		}
	}

	return nil
}

func (pc *ParseContext) getNontermHook (nonterm string) (res NontermHookInstance, e error) {
	h, f := pc.nontermHooks[nonterm]
	if !f {
		h, f = pc.nontermHooks[AnyNonterm]
	}
	if f {
		res, e = h.BeginNonterm(nonterm, pc)
	} else {
		res = &defaultHookInstance{}
		e = nil
	}
	return
}

func (pc *ParseContext) nextToken () (result *lexer.Token, e error) {
	if len(pc.tokens) > 0 {
		result = pc.tokens[0]
		pc.tokens = pc.tokens[1 :]
	} else {
		result, e = pc.fetchToken(pc.nonterm.lexer)
	}

	return
}

func (pc *ParseContext) fetchToken (l *lexer.Lexer) (*lexer.Token, error) {
	for len(pc.tokens) == 0 {
		result, e := l.Next()
		if result == nil || e != nil {
			return nil, e
		}

		h, f := pc.tokenHooks[result.Type()]
		if !f {
			h, f = pc.tokenHooks[AnyTokenType]
		}
		if !f {
			if pc.isAsideToken(result) {
				continue
			}

			return result, nil
		}

		emit, e := h.HandleToken(result, pc)
		if e != nil {
			return nil, e
		}

		if emit {
			pc.tokens = append(pc.tokens, result)
		}
	}

	result := pc.tokens[0]
	pc.tokens = pc.tokens[1 :]
	return result, nil
}

func (pc *ParseContext) isAsideToken (t *lexer.Token) bool {
	terms := pc.parser.grammar.Terms
	i := t.Type()
	return (i >= 0 && i < len(terms) && terms[i].Flags & grammar.AsideTerm != 0)
}
