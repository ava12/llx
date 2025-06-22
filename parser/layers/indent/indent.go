/*
Package indent contains layer template to emit specific external tokens
when source indentation level changes.

Contains no exported definitions. Registers a hook layer template named "indent",
panics if a template with this name is already registered.

The layer definition looks like:

	@indent space(space, nl) on-indent(in) on-dedent(de);

All commands are required, each command must be used exactly once.

	space(<token_type>, ...)

Takes one or more type names of tokens consisting of space and/or newline symbols (U+000a).
These are the only token types used to detect line breaks and indentations.

	on-indent(<token_type>)

Takes exactly one type name of a token that must be emitted every time an indentation level increases.

	on-dedent(<token_type>)

Takes exactly one type name of a token that must be emitted every time an indentation level decreases.

Indented source line starts with zero or more spacing symbols followed by a non-aside token.
Empty lines and lines containing only aside tokens are ignored.
All other variants (e.g. a line starting with a comment followed by a non-aside token) emit an error.
The first non-aside token in a source code must have zero indentation level.

The layer treats indentations as pairs: each on-indent token must match an on-dedent token.
The layer keeps a stack of nested indentations. The layer peeks the next token and checks
current indentation level if that token is a non-aside one.
Emits one on-indent token if valid indentation is a prefix of current line indentation.
Emits required number of on-dedent tokens if current line indentation equals one of saved indentations.
Does nothing if current line indentation is valid.
Otherwise (current indentation is shorter than the valid one and matches none of stack entries) emits an error.
Emits on-dedent token for each indentation stack item if the peeked token is an end-of-input.
*/
package indent

import (
	"bytes"
	"context"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/parser/layers/common"
)

const (
	layerName = "indent"

	spaceCommand    = "space"
	onIndentCommand = "on-indent"
	onDedentCommand = "on-dedent"
)

type tokenRole byte

const (
	commonRole tokenRole = iota
	spaceRole
	asideRole
	emptyRole
	eoiRole
)

type state byte

const (
	validIndent   state = iota // after valid indent, no in/dedent token needed
	incIndent                  // after valid indent deeper than current one, no indent token sent yet
	decIndent                  // after valid indent shallower than current one, no dedent tokens sent yet
	invalidIndent              // got invalid indent
	indentAside                // aside token after indent
	indented                   // got non-empty non-space non-aside token, in/dedent tokens sent if needed
)

const (
	errInvalidIndent = "invalid indentation"
)

func init() {
	e := parser.RegisterHookLayer(layerName, template{})
	if e != nil {
		panic(e)
	}
}

type layerState struct {
	parser                 *parser.Parser
	indentType, dedentType string
	roles                  []tokenRole
	indentStack            [][]byte
	validIndent            []byte
	currentIndent          []byte
	state                  state
}

type layer struct {
	parser                 *parser.Parser
	indentType, dedentType string
	roles                  []tokenRole
}

func (ls *layerState) handle(tok *parser.Token, tc *parser.TokenContext) (bool, []*parser.Token, error) {
	var extra []*parser.Token
	var e error
	emit := true
	role := ls.role(tok)

	switch role {
	case eoiRole:
		emit = false
		extra, e = ls.handleEoi(tok)
	case commonRole:
		emit, extra, e = ls.handleCommon(tok)
	case spaceRole:
		emit, extra, e = ls.handleSpace(tok, tc)
	case asideRole:
		e = ls.handleAside(tc)
	}

	return emit, extra, e
}

func (ls *layerState) role(tok *parser.Token) tokenRole {
	tt := tok.Type()
	if tt == lexer.EoiTokenType {
		return eoiRole
	}

	if tt < 0 || len(tok.Content()) == 0 {
		return emptyRole
	}

	if tt < len(ls.roles) {
		return ls.roles[tt]
	}

	return commonRole
}

func (ls *layerState) indent() ([]*parser.Token, error) {
	ls.indentStack = append(ls.indentStack, ls.validIndent)
	ls.validIndent = ls.currentIndent
	indent, e := ls.parser.MakeToken(ls.indentType, nil)
	if e != nil {
		return nil, e
	}

	return []*parser.Token{indent}, nil
}

func (ls *layerState) dedent() ([]*parser.Token, error) {
	var result []*parser.Token
	l := len(ls.currentIndent)

	for i := len(ls.indentStack) - 1; i >= 0; i-- {
		dedent, e := ls.parser.MakeToken(ls.dedentType, nil)
		if e != nil {
			return nil, e
		}

		result = append(result, dedent)
		if len(ls.indentStack[i]) == l {
			ls.indentStack = ls.indentStack[0:i]
			break
		}
	}

	ls.validIndent = ls.currentIndent
	return result, nil
}

func (ls *layerState) handleCommon(tok *parser.Token) (bool, []*parser.Token, error) {
	emit := true
	var extra []*parser.Token
	var e error

	switch ls.state {

	case validIndent:
		ls.state = indented

	case incIndent:
		extra, e = ls.indent()
		emit = false
		if e == nil {
			extra = append(extra, tok)
			ls.state = indented
		}

	case decIndent:
		extra, e = ls.dedent()
		emit = false
		if e == nil {
			extra = append(extra, tok)
			ls.state = indented
		}

	case invalidIndent, indentAside:
		e = common.MakeWrongTokenError(layerName, tok, errInvalidIndent)
	}

	return emit, extra, e
}

func (ls *layerState) handleSpace(tok *parser.Token, tc *parser.TokenContext) (bool, []*parser.Token, error) {
	content, nlFound := ls.stripNl(tok.Content())
	if nlFound {
		ls.currentIndent = nil
		if len(ls.validIndent) == 0 {
			ls.state = validIndent
		} else {
			ls.state = decIndent
		}
	}

	if ls.state == indented {
		return true, nil, nil
	}

	if ls.state != indentAside {
		ls.state = ls.adjustIndent(content)
	}
	nextTok, e := tc.PeekToken()
	if e != nil {
		return false, nil, e
	}

	role := ls.role(nextTok)
	if role != commonRole {
		return true, nil, nil
	}

	var extra []*parser.Token
	switch ls.state {
	case invalidIndent, indentAside:
		e = common.MakeWrongTokenError(layerName, tok, errInvalidIndent)
	case incIndent:
		extra, e = ls.indent()
	case decIndent:
		extra, e = ls.dedent()
	}

	if e == nil {
		ls.state = indented
	}

	return true, extra, e
}

func (ls *layerState) stripNl(content []byte) ([]byte, bool) {
	nlPos := bytes.LastIndexByte(content, '\n')
	if nlPos >= 0 {
		return content[nlPos+1:], true
	} else {
		return content, false
	}
}

func (ls *layerState) adjustIndent(content []byte) state {
	if len(ls.currentIndent) == 0 {
		ls.currentIndent = content
	} else { // fetching multiple space tokens in a row, just in case...
		tmp := make([]byte, len(ls.currentIndent), len(ls.currentIndent)+len(content))
		copy(tmp, ls.currentIndent) // avoid spoiling source file
		ls.currentIndent = append(ls.currentIndent, content...)
	}

	currentLen := len(ls.currentIndent)
	validLen := len(ls.validIndent)
	if currentLen > validLen {
		if bytes.HasPrefix(ls.currentIndent, ls.validIndent) {
			return incIndent
		}

	} else {
		if bytes.HasPrefix(ls.validIndent, ls.currentIndent) {
			if currentLen == validLen {
				return validIndent

			} else {
				for i := len(ls.indentStack) - 1; i >= 0; i-- {
					if len(ls.indentStack[i]) == currentLen {
						return decIndent
					}
				}
			}
		}
	}

	return invalidIndent
}

func (ls *layerState) handleAside(tc *parser.TokenContext) error {
	if ls.state == indented {
		return nil
	}

	ls.state = indentAside
	tok, e := tc.PeekToken()
	if e != nil || tok == nil || ls.role(tok) != commonRole {
		return e
	}

	return common.MakeWrongTokenError(layerName, tok, errInvalidIndent)
}

func (ls *layerState) handleEoi(tok *parser.Token) ([]*parser.Token, error) {
	var result []*parser.Token

	if len(ls.indentStack) != 0 {
		for range ls.indentStack {
			dedent, e := ls.parser.MakeToken(ls.dedentType, nil)
			if e != nil {
				return nil, e
			}

			result = append(result, dedent)
		}
	}

	return append(result, tok), nil
}

func (l *layer) Init(context.Context, *parser.ParseContext) parser.Hooks {
	ls := &layerState{
		parser:     l.parser,
		indentType: l.indentType,
		dedentType: l.dedentType,
		roles:      l.roles,
	}

	hook := func(_ context.Context, token *parser.Token, tc *parser.TokenContext) (emit bool, extra []*parser.Token, e error) {
		return ls.handle(token, tc)
	}

	return parser.Hooks{
		Tokens: parser.TokenHooks{
			parser.AnyToken: hook,
			parser.EoiToken: hook,
		},
	}
}

type template struct{}

func (t template) Setup(commands []grammar.LayerCommand, p *parser.Parser) (parser.HookLayer, error) {
	maxType := 0
	gotSpaces := false
	roleMap := make(map[int]tokenRole)
	l := &layer{
		parser: p,
	}

	for i, tokenDef := range p.Tokens() {
		if tokenDef.Flags&grammar.AsideToken != 0 {
			roleMap[i] = asideRole
			maxType = i
		}
	}

	for _, command := range commands {
		switch command.Command {

		case spaceCommand:
			gotSpaces = true
			if len(command.Arguments) == 0 {
				return nil, common.MakeNumberOfArgumentsError(layerName, command.Command, 1, 0)
			}

			for _, ttn := range command.Arguments {
				tt, valid := p.TokenType(ttn)
				if !valid {
					return nil, common.MakeUnknownTokenTypeError(layerName, command.Command, ttn)
				}

				if tt >= maxType {
					maxType = tt
				}
				roleMap[tt] = spaceRole
			}

		case onIndentCommand, onDedentCommand:
			if len(command.Arguments) != 1 {
				return nil, common.MakeNumberOfArgumentsError(layerName, command.Command, 1, len(command.Arguments))
			}

			ttn := command.Arguments[0]
			tt, valid := p.TokenType(ttn)
			if !valid || tt < 0 {
				return nil, common.MakeUnknownTokenTypeError(layerName, command.Command, ttn)
			}

			var prevType string
			if command.Command == onIndentCommand {
				prevType = l.indentType
				l.indentType = ttn
			} else {
				prevType = l.dedentType
				l.dedentType = ttn
			}
			if prevType != "" {
				return nil, common.MakeCommandAlreadyUsedError(layerName, command.Command)
			}
		}
	}

	var missingCommand string
	if !gotSpaces {
		missingCommand = spaceCommand
	} else if l.indentType == "" {
		missingCommand = onIndentCommand
	} else if l.dedentType == "" {
		missingCommand = onDedentCommand
	}
	if missingCommand != "" {
		return nil, common.MakeMissingCommandError(layerName, missingCommand)
	}

	l.roles = make([]tokenRole, maxType+1)
	for tt, role := range roleMap {
		l.roles[tt] = role
	}

	return l, nil
}
