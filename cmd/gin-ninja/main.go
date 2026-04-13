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
out = filepath.Join(filepath.Dir(cfg.ModelFile), defaultOutputName(cfg.Model))
}
if err := codegen.WriteCRUDFile(cfg, out); err != nil {
fmt.Fprintf(stderr, "generate crud scaffold: %v\n", err)
return 1
}
fmt.Fprintf(stdout, "generated %s\n", out)
return 0
}

func defaultOutputName(model string) string {
return toSnake(model) + "_crud_gen.go"
}

func toSnake(input string) string {
if input == "" {
return input
}
var out []rune
for i, r := range input {
if i > 0 && r >= 'A' && r <= 'Z' {
out = append(out, '_')
}
if r >= 'A' && r <= 'Z' {
r = r - 'A' + 'a'
}
out = append(out, r)
}
return string(out)
}

func printUsage(w io.Writer) {
fmt.Fprintln(w, "Usage:")
fmt.Fprintln(w, "  gin-ninja generate crud -model <Name> -model-file <path> [-output <path>]")
}

func printGenerateUsage(w io.Writer) {
fmt.Fprintln(w, "Usage:")
fmt.Fprintln(w, "  gin-ninja generate crud -model <Name> -model-file <path> [-output <path>]")
}
