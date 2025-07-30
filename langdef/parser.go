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

// ParseString parses grammar description and returns a grammar on success.
// Returns nil and llx.Error on error.
func ParseString(name, content string) (*grammar.Grammar, error) {
	return Parse(source.New(name, []byte(content)))
}

// ParseBytes parses grammar description and returns a grammar on success.
// Returns nil and llx.Error on error.
func ParseBytes(name string, content []byte) (*grammar.Grammar, error) {
	return Parse(source.New(name, content))
}

// Parse parses grammar description and returns a grammar on success.
// Returns nil and llx.Error on error.
func Parse(s *source.Source) (*grammar.Grammar, error) {
	c := newParseContext()
	result, e := c.Parse(s)
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
	stringTok       = "string"
	nameTok         = "name"
	dirTok          = "dir"
	templateNameTok = "template-name"
	tokenNameTok    = "token-name"
	regexpTok       = "regexp"
	opTok           = "op"
	wrongTok        = ""
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

const (
	asideDir    = "!aside"
	caselessDir = "!caseless"
	errorDir    = "!error"
	externDir   = "!extern"
	groupDir    = "!group"
	literalDir  = "!literal"
	reservedDir = "!reserved"
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
	q                    *source.Queue
	result               *parseResult
	literals             []literalToken
	tokenIndex           tokenIndex
	literalIndex         tokenIndex
	extraTokens          []extraToken
	extraIndex           map[string]int
	savedToken           *lexer.Token
	templates            map[string]string
	currentGroup         int
	restrictLiteralTypes bool
	restrictLiterals     bool
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

var llxLexer *lexer.Lexer

func init() {
	tokenTypes := []lexer.TokenType{
		{1, stringTok},
		{2, nameTok},
		{3, dirTok},
		{4, templateNameTok},
		{5, tokenNameTok},
		{6, regexpTok},
		{7, opTok},
		{lexer.ErrorTokenType, wrongTok},
	}

	re := regexp.MustCompile(
		`^(?:\s+|#[^\n]*|` +
			`((?:"(?:[^\\"]|\\.)*")|(?:'.*?'))|` +
			`([a-zA-Z_][a-zA-Z_0-9-]*)|` +
			`(![a-z]+)|` +
			`(\$\$[a-zA-Z_][a-zA-Z_0-9-]*)|` +
			`(\$(?:[a-zA-Z_][a-zA-Z_0-9-]*)?)|` +
			`(\/(?:[^\\\/]|\\.)+\/)|` +
			`([(){}\[\]=|,;@])|` +
			`(['"/!].{0,10}))`)

	llxLexer = lexer.New(re, tokenTypes)
}

func newParseContext() *parseContext {
	return &parseContext{
		q:            source.NewQueue(),
		result:       newParseResult(),
		literals:     make([]literalToken, 0),
		tokenIndex:   make(tokenIndex),
		literalIndex: make(tokenIndex),
		extraTokens:  make([]extraToken, 0),
		extraIndex:   make(map[string]int),
		templates:    make(map[string]string),
	}
}

func (c *parseContext) Parse(s *source.Source) (*parseResult, error) {
	c.q.Append(s)

	var e error
	var t *lexer.Token

	for e == nil {
		t, e = c.fetch([]string{
			nameTok, dirTok, opTok, templateNameTok, tokenNameTok,
		}, true, nil)
		if e != nil {
			return nil, e
		}

		if t == nil || t.TypeName() == nameTok {
			break
		}

		switch t.TypeName() {
		case dirTok:
			e = c.parseDir(t)

		case opTok:
			e = c.parseLayerDef(t)

		case templateNameTok:
			name := t.Text()[2:]
			_, has := c.templates[name]
			if has {
				return nil, templateDefinedError(t, name)
			}

			e = c.parseTemplateDef(name)

		case tokenNameTok:
			name := t.Text()[1:]
			i, has := c.tokenIndex[name]
			if (has && c.result.Tokens[i].Re != "") || name == "" {
				return nil, defTokenError(t)
			}

			e = c.parseTokenDef(name)
		}
	}
	if e != nil {
		return nil, e
	}

	if len(c.result.Tokens)+len(c.extraTokens) >= grammar.MaxTokenType {
		return nil, tokenTypeNumberError(t)
	}

	for _, et := range c.extraTokens {
		_, has := c.extraIndex[et.name]
		if has {
			if et.flags&grammar.ExternalToken != 0 {
				c.addToken(et.name, "", et.flags)
			} else {
				return nil, undefinedTokenError(et.name)
			}
		}
	}

	if c.restrictLiteralTypes {
		for i, t := range c.result.Tokens {
			if (t.Flags & grammar.LiteralToken) != 0 {
				break
			}

			c.result.Tokens[i].Flags ^= grammar.NoLiteralsToken
		}
	}

	c.literalIndex = make(tokenIndex)
	for _, lt := range c.literals {
		c.useLiteralToken(lt.name, lt.flags)
	}

	nti := c.result.NIndex
	for e == nil && t != nil && !isEof(t) {
		if t.TypeName() == opTok {
			e = c.parseLayerDef(t)
		} else {
			_, has := nti[t.Text()]
			if has && nti[t.Text()].Chunk != nil {
				return nil, defNodeError(t)
			}

			e = c.parseNodeDef(t.Text())
		}
		if e == nil {
			t, e = c.fetch([]string{
				nameTok, opTok, lexer.EofTokenName, lexer.EoiTokenName,
			}, true, nil)
		}
	}

	return c.result, e
}

func (c *parseContext) put(t *lexer.Token) {
	if c.savedToken != nil {
		panic("cannot put " + t.TypeName() + " token: already put " + c.savedToken.TypeName())
	}

	c.savedToken = t
}

func isEof(t *lexer.Token) bool {
	tt := t.Type()
	return (tt == lexer.EofTokenType || tt == lexer.EoiTokenType)
}

func (c *parseContext) fetch(types []string, strict bool, e error) (*lexer.Token, error) {
	if e != nil {
		return nil, e
	}

	token := c.savedToken
	if token == nil {
		token, e = llxLexer.Next(c.q)
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
		c.savedToken = nil
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

	c.put(token)
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

		codePoint, e := strconv.ParseUint(string(content[2:hexLen+2]), 16, 32)
		if e != nil {
			return 0, invalidEscapeError(token, string(content))
		}

		if utf8.ValidRune(rune(codePoint)) {
			return rune(codePoint), nil
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

func (c *parseContext) fetchOne(typ string, strict bool, e error) (*lexer.Token, error) {
	return c.fetch([]string{typ}, strict, e)
}

func (c *parseContext) fetchAll(types []string, e error) ([]*lexer.Token, error) {
	if e != nil {
		return nil, e
	}

	result := make([]*lexer.Token, 0)
	for {
		t, e := c.fetch(types, false, nil)
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

func (c *parseContext) skip(types []string, e error) error {
	if e != nil {
		return e
	}

	_, e = c.fetch(types, true, nil)
	return e
}

func (c *parseContext) skipOne(typ string, e error) error {
	return c.skip([]string{typ}, e)
}

func (c *parseContext) addToken(name, re string, flags grammar.TokenFlags) int {
	var t extraToken
	i, has := c.extraIndex[name]
	if has {
		t = c.extraTokens[i]
		delete(c.extraIndex, name)
	}
	c.result.Tokens = append(c.result.Tokens, grammar.Token{name, re, t.group, flags | t.flags})
	index := len(c.result.Tokens) - 1
	c.tokenIndex[name] = index
	return index
}

func (c *parseContext) addLiteralToken(name string, flags grammar.TokenFlags) {
	_, has := c.literalIndex[name]
	if !has {
		c.literalIndex[name] = len(c.literals)
		c.literals = append(c.literals, literalToken{name, flags})
	}
}

func (c *parseContext) useLiteralToken(name string, flags grammar.TokenFlags) int {
	i, has := c.literalIndex[name]
	if has {
		return i
	}

	i = len(c.result.Tokens)
	c.result.Tokens = append(c.result.Tokens, grammar.Token{name, "", 0, flags | grammar.LiteralToken})
	c.literalIndex[name] = i
	return i
}

func (c *parseContext) addExtraToken(name string) int {
	i, has := c.extraIndex[name]
	if !has {
		i = len(c.extraTokens)
		c.extraTokens = append(c.extraTokens, extraToken{name: name})
		c.extraIndex[name] = i
	}
	return i
}

func (c *parseContext) addTokenFlag(name string, flag grammar.TokenFlags) {
	i, has := c.tokenIndex[name]
	if has {
		c.result.Tokens[i].Flags |= flag
	} else {
		i = c.addExtraToken(name)
		c.extraTokens[i].flags |= flag
	}
}

func (c *parseContext) parseDir(tok *lexer.Token) error {
	name := tok.Text()
	var e error

	switch name {
	case asideDir, caselessDir, externDir, errorDir:
		e = c.parseTokenFlagDir(name)
	case groupDir:
		e = c.parseGroupDir()
	case reservedDir:
		e = c.parseReservedDir()
	case literalDir:
		e = c.parseLiteralDir()
	default:
		e = unknownDirectiveError(tok)
	}

	return e
}

func (c *parseContext) parseTokenFlagDir(name string) error {
	tokens, e := c.fetchAll([]string{tokenNameTok}, nil)
	e = c.skipOne(semicolonTok, e)
	if e != nil {
		return e
	}

	var flag grammar.TokenFlags = 0
	switch name {
	case asideDir:
		flag = grammar.AsideToken
	case caselessDir:
		flag = grammar.CaselessToken
	case externDir:
		flag = grammar.ExternalToken
	case errorDir:
		flag = grammar.ErrorToken
	}
	for _, token := range tokens {
		c.addTokenFlag(token.Text()[1:], flag)
	}

	return nil
}

func (c *parseContext) parseGroupDir() error {
	tokens, e := c.fetchAll([]string{tokenNameTok}, nil)
	e = c.skipOne(semicolonTok, e)
	if e != nil {
		return e
	}

	c.currentGroup++
	for _, token := range tokens {
		name := token.Text()[1:]
		i, has := c.tokenIndex[name]
		if has {
			if c.result.Tokens[i].Group != 0 {
				return reassignedGroupError(name)
			}

			c.result.Tokens[i].Group = c.currentGroup
		} else {
			i = c.addExtraToken(name)
			c.extraTokens[i].group = c.currentGroup
		}
	}
	return nil
}

func (c *parseContext) parseReservedDir() error {
	flags := grammar.ReservedToken
	tokens, e := c.fetchAll([]string{stringTok}, nil)
	e = c.skipOne(semicolonTok, e)
	if e != nil {
		return e
	}

	for _, t := range tokens {
		text := t.Text()
		c.addLiteralToken(text[1:len(text)-1], flags)
	}
	return nil
}

func (c *parseContext) parseLiteralDir() error {
	tokens, e := c.fetchAll([]string{stringTok, tokenNameTok}, nil)
	e = c.skipOne(semicolonTok, e)
	if e != nil {
		return e
	}

	for _, t := range tokens {
		text := t.Text()
		if t.TypeName() == tokenNameTok {
			c.restrictLiteralTypes = true
			c.addTokenFlag(text[1:], grammar.NoLiteralsToken) // will be xor-ed later
		} else {
			c.restrictLiterals = true
			c.addLiteralToken(text[1:len(text)-1], 0)
		}
	}
	return nil
}

func (c *parseContext) parseLayerDef(t *lexer.Token) error {
	if t.Text() != "@" {
		return unexpectedTokenError(t)
	}

	token, e := c.fetchOne(nameTok, true, nil)
	if e != nil {
		return e
	}

	layer := grammar.Layer{Type: token.Text()}

	for {
		token, e = c.fetch([]string{nameTok, semicolonTok}, true, nil)
		if e != nil {
			return e
		}

		if token.TypeName() == opTok {
			break
		}

		command := grammar.LayerCommand{Command: token.Text()}
		e = c.skipOne(lBraceTok, nil)
		if e != nil {
			return e
		}

		token, _ = c.fetchOne(rBraceTok, false, e)
		if token != nil {
			layer.Commands = append(layer.Commands, command)
			continue
		}

		for {
			token, e = c.fetch([]string{stringTok, nameTok}, true, nil)
			if e != nil {
				return e
			}

			arg := token.Text()
			if token.TypeName() == stringTok {
				arg = arg[1 : len(arg)-1]
			}
			command.Arguments = append(command.Arguments, arg)

			token, e = c.fetch([]string{commaTok, rBraceTok}, true, nil)
			if e != nil {
				return e
			}

			if token.Text() == rBraceTok {
				break
			}
		}

		layer.Commands = append(layer.Commands, command)
	}

	c.result.Layers = append(c.result.Layers, layer)
	return nil
}

func (c *parseContext) parseTemplateDef(name string) error {
	e := c.skipOne(equTok, nil)
	re, e := c.fetchRegexp(e)
	e = c.skipOne(semicolonTok, e)
	if e != nil {
		return e
	}

	c.templates[name] = re

	return nil
}

func (c *parseContext) parseTokenDef(name string) error {
	e := c.skipOne(equTok, nil)
	re, e := c.fetchRegexp(e)
	e = c.skipOne(semicolonTok, e)
	if e != nil {
		return e
	}

	c.addToken(name, re, 0)

	return nil
}

func (c *parseContext) fetchRegexp(e error) (string, error) {
	types := []string{regexpTok, nameTok}
	tokens, e := c.fetchAll(types, e)
	if e != nil {
		return "", e
	}

	if len(tokens) == 0 {
		_, e = c.fetch(types, true, nil)
		return "", e
	}

	var contents []byte
	for _, token := range tokens {
		switch token.TypeName() {
		case regexpTok:
			content := token.Content()
			contents = append(contents, content[1:len(content)-1]...)

		case nameTok:
			content, has := c.templates[token.Text()]
			if !has {
				return "", unknownTemplateError(token, token.Text())
			}

			contents = append(contents, content...)
		}
	}

	re := string(contents)
	_, e = regexp.Compile(re)
	if e != nil {
		return "", regexpError(tokens[0], e)
	}

	return re, nil
}

func (c *parseContext) addNode(name string, define bool) *nodeItem {
	var group *groupChunk = nil
	if define {
		group = newGroupChunk(false, false)
	}
	result := c.result.NIndex[name]
	if result != nil {
		if result.Chunk == nil && define {
			result.Chunk = group
		}
		return result
	}

	result = &nodeItem{len(c.result.Nodes), ints.NewSet(), ints.NewSet(), group}
	c.result.NIndex[name] = result
	c.result.Nodes = append(c.result.Nodes, grammar.Node{name, 0})
	return result
}

func (c *parseContext) parseNodeDef(name string) error {
	nt := c.addNode(name, true)
	e := c.skipOne(equTok, nil)
	e = c.parseGroup(name, nt.Chunk, e)
	e = c.skipOne(semicolonTok, e)
	return e
}

func (c *parseContext) parseGroup(name string, group complexChunk, e error) error {
	if e != nil {
		return e
	}

	for {
		item, e := c.parseVariants(name)
		if e != nil {
			return e
		}

		group.Append(item)
		t, e := c.fetchOne(commaTok, false, nil)
		if t == nil {
			return e
		}
	}
}

func (c *parseContext) parseVariants(name string) (chunk, error) {
	ch, e := c.parseVariant(name)
	t, e := c.fetchOne(pipeTok, false, e)
	if e != nil {
		return nil, e
	} else if t == nil {
		return ch, nil
	}

	result := newVariantChunk()
	result.Append(ch)

	for {
		ch, e = c.parseVariant(name)
		t, e = c.fetchOne(pipeTok, false, e)
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

func (c *parseContext) parseVariant(name string) (chunk, error) {
	variantHeads := []string{nameTok, tokenNameTok, stringTok, lBraceTok, lSquareTok, lCurlyTok}
	t, e := c.fetch(variantHeads, true, nil)
	if e != nil {
		return nil, e
	}

	var (
		index int
		f     bool
	)
	switch t.TypeName() {
	case nameTok:
		nt := c.addNode(t.Text(), false)
		c.result.NIndex[name].DependsOn.Add(nt.Index)
		return newNodeChunk(t.Text(), nt), nil

	case tokenNameTok:
		tokenName := t.Text()[1:]
		if tokenName == "" {
			return eoiChunk{}, nil
		}

		index, f = c.tokenIndex[tokenName]
		if !f {
			return nil, tokenError(t)
		}

		if (c.result.Tokens[index].Flags & unusedToken) != 0 {
			return nil, wrongTokenError(t)
		}
		return newTokenChunk(index), nil

	case stringTok:
		name = t.Text()[1 : len(t.Text())-1]
		index, f = c.literalIndex[name]
		if !f {
			if c.restrictLiterals {
				return nil, unknownLiteralError(name)
			}

			index = c.useLiteralToken(name, 0)
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
	e = c.parseGroup(name, result, nil)
	e = c.skipOne(lastToken, e)
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
