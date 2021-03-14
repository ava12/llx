package langdef

import (
	"strings"

	err "llx/errors"
	"llx/lexer"
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
	RedefineGroupError
	WrongGroupError
	UnresolvedGroupError
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

func redefineGroupError (token *lexer.Token, name string) *err.Error {
	return err.FormatPos(token, RedefineGroupError, "cannot redefine group for %s nonterminal", name)
}

func wrongGroupError (expected, got int, nonterm, term string) *err.Error {
	msg := "wrong group for %q nonterminal: expecting %d, but %s terminal belongs to %d"
	return err.Format(WrongGroupError, msg, nonterm, expected, term, got)
}

func unresolvedGroupError (nonterm string) *err.Error {
	return err.Format(UnresolvedGroupError, "cannot determine term group for %q nonterminal", nonterm)
}
