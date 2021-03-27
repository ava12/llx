package langdef

import (
	"strings"

	err "github.com/ava12/llx/errors"
	"github.com/ava12/llx/lexer"
)

const (
	UnexpectedEofError = iota + 1
	UnexpectedTokenError
	UnknownTerminalError
	WrongTerminalError
	TerminalDefinedError
	NonterminalDefinedError
	WrongRegexpError
	UnknownNonterminalError
	UnusedNonterminalError
	UnresolvedError
	RecursionError
	GroupNumberError
	UnresolvedGroupsError
	DisjointGroupsError
	UndefinedTerminalError
)

func eofError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, UnexpectedEofError, "unexpected EoF")
}

func tokenError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, UnexpectedTokenError, "unexpected %s token", token.TypeName())
}

func termError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, UnknownTerminalError, "unknown terminal %q ", token.Text())
}

func wrongTermError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, WrongTerminalError, "cannot use terminal %q in definitions", token.Text())
}

func defTermError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, TerminalDefinedError, "terminal %q already defined", token.Text())
}

func defNontermError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, NonterminalDefinedError, "nonterminal %q already defined", token.Text())
}

func regexpError (token *lexer.Token, e error) *err.Error {
	return err.FormatPos(token, WrongRegexpError, "incorrect RegExp %s (%s)", token.Text(), e.Error())
}

func unknownNontermError (names []string) *err.Error {
	return err.Format(UnknownNonterminalError, "undefined nonterminals: " + strings.Join(names, ", "))
}

func unusedNontermError (names []string) *err.Error {
	return err.Format(UnusedNonterminalError, "unused nonterminals: " + strings.Join(names, ", "))
}

func unresolvedError (names []string) *err.Error {
	return err.Format(UnresolvedError, "cannot resolve dependencies for nonterminals: " + strings.Join(names, ", "))
}

func recursionError (names []string) *err.Error {
	return err.Format(RecursionError, "found left-recursive nonterminals: " + strings.Join(names, ", "))
}

func groupNumberError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, GroupNumberError, "too many terminal groups")
}

func unresolvedGroupsError (text string) *err.Error {
	return err.Format(UnresolvedGroupsError, "cannot detect terminal groups for %q literal", text)
}

func disjointGroupsError (nonterm string, state int, term string) *err.Error {
	return err.Format(DisjointGroupsError, "disjoint terminal groups for %q nonterminal, state %d, term %q", nonterm, state, term)
}

func undefinedTermError (name string) *err.Error {
	return err.Format(UndefinedTerminalError, "terminal %q mentioned but not defined", name)
}
