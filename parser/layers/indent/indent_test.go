package indent

import (
	"context"
	"fmt"
	"testing"

	"github.com/ava12/llx/internal/test"
	"github.com/ava12/llx/langdef"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/parser/layers/common"
	pt "github.com/ava12/llx/parser/test"
)

func TestConfigErrors(t *testing.T) {
	grammar := `!aside $space; !extern $in $de; $space = /\s+/; $char = /\w/; g = {$char};`
	samples := []struct {
		layer string
		code  int
	}{
		{"@indent on-indent(in) on-dedent(de);", common.MissingCommandError},
		{"@indent space(space) on-indent(in);", common.MissingCommandError},
		{"@indent space(space) on-dedent(de);", common.MissingCommandError},
		{"@indent space(foo);", common.UnknownTokenTypeError},
		{"@indent on-indent(foo);", common.UnknownTokenTypeError},
		{"@indent on-dedent(foo);", common.UnknownTokenTypeError},
		{"@indent space();", common.NumberOfArgumentsError},
		{"@indent on-indent();", common.NumberOfArgumentsError},
		{"@indent on-indent(in, de);", common.NumberOfArgumentsError},
		{"@indent on-dedent();", common.NumberOfArgumentsError},
		{"@indent on-dedent(in, de);", common.NumberOfArgumentsError},
		{"@indent on-indent(in) on-indent(de);", common.CommandAlreadyUsedError},
		{"@indent on-dedent(in) on-dedent(de);", common.CommandAlreadyUsedError},
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

func TestRuntimeErrors(t *testing.T) {
	grammar := `!aside $space $comment; !extern $in $de; $space = /\s/; $comment = /\{.*?\}/; $name = /\w+/;
		g = st, {st}; st = ('do', $in, st, {st}, $de) | $name;
		@indent space(space) on-indent(in) on-dedent(de);`

	samples := []struct {
		src  string
		code int
	}{
		{"foo do bar", parser.UnexpectedTokenError},
		{"foo do\nbar", parser.UnexpectedTokenError},
		{"foo\ndo\nbar", parser.UnexpectedTokenError},
		{"foo\n  bar", parser.RemainingSourceError},
		{"do\n  foo\n bar", common.WrongTokenError},
		{"do\n\tfoo\n bar", common.WrongTokenError},
		{" foo", parser.UnexpectedTokenError},
		{"do\n  foo\n    bar", parser.UnexpectedTokenError},
		{"do\n  foo\n bar", common.WrongTokenError},
		{"{c}foo", common.WrongTokenError},
		{"do\n foo\n {c}bar", common.WrongTokenError},
		{"do\n foo\n{c} bar", common.WrongTokenError},
		{"do\n{c}foo", common.WrongTokenError},
	}

	g, e := langdef.ParseString("", grammar)
	test.ExpectNoError(t, e)

	p, e := parser.New(g)
	test.ExpectNoError(t, e)

	for i, sample := range samples {
		name := fmt.Sprintf("sample #%d (%s)", i, sample.src)
		t.Run(name, func(t *testing.T) {
			_, e = p.ParseString(context.Background(), name, sample.src, parser.Hooks{}, parser.WithFullSource())
			test.ExpectErrorCode(t, sample.code, e)
		})
	}
}

func TestParse(t *testing.T) {
	grammar := `!aside $space $comment; !extern $in $de; $space = /\s+/; $comment = /\{.*?\}/; $name = /\w+/;
		g = st, {st}; st = ('do', $in, st, {st}, $de) | $name;
		@indent space(space) on-indent(in) on-dedent(de);`

	samples := []struct {
		src, expected string
	}{
		{"foo", "(st foo)"},
		{"foo\n {c}\nbar", "(st foo) (st bar)"},
		{"do\n  foo\nbar", "(st do in (st foo) de) (st bar)"},
		{"do\n\tfoo\n\tbar", "(st do in (st foo) (st bar) de)"},
		{"do\n\tfoo  \n  {c}  \n\tbar", "(st do in (st foo) (st bar) de)"},
		{"do\n\tfoo\n {c} \n\tdo\n\t\tbar\n\tbaz\n\t\n", "(st do in (st foo) (st do in (st bar) de) (st baz) de)"},
	}

	g, e := langdef.ParseString("", grammar)
	test.ExpectNoError(t, e)

	p, e := parser.New(g)
	test.ExpectNoError(t, e)

	for i, sample := range samples {
		name := fmt.Sprintf("sample #%d (%s)", i, sample.src)
		t.Run(name, func(t *testing.T) {
			n, e := pt.ParseAsTestNode(context.Background(), p, sample.src, nil, nil)
			test.ExpectNoError(t, e)

			e = pt.NewTreeValidator(n, sample.expected).Validate()
			test.ExpectNoError(t, e)
		})
	}
}

func TestTokenOrder(t *testing.T) {
	grammar := `!aside $space $comment; !extern $in $de; $space = /\s+/; $comment = /\{.*?\}/; $name = /\w+/;
		g = st, {st}; st = ('do', $in, st, {st}, $de) | $name;
		@indent space(space) on-indent(in) on-dedent(de);`

	samples := []struct {
		src, expected string
	}{
		{"foo", "foo"},
		{"foo\nbar", "foo\nbar"},
		{"do\n  foo\nbar", "do(\n  foo)\nbar"},
		{"do\n  foo\n  bar", "do(\n  foo\n  bar)"},
		{"foo\n\t{c}\n  \nbar", "foo\n\t{c}\n  \nbar"},
		{"do\n do\n  do\n   foo\n", "do(\n do(\n  do(\n   foo)))\n"},
		{"do\n{c}\n foo", "do\n{c}(\n foo)"},
	}

	g, e := langdef.ParseString("", grammar)
	test.ExpectNoError(t, e)

	p, e := parser.New(g)
	test.ExpectNoError(t, e)

	var result []byte
	hooks := parser.Hooks{
		Tokens: parser.TokenHooks{
			parser.AnyToken: func(_ context.Context, token *parser.Token, _ *parser.TokenContext) (bool, []*parser.Token, error) {
				switch token.TypeName() {
				case "in":
					result = append(result, '(')
				case "de":
					result = append(result, ')')
				default:
					result = append(result, token.Content()...)
				}
				return true, nil, nil
			},
		},
	}

	for i, sample := range samples {
		name := fmt.Sprintf("sample #%d (%s)", i, sample.src)
		t.Run(name, func(t *testing.T) {
			result = nil
			_, e := p.ParseString(context.Background(), "", sample.src, hooks)
			test.ExpectNoError(t, e)

			test.ExpectString(t, sample.expected, string(result))
		})
	}
}
