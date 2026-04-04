package app

import (
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
