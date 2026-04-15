package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGenerateCRUD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte("package demo\n\ntype User struct {\n\tID uint `json:\"id\"`\n\tName string `json:\"name\"`\n}\n"), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"generate", "crud", "-model", "User", "-model-file", modelFile})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	outputFile := filepath.Join(dir, "user_crud_gen.go")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if !strings.Contains(string(content), "func RegisterUserCRUDRoutes") {
		t.Fatalf("unexpected generated content\n%s", content)
	}
	if !strings.Contains(stdout.String(), outputFile) {
		t.Fatalf("stdout missing generated path: %s", stdout.String())
	}
}

func TestRunGenerateCRUDRequiresFlags(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"generate", "crud"})
	if code != 2 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "-model and -model-file are required") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunStartProject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "mysite")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"startproject",
		"mysite",
		"-module", "github.com/acme/mysite",
		"-output", outputDir,
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	checks := []string{
		filepath.Join(outputDir, "go.mod"),
		filepath.Join(outputDir, "main.go"),
		filepath.Join(outputDir, "config.yaml"),
		filepath.Join(outputDir, "app", "models.go"),
		filepath.Join(outputDir, "app", "repos.go"),
		filepath.Join(outputDir, "app", "schemas.go"),
		filepath.Join(outputDir, "app", "apis.go"),
		filepath.Join(outputDir, "app", "routers.go"),
	}
	for _, path := range checks {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}
	if !strings.Contains(stdout.String(), outputDir) {
		t.Fatalf("stdout missing scaffold path: %s", stdout.String())
	}
}

func TestRunStartApp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "blog")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"startapp",
		"blog",
		"-output", outputDir,
		"-package", "blog",
		"-model", "Post",
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	for _, file := range []string{"models.go", "repos.go", "schemas.go", "apis.go", "routers.go"} {
		path := filepath.Join(outputDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}
	if !strings.Contains(stdout.String(), outputDir) {
		t.Fatalf("stdout missing scaffold path: %s", stdout.String())
	}
}
