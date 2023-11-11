package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ava12/llx"
	"github.com/ava12/llx/examples/calc/internal"
	"github.com/ava12/llx/parser"
)

func writeError(text string) {
	fmt.Println(" !", text)
}

func writeNumber(x float64) {
	fmt.Printf(" : %.12g\n", x)
}

var reader = bufio.NewReader(os.Stdin)

func scan() (res string, e error) {
	res, e = reader.ReadString('\n')
	res = strings.TrimRight(res, "\r\n")
	return
}

func showHelp() {
	fmt.Print(`
You can:
  - compute an expression and show its result: <expression> (without brackets)
  - set a variable: <var_name> = <expression>
  - set a function: <name> (<arg_name> [, <arg_name> ...]) = <expression>

Expressions may contain numbers, arithmetic operators (+, -, *, /),
^ for exponentiation, brackets, variable (or argument) names, and 
function calls ( <name>([<arg>[,<arg>]]) ). An argument may be any expression.

Unary minus must be the first part of (sub)expression. Inputs like
"- x + y", "x / (-y)", or "x - (-(-y))" are correct, but "x * -y" or
"x ^ (- - y)" are not.

Names start with a letter and can contain letters, digits, and underscores.
Names are case-sensitive, so F and f are two different names.

Variables and functions use separate namespaces, so you can have both
variable X and function X.

`)
}

func main() {
	fmt.Println("A simple line calculator. help for quick help, empty line to exit.")
	fmt.Println()
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

		if strings.Contains(input, "help") {
			appendInput = false
			showHelp()
			continue
		}

		if appendInput {
			input = prevInput + "\n" + input
			appendInput = false
		}

		res, e := internal.Compute(input)
		if e == nil {
			writeNumber(res)
		} else {
			ee, f := e.(*llx.Error)
			if f && ee.Code == parser.UnexpectedEoiError {
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
