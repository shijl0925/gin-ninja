package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shijl0925/gin-ninja/cmd/gin-ninja-cli/internal/codegen"
	"go.yaml.in/yaml/v3"
)

type scaffoldPreset struct {
	Name        string `json:"name" yaml:"name"`
	Module      string `json:"module" yaml:"module"`
	Output      string `json:"output" yaml:"output"`
	AppDir      string `json:"app_dir" yaml:"app_dir"`
	PackageName string `json:"package" yaml:"package"`
	ModelName   string `json:"model" yaml:"model"`
	Database    string `json:"database" yaml:"database"`
	Template    string `json:"template" yaml:"template"`
	WithTests   *bool  `json:"with_tests" yaml:"with_tests"`
	WithAuth    *bool  `json:"with_auth" yaml:"with_auth"`
	WithAdmin   *bool  `json:"with_admin" yaml:"with_admin"`
	WithGormx   *bool  `json:"with_gormx" yaml:"with_gormx"`
	Force       *bool  `json:"force" yaml:"force"`
}

const defaultScaffoldAppDir = "app"

const (
	defaultProjectScaffoldDatabase = string(codegen.ScaffoldDatabaseSQLite)
	defaultAppScaffoldDatabase     = string(codegen.ScaffoldDatabaseNone)
)

var scaffoldTemplateChoices = []helpItem{
	{name: "minimal", usage: "Basic CRUD structure"},
	{name: "standard", usage: "Broader starter structure for everyday development"},
	{name: "auth", usage: "Adds auth-oriented scaffold files"},
	{name: "admin", usage: "Adds admin-oriented scaffold files"},
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
		printGenerateUsage(stderr)
	}
	model := fs.String("model", "", "Go struct name to scaffold")
	modelFile := fs.String("model-file", "", "Go source file containing the model struct")
	output := fs.String("output", "", "Output file path (defaults next to the model file)")
	packageName := fs.String("package", "", "Override generated package name")
	tag := fs.String("tag", "", "Override generated router tag name")
	withGormX := fs.Bool("with-gormx", false, "Generate gormx-based CRUD code")
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
		printStartProjectUsage(stderr)
	}
	module := fs.String("module", "", "Go module path for the generated project")
	output := fs.String("output", "", "Output directory (defaults to the project name)")
	appDir := fs.String("app-dir", "", "Relative app package directory inside the generated project")
	database := fs.String("database", "", "Database driver scaffold: sqlite, mysql, postgres, or none")
	configPath := fs.String("config", "", "Load scaffold preset from YAML or JSON")
	templateName := fs.String("template", "", "Scaffold template: minimal, standard, auth, admin")
	withTests := fs.Bool("with-tests", false, "Generate starter tests alongside scaffolded files")
	withAuth := fs.Bool("with-auth", false, "Include JWT auth scaffold files")
	withAdmin := fs.Bool("with-admin", false, "Include admin scaffold files")
	withGormx := fs.Bool("with-gormx", false, "Generate scaffold code using gormx repositories")
	force := fs.Bool("force", false, "Allow writing into an existing non-empty output directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	preset, err := loadScaffoldPreset(strings.TrimSpace(*configPath))
	if err != nil {
		fmt.Fprintf(stderr, "load scaffold preset: %v\n", err)
		return 1
	}
	set := visitedFlags(fs)

	name, status := resolveLeadingName(nameArg, fs.Args(), strings.TrimSpace(preset.Name))
	switch status {
	case nameStatusTooMany:
		fmt.Fprintln(stderr, "startproject accepts only one project name")
		return 2
	case nameStatusMissing:
		fmt.Fprintln(stderr, "startproject requires exactly one project name")
		return 2
	}

	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Fprintln(stderr, "project name must not be empty")
		return 2
	}

	out := mergeStringFlag(strings.TrimSpace(*output), set["output"], strings.TrimSpace(preset.Output), name)
	mod := mergeStringFlag(strings.TrimSpace(*module), set["module"], strings.TrimSpace(preset.Module), name)

	if err := codegen.WriteProjectScaffold(codegen.ProjectScaffoldConfig{
		Name:      name,
		Module:    mod,
		AppDir:    mergeStringFlag(strings.TrimSpace(*appDir), set["app-dir"], strings.TrimSpace(preset.AppDir), ""),
		Database:  mergeStringFlag(strings.TrimSpace(*database), set["database"], strings.TrimSpace(preset.Database), defaultProjectScaffoldDatabase),
		Template:  mergeStringFlag(strings.TrimSpace(*templateName), set["template"], strings.TrimSpace(preset.Template), string(codegen.ScaffoldTemplateMinimal)),
		WithTests: mergeBoolFlag(*withTests, set["with-tests"], preset.WithTests, false),
		WithAuth:  mergeBoolFlag(*withAuth, set["with-auth"], preset.WithAuth, false),
		WithAdmin: mergeBoolFlag(*withAdmin, set["with-admin"], preset.WithAdmin, false),
		WithGormx: boolPtr(mergeBoolFlag(*withGormx, set["with-gormx"], preset.WithGormx, false)),
		Force:     mergeBoolFlag(*force, set["force"], preset.Force, false),
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
		printStartAppUsage(stderr)
	}
	output := fs.String("output", "", "Output directory (defaults to the app name)")
	packageName := fs.String("package", "", "Override the generated Go package name")
	modelName := fs.String("model", "", "Override the generated model name")
	database := fs.String("database", "", "Database driver import to add: sqlite, mysql, postgres, or none")
	configPath := fs.String("config", "", "Load scaffold preset from YAML or JSON")
	templateName := fs.String("template", "", "Scaffold template: minimal, standard, auth, admin")
	withTests := fs.Bool("with-tests", false, "Generate starter tests alongside scaffolded files")
	withAuth := fs.Bool("with-auth", false, "Include JWT auth scaffold files")
	withAdmin := fs.Bool("with-admin", false, "Include admin scaffold files")
	withGormx := fs.Bool("with-gormx", false, "Generate scaffold code using gormx repositories")
	force := fs.Bool("force", false, "Allow writing into an existing non-empty output directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	preset, err := loadScaffoldPreset(strings.TrimSpace(*configPath))
	if err != nil {
		fmt.Fprintf(stderr, "load scaffold preset: %v\n", err)
		return 1
	}
	set := visitedFlags(fs)

	name, status := resolveLeadingName(nameArg, fs.Args(), strings.TrimSpace(preset.Name))
	switch status {
	case nameStatusTooMany:
		fmt.Fprintln(stderr, "startapp accepts only one app name")
		return 2
	case nameStatusMissing:
		fmt.Fprintln(stderr, "startapp requires exactly one app name")
		return 2
	}
	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Fprintln(stderr, "app name must not be empty")
		return 2
	}

	out := mergeStringFlag(strings.TrimSpace(*output), set["output"], strings.TrimSpace(preset.Output), name)

	if err := codegen.WriteAppScaffold(codegen.AppScaffoldConfig{
		Name:        name,
		PackageName: mergeStringFlag(strings.TrimSpace(*packageName), set["package"], strings.TrimSpace(preset.PackageName), ""),
		ModelName:   mergeStringFlag(strings.TrimSpace(*modelName), set["model"], strings.TrimSpace(preset.ModelName), ""),
		Database:    mergeStringFlag(strings.TrimSpace(*database), set["database"], strings.TrimSpace(preset.Database), defaultAppScaffoldDatabase),
		Template:    mergeStringFlag(strings.TrimSpace(*templateName), set["template"], strings.TrimSpace(preset.Template), string(codegen.ScaffoldTemplateMinimal)),
		WithTests:   mergeBoolFlag(*withTests, set["with-tests"], preset.WithTests, false),
		WithAuth:    mergeBoolFlag(*withAuth, set["with-auth"], preset.WithAuth, false),
		WithAdmin:   mergeBoolFlag(*withAdmin, set["with-admin"], preset.WithAdmin, false),
		WithGormx:   boolPtr(mergeBoolFlag(*withGormx, set["with-gormx"], preset.WithGormx, false)),
		Force:       mergeBoolFlag(*force, set["force"], preset.Force, false),
	}, out); err != nil {
		fmt.Fprintf(stderr, "create app scaffold: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created app scaffold in %s\n", out)
	return 0
}

func runInit(stdin io.Reader, stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		printInitUsage(stderr)
	}
	kindFlag := fs.String("kind", "", "Scaffold kind: project or app")
	configPath := fs.String("config", "", "Optional YAML/JSON preset to seed answers")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "init does not accept positional arguments")
		return 2
	}

	preset, err := loadScaffoldPreset(strings.TrimSpace(*configPath))
	if err != nil {
		fmt.Fprintf(stderr, "load scaffold preset: %v\n", err)
		return 1
	}

	reader := bufio.NewReader(stdin)
	kind := strings.ToLower(strings.TrimSpace(*kindFlag))
	if kind == "" {
		kind, err = promptChoice(stdout, reader, "What do you want to create?", "", []string{"project", "app"})
		if err != nil {
			fmt.Fprintf(stderr, "read scaffold kind: %v\n", err)
			return 1
		}
	}

	switch kind {
	case "project":
		return runInitProject(reader, stdout, stderr, preset)
	case "app":
		return runInitApp(reader, stdout, stderr, preset)
	default:
		fmt.Fprintln(stderr, "init kind must be either project or app")
		return 2
	}
}

func runInitProject(reader *bufio.Reader, stdout, stderr io.Writer, preset scaffoldPreset) int {
	name, err := promptRequired(stdout, reader, "Project name", strings.TrimSpace(preset.Name))
	if err != nil {
		fmt.Fprintf(stderr, "read project name: %v\n", err)
		return 1
	}
	module, err := promptString(stdout, reader, "Module path", mergeStringFlag("", false, strings.TrimSpace(preset.Module), name))
	if err != nil {
		fmt.Fprintf(stderr, "read module path: %v\n", err)
		return 1
	}
	output, err := promptString(stdout, reader, "Output directory", mergeStringFlag("", false, strings.TrimSpace(preset.Output), name))
	if err != nil {
		fmt.Fprintf(stderr, "read output directory: %v\n", err)
		return 1
	}
	appDir, err := promptString(stdout, reader, "App package directory", mergeStringFlag("", false, strings.TrimSpace(preset.AppDir), defaultScaffoldAppDir))
	if err != nil {
		fmt.Fprintf(stderr, "read app package directory: %v\n", err)
		return 1
	}
	templateName, err := promptChoice(stdout, reader, "Template preset", mergeStringFlag("", false, strings.TrimSpace(preset.Template), string(codegen.ScaffoldTemplateMinimal)), []string{
		string(codegen.ScaffoldTemplateMinimal),
		string(codegen.ScaffoldTemplateStandard),
		string(codegen.ScaffoldTemplateAuth),
		string(codegen.ScaffoldTemplateAdmin),
	})
	if err != nil {
		fmt.Fprintf(stderr, "read template preset: %v\n", err)
		return 1
	}
	database, err := promptChoice(stdout, reader, "Database driver", mergeStringFlag("", false, strings.TrimSpace(preset.Database), defaultProjectScaffoldDatabase), []string{
		string(codegen.ScaffoldDatabaseSQLite),
		string(codegen.ScaffoldDatabaseMySQL),
		string(codegen.ScaffoldDatabasePostgres),
		string(codegen.ScaffoldDatabaseNone),
	})
	if err != nil {
		fmt.Fprintf(stderr, "read database driver: %v\n", err)
		return 1
	}
	withTests, err := promptBool(stdout, reader, "Add starter tests?", boolValueOrDefault(preset.WithTests, false))
	if err != nil {
		fmt.Fprintf(stderr, "read test preference: %v\n", err)
		return 1
	}
	withGormx, err := promptBool(stdout, reader, "Use gormx repositories/services?", boolValueOrDefault(preset.WithGormx, false))
	if err != nil {
		fmt.Fprintf(stderr, "read gormx preference: %v\n", err)
		return 1
	}
	if err := codegen.WriteProjectScaffold(codegen.ProjectScaffoldConfig{
		Name:      name,
		Module:    module,
		AppDir:    appDir,
		Database:  database,
		Template:  templateName,
		WithTests: withTests,
		WithGormx: boolPtr(withGormx),
	}, output); err != nil {
		fmt.Fprintf(stderr, "create project scaffold: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created project scaffold in %s\n", output)
	return 0
}

func runInitApp(reader *bufio.Reader, stdout, stderr io.Writer, preset scaffoldPreset) int {
	name, err := promptRequired(stdout, reader, "App name", strings.TrimSpace(preset.Name))
	if err != nil {
		fmt.Fprintf(stderr, "read app name: %v\n", err)
		return 1
	}
	output, err := promptString(stdout, reader, "Output directory", mergeStringFlag("", false, strings.TrimSpace(preset.Output), name))
	if err != nil {
		fmt.Fprintf(stderr, "read output directory: %v\n", err)
		return 1
	}
	packageName, err := promptString(stdout, reader, "Go package name", strings.TrimSpace(preset.PackageName))
	if err != nil {
		fmt.Fprintf(stderr, "read package name: %v\n", err)
		return 1
	}
	modelName, err := promptString(stdout, reader, "Model name", strings.TrimSpace(preset.ModelName))
	if err != nil {
		fmt.Fprintf(stderr, "read model name: %v\n", err)
		return 1
	}
	templateName, err := promptChoice(stdout, reader, "Template preset", mergeStringFlag("", false, strings.TrimSpace(preset.Template), string(codegen.ScaffoldTemplateMinimal)), []string{
		string(codegen.ScaffoldTemplateMinimal),
		string(codegen.ScaffoldTemplateStandard),
		string(codegen.ScaffoldTemplateAuth),
		string(codegen.ScaffoldTemplateAdmin),
	})
	if err != nil {
		fmt.Fprintf(stderr, "read template preset: %v\n", err)
		return 1
	}
	database, err := promptChoice(stdout, reader, "Database driver import", mergeStringFlag("", false, strings.TrimSpace(preset.Database), defaultAppScaffoldDatabase), []string{
		string(codegen.ScaffoldDatabaseSQLite),
		string(codegen.ScaffoldDatabaseMySQL),
		string(codegen.ScaffoldDatabasePostgres),
		string(codegen.ScaffoldDatabaseNone),
	})
	if err != nil {
		fmt.Fprintf(stderr, "read database driver import: %v\n", err)
		return 1
	}
	withTests, err := promptBool(stdout, reader, "Add starter tests?", boolValueOrDefault(preset.WithTests, false))
	if err != nil {
		fmt.Fprintf(stderr, "read test preference: %v\n", err)
		return 1
	}
	withGormx, err := promptBool(stdout, reader, "Use gormx repositories/services?", boolValueOrDefault(preset.WithGormx, false))
	if err != nil {
		fmt.Fprintf(stderr, "read gormx preference: %v\n", err)
		return 1
	}
	if err := codegen.WriteAppScaffold(codegen.AppScaffoldConfig{
		Name:        name,
		PackageName: packageName,
		ModelName:   modelName,
		Database:    database,
		Template:    templateName,
		WithTests:   withTests,
		WithGormx:   boolPtr(withGormx),
	}, output); err != nil {
		fmt.Fprintf(stderr, "create app scaffold: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created app scaffold in %s\n", output)
	return 0
}

func printStartProjectUsage(w io.Writer) {
	p := newHelpPrinter(w)
	fmt.Fprintln(w, p.title("startproject"))
	fmt.Fprintln(w, "Create a new gin-ninja project scaffold.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Best for"))
	fmt.Fprintln(w, "  Bootstrapping a brand-new service; the minimal template is the recommended default.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Usage"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startproject <name> [-module <module>] [-output <path>] [-config <path>] [-template <minimal|standard|auth|admin>] [-database <sqlite|mysql|postgres|none>]"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Recommended flow"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startproject mysite -module github.com/acme/mysite"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startproject mysite -template standard"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startproject mysite -template auth"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startproject mysite -template admin"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startproject mysite -config ./scaffold.yaml"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Template choices"))
	printHelpItems(w, p.command, scaffoldTemplateChoices)
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Key options"))
	printFlagGroup(w, []flagHelp{
		{name: "-module <module>", usage: "Go module path (defaults to the project name)"},
		{name: "-output <path>", usage: "Output directory (defaults to the project name)"},
		{name: "-database <driver>", usage: "Database scaffold to wire: sqlite, mysql, postgres, or none (default: sqlite)"},
		{name: "-config <path>", usage: "Load options from a YAML or JSON preset file"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Template options"))
	printFlagGroup(w, []flagHelp{
		{name: "-template <preset>", usage: "Choose minimal, standard, auth, or admin (default: minimal)"},
		{name: "-with-tests", usage: "Add starter tests on top of the selected template"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Advanced options"))
	printFlagGroup(w, []flagHelp{
		{name: "-app-dir <path>", usage: fmt.Sprintf("Relative app package directory inside the project (default: %s)", defaultScaffoldAppDir)},
		{name: "-with-auth", usage: "Force-enable auth scaffold files"},
		{name: "-with-admin", usage: "Force-enable admin scaffold files"},
		{name: "-with-gormx", usage: "Generate gormx repositories/services (default: false)"},
		{name: "-force", usage: "Allow writing into an existing non-empty output directory"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Tips"))
	fmt.Fprintln(w, "  CLI flags override values loaded from -config.")
	fmt.Fprintln(w, "  Use startapp later to add more domain packages inside the generated project.")
}

func printStartAppUsage(w io.Writer) {
	p := newHelpPrinter(w)
	fmt.Fprintln(w, p.title("startapp"))
	fmt.Fprintln(w, "Create a new gin-ninja app scaffold.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Best for"))
	fmt.Fprintln(w, "  Existing projects that need a new domain package; the minimal template is the recommended default.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Usage"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startapp <name> [-output <path>] [-package <name>] [-model <name>] [-config <path>] [-template <minimal|standard|auth|admin>] [-database <sqlite|mysql|postgres|none>]"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Recommended flow"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startapp blog"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startapp accounts -template standard"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startapp accounts -template auth"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startapp accounts -template admin"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli startapp accounts -config ./scaffold.yaml"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Template choices"))
	printHelpItems(w, p.command, scaffoldTemplateChoices)
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Key options"))
	printFlagGroup(w, []flagHelp{
		{name: "-output <path>", usage: "Output directory (defaults to the app name)"},
		{name: "-package <name>", usage: "Override the generated Go package name"},
		{name: "-model <name>", usage: "Override the generated model name"},
		{name: "-database <driver>", usage: "Add a matching driver import file for sqlite, mysql, postgres, or none (default: none)"},
		{name: "-config <path>", usage: "Load options from a YAML or JSON preset file"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Template options"))
	printFlagGroup(w, []flagHelp{
		{name: "-template <preset>", usage: "Choose minimal, standard, auth, or admin (default: minimal)"},
		{name: "-with-tests", usage: "Add starter tests on top of the selected template"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Advanced options"))
	printFlagGroup(w, []flagHelp{
		{name: "-with-auth", usage: "Force-enable auth scaffold files"},
		{name: "-with-admin", usage: "Force-enable admin scaffold files"},
		{name: "-with-gormx", usage: "Generate gormx repositories/services (default: false)"},
		{name: "-force", usage: "Allow writing into an existing non-empty output directory"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Tips"))
	fmt.Fprintln(w, "  CLI flags override values loaded from -config.")
	fmt.Fprintln(w, "  Use generate crud after scaffolding when you already have a model and want CRUD handlers.")
}

func printInitUsage(w io.Writer) {
	p := newHelpPrinter(w)
	fmt.Fprintln(w, p.title("init"))
	fmt.Fprintln(w, "Run an interactive scaffold wizard for a project or app.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Best for"))
	fmt.Fprintln(w, "  Users who want guided prompts instead of remembering scaffold flags.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Usage"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli init [-kind <project|app>] [-config <path>]"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Recommended flow"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli init"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli init -kind project"))
	fmt.Fprintf(w, "  %s\n", p.command("gin-ninja-cli init -kind app -config ./scaffold.yaml"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Options"))
	printFlagGroup(w, []flagHelp{
		{name: "-kind <project|app>", usage: "Skip the first prompt and choose the scaffold flow up front"},
		{name: "-config <path>", usage: "Seed wizard answers from a YAML or JSON preset file"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, p.section("Tips"))
	fmt.Fprintln(w, "  Use -kind when you already know whether you want a project scaffold or an app scaffold.")
}

func loadScaffoldPreset(path string) (scaffoldPreset, error) {
	if path == "" {
		return scaffoldPreset{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return scaffoldPreset{}, err
	}
	var preset scaffoldPreset
	if err := yaml.Unmarshal(data, &preset); err != nil {
		return scaffoldPreset{}, err
	}
	return preset, nil
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	set := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		set[f.Name] = true
	})
	return set
}

type nameStatus int

const (
	nameStatusResolved nameStatus = iota
	nameStatusMissing
	nameStatusTooMany
)

func resolveLeadingName(leading string, rest []string, fallback string) (string, nameStatus) {
	if leading != "" {
		if len(rest) != 0 {
			return "", nameStatusTooMany
		}
		return leading, nameStatusResolved
	}
	if len(rest) > 1 {
		return "", nameStatusTooMany
	}
	if len(rest) == 1 {
		return rest[0], nameStatusResolved
	}
	if fallback != "" {
		return fallback, nameStatusResolved
	}
	return "", nameStatusMissing
}

func mergeStringFlag(flagValue string, flagSet bool, presetValue string, fallback string) string {
	if flagSet {
		return flagValue
	}
	if presetValue != "" {
		return presetValue
	}
	return fallback
}

func mergeBoolFlag(flagValue bool, flagSet bool, presetValue *bool, fallback bool) bool {
	if flagSet {
		return flagValue
	}
	if presetValue != nil {
		return *presetValue
	}
	return fallback
}

func promptRequired(stdout io.Writer, reader *bufio.Reader, label, defaultValue string) (string, error) {
	for {
		value, err := promptString(stdout, reader, label, defaultValue)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), nil
		}
		defaultValue = ""
	}
}

func promptString(stdout io.Writer, reader *bufio.Reader, label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(stdout, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(stdout, "%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	value := strings.TrimSpace(line)
	if value == "" {
		value = defaultValue
	}
	return value, nil
}

func promptChoice(stdout io.Writer, reader *bufio.Reader, label, defaultValue string, allowed []string) (string, error) {
	options := strings.Join(allowed, "/")
	for {
		value, err := promptString(stdout, reader, label+" ("+options+")", defaultValue)
		if err != nil {
			return "", err
		}
		value = strings.ToLower(strings.TrimSpace(value))
		for _, option := range allowed {
			if value == option {
				return value, nil
			}
		}
		fmt.Fprintf(stdout, "Please choose one of: %s\n", options)
	}
}

func promptBool(stdout io.Writer, reader *bufio.Reader, label string, defaultValue bool) (bool, error) {
	suffix := "y/N"
	if defaultValue {
		suffix = "Y/n"
	}
	for {
		value, err := promptString(stdout, reader, label+" ["+suffix+"]", "")
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "":
			return defaultValue, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		fmt.Fprintln(stdout, "Please answer yes or no.")
	}
}

func consumeLeadingName(args []string) (string, []string) {
	if len(args) == 0 || strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		return "", args
	}
	return args[0], args[1:]
}

func boolPtr(v bool) *bool {
	return &v
}

func boolValueOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
