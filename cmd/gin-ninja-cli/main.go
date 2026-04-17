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
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  gin-ninja-cli generate crud -model <Name> -model-file <path> [-output <path>] [-package <name>] [-tag <name>] [-with-gormx]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	model := fs.String("model", "", "Go struct name to scaffold")
	modelFile := fs.String("model-file", "", "Go source file containing the model struct")
	output := fs.String("output", "", "Output file path (defaults next to the model file)")
	packageName := fs.String("package", "", "Override generated package name")
	tag := fs.String("tag", "", "Override generated router tag name")
	withGormX := fs.Bool("with-gormx", true, "Generate gormx-based CRUD code (set false for native GORM)")
	if err := fs.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	cfg := codegen.CRUDConfig{
		ModelFile:   strings.TrimSpace(*modelFile),
		Model:       strings.TrimSpace(*model),
		PackageName: strings.TrimSpace(*packageName),
		Tag:         strings.TrimSpace(*tag),
		WithGormX:   withGormX,
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
	nameArg, args := consumeLeadingName(args)

	fs := flag.NewFlagSet("startproject", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  gin-ninja-cli startproject <name> [-module <module>] [-output <path>] [-app-dir <path>] [-template <minimal|standard|auth|admin>] [-with-tests] [-with-auth] [-with-admin] [-with-gormx] [-force]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	module := fs.String("module", "", "Go module path for the generated project")
	output := fs.String("output", "", "Output directory (defaults to the project name)")
	appDir := fs.String("app-dir", "", "Relative app package directory inside the generated project (default: app)")
	templateName := fs.String("template", string(codegen.ScaffoldTemplateMinimal), "Scaffold template: minimal, standard, auth, admin")
	withTests := fs.Bool("with-tests", false, "Generate starter tests alongside scaffolded files")
	withAuth := fs.Bool("with-auth", false, "Include JWT auth scaffold files")
	withAdmin := fs.Bool("with-admin", false, "Include admin scaffold files")
	withGormx := fs.Bool("with-gormx", true, "Generate scaffold code using gormx repositories")
	force := fs.Bool("force", false, "Allow writing into an existing non-empty output directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if nameArg == "" && fs.NArg() != 1 {
		fmt.Fprintln(stderr, "startproject requires exactly one project name")
		return 2
	}
	if nameArg != "" && fs.NArg() != 0 {
		fmt.Fprintln(stderr, "startproject accepts only one project name")
		return 2
	}

	name := strings.TrimSpace(nameArg)
	if name == "" {
		name = strings.TrimSpace(fs.Arg(0))
	}
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
		Name:      name,
		Module:    mod,
		AppDir:    strings.TrimSpace(*appDir),
		Template:  strings.TrimSpace(*templateName),
		WithTests: *withTests,
		WithAuth:  *withAuth,
		WithAdmin: *withAdmin,
		WithGormx: boolPtr(*withGormx),
		Force:     *force,
	}, out); err != nil {
		fmt.Fprintf(stderr, "create project scaffold: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created project scaffold in %s\n", out)
	return 0
}

func runStartApp(stdout, stderr io.Writer, args []string) int {
	nameArg, args := consumeLeadingName(args)

	fs := flag.NewFlagSet("startapp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  gin-ninja-cli startapp <name> [-output <path>] [-package <name>] [-model <name>] [-template <minimal|standard|auth|admin>] [-with-tests] [-with-auth] [-with-admin] [-with-gormx] [-force]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	output := fs.String("output", "", "Output directory (defaults to the app name)")
	packageName := fs.String("package", "", "Override the generated Go package name")
	modelName := fs.String("model", "", "Override the generated model name")
	templateName := fs.String("template", string(codegen.ScaffoldTemplateMinimal), "Scaffold template: minimal, standard, auth, admin")
	withTests := fs.Bool("with-tests", false, "Generate starter tests alongside scaffolded files")
	withAuth := fs.Bool("with-auth", false, "Include JWT auth scaffold files")
	withAdmin := fs.Bool("with-admin", false, "Include admin scaffold files")
	withGormx := fs.Bool("with-gormx", true, "Generate scaffold code using gormx repositories")
	force := fs.Bool("force", false, "Allow writing into an existing non-empty output directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if nameArg == "" && fs.NArg() != 1 {
		fmt.Fprintln(stderr, "startapp requires exactly one app name")
		return 2
	}
	if nameArg != "" && fs.NArg() != 0 {
		fmt.Fprintln(stderr, "startapp accepts only one app name")
		return 2
	}

	name := strings.TrimSpace(nameArg)
	if name == "" {
		name = strings.TrimSpace(fs.Arg(0))
	}
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
		Template:    strings.TrimSpace(*templateName),
		WithTests:   *withTests,
		WithAuth:    *withAuth,
		WithAdmin:   *withAdmin,
		WithGormx:   boolPtr(*withGormx),
		Force:       *force,
	}, out); err != nil {
		fmt.Fprintf(stderr, "create app scaffold: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created app scaffold in %s\n", out)
	return 0
}

func consumeLeadingName(args []string) (string, []string) {
	if len(args) == 0 || strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		return "", args
	}
	return args[0], args[1:]
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gin-ninja-cli startproject <name> [-module <module>] [-output <path>] [-app-dir <path>] [-template <minimal|standard|auth|admin>] [-with-tests] [-with-auth] [-with-admin] [-with-gormx] [-force]")
	fmt.Fprintln(w, "  gin-ninja-cli startapp <name> [-output <path>] [-package <name>] [-model <name>] [-template <minimal|standard|auth|admin>] [-with-tests] [-with-auth] [-with-admin] [-with-gormx] [-force]")
	fmt.Fprintln(w, "  gin-ninja-cli generate crud -model <Name> -model-file <path> [-output <path>] [-package <name>] [-tag <name>] [-with-gormx]")
}

func boolPtr(v bool) *bool {
	return &v
}

func printGenerateUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gin-ninja-cli generate crud -model <Name> -model-file <path> [-output <path>] [-package <name>] [-tag <name>] [-with-gormx]")
}
