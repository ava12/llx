/*
Package restrict contains layer template to restrict specific nodes based on ancestor node types.

Contains no exported definitions. Registers a hook layer template named "restrict",
panics if a template with this name is already registered.

This layer is useful when there's need to restrict usage of some syntax constructs,
but doing it grammatically is very hard. E.g. a "continue" statement may appear inside many nodes,
but only if they are wrapped in a "for" or a "while" loop body. Outside a loop body this statement
is a syntax error. Technically, this problem may be solved by splitting nodes which can potentially
contain this statement into two kinds: inside a loop body and outside it,
like "loop-expression" and "expression". But this solution is bulky and ugly,
and the problem becomes even worse if there is a "break" statement which can appear
inside a "switch" block as well.

The layer stores a list of "special" nodes which either allow or forbid specific restricted nodes.
When a hook for new restricted node is triggered the layer searches the the open node stack.
If there are "special" nodes in the stack and the closest "special" ancestor is a forbidding one,
then an error is emitted.

The layer definition looks like:

	@restrict node(continue-stmt) allow-in(for-body, while-body) forbid-in(function-body, root);

"node" and "forbid-in" commands are required.

	node(<restricted_node>, ...)

Takes one or more names of restricted nodes. Names must not repeat.

	forbid-in(<ancestor_node>, ...)

Takes one or more names of nodes forbidding restricted nodes. Names must not repeat.

	allow-in(<ancestor_node>, ...)

Takes one or more names of nodes allowing restricted nodes. Names must not repeat.
*/
package restrict

import (
	"context"
	"fmt"

	"github.com/ava12/llx/grammar"
	"github.com/ava12/llx/parser"
	"github.com/ava12/llx/parser/layers/common"
)

const (
	layerName = "restrict"

	nodeCommand     = "node"
	allowInCommand  = "allow-in"
	forbidInCommand = "forbid-in"
)

const (
	errListed = "this node is already listed"
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
	result := layer{
		Nodes: make(parser.NodeHooks),
	}
	specialNodes := make(map[string]bool)

	hook := func(ctx context.Context, node string, tok *parser.Token, nc *parser.NodeContext) (parser.NodeHookInstance, error) {
		for _, ancestor := range nc.NodeStack() {
			allow, isSpecial := specialNodes[ancestor]
			if isSpecial {
				if allow {
					break

				} else {
					reason := fmt.Sprintf("%s is not allowed inside %s", node, ancestor)
					return nil, common.MakeWrongTokenError(layerName, tok, reason)
				}
			}
		}

		return nil, nil
	}

	restricted := false

	for _, command := range commands {
		switch command.Command {

		case nodeCommand:
			if len(command.Arguments) == 0 {
				return nil, common.MakeNumberOfArgumentsError(layerName, command.Command, 1, 0)
			}

			for _, node := range command.Arguments {
				_, has := result.Nodes[node]
				if has {
					return nil, common.MakeInvalidArgumentError(layerName, nodeCommand, node, errListed)
				}

				result.Nodes[node] = hook
			}

		case allowInCommand, forbidInCommand:
			if len(command.Arguments) == 0 {
				return nil, common.MakeNumberOfArgumentsError(layerName, command.Command, 1, 0)
			}

			allow := command.Command == allowInCommand
			if !allow {
				restricted = true
			}
			for _, node := range command.Arguments {
				_, has := specialNodes[node]
				if has {
					return nil, common.MakeInvalidArgumentError(layerName, command.Command, node, errListed)
				}

				specialNodes[node] = allow
			}

		default:
			return nil, common.MakeUnknownCommandError(layerName, command.Command)
		}
	}

	if len(result.Nodes) == 0 {
		return nil, common.MakeMissingCommandError(layerName, nodeCommand)
	}

	if !restricted {
		return nil, common.MakeMissingCommandError(layerName, forbidInCommand)
	}

	return result, nil
}
