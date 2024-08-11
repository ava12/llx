package langdef

import (
	"strings"

	"github.com/ava12/llx"
	"github.com/ava12/llx/lexer"
)

// Error codes used by langdef.Parse* functions:
const (
	// EoF reached when token expected
	UnexpectedEofError = llx.LangDefErrors + iota
	// fetched token of unexpected type or with unexpected content
	UnexpectedTokenError
	// node definition uses undefined token type
	UnknownTokenError
	// node definition uses aside or error token
	WrongTokenError
	// redefining already defined token type
	TokenDefinedError
	// redefining already defined node
	NodeDefinedError
	// error in regular expression
	WrongRegexpError
	// node definition uses node that was never defined
	UnknownNodeError
	// found node that is defined but not used
	UnusedNodeError
	// cannot resolve node dependencies, this maybe a circular cross-reference (e. g. foo = bar; bar = foo;)
	UnresolvedError
	// left-recursive node definition found
	RecursionError
	// too many token types
	TokenTypeNumberError
	// cannot associate string literal with any token type
	UnresolvedTokenTypesError
	// token type listed in directive is not defined
	UndefinedTokenError
	// node definition uses string literal that is not whitelisted
	UnknownLiteralError
	// trying to move token to new group more than once
	ReassignedGroupError
)

func eofError(token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, UnexpectedEofError, "unexpected EoF")
}

func unexpectedTokenError(token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, UnexpectedTokenError, "unexpected %s token", token.TypeName())
}

func tokenError(token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, UnknownTokenError, "unknown token %q ", token.Text())
}

func wrongTokenError(token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, WrongTokenError, "cannot use token %q in definitions", token.Text())
}

func defTokenError(token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, TokenDefinedError, "token %q already defined", token.Text())
}

func defNodeError(token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, NodeDefinedError, "node %q already defined", token.Text())
}

func regexpError(token *lexer.Token, e error) *llx.Error {
	return llx.FormatErrorPos(token, WrongRegexpError, "incorrect RegExp %s (%s)", token.Text(), e.Error())
}

func unknownNodeError(names []string) *llx.Error {
	return llx.FormatError(UnknownNodeError, "undefined nodes: "+strings.Join(names, ", "))
}

func unusedNodeError(names []string) *llx.Error {
	return llx.FormatError(UnusedNodeError, "unused nodes: "+strings.Join(names, ", "))
}

func unresolvedError(names []string) *llx.Error {
	return llx.FormatError(UnresolvedError, "cannot resolve dependencies for nodes: "+strings.Join(names, ", "))
}

func recursionError(names []string) *llx.Error {
	return llx.FormatError(RecursionError, "found left-recursive nodes: "+strings.Join(names, ", "))
}

func tokenTypeNumberError(token *lexer.Token) *llx.Error {
	return llx.FormatErrorPos(token, TokenTypeNumberError, "too many token types")
}

func unresolvedTokenTypesError(text string) *llx.Error {
	return llx.FormatError(UnresolvedTokenTypesError, "cannot detect token groups for %q literal", text)
}

func undefinedTokenError(name string) *llx.Error {
	return llx.FormatError(UndefinedTokenError, "token %q mentioned but not defined", name)
}

func unknownLiteralError(text string) *llx.Error {
	return llx.FormatError(UnknownLiteralError, "cannot use %q literal: it is not whitelisted", text)
}

func reassignedGroupError(name string) *llx.Error {
	return llx.FormatError(ReassignedGroupError, "cannot move %q token to another group again", name)
}
