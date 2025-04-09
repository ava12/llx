package llx_test

import (
	"context"
	"fmt"

	"github.com/ava12/llx/langdef"
	"github.com/ava12/llx/parser"
)

func Example() {
	input := `
foo = hello
bar = world
[sec]
baz =
[sec.subsec]
qux = !
`
	grammar := `
$space = /[ \t\r]+/; $nl = /\n/;
$op = /[=\[\]]/;
$name = /[a-z]+/; $sec-name = /[a-z]+(?:\.[a-z]+)*/;
$value = /[^\n]+/;

!aside $space;
!group $sec-name; !group $value $nl; !group $op $name $nl;

config = {section | value | $nl};
section = '[', $sec-name, ']', $nl;
value = $name, '=', [$value], $nl;
`
	configGrammar, e := langdef.ParseString("example grammar", grammar)
	if e != nil {
		fmt.Println(e)
		return
	}

	configParser, e := parser.New(configGrammar)
	if e != nil {
		panic(e)
	}

	result := make(map[string]string)
	prefix, name, value := "", "", ""
	hooks := parser.Hooks{Tokens: parser.TokenHooks{
		parser.AnyToken: func(_ context.Context, t *parser.Token, _ *parser.TokenContext) (emit bool, extra []*parser.Token, e error) {
			switch t.TypeName() {
			case "sec-name":
				prefix = t.Text() + "."
			case "name":
				name = prefix + t.Text()
			case "value":
				value = t.Text()
			case "nl":
				if name != "" {
					result[name] = value
					name, value = "", ""
				}
			}
			return true, nil, nil
		},
	}}
	_, e = configParser.ParseString(context.Background(), "input", input, hooks)
	if e == nil {
		fmt.Println(result)
	} else {
		fmt.Println(e)
	}
}
