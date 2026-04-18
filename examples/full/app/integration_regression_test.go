package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	admin "github.com/shijl0925/gin-ninja/admin"
	"github.com/shijl0925/gin-ninja/filter"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
	"github.com/shijl0925/gin-ninja/order"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pagination"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type regressionTag struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"not null" json:"name"`
	Code string `gorm:"uniqueIndex;not null" json:"code"`
}

type regressionRecord struct {
	gorm.Model
	Name       string          `gorm:"not null" json:"name" binding:"required"`
	Email      string          `gorm:"uniqueIndex;not null" json:"email" binding:"required,email"`
	Password   string          `gorm:"not null" json:"password" binding:"required,min=8" admin:"component:password" ninja:"writeOnly"`
	InviteCode string          `gorm:"column:invite_code;type:varchar(64)" json:"invite_code" ninja:"createOnly"`
	StatusNote string          `gorm:"column:status_note;type:text" json:"status_note" crud:"updateOnly"`
	Age        int             `gorm:"not null;default:0" json:"age"`
	Score      float64         `gorm:"not null;default:0" json:"score"`
	Balance    int64           `gorm:"not null;default:0" json:"balance"`
	Active     bool            `gorm:"not null;default:false" json:"active"`
	StartsAt   time.Time       `gorm:"not null" json:"starts_at"`
	ArchivedAt *time.Time      `json:"archived_at"`
	Nickname   *string         `json:"nickname"`
	Level      *int            `json:"level"`
	Verified   *bool           `json:"verified"`
	Tags       []regressionTag `gorm:"many2many:regression_record_tags;" json:"-"`
	TagIDs     []uint          `gorm:"-" json:"tag_ids" admin:"label:Tags;relation:regression_tags"`
}

type regressionRecordTag struct {
	RegressionRecordID uint `gorm:"column:regression_record_id"`
	RegressionTagID    uint `gorm:"column:regression_tag_id"`
}

type regressionRecordOut struct {
	ninja.ModelSchema[regressionRecord] `fields:"id,name,email,invite_code,status_note,age,score,balance,active,starts_at,archived_at,nickname,level,verified,tag_ids" exclude:"password"`
}

type regressionRecordPayload struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	InviteCode string     `json:"invite_code"`
	StatusNote string     `json:"status_note"`
	Age        int        `json:"age"`
	Score      float64    `json:"score"`
	Balance    int64      `json:"balance"`
	Active     bool       `json:"active"`
	StartsAt   time.Time  `json:"starts_at"`
	ArchivedAt *time.Time `json:"archived_at"`
	Nickname   *string    `json:"nickname"`
	Level      *int       `json:"level"`
	Verified   *bool      `json:"verified"`
	TagIDs     []uint     `json:"tag_ids"`
}

type listRegressionRecordsInput struct {
	pagination.PageInput
	Sort   string `form:"sort" order:"id|name|email|age|score|balance|active|starts_at"`
	Search string `form:"search" filter:"name|email,like"`
	Active *bool  `form:"active" filter:"active,eq"`
	MinAge *int   `form:"min_age" filter:"age,ge"`
}

type getRegressionRecordInput struct {
	ID uint `path:"id" binding:"required"`
}

type createRegressionRecordInput struct {
	Name       string     `json:"name" binding:"required"`
	Email      string     `json:"email" binding:"required,email"`
	Password   string     `json:"password" binding:"required,min=8"`
	InviteCode string     `json:"invite_code"`
	Age        int        `json:"age" binding:"omitempty,min=0,max=150"`
	Score      float64    `json:"score"`
	Balance    int64      `json:"balance"`
	Active     bool       `json:"active"`
	StartsAt   time.Time  `json:"starts_at" binding:"required"`
	ArchivedAt *time.Time `json:"archived_at"`
	Nickname   *string    `json:"nickname"`
	Level      *int       `json:"level"`
	Verified   *bool      `json:"verified"`
	TagIDs     []uint     `json:"tag_ids"`
}

type updateRegressionRecordInput struct {
	ID         uint       `path:"id" binding:"required"`
	Name       *string    `json:"name"`
	Email      *string    `json:"email" binding:"omitempty,email"`
	Password   *string    `json:"password" binding:"omitempty,min=8"`
	StatusNote *string    `json:"status_note"`
	Age        *int       `json:"age" binding:"omitempty,min=0,max=150"`
	Score      *float64   `json:"score"`
	Balance    *int64     `json:"balance"`
	Active     *bool      `json:"active"`
	StartsAt   *time.Time `json:"starts_at"`
	ArchivedAt *time.Time `json:"archived_at"`
	Nickname   *string    `json:"nickname"`
	Level      *int       `json:"level"`
	Verified   *bool      `json:"verified"`
	TagIDs     []uint     `json:"tag_ids"`
}

type deleteRegressionRecordInput struct {
	ID uint `path:"id" binding:"required"`
}

func (r *regressionRecord) BeforeSave(*gorm.DB) error {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	r.InviteCode = strings.ToUpper(strings.TrimSpace(r.InviteCode))
	if strings.TrimSpace(r.Password) == "" {
		r.Password = ""
		return nil
	}
	if isHashedPassword(r.Password) {
		return nil
	}
	if len(r.Password) < 8 {
		return ninja.NewErrorWithCode(400, "BAD_REQUEST", `field "password" must be at least 8 characters`)
	}
	r.Password = hashPassword(r.Password)
	return nil
}

func (r *regressionRecord) AfterSave(tx *gorm.DB) error {
	return syncRegressionTags(tx, r, r.TagIDs)
}

func (r *regressionRecord) AfterFind(*gorm.DB) error {
	r.syncTagIDs()
	return nil
}

func (r *regressionRecord) syncTagIDs() {
	if len(r.Tags) == 0 {
		r.TagIDs = nil
		return
	}
	ids := make([]uint, 0, len(r.Tags))
	for _, tag := range r.Tags {
		ids = append(ids, tag.ID)
	}
	r.TagIDs = ids
}

func syncRegressionTags(tx *gorm.DB, record *regressionRecord, tagIDs []uint) error {
	if record == nil || tagIDs == nil {
		return nil
	}
	normalized := make([]uint, 0, len(tagIDs))
	seen := make(map[uint]struct{}, len(tagIDs))
	for _, id := range tagIDs {
		if id == 0 {
			return ninja.NewErrorWithCode(400, "BAD_REQUEST", `field "tag_ids" must not contain zero`)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if err := tx.Where("regression_record_id = ?", record.ID).Delete(&regressionRecordTag{}).Error; err != nil {
		return err
	}
	if len(normalized) == 0 {
		record.Tags = nil
		record.TagIDs = []uint{}
		return nil
	}

	var tags []regressionTag
	if err := tx.Where("id IN ?", normalized).Find(&tags).Error; err != nil {
		return err
	}
	tagByID := make(map[uint]regressionTag, len(tags))
	for _, tag := range tags {
		tagByID[tag.ID] = tag
	}
	ordered := make([]regressionTag, 0, len(normalized))
	links := make([]regressionRecordTag, 0, len(normalized))
	for _, id := range normalized {
		tag, ok := tagByID[id]
		if !ok {
			return ninja.NewErrorWithCode(400, "BAD_REQUEST", "tag "+strconv.FormatUint(uint64(id), 10)+" does not exist")
		}
		ordered = append(ordered, tag)
		links = append(links, regressionRecordTag{RegressionRecordID: record.ID, RegressionTagID: id})
	}
	if err := tx.Create(&links).Error; err != nil {
		return err
	}
	record.Tags = ordered
	record.TagIDs = normalized
	return nil
}

func regressionDB(ctx *ninja.Context) *gorm.DB {
	if ctx != nil && ctx.Context != nil {
		return orm.WithContext(ctx.Context)
	}
	return orm.GetBaseDB(nil)
}

func toRegressionRecordOut(item regressionRecord) (*regressionRecordOut, error) {
	return ninja.BindModelSchema[regressionRecordOut](item)
}

func loadRegressionRecord(db *gorm.DB, id uint) (regressionRecord, error) {
	var item regressionRecord
	err := db.Preload("Tags").First(&item, id).Error
	return item, err
}

func listRegressionRecords(ctx *ninja.Context, in *listRegressionRecordsInput) (*pagination.Page[regressionRecordOut], error) {
	db := regressionDB(ctx).Model(&regressionRecord{})
	var err error
	if db, err = filter.ApplyDB(db, in); err != nil {
		return nil, ninja.NewErrorWithCode(400, "BAD_FILTER", err.Error())
	}
	if db, err = order.ApplyDB(db, in); err != nil {
		return nil, ninja.NewErrorWithCode(400, "BAD_SORT", err.Error())
	}

	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	var items []regressionRecord
	if err := db.Preload("Tags").Offset((in.GetPage() - 1) * in.GetSize()).Limit(in.GetSize()).Find(&items).Error; err != nil {
		return nil, err
	}

	out := make([]regressionRecordOut, len(items))
	for i := range items {
		bound, err := toRegressionRecordOut(items[i])
		if err != nil {
			return nil, err
		}
		out[i] = *bound
	}
	return pagination.NewPage(out, total, in.PageInput), nil
}

func getRegressionRecord(ctx *ninja.Context, in *getRegressionRecordInput) (*regressionRecordOut, error) {
	item, err := loadRegressionRecord(regressionDB(ctx), in.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	return toRegressionRecordOut(item)
}

func createRegressionRecord(ctx *ninja.Context, in *createRegressionRecordInput) (*regressionRecordOut, error) {
	item := regressionRecord{
		Name:       in.Name,
		Email:      in.Email,
		Password:   in.Password,
		InviteCode: in.InviteCode,
		Age:        in.Age,
		Score:      in.Score,
		Balance:    in.Balance,
		Active:     in.Active,
		StartsAt:   in.StartsAt,
		ArchivedAt: in.ArchivedAt,
		Nickname:   in.Nickname,
		Level:      in.Level,
		Verified:   in.Verified,
		TagIDs:     append([]uint(nil), in.TagIDs...),
	}
	db := regressionDB(ctx)
	if err := db.Create(&item).Error; err != nil {
		return nil, err
	}
	loaded, err := loadRegressionRecord(db, item.ID)
	if err != nil {
		return nil, err
	}
	return toRegressionRecordOut(loaded)
}

func updateRegressionRecord(ctx *ninja.Context, in *updateRegressionRecordInput) (*regressionRecordOut, error) {
	db := regressionDB(ctx)
	item, err := loadRegressionRecord(db, in.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}

	if in.Name != nil {
		item.Name = *in.Name
	}
	if in.Email != nil {
		item.Email = *in.Email
	}
	if in.Password != nil {
		item.Password = *in.Password
	}
	if in.StatusNote != nil {
		item.StatusNote = *in.StatusNote
	}
	if in.Age != nil {
		item.Age = *in.Age
	}
	if in.Score != nil {
		item.Score = *in.Score
	}
	if in.Balance != nil {
		item.Balance = *in.Balance
	}
	if in.Active != nil {
		item.Active = *in.Active
	}
	if in.StartsAt != nil {
		item.StartsAt = *in.StartsAt
	}
	if in.ArchivedAt != nil {
		item.ArchivedAt = in.ArchivedAt
	}
	if in.Nickname != nil {
		item.Nickname = in.Nickname
	}
	if in.Level != nil {
		item.Level = in.Level
	}
	if in.Verified != nil {
		item.Verified = in.Verified
	}
	if in.TagIDs != nil {
		item.TagIDs = append([]uint(nil), in.TagIDs...)
	}
	if err := db.Save(&item).Error; err != nil {
		return nil, err
	}
	loaded, err := loadRegressionRecord(db, item.ID)
	if err != nil {
		return nil, err
	}
	return toRegressionRecordOut(loaded)
}

func deleteRegressionRecord(ctx *ninja.Context, in *deleteRegressionRecordInput) error {
	db := regressionDB(ctx)
	if _, err := loadRegressionRecord(db, in.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ninja.NotFoundError()
		}
		return err
	}
	return db.Delete(&regressionRecord{}, in.ID).Error
}

func newRegressionAdminSite() *admin.Site {
	site := admin.NewSite(admin.WithPermissionChecker(requireAuthenticatedAdmin))
	site.MustRegisterModel(&admin.ModelResource{
		Name:         "regression_tags",
		Path:         "/regression-tags",
		Model:        regressionTag{},
		ListFields:   []string{"id", "name", "code"},
		DetailFields: []string{"id", "name", "code"},
		CreateFields: []string{"name", "code"},
		UpdateFields: []string{"name", "code"},
		FilterFields: []string{"id", "code"},
		SortFields:   []string{"id", "name", "code"},
		SearchFields: []string{"name", "code"},
	})
	site.MustRegisterModel(&admin.ModelResource{
		Name:         "regression_records",
		Path:         "/regression-records",
		Model:        regressionRecord{},
		Preloads:     []string{"Tags"},
		ListFields:   []string{"id", "name", "email", "invite_code", "status_note", "age", "score", "balance", "active", "starts_at", "nickname", "level", "verified", "tag_ids"},
		DetailFields: []string{"id", "name", "email", "invite_code", "status_note", "age", "score", "balance", "active", "starts_at", "archived_at", "nickname", "level", "verified", "tag_ids"},
		CreateFields: []string{"name", "email", "password", "invite_code", "age", "score", "balance", "active", "starts_at", "archived_at", "nickname", "level", "verified", "tag_ids"},
		UpdateFields: []string{"name", "email", "password", "status_note", "age", "score", "balance", "active", "starts_at", "archived_at", "nickname", "level", "verified", "tag_ids"},
		FilterFields: []string{"active", "age", "starts_at"},
		SortFields:   []string{"id", "name", "email", "age", "score", "balance", "active", "starts_at"},
		SearchFields: []string{"name", "email"},
	})
	return site
}

func newRegressionIntegrationAPI(t *testing.T) (*ninja.NinjaAPI, *gorm.DB) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := db.AutoMigrate(&regressionTag{}, &regressionRecord{}, &regressionRecordTag{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	orm.Init(db)

	api := ninja.New(ninja.Config{
		Title:             "regression integration",
		Version:           "test",
		Prefix:            "/api",
		DisableGinDefault: true,
	})
	orm.RegisterDefaultErrorMappers(api)
	api.UseGin(orm.Middleware(db))

	crudRouter := ninja.NewRouter("/regression-records", ninja.WithTags("Regression"))
	ninja.Get(crudRouter, "/", listRegressionRecords, ninja.Paginated[regressionRecordOut]())
	ninja.Get(crudRouter, "/:id", getRegressionRecord)
	ninja.Post(crudRouter, "/", createRegressionRecord, ninja.WithTransaction())
	ninja.Patch(crudRouter, "/:id", updateRegressionRecord, ninja.WithTransaction())
	ninja.Delete(crudRouter, "/:id", deleteRegressionRecord, ninja.WithTransaction())
	api.AddRouter(crudRouter)

	adminRouter := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
	adminRouter.UseGin(func(c *gin.Context) {
		if raw := strings.TrimSpace(c.GetHeader("X-User-ID")); raw != "" {
			id, _ := strconv.Atoi(raw)
			c.Set(contextkeys.JWTClaims, exampleAdminClaims{id: uint(id)})
		}
		c.Next()
	})
	newRegressionAdminSite().Mount(adminRouter)
	api.AddRouter(adminRouter)

	return api, db
}

func performRegressionJSON(t *testing.T, api *ninja.NinjaAPI, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
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
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(recorder, req)
	return recorder
}

func decodeJSONBody[T any](t *testing.T, body *bytes.Buffer) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(body.Bytes(), &out); err != nil {
		t.Fatalf("json.Unmarshal: %v body=%s", err, body.String())
	}
	return out
}

func seedRegressionTags(t *testing.T, db *gorm.DB) []regressionTag {
	t.Helper()
	tags := []regressionTag{
		{Name: "Alpha", Code: "alpha"},
		{Name: "Beta", Code: "beta"},
		{Name: "Gamma", Code: "gamma"},
	}
	for i := range tags {
		if err := db.Create(&tags[i]).Error; err != nil {
			t.Fatalf("create tag %d: %v", i, err)
		}
	}
	return tags
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestRegressionCRUDRoutesExerciseRichModelScenarios(t *testing.T) {
	api, db := newRegressionIntegrationAPI(t)
	tags := seedRegressionTags(t, db)

	nickname := " Ally "
	level := 7
	verified := true
	startsAt := time.Date(2026, 4, 18, 8, 30, 0, 0, time.UTC)
	archivedAt := startsAt.Add(2 * time.Hour)

	createResp := performRegressionJSON(t, api, http.MethodPost, "/api/regression-records/", createRegressionRecordInput{
		Name:       "  Alpha Runner  ",
		Email:      "  ALPHA@EXAMPLE.COM  ",
		Password:   "password123",
		InviteCode: "invite-me",
		Age:        28,
		Score:      98.75,
		Balance:    123456789,
		Active:     true,
		StartsAt:   startsAt,
		ArchivedAt: &archivedAt,
		Nickname:   &nickname,
		Level:      &level,
		Verified:   &verified,
		TagIDs:     []uint{tags[1].ID, tags[1].ID, tags[0].ID},
	}, nil)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	created := decodeJSONBody[regressionRecordPayload](t, createResp.Body)
	if created.Name != "Alpha Runner" || created.Email != "alpha@example.com" || created.InviteCode != "INVITE-ME" {
		t.Fatalf("unexpected create payload: %+v", created)
	}
	if created.StatusNote != "" || created.Score != 98.75 || created.Balance != 123456789 || !created.Active {
		t.Fatalf("unexpected create payload: %+v", created)
	}
	if len(created.TagIDs) != 2 || created.TagIDs[0] != tags[1].ID || created.TagIDs[1] != tags[0].ID {
		t.Fatalf("expected normalized tag ids, got %+v", created)
	}
	var rawCreate map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &rawCreate); err != nil {
		t.Fatalf("json.Unmarshal raw create: %v", err)
	}
	if _, ok := rawCreate["password"]; ok {
		t.Fatalf("expected password to be hidden, got %+v", rawCreate)
	}

	var stored regressionRecord
	if err := db.Preload("Tags").First(&stored, created.ID).Error; err != nil {
		t.Fatalf("db.First: %v", err)
	}
	if !isHashedPassword(stored.Password) || stored.Password == "password123" {
		t.Fatalf("expected hashed password, got %q", stored.Password)
	}
	if len(stored.Tags) != 2 {
		t.Fatalf("expected 2 tags after create, got %d", len(stored.Tags))
	}

	if got := performRegressionJSON(t, api, http.MethodPost, "/api/regression-records/", createRegressionRecordInput{
		Name:     "Other",
		Email:    "not-an-email",
		Password: "password123",
		StartsAt: startsAt,
	}, nil); got.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid email 400, got %d body=%s", got.Code, got.Body.String())
	}

	secondResp := performRegressionJSON(t, api, http.MethodPost, "/api/regression-records/", createRegressionRecordInput{
		Name:       "Beta Person",
		Email:      "beta@example.com",
		Password:   "password123",
		InviteCode: "beta",
		Age:        34,
		Score:      55.5,
		Balance:    99,
		Active:     false,
		StartsAt:   startsAt.Add(24 * time.Hour),
		TagIDs:     []uint{tags[2].ID},
	}, nil)
	if secondResp.Code != http.StatusCreated {
		t.Fatalf("second create status=%d body=%s", secondResp.Code, secondResp.Body.String())
	}
	second := decodeJSONBody[regressionRecordPayload](t, secondResp.Body)

	active := true
	minAge := 20
	listResp := performRegressionJSON(t, api, http.MethodGet, "/api/regression-records/?search=alpha&active=true&min_age=20&sort=-score&page=1&size=5", nil, nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	listPage := decodeJSONBody[pagination.Page[regressionRecordPayload]](t, listResp.Body)
	if listPage.Total != 1 || len(listPage.Items) != 1 || listPage.Items[0].ID != created.ID {
		t.Fatalf("unexpected filtered list: %+v active=%v minAge=%v", listPage, active, minAge)
	}

	injectionResp := performRegressionJSON(t, api, http.MethodGet, "/api/regression-records/?search=%27%20OR%201%3D1%20--&page=1&size=5", nil, nil)
	if injectionResp.Code != http.StatusOK {
		t.Fatalf("injection list status=%d body=%s", injectionResp.Code, injectionResp.Body.String())
	}
	injectionPage := decodeJSONBody[pagination.Page[regressionRecordPayload]](t, injectionResp.Body)
	if injectionPage.Total != 0 || len(injectionPage.Items) != 0 {
		t.Fatalf("expected injection-like search to return no rows, got %+v", injectionPage)
	}

	detailResp := performRegressionJSON(t, api, http.MethodGet, "/api/regression-records/"+strconv.FormatUint(uint64(created.ID), 10), nil, nil)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", detailResp.Code, detailResp.Body.String())
	}
	detail := decodeJSONBody[regressionRecordPayload](t, detailResp.Body)
	if detail.Email != "alpha@example.com" || detail.ArchivedAt == nil || detail.StartsAt.UTC() != startsAt {
		t.Fatalf("unexpected detail payload: %+v", detail)
	}

	newName := "Alpha Updated"
	newEmail := "alpha.updated@example.com"
	newPassword := "updatedpass123"
	statusNote := "synced from patch"
	newAge := 29
	newScore := 77.25
	newBalance := int64(987654321)
	newActive := false
	newStartsAt := startsAt.Add(48 * time.Hour)
	newNickname := "Updated"
	newLevel := 9
	newVerified := false
	updateResp := performRegressionJSON(t, api, http.MethodPatch, "/api/regression-records/"+strconv.FormatUint(uint64(created.ID), 10), updateRegressionRecordInput{
		Name:       &newName,
		Email:      &newEmail,
		Password:   &newPassword,
		StatusNote: &statusNote,
		Age:        &newAge,
		Score:      &newScore,
		Balance:    &newBalance,
		Active:     &newActive,
		StartsAt:   &newStartsAt,
		Nickname:   &newNickname,
		Level:      &newLevel,
		Verified:   &newVerified,
		TagIDs:     []uint{tags[2].ID},
	}, nil)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	updated := decodeJSONBody[regressionRecordPayload](t, updateResp.Body)
	if updated.Name != newName || updated.Email != newEmail || updated.StatusNote != statusNote || updated.Active {
		t.Fatalf("unexpected update payload: %+v", updated)
	}
	if len(updated.TagIDs) != 1 || updated.TagIDs[0] != tags[2].ID {
		t.Fatalf("expected updated tags, got %+v", updated)
	}

	var reloaded regressionRecord
	if err := db.Preload("Tags").First(&reloaded, created.ID).Error; err != nil {
		t.Fatalf("reload updated: %v", err)
	}
	if reloaded.InviteCode != "INVITE-ME" || reloaded.StatusNote != statusNote || !isHashedPassword(reloaded.Password) || checkPassword(stored.Password, newPassword) {
		t.Fatalf("unexpected stored update state: %+v", reloaded)
	}
	if !checkPassword(reloaded.Password, newPassword) {
		t.Fatalf("expected updated password hash to match new password")
	}
	if len(reloaded.Tags) != 1 || reloaded.Tags[0].ID != tags[2].ID {
		t.Fatalf("expected updated tag relation, got %+v", reloaded.Tags)
	}

	badTagResp := performRegressionJSON(t, api, http.MethodPatch, "/api/regression-records/"+strconv.FormatUint(uint64(created.ID), 10), updateRegressionRecordInput{
		TagIDs: []uint{999999},
	}, nil)
	if badTagResp.Code != http.StatusBadRequest {
		t.Fatalf("expected missing tag 400, got %d body=%s", badTagResp.Code, badTagResp.Body.String())
	}

	deleteResp := performRegressionJSON(t, api, http.MethodDelete, "/api/regression-records/"+strconv.FormatUint(uint64(created.ID), 10), nil, nil)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	missingResp := performRegressionJSON(t, api, http.MethodGet, "/api/regression-records/"+strconv.FormatUint(uint64(created.ID), 10), nil, nil)
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("expected deleted detail 404, got %d body=%s", missingResp.Code, missingResp.Body.String())
	}

	finalListResp := performRegressionJSON(t, api, http.MethodGet, "/api/regression-records/?sort=-age&page=1&size=5", nil, nil)
	if finalListResp.Code != http.StatusOK {
		t.Fatalf("final list status=%d body=%s", finalListResp.Code, finalListResp.Body.String())
	}
	finalPage := decodeJSONBody[pagination.Page[regressionRecordPayload]](t, finalListResp.Body)
	if finalPage.Total != 1 || len(finalPage.Items) != 1 || finalPage.Items[0].ID != second.ID {
		t.Fatalf("unexpected final page: %+v", finalPage)
	}
}

func TestRegressionAdminCRUDAndMetadataExerciseRichModelScenarios(t *testing.T) {
	api, db := newRegressionIntegrationAPI(t)
	tags := seedRegressionTags(t, db)
	headers := map[string]string{"X-User-ID": "1"}

	metaResp := performRegressionJSON(t, api, http.MethodGet, "/admin/resources/regression_records/meta", nil, headers)
	if metaResp.Code != http.StatusOK {
		t.Fatalf("meta status=%d body=%s", metaResp.Code, metaResp.Body.String())
	}
	meta := decodeJSONBody[admin.ResourceMetadata](t, metaResp.Body)
	if !containsString(meta.CreateFields, "invite_code") || containsString(meta.UpdateFields, "invite_code") {
		t.Fatalf("expected invite_code create-only metadata, got %+v", meta)
	}
	if containsString(meta.CreateFields, "status_note") || !containsString(meta.UpdateFields, "status_note") {
		t.Fatalf("expected status_note update-only metadata, got %+v", meta)
	}
	var passwordField, tagIDsField *admin.FieldMeta
	for i := range meta.Fields {
		switch meta.Fields[i].Name {
		case "password":
			passwordField = &meta.Fields[i]
		case "tag_ids":
			tagIDsField = &meta.Fields[i]
		}
	}
	if passwordField == nil || passwordField.List || passwordField.Detail || !passwordField.Create || !passwordField.Update || passwordField.Component != "password" {
		t.Fatalf("unexpected password metadata: %+v", passwordField)
	}
	if tagIDsField == nil || tagIDsField.Relation == nil || tagIDsField.Relation.Resource != "regression_tags" {
		t.Fatalf("unexpected tag_ids metadata: %+v", tagIDsField)
	}

	startsAt := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	archivedAt := startsAt.Add(3 * time.Hour)
	createDisallowed := performRegressionJSON(t, api, http.MethodPost, "/admin/resources/regression_records", map[string]any{
		"name":        "Admin User",
		"email":       "admin@example.com",
		"password":    "password123",
		"invite_code": "admin-create",
		"status_note": "should fail",
		"starts_at":   startsAt,
	}, headers)
	if createDisallowed.Code != http.StatusBadRequest {
		t.Fatalf("expected update-only field create rejection, got %d body=%s", createDisallowed.Code, createDisallowed.Body.String())
	}

	createResp := performRegressionJSON(t, api, http.MethodPost, "/admin/resources/regression_records", map[string]any{
		"name":        "  Admin User  ",
		"email":       "ADMIN@EXAMPLE.COM",
		"password":    "password123",
		"invite_code": "admin-create",
		"age":         41,
		"score":       61.5,
		"balance":     8888,
		"active":      true,
		"starts_at":   startsAt.Format(time.RFC3339),
		"archived_at": archivedAt.Format(time.RFC3339),
		"nickname":    "Boss",
		"level":       12,
		"verified":    true,
		"tag_ids":     []uint{tags[0].ID, tags[2].ID, tags[2].ID},
	}, headers)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("admin create status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	created := decodeJSONBody[admin.ResourceRecordOutput](t, createResp.Body)
	if created.Item["email"] != "admin@example.com" || created.Item["invite_code"] != "ADMIN-CREATE" {
		t.Fatalf("unexpected admin create item: %+v", created.Item)
	}
	if _, ok := created.Item["password"]; ok {
		t.Fatalf("expected password hidden in admin response, got %+v", created.Item)
	}
	tagValues, ok := created.Item["tag_ids"].([]any)
	if !ok || len(tagValues) != 2 {
		t.Fatalf("expected normalized tag ids, got %+v", created.Item["tag_ids"])
	}

	listResp := performRegressionJSON(t, api, http.MethodGet, "/admin/resources/regression_records?search=admin&active=true&sort=-age", nil, headers)
	if listResp.Code != http.StatusOK {
		t.Fatalf("admin list status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	list := decodeJSONBody[admin.ResourceListOutput](t, listResp.Body)
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0]["email"] != "admin@example.com" {
		t.Fatalf("unexpected admin list: %+v", list)
	}

	idValue, ok := created.Item["id"].(float64)
	if !ok {
		t.Fatalf("expected numeric id in admin create item, got %+v", created.Item["id"])
	}
	id := uint(idValue)

	updateDisallowed := performRegressionJSON(t, api, http.MethodPut, "/admin/resources/regression_records/"+strconv.FormatUint(uint64(id), 10), map[string]any{
		"invite_code": "should-fail",
	}, headers)
	if updateDisallowed.Code != http.StatusBadRequest {
		t.Fatalf("expected create-only field update rejection, got %d body=%s", updateDisallowed.Code, updateDisallowed.Body.String())
	}

	updateResp := performRegressionJSON(t, api, http.MethodPut, "/admin/resources/regression_records/"+strconv.FormatUint(uint64(id), 10), map[string]any{
		"name":        "Admin Updated",
		"email":       "admin.updated@example.com",
		"password":    "updatedpass123",
		"status_note": "updated via admin",
		"age":         42,
		"score":       62.5,
		"balance":     9999,
		"active":      false,
		"starts_at":   startsAt.Add(24 * time.Hour).Format(time.RFC3339),
		"tag_ids":     []uint{tags[1].ID},
	}, headers)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("admin update status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	updated := decodeJSONBody[admin.ResourceRecordOutput](t, updateResp.Body)
	if updated.Item["status_note"] != "updated via admin" || updated.Item["invite_code"] != "ADMIN-CREATE" {
		t.Fatalf("unexpected admin update item: %+v", updated.Item)
	}

	var stored regressionRecord
	if err := db.Preload("Tags").First(&stored, id).Error; err != nil {
		t.Fatalf("reload admin updated: %v", err)
	}
	if stored.Email != "admin.updated@example.com" || stored.InviteCode != "ADMIN-CREATE" || !checkPassword(stored.Password, "updatedpass123") {
		t.Fatalf("unexpected stored admin state: %+v", stored)
	}
	if len(stored.Tags) != 1 || stored.Tags[0].ID != tags[1].ID {
		t.Fatalf("expected admin-updated tag relation, got %+v", stored.Tags)
	}

	detailResp := performRegressionJSON(t, api, http.MethodGet, "/admin/resources/regression_records/"+strconv.FormatUint(uint64(id), 10), nil, headers)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("admin detail status=%d body=%s", detailResp.Code, detailResp.Body.String())
	}
	detail := decodeJSONBody[admin.ResourceRecordOutput](t, detailResp.Body)
	if detail.Item["email"] != "admin.updated@example.com" || detail.Item["status_note"] != "updated via admin" {
		t.Fatalf("unexpected admin detail item: %+v", detail.Item)
	}

	deleteResp := performRegressionJSON(t, api, http.MethodDelete, "/admin/resources/regression_records/"+strconv.FormatUint(uint64(id), 10), nil, headers)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("admin delete status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	missingResp := performRegressionJSON(t, api, http.MethodGet, "/admin/resources/regression_records/"+strconv.FormatUint(uint64(id), 10), nil, headers)
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("expected deleted admin detail 404, got %d body=%s", missingResp.Code, missingResp.Body.String())
	}
}
