/*
Package grammar contains data types used by LL(*) parser.

Grammar is represented in the form of nondeterministic state machine.
The machine uses stack holding current syntax tree node and its ancestor nodes together with their current states.

State contains map of transition rules (or ambiguous variants of rules).
A key is either a token type id, a literal id, or AnyToken value.
When parser tries to match a rule for current token the priorities are:
  - a literal id;
  - a token type id (unless the token text matches some reserved literal);
  - AnyToken.

Transition rule contains the next state index (or FinalState value
if parsing of current node is finished) and new node index
(or SameNode value if no nested node is pushed).

Each time parser enters FinalState it drops current node.

When a rule is applied:
  - if new node is SameNode:
  - current state is set to next state;
  - if matched rule key is not AnyToken: next token is read;
  - else:
  - new node and its initial state index are pushed;
*/
package grammar

const (
	// RootNode is the index of root node in Grammar.Nodes.
	RootNode = 0
)

// BitSet is a general bit set, where i-th bit represents item with index i.
type BitSet = uint64

// MaxTokenType is a maximum allowed token type, not counting literals.
const MaxTokenType = 63

// TokenFlags contain information about token type.
type TokenFlags int

// Token contains information about some token or literal.
type Token struct {
	// Name is either a token type name or exact text for a literal.
	Name string

	// Re is RE2 expression defining this token type. Must not contain capturing groups.
	// Empty string for literal and external tokens.
	Re string

	// Group is the 0-based token group number, where 0 is the default one.
	// Each group effectively defines a separate lexer.
	Group int

	// Flags contain information about token type.
	Flags TokenFlags
}

const (
	// LiteralToken matches exact text of some token.
	LiteralToken TokenFlags = 1 << iota

	// ExternalToken is a token type that is not actually present in source code,
	// but can be emitted by token hooks.
	// E.g. fake $begin and $end can be emitted when source code indentation level changes.
	ExternalToken

	// SideToken is a token type that does not affect syntax and thus
	// not fed to parser normally, but can be hooked. E.g. space or comment.
	SideToken

	// ErrorToken is a token type that represents some lexical error, e.g. unmatched opening quote.
	// This token automatically generates an error message containing captured text.
	ErrorToken

	// CaselessToken text consists of case-insensitive symbols.
	// Parser converts its text to uppercase before comparing it with a literal.
	CaselessToken

	// ReservedToken marks literal that represents a reserved word.
	ReservedToken

	// NoLiteralsToken marks token type that cannot match any literal, e.g. raw text in HTML.
	NoLiteralsToken
)

// LayerCommand is a configuration command for hook layer initializer.
type LayerCommand struct {
	// Command is command name.
	Command string
	// Arguments is a list of command arguments. May be empty.
	Arguments []string `json:",omitempty"`
}

// Layer is a configuration for built-in token and/or node hook layer.
type Layer struct {
	// Type is a type name for hook layer.
	Type string
	// Commands is a list of configuration commands applied in order by hook layer initializer.
	// May be empty.
	Commands []LayerCommand `json:",omitempty"`
}

// Node contains information about some syntax tree node.
type Node struct {
	// Name of node.
	Name string

	// FirstState is an index of initial state for this node.
	FirstState int
}

const (
	// AnyToken stored in Rule.Token matches any token except EoF.
	// Used for fallback rules to skip optional or repeated parts of nodes.
	AnyToken = -1

	// FinalState stored in Rule.State means that current node must be finalized and dropped.
	FinalState = -1

	// SameNode stored in Rule.Node means that no nested node is pushed at this point.
	SameNode = -1
)

// Rule contains a grammar rule for some set of tokens at some parsing state.
type Rule struct {
	// Token is either an index in Grammar.Tokens slice or AnyToken.
	Token int

	// State is the index of next state or FinalState if this rule is the final one for node.
	State int

	// Node contains either SameNode or the index of nested node to push.
	Node int
}

// MultiRule defines a list of ambiguous rules for some set of tokens at some parsing state.
type MultiRule struct {
	// Token is an index in Grammar.Tokens slice.
	Token int

	// LowRule is the low index of the rule sub-slice for this token type.
	LowRule int

	// HighRule is the high index of the rule sub-slice for this token type.
	HighRule int
}

// State represents a parsing state.
type State struct {
	// TokenTypes is the set of all token types acceptable at this point.
	TokenTypes BitSet `json:",omitempty"`

	// LowMultiRule is the low index of the multi-rule sub-slice for this state.
	// 0 if not used.
	LowMultiRule int `json:",omitempty"`

	// HighMultiRule is the high index of the multi-rule sub-slice for this state.
	// 0 if not used.
	HighMultiRule int `json:",omitempty"`

	// LowRule is the low index of the rule sub-slice for this state.
	// 0 if not used.
	LowRule int `json:",omitempty"`

	// HighRule is the high index of the rule sub-slice for this state.
	// 0 if not used.
	HighRule int `json:",omitempty"`
}

// Grammar holds all information required to make a parser.
type Grammar struct {
	// Tokens is a list of tokens defined in grammar.
	// First go defined token types, then external tokens, and the last are literals.
	Tokens []Token

	// Layers is a list of hook layers applied to fetched tokens and generated nodes, top layer first.
	Layers []Layer

	// Nodes is a list of defined nodes.
	Nodes []Node

	// States is a list of all parsing states for all nodes, grouped by node.
	States []State

	// MultiRules is a list of all ambiguous rule entries for all states.
	// Grouped by state, entries in a group are sorted by Token field.
	MultiRules []MultiRule

	// Rules is a list of all parsing rules for all states.
	// Grouped by state, entries in a group are sorted by Token field.
	Rules []Rule
}
