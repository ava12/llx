package tree_test

import (
	"fmt"
	"github.com/ava12/llx/langdef"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/tree"
)

func ExampleWalk() {
	input := "foo=bar"
	grammar := `$name = /\w+/; $op = /=/; g = var, "=", value; var = $name; value = $name;`
	g, e := langdef.ParseString("grammar", grammar)
	if e != nil {
		fmt.Println(e)
		return
	}

	p, _ := parser.New(g)
	h := parser.Hooks{Nodes: parser.NodeHooks{
		parser.AnyNode: tree.NodeHook,
	}}

	root, e := p.ParseString("input", input, &h)
	if e != nil {
		fmt.Println(e)
		return
	}

	indent := "----------"
	visitor := func(stat tree.WalkStat) tree.WalkerFlags {
		el := stat.Element
		if el.IsNode() {
			fmt.Printf("%s%s:\n", indent[:stat.Level*2], el.TypeName())
		} else {
			fmt.Printf("%s%s %q\n", indent[:stat.Level*2], el.TypeName(), el.Token().Text())
		}
		return 0
	}
	tree.Walk(root.(tree.Element), tree.WalkLtr, visitor)
	// Output:
	// g:
	// --var:
	// ----name "foo"
	// --op "="
	// --value:
	// ----name "bar"
}
