package ninja

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UploadedFile wraps a multipart file and exposes convenience helpers.
type UploadedFile struct {
	*multipart.FileHeader
}

func newUploadedFile(header *multipart.FileHeader) *UploadedFile {
	if header == nil {
		return nil
	}
	return &UploadedFile{FileHeader: header}
}

// Open delegates to the underlying multipart.FileHeader.
func (f *UploadedFile) Open() (multipart.File, error) {
	if f == nil || f.FileHeader == nil {
		return nil, BadRequestError()
	}
	return f.FileHeader.Open()
}

// Bytes reads the uploaded file content into memory.
func (f *UploadedFile) Bytes() ([]byte, error) {
	file, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

type responseWriter interface {
	writeTo(*gin.Context, int)
}

// Download represents a binary/file response.
type Download struct {
	Filename    string
	ContentType string
	Data        []byte
	Reader      io.Reader
	Size        int64
	Inline      bool
	Headers     map[string]string
}

// NewDownload returns a binary response backed by a byte slice.
func NewDownload(filename, contentType string, data []byte) *Download {
	return &Download{
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
		Size:        int64(len(data)),
	}
}

// NewDownloadReader returns a binary response backed by a reader.
func NewDownloadReader(filename, contentType string, size int64, reader io.Reader) *Download {
	return &Download{
		Filename:    filename,
		ContentType: contentType,
		Reader:      reader,
		Size:        size,
	}
}

func (d *Download) writeTo(c *gin.Context, status int) {
	if d == nil {
		c.Status(http.StatusNoContent)
		c.Writer.WriteHeaderNow()
		return
	}

	contentType := d.ContentType
	if contentType == "" {
		switch {
		case len(d.Data) > 0:
			contentType = http.DetectContentType(d.Data)
		default:
			contentType = "application/octet-stream"
		}
	}

	headers := map[string]string{}
	for k, v := range d.Headers {
		headers[k] = v
	}
	if d.Filename != "" {
		disposition := "attachment"
		if d.Inline {
			disposition = "inline"
		}
		headers["Content-Disposition"] = formatDisposition(disposition, d.Filename)
	}

	if d.Reader != nil {
		if c.Request.Method == http.MethodHead {
			writeDownloadHead(c, status, contentType, d.Size, headers)
			return
		}
		c.DataFromReader(status, d.Size, contentType, d.Reader, headers)
		return
	}

	data := d.Data
	if data == nil {
		data = []byte{}
	}
	if c.Request.Method == http.MethodHead {
		writeDownloadHead(c, status, contentType, int64(len(data)), headers)
		return
	}
	c.DataFromReader(status, int64(len(data)), contentType, bytes.NewReader(data), headers)
}

var multipartFileHeaderType = reflect.TypeOf(multipart.FileHeader{})
var uploadedFileType = reflect.TypeOf(UploadedFile{})
var downloadType = reflect.TypeOf(Download{})

func isUploadedFileType(t reflect.Type) bool {
	return deref(t) == uploadedFileType
}

func isUploadedFilePointerType(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && deref(t) == uploadedFileType
}

func isUploadedFileSliceType(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && isUploadedFilePointerType(t.Elem())
}

func isMultipartFileHeaderPointerType(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && deref(t) == multipartFileHeaderType
}

func isMultipartFileHeaderSliceType(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && isMultipartFileHeaderPointerType(t.Elem())
}

func isDownloadType(t reflect.Type) bool {
	return deref(t) == downloadType
}

func formatDisposition(disposition, filename string) string {
	value := mime.FormatMediaType(disposition, map[string]string{"filename": filename})
	if value != "" {
		return value
	}
	return disposition
}

func writeDownloadHead(c *gin.Context, status int, contentType string, size int64, headers map[string]string) {
	for key, value := range headers {
		c.Header(key, value)
	}
	c.Header("Content-Type", contentType)
	if size >= 0 {
		c.Header("Content-Length", strconv.FormatInt(size, 10))
	}
	c.Status(status)
	c.Writer.WriteHeaderNow()
}
