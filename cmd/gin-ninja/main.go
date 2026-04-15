package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shijl0925/gin-ninja/codegen"
)

func main() {
	os.Exit(run(os.Stdout, os.Stderr, os.Args[1:]))
}

func run(stdout, stderr io.Writer, args []string) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "generate":
		return runGenerate(stdout, stderr, args[1:])
	case "startproject":
		return runStartProject(stdout, stderr, args[1:])
	case "startapp":
		return runStartApp(stdout, stderr, args[1:])
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runGenerate(stdout, stderr io.Writer, args []string) int {
	if len(args) == 0 {
		printGenerateUsage(stderr)
		return 2
	}
	if args[0] != "crud" {
		fmt.Fprintf(stderr, "unknown generate target %q\n\n", args[0])
		printGenerateUsage(stderr)
		return 2
	}

	fs := flag.NewFlagSet("crud", flag.ContinueOnError)
	fs.SetOutput(stderr)
	model := fs.String("model", "", "Go struct name to scaffold")
	modelFile := fs.String("model-file", "", "Go source file containing the model struct")
	output := fs.String("output", "", "Output file path (defaults next to the model file)")
	packageName := fs.String("package", "", "Override generated package name")
	tag := fs.String("tag", "", "Override generated router tag name")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	cfg := codegen.CRUDConfig{
		ModelFile:   strings.TrimSpace(*modelFile),
		Model:       strings.TrimSpace(*model),
		PackageName: strings.TrimSpace(*packageName),
		Tag:         strings.TrimSpace(*tag),
	}
	if cfg.ModelFile == "" || cfg.Model == "" {
		fmt.Fprintln(stderr, "-model and -model-file are required")
		return 2
	}

	out := strings.TrimSpace(*output)
	if out == "" {
		out = filepath.Join(filepath.Dir(cfg.ModelFile), codegen.DefaultOutputName(cfg.Model))
	}
	if err := codegen.WriteCRUDFile(cfg, out); err != nil {
		fmt.Fprintf(stderr, "generate crud scaffold: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "generated %s\n", out)
	return 0
}

func runStartProject(stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("startproject", flag.ContinueOnError)
	fs.SetOutput(stderr)
	module := fs.String("module", "", "Go module path for the generated project")
	output := fs.String("output", "", "Output directory (defaults to the project name)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "startproject requires exactly one project name")
		return 2
	}

	name := strings.TrimSpace(fs.Arg(0))
	if name == "" {
		fmt.Fprintln(stderr, "project name must not be empty")
		return 2
	}
	out := strings.TrimSpace(*output)
	if out == "" {
		out = name
	}
	mod := strings.TrimSpace(*module)
	if mod == "" {
		mod = name
	}

	if err := codegen.WriteProjectScaffold(codegen.ProjectScaffoldConfig{
		Name:   name,
		Module: mod,
	}, out); err != nil {
		fmt.Fprintf(stderr, "create project scaffold: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created project scaffold in %s\n", out)
	return 0
}

func runStartApp(stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("startapp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	output := fs.String("output", "", "Output directory (defaults to the app name)")
	packageName := fs.String("package", "", "Override the generated Go package name")
	modelName := fs.String("model", "", "Override the generated model name")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "startapp requires exactly one app name")
		return 2
	}

	name := strings.TrimSpace(fs.Arg(0))
	if name == "" {
		fmt.Fprintln(stderr, "app name must not be empty")
		return 2
	}
	out := strings.TrimSpace(*output)
	if out == "" {
		out = name
	}

	if err := codegen.WriteAppScaffold(codegen.AppScaffoldConfig{
		Name:        name,
		PackageName: strings.TrimSpace(*packageName),
		ModelName:   strings.TrimSpace(*modelName),
	}, out); err != nil {
		fmt.Fprintf(stderr, "create app scaffold: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created app scaffold in %s\n", out)
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gin-ninja startproject <name> [-module <module>] [-output <path>]")
	fmt.Fprintln(w, "  gin-ninja startapp <name> [-output <path>] [-package <name>] [-model <name>]")
	fmt.Fprintln(w, "  gin-ninja generate crud -model <Name> -model-file <path> [-output <path>]")
}

func printGenerateUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gin-ninja generate crud -model <Name> -model-file <path> [-output <path>]")
}
