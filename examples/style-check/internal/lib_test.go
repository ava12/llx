package internal

import (
	"strings"
	"testing"

	"github.com/ava12/llx/source"
)

func newSrc (content string) *source.Source {
	return source.New("src", []byte(content))
}

func TestCorrectFile (t *testing.T) {
	src := `
/* Correct source file. Should emit no errors. */

int i, j;
float f[3];

typedef struct {
    int   i[1];
    float f;
    bar   s;
} foo;
`
	reports, e := Check(newSrc(src))
	if e != nil {
		t.Fatalf("unexpected error: %s", e.Error())
	}
	if len(reports) > 0 {
		t.Fatalf("unexpected style errors: %v", reports)
	}
}

func checkErrors (t *testing.T, index int, src string, errors []string) {
	r, e := Check(source.New("", []byte(src)))
	if e != nil {
		t.Errorf("sample #%d (%q): unexpected error: %s", index, src, e.Error())
		return
	}

	if len(r) != len(errors) {
		t.Errorf("sample #%d (%q): expecting %d error(s), got %d (%v)", index, src, len(errors), len(r), r)
		return
	}

	for i, err := range errors {
		if !strings.HasPrefix(r[i].Message, err) {
			t.Errorf("sample #%d (%q): expecting: %v, got %v", index, src, errors, r)
			return
		}
	}
}

func TestSingleErrors (t *testing.T) {
	samples := []struct{
		src, err string
	}{
		{"typedef struct {\n    int a;\n    float b;\n} s;\n", ErrAlign},
		{"  int foo;\n  int bar;\n", ErrIndent},
		{"struct {\n    a int;\n} foo;\n", ErrLiteral},
		{"int i;", ErrNoEofNl},
		{"int a,b;\n", ErrNoSpace},
		{"typedef struct {\n    int i;\n}foo;\n", ErrNoSpace},
		{"typedef struct{\n    int i;\n} foo;\n", ErrNoSpace},
		{"int a [2];\n", ErrWrongSpace},
		{"int a[ 2];\n", ErrWrongSpace},
		{"int a[2 ];\n", ErrWrongSpace},
		{"int a[2] ;\n", ErrWrongSpace},
		{"int a ;\n", ErrWrongSpace},
		{"int a , b;\n", ErrWrongSpace},
		{"int  a;\n", ErrSpaces},
		{"int\ta;\n", ErrTab},
		{"int a;\n \n", ErrTrailSpace},
	}

	for i, s := range samples {
		checkErrors(t, i, s.src, []string{s.err})
	}
}

func TestMultipleErrors (t *testing.T) {
	samples := []struct{
		src string
		errors []string
	}{
		{"typedef struct {int a;\nint b;\n} foo;\n", []string{ErrSameLine, ErrIndent, ErrAlign}},
		{"struct {\n    int a;\n    float b;\n} foo;", []string{ErrLiteral, ErrAlign, ErrNoEofNl}},
	}

	for i, s := range samples {
		checkErrors(t, i, s.src, s.errors)
	}
}
