package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func boolPtr(v bool) *bool {
	return &v
}

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
		`ninja.Post(router, "/", CreateUser, ninja.Summary("Create user"), ninja.WithTransaction())`,
		`ninja.Patch(router, "/:id", UpdateUser, ninja.Summary("Patch user"), ninja.WithTransaction())`,
		`ninja.Delete(router, "/:id", DeleteUser, ninja.Summary("Delete user"), ninja.WithTransaction())`,
		"items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), opts...)",
		"return toUserOut(item)",
		"if err := repo.Insert(item, gormx.UseDB(db)); err != nil {",
		"func loadUserByID(db *gorm.DB, id uint) (User, error)",
		"if err := repo.UpdateByOpts(updates, gormx.UseDB(db), gormx.Where(\"id = ?\", in.ID)); err != nil {",
		"if _, err := repo.SelectOneByOpts(gormx.UseDB(db), gormx.Where(\"id = ?\", in.ID)); err != nil {",
		"return repo.DeleteByOpts(gormx.UseDB(db), gormx.Where(\"id = ?\", in.ID))",
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

func TestGenerateCRUDWithNativeGORM(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type User struct {
	ID    uint   `+"`json:\"id\"`"+`
	Name  string `+"`json:\"name\" crud:\"filter,sort,search\"`"+`
	Email string `+"`json:\"email\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "User", WithGormX: boolPtr(false)})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	generated := string(content)

	checks := []string{
		`query := db.Model(&User{})`,
		`query, err = filter.ApplyDB(query, in)`,
		`countQuery := query.Session(&gorm.Session{})`,
		`query, err = order.ApplyDB(query, in)`,
		`if err := db.Create(item).Error; err != nil {`,
		`if err := db.Model(&User{}).Where("id = ?", in.ID).Updates(updates).Error; err != nil {`,
		`if err := query.Where("id = ?", id).First(&item).Error; err != nil {`,
		`return db.Model(&User{}).Where("id = ?", in.ID).Delete(&User{}).Error`,
	}
	for _, check := range checks {
		if !strings.Contains(generated, check) {
			t.Fatalf("generated content missing %q\n%s", check, generated)
		}
	}

	for _, unexpected := range []string{
		`gormx.`,
		`type IUserRepo interface`,
		`func NewUserRepo()`,
		`func applyUserFilters(`,
		`func applyUserSort(`,
		`func buildUserFilterExpr(`,
	} {
		if strings.Contains(generated, unexpected) {
			t.Fatalf("generated content unexpectedly contained %q\n%s", unexpected, generated)
		}
	}
}

func TestGenerateCRUDNativeGORMNoFilterOrSort(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type Widget struct {
	ID    uint   `+"`json:\"id\"`"+`
	Name  string `+"`json:\"name\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Widget", WithGormX: boolPtr(false)})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	generated := string(content)

	// Generated code must compile: no orphan `var err error` when no filter/sort fields
	if strings.Contains(generated, "var err error") {
		t.Fatalf("generated code must not declare unused 'var err error' when no filter or sort fields:\n%s", generated)
	}
	if strings.Contains(generated, "filter.ApplyDB") {
		t.Fatalf("generated code must not call filter.ApplyDB when no filter fields:\n%s", generated)
	}
	if strings.Contains(generated, "order.ApplyDB") {
		t.Fatalf("generated code must not call order.ApplyDB when no sort fields:\n%s", generated)
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
		"func loadSessionByID(db *gorm.DB, id string) (Session, error)",
		"Token string `json:\"token\" binding:\"required\"`",
		"if err := repo.UpdateByOpts(updates, gormx.UseDB(db), gormx.Where(\"id = ?\", in.ID)); err != nil {",
		"if _, err := repo.SelectOneByOpts(gormx.UseDB(db), gormx.Where(\"id = ?\", in.ID)); err != nil {",
		"return repo.DeleteByOpts(gormx.UseDB(db), gormx.Where(\"id = ?\", in.ID))",
	}
	for _, check := range checks {
		if !strings.Contains(generated, check) {
			t.Fatalf("generated content missing %q\n%s", check, generated)
		}
	}

	if strings.Contains(generated, "int(in.ID)") {
		t.Fatalf("expected generated content to avoid int conversion for non-int IDs\n%s", generated)
	}
}

func TestGenerateCRUDStringPrimaryKeyAllowsCreateInputID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type Session struct {
	ID   string `+"`json:\"id\" gorm:\"primaryKey\"`"+`
	Name string `+"`json:\"name\"`"+`
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
		"ID string `json:\"id\"`",
		"item.ID = in.ID",
		"loaded, err := loadSessionByID(db, item.ID)",
	}
	for _, check := range checks {
		if !strings.Contains(generated, check) {
			t.Fatalf("generated content missing %q\n%s", check, generated)
		}
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

func TestGenerateCRUDBuildsAndRegistersPatchRoute(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type User struct {
	ID   uint   `+"`json:\"id\"`"+`
	Name string `+"`json:\"name\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "User"})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "user_crud_gen.go"), content, 0o644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "route_test.go"), []byte(`package demo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ninja "github.com/shijl0925/gin-ninja"
)

func TestGeneratedRoutesUsePatchForPartialUpdates(t *testing.T) {
	api := ninja.New(ninja.Config{})
	router := ninja.NewRouter("/users")
	RegisterUserCRUDRoutes(router)
	api.AddRouter(router)

	putReq := httptest.NewRequest(http.MethodPut, "/users/1", strings.NewReader("{}"))
	putReq.Header.Set("Content-Type", "application/json")
	putResp := httptest.NewRecorder()
	api.Handler().ServeHTTP(putResp, putReq)
	if putResp.Code != http.StatusNotFound {
		t.Fatalf("expected PUT update route to be absent, got %d body=%s", putResp.Code, putResp.Body.String())
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/users/1", strings.NewReader("{"))
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp := httptest.NewRecorder()
	api.Handler().ServeHTTP(patchResp, patchReq)
	if patchResp.Code != http.StatusBadRequest {
		t.Fatalf("expected PATCH route to bind request before hitting persistence, got %d body=%s", patchResp.Code, patchResp.Body.String())
	}
}
`), 0o644); err != nil {
		t.Fatalf("write route test: %v", err)
	}

	runGoTest(t, dir)
}

func TestGenerateCRUDBuildsWithNativeGORM(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type User struct {
	ID   uint   `+"`json:\"id\"`"+`
	Name string `+"`json:\"name\" crud:\"filter,sort,search\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "User", WithGormX: boolPtr(false)})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "user_crud_gen.go"), content, 0o644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "route_test.go"), []byte(`package demo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ninja "github.com/shijl0925/gin-ninja"
)

func TestGeneratedNativeGORMRoutesUsePatchForPartialUpdates(t *testing.T) {
	api := ninja.New(ninja.Config{})
	router := ninja.NewRouter("/users")
	RegisterUserCRUDRoutes(router)
	api.AddRouter(router)

	putReq := httptest.NewRequest(http.MethodPut, "/users/1", strings.NewReader("{}"))
	putReq.Header.Set("Content-Type", "application/json")
	putResp := httptest.NewRecorder()
	api.Handler().ServeHTTP(putResp, putReq)
	if putResp.Code != http.StatusNotFound {
		t.Fatalf("expected PUT update route to be absent, got %d body=%s", putResp.Code, putResp.Body.String())
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/users/1", strings.NewReader("{"))
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp := httptest.NewRecorder()
	api.Handler().ServeHTTP(patchResp, patchReq)
	if patchResp.Code != http.StatusBadRequest {
		t.Fatalf("expected PATCH route to bind request before hitting persistence, got %d body=%s", patchResp.Code, patchResp.Body.String())
	}
}
`), 0o644); err != nil {
		t.Fatalf("write route test: %v", err)
	}

	runGoTest(t, dir)
}

func TestGenerateCRUDSupportsCommonNamedAndContainerTypes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

import (
	"database/sql"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type Audit struct {
	Source string `+"`json:\"source\"`"+`
}

type Profile struct {
	ID         uint            `+"`json:\"id\"`"+`
	Nickname   sql.NullString  `+"`json:\"nickname\"`"+`
	Aliases    []sql.NullInt64 `+"`json:\"aliases\"`"+`
	Metadata   map[string]any  `+"`json:\"metadata\"`"+`
	Attributes json.RawMessage `+"`json:\"attributes\"`"+`
	Audit      Audit           `+"`json:\"audit\"`"+`
	CreatedAt  time.Time       `+"`json:\"created_at\"`"+`
	DeletedAt  gorm.DeletedAt  `+"`json:\"deleted_at\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Profile"})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile_crud_gen.go"), content, 0o644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "types_test.go"), []byte(`package demo

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
)

func TestGeneratedInputsKeepCommonTypes(t *testing.T) {
	var create CreateProfileInput
	create.Nickname = sql.NullString{}
	create.Aliases = []sql.NullInt64{{}}
	create.Metadata = map[string]any{"source": "seed"}
	create.Attributes = json.RawMessage(`+"`{\"ok\":true}`"+`)
	create.Audit = Audit{Source: "seed"}

	var update UpdateProfileInput
	nickname := sql.NullString{}
	aliases := []sql.NullInt64{{}}
	metadata := map[string]any{"source": "patch"}
	attributes := json.RawMessage(`+"`{\"ok\":true}`"+`)
	audit := Audit{Source: "patch"}
	update.Nickname = &nickname
	update.Aliases = &aliases
	update.Metadata = &metadata
	update.Attributes = &attributes
	update.Audit = &audit

	if got := reflect.TypeOf(update.Nickname).String(); got != "*sql.NullString" {
		t.Fatalf("unexpected nickname update type %q", got)
	}
	if got := reflect.TypeOf(update.Metadata).String(); got != "*map[string]interface {}" {
		t.Fatalf("unexpected metadata update type %q", got)
	}
}
`), 0o644); err != nil {
		t.Fatalf("write types test: %v", err)
	}

	runGoTest(t, dir)
}

func TestGenerateCRUDQueryRelationSupport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type User struct {
	ID   uint   `+"`json:\"id\"`"+`
	Name string `+"`json:\"name\" crud:\"search,sort\"`"+`
}

type Tag struct {
	ID   uint   `+"`json:\"id\"`"+`
	Name string `+"`json:\"name\"`"+`
}

type Task struct {
	ID        uint   `+"`json:\"id\"`"+`
	Title     string `+"`json:\"title\"`"+`
	ProjectID uint   `+"`json:\"project_id\"`"+`
}

type Project struct {
	ID      uint   `+"`json:\"id\"`"+`
	Name    string `+"`json:\"name\" crud:\"filter,sort,search\"`"+`
	Status  string `+"`json:\"status\" crud:\"filter:like,sort,search\"`"+`
	OwnerID uint   `+"`json:\"owner_id\" crud:\"filter,sort\"`"+`
	Owner   User   `+"`gorm:\"foreignKey:OwnerID\" json:\"-\"`"+`
	Tasks   []Task `+"`gorm:\"foreignKey:ProjectID\" json:\"-\"`"+`
	Tags    []Tag  `+"`gorm:\"many2many:project_tags;\" json:\"-\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Project"})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	generated := string(content)

	checks := []string{
		`Name    *string ` + "`form:\"name\" filter:\"name,eq\"`" + ``,
		`Status  *string ` + "`form:\"status\" filter:\"status,like\"`" + ``,
		`OwnerID *uint   ` + "`form:\"owner_id\" filter:\"owner_id,eq\"`" + ``,
		`Sort    string  ` + "`form:\"sort\" order:\"name|status|owner_id\" description:\"Validated sort fields\"`" + ``,
		`Search  string  ` + "`form:\"search\" filter:\"name|status,like\" description:\"Keyword search\"`" + ``,
		`Owner                      *ProjectOwnerOut  ` + "`json:\"owner,omitempty\"`" + ``,
		`Tasks                      []ProjectTasksOut ` + "`json:\"tasks,omitempty\"`" + ``,
		`Tags                       []ProjectTagsOut  ` + "`json:\"tags,omitempty\"`" + ``,
		`TagsIDs []uint ` + "`json:\"tags_ids\"`" + ``,
		`TasksIDs []uint ` + "`json:\"tasks_ids\"`" + ``,
		`query.Preload("Owner")`,
		`query.Preload("Tasks")`,
		`query.Preload("Tags")`,
		`filterOpts, err := filter.BuildOptions(in)`,
		`if err := order.ApplyOrder(query, in); err != nil {`,
		`func syncProjectTagsRelations(db *gorm.DB, item *Project, ids []uint) error {`,
		`func syncProjectTasksRelations(db *gorm.DB, item *Project, ids []uint) error {`,
	}
	for _, check := range checks {
		if !strings.Contains(generated, check) {
			t.Fatalf("generated content missing %q\n%s", check, generated)
		}
	}
}

func TestGenerateCRUDDoesNotTreatHyphenatedGORMValuesAsSkippedFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type User struct {
	ID   uint   `+"`json:\"id\"`"+`
	Name string `+"`json:\"name\"`"+`
}

type Project struct {
	ID      uint   `+"`json:\"id\"`"+`
	OwnerID uint   `+"`json:\"owner_id\"`"+`
	Slug    string `+"`json:\"slug\" gorm:\"index:idx-project-slug\"`"+`
	Owner   User   `+"`gorm:\"foreignKey:OwnerID;comment:owner-link\" json:\"-\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Project"})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	generated := string(content)

	checks := []string{
		`Slug string ` + "`json:\"slug\"`" + ``,
		`Slug *string ` + "`json:\"slug\"`" + ``,
		`*ProjectOwnerOut ` + "`json:\"owner,omitempty\"`" + ``,
		`query.Preload("Owner")`,
	}
	for _, check := range checks {
		if !strings.Contains(generated, check) {
			t.Fatalf("generated content missing %q\n%s", check, generated)
		}
	}
}

func TestGenerateCRUDUsesExplicitForeignKeyFieldName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type User struct {
	ID   uint   `+"`json:\"id\"`"+`
	Name string `+"`json:\"name\"`"+`
}

type Project struct {
	ID       uint `+"`json:\"id\"`"+`
	OwnerKey uint `+"`json:\"owner_key\"`"+`
	Owner    User `+"`gorm:\"foreignKey:OwnerKey\" json:\"-\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Project"})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	generated := string(content)

	if !strings.Contains(generated, "OwnerKey uint `json:\"owner_key\"`") {
		t.Fatalf("expected create input to reuse explicit foreign key field\n%s", generated)
	}
	if strings.Contains(generated, "OwnerID uint `json:\"owner_id\"`") || strings.Contains(generated, "syncProjectOwnerRelation") {
		t.Fatalf("expected explicit foreign key field to avoid synthetic owner_id association input\n%s", generated)
	}
}

func TestGenerateCRUDRejectsBelongsToWithoutForeignKeyField(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type User struct {
	ID uint `+"`json:\"id\"`"+`
}

type Project struct {
	ID    uint `+"`json:\"id\"`"+`
	Owner User `+"`gorm:\"foreignKey:OwnerID\" json:\"-\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	_, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Project"})
	if err == nil || !strings.Contains(err.Error(), `requires exported foreign key field "OwnerID"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateCRUDSupportsRichSelfReferentialRelations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

import "gorm.io/gorm"

type Record struct {
	gorm.Model
	Email    string    `+"`json:\"email\" binding:\"required,email\"`"+`
	Comments []Comment `+"`gorm:\"foreignKey:RecordID\" json:\"-\"`"+`
}

type Comment struct {
	gorm.Model
	RecordID uint      `+"`json:\"record_id\" binding:\"required\"`"+`
	ParentID *uint     `+"`json:\"parent_id\"`"+`
	Body     string    `+"`json:\"body\" binding:\"required\"`"+`
	Record   Record    `+"`gorm:\"foreignKey:RecordID\" json:\"-\"`"+`
	Parent   *Comment  `+"`gorm:\"foreignKey:ParentID\" json:\"-\"`"+`
	Children []Comment `+"`gorm:\"foreignKey:ParentID\" json:\"-\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Comment"})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	generated := string(content)

	checks := []string{
		`RecordID uint ` + "`json:\"record_id\" binding:\"required\"`" + ``,
		`ParentID *uint ` + "`json:\"parent_id\"`" + ``,
		`Record                     *CommentRecordOut`,
		`Parent                     *CommentParentOut`,
		`Children                   []CommentChildrenOut`,
		`query.Preload("Record")`,
		`query.Preload("Parent")`,
		`query.Preload("Children")`,
		`func syncCommentChildrenRelations(db *gorm.DB, item *Comment, ids []uint) error {`,
	}
	for _, check := range checks {
		if !strings.Contains(generated, check) {
			t.Fatalf("generated content missing %q\n%s", check, generated)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "comment_crud_gen.go"), content, 0o644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}
	runGoTest(t, dir)
}

func TestGenerateCRUDRoutesRunForRichSelfReferentialRelations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

import "gorm.io/gorm"

type Record struct {
	gorm.Model
	Email    string    `+"`json:\"email\" binding:\"required,email\"`"+`
	Comments []Comment `+"`gorm:\"foreignKey:RecordID\" json:\"-\"`"+`
}

type Comment struct {
	gorm.Model
	RecordID uint      `+"`json:\"record_id\" binding:\"required\"`"+`
	ParentID *uint     `+"`json:\"parent_id\"`"+`
	Body     string    `+"`json:\"body\" binding:\"required\"`"+`
	Record   Record    `+"`gorm:\"foreignKey:RecordID\" json:\"-\"`"+`
	Parent   *Comment  `+"`gorm:\"foreignKey:ParentID\" json:\"-\"`"+`
	Children []Comment `+"`gorm:\"foreignKey:ParentID\" json:\"-\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	content, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "Comment"})
	if err != nil {
		t.Fatalf("GenerateCRUD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "comment_crud_gen.go"), content, 0o644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "route_runtime_test.go"), []byte(`package demo

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/orm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newGeneratedCommentAPI(t *testing.T) (*ninja.NinjaAPI, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := db.AutoMigrate(&Record{}, &Comment{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	orm.Init(db)

	api := ninja.New(ninja.Config{DisableGinDefault: true})
	api.UseGin(orm.Middleware(db))
	router := ninja.NewRouter("/comments")
	RegisterCommentCRUDRoutes(router)
	api.AddRouter(router)
	return api, db
}

func performGeneratedCommentJSON(t *testing.T, api *ninja.NinjaAPI, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	return rec
}

func TestGeneratedCommentCRUDRoutesRunAgainstSQLite(t *testing.T) {
	api, db := newGeneratedCommentAPI(t)

	record := Record{Email: "record@example.com"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("db.Create record: %v", err)
	}

	rootResp := performGeneratedCommentJSON(t, api, http.MethodPost, "/comments/", map[string]any{
		"record_id": record.ID,
		"body":      "root",
	})
	if rootResp.Code != http.StatusCreated {
		t.Fatalf("create root status=%d body=%s", rootResp.Code, rootResp.Body.String())
	}
	var root map[string]any
	if err := json.Unmarshal(rootResp.Body.Bytes(), &root); err != nil {
		t.Fatalf("json.Unmarshal root: %v", err)
	}
	rootID := uint(root["id"].(float64))
	if got := uint(root["record_id"].(float64)); got != record.ID {
		t.Fatalf("expected record_id %d, got %+v", record.ID, root)
	}

	childResp := performGeneratedCommentJSON(t, api, http.MethodPost, "/comments/", map[string]any{
		"record_id": record.ID,
		"parent_id": rootID,
		"body":      "child",
	})
	if childResp.Code != http.StatusCreated {
		t.Fatalf("create child status=%d body=%s", childResp.Code, childResp.Body.String())
	}
	var child map[string]any
	if err := json.Unmarshal(childResp.Body.Bytes(), &child); err != nil {
		t.Fatalf("json.Unmarshal child: %v", err)
	}
	childID := uint(child["id"].(float64))

	detailResp := performGeneratedCommentJSON(t, api, http.MethodGet, "/comments/"+strconv.FormatUint(uint64(childID), 10), nil)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", detailResp.Code, detailResp.Body.String())
	}
	var detail map[string]any
	if err := json.Unmarshal(detailResp.Body.Bytes(), &detail); err != nil {
		t.Fatalf("json.Unmarshal detail: %v", err)
	}
	if got := uint(detail["parent_id"].(float64)); got != rootID {
		t.Fatalf("expected parent_id %d, got %+v", rootID, detail)
	}

	updateResp := performGeneratedCommentJSON(t, api, http.MethodPatch, "/comments/"+strconv.FormatUint(uint64(rootID), 10), map[string]any{
		"body": "root updated",
	})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}

	listResp := performGeneratedCommentJSON(t, api, http.MethodGet, "/comments/?page=1&size=10", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	var page struct {
		Items []map[string]any `+"`json:\"items\"`"+`
		Total int64            `+"`json:\"total\"`"+`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &page); err != nil {
		t.Fatalf("json.Unmarshal page: %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("expected two comments in page, got %+v", page)
	}

	deleteResp := performGeneratedCommentJSON(t, api, http.MethodDelete, "/comments/"+strconv.FormatUint(uint64(rootID), 10), nil)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}
`), 0o644); err != nil {
		t.Fatalf("write route runtime test: %v", err)
	}

	runGoTest(t, dir)
}

func runGoTest(t *testing.T, dir string) {
	t.Helper()
	requireIntegration(t)

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repo root")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
	goMod := "module demo\n\ngo 1.26\n\nrequire github.com/shijl0925/gin-ninja v0.0.0\n\nreplace github.com/shijl0925/gin-ninja => " + repoRoot + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	modTidy := exec.Command("go", "mod", "tidy")
	modTidy.Dir = dir
	if output, err := modTidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy temp module: %v\n%s", err, output)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test temp module: %v\n%s", err, output)
	}
}
