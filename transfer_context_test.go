package ninja

import (
	"context"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
	"github.com/shijl0925/gin-ninja/pkg/i18n"
)

func TestUploadedFileNilHelpers(t *testing.T) {
	t.Parallel()

	var file *UploadedFile

	if _, err := file.Open(); !IsBadRequest(err) {
		t.Fatalf("Open() error = %v, want bad request", err)
	}
	if _, err := file.Bytes(); !IsBadRequest(err) {
		t.Fatalf("Bytes() error = %v, want bad request", err)
	}
}

func TestTransferTypeHelpers(t *testing.T) {
	t.Parallel()

	if !isMultipartFileHeaderSliceType(reflect.TypeOf([]*multipart.FileHeader{})) {
		t.Fatal("expected []*multipart.FileHeader to be detected as multipart header slice")
	}
	if isMultipartFileHeaderSliceType(reflect.TypeOf([]multipart.FileHeader{})) {
		t.Fatal("expected []multipart.FileHeader not to be treated as multipart header pointer slice")
	}
}

func TestDownloadWriteToDataDefaultsAndHeaders(t *testing.T) {
	t.Parallel()

	c, w := newTestContext(http.MethodGet, "/download", "")
	download := NewDownload("report.txt", "", []byte("hello world"))
	download.Headers = map[string]string{"X-Test": "value"}

	download.writeTo(c, http.StatusCreated)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if body := w.Body.String(); body != "hello world" {
		t.Fatalf("body = %q, want %q", body, "hello world")
	}
	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", got)
	}
	if got := w.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") || !strings.Contains(got, "report.txt") {
		t.Fatalf("Content-Disposition = %q, want attachment filename", got)
	}
	if got := w.Header().Get("X-Test"); got != "value" {
		t.Fatalf("X-Test = %q, want value", got)
	}
}

func TestDownloadWriteToReaderAndNilDownload(t *testing.T) {
	t.Parallel()

	t.Run("reader-backed download", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/download", "")
		download := NewDownloadReader("inline.csv", "", 5, strings.NewReader("a,b\n"))
		download.Inline = true

		download.writeTo(c, http.StatusAccepted)

		if w.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
		}
		if body := w.Body.String(); body != "a,b\n" {
			t.Fatalf("body = %q, want %q", body, "a,b\n")
		}
		if got := w.Header().Get("Content-Type"); got != "application/octet-stream" {
			t.Fatalf("Content-Type = %q, want application/octet-stream", got)
		}
		if got := w.Header().Get("Content-Disposition"); !strings.Contains(got, "inline") || !strings.Contains(got, "inline.csv") {
			t.Fatalf("Content-Disposition = %q, want inline filename", got)
		}
	})

	t.Run("nil download", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/download", "")
		var download *Download

		download.writeTo(c, http.StatusAccepted)

		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
		}
		if w.Body.Len() != 0 {
			t.Fatalf("expected empty body, got %q", w.Body.String())
		}
	})
}

func TestTransferTypeHelpersAndDisposition(t *testing.T) {
	t.Parallel()

	if !isUploadedFileType(reflect.TypeOf(UploadedFile{})) {
		t.Fatal("expected UploadedFile to be recognized")
	}
	if !isUploadedFilePointerType(reflect.TypeOf(&UploadedFile{})) {
		t.Fatal("expected *UploadedFile to be recognized")
	}
	if !isUploadedFileSliceType(reflect.TypeOf([]*UploadedFile{})) {
		t.Fatal("expected []*UploadedFile to be recognized")
	}
	if !isDownloadType(reflect.TypeOf(Download{})) {
		t.Fatal("expected Download to be recognized")
	}
	if got := formatDisposition("attachment", "report.txt"); !strings.Contains(got, "attachment") || !strings.Contains(got, "report.txt") {
		t.Fatalf("formatDisposition() = %q", got)
	}
}

func TestContextHelpersAndErrorUtilities(t *testing.T) {
	t.Parallel()

	c, _ := newTestContext(http.MethodGet, "/ctx", "")
	baseCtx := context.WithValue(c.Request.Context(), "trace", "trace-123")
	c.Request = c.Request.WithContext(baseCtx)
	c.Set(requestIDContextKey, "req-1")
	c.Set(contextkeys.JWTClaims, contextClaims{userID: 7})

	ctx := newContext(c)
	if got := ctx.StdContext().Value("trace"); got != "trace-123" {
		t.Fatalf("StdContext().Value(trace) = %v, want trace-123", got)
	}
	if got := ctx.Value("trace"); got != "trace-123" {
		t.Fatalf("Value(trace) = %v, want trace-123", got)
	}
	if got := ctx.RequestID(); got != "req-1" {
		t.Fatalf("RequestID() = %q, want req-1", got)
	}
	if got := ctx.GetUserID(); got != 7 {
		t.Fatalf("GetUserID() = %d, want 7", got)
	}
	if got := ctx.Locale(); got != i18n.Default {
		t.Fatalf("Locale() = %q, want %q", got, i18n.Default)
	}

	c.Set(contextkeys.Locale, i18n.Zh)
	if got := ctx.Locale(); got != i18n.Zh {
		t.Fatalf("Locale() = %q, want %q", got, i18n.Zh)
	}
	if got := ctx.T("not_found"); got != "资源不存在" {
		t.Fatalf("T(not_found) = %q", got)
	}

	if err := ctx.BeginTx(); err == nil {
		t.Fatal("expected BeginTx() to fail when transaction handlers are unavailable")
	}
	if err := ctx.CommitTx(); err == nil {
		t.Fatal("expected CommitTx() to fail when transaction handlers are unavailable")
	}
	if err := ctx.RollbackTx(); err == nil {
		t.Fatal("expected RollbackTx() to fail when transaction handlers are unavailable")
	}

	previousBegin := contextBeginTx
	previousCommit := contextCommitTx
	previousRollback := contextRollbackTx
	t.Cleanup(func() {
		contextBeginTx = previousBegin
		contextCommitTx = previousCommit
		contextRollbackTx = previousRollback
	})

	beginCalled := false
	commitCalled := false
	rollbackCalled := false
	contextBeginTx = func(*gin.Context) error {
		beginCalled = true
		return nil
	}
	contextCommitTx = func(*gin.Context) error {
		commitCalled = true
		return nil
	}
	contextRollbackTx = func(*gin.Context) error {
		rollbackCalled = true
		return nil
	}

	if err := ctx.BeginTx(); err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	if err := ctx.CommitTx(); err != nil {
		t.Fatalf("CommitTx() error = %v", err)
	}
	if err := ctx.RollbackTx(); err != nil {
		t.Fatalf("RollbackTx() error = %v", err)
	}
	if !beginCalled || !commitCalled || !rollbackCalled {
		t.Fatalf("expected transaction handlers to run, got begin=%v commit=%v rollback=%v", beginCalled, commitCalled, rollbackCalled)
	}

	badRequest := BadRequestError()
	if !IsBadRequest(badRequest) {
		t.Fatal("expected IsBadRequest to match BadRequestError")
	}
	badRequest.Message = "changed"
	if fresh := BadRequestError(); fresh.Message != "bad request" {
		t.Fatalf("expected BadRequestError() to return a clone, got %q", fresh.Message)
	}

	detail := map[string]any{"field": "email"}
	businessErr := NewBusinessErrorWithDetail(1001, "invalid", detail)
	if businessErr.Detail == nil {
		t.Fatal("expected business error detail to be preserved")
	}
}
