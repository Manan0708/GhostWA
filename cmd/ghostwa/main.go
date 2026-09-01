// Package main is the program entry point.
//
// It does almost nothing: collect arguments, call the CLI, then exit
// with the code the CLI returned.
package main

import (
	"os"

	"github.com/Manan0708/GhostWA/internal/cli"
)

func main() {
	// os.Args[0] is the program path (e.g. "wacli.exe").
	// os.Args[1:] is the slice of arguments the user typed after the command.
	//
	// os.Exit ends the process with an exit code. Unlike returning from main
	// (always treated as 0), this lets us report failure to the shell.
	code := cli.Run(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
