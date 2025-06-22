package restrict

import (
	"context"
	"fmt"
	"testing"

	"github.com/ava12/llx/internal/test"
	"github.com/ava12/llx/langdef"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/parser/layers/common"
)

func TestConfigErrors(t *testing.T) {
	grammar := `!aside $space; !extern $in $de; $space = /\s+/; $char = /\w/; g = {$char};`
	samples := []struct {
		layer string
		code  int
	}{
		{"@restrict node(foo) forbid-in(bar) allow-in(baz);", 0},
		{"@restrict node(foo) forbid-in(bar);", 0},
		{"@restrict node(foo, bar) forbid-in(baz, qux) allow-in(one, two);", 0},

		{"@restrict foo();", common.UnknownCommandError},
		{"@restrict node(foo) allow-in(bar);", common.MissingCommandError},
		{"@restrict allow-in(foo) forbid-in(bar);", common.MissingCommandError},
		{"@restrict node();", common.NumberOfArgumentsError},
		{"@restrict allow-in();", common.NumberOfArgumentsError},
		{"@restrict forbid-in();", common.NumberOfArgumentsError},
		{"@restrict node(foo, foo);", common.InvalidArgumentError},
		{"@restrict node(foo) node(foo);", common.InvalidArgumentError},
		{"@restrict allow-in(foo, foo);", common.InvalidArgumentError},
		{"@restrict allow-in(foo) allow-in(foo);", common.InvalidArgumentError},
		{"@restrict forbid-in(foo, foo);", common.InvalidArgumentError},
		{"@restrict forbid-in(foo) forbid-in(foo);", common.InvalidArgumentError},
	}

	for i, sample := range samples {
		name := fmt.Sprintf("sample #%d", i)
		t.Run(name, func(t *testing.T) {
			g, e := langdef.ParseString("", grammar+sample.layer)
			test.Assert(t, e == nil, "unexpected error: %s", e)
			_, e = parser.New(g)
			if sample.code == 0 {
				test.ExpectNoError(t, e)
			} else {
				test.ExpectErrorCode(t, sample.code, e)
			}
		})
	}
}

func TestRuntimeErrors(t *testing.T) {
	grammar := `!aside $space; $space = /\s+/; $op = /[()]/; $name = /\w+/;
		@restrict node(apple) allow-in(grass, tree) forbid-in(dirt, box);
		g = grass | dirt | rock; apple = 'apple'; rock = 'rock', '(', {place | $name}, ')';
		grass = 'grass', '(', {place | $name}, ')'; dirt = 'dirt', '(', {place | $name}, ')';
		place = tree | box | plate; tree = 'tree', '(', {apple | $name}, ')';
		box = 'box', '(', {apple | $name}, ')'; plate = 'plate', '(', {apple | $name}, ')';
	`
	samples := []struct {
		src   string
		valid bool
	}{
		{"grass(apple)", true},
		{"rock(apple)", true},
		{"dirt(apple)", true},
		{"dirt(plate(apple))", false},
		{"grass(plate(apple))", true},
		{"grass(tree(apple))", true},
		{"dirt(tree(apple))", true},
		{"rock(plate(apple))", true},
		{"grass(box(apple))", false},
		{"dirt(plate(apple))", false},
		{"dirt(box(pear))", true},
	}

	g, e := langdef.ParseString("", grammar)
	test.ExpectNoError(t, e)

	p, e := parser.New(g)
	test.ExpectNoError(t, e)

	for i, sample := range samples {
		name := fmt.Sprintf("sample #%d (%q)", i, sample.src)
		t.Run(name, func(t *testing.T) {
			_, e := p.ParseString(context.Background(), "", sample.src, parser.Hooks{})
			if sample.valid {
				test.ExpectNoError(t, e)
			} else {
				test.ExpectErrorCode(t, common.WrongTokenError, e)
			}
		})
	}
}
