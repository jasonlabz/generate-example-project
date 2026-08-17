// Package humax provides Huma response envelopes, pagination, and file streams.
package humax

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// Envelope is the common JSON response body for Huma handlers.
type Envelope[T any] struct {
	Code        int    `json:"code"`
	Message     string `json:"message,omitempty"`
	ErrTrace    string `json:"err_trace,omitempty"`
	Version     string `json:"version"`
	CurrentTime string `json:"current_time"`
	Data        T      `json:"data"`
}

// New creates a successful response envelope with the current local time.
func New[T any](version string, data T) *Envelope[T] {
	return &Envelope[T]{
		Version:     version,
		CurrentTime: time.Now().Format(time.DateTime),
		Data:        data,
	}
}

// NewError creates an error response envelope with the current local time.
func NewError[T any](version string, data T, code int, message, trace string) *Envelope[T] {
	envelope := New(version, data)
	envelope.Code = code
	envelope.Message = message
	envelope.ErrTrace = trace
	return envelope
}

// Output is a successful Huma response body.
type Output[T any] struct {
	Body *Envelope[T]
}

// Success creates a successful Huma response.
func Success[T any](version string, data T) *Output[T] {
	return &Output[T]{Body: New(version, data)}
}

// Result returns a successful response or a status-aware Huma error.
func Result[T any](version string, data T, err error) (*Output[T], error) {
	if err != nil {
		return nil, InternalServerError(version, err)
	}
	return Success(version, data), nil
}

// Pagination describes a page of results.
type Pagination struct {
	Page      int64 `json:"page"`
	PageSize  int64 `json:"page_size"`
	PageCount int64 `json:"page_count"`
	Total     int64 `json:"total"`
}

// GetPageCount calculates the total page count.
func (p *Pagination) GetPageCount() {
	if p.PageSize <= 0 || p.Total <= 0 {
		p.PageCount = 0
		return
	}
	p.PageCount = (p.Total + p.PageSize - 1) / p.PageSize
}

// GetOffset returns the zero-based offset for the current page.
func (p *Pagination) GetOffset() int64 {
	if p.Page <= 1 || p.PageSize <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}

// PaginationEnvelope is a successful response envelope with pagination metadata.
type PaginationEnvelope[T any] struct {
	Code        int         `json:"code"`
	Message     string      `json:"message,omitempty"`
	ErrTrace    string      `json:"err_trace,omitempty"`
	Version     string      `json:"version"`
	CurrentTime string      `json:"current_time"`
	Data        T           `json:"data"`
	Pagination  *Pagination `json:"pagination,omitempty"`
}

// PaginationOutput is a successful paginated Huma response body.
type PaginationOutput[T any] struct {
	Body *PaginationEnvelope[T]
}

// PaginationSuccess creates a successful paginated Huma response.
func PaginationSuccess[T any](version string, data T, pagination *Pagination) *PaginationOutput[T] {
	envelope := New(version, data)
	return &PaginationOutput[T]{
		Body: &PaginationEnvelope[T]{
			Code:        envelope.Code,
			Message:     envelope.Message,
			ErrTrace:    envelope.ErrTrace,
			Version:     envelope.Version,
			CurrentTime: envelope.CurrentTime,
			Data:        envelope.Data,
			Pagination:  pagination,
		},
	}
}

// PaginationResult returns a paginated response or a status-aware Huma error.
func PaginationResult[T any](version string, data T, err error, pagination *Pagination) (*PaginationOutput[T], error) {
	if err != nil {
		return nil, InternalServerError(version, err)
	}
	return PaginationSuccess(version, data, pagination), nil
}

// Error is a uniform error response that implements huma.StatusError.
type Error struct {
	*Envelope[[]any]
	status int
	cause  error
}

// InternalServerError converts an unexpected error into a 500 response.
func InternalServerError(version string, cause error) *Error {
	if cause == nil {
		cause = errors.New(http.StatusText(http.StatusInternalServerError))
	}
	return &Error{
		Envelope: NewError(version, []any{}, 0, cause.Error(), cause.Error()),
		status:   http.StatusInternalServerError,
		cause:    cause,
	}
}

// Error implements error.
func (e *Error) Error() string {
	if e == nil || e.cause == nil {
		return ""
	}
	return e.cause.Error()
}

// GetStatus implements huma.StatusError.
func (e *Error) GetStatus() int {
	return e.status
}

// FileDownloadConfig configures a streamed file response.
type FileDownloadConfig struct {
	Filename    string
	Preview     bool
	ContentType string
	Content     []byte
	Reader      io.Reader
	Filepath    string
	Disposition string
	BufferSize  int
	DeleteAfter bool
}

// File creates a Huma stream response from content, a reader, or a file path.
func File(version string, config *FileDownloadConfig) (*huma.StreamResponse, error) {
	if config == nil {
		return nil, InternalServerError(version, errors.New("file download config is nil"))
	}

	fileConfig := *config
	if fileConfig.ContentType == "" {
		fileConfig.ContentType = "application/octet-stream"
	}
	if fileConfig.Disposition == "" {
		fileConfig.Disposition = "attachment"
	}
	if fileConfig.Preview {
		fileConfig.Disposition = "inline"
	}
	if fileConfig.BufferSize <= 0 {
		fileConfig.BufferSize = 4096
	}

	reader, closeReader, contentLength, err := openFileSource(&fileConfig)
	if err != nil {
		return nil, InternalServerError(version, err)
	}

	return &huma.StreamResponse{Body: func(ctx huma.Context) {
		ctx.SetHeader("Content-Type", fileConfig.ContentType)
		ctx.SetHeader("Content-Disposition", mime.FormatMediaType(fileConfig.Disposition, map[string]string{"filename": downloadFilename(fileConfig.Filename)}))
		ctx.SetHeader("Content-Transfer-Encoding", "binary")
		ctx.SetHeader("Cache-Control", "no-cache")
		if contentLength >= 0 {
			ctx.SetHeader("Content-Length", fmt.Sprintf("%d", contentLength))
		}
		defer closeReader()

		if _, copyErr := io.CopyBuffer(ctx.BodyWriter(), reader, make([]byte, fileConfig.BufferSize)); copyErr == nil && fileConfig.DeleteAfter && fileConfig.Filepath != "" {
			_ = os.Remove(fileConfig.Filepath)
		}
	}}, nil
}

// FileWithError returns a uniform error response when err is non-nil, otherwise streams the file.
func FileWithError(version string, config *FileDownloadConfig, err error) (*huma.StreamResponse, error) {
	if err != nil {
		return nil, InternalServerError(version, err)
	}
	return File(version, config)
}

// FileResult creates a streamed file response.
func FileResult(version string, config *FileDownloadConfig) (*huma.StreamResponse, error) {
	return File(version, config)
}

// FileResultWithError creates a streamed file response or a status-aware error.
func FileResultWithError(version string, config *FileDownloadConfig, err error) (*huma.StreamResponse, error) {
	return FileWithError(version, config, err)
}

// SimpleFile creates a file response from a local file path.
func SimpleFile(version, filePath, fileName string) (*huma.StreamResponse, error) {
	return File(version, &FileDownloadConfig{
		Filepath:    filePath,
		Filename:    fileName,
		ContentType: contentType(filePath),
	})
}

// SimpleFileDownload creates a streamed file response from a local file path.
func SimpleFileDownload(version, filePath, fileName string) (*huma.StreamResponse, error) {
	return SimpleFile(version, filePath, fileName)
}

func openFileSource(config *FileDownloadConfig) (io.Reader, func(), int64, error) {
	if config.Filepath != "" {
		file, err := os.Open(config.Filepath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil, 0, fmt.Errorf("file not found: %s", config.Filepath)
			}
			return nil, nil, 0, fmt.Errorf("open file: %w", err)
		}
		fileInfo, err := file.Stat()
		if err != nil {
			_ = file.Close()
			return nil, nil, 0, fmt.Errorf("stat file: %w", err)
		}
		if config.Filename == "" {
			config.Filename = filepath.Base(config.Filepath)
		}
		return file, func() { _ = file.Close() }, fileInfo.Size(), nil
	}
	if config.Reader != nil {
		return config.Reader, func() {}, -1, nil
	}
	if config.Content != nil {
		return bytes.NewReader(config.Content), func() {}, int64(len(config.Content)), nil
	}
	return nil, nil, 0, errors.New("no file content provided")
}

func downloadFilename(filename string) string {
	if filename == "" {
		return "download"
	}
	return filename
}

func contentType(fileName string) string {
	if contentType := mime.TypeByExtension(filepath.Ext(fileName)); contentType != "" {
		return contentType
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".pdf":
		return "application/pdf"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".rar":
		return "application/x-rar-compressed"
	default:
		return "application/octet-stream"
	}
}
