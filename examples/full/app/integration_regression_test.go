package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
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
	Name       string              `gorm:"not null" json:"name" binding:"required"`
	Email      string              `gorm:"uniqueIndex;not null" json:"email" binding:"required,email"`
	Password   string              `gorm:"not null" json:"password" binding:"required,min=8" admin:"component:password" ninja:"writeOnly"`
	InviteCode string              `gorm:"column:invite_code;type:varchar(64)" json:"invite_code" ninja:"createOnly"`
	StatusNote string              `gorm:"column:status_note;type:text" json:"status_note" crud:"updateOnly"`
	Age        int                 `gorm:"not null;default:0" json:"age"`
	Score      float64             `gorm:"not null;default:0" json:"score"`
	Balance    int64               `gorm:"not null;default:0" json:"balance"`
	Active     bool                `gorm:"not null;default:false" json:"active"`
	StartsAt   time.Time           `gorm:"not null" json:"starts_at"`
	ArchivedAt *time.Time          `json:"archived_at"`
	Nickname   *string             `json:"nickname"`
	Level      *int                `json:"level"`
	Verified   *bool               `json:"verified"`
	Tags       []regressionTag     `gorm:"many2many:regression_record_tags;" json:"-"`
	Comments   []regressionComment `gorm:"foreignKey:RecordID" json:"-"`
	TagIDs     []uint              `gorm:"-" json:"tag_ids" admin:"label:Tags;relation:regression_tags"`
	CommentIDs []uint              `gorm:"-" json:"comment_ids" admin:"label:Comments;relation:regression_comments;readonly"`
}

type regressionRecordTag struct {
	RegressionRecordID uint `gorm:"column:regression_record_id"`
	RegressionTagID    uint `gorm:"column:regression_tag_id"`
}

type regressionComment struct {
	gorm.Model
	RecordID uint                `gorm:"column:record_id;not null;index" json:"record_id" binding:"required" admin:"label:Record;relation:regression_records"`
	ParentID *uint               `gorm:"column:parent_id;index" json:"parent_id" admin:"label:Parent;relation:regression_comments"`
	Body     string              `gorm:"column:body;type:text;not null" json:"body" binding:"required"`
	IsPinned bool                `gorm:"column:is_pinned;not null;default:false" json:"is_pinned"`
	Record   regressionRecord    `gorm:"foreignKey:RecordID" json:"-"`
	Parent   *regressionComment  `gorm:"foreignKey:ParentID" json:"-"`
	Children []regressionComment `gorm:"foreignKey:ParentID" json:"-"`
	ChildIDs []uint              `gorm:"-" json:"child_ids" admin:"label:Children;relation:regression_comments;readonly"`
}

type regressionRecordOut struct {
	ninja.ModelSchema[regressionRecord] `fields:"id,name,email,invite_code,status_note,age,score,balance,active,starts_at,archived_at,nickname,level,verified,tag_ids,comment_ids" exclude:"password"`
}

type regressionCommentOut struct {
	ninja.ModelSchema[regressionComment] `fields:"id,record_id,parent_id,body,is_pinned,child_ids"`
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
	CommentIDs []uint     `json:"comment_ids"`
}

type regressionCommentPayload struct {
	ID       uint   `json:"id"`
	RecordID uint   `json:"record_id"`
	ParentID *uint  `json:"parent_id"`
	Body     string `json:"body"`
	IsPinned bool   `json:"is_pinned"`
	ChildIDs []uint `json:"child_ids"`
}

type listRegressionRecordsInput struct {
	pagination.PageInput
	Sort   string `form:"sort" order:"id|name|email|age|score|balance|active|starts_at"`
	Search string `form:"search" filter:"name|email,like"`
	Active *bool  `form:"active" filter:"active,eq"`
	MinAge *int   `form:"min_age" filter:"age,ge"`
}

type getRegressionRecordInput struct {
	ID uint `path:"id" json:"-" binding:"required"`
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
	ID         uint       `path:"id" json:"-" binding:"required"`
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
	ID uint `path:"id" json:"-" binding:"required"`
}

type listRegressionCommentsInput struct {
	pagination.PageInput
	Sort     string `form:"sort" order:"id|record_id|parent_id|body|is_pinned|created_at"`
	Search   string `form:"search" filter:"body,like"`
	RecordID *uint  `form:"record_id" filter:"record_id,eq"`
	ParentID *uint  `form:"parent_id" filter:"parent_id,eq"`
}

type getRegressionCommentInput struct {
	ID uint `path:"id" json:"-" binding:"required"`
}

type createRegressionCommentInput struct {
	RecordID uint   `json:"record_id" binding:"required"`
	ParentID *uint  `json:"parent_id"`
	Body     string `json:"body" binding:"required"`
	IsPinned bool   `json:"is_pinned"`
}

type updateRegressionCommentInput struct {
	ID       uint    `path:"id" json:"-" binding:"required"`
	ParentID *uint   `json:"parent_id"`
	Body     *string `json:"body"`
	IsPinned *bool   `json:"is_pinned"`
}

type deleteRegressionCommentInput struct {
	ID uint `path:"id" json:"-" binding:"required"`
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
	r.syncCommentIDs()
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

func (r *regressionRecord) syncCommentIDs() {
	if len(r.Comments) == 0 {
		r.CommentIDs = nil
		return
	}
	ids := make([]uint, 0, len(r.Comments))
	for _, comment := range r.Comments {
		ids = append(ids, comment.ID)
	}
	r.CommentIDs = ids
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

func (c *regressionComment) BeforeSave(*gorm.DB) error {
	c.Body = strings.TrimSpace(c.Body)
	if c.Body == "" {
		return ninja.NewErrorWithCode(400, "BAD_REQUEST", `field "body" is required`)
	}
	return nil
}

func (c *regressionComment) AfterFind(*gorm.DB) error {
	c.syncChildIDs()
	return nil
}

func (c *regressionComment) syncChildIDs() {
	if len(c.Children) == 0 {
		c.ChildIDs = nil
		return
	}
	ids := make([]uint, 0, len(c.Children))
	for _, child := range c.Children {
		ids = append(ids, child.ID)
	}
	c.ChildIDs = ids
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

func toRegressionCommentOut(item regressionComment) (*regressionCommentOut, error) {
	return ninja.BindModelSchema[regressionCommentOut](item)
}

func loadRegressionRecord(db *gorm.DB, id uint) (regressionRecord, error) {
	var item regressionRecord
	err := db.Preload("Tags").Preload("Comments").First(&item, id).Error
	return item, err
}

func loadRegressionComment(db *gorm.DB, id uint) (regressionComment, error) {
	var item regressionComment
	err := db.Preload("Children").First(&item, id).Error
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

func listRegressionComments(ctx *ninja.Context, in *listRegressionCommentsInput) (*pagination.Page[regressionCommentOut], error) {
	db := regressionDB(ctx).Model(&regressionComment{})
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

	var items []regressionComment
	if err := db.Preload("Children").Offset((in.GetPage() - 1) * in.GetSize()).Limit(in.GetSize()).Find(&items).Error; err != nil {
		return nil, err
	}

	out := make([]regressionCommentOut, len(items))
	for i := range items {
		bound, err := toRegressionCommentOut(items[i])
		if err != nil {
			return nil, err
		}
		out[i] = *bound
	}
	return pagination.NewPage(out, total, in.PageInput), nil
}

func getRegressionComment(ctx *ninja.Context, in *getRegressionCommentInput) (*regressionCommentOut, error) {
	item, err := loadRegressionComment(regressionDB(ctx), in.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	return toRegressionCommentOut(item)
}

func createRegressionComment(ctx *ninja.Context, in *createRegressionCommentInput) (*regressionCommentOut, error) {
	item := regressionComment{
		RecordID: in.RecordID,
		ParentID: in.ParentID,
		Body:     in.Body,
		IsPinned: in.IsPinned,
	}
	db := regressionDB(ctx)
	if err := ensureRegressionCommentRelations(db, item.RecordID, item.ParentID, 0); err != nil {
		return nil, err
	}
	if err := db.Create(&item).Error; err != nil {
		return nil, err
	}
	loaded, err := loadRegressionComment(db, item.ID)
	if err != nil {
		return nil, err
	}
	return toRegressionCommentOut(loaded)
}

func updateRegressionComment(ctx *ninja.Context, in *updateRegressionCommentInput) (*regressionCommentOut, error) {
	db := regressionDB(ctx)
	item, err := loadRegressionComment(db, in.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	item.ParentID = in.ParentID
	if in.Body != nil {
		item.Body = *in.Body
	}
	if in.IsPinned != nil {
		item.IsPinned = *in.IsPinned
	}
	if err := ensureRegressionCommentRelations(db, item.RecordID, item.ParentID, item.ID); err != nil {
		return nil, err
	}
	if err := db.Save(&item).Error; err != nil {
		return nil, err
	}
	loaded, err := loadRegressionComment(db, item.ID)
	if err != nil {
		return nil, err
	}
	return toRegressionCommentOut(loaded)
}

func deleteRegressionComment(ctx *ninja.Context, in *deleteRegressionCommentInput) error {
	db := regressionDB(ctx)
	if _, err := loadRegressionComment(db, in.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ninja.NotFoundError()
		}
		return err
	}
	return db.Delete(&regressionComment{}, in.ID).Error
}

func ensureRegressionCommentRelations(db *gorm.DB, recordID uint, parentID *uint, selfID uint) error {
	if db == nil {
		return nil
	}
	if _, err := loadRegressionRecord(db, recordID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ninja.NewErrorWithCode(400, "BAD_REQUEST", fmt.Sprintf("record %d does not exist", recordID))
		}
		return err
	}
	if parentID == nil {
		return nil
	}
	if *parentID == selfID && selfID != 0 {
		return ninja.NewErrorWithCode(400, "BAD_REQUEST", `field "parent_id" must not reference itself`)
	}
	parent, err := loadRegressionComment(db, *parentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ninja.NewErrorWithCode(400, "BAD_REQUEST", fmt.Sprintf("comment %d does not exist", *parentID))
		}
		return err
	}
	if parent.RecordID != recordID {
		return ninja.NewErrorWithCode(400, "BAD_REQUEST", `field "parent_id" must reference a comment in the same record`)
	}
	return nil
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
		Preloads:     []string{"Tags", "Comments"},
		ListFields:   []string{"id", "name", "email", "invite_code", "status_note", "age", "score", "balance", "active", "starts_at", "nickname", "level", "verified", "tag_ids", "comment_ids"},
		DetailFields: []string{"id", "name", "email", "invite_code", "status_note", "age", "score", "balance", "active", "starts_at", "archived_at", "nickname", "level", "verified", "tag_ids", "comment_ids"},
		CreateFields: []string{"name", "email", "password", "invite_code", "age", "score", "balance", "active", "starts_at", "archived_at", "nickname", "level", "verified", "tag_ids"},
		UpdateFields: []string{"name", "email", "password", "status_note", "age", "score", "balance", "active", "starts_at", "archived_at", "nickname", "level", "verified", "tag_ids"},
		FilterFields: []string{"active", "age", "starts_at"},
		SortFields:   []string{"id", "name", "email", "age", "score", "balance", "active", "starts_at"},
		SearchFields: []string{"name", "email"},
	})
	site.MustRegisterModel(&admin.ModelResource{
		Name:         "regression_comments",
		Path:         "/regression-comments",
		Model:        regressionComment{},
		Preloads:     []string{"Children"},
		ListFields:   []string{"id", "record_id", "parent_id", "body", "is_pinned", "child_ids"},
		DetailFields: []string{"id", "record_id", "parent_id", "body", "is_pinned", "child_ids", "createdAt", "updatedAt"},
		CreateFields: []string{"record_id", "parent_id", "body", "is_pinned"},
		UpdateFields: []string{"parent_id", "body", "is_pinned"},
		FilterFields: []string{"record_id", "parent_id", "is_pinned"},
		SortFields:   []string{"id", "record_id", "parent_id", "body", "is_pinned", "createdAt"},
		SearchFields: []string{"body"},
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
	prepareRegressionSchema(t, db)
	return newRegressionIntegrationAPIWithDB(t, db), db
}

func newRegressionIntegrationAPIWithDB(t *testing.T, db *gorm.DB) *ninja.NinjaAPI {
	t.Helper()

	orm.Init(db)
	api := ninja.New(ninja.Config{
		Title:             "regression integration",
		Version:           "test",
		DisableGinDefault: true,
	})
	orm.RegisterDefaultErrorMappers(api)
	api.UseGin(orm.Middleware(db))

	recordRouter := ninja.NewRouter("/api/regression-records", ninja.WithTags("Regression"))
	ninja.Get(recordRouter, "/", listRegressionRecords, ninja.Paginated[regressionRecordOut]())
	ninja.Get(recordRouter, "/:id", getRegressionRecord)
	ninja.Post(recordRouter, "/", createRegressionRecord, ninja.WithTransaction())
	ninja.Patch(recordRouter, "/:id", updateRegressionRecord, ninja.WithTransaction())
	ninja.Delete(recordRouter, "/:id", deleteRegressionRecord, ninja.WithTransaction())
	api.AddRouter(recordRouter)

	commentRouter := ninja.NewRouter("/api/regression-comments", ninja.WithTags("Regression Comments"))
	ninja.Get(commentRouter, "/", listRegressionComments, ninja.Paginated[regressionCommentOut]())
	ninja.Get(commentRouter, "/:id", getRegressionComment)
	ninja.Post(commentRouter, "/", createRegressionComment, ninja.WithTransaction())
	ninja.Patch(commentRouter, "/:id", updateRegressionComment, ninja.WithTransaction())
	ninja.Delete(commentRouter, "/:id", deleteRegressionComment, ninja.WithTransaction())
	api.AddRouter(commentRouter)

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
	return api
}

func prepareRegressionSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	if db == nil {
		t.Fatal("db must not be nil")
	}
	if err := db.Migrator().DropTable(&regressionComment{}, &regressionRecordTag{}, &regressionRecord{}, &regressionTag{}); err != nil {
		t.Fatalf("DropTable: %v", err)
	}
	if err := db.AutoMigrate(&regressionTag{}, &regressionRecord{}, &regressionRecordTag{}, &regressionComment{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
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

func sameUintSet(got []uint, want ...uint) bool {
	if len(got) != len(want) {
		return false
	}
	counts := make(map[uint]int, len(want))
	for _, item := range want {
		counts[item]++
	}
	for _, item := range got {
		counts[item]--
		if counts[item] < 0 {
			return false
		}
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

type regressionExternalDBEnv struct {
	driver   string
	dsn      string
	host     string
	port     string
	user     string
	password string
	database string
	charset  string
	loc      string
	sslMode  string
	timeZone string
}

func loadRegressionExternalDBEnv(prefix string) (regressionExternalDBEnv, bool) {
	cfg := regressionExternalDBEnv{
		driver:   strings.TrimSpace(os.Getenv(prefix + "_DRIVER")),
		dsn:      strings.TrimSpace(os.Getenv(prefix + "_DSN")),
		host:     strings.TrimSpace(os.Getenv(prefix + "_HOST")),
		port:     strings.TrimSpace(os.Getenv(prefix + "_PORT")),
		user:     strings.TrimSpace(os.Getenv(prefix + "_USER")),
		password: strings.TrimSpace(os.Getenv(prefix + "_PASSWORD")),
		database: strings.TrimSpace(os.Getenv(prefix + "_DB")),
		charset:  strings.TrimSpace(os.Getenv(prefix + "_CHARSET")),
		loc:      strings.TrimSpace(os.Getenv(prefix + "_LOC")),
		sslMode:  strings.TrimSpace(os.Getenv(prefix + "_SSLMODE")),
		timeZone: strings.TrimSpace(os.Getenv(prefix + "_TIME_ZONE")),
	}
	if cfg.dsn != "" {
		return cfg, true
	}
	ok := cfg.host != "" && cfg.database != ""
	return cfg, ok
}

func openRegressionExternalDB(t *testing.T, prefix string) *gorm.DB {
	t.Helper()

	cfg, ok := loadRegressionExternalDBEnv(prefix)
	if !ok {
		switch prefix {
		case "GIN_NINJA_TEST_MYSQL":
			t.Skip("set GIN_NINJA_TEST_MYSQL_DSN or GIN_NINJA_TEST_MYSQL_HOST/GIN_NINJA_TEST_MYSQL_DB to run MySQL regression integration tests")
		case "GIN_NINJA_TEST_POSTGRES":
			t.Skip("set GIN_NINJA_TEST_POSTGRES_DSN or GIN_NINJA_TEST_POSTGRES_HOST/GIN_NINJA_TEST_POSTGRES_DB to run PostgreSQL regression integration tests")
		default:
			t.Skip("external regression database is not configured")
		}
	}

	driver := strings.ToLower(strings.TrimSpace(cfg.driver))
	if driver == "" {
		if strings.Contains(strings.ToLower(prefix), "mysql") {
			driver = "mysql"
		} else {
			driver = "postgres"
		}
	}
	var dialector gorm.Dialector
	switch driver {
	case "mysql":
		dsn := cfg.dsn
		if dsn == "" {
			charset := cfg.charset
			if charset == "" {
				charset = "utf8mb4"
			}
			loc := cfg.loc
			if loc == "" {
				loc = "UTC"
			}
			user := cfg.user
			if user == "" {
				user = "root"
			}
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=%s&loc=%s", user, cfg.password, cfg.host, defaultString(cfg.port, "3306"), cfg.database, charset, loc)
		}
		dialector = mysql.Open(dsn)
	case "postgres", "postgresql":
		dsn := cfg.dsn
		if dsn == "" {
			user := cfg.user
			if user == "" {
				user = "postgres"
			}
			sslMode := cfg.sslMode
			if sslMode == "" {
				sslMode = "disable"
			}
			dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", cfg.host, defaultString(cfg.port, "5432"), user, cfg.password, cfg.database, sslMode)
			if cfg.timeZone != "" {
				dsn += " TimeZone=" + cfg.timeZone
			}
		}
		dialector = postgres.Open(dsn)
	default:
		t.Fatalf("unsupported external regression driver %q", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("gorm.Open external db: %v", err)
	}
	prepareRegressionSchema(t, db)
	return db
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func runRegressionCommentAndBulkDeleteScenario(t *testing.T, api *ninja.NinjaAPI, db *gorm.DB) {
	t.Helper()

	tags := seedRegressionTags(t, db)
	recordResp := performRegressionJSON(t, api, http.MethodPost, "/api/regression-records/", createRegressionRecordInput{
		Name:       "Bulk Parent",
		Email:      "bulk.parent@example.com",
		Password:   "password123",
		InviteCode: "bulk-parent",
		Age:        35,
		Score:      44.4,
		Balance:    200,
		Active:     true,
		StartsAt:   time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC),
		TagIDs:     []uint{tags[0].ID},
	}, nil)
	if recordResp.Code != http.StatusCreated {
		t.Fatalf("create record status=%d body=%s", recordResp.Code, recordResp.Body.String())
	}
	record := decodeJSONBody[regressionRecordPayload](t, recordResp.Body)

	rootResp := performRegressionJSON(t, api, http.MethodPost, "/api/regression-comments/", createRegressionCommentInput{
		RecordID: record.ID,
		Body:     "root comment",
	}, nil)
	if rootResp.Code != http.StatusCreated {
		t.Fatalf("create root comment status=%d body=%s", rootResp.Code, rootResp.Body.String())
	}
	root := decodeJSONBody[regressionCommentPayload](t, rootResp.Body)

	childResp := performRegressionJSON(t, api, http.MethodPost, "/api/regression-comments/", createRegressionCommentInput{
		RecordID: record.ID,
		ParentID: &root.ID,
		Body:     "child comment",
		IsPinned: true,
	}, nil)
	if childResp.Code != http.StatusCreated {
		t.Fatalf("create child comment status=%d body=%s", childResp.Code, childResp.Body.String())
	}
	child := decodeJSONBody[regressionCommentPayload](t, childResp.Body)

	otherRecordResp := performRegressionJSON(t, api, http.MethodPost, "/api/regression-records/", createRegressionRecordInput{
		Name:       "Other Parent",
		Email:      "other.parent@example.com",
		Password:   "password123",
		InviteCode: "other-parent",
		StartsAt:   time.Date(2026, 4, 21, 9, 0, 0, 0, time.UTC),
	}, nil)
	if otherRecordResp.Code != http.StatusCreated {
		t.Fatalf("create other record status=%d body=%s", otherRecordResp.Code, otherRecordResp.Body.String())
	}
	otherRecord := decodeJSONBody[regressionRecordPayload](t, otherRecordResp.Body)

	crossRecordResp := performRegressionJSON(t, api, http.MethodPost, "/api/regression-comments/", createRegressionCommentInput{
		RecordID: otherRecord.ID,
		ParentID: &root.ID,
		Body:     "cross-record child",
	}, nil)
	if crossRecordResp.Code != http.StatusBadRequest {
		t.Fatalf("expected cross-record parent rejection, got %d body=%s", crossRecordResp.Code, crossRecordResp.Body.String())
	}

	commentDetailResp := performRegressionJSON(t, api, http.MethodGet, "/api/regression-comments/"+strconv.FormatUint(uint64(root.ID), 10), nil, nil)
	if commentDetailResp.Code != http.StatusOK {
		t.Fatalf("comment detail status=%d body=%s", commentDetailResp.Code, commentDetailResp.Body.String())
	}
	rootDetail := decodeJSONBody[regressionCommentPayload](t, commentDetailResp.Body)
	if len(rootDetail.ChildIDs) != 1 || rootDetail.ChildIDs[0] != child.ID {
		t.Fatalf("expected child_ids on parent comment, got %+v", rootDetail)
	}

	recordDetailResp := performRegressionJSON(t, api, http.MethodGet, "/api/regression-records/"+strconv.FormatUint(uint64(record.ID), 10), nil, nil)
	if recordDetailResp.Code != http.StatusOK {
		t.Fatalf("record detail status=%d body=%s", recordDetailResp.Code, recordDetailResp.Body.String())
	}
	recordDetail := decodeJSONBody[regressionRecordPayload](t, recordDetailResp.Body)
	if !sameUintSet(recordDetail.CommentIDs, root.ID, child.ID) {
		t.Fatalf("expected comment_ids on record detail, got %+v", recordDetail)
	}

	newBody := "child comment updated"
	unpin := false
	updateChildResp := performRegressionJSON(t, api, http.MethodPatch, "/api/regression-comments/"+strconv.FormatUint(uint64(child.ID), 10), updateRegressionCommentInput{
		ParentID: &root.ID,
		Body:     &newBody,
		IsPinned: &unpin,
	}, nil)
	if updateChildResp.Code != http.StatusOK {
		t.Fatalf("update child comment status=%d body=%s", updateChildResp.Code, updateChildResp.Body.String())
	}

	headers := map[string]string{"X-User-ID": "1"}
	metaResp := performRegressionJSON(t, api, http.MethodGet, "/admin/resources/regression-comments/meta", nil, headers)
	if metaResp.Code != http.StatusOK {
		t.Fatalf("admin comment meta status=%d body=%s", metaResp.Code, metaResp.Body.String())
	}
	meta := decodeJSONBody[admin.ResourceMetadata](t, metaResp.Body)
	var recordField, parentField, childIDsField *admin.FieldMeta
	for i := range meta.Fields {
		switch meta.Fields[i].Name {
		case "record_id":
			recordField = &meta.Fields[i]
		case "parent_id":
			parentField = &meta.Fields[i]
		case "child_ids":
			childIDsField = &meta.Fields[i]
		}
	}
	if recordField == nil || recordField.Relation == nil || recordField.Relation.Resource != "regression_records" {
		t.Fatalf("unexpected record_id relation metadata: %+v", recordField)
	}
	if parentField == nil || parentField.Relation == nil || parentField.Relation.Resource != "regression_comments" {
		t.Fatalf("unexpected parent_id relation metadata: %+v", parentField)
	}
	if childIDsField == nil || !childIDsField.ReadOnly || childIDsField.Relation == nil || childIDsField.Relation.Resource != "regression_comments" {
		t.Fatalf("unexpected child_ids metadata: %+v", childIDsField)
	}

	adminCreateResp := performRegressionJSON(t, api, http.MethodPost, "/admin/resources/regression-comments", map[string]any{
		"record_id": record.ID,
		"body":      "standalone admin comment",
		"is_pinned": true,
	}, headers)
	if adminCreateResp.Code != http.StatusCreated {
		t.Fatalf("admin create comment status=%d body=%s", adminCreateResp.Code, adminCreateResp.Body.String())
	}
	adminCreated := decodeJSONBody[admin.ResourceRecordOutput](t, adminCreateResp.Body)
	if _, ok := adminCreated.Item["id"].(float64); !ok {
		t.Fatalf("expected numeric admin comment id, got %+v", adminCreated.Item["id"])
	}

	adminListResp := performRegressionJSON(t, api, http.MethodGet, "/admin/resources/regression-comments?record_id="+strconv.FormatUint(uint64(record.ID), 10)+"&search=comment&sort=-id", nil, headers)
	if adminListResp.Code != http.StatusOK {
		t.Fatalf("admin comment list status=%d body=%s", adminListResp.Code, adminListResp.Body.String())
	}
	adminList := decodeJSONBody[admin.ResourceListOutput](t, adminListResp.Body)
	if adminList.Total < 3 {
		t.Fatalf("expected at least 3 comments in admin list, got %+v", adminList)
	}

	bulkAResp := performRegressionJSON(t, api, http.MethodPost, "/api/regression-records/", createRegressionRecordInput{
		Name:       "Bulk Delete A",
		Email:      "bulk.delete.a@example.com",
		Password:   "password123",
		InviteCode: "bulk-a",
		StartsAt:   time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC),
	}, nil)
	if bulkAResp.Code != http.StatusCreated {
		t.Fatalf("create bulk record A status=%d body=%s", bulkAResp.Code, bulkAResp.Body.String())
	}
	bulkA := decodeJSONBody[regressionRecordPayload](t, bulkAResp.Body)

	bulkBResp := performRegressionJSON(t, api, http.MethodPost, "/api/regression-records/", createRegressionRecordInput{
		Name:       "Bulk Delete B",
		Email:      "bulk.delete.b@example.com",
		Password:   "password123",
		InviteCode: "bulk-b",
		StartsAt:   time.Date(2026, 4, 23, 9, 0, 0, 0, time.UTC),
	}, nil)
	if bulkBResp.Code != http.StatusCreated {
		t.Fatalf("create bulk record B status=%d body=%s", bulkBResp.Code, bulkBResp.Body.String())
	}
	bulkB := decodeJSONBody[regressionRecordPayload](t, bulkBResp.Body)

	bulkDeleteResp := performRegressionJSON(t, api, http.MethodPost, "/admin/resources/regression-records/bulk-delete", map[string]any{
		"ids": []uint{bulkA.ID, bulkB.ID},
	}, headers)
	if bulkDeleteResp.Code != http.StatusCreated {
		t.Fatalf("admin record bulk delete status=%d body=%s", bulkDeleteResp.Code, bulkDeleteResp.Body.String())
	}
	deleted := decodeJSONBody[admin.BulkDeleteOutput](t, bulkDeleteResp.Body)
	if deleted.Deleted != 2 {
		t.Fatalf("expected bulk delete of 2 records, got %+v", deleted)
	}
	for _, id := range []uint{bulkA.ID, bulkB.ID} {
		resp := performRegressionJSON(t, api, http.MethodGet, "/api/regression-records/"+strconv.FormatUint(uint64(id), 10), nil, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected bulk deleted record %d to be missing, got %d body=%s", id, resp.Code, resp.Body.String())
		}
	}
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
		Email:      "ALPHA@EXAMPLE.COM",
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
	if !sameUintSet(created.TagIDs, tags[0].ID, tags[1].ID) {
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
	}, nil); got.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected invalid email 422, got %d body=%s", got.Code, got.Body.String())
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

	metaResp := performRegressionJSON(t, api, http.MethodGet, "/admin/resources/regression-records/meta", nil, headers)
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
	createDisallowed := performRegressionJSON(t, api, http.MethodPost, "/admin/resources/regression-records", map[string]any{
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

	createResp := performRegressionJSON(t, api, http.MethodPost, "/admin/resources/regression-records", map[string]any{
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

	listResp := performRegressionJSON(t, api, http.MethodGet, "/admin/resources/regression-records?search=admin&active=true&sort=-age", nil, headers)
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

	updateDisallowed := performRegressionJSON(t, api, http.MethodPut, "/admin/resources/regression-records/"+strconv.FormatUint(uint64(id), 10), map[string]any{
		"invite_code": "should-fail",
	}, headers)
	if updateDisallowed.Code != http.StatusBadRequest {
		t.Fatalf("expected create-only field update rejection, got %d body=%s", updateDisallowed.Code, updateDisallowed.Body.String())
	}

	updateResp := performRegressionJSON(t, api, http.MethodPut, "/admin/resources/regression-records/"+strconv.FormatUint(uint64(id), 10), map[string]any{
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

	detailResp := performRegressionJSON(t, api, http.MethodGet, "/admin/resources/regression-records/"+strconv.FormatUint(uint64(id), 10), nil, headers)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("admin detail status=%d body=%s", detailResp.Code, detailResp.Body.String())
	}
	detail := decodeJSONBody[admin.ResourceRecordOutput](t, detailResp.Body)
	if detail.Item["email"] != "admin.updated@example.com" || detail.Item["status_note"] != "updated via admin" {
		t.Fatalf("unexpected admin detail item: %+v", detail.Item)
	}

	deleteResp := performRegressionJSON(t, api, http.MethodDelete, "/admin/resources/regression-records/"+strconv.FormatUint(uint64(id), 10), nil, headers)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("admin delete status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	missingResp := performRegressionJSON(t, api, http.MethodGet, "/admin/resources/regression-records/"+strconv.FormatUint(uint64(id), 10), nil, headers)
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("expected deleted admin detail 404, got %d body=%s", missingResp.Code, missingResp.Body.String())
	}
}

func TestRegressionBulkDeleteAndSelfReferentialCommentScenarios(t *testing.T) {
	api, db := newRegressionIntegrationAPI(t)
	runRegressionCommentAndBulkDeleteScenario(t, api, db)
}

func TestRegressionMySQLDialectScenarios(t *testing.T) {
	db := openRegressionExternalDB(t, "GIN_NINJA_TEST_MYSQL")
	api := newRegressionIntegrationAPIWithDB(t, db)
	runRegressionCommentAndBulkDeleteScenario(t, api, db)
}

func TestRegressionPostgresDialectScenarios(t *testing.T) {
	db := openRegressionExternalDB(t, "GIN_NINJA_TEST_POSTGRES")
	api := newRegressionIntegrationAPIWithDB(t, db)
	runRegressionCommentAndBulkDeleteScenario(t, api, db)
}
