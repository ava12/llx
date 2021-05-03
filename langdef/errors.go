package langdef

import (
	"strings"

	"github.com/ava12/llx"
	"github.com/ava12/llx/lexer"
)

const (
	UnexpectedEofError = iota + 1
	UnexpectedTokenError
	UnknownTokenError
	WrongTokenError
	TokenDefinedError
	NonTerminalDefinedError
	WrongRegexpError
	UnknownNonTerminalError
	UnusedNonTerminalError
	UnresolvedError
	RecursionError
	GroupNumberError
	UnresolvedGroupsError
	DisjointGroupsError
	UndefinedTokenError
)

func eofError (token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, UnexpectedEofError, "unexpected EoF")
}

func unexpectedTokenError (token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, UnexpectedTokenError, "unexpected %s token", token.TypeName())
}

func tokenError (token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, UnknownTokenError, "unknown token %q ", token.Text())
}

func wrongTokenError (token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, WrongTokenError, "cannot use token %q in definitions", token.Text())
}

func defTokenError (token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, TokenDefinedError, "token %q already defined", token.Text())
}

func defNonTermError (token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, NonTerminalDefinedError, "non-terminal %q already defined", token.Text())
}

func regexpError (token *lexer.Token, e error) *llx.Error {
	return llx.FormatErrorPos(token, WrongRegexpError, "incorrect RegExp %s (%s)", token.Text(), e.Error())
}

func unknownNonTermError (names []string) *llx.Error {
	return llx.FormatError(UnknownNonTerminalError, "undefined non-terminals: " + strings.Join(names, ", "))
}

func unusedNonTermError (names []string) *llx.Error {
	return llx.FormatError(UnusedNonTerminalError, "unused non-terminals: " + strings.Join(names, ", "))
}

func unresolvedError (names []string) *llx.Error {
	return llx.FormatError(UnresolvedError, "cannot resolve dependencies for non-terminals: " + strings.Join(names, ", "))
}

func recursionError (names []string) *llx.Error {
	return llx.FormatError(RecursionError, "found left-recursive non-terminals: " + strings.Join(names, ", "))
}

func groupNumberError (token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, GroupNumberError, "too many token groups")
}

func unresolvedGroupsError (text string) *llx.Error {
	return llx.FormatError(UnresolvedGroupsError, "cannot detect token groups for %q literal", text)
}

func disjointGroupsError (nonTerm string, state int, token string) *llx.Error {
	return llx.FormatError(DisjointGroupsError, "disjoint token groups for %q non-terminal, state %d, token %q", nonTerm, state, token)
}

func undefinedTokenError (name string) *llx.Error {
	return llx.FormatError(UndefinedTokenError, "token %q mentioned but not defined", name)
}
