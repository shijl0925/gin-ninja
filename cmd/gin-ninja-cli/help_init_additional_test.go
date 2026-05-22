package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelpTopics(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"help", "startapp"}, want: "Create a new gin-ninja app scaffold"},
		{args: []string{"help", "init"}, want: "interactive scaffold wizard"},
		{args: []string{"help", "generate"}, want: "Generate CRUD scaffold code"},
		{args: []string{"help", "generate", "crud"}, want: "Generate CRUD scaffold code"},
		{args: []string{"help", "makemigrations"}, want: "timestamped SQL migration"},
		{args: []string{"help", "migrate"}, want: "Apply pending migrations"},
		{args: []string{"help", "showmigrations"}, want: "List migration files"},
		{args: []string{"help", "sqlmigrate"}, want: "Print SQL"},
		{args: []string{"-h"}, want: "DECISION GUIDE"},
		{args: []string{"--help"}, want: "DECISION GUIDE"},
	} {
		tc := tc
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(&stdout, &stderr, tc.args); code != 0 {
				t.Fatalf("run(%v) code = %d stderr=%q", tc.args, code, stderr.String())
			}
			if !strings.Contains(stdout.String(), tc.want) {
				t.Fatalf("run(%v) missing %q in stdout:\n%s", tc.args, tc.want, stdout.String())
			}
		})
	}
}

func TestRunHelpUnknownTopic(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(&stdout, &stderr, []string{"help", "unknown", "topic"}); code != 2 {
		t.Fatalf("run help unknown code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), `unknown help topic "unknown topic"`) {
		t.Fatalf("expected unknown help topic, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "DECISION GUIDE") {
		t.Fatalf("expected root usage after unknown topic, got %q", stderr.String())
	}
}

func TestHelpPrinterBranches(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if shouldUseHelpColor(os.Stdout) {
		t.Fatal("NO_COLOR should disable terminal color")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("CI", "1")
	if shouldUseHelpColor(os.Stdout) {
		t.Fatal("CI should disable terminal color")
	}
	t.Setenv("CI", "")
	t.Setenv("TERM", "dumb")
	if shouldUseHelpColor(os.Stdout) {
		t.Fatal("TERM=dumb should disable terminal color")
	}

	var out bytes.Buffer
	printHelpItems(&out, strings.ToUpper, []helpItem{
		{name: "a", usage: "first"},
		{name: "long", usage: "second"},
	})
	if got := out.String(); !strings.Contains(got, "A     first") || !strings.Contains(got, "LONG  second") {
		t.Fatalf("unexpected aligned help items: %q", got)
	}
}

func TestRunInitInteractiveProjectAndApp(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "guided-project")
	projectInput := strings.Join([]string{
		"guided",
		"github.com/acme/guided",
		projectDir,
		"internal/app",
		"standard",
		"none",
		"yes",
		"no",
	}, "\n") + "\n"
	var stdout, stderr bytes.Buffer
	if code := runWithInput(strings.NewReader(projectInput), &stdout, &stderr, []string{"init", "-kind", "project"}); code != 0 {
		t.Fatalf("init project code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(filepath.Join(projectDir, "internal", "app", "scaffold_test.go")); err != nil {
		t.Fatalf("expected guided project test scaffold: %v", err)
	}

	appDir := filepath.Join(t.TempDir(), "guided-app")
	appInput := strings.Join([]string{
		"blog",
		appDir,
		"blogpkg",
		"Post",
		"auth",
		"sqlite",
		"yes",
		"yes",
	}, "\n") + "\n"
	stdout.Reset()
	stderr.Reset()
	if code := runWithInput(strings.NewReader(appInput), &stdout, &stderr, []string{"init", "-kind", "app"}); code != 0 {
		t.Fatalf("init app code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(filepath.Join(appDir, "auth.go")); err != nil {
		t.Fatalf("expected guided app auth scaffold: %v", err)
	}
	if _, err := os.Stat(filepath.Join(appDir, "scaffold_test.go")); err != nil {
		t.Fatalf("expected guided app test scaffold: %v", err)
	}
}

func TestRunInitValidationBranches(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input io.Reader
		args  []string
		code  int
		want  string
	}{
		{name: "help", args: []string{"init", "-h"}, code: 0, want: "interactive scaffold wizard"},
		{name: "positional", args: []string{"init", "extra"}, code: 2, want: "does not accept positional"},
		{name: "unknown kind", args: []string{"init", "-kind", "site"}, code: 2, want: "kind must be either"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			if input == nil {
				input = strings.NewReader("")
			}
			var stdout, stderr bytes.Buffer
			if code := runWithInput(input, &stdout, &stderr, tc.args); code != tc.code {
				t.Fatalf("code = %d, want %d stderr=%q stdout=%q", code, tc.code, stderr.String(), stdout.String())
			}
			combined := stdout.String() + stderr.String()
			if !strings.Contains(combined, tc.want) {
				t.Fatalf("missing %q in output stdout=%q stderr=%q", tc.want, stdout.String(), stderr.String())
			}
		})
	}
}

func TestScaffoldPresetAndPromptHelpers(t *testing.T) {
	dir := t.TempDir()
	presetPath := filepath.Join(dir, "scaffold.yaml")
	if err := os.WriteFile(presetPath, []byte(`
name: presetapp
output: `+filepath.Join(dir, "preset-app")+`
package: presetpkg
model: PresetModel
database: none
template: admin
with_tests: true
with_gormx: true
force: true
`), 0o600); err != nil {
		t.Fatalf("write preset: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if code := run(&stdout, &stderr, []string{"startapp", "-config", presetPath}); code != 0 {
		t.Fatalf("startapp preset code = %d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "preset-app", "admin.go")); err != nil {
		t.Fatalf("expected admin scaffold from preset: %v", err)
	}

	if _, err := loadScaffoldPreset(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("expected missing preset error")
	}
	if got := mergeStringFlag("flag", true, "preset", "fallback"); got != "flag" {
		t.Fatalf("mergeStringFlag flag = %q", got)
	}
	if got := mergeBoolFlag(false, true, boolPtr(true), true); got {
		t.Fatal("mergeBoolFlag should prefer explicit false flag")
	}
	if got := boolValueOrDefault(boolPtr(true), false); !got {
		t.Fatal("boolValueOrDefault should return pointer value")
	}
}
