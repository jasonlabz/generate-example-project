// Package humax provides Huma response envelopes, pagination, error handling and file streams.
package humax

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/danielgtaylor/huma/v2"
	potatoErrors "github.com/jasonlabz/potato/errors"

	"github.com/jasonlabz/generate-example-project/common/apperr"
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

// Result returns a successful response or a shared error envelope.
func Result[T any](version string, data T, err error) (*Output[T], error) {
	if err != nil {
		return nil, MapError(version, err)
	}
	return Success(version, data), nil
}

// Wrap 把「返回 data 的 handler」包装成 Huma handler，自动用 Envelope 加壳。
// controller 的 handler 只产出业务数据，信封统一由 humax 封装。
func Wrap[I, T any](version string, handler func(context.Context, *I) (T, error)) func(context.Context, *I) (*Output[T], error) {
	return func(ctx context.Context, in *I) (*Output[T], error) {
		item, err := handler(ctx, in)
		if err != nil {
			return nil, mapError(ctx, version, err)
		}
		return Success(version, zeroIfNil(item)), nil
	}
}

// zeroIfNil 把 nil 值归一化为对应类型的非 nil 零值（nil 切片→空切片、nil map→空 map、nil 指针→零值结构体指针），避免 data 输出 null。
func zeroIfNil[T any](item T) T {
	v := reflect.ValueOf(item)
	if v.IsValid() {
		switch v.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
			if !v.IsNil() {
				return item
			}
		default:
			return item
		}
	}

	var zero T
	rv := reflect.ValueOf(&zero).Elem()
	switch rv.Kind() {
	case reflect.Slice:
		rv.Set(reflect.MakeSlice(rv.Type(), 0, 0))
	case reflect.Map:
		rv.Set(reflect.MakeMap(rv.Type()))
	case reflect.Ptr:
		rv.Set(reflect.New(rv.Type().Elem()))
	}
	return zero
}

// WrapList 把「返回单值的 handler」包装成 Huma handler，单值包一层 []T 作为 data。
// 前端统一按数组消费，单值接口也以 [item] 形式返回。
func WrapList[I, T any](version string, handler func(context.Context, *I) (T, error)) func(context.Context, *I) (*Output[[]T], error) {
	return func(ctx context.Context, in *I) (*Output[[]T], error) {
		item, err := handler(ctx, in)
		if err != nil {
			return nil, mapError(ctx, version, err)
		}
		return Success(version, []T{zeroIfNil(item)}), nil
	}
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

// PaginationResult returns a paginated response or a shared error envelope.
func PaginationResult[T any](version string, data T, err error, pagination *Pagination) (*PaginationOutput[T], error) {
	if err != nil {
		return nil, MapError(version, err)
	}
	return PaginationSuccess(version, data, pagination), nil
}

// WrapPage 把「返回 data + pagination 的 handler」包装成 Huma handler，自动用 PaginationEnvelope 加壳。
func WrapPage[I, T any](version string, handler func(context.Context, *I) ([]T, *Pagination, error)) func(context.Context, *I) (*PaginationOutput[[]T], error) {
	return func(ctx context.Context, in *I) (*PaginationOutput[[]T], error) {
		list, page, err := handler(ctx, in)
		if err != nil {
			return nil, mapError(ctx, version, err)
		}
		return PaginationSuccess(version, list, page), nil
	}
}

// Error is a uniform error response that implements huma.StatusError.
type Error struct {
	*Envelope[[]any]
	status int
	cause  error
}

var exposeErrorDetails atomic.Bool

// ConfigureErrorDetails 控制是否把详细报错链写入 err_trace。
//
// 打开后，任何错误（4xx / 5xx 一视同仁）都会带上 err_trace：message 保留概括性文案供
// 调用方展示，err_trace 给出完整错误链供开发者定位。它会把表名、SQL、下游地址等内部
// 细节暴露给调用方，必须只在可信环境下启用（Router 中按 gin.IsDebugging() 驱动）。
func ConfigureErrorDetails(expose bool) {
	exposeErrorDetails.Store(expose)
}

// ErrorReporter 在错误被映射为响应时回调，供上层记录服务端日志或上报监控。
//
// 之所以需要它：huma 的中间件拿不到 handler 返回的 error（humagin 适配器不会把 error
// 写进 gin.Errors），中间件只能看到已经脱敏的响应体。想在生产环境保留完整错误链，
// 必须在错误离开 handler 的这一刻把它交出来。
//
// err 是 handler 返回的原始错误（保留完整错误链），status 是即将返回的 HTTP 状态码。
// 回调同步执行，实现方必须自行保证不阻塞、不 panic——它位于请求的关键路径上。
type ErrorReporter func(ctx context.Context, err error, status int)

var errorReporter atomic.Value

// SetErrorReporter 注册全局错误回调；传 nil 取消注册。
// 约定在 Router 初始化时调用一次。
func SetErrorReporter(reporter ErrorReporter) {
	if reporter == nil {
		errorReporter.Store(ErrorReporter(nil))
		return
	}
	errorReporter.Store(reporter)
}

// mapError 把错误映射为响应，并通知已注册的 ErrorReporter。
func mapError(ctx context.Context, version string, err error) error {
	notifyErrorReporter(ctx, err)
	return MapError(version, err)
}

// ReportError 通知 ErrorReporter 后原样返回错误。
// 供不走 Wrap 的 handler 使用——例如直接返回 *huma.StreamResponse 的文件流接口，
// 它们的错误不经过 mapError，需要在返回前显式上报。
func ReportError(ctx context.Context, err error) error {
	notifyErrorReporter(ctx, err)
	return err
}

// notifyErrorReporter 回调已注册的 ErrorReporter；未注册时为空操作。
func notifyErrorReporter(ctx context.Context, err error) {
	reporter, _ := errorReporter.Load().(ErrorReporter)
	if reporter == nil || err == nil {
		return
	}
	reporter(ctx, err, resolveStatus(err))
}

// resolveStatus 从错误推导即将返回的 HTTP 状态码，与 FromError 的映射保持一致。
func resolveStatus(err error) int {
	var statusError huma.StatusError
	if errors.As(err, &statusError) {
		return statusError.GetStatus()
	}
	var catalogError potatoErrors.IError
	if errors.As(err, &catalogError) {
		if spec, ok := apperr.Lookup(catalogError.Code()); ok {
			return spec.HTTPStatus
		}
	}
	return http.StatusInternalServerError
}

// InternalServerError converts an unexpected error into a 500 response.
func InternalServerError(version string, cause error) *Error {
	if cause == nil {
		cause = errors.New(apperr.Internal.Message())
	}
	return newCatalogError(version, apperr.Internal, apperr.Internal.WithErr(cause), cause)
}

// BusinessError 构造一个预期内的业务失败响应。
//
// 与 InternalServerError 的区别：业务失败对调用方是可预期、可处理的，因此响应体仍是
// 统一的 Envelope（非零 code + 公开 message），不会退回 huma 的 RFC7807 格式；
// HTTP 状态由 apperr 目录中该 code 登记的语义决定（例如 100004001 → 404）。
//
// code 应当取 apperr 中已登记的错误码；传入未登记的 code 时回退为 InvalidRequest 的
// HTTP 语义，但保留调用方传入的 code 与 message，避免前端拿到无法定位的错误。
func BusinessError(version string, code int, message string) *Error {
	spec, ok := apperr.Lookup(code)
	if !ok {
		spec = apperr.InvalidRequest
	}
	if message == "" {
		message = spec.Message()
	}
	trace := ""
	if exposeErrorDetails.Load() {
		// 业务失败由调用方直接判定，没有更深的错误来源，因此用与 IError.Error() 一致的
		// "code -> message" 形式，保证调试时 err_trace 始终有内容可看。
		trace = fmt.Sprintf("%d -> %s", code, message)
	}
	return &Error{
		Envelope: NewError(version, []any{}, code, message, trace),
		status:   spec.HTTPStatus,
	}
}

// FromError maps a service error to a shared error envelope.
func FromError(version string, err error) *Error {
	if err == nil {
		return nil
	}
	var sharedError *Error
	if errors.As(err, &sharedError) {
		return sharedError
	}
	var ex potatoErrors.IError
	if errors.As(err, &ex) {
		if spec, ok := apperr.Lookup(ex.Code()); ok {
			return newCatalogError(version, spec, ex, ex)
		}
	}
	return InternalServerError(version, err)
}

// MapError preserves shared errors and converts unexpected errors to a safe 500 response.
func MapError(version string, err error) error {
	return FromError(version, err)
}

// ConfigureHumaErrorFactory makes Huma input errors use the shared envelope.
// Call it once during router setup before registering operations.
func ConfigureHumaErrorFactory(version string) {
	newError := func(status int, message string, details ...error) huma.StatusError {
		spec := apperr.ForHTTPStatus(status)
		if spec.HTTPStatus >= http.StatusInternalServerError {
			return InternalServerError(version, errors.New(message))
		}
		return newCatalogError(version, spec, spec.WithMessage(validationMessage(message, details)), nil)
	}
	huma.NewError = newError
	huma.NewErrorWithContext = func(_ huma.Context, status int, message string, details ...error) huma.StatusError {
		return newError(status, message, details...)
	}
}

func newCatalogError(version string, spec apperr.Spec, mapped potatoErrors.IError, cause error) *Error {
	message := mapped.Message()
	if spec.HTTPStatus >= http.StatusInternalServerError {
		message = spec.Message()
	}
	trace := ""
	if exposeErrorDetails.Load() {
		trace = detailTrace(cause, mapped)
	}
	return &Error{
		Envelope: NewError(version, []any{}, spec.Code(), message, trace),
		status:   spec.HTTPStatus,
		cause:    cause,
	}
}

// detailTrace 返回写入 err_trace 的详细报错链。
//
// 调试模式下不做任何过滤：能拿到 cause 就输出它的完整 Error()——既包含 fmt.Errorf 逐层
// 包装的上下文，也包含 apperr 的 "code -> message" 前缀；拿不到 cause（如纯 WithMessage
// 构造的业务错误、参数校验）就回退到目录错误本身，保证调试时一定看得到报错来源。
func detailTrace(cause error, mapped potatoErrors.IError) string {
	if cause != nil {
		return cause.Error()
	}
	if mapped != nil {
		return mapped.Error()
	}
	return ""
}

// Error implements error.
// 预期内的业务失败没有内部 cause，此时回退为公开 message，避免服务端日志拿到空字符串。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.Envelope.Message
}

// GetStatus implements huma.StatusError.
func (e *Error) GetStatus() int {
	return e.status
}

// ContentType keeps error responses on the JSON media type used by the shared envelope.
func (*Error) ContentType(contentType string) string {
	return strings.Replace(contentType, "problem+", "", 1)
}

func validationMessage(message string, details []error) string {
	parts := make([]string, 0, len(details))
	for _, detail := range details {
		if detail == nil {
			continue
		}
		errorDetail, ok := detail.(huma.ErrorDetailer)
		if !ok {
			continue
		}
		value := errorDetail.ErrorDetail()
		if value.Message == "" {
			continue
		}
		validationMessage := translateValidationMessage(value.Message)
		if value.Location != "" {
			validationMessage = value.Location + ": " + validationMessage
		}
		parts = append(parts, validationMessage)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "; ")
	}
	return message
}

func translateValidationMessage(message string) string {
	switch {
	case strings.Contains(message, "is a required field"), message == "required":
		return "为必填项"
	case strings.Contains(message, "expected number >="):
		return "必须大于等于" + lastNumber(message)
	case strings.Contains(message, "expected number <="):
		return "必须小于等于" + lastNumber(message)
	case strings.Contains(message, "expected number >"):
		return "必须大于" + lastNumber(message)
	case strings.Contains(message, "expected number <"):
		return "必须小于" + lastNumber(message)
	case strings.Contains(message, "expected string length >="):
		return "长度不能小于" + lastNumber(message)
	case strings.Contains(message, "expected string length <="):
		return "长度不能大于" + lastNumber(message)
	case strings.Contains(message, "expected string length >"):
		return "长度须大于" + lastNumber(message)
	case strings.Contains(message, "expected string length <"):
		return "长度须小于" + lastNumber(message)
	case strings.Contains(message, "must be a valid"):
		return "格式不合法"
	case strings.Contains(message, "oneof"):
		return "取值不在允许范围内"
	default:
		return message
	}
}

func lastNumber(message string) string {
	fields := strings.Fields(message)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

// BinaryResponse describes a binary/file download in the OpenAPI doc.
// StreamResponse 不携带 media type，调用方把它挂到 Operation.Responses 上，文档里就能看到具体的下载类型与 binary schema。
func BinaryResponse(description, contentType string) *huma.Response {
	return &huma.Response{
		Description: description,
		Content: map[string]*huma.MediaType{
			contentType: {Schema: &huma.Schema{Type: "string", Format: "binary"}},
		},
	}
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
		return nil, FromError(version, err)
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
