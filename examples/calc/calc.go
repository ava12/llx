package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/ava12/llx/errors"
	"github.com/ava12/llx/examples/calc/lib"
	"github.com/ava12/llx/parser"
)

func writeError (text string) {
	fmt.Println(" !", text)
}

func writeNumber (x float64) {
	fmt.Printf(" : %.12g\n", x)
}

var reader = bufio.NewReader(os.Stdin)

func scan () (res string, e error) {
	res, e = reader.ReadString('\n')
	l := len(res)
	if l > 0 && res[l - 1] == '\n' {
		res = res[: l - 1]
	}
	return
}

func main () {
	prevInput := ""
	appendInput := false

	for {
		var input string
		if appendInput {
			fmt.Print("-> ")
		} else {
			fmt.Print(">> ")
		}
		input, e := scan()
		if input == "" || e != nil {
			break
		}

		if appendInput {
			input = prevInput + " " + input
			appendInput = false
		}

		res, e := lib.Compute(input)
		if e == nil {
			writeNumber(res)
		} else {
			ee, f := e.(*errors.Error)
			if f && ee.Code == parser.UnexpectedEofError {
				appendInput = true
				prevInput = input
			} else {
				writeError(e.Error())
			}
			continue
		}
	}

	fmt.Println()
}
