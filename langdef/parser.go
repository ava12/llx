package langdef

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/internal/ints"
	"github.com/ava12/llx/internal/queue"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/source"
)

const (
	unusedToken = grammar.AsideToken | grammar.ErrorToken
)

type nodeItem struct {
	Index       int
	DependsOn   *ints.Set
	FirstTokens *ints.Set
	Chunk       *groupChunk
}

type tokenIndex map[string]int
type nodeIndex map[string]*nodeItem

type chunk interface {
	FirstTokens() *ints.Set
	IsOptional() bool
	BuildStates(g *parseResult, stateIndex, nextIndex int)
}

type complexChunk interface {
	chunk
	Append(chunk)
}

// ParseString parses grammar description and returns grammar on success.
// Returns nil and llx.Error on error.
func ParseString(name, content string) (*grammar.Grammar, error) {
	return Parse(source.New(name, []byte(content)))
}

// ParseBytes parses grammar description and returns grammar on success.
// Returns nil and llx.Error on error.
func ParseBytes(name string, content []byte) (*grammar.Grammar, error) {
	return Parse(source.New(name, content))
}

// Parse parses grammar description and returns grammar on success.
// Returns nil and llx.Error on error.
func Parse(s *source.Source) (*grammar.Grammar, error) {
	result, e := parseLangDef(s)
	if e != nil {
		return nil, e
	}

	e = assignTokenGroups(result, e)
	e = findUndefinedNodes(result.NIndex, e)
	e = findUnusedNodes(result.Nodes, result.NIndex, e)
	e = resolveDependencies(result.Nodes, result.NIndex, e)
	e = buildStates(result, e)
	e = findRecursions(result, e)
	e = assignStateTokenTypes(result, e)

	return buildGrammar(result, e)
}

const (
	stringTok     = "string"
	nameTok       = "name"
	dirTok        = "dir"
	literalDirTok = "literal"
	mixedDirTok   = "mixed"
	groupDirTok   = "group-dir"
	tokenNameTok  = "token-name"
	regexpTok     = "regexp"
	opTok         = "op"
	wrongTok      = ""
)

const (
	stringTokType = 1
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
	name  string
	group int
	flags grammar.TokenFlags
}

type literalToken struct {
	name  string
	flags grammar.TokenFlags
}

type parseContext struct {
	q            *source.Queue
	l            *lexer.Lexer
	g            *parseResult
	lts          []literalToken
	ti, lti      tokenIndex
	ets          []extraToken
	eti          map[string]int
	currentGroup int
	restrictLtts bool
	restrictLs   bool
}

type escapeCharEntry struct {
	substitute, hexLen byte
}

var escapeCharMap = map[byte]escapeCharEntry{
	'\\': {'\\', 0},
	'"':  {'"', 0},
	'n':  {'\n', 0},
	'r':  {'\r', 0},
	't':  {'\t', 0},
	'x':  {0, 2},
	'u':  {0, 4},
	'U':  {0, 8},
}

func init() {
	tokenTypes = []lexer.TokenType{
		{1, stringTok},
		{2, nameTok},
		{3, dirTok},
		{4, literalDirTok},
		{5, mixedDirTok},
		{6, groupDirTok},
		{7, tokenNameTok},
		{8, regexpTok},
		{9, opTok},
		{lexer.ErrorTokenType, wrongTok},
	}
}

func parseLangDef(s *source.Source) (*parseResult, error) {
	var e error

	re := regexp.MustCompile(
		`\s+|#[^\n]*|` +
			`((?:"(?:[^\\"]|\\.)*")|(?:'.*?'))|` +
			`([a-zA-Z_][a-zA-Z_0-9-]*)|` +
			`(!(?:aside|caseless|error|extern)\b)|` +
			`(!reserved\b)|` +
			`(!literal\b)|` +
			`(!group\b)|` +
			`(\$[a-zA-Z_][a-zA-Z_0-9-]*)|` +
			`(\/(?:[^\\\/]|\\.)+\/)|` +
			`([(){}\[\]=|,;@])|` +
			`(['"/!].{0,10})`)

	q := source.NewQueue().Append(s)
	l := lexer.New(re, tokenTypes)
	ets := make([]extraToken, 0)
	eti := make(map[string]int)
	ti := tokenIndex{}
	lti := tokenIndex{}
	g := newParseResult()
	c := &parseContext{q, l, g, make([]literalToken, 0), ti, lti, ets, eti, 0, false, false}

	var t *lexer.Token
	for e == nil {
		t, e = fetch(q, l, []string{nameTok, dirTok, literalDirTok, mixedDirTok, groupDirTok, opTok, tokenNameTok}, true, nil)
		if e != nil {
			return nil, e
		}

		if t == nil || t.TypeName() == nameTok {
			break
		}

		switch t.TypeName() {
		case dirTok:
			e = parseDir(t.Text(), c)

		case groupDirTok:
			e = parseGroupDir(c)

		case literalDirTok:
			e = parseLiteralDir(t.Text(), c)

		case mixedDirTok:
			e = parseMixedDir(t.Text(), c)

		case opTok:
			e = parseLayerDef(t, c)

		case tokenNameTok:
			name := t.Text()[1:]
			i, has := ti[name]
			if has && g.Tokens[i].Re != "" {
				return nil, defTokenError(t)
			}

			e = parseTokenDef(name, c)
		}
	}
	if e != nil {
		return nil, e
	}

	if len(c.g.Tokens)+len(c.ets) >= grammar.MaxTokenType {
		return nil, tokenTypeNumberError(t)
	}

	for _, et := range c.ets {
		_, has := c.eti[et.name]
		if has {
			if et.flags&grammar.ExternalToken != 0 {
				addToken(et.name, "", et.flags, c)
			} else {
				return nil, undefinedTokenError(et.name)
			}
		}
	}

	if c.restrictLtts {
		for i, t := range g.Tokens {
			if (t.Flags & grammar.LiteralToken) != 0 {
				break
			}

			g.Tokens[i].Flags ^= grammar.NoLiteralsToken
		}
	}

	c.lti = make(tokenIndex)
	for _, lt := range c.lts {
		useLiteralToken(lt.name, lt.flags, c)
	}

	nti := g.NIndex
	for e == nil && t != nil && !isEof(t) {
		if t.TypeName() == opTok {
			e = parseLayerDef(t, c)
		} else {
			_, has := nti[t.Text()]
			if has && nti[t.Text()].Chunk != nil {
				return nil, defNodeError(t)
			}

			e = parseNodeDef(t.Text(), c)
		}
		if e == nil {
			t, e = fetch(q, l, []string{nameTok, opTok, lexer.EofTokenName, lexer.EoiTokenName}, true, nil)
		}
	}

	return g, e
}

var savedToken *lexer.Token

func put(t *lexer.Token) {
	if savedToken != nil {
		panic("cannot put " + t.TypeName() + " token: already put " + savedToken.TypeName())
	}

	savedToken = t
}

func isEof(t *lexer.Token) bool {
	tt := t.Type()
	return (tt == lexer.EofTokenType || tt == lexer.EoiTokenType)
}

func fetch(q *source.Queue, l *lexer.Lexer, types []string, strict bool, e error) (*lexer.Token, error) {
	if e != nil {
		return nil, e
	}

	token := savedToken
	if token == nil {
		token, e = l.Next(q)
		if e != nil {
			return nil, e
		}

		if token.TypeName() == stringTok {
			token, e = processStringToken(token)
			if e != nil {
				return nil, e
			}
		}
	} else {
		savedToken = nil
	}

	for _, typ := range types {
		if token.TypeName() == typ || token.Text() == typ {
			return token, nil
		}
	}

	if isEof(token) {
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

func processStringToken(token *lexer.Token) (*lexer.Token, error) {
	content := token.Content()
	if content[0] != '"' || bytes.IndexByte(content, '\\') < 0 {
		return token, nil
	}

	var peekRune = func(content []byte, hexLen int) (rune, error) {
		if len(content) < hexLen+3 {
			return 0, invalidEscapeError(token, string(content))
		}

		codepoint, e := strconv.ParseUint(string(content[2:hexLen+2]), 16, 32)
		if e != nil {
			return 0, invalidEscapeError(token, string(content))
		}

		if utf8.ValidRune(rune(codepoint)) {
			return rune(codepoint), nil
		} else {
			return 0, invalidRuneError(token, string(content[2:hexLen+2]))
		}
	}

	result := make([]byte, 0, len(content))
	for {
		slashPos := bytes.IndexByte(content, '\\')
		if slashPos < 0 {
			result = append(result, content...)
			break
		}

		if slashPos > 0 {
			result = append(result, content[:slashPos]...)
			content = content[slashPos:]
		}

		letter := content[1]
		entry, valid := escapeCharMap[letter]
		if !valid {
			return nil, invalidEscapeError(token, string(content[:2]))
		}

		if entry.hexLen == 0 {
			result = append(result, entry.substitute)
			content = content[2:]
		} else {
			r, e := peekRune(content, int(entry.hexLen))
			if e != nil {
				return nil, e
			}

			result = utf8.AppendRune(result, r)
			content = content[entry.hexLen+2:]
		}
	}

	return lexer.NewToken(stringTokType, stringTok, result, token.Pos()), nil
}

func fetchOne(q *source.Queue, l *lexer.Lexer, typ string, strict bool, e error) (*lexer.Token, error) {
	return fetch(q, l, []string{typ}, strict, e)
}

func fetchAll(q *source.Queue, l *lexer.Lexer, types []string, e error) ([]*lexer.Token, error) {
	if e != nil {
		return nil, e
	}

	result := make([]*lexer.Token, 0)
	for {
		t, e := fetch(q, l, types, false, nil)
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

func skip(q *source.Queue, l *lexer.Lexer, types []string, e error) error {
	if e != nil {
		return e
	}

	_, e = fetch(q, l, types, true, nil)
	return e
}

func skipOne(q *source.Queue, l *lexer.Lexer, typ string, e error) error {
	return skip(q, l, []string{typ}, e)
}

func addToken(name, re string, flags grammar.TokenFlags, c *parseContext) int {
	var t extraToken
	i, has := c.eti[name]
	if has {
		t = c.ets[i]
		delete(c.eti, name)
	}
	c.g.Tokens = append(c.g.Tokens, grammar.Token{name, re, t.group, flags | t.flags})
	index := len(c.g.Tokens) - 1
	c.ti[name] = index
	return index
}

func addLiteralToken(name string, flags grammar.TokenFlags, c *parseContext) {
	_, has := c.lti[name]
	if !has {
		c.lti[name] = len(c.lts)
		c.lts = append(c.lts, literalToken{name, flags})
	}
}

func useLiteralToken(name string, flags grammar.TokenFlags, c *parseContext) int {
	i, has := c.lti[name]
	if has {
		return i
	}

	i = len(c.g.Tokens)
	c.g.Tokens = append(c.g.Tokens, grammar.Token{name, "", 0, flags | grammar.LiteralToken})
	c.lti[name] = i
	return i
}

func addExtraToken(name string, c *parseContext) int {
	i, has := c.eti[name]
	if !has {
		i = len(c.ets)
		c.ets = append(c.ets, extraToken{name: name})
		c.eti[name] = i
	}
	return i
}

func addTokenFlag(name string, flag grammar.TokenFlags, c *parseContext) {
	i, has := c.ti[name]
	if has {
		c.g.Tokens[i].Flags |= flag
	} else {
		i = addExtraToken(name, c)
		c.ets[i].flags |= flag
	}
}

func parseDir(name string, c *parseContext) error {
	tokens, e := fetchAll(c.q, c.l, []string{tokenNameTok}, nil)
	e = skipOne(c.q, c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	var flag grammar.TokenFlags = 0
	switch name {
	case "!aside":
		flag = grammar.AsideToken
	case "!caseless":
		flag = grammar.CaselessToken
	case "!extern":
		flag = grammar.ExternalToken
	case "!error":
		flag = grammar.ErrorToken
	}
	for _, token := range tokens {
		addTokenFlag(token.Text()[1:], flag, c)
	}

	return nil
}

func parseGroupDir(c *parseContext) error {
	tokens, e := fetchAll(c.q, c.l, []string{tokenNameTok}, nil)
	e = skipOne(c.q, c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	c.currentGroup++
	for _, token := range tokens {
		name := token.Text()[1:]
		i, has := c.ti[name]
		if has {
			if c.g.Tokens[i].Group != 0 {
				return reassignedGroupError(name)
			}

			c.g.Tokens[i].Group = c.currentGroup
		} else {
			i = addExtraToken(name, c)
			c.ets[i].group = c.currentGroup
		}
	}
	return nil
}

func parseLiteralDir(dir string, c *parseContext) error {
	flags := grammar.LiteralToken
	if dir == "!reserved" {
		flags |= grammar.ReservedToken
	}
	tokens, e := fetchAll(c.q, c.l, []string{stringTok}, nil)
	e = skipOne(c.q, c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	for _, t := range tokens {
		text := t.Text()
		addLiteralToken(text[1:len(text)-1], flags, c)
	}
	return nil
}

func parseMixedDir(_ string, c *parseContext) error {
	tokens, e := fetchAll(c.q, c.l, []string{stringTok, tokenNameTok}, nil)
	e = skipOne(c.q, c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	for _, t := range tokens {
		text := t.Text()
		if t.TypeName() == tokenNameTok {
			c.restrictLtts = true
			addTokenFlag(text[1:], grammar.NoLiteralsToken, c)
		} else {
			c.restrictLs = true
			addLiteralToken(text[1:len(text)-1], 0, c)
		}
	}
	return nil
}

func parseLayerDef(t *lexer.Token, c *parseContext) error {
	if t.Text() != "@" {
		return unexpectedTokenError(t)
	}

	token, e := fetchOne(c.q, c.l, nameTok, true, nil)
	if e != nil {
		return e
	}

	layer := grammar.Layer{Type: token.Text()}

	for {
		token, e = fetch(c.q, c.l, []string{nameTok, semicolonTok}, true, nil)
		if e != nil {
			return e
		}

		if token.TypeName() == opTok {
			break
		}

		command := grammar.LayerCommand{Command: token.Text()}
		e = skipOne(c.q, c.l, lBraceTok, nil)
		if e != nil {
			return e
		}

		token, _ = fetchOne(c.q, c.l, rBraceTok, false, e)
		if token != nil {
			layer.Commands = append(layer.Commands, command)
			continue
		}

		for {
			token, e = fetch(c.q, c.l, []string{stringTok, nameTok}, true, nil)
			if e != nil {
				return e
			}

			arg := token.Text()
			if token.TypeName() == stringTok {
				arg = arg[1 : len(arg)-1]
			}
			command.Arguments = append(command.Arguments, arg)

			token, e = fetch(c.q, c.l, []string{commaTok, rBraceTok}, true, nil)
			if e != nil {
				return e
			}

			if token.Text() == rBraceTok {
				break
			}
		}

		layer.Commands = append(layer.Commands, command)
	}

	c.g.Layers = append(c.g.Layers, layer)
	return nil
}

func parseTokenDef(name string, c *parseContext) error {
	e := skipOne(c.q, c.l, equTok, nil)
	token, e := fetchOne(c.q, c.l, regexpTok, true, e)
	e = skipOne(c.q, c.l, semicolonTok, e)
	if e != nil {
		return e
	}

	re := token.Text()[1 : len(token.Text())-1]
	_, e = regexp.Compile(re)
	if e != nil {
		return regexpError(token, e)
	}

	addToken(name, re, 0, c)

	return nil
}

func addNode(name string, c *parseContext, define bool) *nodeItem {
	var group *groupChunk = nil
	if define {
		group = newGroupChunk(false, false)
	}
	result := c.g.NIndex[name]
	if result != nil {
		if result.Chunk == nil && define {
			result.Chunk = group
		}
		return result
	}

	result = &nodeItem{len(c.g.Nodes), ints.NewSet(), ints.NewSet(), group}
	c.g.NIndex[name] = result
	c.g.Nodes = append(c.g.Nodes, grammar.Node{name, 0})
	return result
}

func parseNodeDef(name string, c *parseContext) error {
	nt := addNode(name, c, true)
	e := skipOne(c.q, c.l, equTok, nil)
	e = parseGroup(name, nt.Chunk, c, e)
	e = skipOne(c.q, c.l, semicolonTok, e)
	return e
}

func parseGroup(name string, group complexChunk, c *parseContext, e error) error {
	if e != nil {
		return e
	}

	for {
		item, e := parseVariants(name, c)
		if e != nil {
			return e
		}

		group.Append(item)
		t, e := fetchOne(c.q, c.l, commaTok, false, nil)
		if t == nil {
			return e
		}
	}
}

func parseVariants(name string, c *parseContext) (chunk, error) {
	ch, e := parseVariant(name, c)
	t, e := fetchOne(c.q, c.l, pipeTok, false, e)
	if e != nil {
		return nil, e
	} else if t == nil {
		return ch, nil
	}

	result := newVariantChunk()
	result.Append(ch)

	for {
		ch, e = parseVariant(name, c)
		t, e = fetchOne(c.q, c.l, pipeTok, false, e)
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

func parseVariant(name string, c *parseContext) (chunk, error) {
	variantHeads := []string{nameTok, tokenNameTok, stringTok, lBraceTok, lSquareTok, lCurlyTok}
	t, e := fetch(c.q, c.l, variantHeads, true, nil)
	if e != nil {
		return nil, e
	}

	var (
		index int
		f     bool
	)
	switch t.TypeName() {
	case nameTok:
		nt := addNode(t.Text(), c, false)
		c.g.NIndex[name].DependsOn.Add(nt.Index)
		return newNodeChunk(t.Text(), nt), nil

	case tokenNameTok:
		index, f = c.ti[t.Text()[1:]]
		if !f {
			return nil, tokenError(t)
		}

		if (c.g.Tokens[index].Flags & unusedToken) != 0 {
			return nil, wrongTokenError(t)
		}

		return newTokenChunk(index), nil

	case stringTok:
		name = t.Text()[1 : len(t.Text())-1]
		index, f = c.lti[name]
		if !f {
			if c.restrictLs {
				return nil, unknownLiteralError(name)
			}

			index = useLiteralToken(name, 0, c)
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
	e = skipOne(c.q, c.l, lastToken, e)
	if e != nil {
		return nil, e
	}

	return result, nil
}

func findUndefinedNodes(nti nodeIndex, e error) error {
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
		return unknownNodeError(uns)
	}

	return nil
}

func findUnusedNodes(nts []grammar.Node, nti nodeIndex, e error) error {
	if e != nil {
		return e
	}

	unreachedNts := ints.NewSet()
	for i := 0; i < len(nts); i++ {
		unreachedNts.Add(i)
	}
	searchQueue := queue.New[int](0)
	for {
		index, fetched := searchQueue.First()
		if !fetched {
			break
		}

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
		return unusedNodeError(nodeNames(nts, unreachedNts))
	}
}

func resolveDependencies(nts []grammar.Node, nti nodeIndex, e error) error {
	if e != nil {
		return e
	}

	affects := make(map[int][]int)
	resolveQueue := queue.New[int]()

	for _, item := range nti {
		if item.DependsOn.IsEmpty() {
			resolveQueue.Append(item.Index)
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

	for {
		k, fetched := resolveQueue.First()
		if !fetched {
			break
		}

		for _, index := range affects[k] {
			item := nti[nts[index].Name]
			item.DependsOn.Remove(k)
			if item.DependsOn.IsEmpty() {
				resolveQueue.Append(index)
				item.FirstTokens = item.Chunk.FirstTokens()
			}
		}
	}

	for _, item := range nti {
		if !item.DependsOn.IsEmpty() {
			resolveQueue.Append(item.Index)
		}
	}

	if resolveQueue.IsEmpty() {
		return nil
	}

	indexes := resolveQueue.Items()
	tokenAdded := true
	for tokenAdded {
		tokenAdded = false

		for _, index := range indexes {
			item := nti[nts[index].Name]
			firstTokens := item.Chunk.FirstTokens()
			if !firstTokens.IsEmpty() && !ints.Subtract(firstTokens, item.FirstTokens).IsEmpty() {
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

func buildStates(g *parseResult, e error) error {
	if e != nil {
		return e
	}

	nti := g.NIndex
	for i, nt := range g.Nodes {
		firstState, _ := g.AddState()
		item := nti[nt.Name]
		g.Nodes[i].FirstState = firstState
		item.Chunk.BuildStates(g, firstState, grammar.FinalState)
	}

	return nil
}

func findRecursions(g *parseResult, e error) error {
	if e != nil {
		return e
	}

	ntis := ints.NewSet()
	for i := range g.Nodes {
		if ntIsRecursive(g, i, ints.NewSet()) {
			ntis.Add(i)
		}
	}

	if ntis.IsEmpty() {
		return nil
	} else {
		return recursionError(nodeNames(g.Nodes, ntis))
	}
}

func ntIsRecursive(g *parseResult, index int, visited *ints.Set) bool {
	visited.Add(index)
	st := g.States[g.Nodes[index].FirstState]
	for _, rs := range st.Rules {
		for _, r := range rs {
			if r.Node != grammar.SameNode && (visited.Contains(r.Node) || ntIsRecursive(g, r.Node, visited.Copy())) {
				return true
			}
		}
	}
	return false
}

func assignTokenGroups(g *parseResult, e error) error {
	if e != nil {
		return e
	}

	var (
		rcnt int
		t    grammar.Token
	)
	res := make(map[int]*regexp.Regexp)
	ts := g.Tokens
	g.TTypes = make([]grammar.BitSet, len(ts))

	for rcnt, t = range ts {
		if t.Re == "" {
			break
		}

		if (t.Flags & grammar.NoLiteralsToken) == 0 {
			res[rcnt] = regexp.MustCompile(`(?s:` + t.Re + `)`)
		}
	}
	rts := ts[:rcnt]

	for rcnt < len(ts) && ts[rcnt].Flags&grammar.LiteralToken == 0 {
		rcnt++
	}
	lts := ts[rcnt:]
	for i := 0; i < rcnt; i++ {
		g.TTypes[i] = 1 << i
	}

	for i, lt := range lts {
		caseless := (lt.Name == strings.ToUpper(lt.Name))
		for j, re := range res {
			rt := rts[j]
			if (rt.Flags&grammar.CaselessToken == 0 || caseless) && re.FindString(lt.Name) == lt.Name {
				g.TTypes[rcnt+i] |= 1 << j
			}
		}
		if g.TTypes[rcnt+i] == 0 {
			return unresolvedTokenTypesError(lt.Name)
		}
	}

	return nil
}

func assignStateTokenTypes(g *parseResult, e error) error {
	if e != nil {
		return e
	}

	var defaultTypes grammar.BitSet
	for i, t := range g.Tokens {
		if t.Flags&grammar.AsideToken != 0 {
			defaultTypes |= 1 << i
		}
	}

	totalNts := len(g.Nodes)
	for i, nt := range g.Nodes {
		var lastState int
		if i >= totalNts-1 {
			lastState = len(g.States)
		} else {
			lastState = g.Nodes[i+1].FirstState
		}

		for j := nt.FirstState; j < lastState; j++ {
			st := g.States[j]
			types := defaultTypes
			for k := range st.Rules {
				if k >= 0 {
					types |= g.TTypes[k]
				}
			}

			g.States[j].Types = types
		}
	}

	return nil
}

func nodeNames(nts []grammar.Node, ntis *ints.Set) []string {
	indexes := ntis.ToSlice()
	names := make([]string, len(indexes))
	for i, index := range indexes {
		names[i] = nts[index].Name
	}
	return names
}

func buildGrammar(pr *parseResult, e error) (*grammar.Grammar, error) {
	if e != nil {
		return nil, e
	}

	return pr.BuildGrammar(), nil
}
