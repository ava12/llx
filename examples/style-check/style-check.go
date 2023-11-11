package main

import (
	"fmt"
	"os"

	"github.com/ava12/llx/examples/style-check/internal"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage is  style-check <file>")
		os.Exit(1)
	}

	reports, e := internal.CheckFile(os.Args[1])
	if e != nil {
		fmt.Printf("error: %s\n", e)
		os.Exit(1)
	}

	if len(reports) == 0 {
		os.Exit(0)
	}

	fmt.Println("style errors:")
	for _, r := range reports {
		fmt.Printf("  %d:%d: %s\n", r.Line, r.Col, r.Message)
	}
	os.Exit(2)
}
