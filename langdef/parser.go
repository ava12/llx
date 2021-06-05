package langdef

import (
	"math/bits"
	"regexp"

	"github.com/ava12/llx"
	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/source"
)

const (
	unusedToken = grammar.AsideToken | grammar.ErrorToken
	noGroup     = -1
	maxGroup    = 30
)


type chunk interface {
	FirstTokens () llx.IntSet
	IsOptional () bool
	BuildStates (g *grammar.Grammar, stateIndex, nextIndex int) error
}

type complexChunk interface {
	chunk
	Append (chunk)
}

func ParseString (name, content string) (*grammar.Grammar, error) {
	return Parse(source.New(name, []byte(content)))
}

func ParseBytes (name string, content []byte) (*grammar.Grammar, error) {
	return Parse(source.New(name, content))
}

type nonTermItem struct {
	Index       int
	DependsOn   llx.IntSet
	FirstTokens llx.IntSet
	Chunk       *groupChunk
}

type tokenIndex map[string]int
type nonTermIndex map[string]*nonTermItem


func Parse (s *source.Source) (*grammar.Grammar, error) {
	result := &grammar.Grammar{
		Tokens:   make([]grammar.Token, 0),
		NonTerms: make([]grammar.NonTerm, 0),
	}

	nti := nonTermIndex{}
	var e error

	e = parseLangDef(s, result, nti)
	e = assignTokenGroups(result, e)
	e = findUndefinedNonTerminals(nti, e)
	e = findUnusedNonTerminals(result.NonTerms, nti, e)
	e = resolveDependencies(result.NonTerms, nti, e)
	e = buildStates(result, nti, e)
	e = findRecursions(result, e)
	e = assignStateGroups(result, e)

	if e != nil {
		return nil, e
	}

	return result, nil
}

const (
	stringTok     = "string"
	nameTok       = "name"
	dirTok        = "dir"
	literalDirTok = "literal"
	groupDirTok   = "group-dir"
	tokenNameTok  = "token-name"
	regexpTok     = "regexp"
	opTok         = "op"
	wrongTok      = ""
)

const (
	equTok       = "="
	commaTok     = ","
	semicolonTok = ";"
	pipeTok      = "|"
	lBraceTok    = "("
	rBraceTok    = ")"
	lSquareTok   = "["
	rSquareTok   = "]"
	lCurlyTok    = "{"
	rCurlyTok    = "}"
)

var (
	tokenTypes []lexer.TokenType
)

type extraToken struct {
	name   string
	groups int
	flags  grammar.TokenFlags
}

type parseContext struct {
	l            *lexer.Lexer
	g            *grammar.Grammar
	lts          []string
	ti, lti      tokenIndex
	nti          nonTermIndex
	ets          []extraToken
	eti          map[string]int
	currentGroup int
}

func init () {
	tokenTypes = []lexer.TokenType {
		{1, stringTok},
		{2, nameTok},
		{3, dirTok},
		{4, literalDirTok},
		{5, groupDirTok},
		{6, tokenNameTok},
		{7, regexpTok},
		{8, opTok},
		{lexer.ErrorTokenType, wrongTok},
	}
}

func parseLangDef (s *source.Source, g *grammar.Grammar, nti nonTermIndex) error {
	var e error

	re := regexp.MustCompile(
		"\\s+|#.*?(?:\\n|$)|" +
		"((?:\".*?\")|(?:'.*?'))|" +
		"([a-zA-Z_][a-zA-Z_0-9-]*)|" +
		"(!(?:aside|error|extern|shrink)\\b)|" +
		"(!literal\\b)|" +
		"(!group\\b)|" +
		"(\\$[a-zA-Z_][a-zA-Z_0-9-]*)|" +
		"(/(?:[^\\\\/]|\\\\.)+/)|" +
		"([(){}\\[\\]=|,;])|" +
		"(['\"/!].{0,10})")

	l := lexer.New(re, tokenTypes, source.NewQueue().Append(s))
	ets := make([]extraToken, 0)
	eti := make(map[string]int)
	ti := tokenIndex{}
	lti := tokenIndex{}
	c := &parseContext{l, g, make([]string, 0), ti, lti, nti, ets, eti, 0}

	var t *lexer.Token
	for e == nil {
		t, e = fetch(l, []string{nameTok, dirTok, groupDirTok, literalDirTok, tokenNameTok}, true, nil)
		if e != nil {
			return e
		}

		if t == nil || t.TypeName() == nameTok {
			break
		}

		switch t.TypeName() {
		case dirTok:
			e = parseDir(t.Text(), c)

		case groupDirTok:
			if c.currentGroup > maxGroup {
				e = groupNumberError(t)
			} else {
				e = parseGroupDir(c)
			}

		case literalDirTok:
			e = parseLiteralDir(c)

		case tokenNameTok:
			name := t.Text()[1:]
			i, has := ti[name]
			if has && g.Tokens[i].Re != "" {
				return defTokenError(t)
			}
			e = parseTokenDef(name, c)
		}
	}
	if e != nil {
		return e
	}

	for _, et := range c.ets {
		_, has := c.eti[et.name]
		if has {
			if et.flags & grammar.ExternalToken != 0 {
				addToken(et.name, "", et.groups, et.flags, c)
			} else {
				return undefinedTokenError(et.name)
			}
		}
	}

	firstLiteral := len(c.g.Tokens)
	for i, name := range c.lts {
		addLiteralToken(name, c)
		c.lti[name] = i + firstLiteral
	}

	for e == nil && t != nil && t.Type() != lexer.EofTokenType {
		_, has := nti[t.Text()]
		if has && nti[t.Text()].Chunk != nil {
			return defNonTermError(t)
		}

		e = parseNonTermDef(t.Text(), c)
		if e == nil {
			t, e = fetch(l, []string{nameTok, lexer.EofTokenName}, true, nil)
		}
	}

	return e
}

var savedToken *lexer.Token

func put (t *lexer.Token) {
	if savedToken != nil {
		panic("cannot put " + t.TypeName() + " token: already put " + savedToken.TypeName())
	}

	savedToken = t
}

func fetch (l *lexer.Lexer, types []string, strict bool, e error) (*lexer.Token, error) {
	if e != nil {
		return nil, e
	}

	token := savedToken
	if token == nil {
		token, e = l.Next()
		if e != nil {
			return nil, e
		}

	} else {
		savedToken = nil
	}

	if token == nil {
		token = lexer.EofToken(l.Source())
	}

	for _, typ := range types {
		if token.TypeName() == typ || token.Text() == typ {
			return token, nil
		}
	}

	if token.Type() == lexer.EofTokenType {
		if strict {
			return nil, eofError(token)
		} else {
			return token, nil
		}
	}

	if strict {
		return nil, unexpectedTokenError(token)
	}

	put(token)
	return nil, nil
}

func fetchOne (l *lexer.Lexer, typ string, strict bool, e error) (*lexer.Token, error) {
	return fetch(l, []string{typ}, strict, e)
}

func fetchAll (l *lexer.Lexer, types []string, e error) ([]*lexer.Token, error) {
	if e != nil {
		return nil, e
	}

	result := make([]*lexer.Token, 0)
	for {
		t, e := fetch(l, types, false, nil)
		if e != nil {
			return nil, e
		}

		if t == nil {
			break
		}

		result = append(result, t)
	}

	return result, nil
}

func skip (l *lexer.Lexer, types []string, e error) error {
	if e != nil {
		return e
	}

	_, e = fetch(l, types, true, nil)
	return e
}

func skipOne (l *lexer.Lexer, typ string, e error) error {
	return skip(l, []string{typ}, e)
}

func addToken (name, re string, groups int, flags grammar.TokenFlags, c *parseContext) int {
	var t extraToken
	i, has := c.eti[name]
	if has {
		t = c.ets[i]
		delete(c.eti, name)
	}
	c.g.Tokens = append(c.g.Tokens, grammar.Token{name, re, groups | t.groups, flags | t.flags})
	index := len(c.g.Tokens) - 1
	c.ti[name] = index
	return index
}

func addLiteralToken (name string, c *parseContext) int {
	i, has := c.lti[name]
	if has {
		return i
	}

	i = len(c.g.Tokens)
	c.g.Tokens = append(c.g.Tokens, grammar.Token{name, "", 0, grammar.LiteralToken})
	c.lti[name] = i
	return i
}

func addExtraToken (name string, c *parseContext) int {
	i, has := c.eti[name]
	if !has {
		i = len(c.ets)
		c.ets = append(c.ets, extraToken{name : name})
		c.eti[name] = i
	}
	return i
}

func addTokenFlag (name string, flag grammar.TokenFlags, c *parseContext) {
	i, has := c.ti[name]
	if has {
		c.g.Tokens[i].Flags |= flag
	} else {
		i = addExtraToken(name, c)
		c.ets[i].flags |= flag
	}
}

func addTokenGroups (token *lexer.Token, groups int, c *parseContext) {
	name := token.Text()[1 :]
	i, has := c.ti[name]
	if has {
		c.g.Tokens[i].Groups |= groups
	} else {
		i = addExtraToken(name, c)
		c.ets[i].groups |= groups
	}
}

func parseDir (name string, c *parseContext) error {
	tokens, e := fetchAll(c.l, []string{tokenNameTok}, nil)
	e = skipOne(c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	var flag grammar.TokenFlags = 0
	switch name {
	case "!aside":
		flag = grammar.AsideToken
	case "!extern":
		flag = grammar.ExternalToken
	case "!error":
		flag = grammar.ErrorToken
	case "!shrink":
		flag = grammar.ShrinkableToken
	}
	for _, token := range tokens {
		addTokenFlag(token.Text()[1 :], flag, c)
	}

	return nil
}

func parseGroupDir (c *parseContext) error {
	tokens, e := fetchAll(c.l, []string{tokenNameTok}, nil)
	e = skipOne(c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	if len(tokens) == 0 {
		return nil
	}

	groups := 1 << c.currentGroup
	for _, token := range tokens {
		addTokenGroups(token, groups, c)
	}

	c.currentGroup++
	return nil
}

func parseLiteralDir (c *parseContext) error {
	tokens, e := fetchAll(c.l, []string{stringTok}, nil)
	e = skipOne(c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	for _, t := range tokens {
		text := t.Text()
		addLiteralToken(text[1 : len(text) - 1], c)
	}
	return nil
}

func parseTokenDef (name string, c *parseContext) error {
	e := skipOne(c.l, equTok, nil)
	token, e := fetchOne(c.l, regexpTok, true, e)
	e = skipOne(c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	re := token.Text()[1 : len(token.Text()) - 1]
	_, e = regexp.Compile(re)
	if e != nil {
		return regexpError(token, e)
	}

	addToken(name, re, 0, 0, c)

	return nil
}

func addNonTerm (name string, c *parseContext, define bool) *nonTermItem {
	var group *groupChunk = nil
	if define {
		group = newGroupChunk(false, false)
	}
	result := c.nti[name]
	if result != nil {
		if result.Chunk == nil && define {
			result.Chunk = group
		}
		return result
	}

	result = &nonTermItem{len(c.g.NonTerms), llx.NewIntSet(), llx.NewIntSet(), group}
	c.nti[name] = result
	c.g.NonTerms = append(c.g.NonTerms, grammar.NonTerm{name, 0})
	return result
}

func parseNonTermDef (name string, c *parseContext) error {
	nt := addNonTerm(name, c, true)
	e := skipOne(c.l, equTok, nil)
	e = parseGroup(name, nt.Chunk, c, e)
	e = skipOne(c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	return nil
}

func parseGroup (name string, group complexChunk, c *parseContext, e error) error {
	if e != nil {
		return e
	}

	for {
		item, e := parseVariants(name, c)
		if e != nil {
			return e
		}

		group.Append(item)
		t, e := fetchOne(c.l, commaTok, false, nil)
		if t == nil {
			return e
		}
	}
}

func parseVariants (name string, c *parseContext) (chunk, error) {
	ch, e := parseVariant(name, c)
	t, e := fetchOne(c.l, pipeTok, false, e)
	if e != nil {
		return nil, e
	} else if t == nil {
		return ch, nil
	}

	result := newVariantChunk()
	result.Append(ch)

	for {
		ch, e = parseVariant(name, c)
		t, e = fetchOne(c.l, pipeTok, false, e)
		if e != nil {
			return nil, e
		}

		result.Append(ch)
		if t == nil {
			break
		}
	}

	return result, nil
}

func parseVariant (name string, c *parseContext) (chunk, error) {
	variantHeads := []string{nameTok, tokenNameTok, stringTok, lBraceTok, lSquareTok, lCurlyTok}
	t, e := fetch(c.l, variantHeads, true, nil)
	if e != nil {
		return nil, e
	}

	var (
		index int
		f bool
	)
	switch t.TypeName() {
	case nameTok:
		nt := addNonTerm(t.Text(), c, false)
		c.nti[name].DependsOn.Add(nt.Index)
		return newNonTermChunk(t.Text(), nt), nil

	case tokenNameTok:
		index, f = c.ti[t.Text()[1 :]]
		if !f {
			return nil, tokenError(t)
		}

		if (c.g.Tokens[index].Flags & unusedToken) != 0 {
			return nil, wrongTokenError(t)
		}

		return newTokenChunk(index), nil

	case stringTok:
		name = t.Text()[1 : len(t.Text()) - 1]
		index, f = c.lti[name]
		if !f {
			index = addLiteralToken(name, c)
		}
		return newTokenChunk(index), nil
	}

	repeated := (t.Text() == "{")
	optional := (t.Text() != "(")
	var lastToken string
	if repeated {
		lastToken = rCurlyTok
	} else if optional {
		lastToken = rSquareTok
	} else {
		lastToken = rBraceTok
	}

	result := newGroupChunk(optional, repeated)
	e = parseGroup(name, result, c, nil)
	e = skipOne(c.l, lastToken, e)
	if e != nil {
		return nil, e
	}

	return result, nil
}

func findUndefinedNonTerminals (nti nonTermIndex, e error) error {
	if e != nil {
		return e
	}

	uns := make([]string, 0)
	for name, item := range nti {
		if item.Chunk == nil {
			uns = append(uns, name)
		}
	}

	if len(uns) > 0 {
		return unknownNonTermError(uns)
	}

	return nil
}

func findUnusedNonTerminals (nts []grammar.NonTerm, nti nonTermIndex, e error) error {
	if e != nil {
		return e
	}

	unreachedNts := llx.NewIntSet()
	for i := 0; i < len(nts); i++ {
		unreachedNts.Add(i)
	}
	searchQueue := llx.NewIntQueue(0)
	for !searchQueue.IsEmpty() {
		index := searchQueue.Head()
		if !unreachedNts.Contains(index) {
			continue
		}

		unreachedNts.Remove(index)
		for _, i := range nti[nts[index].Name].DependsOn.ToSlice() {
			searchQueue.Append(i)
		}
	}

	if unreachedNts.IsEmpty() {
		return nil
	} else {
		return unusedNonTermError(nonTermNames(nts, unreachedNts))
	}
}

func resolveDependencies (nts []grammar.NonTerm, nti nonTermIndex, e error) error {
	if e != nil {
		return e
	}

	affects := make(map[int][]int)
	queue := llx.NewIntQueue()

	for _, item := range nti {
		if item.DependsOn.IsEmpty() {
			queue.Append(item.Index)
			item.FirstTokens = item.Chunk.FirstTokens()
			continue
		}

		for _, k := range item.DependsOn.ToSlice() {
			_, has := affects[k]
			if !has {
				affects[k] = []int{item.Index}
			} else {
				affects[k] = append(affects[k], item.Index)
			}
		}
	}

	for !queue.IsEmpty() {
		k := queue.Head()
		for _, index := range affects[k] {
			item := nti[nts[index].Name]
			item.DependsOn.Remove(k)
			if item.DependsOn.IsEmpty() {
				queue.Append(index)
				item.FirstTokens = item.Chunk.FirstTokens()
			}
		}
	}

	for _, item := range nti {
		if !item.DependsOn.IsEmpty() {
			queue.Append(item.Index)
		}
	}

	if queue.IsEmpty() {
		return nil
	}

	indexes := queue.Items()
	tokenAdded := true
	for tokenAdded {
		tokenAdded = false

		for _, index := range indexes {
			item := nti[nts[index].Name]
			firstTokens := item.Chunk.FirstTokens()
			if !firstTokens.IsEmpty() && !llx.Subtract(firstTokens, item.FirstTokens).IsEmpty() {
				item.FirstTokens.Union(firstTokens)
				tokenAdded = true
			}
		}
	}

	names := make([]string, 0, len(indexes))
	for _, index := range indexes {
		name := nts[index].Name
		if nti[name].FirstTokens.IsEmpty() {
			names = append(names, name)
		}
	}
	if len(names) > 0 {
		return unresolvedError(names)
	}

	return nil
}

func buildStates (g *grammar.Grammar, nti nonTermIndex, e error) error {
	if e != nil {
		return e
	}

	for i, nt := range g.NonTerms {
		firstState := len(g.States)
		g.States = append(g.States, grammar.State{
			noGroup,
			map[int]grammar.Rule{},
			map[int][]grammar.Rule{},
		})
		item := nti[nt.Name]
		g.NonTerms[i].FirstState = firstState
		e = item.Chunk.BuildStates(g, firstState, grammar.FinalState)
		if e != nil {
			ee, f := e.(*llx.Error)
			if f && ee.Code == EmptyRepeatableError {
				e = emptyRepeatableError(nt.Name)
			}
			return e
		}

		for i, state := range g.States[firstState :] {
			if len(state.Rules) == 0 {
				g.States[i + firstState].Rules = nil
			}
			if len(state.MultiRules) == 0 {
				g.States[i + firstState].MultiRules = nil
			}
		}
	}

	return nil
}

func findRecursions (g *grammar.Grammar, e error) error {
	if e != nil {
		return e
	}

	ntis := llx.NewIntSet()
	for i := range g.NonTerms {
		if ntIsRecursive(g, i, llx.NewIntSet()) {
			ntis.Add(i)
		}
	}

	if ntis.IsEmpty() {
		return nil
	} else {
		return recursionError(nonTermNames(g.NonTerms, ntis))
	}
}

func ntIsRecursive (g *grammar.Grammar, index int, visited llx.IntSet) bool {
	visited.Add(index)
	st := g.States[g.NonTerms[index].FirstState]
	if len(st.Rules) > 0 {
		for _, r := range st.Rules {
			if r.NonTerm != grammar.SameNonTerm && (visited.Contains(r.NonTerm) || ntIsRecursive(g, r.NonTerm, visited.Copy())) {
				return true
			}
		}
	}
	if len(st.MultiRules) > 0 {
		for _, rs := range st.MultiRules {
			for _, r := range rs {
				if r.NonTerm != grammar.SameNonTerm && (visited.Contains(r.NonTerm) || ntIsRecursive(g, r.NonTerm, visited.Copy())) {
					return true
				}
			}
		}
	}
	return false
}

func assignTokenGroups (g *grammar.Grammar, e error) error {
	if e != nil {
		return e
	}

	var (
		rcnt, groups int
	)
	res := make([]*regexp.Regexp, 0, len(g.Tokens))
	ts := g.Tokens
	for rcnt = 0; rcnt < len(g.Tokens) && ts[rcnt].Re != ""; rcnt++ {
		res = append(res, regexp.MustCompile(ts[rcnt].Re))
		groups |= ts[rcnt].Groups
	}

	rts := ts[: rcnt]
	lts := ts[rcnt :]
	defaultGroups := 1 << bits.Len(uint(groups))
	for i, rt := range rts {
		if rt.Groups == 0 {
			rts[i].Groups = defaultGroups
		}
	}

	for i, lt := range lts {
		for j, rt := range rts {
			if res[j].FindString(lt.Name) == lt.Name {
				lts[i].Groups |= rt.Groups
			}
		}
		if lts[i].Groups == 0 {
			return unresolvedGroupsError(lt.Name)
		}
	}

	return nil
}

func assignStateGroups (g *grammar.Grammar, e error) error {
	if e != nil {
		return e
	}

	totalNts := len(g.NonTerms)
	for i, nt := range g.NonTerms {
		var lastState int
		if i >= totalNts - 1 {
			lastState = len(g.States)
		} else {
			lastState = g.NonTerms[i + 1].FirstState
		}

		for j := nt.FirstState; j < lastState; j++ {
			st := g.States[j]
			groups := -1
			for k := range st.Rules {
				if k >= 0 {
					groups &= g.Tokens[k].Groups
					if groups == 0 {
						return disjointGroupsError(g.NonTerms[i].Name, j, g.Tokens[k].Name)
					}
				}
			}

			for k := range st.MultiRules {
				if k >= 0 {
					groups &= g.Tokens[k].Groups
					if groups == 0 {
						return disjointGroupsError(g.NonTerms[i].Name, j, g.Tokens[k].Name)
					}
				}
			}

			g.States[j].Group = bits.Len(uint(groups)) - 1
		}
	}

	return nil
}

func nonTermNames (nts []grammar.NonTerm, ntis llx.IntSet) []string {
	indexes := ntis.ToSlice()
	names := make([]string, len(indexes))
	for i, index := range indexes {
		names[i] = nts[index].Name
	}
	return names
}