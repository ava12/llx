package langdef

import (
	"strings"

	err "github.com/ava12/llx/errors"
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

func eofError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, UnexpectedEofError, "unexpected EoF")
}

func unexpectedTokenError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, UnexpectedTokenError, "unexpected %s token", token.TypeName())
}

func tokenError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, UnknownTokenError, "unknown token %q ", token.Text())
}

func wrongTokenError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, WrongTokenError, "cannot use token %q in definitions", token.Text())
}

func defTokenError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, TokenDefinedError, "token %q already defined", token.Text())
}

func defNonTermError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, NonTerminalDefinedError, "non-terminal %q already defined", token.Text())
}

func regexpError (token *lexer.Token, e error) *err.Error {
	return err.FormatPos(token, WrongRegexpError, "incorrect RegExp %s (%s)", token.Text(), e.Error())
}

func unknownNonTermError (names []string) *err.Error {
	return err.Format(UnknownNonTerminalError, "undefined non-terminals: " + strings.Join(names, ", "))
}

func unusedNonTermError (names []string) *err.Error {
	return err.Format(UnusedNonTerminalError, "unused non-terminals: " + strings.Join(names, ", "))
}

func unresolvedError (names []string) *err.Error {
	return err.Format(UnresolvedError, "cannot resolve dependencies for non-terminals: " + strings.Join(names, ", "))
}

func recursionError (names []string) *err.Error {
	return err.Format(RecursionError, "found left-recursive non-terminals: " + strings.Join(names, ", "))
}

func groupNumberError (token *lexer.Token) *err.Error {
	return err.FormatPos(token, GroupNumberError, "too many token groups")
}

func unresolvedGroupsError (text string) *err.Error {
	return err.Format(UnresolvedGroupsError, "cannot detect token groups for %q literal", text)
}

func disjointGroupsError (nonTerm string, state int, token string) *err.Error {
	return err.Format(DisjointGroupsError, "disjoint token groups for %q non-terminal, state %d, token %q", nonTerm, state, token)
}

func undefinedTokenError (name string) *err.Error {
	return err.Format(UndefinedTokenError, "token %q mentioned but not defined", name)
}
