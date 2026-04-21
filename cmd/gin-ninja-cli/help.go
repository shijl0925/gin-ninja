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
	p := newHelpPrinter(w)
	fmt.Fprintln(w, p.title("gin-ninja-cli"))
	fmt.Fprintln(w, "Build projects, apps, CRUD scaffolds, and migrations for gin-ninja.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Usage"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli <command> [arguments]"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Start here"))
	printHelpItems(w, p.command, []helpItem{
		{name: "New project", usage: "gin-ninja-cli startproject mysite -module github.com/acme/mysite"},
		{name: "Add app", usage: "gin-ninja-cli startapp blog"},
		{name: "Generate CRUD", usage: "gin-ninja-cli generate crud -model User -model-file ./app/models.go"},
		{name: "Interactive", usage: "gin-ninja-cli init"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Decision guide"))
	printHelpItems(w, p.command, []helpItem{
		{name: "startproject", usage: "Create a new project scaffold; recommended first step for a new service"},
		{name: "startapp", usage: "Add a new app inside an existing gin-ninja project"},
		{name: "generate crud", usage: "Generate CRUD handlers from an existing model struct"},
		{name: "init", usage: "Use the interactive wizard when you want prompts instead of flags"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Other commands"))
	printHelpItems(w, p.command, []helpItem{
		{name: "makemigrations", usage: "Generate a SQL migration from MigrationModels()"},
		{name: "migrate", usage: "Apply or roll back migrations"},
		{name: "showmigrations", usage: "Show migration status"},
		{name: "sqlmigrate", usage: "Print SQL for a migration file"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.note("Run 'gin-ninja-cli help <command>' for command details."))
}

func printGenerateUsage(w io.Writer) {
	p := newHelpPrinter(w)
	fmt.Fprintln(w, p.title("generate crud"))
	fmt.Fprintln(w, "Generate CRUD scaffold code from an existing model struct.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Best for"))
	fmt.Fprintln(w, "  Existing projects that already have a model and want typed CRUD handlers quickly.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Usage"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli generate crud -model <Name> -model-file <path> [-output <path>] [-package <name>] [-tag <name>] [-with-gormx]"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Recommended flow"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli generate crud -model User -model-file ./app/models.go"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli generate crud -model Account -model-file ./app/models.go -with-gormx"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Key options"))
	printFlagGroup(w, []flagHelp{
		{name: "-model <Name>", usage: "Go struct name to scaffold"},
		{name: "-model-file <path>", usage: "Go source file containing the model struct"},
		{name: "-output <path>", usage: "Output file path (defaults next to the model file)"},
		{name: "-package <name>", usage: "Override the generated package name"},
		{name: "-tag <name>", usage: "Override the generated router tag name"},
		{name: "-with-gormx", usage: "Generate gormx-based CRUD code (default: false)"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Tips"))
	fmt.Fprintln(w, "  Use startproject or startapp first when you need a scaffolded package before CRUD generation.")
}

func printMakeMigrationsUsage(w io.Writer) {
	p := newHelpPrinter(w)
	fmt.Fprintln(w, p.title("makemigrations"))
	fmt.Fprintln(w, "Generate a timestamped SQL migration from MigrationModels().")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Usage"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli makemigrations [-config <path>] [-app-dir <path>] [-migrations-dir <path>] [-name <name>]"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Options"))
	printFlagGroup(w, []flagHelp{
		{name: "-config <path>", usage: "Project config file path (default: config.yaml)"},
		{name: "-app-dir <path>", usage: "Relative app package directory containing MigrationModels()"},
		{name: "-migrations-dir <path>", usage: "Relative migrations directory (default: migrations)"},
		{name: "-name <name>", usage: "Optional migration name suffix"},
	})
}

func printMigrateUsage(w io.Writer) {
	p := newHelpPrinter(w)
	fmt.Fprintln(w, p.title("migrate"))
	fmt.Fprintln(w, "Apply pending migrations or roll back to a target version.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Usage"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli migrate [target|zero] [-config <path>] [-migrations-dir <path>]"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Options"))
	printFlagGroup(w, []flagHelp{
		{name: "-config <path>", usage: "Project config file path (default: config.yaml)"},
		{name: "-migrations-dir <path>", usage: "Relative migrations directory (default: migrations)"},
	})
}

func printShowMigrationsUsage(w io.Writer) {
	p := newHelpPrinter(w)
	fmt.Fprintln(w, p.title("showmigrations"))
	fmt.Fprintln(w, "List migration files and whether each one has been applied.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Usage"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli showmigrations [-config <path>] [-migrations-dir <path>]"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Options"))
	printFlagGroup(w, []flagHelp{
		{name: "-config <path>", usage: "Project config file path (default: config.yaml)"},
		{name: "-migrations-dir <path>", usage: "Relative migrations directory (default: migrations)"},
	})
}

func printSQLMigrateUsage(w io.Writer) {
	p := newHelpPrinter(w)
	fmt.Fprintln(w, p.title("sqlmigrate"))
	fmt.Fprintln(w, "Print SQL for an existing migration file.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Usage"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli sqlmigrate <migration> [-config <path>] [-migrations-dir <path>] [-direction <up|down|all>]"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Options"))
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
	p := newHelpPrinter(w)
	for _, item := range flags {
		fmt.Fprintf(w, "  %s %s\n", p.command(fmt.Sprintf("%-28s", item.name)), item.usage)
	}
}
