package app

import (
	"fmt"
	"strings"
	"time"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/pagination"
)

// EchoRequestMeta demonstrates cookie/header/query binding and defaults.
func EchoRequestMeta(ctx *ninja.Context, in *RequestMetaInput) (*RequestMetaOutput, error) {
	return &RequestMetaOutput{
		Session: in.Session,
		TraceID: in.TraceID,
		Lang:    in.Lang,
		Verbose: in.Verbose,
	}, nil
}

// SlowOperation blocks until timeout/cancellation to demonstrate Timeout().
func SlowOperation(ctx *ninja.Context, _ *struct{}) (*SlowDemoOutput, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(500 * time.Millisecond):
		return &SlowDemoOutput{Status: "completed"}, nil
	}
}

// LimitedOperation demonstrates per-operation rate limiting.
func LimitedOperation(ctx *ninja.Context, _ *struct{}) (*LimitedDemoOutput, error) {
	return &LimitedDemoOutput{Status: "allowed"}, nil
}

// HiddenOperation remains routable but excluded from OpenAPI.
func HiddenOperation(ctx *ninja.Context, _ *struct{}) (*HiddenDemoOutput, error) {
	return &HiddenDemoOutput{Status: "hidden route is reachable"}, nil
}

// ListFeatureDemos returns a static paginated dataset for OpenAPI/testing demos.
func ListFeatureDemos(ctx *ninja.Context, in *FeatureListInput) (*pagination.Page[FeatureItemOut], error) {
	items := []FeatureItemOut{
		{Code: "cookie-binding", Title: "Cookie binding", Enabled: true},
		{Code: "extra-responses", Title: "Extra OpenAPI responses", Enabled: true},
		{Code: "hidden-route", Title: "Exclude from OpenAPI", Enabled: true},
		{Code: "defaults", Title: "Header/query/cookie defaults", Enabled: true},
		{Code: "tag-description", Title: "OpenAPI tag descriptions", Enabled: true},
		{Code: "timeout", Title: "Operation timeout", Enabled: true},
		{Code: "rate-limit", Title: "Operation rate limit", Enabled: true},
		{Code: "paginated-response", Title: "Standard paginated response declaration", Enabled: true},
		{Code: "file-upload", Title: "Multipart file upload binding", Enabled: true},
		{Code: "file-download", Title: "Binary download response", Enabled: true},
	}

	filtered := make([]FeatureItemOut, 0, len(items))
	for _, item := range items {
		if in.Search == "" || in.Search == "demo" || containsFeature(item, in.Search) {
			filtered = append(filtered, item)
		}
	}

	page := in.GetPage()
	size := in.GetSize()
	start := (page - 1) * size
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + size
	if end > len(filtered) {
		end = len(filtered)
	}

	return pagination.NewPage(filtered[start:end], int64(len(filtered)), in.PageInput), nil
}

func containsFeature(item FeatureItemOut, search string) bool {
	search = strings.ToLower(search)
	return strings.Contains(strings.ToLower(item.Code), search) || strings.Contains(strings.ToLower(item.Title), search)
}

// UploadSingleDemo demonstrates single-file upload with mixed multipart fields.
func UploadSingleDemo(ctx *ninja.Context, in *UploadSingleInput) (*UploadDemoOutput, error) {
	return &UploadDemoOutput{
		Title:     in.Title,
		Filename:  in.File.Filename,
		Size:      in.File.Size,
		FileCount: 1,
		Names:     []string{in.File.Filename},
	}, nil
}

// UploadManyDemo demonstrates multi-file upload binding.
func UploadManyDemo(ctx *ninja.Context, in *UploadManyInput) (*UploadDemoOutput, error) {
	names := make([]string, 0, len(in.Files))
	for _, file := range in.Files {
		names = append(names, file.Filename)
	}
	return &UploadDemoOutput{
		Category:  in.Category,
		FileCount: len(in.Files),
		Names:     names,
	}, nil
}

// DownloadDemo demonstrates binary/file download responses.
func DownloadDemo(ctx *ninja.Context, _ *struct{}) (*ninja.Download, error) {
	data := []byte("gin-ninja file download demo\n")
	return ninja.NewDownload("demo.txt", "text/plain; charset=utf-8", data), nil
}

// DownloadNamedDemo demonstrates reader-backed downloads.
func DownloadNamedDemo(ctx *ninja.Context, _ *struct{}) (*ninja.Download, error) {
	body := fmt.Sprintf("request_id=%s\n", ctx.RequestID())
	return ninja.NewDownloadReader("request.txt", "text/plain; charset=utf-8", int64(len(body)), strings.NewReader(body)), nil
}
