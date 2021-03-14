package errors

import (
	"fmt"
)

type Error struct {
	Code int
	Message string
	SourceName string
	Line, Col int
}

type SourcePos interface {
	SourceName () string
	Line () int
	Col () int
}

func New (code int, msg, name string, line, col int) *Error {
	if name != "" && line != 0 && col != 0 {
		msg += fmt.Sprintf(" in %s at line %d col %d", name, line, col)
	}
	return &Error{code, msg, name, line, col}
}

func (e *Error) Error () string {
	return e.Message
}

func Format (code int, msg string, params ...interface{}) *Error {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return New(code, msg, "", 0, 0)
}

func FormatPos (pos SourcePos, code int, msg string, params ...interface{}) *Error {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return New(code, msg, pos.SourceName(), pos.Line(), pos.Col())
}
