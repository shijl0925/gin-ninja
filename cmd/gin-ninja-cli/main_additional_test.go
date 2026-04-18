package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUsageAndCommandErrors(t *testing.T) {
	t.Parallel()

	t.Run("no args", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(&stdout, &stderr, nil); code != 2 {
			t.Fatalf("run() code = %d, want 2", code)
		}
		if !strings.Contains(stderr.String(), "Usage:") {
			t.Fatalf("expected usage in stderr, got %q", stderr.String())
		}
	})

	t.Run("help", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(&stdout, &stderr, []string{"help"}); code != 0 {
			t.Fatalf("run() code = %d, want 0", code)
		}
		if !strings.Contains(stdout.String(), "Scaffold commands:") {
			t.Fatalf("expected usage in stdout, got %q", stdout.String())
		}
	})

	t.Run("help topic", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(&stdout, &stderr, []string{"help", "startproject"}); code != 0 {
			t.Fatalf("run() code = %d, want 0", code)
		}
		if !strings.Contains(stdout.String(), "Basic options:") {
			t.Fatalf("expected topic help in stdout, got %q", stdout.String())
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(&stdout, &stderr, []string{"nope"}); code != 2 {
			t.Fatalf("run() code = %d, want 2", code)
		}
		if !strings.Contains(stderr.String(), `unknown command "nope"`) {
			t.Fatalf("expected unknown command message, got %q", stderr.String())
		}
	})
}

func TestRunGenerateAdditionalBranches(t *testing.T) {
	t.Parallel()

	t.Run("missing target", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(&stdout, &stderr, []string{"generate"}); code != 2 {
			t.Fatalf("run() code = %d, want 2", code)
		}
		if !strings.Contains(stderr.String(), "generate crud") {
			t.Fatalf("expected generate usage, got %q", stderr.String())
		}
	})

	t.Run("unknown target", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(&stdout, &stderr, []string{"generate", "schema"}); code != 2 {
			t.Fatalf("run() code = %d, want 2", code)
		}
		if !strings.Contains(stderr.String(), `unknown generate target "schema"`) {
			t.Fatalf("expected unknown target message, got %q", stderr.String())
		}
	})

	t.Run("help flag", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(&stdout, &stderr, []string{"generate", "crud", "-h"}); code != 0 {
			t.Fatalf("run() code = %d, want 0", code)
		}
		if !strings.Contains(stderr.String(), "Examples:") {
			t.Fatalf("expected flag help in stderr, got %q", stderr.String())
		}
	})

	t.Run("write error", func(t *testing.T) {
		dir := t.TempDir()
		outputDir := filepath.Join(dir, "out")
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		modelFile := filepath.Join(dir, "models.go")
		if err := os.WriteFile(modelFile, []byte("package demo\n\ntype User struct{ ID uint }\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		var stdout, stderr bytes.Buffer
		code := run(&stdout, &stderr, []string{
			"generate", "crud",
			"-model", "User",
			"-model-file", modelFile,
			"-output", outputDir,
		})
		if code != 1 {
			t.Fatalf("run() code = %d, want 1 stderr=%q", code, stderr.String())
		}
		if !strings.Contains(stderr.String(), "generate crud scaffold") {
			t.Fatalf("expected scaffold error, got %q", stderr.String())
		}
	})
}

func TestRunStartProjectAndAppValidation(t *testing.T) {
	t.Parallel()

	t.Run("consume leading name", func(t *testing.T) {
		name, rest := consumeLeadingName([]string{"demo", "-force"})
		if name != "demo" || len(rest) != 1 || rest[0] != "-force" {
			t.Fatalf("unexpected consumeLeadingName result: name=%q rest=%v", name, rest)
		}
		name, rest = consumeLeadingName([]string{"-force"})
		if name != "" || len(rest) != 1 || rest[0] != "-force" {
			t.Fatalf("unexpected consumeLeadingName flag result: name=%q rest=%v", name, rest)
		}
	})

	t.Run("startproject help", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(&stdout, &stderr, []string{"startproject", "-h"}); code != 0 {
			t.Fatalf("run() code = %d, want 0", code)
		}
		if !strings.Contains(stderr.String(), "Template options:") {
			t.Fatalf("expected startproject help, got %q", stderr.String())
		}
	})

	t.Run("startproject validation", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
			msg  string
		}{
			{name: "missing name", args: []string{"startproject"}, msg: "requires exactly one project name"},
			{name: "duplicate names", args: []string{"startproject", "one", "two"}, msg: "accepts only one project name"},
			{name: "blank name", args: []string{"startproject", "   "}, msg: "project name must not be empty"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				if code := run(&stdout, &stderr, tc.args); code != 2 {
					t.Fatalf("run() code = %d, want 2", code)
				}
				if !strings.Contains(stderr.String(), tc.msg) {
					t.Fatalf("expected %q in stderr, got %q", tc.msg, stderr.String())
				}
			})
		}
	})

	t.Run("startapp help and validation", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(&stdout, &stderr, []string{"startapp", "-h"}); code != 0 {
			t.Fatalf("run() code = %d, want 0", code)
		}
		if !strings.Contains(stderr.String(), "Advanced overrides:") {
			t.Fatalf("expected startapp help, got %q", stderr.String())
		}

		for _, tc := range []struct {
			args []string
			msg  string
		}{
			{args: []string{"startapp"}, msg: "requires exactly one app name"},
			{args: []string{"startapp", "one", "two"}, msg: "accepts only one app name"},
			{args: []string{"startapp", "   "}, msg: "app name must not be empty"},
		} {
			stdout.Reset()
			stderr.Reset()
			if code := run(&stdout, &stderr, tc.args); code != 2 {
				t.Fatalf("run(%v) code = %d, want 2", tc.args, code)
			}
			if !strings.Contains(stderr.String(), tc.msg) {
				t.Fatalf("expected %q in stderr, got %q", tc.msg, stderr.String())
			}
		}
	})
}

func TestPrintUsageHelpers(t *testing.T) {
	t.Parallel()

	var usage, generate bytes.Buffer
	printRootUsage(&usage)
	printGenerateUsage(&generate)

	if !strings.Contains(usage.String(), "generate crud") {
		t.Fatalf("expected full usage output, got %q", usage.String())
	}
	if !strings.Contains(usage.String(), "Scaffold commands:") {
		t.Fatalf("expected grouped usage output, got %q", usage.String())
	}
	if !strings.Contains(generate.String(), "generate crud") {
		t.Fatalf("expected generate usage output, got %q", generate.String())
	}
	if got := boolPtr(true); got == nil || !*got {
		t.Fatalf("expected boolPtr(true) to return a true pointer, got %v", got)
	}
}
