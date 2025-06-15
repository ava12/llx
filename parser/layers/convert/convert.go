/*
Package convert contains layer template to convert specific tokens to other tokens
with different type and/or content.

Contains no exported definitions. Registers a hook layer template named "convert",
panics if a template with this name is already registered.

The layer definition looks like:

	@convert input-type(digraph) output-type(op) save-position() convert('(.', '[', '.)', ']');

Must contain at least one "convert" command:

	convert(<from>, <to>, <from>, <to>, ...)

Takes non-zero number of pairs of arguments, the first argument in a pair defines an input token text
and the second one defines an output token text.

A definition may also contain optional commands:

	input-type(<token_type>)

Takes a single argument defining hooked token type name.
By default all token types are hooked.

	output-type(<token_type>)

Takes a single argument defining an output token type name.
By default an output token has the same type as an input one.

	save-position()

Takes no arguments, instructs the layer to copy position information from an input token to an output one.
By default an output token has no position information.
*/
package convert

import (
	"context"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/internal/bmap"
	"github.com/ava12/llx/lexer"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/parser/layers/common"
	"github.com/ava12/llx/source"
)

const (
	layerName = "convert"

	convertCommand      = "convert"
	inputTypeCommand    = "input-type"
	outputTypeCommand   = "output-type"
	savePositionCommand = "save-position"

	convDefinedErr = "this conversion is already defined"
)

func init() {
	e := parser.RegisterHookLayer(layerName, template{})
	if e != nil {
		panic(e)
	}
}

type layer parser.Hooks

func (l layer) Init(context.Context, *parser.ParseContext) parser.Hooks {
	return parser.Hooks(l)
}

type template struct{}

func (t template) Setup(commands []grammar.LayerCommand, p *parser.Parser) (parser.HookLayer, error) {
	inputTypeName := parser.AnyToken
	outputTypeName := ""
	var outputType int
	replaces := bmap.New[[]byte](len(commands))
	savePosition := false
	gotReplaces := false
	var dynamicType bool

	for _, command := range commands {
		switch command.Command {

		case inputTypeCommand, outputTypeCommand:
			if len(command.Arguments) != 1 {
				return nil, common.MakeNumberOfArgumentsError(layerName, command.Command, 1, len(command.Arguments))
			}

			_, valid := p.TokenType(command.Arguments[0])
			if !valid {
				return nil, common.MakeUnknownTokenTypeError(layerName, command.Command, command.Arguments[0])
			}

			if command.Command == inputTypeCommand {
				if inputTypeName != "" {
					return nil, common.MakeCommandAlreadyUsedError(layerName, command.Command)
				}

				inputTypeName = command.Arguments[0]
			} else {
				if outputTypeName != "" {
					return nil, common.MakeCommandAlreadyUsedError(layerName, command.Command)
				}

				outputTypeName = command.Arguments[0]
			}

		case convertCommand:
			gotReplaces = true
			l := len(command.Arguments)
			if l == 0 || l&1 != 0 {
				return nil, common.MakeNumberOfArgumentsError(layerName, command.Command, (l+2)&^1, l)
			}

			for i := 0; i < l; i += 2 {
				from := command.Arguments[i]
				_, has := replaces.GetString(from)
				if has {
					return nil, common.MakeInvalidArgumentError(layerName, convertCommand, from, convDefinedErr)
				}

				replaces.SetString(from, []byte(command.Arguments[i+1]))
			}

		case savePositionCommand:
			savePosition = true

		default:
			return nil, common.MakeUnknownCommandError(layerName, command.Command)
		}
	}

	if !gotReplaces {
		return nil, common.MakeMissingCommandError(layerName, convertCommand)
	}

	if outputTypeName == "" && inputTypeName != "" {
		outputTypeName = inputTypeName
	}
	dynamicType = outputTypeName == ""
	if !dynamicType {
		outputType, _ = p.TokenType(outputTypeName)
	}

	return layer{
		Tokens: parser.TokenHooks{
			inputTypeName: func(_ context.Context, token *parser.Token, _ *parser.TokenContext) (emit bool, extra []*parser.Token, e error) {
				replace, has := replaces.Get(token.Content())
				if !has {
					return true, nil, nil
				}

				var pos source.Pos
				if savePosition {
					pos = token.Pos()
				}
				var newToken *parser.Token
				if dynamicType {
					newToken = lexer.NewToken(token.Type(), token.TypeName(), replace, pos)
				} else {
					newToken = lexer.NewToken(outputType, outputTypeName, replace, pos)
				}

				return false, []*parser.Token{newToken}, nil
			},
		},
	}, nil
}
