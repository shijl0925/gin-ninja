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
Name            string    `+"`json:\"name\" binding:\"required\" description:\"Full name\"`"+`
Email           string    `+"`json:\"email\" binding:\"required,email\"`"+`
PasswordHash    string    `+"`json:\"password_hash\" ninja:\"writeOnly\" binding:\"required,min=8\"`"+`
InviteCode      string    `+"`json:\"invite_code\" ninja:\"createOnly\"`"+`
StatusNote      string    `+"`json:\"status_note\" crud:\"updateOnly\"`"+`
LegacySecret    string    `+"`json:\"-\"`"+`
Age             int       `+"`json:\"age\"`"+`
IsAdmin         bool      `+"`json:\"is_admin\"`"+`
Created         time.Time `+"`json:\"created\"`"+`
Roles           []string  `+"`gorm:\"-\" json:\"roles\"`"+`
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
		"ninja.ModelSchema[User] `fields:\"id,name,email,invite_code,status_note,age,is_admin,created\"`",
		"type IUserRepo interface",
		"func NewUserRepo() IUserRepo",
		"func toUserOut(item User) (*UserOut, error)",
		"type CreateUserInput struct",
		"Name string `json:\"name\" binding:\"required\" description:\"Full name\"`",
		"PasswordHash string `json:\"password_hash\" binding:\"required,min=8\"`",
		"InviteCode string `json:\"invite_code\"`",
		"LegacySecret string `json:\"legacySecret\"`",
		"Email *string `json:\"email\" binding:\"omitempty,email\"`",
		"PasswordHash *string `json:\"password_hash\" binding:\"omitempty,min=8\"`",
		"StatusNote *string `json:\"status_note\"`",
		"LegacySecret *string `json:\"legacySecret\"`",
		"Created *time.Time `json:\"created\"`",
		"func RegisterUserCRUDRoutes(router *ninja.Router)",
		"func ListUsers(ctx *ninja.Context, in *ListUsersInput)",
		"func GetUser(ctx *ninja.Context, in *GetUserInput)",
		"items, total, err := repo.SelectPage(in.GetPage(), in.GetSize())",
		"return toUserOut(item)",
		"if err := repo.Insert(item); err != nil {",
		"item, err := repo.SelectOneById(int(in.ID))",
		"if err := repo.UpdateById(int(in.ID), updates); err != nil {",
		"if _, err := repo.SelectOneById(int(in.ID)); err != nil {",
		"return repo.DeleteById(int(in.ID))",
	}
	for _, check := range checks {
		if !strings.Contains(generated, check) {
			t.Fatalf("generated content missing %q\n%s", check, generated)
		}
	}

	if strings.Contains(generated, "fields:\"id,name,email,password,age,is_admin,created\"") {
		t.Fatalf("expected hidden fields to be excluded from output schema\n%s", generated)
	}
	for _, unexpected := range []string{
		"InviteCode *string `json:\"invite_code\"`",
		"StatusNote string `json:\"status_note\"`",
		"PasswordHash string `json:\"passwordHash\"`",
	} {
		if strings.Contains(generated, unexpected) {
			t.Fatalf("generated content unexpectedly contained %q\n%s", unexpected, generated)
		}
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

func TestGenerateCRUDStringIDFallsBackToOpts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type Session struct {
	ID    string `+"`json:\"id\" gorm:\"primaryKey\"`"+`
	Token string `+"`json:\"-\" binding:\"required\"`"+`
	Name  string `+"`json:\"name\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Session"})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	generated := string(content)

	checks := []string{
		"item, err := repo.SelectOneByOpts(gormx.Where(\"id = ?\", in.ID))",
		"Token string `json:\"token\" binding:\"required\"`",
		"if err := repo.UpdateByOpts(updates, gormx.Where(\"id = ?\", in.ID)); err != nil {",
		"if _, err := repo.SelectOneByOpts(gormx.Where(\"id = ?\", in.ID)); err != nil {",
		"return repo.DeleteByOpts(gormx.Where(\"id = ?\", in.ID))",
	}
	for _, check := range checks {
		if !strings.Contains(generated, check) {
			t.Fatalf("generated content missing %q\n%s", check, generated)
		}
	}

	if strings.Contains(generated, "int(in.ID)") {
		t.Fatalf("expected non-integer IDs to avoid int conversion\n%s", generated)
	}
}

func TestGenerateCRUDUsesSnakeCaseColumnsForAcronyms(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type Membership struct {
	ID      uint `+"`json:\"id\"`"+`
	UserID  uint `+"`json:\"user_id\"`"+`
	RoleIDs []uint `+"`gorm:\"-\" json:\"role_ids\"`"+`
	APIKey  string `+"`json:\"api_key\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Membership"})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	generated := string(content)

	checks := []string{
		`updates["user_id"] = *in.UserID`,
		`updates["api_key"] = *in.APIKey`,
	}
	for _, check := range checks {
		if !strings.Contains(generated, check) {
			t.Fatalf("generated content missing %q\n%s", check, generated)
		}
	}

	if strings.Contains(generated, `updates["user_i_d"]`) || strings.Contains(generated, `updates["a_p_i_key"]`) {
		t.Fatalf("expected acronym fields to use stable snake_case\n%s", generated)
	}
}
