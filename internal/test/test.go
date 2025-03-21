package test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/ava12/llx"
)

func fatalf(t *testing.T, message string, params ...any) {
	if len(params) > 0 {
		message = fmt.Sprintf(message, params...)
	}
	_, thisFile, _, _ := runtime.Caller(0)
	file := thisFile
	line := 0
	for i := 2; file == thisFile; i++ {
		_, file, line, _ = runtime.Caller(i)
	}
	t.Fatalf("%s at %s:%d", message, file, line)
}

func Assert(t *testing.T, cond bool, message string, params ...any) {
	if !cond {
		fatalf(t, message, params...)
	}
}

func Expect(t *testing.T, cond bool, expected, got any) {
	if !cond {
		fatalf(t, "expecting %v, got %v", expected, got)
	}
}

func ExpectBool(t *testing.T, expected, got bool) {
	Expect(t, expected == got, expected, got)
}

func ExpectInt(t *testing.T, expected, got int) {
	Expect(t, expected == got, expected, got)
}

func ExpectErrorCode(t *testing.T, expected int, e error) {
	if e != nil {
		ee, valid := e.(*llx.Error)
		if valid && ee.Code == expected {
			return
		}
	}

	fatalf(t, "expecting error code %d, got %v", expected, e)
}
