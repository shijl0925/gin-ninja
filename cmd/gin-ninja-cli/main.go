package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	os.Exit(runWithInput(os.Stdin, os.Stdout, os.Stderr, os.Args[1:]))
}

func run(stdout, stderr io.Writer, args []string) int {
	return runWithInput(strings.NewReader(""), stdout, stderr, args)
}

func runWithInput(stdin io.Reader, stdout, stderr io.Writer, args []string) int {
	if len(args) == 0 {
		printRootUsage(stderr)
		return 2
	}

	switch args[0] {
	case "generate":
		return runGenerate(stdout, stderr, args[1:])
	case "startproject":
		return runStartProject(stdout, stderr, args[1:])
	case "startapp":
		return runStartApp(stdout, stderr, args[1:])
	case "init":
		return runInit(stdin, stdout, stderr, args[1:])
	case "makemigrations":
		return runMakeMigrations(stdout, stderr, args[1:])
	case "migrate":
		return runMigrate(stdout, stderr, args[1:])
	case "showmigrations":
		return runShowMigrations(stdout, stderr, args[1:])
	case "sqlmigrate":
		return runSQLMigrate(stdout, stderr, args[1:])
	case "help":
		return runHelp(stdout, stderr, args[1:])
	case "-h", "--help":
		printRootUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printRootUsage(stderr)
		return 2
	}
}
