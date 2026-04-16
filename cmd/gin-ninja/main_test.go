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

func TestRunGenerateCRUDWithNativeGORM(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte("package demo\n\ntype User struct {\n\tID uint `json:\"id\"`\n\tName string `json:\"name\" crud:\"filter,sort,search\"`\n}\n"), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"generate", "crud", "-model", "User", "-model-file", modelFile, "-with-gormx=false"})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	outputFile := filepath.Join(dir, "user_crud_gen.go")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if strings.Contains(string(content), "gormx.") {
		t.Fatalf("expected native GORM output\n%s", content)
	}
	if !strings.Contains(string(content), `query := db.Model(&User{})`) {
		t.Fatalf("expected native GORM query code\n%s", content)
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
