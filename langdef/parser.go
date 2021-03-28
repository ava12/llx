package langdef

import (
	"math/bits"
	"regexp"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/source"
	"github.com/ava12/llx/util/intqueue"
	"github.com/ava12/llx/util/intset"
)

const (
	unusedTerm = grammar.AsideTerm | grammar.ErrorTerm
	noGroup = -1
	maxGroup = 30
)


type chunk interface {
	FirstTerms () intset.T
	IsOptional () bool
	BuildStates (nonterm *grammar.Nonterm, stateIndex, nextIndex int)
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

type nontermItem struct {
	Index int
	DependsOn, FirstTerms intset.T
	Chunk *groupChunk
}

type termIndex map[string]int
type nontermIndex map[string]*nontermItem


func Parse (s *source.Source) (*grammar.Grammar, error) {
	result := &grammar.Grammar{
		Terms: make([]grammar.Term, 0),
		Nonterms: make([]grammar.Nonterm, 0),
	}

	nti := nontermIndex{}
	var e error

	e = parseLangDef(s, result, nti)
	e = assignTermGroups(result, e)
	e = findUndefinedNonterminals(nti, e)
	e = findUnusedNonterminals(result.Nonterms, nti, e)
	e = resolveDependencies(result.Nonterms, nti, e)
	e = buildStates(result.Nonterms, nti, e)
	e = findRecursions(result.Nonterms, e)
	e = assignStateGroups(result, e)

	if e != nil {
		return nil, e
	}

	return result, nil
}

const (
	stringTok = "string"
	nameTok = "name"
	dirTok = "dir"
	literalDirTok = "literal"
	groupDirTok = "group-dir"
	termNameTok = "term-name"
	regexpTok = "regexp"
	opTok = "op"
	wrongTok = ""

	equTok = "="
	commaTok = ","
	semicolonTok = ";"
	pipeTok = "|"
	lBraceTok = "("
	rBraceTok = ")"
	lSquareTok = "["
	rSquareTok = "]"
	lCurlyTok = "{"
	rCurlyTok = "}"
)

var (
	tokenTypes []lexer.TokenType
)

type extraTerm struct {
	name string
	groups int
	flags grammar.TermFlags
}

type parseContext struct {
	l *lexer.Lexer
	g *grammar.Grammar
	lts []string
	ti, lti termIndex
	nti nontermIndex
	ets []extraTerm
	eti map[string]int
	currentGroup int
}

func init () {
	tokenTypes = []lexer.TokenType {
		{1, stringTok},
		{2, nameTok},
		{3, dirTok},
		{4, literalDirTok},
		{5, groupDirTok},
		{6, termNameTok},
		{7, regexpTok},
		{8, opTok},
		{lexer.ErrorTokenType, wrongTok},
	}
}

func parseLangDef (s *source.Source, g *grammar.Grammar, nti nontermIndex) error {
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
	ets := make([]extraTerm, 0)
	eti := make(map[string]int)
	ti := termIndex{}
	lti := termIndex{}
	c := &parseContext{l, g, make([]string, 0), ti, lti, nti, ets, eti, 0}

	var t *lexer.Token
	for e == nil {
		t, e = fetch(l, []string{nameTok, dirTok, groupDirTok, literalDirTok, termNameTok}, true, nil)
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

		case termNameTok:
			name := t.Text()[1:]
			i, has := ti[name]
			if has && g.Terms[i].Re != "" {
				return defTermError(t)
			}
			e = parseTermDef(name, c)
		}
	}
	if e != nil {
		return e
	}

	for _, et := range c.ets {
		_, has := c.eti[et.name]
		if has {
			if et.flags & grammar.ExternalTerm != 0 {
				addTerm(et.name, "", et.groups, et.flags, c)
			} else {
				return undefinedTermError(et.name)
			}
		}
	}

	firstLiteral := len(c.g.Terms)
	for i, name := range c.lts {
		addLiteralTerm(name, c)
		c.lti[name] = i + firstLiteral
	}

	for e == nil && t != nil && t.Type() != lexer.EofTokenType {
		_, has := nti[t.Text()]
		if has && nti[t.Text()].Chunk != nil {
			return defNontermError(t)
		}

		e = parseNontermDef(t.Text(), c)
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
		return nil, tokenError(token)
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

func addTerm (name, re string, groups int, flags grammar.TermFlags, c *parseContext) int {
	var t extraTerm
	i, has := c.eti[name]
	if has {
		t = c.ets[i]
		delete(c.eti, name)
	}
	c.g.Terms = append(c.g.Terms, grammar.Term{name, re, groups | t.groups, flags | t.flags})
	index := len(c.g.Terms) - 1
	c.ti[name] = index
	return index
}

func addLiteralTerm (name string, c *parseContext) int {
	i, has := c.lti[name]
	if has {
		return i
	}

	i = len(c.g.Terms)
	c.g.Terms = append(c.g.Terms, grammar.Term{name, "", 0, grammar.LiteralTerm})
	c.lti[name] = i
	return i
}

func addExtraTerm (name string, c *parseContext) int {
	i, has := c.eti[name]
	if !has {
		i = len(c.ets)
		c.ets = append(c.ets, extraTerm{name : name})
		c.eti[name] = i
	}
	return i
}

func addTermFlag (name string, flag grammar.TermFlags, c *parseContext) {
	i, has := c.ti[name]
	if has {
		c.g.Terms[i].Flags |= flag
	} else {
		i = addExtraTerm(name, c)
		c.ets[i].flags |= flag
	}
}

func addTermGroups(token *lexer.Token, groups int, c *parseContext) {
	name := token.Text()[1 :]
	i, has := c.ti[name]
	if has {
		c.g.Terms[i].Groups |= groups
	} else {
		i = addExtraTerm(name, c)
		c.ets[i].groups |= groups
	}
}

func parseDir (name string, c *parseContext) error {
	tokens, e := fetchAll(c.l, []string{termNameTok}, nil)
	e = skipOne(c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	var flag grammar.TermFlags = 0
	switch name {
	case "!aside":
		flag = grammar.AsideTerm
	case "!extern":
		flag = grammar.ExternalTerm
	case "!error":
		flag = grammar.ErrorTerm
	case "!shrink":
		flag = grammar.ShrinkableTerm
	}
	for _, token := range tokens {
		addTermFlag(token.Text()[1 :], flag, c)
	}

	return nil
}

func parseGroupDir (c *parseContext) error {
	tokens, e := fetchAll(c.l, []string{termNameTok}, nil)
	e = skipOne(c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	if len(tokens) == 0 {
		return nil
	}

	groups := 1 << c.currentGroup
	for _, token := range tokens {
		addTermGroups(token, groups, c)
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
		addLiteralTerm(text[1 : len(text) - 1], c)
	}
	return nil
}

func parseTermDef (name string, c *parseContext) error {
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

	addTerm(name, re, 0, 0, c)

	return nil
}

func addNonterm (name string, c *parseContext, define bool) *nontermItem {
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

	result = &nontermItem{len(c.g.Nonterms), intset.New(), intset.New(), group}
	c.nti[name] = result
	c.g.Nonterms = append(c.g.Nonterms, grammar.Nonterm{name, nil})
	return result
}

func parseNontermDef (name string, c *parseContext) error {
	nt := addNonterm(name, c, true)
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
	variantHeads := []string{nameTok, termNameTok, stringTok, lBraceTok, lSquareTok, lCurlyTok}
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
		nt := addNonterm(t.Text(), c, false)
		c.nti[name].DependsOn.Add(nt.Index)
		return newNontermChunk(t.Text(), nt), nil

	case termNameTok:
		index, f = c.ti[t.Text()[1 :]]
		if !f {
			return nil, termError(t)
		}

		if (c.g.Terms[index].Flags & unusedTerm) != 0 {
			return nil, wrongTermError(t)
		}

		return newTermChunk(index), nil

	case stringTok:
		name = t.Text()[1 : len(t.Text()) - 1]
		index, f = c.lti[name]
		if !f {
			index = addLiteralTerm(name, c)
		}
		return newTermChunk(index), nil
	}

	repeated := (t.Text() == "{")
	optional := (t.Text() != "(")
	var lastTerm string
	if repeated {
		lastTerm = rCurlyTok
	} else if optional {
		lastTerm = rSquareTok
	} else {
		lastTerm = rBraceTok
	}

	result := newGroupChunk(optional, repeated)
	e = parseGroup(name, result, c, nil)
	e = skipOne(c.l, lastTerm, e)
	if e != nil {
		return nil, e
	}

	return result, nil
}

func findUndefinedNonterminals (nti nontermIndex, e error) error {
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
		return unknownNontermError(uns)
	}

	return nil
}

func findUnusedNonterminals (nts []grammar.Nonterm, nti nontermIndex, e error) error {
	if e != nil {
		return e
	}

	unreachedNts := intset.New()
	for i := 0; i < len(nts); i++ {
		unreachedNts.Add(i)
	}
	searchQueue := intqueue.New(0)
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
		return unusedNontermError(nontermNames(nts, unreachedNts))
	}
}

func resolveDependencies (nts []grammar.Nonterm, nti nontermIndex, e error) error {
	if e != nil {
		return e
	}

	affects := make(map[int][]int)
	queue := intqueue.New()

	for _, item := range nti {
		if item.DependsOn.IsEmpty() {
			queue.Append(item.Index)
			item.FirstTerms = item.Chunk.FirstTerms()
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
				item.FirstTerms = item.Chunk.FirstTerms()
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
	termAdded := true
	for termAdded {
		termAdded = false

		for _, index := range indexes {
			item := nti[nts[index].Name]
			firstTerms := item.Chunk.FirstTerms()
			if !firstTerms.IsEmpty() && !intset.Subtract(firstTerms, item.FirstTerms).IsEmpty() {
				item.FirstTerms.Union(firstTerms)
				termAdded = true
			}
		}
	}

	names := make([]string, 0, len(indexes))
	for _, index := range indexes {
		name := nts[index].Name
		if nti[name].FirstTerms.IsEmpty() {
			names = append(names, name)
		}
	}
	if len(names) > 0 {
		return unresolvedError(names)
	}

	return nil
}

func buildStates (nts []grammar.Nonterm, nti nontermIndex, e error) error {
	if e != nil {
		return e
	}

	for _, item := range nti {
		nts[item.Index].States = append(nts[item.Index].States, grammar.State {
			noGroup,
			map[int]grammar.Rule{},
			map[int][]grammar.Rule{},
		})
		item.Chunk.BuildStates(&nts[item.Index], 0, grammar.FinalState)
		states := nts[item.Index].States
		for i, state := range states {
			if len(state.Rules) == 0 {
				states[i].Rules = nil
			}
			if len(state.MultiRules) == 0 {
				states[i].MultiRules = nil
			}
		}
	}

	return nil
}

func findRecursions (nts []grammar.Nonterm, e error) error {
	if e != nil {
		return e
	}

	ntis := intset.New()
	for i := range nts {
		if ntIsRecursive(nts, i, intset.New()) {
			ntis.Add(i)
		}
	}

	if ntis.IsEmpty() {
		return nil
	} else {
		return recursionError(nontermNames(nts, ntis))
	}
}

func ntIsRecursive (nts []grammar.Nonterm, index int, visited intset.T) bool {
	visited.Add(index)
	st := nts[index].States[0]
	if len(st.Rules) > 0 {
		for _, r := range st.Rules {
			if r.Nonterm != grammar.SameNonterm && (visited.Contains(r.Nonterm) || ntIsRecursive(nts, r.Nonterm, visited.Copy())) {
				return true
			}
		}
	}
	if len(st.MultiRules) > 0 {
		for _, rs := range st.MultiRules {
			for _, r := range rs {
				if r.Nonterm != grammar.SameNonterm && (visited.Contains(r.Nonterm) || ntIsRecursive(nts, r.Nonterm, visited.Copy())) {
					return true
				}
			}
		}
	}
	return false
}

func assignTermGroups(g *grammar.Grammar, e error) error {
	if e != nil {
		return e
	}

	var (
		rcnt, groups int
	)
	res := make([]*regexp.Regexp, 0, len(g.Terms))
	ts := g.Terms
	for rcnt = 0; rcnt < len(g.Terms) && ts[rcnt].Re != ""; rcnt++ {
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

	for i, nt := range g.Nonterms {
		for j, st := range nt.States {
			groups := -1
			for k := range st.Rules {
				if k >= 0 {
					groups &= g.Terms[k].Groups
					if groups == 0 {
						return disjointGroupsError(g.Nonterms[i].Name, j, g.Terms[k].Name)
					}
				}
			}

			for k := range st.MultiRules {
				if k >= 0 {
					groups &= g.Terms[k].Groups
					if groups == 0 {
						return disjointGroupsError(g.Nonterms[i].Name, j, g.Terms[k].Name)
					}
				}
			}

			nt.States[j].Group = bits.Len(uint(groups)) - 1
		}
	}

	return nil
}

func nontermNames (nts []grammar.Nonterm, ntis intset.T) []string {
	indexes := ntis.ToSlice()
	names := make([]string, len(indexes))
	for i, index := range indexes {
		names[i] = nts[index].Name
	}
	return names
}