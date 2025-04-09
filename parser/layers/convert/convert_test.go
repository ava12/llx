package convert

import (
	"context"
	"fmt"
	"testing"

	"github.com/ava12/llx/internal/test"
	"github.com/ava12/llx/langdef"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/parser/layers/common"
)

func TestErrors(t *testing.T) {
	grammar := `$char = /\w/; g = {$char};`
	samples := []struct {
		layer string
		code  int
	}{
		{"@convert input-type(char);", common.MissingCommandError},
		{"@convert input-type(name);", common.UnknownTokenTypeError},
		{"@convert input(char);", common.UnknownCommandError},
		{"@convert input-type(char, name);", common.NumberOfArgumentsError},
		{"@convert input-type();", common.NumberOfArgumentsError},
		{"@convert convert();", common.NumberOfArgumentsError},
		{"@convert convert(a);", common.NumberOfArgumentsError},
		{"@convert input-type(char) input-type(char);", common.CommandAlreadyUsedError},
	}

	for i, sample := range samples {
		name := fmt.Sprintf("sample #%d", i)
		t.Run(name, func(t *testing.T) {
			g, e := langdef.ParseString("", grammar+sample.layer)
			test.Assert(t, e == nil, "unexpected error: %s", e)
			_, e = parser.New(g)
			test.ExpectErrorCode(t, sample.code, e)
		})
	}
}

func checkToken(t *testing.T, tok *parser.Token, pos, tt int, ttn, text string) {
	if tok.Type() == tt && tok.TypeName() == ttn && tok.Text() == text && tok.Pos().Pos() == pos {
		return
	}

	t.Fatalf("expecting token %s(%d)%q@%d, got %s(%d)%q@%d",
		ttn, tt, text, pos, tok.TypeName(), tok.Type(), tok.Text(), tok.Pos().Pos())
}

func TestSavePosition(t *testing.T) {
	grammar := `$char = /./; g = {$char};`
	src := "abc"
	samples := []struct {
		layer string
		texts []string
		pos   []int
	}{
		{"@convert convert(b, d);", []string{"a", "d", "c"}, []int{0, 0, 2}},
		{"@convert save-position() convert(b, d);", []string{"a", "d", "c"}, []int{0, 1, 2}},
	}

	for i, sample := range samples {
		name := fmt.Sprintf("sample #%d", i)
		t.Run(name, func(t *testing.T) {
			g, e := langdef.ParseString("", grammar+sample.layer)
			test.Assert(t, e == nil, "unexpected error: %s", e)

			p, e := parser.New(g)
			test.Assert(t, e == nil, "unexpected error: %s", e)

			texts := sample.texts
			pos := sample.pos
			handler := func(_ context.Context, token *parser.Token, _ *parser.TokenContext) (emit bool, extra []*parser.Token, e error) {
				test.Assert(t, len(texts) != 0, "excessive token %v", token)

				checkToken(t, token, pos[0], 0, "char", texts[0])

				pos = pos[1:]
				texts = texts[1:]
				return true, nil, nil
			}

			_, e = p.ParseString(context.Background(), "", src, parser.Hooks{
				Tokens: parser.TokenHooks{
					parser.AnyToken: handler,
				},
			})

			test.Assert(t, e == nil, "unexpected error: %s", e)
			test.Assert(t, len(texts) == 0, "expecting %d more tokens", len(texts))
		})
	}
}

func TestConvert(t *testing.T) {
	grammar := `!aside $space; $space = /\s+/; $name = /[a-z]+/; $number = /\d+/; g = {$name | $number};`
	typeNames := []string{"space", "name", "number"}
	src := "foo 1 bar 2 baz"
	tokenPos := []int{0, 4, 6, 10, 12}
	samples := []struct {
		layer string
		texts []string
		types []int
	}{
		{"@convert save-position() convert(bar, qux);",
			[]string{"foo", "1", "qux", "2", "baz"}, []int{1, 2, 1, 2, 1}},
		{"@convert save-position() input-type(name) convert(bar, qux);",
			[]string{"foo", "1", "qux", "2", "baz"}, []int{1, 2, 1, 2, 1}},
		{"@convert save-position() output-type(name) convert(bar, qux);",
			[]string{"foo", "1", "qux", "2", "baz"}, []int{1, 2, 1, 2, 1}},
		{"@convert save-position() input-type(name) output-type(name) convert(bar, qux);",
			[]string{"foo", "1", "qux", "2", "baz"}, []int{1, 2, 1, 2, 1}},
		{"@convert save-position() output-type(number) convert(bar, '3');",
			[]string{"foo", "1", "3", "2", "baz"}, []int{1, 2, 2, 2, 1}},
		{"@convert save-position() input-type(name) output-type(number) convert(bar, '3');",
			[]string{"foo", "1", "3", "2", "baz"}, []int{1, 2, 2, 2, 1}},
	}

	for i, sample := range samples {
		name := fmt.Sprintf("sample #%d", i)
		t.Run(name, func(t *testing.T) {
			g, e := langdef.ParseString("", grammar+sample.layer)
			test.Assert(t, e == nil, "unexpected error: %s", e)

			p, e := parser.New(g)
			test.Assert(t, e == nil, "unexpected error: %s", e)

			pos := tokenPos
			texts := sample.texts
			types := sample.types
			handler := func(_ context.Context, token *parser.Token, _ *parser.TokenContext) (emit bool, extra []*parser.Token, e error) {
				tt := token.Type()
				if tt == 0 {
					return false, nil, nil
				}

				test.Assert(t, len(types) != 0, "excessive token %v", token)

				checkToken(t, token, pos[0], types[0], typeNames[types[0]], texts[0])

				pos = pos[1:]
				texts = texts[1:]
				types = types[1:]
				return true, nil, nil
			}

			_, e = p.ParseString(context.Background(), "", src, parser.Hooks{
				Tokens: parser.TokenHooks{
					parser.AnyToken: handler,
				},
			})

			test.Assert(t, e == nil, "unexpected error: %s", e)
			test.Assert(t, len(texts) == 0, "expecting %d more tokens", len(texts))
		})
	}
}
