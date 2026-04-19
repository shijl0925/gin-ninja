package main

import (
	"fmt"
	"io"
	"strings"
)

func runHelp(stdout, stderr io.Writer, args []string) int {
	switch len(args) {
	case 0:
		printRootUsage(stdout)
		return 0
	case 1:
		switch args[0] {
		case "startproject":
			printStartProjectUsage(stdout)
			return 0
		case "startapp":
			printStartAppUsage(stdout)
			return 0
		case "init":
			printInitUsage(stdout)
			return 0
		case "generate":
			printGenerateUsage(stdout)
			return 0
		case "makemigrations":
			printMakeMigrationsUsage(stdout)
			return 0
		case "migrate":
			printMigrateUsage(stdout)
			return 0
		case "showmigrations":
			printShowMigrationsUsage(stdout)
			return 0
		case "sqlmigrate":
			printSQLMigrateUsage(stdout)
			return 0
		}
	case 2:
		if args[0] == "generate" && args[1] == "crud" {
			printGenerateUsage(stdout)
			return 0
		}
	}

	fmt.Fprintf(stderr, "unknown help topic %q\n\n", strings.Join(args, " "))
	printRootUsage(stderr)
	return 2
}

func printRootUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gin-ninja-cli <command> [arguments]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Scaffold commands:")
	fmt.Fprintln(w, "  startproject   Create a new gin-ninja project scaffold")
	fmt.Fprintln(w, "  startapp       Create a new app scaffold inside a project")
	fmt.Fprintln(w, "  init           Run an interactive scaffold wizard")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Migration commands:")
	fmt.Fprintln(w, "  makemigrations Generate a SQL migration from MigrationModels()")
	fmt.Fprintln(w, "  migrate        Apply or roll back migrations")
	fmt.Fprintln(w, "  showmigrations Show migration status")
	fmt.Fprintln(w, "  sqlmigrate     Print SQL for a migration file")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Code generation:")
	fmt.Fprintln(w, "  generate crud  Generate CRUD handlers from a model struct")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Recommended paths:")
	fmt.Fprintln(w, "  gin-ninja-cli startproject mysite -module github.com/acme/mysite")
	fmt.Fprintln(w, "  gin-ninja-cli startproject mysite -template standard")
	fmt.Fprintln(w, "  gin-ninja-cli startproject mysite -template admin")
	fmt.Fprintln(w, "  gin-ninja-cli init")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'gin-ninja-cli help <command>' for command details.")
}

func printGenerateUsage(w io.Writer) {
	fmt.Fprintln(w, "Generate CRUD scaffold code from an existing model struct.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gin-ninja-cli generate crud -model <Name> -model-file <path> [-output <path>] [-package <name>] [-tag <name>] [-with-gormx]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  gin-ninja-cli generate crud -model User -model-file ./app/models.go")
	fmt.Fprintln(w, "  gin-ninja-cli generate crud -model Account -model-file ./app/models.go -with-gormx")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	printFlagGroup(w, []flagHelp{
		{name: "-model <Name>", usage: "Go struct name to scaffold"},
		{name: "-model-file <path>", usage: "Go source file containing the model struct"},
		{name: "-output <path>", usage: "Output file path (defaults next to the model file)"},
		{name: "-package <name>", usage: "Override the generated package name"},
		{name: "-tag <name>", usage: "Override the generated router tag name"},
		{name: "-with-gormx", usage: "Generate gormx-based CRUD code (default: false)"},
	})
}

func printMakeMigrationsUsage(w io.Writer) {
	fmt.Fprintln(w, "Generate a timestamped SQL migration from MigrationModels().")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gin-ninja-cli makemigrations [-config <path>] [-app-dir <path>] [-migrations-dir <path>] [-name <name>]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	printFlagGroup(w, []flagHelp{
		{name: "-config <path>", usage: "Project config file path (default: config.yaml)"},
		{name: "-app-dir <path>", usage: "Relative app package directory containing MigrationModels()"},
		{name: "-migrations-dir <path>", usage: "Relative migrations directory (default: migrations)"},
		{name: "-name <name>", usage: "Optional migration name suffix"},
	})
}

func printMigrateUsage(w io.Writer) {
	fmt.Fprintln(w, "Apply pending migrations or roll back to a target version.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gin-ninja-cli migrate [target|zero] [-config <path>] [-migrations-dir <path>]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	printFlagGroup(w, []flagHelp{
		{name: "-config <path>", usage: "Project config file path (default: config.yaml)"},
		{name: "-migrations-dir <path>", usage: "Relative migrations directory (default: migrations)"},
	})
}

func printShowMigrationsUsage(w io.Writer) {
	fmt.Fprintln(w, "List migration files and whether each one has been applied.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gin-ninja-cli showmigrations [-config <path>] [-migrations-dir <path>]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	printFlagGroup(w, []flagHelp{
		{name: "-config <path>", usage: "Project config file path (default: config.yaml)"},
		{name: "-migrations-dir <path>", usage: "Relative migrations directory (default: migrations)"},
	})
}

func printSQLMigrateUsage(w io.Writer) {
	fmt.Fprintln(w, "Print SQL for an existing migration file.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gin-ninja-cli sqlmigrate <migration> [-config <path>] [-migrations-dir <path>] [-direction <up|down|all>]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	printFlagGroup(w, []flagHelp{
		{name: "-config <path>", usage: "Project config file path (default: config.yaml)"},
		{name: "-migrations-dir <path>", usage: "Relative migrations directory (default: migrations)"},
		{name: "-direction <up|down|all>", usage: "Choose which SQL section to print (default: all)"},
	})
}

type flagHelp struct {
	name  string
	usage string
}

func printFlagGroup(w io.Writer, flags []flagHelp) {
	for _, item := range flags {
		fmt.Fprintf(w, "  %-28s %s\n", item.name, item.usage)
	}
}
