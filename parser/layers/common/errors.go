/*
Package common contains definitions used by all standard hook layers.
*/
package common

import (
	"github.com/ava12/llx"
	"github.com/ava12/llx/parser"
)

// Error codes emitted by standard hook layers.
const (
	// a layer definition contains unknown command
	UnknownCommandError = llx.LayerErrors + iota
	// a layer definition is missing required command
	MissingCommandError
	// a command has too few or too many arguments
	NumberOfArgumentsError
	// a command has incorrect argument
	InvalidArgumentError
	// repeating a command that must not be repeated
	CommandAlreadyUsedError
	// a command argument doesn't match any known token type
	UnknownTokenTypeError
	// incoming token triggered an error
	WrongTokenError
)

func MakeUnknownCommandError(layer, command string) *llx.Error {
	return llx.FormatError(UnknownCommandError, "unknown command %q for %q layer", command, layer)
}

func MakeMissingCommandError(layer, command string) *llx.Error {
	return llx.FormatError(MissingCommandError, "missing required command %q for %q layer", command, layer)
}

func MakeNumberOfArgumentsError(layer, command string, expected, got int) *llx.Error {
	return llx.FormatError(NumberOfArgumentsError,
		"wrong number of arguments for %q command for %q layer: expecting %d, got %d", command, expected, got)
}

func MakeInvalidArgumentError(layer, command, arg, reason string) *llx.Error {
	return llx.FormatError(InvalidArgumentError, "invalid %q argument for %q command for %q layer: %s",
		arg, command, layer, reason)
}

func MakeCommandAlreadyUsedError(layer, command string) *llx.Error {
	return llx.FormatError(CommandAlreadyUsedError, "%q command for %q layer already used", command, layer)
}

func MakeUnknownTokenTypeError(layer, command, typeName string) *llx.Error {
	return llx.FormatError(UnknownTokenTypeError, "unknown token type %q for %q command for %q layer",
		typeName, command, layer)
}

func MakeWrongTokenError(layer string, token *parser.Token, reason string) *llx.Error {
	text := token.Text()
	if len(text) > 10 {
		text = text[:7] + "..."
	}

	return llx.FormatErrorPos(token, WrongTokenError, "wrong %s (%q) token for %q layer: %s",
		token.TypeName(), text, layer, reason)
}
