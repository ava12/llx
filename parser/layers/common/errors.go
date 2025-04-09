package common

import (
	"github.com/ava12/llx"
)

const (
	UnknownCommandError = llx.LayerErrors + iota
	MissingCommandError
	NumberOfArgumentsError
	InvalidArgumentError
	CommandAlreadyUsedError
	UnknownTokenTypeError
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

func MakeWrongTokenError(layer, token, reason string) *llx.Error {
	return llx.FormatError(WrongTokenError, "wrong %s token for %q layer: %s", token, layer, reason)
}
