package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ava12/llx/examples/conf-edit/internal"
)

type commandList struct {
	key, usage string
	handler    func (*internal.Conf, string)
	validator  func (string) bool
	commands   []string
}

func main () {
	commandLists := []*commandList{
		{key: "r", usage: "remove section", handler: removeSection, validator: internal.IsValidName},
		{key: "a", usage: "add empty section", handler: addSection, validator: internal.IsValidName},
		{key: "c", usage: "remove value", handler: removeValue, validator: internal.IsValidName},
		{key: "s", usage: "set value", handler: setValue, validator: isValidNameValue},
	}
	needHelp := false

	for _, cl := range commandLists {
		flag.Var(cl, cl.key, cl.usage)
	}
	flag.BoolVar(&needHelp, "h", false, "show this help")
	flag.Parse()
	args := flag.Args()

	if needHelp || len(args) != 1 {
		printHelp()
	}

	conf, e := internal.ParseFile(args[0])
	if e == nil {
		for _, cl := range commandLists {
			cl.Run(conf)
		}
		if conf.Updated() {
			_, e = internal.SaveFile(args[0], conf.RootNode)
		}
	}
	if e != nil {
		fmt.Printf("error: %s\n", e.Error())
		os.Exit(2)
	}
}

func printHelp () {
	fmt.Println("Usage is  conf-edit {-r name} {-a name} {-c name} {-s name[=[value]]} conf_file")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("All commands are grouped by type and executed in order:")
	fmt.Println("  remove sections, add sections, remove values, set values.")
	os.Exit(1)
}

func removeSection (conf *internal.Conf, cmd string) {
	conf.RemoveSection(cmd)
}

func addSection (conf *internal.Conf, cmd string) {
	conf.AddSection(cmd)
}

func removeValue (conf *internal.Conf, cmd string) {
	conf.RemoveEntry(cmd)
}

func setValue (conf *internal.Conf, cmd string) {
	parts := strings.SplitN(cmd, "=", 2)
	value := ""
	if len(parts) > 1 {
		value = strings.TrimSpace(parts[1])
	}
	conf.SetEntry(parts[0], value)
}

func (cl *commandList) String () string {
	return "?"
}

func (cl *commandList) Set (s string) error {
	if !cl.validator(s) {
		return fmt.Errorf("incorrect parameter: -%s %s", cl.key, s)
	}

	cl.commands = append(cl.commands, s)
	return nil
}

func (cl *commandList) Run (conf *internal.Conf) {
	for _, cmd := range cl.commands {
		cl.handler(conf, cmd)
	}
}

func isValidNameValue (s string) bool {
	parts := strings.SplitN(s, "=", 2)
	return internal.IsValidName(parts[0])
}
