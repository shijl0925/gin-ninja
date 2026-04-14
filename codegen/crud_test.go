package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCRUD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

import (
"time"

"gorm.io/gorm"
)

type User struct {
gorm.Model
Name     string    `+"`json:\"name\" binding:\"required\" description:\"Full name\"`"+`
Email    string    `+"`json:\"email\" binding:\"required,email\"`"+`
Password string    `+"`json:\"-\"`"+`
Age      int       `+"`json:\"age\"`"+`
IsAdmin  bool      `+"`json:\"is_admin\"`"+`
Created  time.Time `+"`json:\"created\"`"+`
Roles    []string  `+"`gorm:\"-\" json:\"roles\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "User"})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	generated := string(content)

	checks := []string{
		"type UserOut struct",
		"ninja.ModelSchema[User] `fields:\"id,name,email,age,is_admin,created\"`",
		"type IUserRepo interface",
		"func NewUserRepo() IUserRepo",
		"func toUserOut(item User) (*UserOut, error)",
		"type CreateUserInput struct",
		"Name string `json:\"name\" binding:\"required\" description:\"Full name\"`",
		"Email *string `json:\"email\" binding:\"omitempty,email\"`",
		"Created *time.Time `json:\"created\"`",
		"func RegisterUserCRUDRoutes(router *ninja.Router)",
		"func ListUsers(ctx *ninja.Context, in *ListUsersInput)",
		"items, total, err := repo.SelectPage(in.GetPage(), in.GetSize())",
		"return toUserOut(item)",
		"if err := repo.Insert(item); err != nil {",
		"if err := repo.UpdateByOpts(updates, gormx.Where(\"id = ?\", in.ID)); err != nil {",
		"return repo.DeleteByOpts(gormx.Where(\"id = ?\", in.ID))",
	}
	for _, check := range checks {
		if !strings.Contains(generated, check) {
			t.Fatalf("generated content missing %q\n%s", check, generated)
		}
	}

	if strings.Contains(generated, "Password string") {
		t.Fatalf("expected hidden fields to be excluded\n%s", generated)
	}
	if strings.Contains(generated, "Roles []string") {
		t.Fatalf("expected gorm ignored fields to be excluded\n%s", generated)
	}
}

func TestGenerateCRUDRequiresKnownModel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte("package demo\ntype User struct{}\n"), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	_, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Project"})
	if err == nil || !strings.Contains(err.Error(), "model \"Project\" not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
