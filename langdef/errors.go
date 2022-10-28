package langdef

import (
	"strings"

	"github.com/ava12/llx"
	"github.com/ava12/llx/lexer"
)

// Error codes used by langdef.Parse* functions:
const (
	// EoF reached when token expected
	UnexpectedEofError = iota + 1

	// fetched token of unexpected type or with unexpected content
	UnexpectedTokenError

	// non-terminal definition uses undefined token type
	UnknownTokenError

	// non-terminal definition uses aside or error token
	WrongTokenError

	// redefining already defined token type
	TokenDefinedError

	// redefining already defined non-terminal
	NonTerminalDefinedError

	// error in regular expression
	WrongRegexpError

	// non-terminal definition uses non-terminal that was never defined
	UnknownNonTerminalError

	// found non-terminal that is defined but not used
	UnusedNonTerminalError

	// cannot resolve non-terminal dependencies, this maybe a circular cross-reference (e. g. foo = bar; bar = foo;)
	UnresolvedError

	// left-recursive non-terminal found
	RecursionError

	// too many token groups (more than 30)
	GroupNumberError

	// cannot associate string literal with any token type
	UnresolvedGroupsError

	// tokens expected at certain parsing state do not belong to the same token group
	DisjointGroupsError

	// token type listed in directive is not defined
	UndefinedTokenError

	// assigning aside token to some group
	AsideGroupError

	// non-terminal definition uses string literal that is not whitelisted
	UnknownLiteralError
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

func asideGroupError (name string) *llx.Error {
	return llx.FormatError(AsideGroupError, "cannot assign %q token to a separate group: aside tokens belong to all groups", name)
}

func unknownLiteralError (text string) *llx.Error {
	return llx.FormatError(UnknownLiteralError, "cannot use %q literal: it is not whitelisted", text)
}
