/*
Package grammar contains data types used by LL(*) parser.

Grammar is represented in the form of nondeterministic state machine.
The machine uses stack holding currently processed non-terminals (initially the root one)
and saved states. Each non-terminal has its own subset of states.

State contains map of transition rules (or ambiguous variants of rules).
A key is either a token type id, a literal token id, or AnyToken value.
When parser tries to match a rule for current token the priorities are:
  - a literal token id;
  - a token type id (unless the token text matches some reserved literal);
  - AnyToken.

Transition rule contains the next state index (or FinalState value
if parsing of current non-terminal is finished) and nested non-terminal id
(or SameNonTerm value if no non-terminal is pushed).

Each time parser enters FinalState it pops non-terminal and saved state id.

When a rule is applied:
  - if next non-terminal is SameNonTerm:
    - current state is set to next state;
    - if matched rule key is not AnyToken: next token is read;
  - else:
    - next non-terminal and next state id are pushed;
    - current state is set to the initial state of next non-terminal;

*/
package grammar

const (
	// RootNonTerm is the index of root non-terminal in Grammar.NonTerms.
	RootNonTerm  = 0
)

// BitSet is a general bit set, where i-th bit represents item with index i.
type BitSet = int

// TokenFlags contain information about token type.
type TokenFlags int

// Token contains information about some token.
type Token struct {
	// Name is either a token type name (no leading "$") or exact text for a literal token.
	Name   string

	// Re is RE2 expression defining this token type. Must not contain capturing groups.
	// Empty string for literal and external tokens.
	Re     string

	// Groups is a bit set of all groups this token belongs to.
	// First defined group has index 0, "default" group is the last one.
	// For literal tokens this is a union of all Groups of all suitable token types.
	Groups BitSet

	// Flags contain information about token type.
	Flags TokenFlags
}

const (
	// LiteralToken matches exact text of some token.
	LiteralToken TokenFlags = 1 << iota

	// ExternalToken is a token type that is not actually present in source code,
	// but can be emitted by token hooks.
	// E. g. fake $begin and $end can be emitted when source code indentation level changes.
	ExternalToken

	// AsideToken is a token type that does not affect syntax and thus
	// not fed to parser normally, but can be hooked. E. g. space or comment.
	AsideToken

	// ErrorToken is a token type that represents some lexical error, e. g. unmatched opening quote.
	// This token automatically generates an error message containing captured text.
	ErrorToken

	// ShrinkableToken is a token that can be split in smaller parts if there is no suitable rule.
	ShrinkableToken

	// CaselessToken text consists of case-insensitive symbols.
	// Parser converts its text to uppercase before comparing it with a literal.
	CaselessToken

	// ReservedToken marks literal tokens that represent reserved words.
	ReservedToken
)

// NonTerm contains information about some non-terminal.
type NonTerm struct {
	// Name of non-terminal.
	Name       string

	// FirstState is an index of initial state for this non-terminal.
	FirstState int
}

const (
	// AnyToken stored in Rule.Token matches any token except EoF.
	// Used for fallback rules to skip optional or repeated parts of non-terminals.
	AnyToken    = -1

	// FinalState stored in Rule.State means that current non-terminal must be finalized and popped.
	FinalState  = -1

	// SameNonTerm stored in Rule.NonTerm means that no nested non-terminal is pushed at this point.
	SameNonTerm = -1
)

// Rule contains a grammar rule for some set of tokens at some parsing state.
type Rule struct {
	// Token is either an index in Grammar.Tokens slice or AnyToken.
	Token   int

	// State is the index of next state or FinalState if this rule is the final one for non-terminal.
	State   int

	// NonTerm contains either SameNonTerm or the index of nested non-terminal to push.
	NonTerm int
}

type MultiRule struct {
	// Token is an index in Grammar.Tokens slice.
	Token     int

	// LowRule is the low index of the rule sub-slice for this token type.
	LowRule   int

	// HighRule is the high index of the rule sub-slice for this token type.
	HighRule  int
}

// State represents a parsing state for some nonTerminal.
type State struct {
	// Group is 0-based index of token group shared by all tokens
	// that are acceptable at this point.
	Group         int `json:",omitempty"`

	// LowMultiRule is the low index of the multi-rule sub-slice for this token type.
	// 0 if not used.
	LowMultiRule  int `json:",omitempty"`

	// HighMultiRule is the high index of the multi-rule sub-slice for this token type.
	// 0 if not used.
	HighMultiRule int `json:",omitempty"`

	// LowRule is the low index of the rule sub-slice for this token type.
	// 0 if not used.
	LowRule       int `json:",omitempty"`

	// HighRule is the high index of the rule sub-slice for this token type.
	// 0 if not used.
	HighRule      int `json:",omitempty"`
}

// Grammar holds all information required to make a parser.
type Grammar struct {
	// Tokens is a list of tokens defined in grammar.
	// First go defined token types, then external tokens, and the last are literals.
	Tokens     []Token

	// NonTerms is a list of non-terminals.
	NonTerms   []NonTerm

	// States is a list of all parsing states for all non-terminals, grouped by non-terminal.
	States     []State

	// MultiRules is a list of all ambiguous rule entries for all states.
	// Grouped by state, entries in a group are sorted by Token field.
	MultiRules []MultiRule

	// Rules is a list of all parsing rules for all states.
	// Grouped by state, entries in a group are sorted by Token field.
	Rules      []Rule
}
