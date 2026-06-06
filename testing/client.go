// Package ninjatest provides testing helpers for gin-ninja APIs.
package ninjatest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	ninja "github.com/shijl0925/gin-ninja"
)

// TestingT is the subset of testing.TB used by TestClient.
type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

// ClientOption configures a TestClient.
type ClientOption func(*clientConfig)

type clientConfig struct {
	apiConfig      ninja.Config
	defaultHeaders http.Header
	t              TestingT
}

// WithConfig configures the temporary NinjaAPI used when the client target is
// a *ninja.Router.
func WithConfig(config ninja.Config) ClientOption {
	return func(c *clientConfig) {
		c.apiConfig = config
	}
}

// WithT makes request construction, JSON encoding, and response reading errors
// fail the provided test instead of panicking.
func WithT(t TestingT) ClientOption {
	return func(c *clientConfig) {
		c.t = t
	}
}

// WithHeader sets a default header on every request.
func WithHeader(name, value string) ClientOption {
	return func(c *clientConfig) {
		if c.defaultHeaders == nil {
			c.defaultHeaders = http.Header{}
		}
		c.defaultHeaders.Set(name, value)
	}
}

// WithHeaders adds default headers to every request.
func WithHeaders(headers http.Header) ClientOption {
	return func(c *clientConfig) {
		if c.defaultHeaders == nil {
			c.defaultHeaders = http.Header{}
		}
		copyHeader(c.defaultHeaders, headers)
	}
}

// RequestOption configures an individual request.
type RequestOption func(*http.Request)

// Header sets a request header.
func Header(name, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(name, value)
	}
}

// Headers copies request headers.
func Headers(headers http.Header) RequestOption {
	return func(req *http.Request) {
		copyHeader(req.Header, headers)
	}
}

// Query adds one query parameter value to the request URL.
func Query(name, value string) RequestOption {
	return func(req *http.Request) {
		q := req.URL.Query()
		q.Add(name, value)
		req.URL.RawQuery = q.Encode()
	}
}

// Cookie adds a request cookie.
func Cookie(cookie *http.Cookie) RequestOption {
	return func(req *http.Request) {
		if cookie != nil {
			req.AddCookie(cookie)
		}
	}
}

// TestClient executes requests against a NinjaAPI, Router, or http.Handler
// without starting a network listener.
type TestClient struct {
	handler        http.Handler
	defaultHeaders http.Header
	t              TestingT
}

// New creates a TestClient for a *ninja.NinjaAPI, *ninja.Router, or
// http.Handler target.
func New(target any, opts ...ClientOption) *TestClient {
	cfg := clientConfig{defaultHeaders: http.Header{}}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &TestClient{
		handler:        resolveHandler(target, cfg.apiConfig),
		defaultHeaders: cfg.defaultHeaders.Clone(),
		t:              cfg.t,
	}
}

// NewWithT creates a TestClient that reports helper errors through t.
func NewWithT(t TestingT, target any, opts ...ClientOption) *TestClient {
	return New(target, append([]ClientOption{WithT(t)}, opts...)...)
}

func resolveHandler(target any, config ninja.Config) http.Handler {
	switch v := target.(type) {
	case *ninja.NinjaAPI:
		if v == nil {
			panic("ninjatest: nil *ninja.NinjaAPI target")
		}
		return v.Handler()
	case *ninja.Router:
		if v == nil {
			panic("ninjatest: nil *ninja.Router target")
		}
		api := ninja.New(config)
		api.AddRouter(v)
		return api.Handler()
	case http.Handler:
		if v == nil {
			panic("ninjatest: nil http.Handler target")
		}
		return v
	default:
		panic(fmt.Sprintf("ninjatest: unsupported target type %T", target))
	}
}

// NewRequest builds an in-memory request with JSON encoding for struct, map,
// slice, and scalar bodies. io.Reader, []byte, string, and url.Values bodies are
// sent as-is.
func (c *TestClient) NewRequest(method, path string, body any, opts ...RequestOption) *http.Request {
	reader, contentType, err := encodeBody(body)
	if err != nil {
		c.failf("ninjatest: encode request body: %v", err)
		return nil
	}
	req := httptest.NewRequest(method, path, reader)
	copyHeader(req.Header, c.defaultHeaders)
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, opt := range opts {
		opt(req)
	}
	return req
}

// Do executes req against the target handler.
func (c *TestClient) Do(req *http.Request) *Response {
	if req == nil {
		c.failf("ninjatest: nil request")
		return nil
	}
	w := httptest.NewRecorder()
	c.handler.ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		c.failf("ninjatest: read response body: %v", err)
		return nil
	}
	return &Response{
		StatusCode: result.StatusCode,
		Header:     result.Header.Clone(),
		Body:       body,
		Cookies:    result.Cookies(),
	}
}

// Request builds and executes a request.
func (c *TestClient) Request(method, path string, body any, opts ...RequestOption) *Response {
	return c.Do(c.NewRequest(method, path, body, opts...))
}

// Get executes a GET request.
func (c *TestClient) Get(path string, opts ...RequestOption) *Response {
	return c.Request(http.MethodGet, path, nil, opts...)
}

// Post executes a POST request.
func (c *TestClient) Post(path string, body any, opts ...RequestOption) *Response {
	return c.Request(http.MethodPost, path, body, opts...)
}

// Put executes a PUT request.
func (c *TestClient) Put(path string, body any, opts ...RequestOption) *Response {
	return c.Request(http.MethodPut, path, body, opts...)
}

// Patch executes a PATCH request.
func (c *TestClient) Patch(path string, body any, opts ...RequestOption) *Response {
	return c.Request(http.MethodPatch, path, body, opts...)
}

// Delete executes a DELETE request.
func (c *TestClient) Delete(path string, opts ...RequestOption) *Response {
	return c.Request(http.MethodDelete, path, nil, opts...)
}

// Response is the simplified response returned by TestClient.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Cookies    []*http.Cookie
}

// String returns the response body as a string.
func (r *Response) String() string {
	if r == nil {
		return ""
	}
	return string(r.Body)
}

// DecodeJSON decodes the response body as JSON.
func (r *Response) DecodeJSON(out any) error {
	if r == nil {
		return fmt.Errorf("ninjatest: nil response")
	}
	return json.Unmarshal(r.Body, out)
}

// MultipartBody describes a multipart/form-data request body.
type MultipartBody struct {
	Fields url.Values
	Files  []MultipartFile
}

// MultipartFile describes one file part in a multipart/form-data request body.
type MultipartFile struct {
	FieldName string
	FileName  string
	Body      any
}

// Multipart creates a multipart/form-data request body from fields and files.
func Multipart(fields url.Values, files ...MultipartFile) *MultipartBody {
	return &MultipartBody{
		Fields: cloneValues(fields),
		Files:  append([]MultipartFile(nil), files...),
	}
}

// File creates a file part for Multipart. body may be an io.Reader, []byte, or string.
func File(fieldName, fileName string, body any) MultipartFile {
	return MultipartFile{FieldName: fieldName, FileName: fileName, Body: body}
}

func encodeBody(body any) (io.Reader, string, error) {
	switch v := body.(type) {
	case nil:
		return http.NoBody, "", nil
	case *MultipartBody:
		if v == nil {
			return nil, "", fmt.Errorf("nil multipart body")
		}
		return encodeMultipartBody(*v)
	case MultipartBody:
		return encodeMultipartBody(v)
	case io.Reader:
		return v, "", nil
	case []byte:
		return bytes.NewReader(v), "", nil
	case string:
		return strings.NewReader(v), "", nil
	case url.Values:
		return strings.NewReader(v.Encode()), "application/x-www-form-urlencoded", nil
	default:
		payload, err := json.Marshal(v)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(payload), "application/json", nil
	}
}

func encodeMultipartBody(body MultipartBody) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for key, values := range body.Fields {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return nil, "", err
			}
		}
	}
	for _, file := range body.Files {
		if file.FieldName == "" {
			return nil, "", fmt.Errorf("multipart file field name is required")
		}
		part, err := writer.CreateFormFile(file.FieldName, file.FileName)
		if err != nil {
			return nil, "", err
		}
		reader, err := multipartFileReader(file.Body)
		if err != nil {
			return nil, "", err
		}
		if _, err := io.Copy(part, reader); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return bytes.NewReader(buf.Bytes()), writer.FormDataContentType(), nil
}

func multipartFileReader(body any) (io.Reader, error) {
	switch v := body.(type) {
	case nil:
		return nil, fmt.Errorf("multipart file body is required")
	case io.Reader:
		return v, nil
	case []byte:
		return bytes.NewReader(v), nil
	case string:
		return strings.NewReader(v), nil
	default:
		return nil, fmt.Errorf("unsupported multipart file body type %T", body)
	}
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}
	clone := make(url.Values, len(values))
	for key, vals := range values {
		clone[key] = append([]string(nil), vals...)
	}
	return clone
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func (c *TestClient) failf(format string, args ...any) {
	if c != nil && c.t != nil {
		c.t.Helper()
		c.t.Fatalf(format, args...)
	}
	panic(fmt.Sprintf(format, args...))
}
