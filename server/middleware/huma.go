package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jasonlabz/potato/consts"
	"github.com/jasonlabz/potato/log"
	"github.com/jasonlabz/potato/utils"

	"github.com/jasonlabz/generate-example-project/common/resource"
)

const requestBodyMaxLen = 204800

type humaBodyLog struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (bl *humaBodyLog) Header() http.Header {
	return bl.ResponseWriter.Header()
}

func (bl *humaBodyLog) Write(b []byte) (int, error) {
	_, _ = bl.body.Write(b)
	return bl.ResponseWriter.Write(b)
}

func (bl *humaBodyLog) WriteHeader(statusCode int) {
	bl.ResponseWriter.WriteHeader(statusCode)
}

func logBytes(src []byte, maxLen int) []byte {
	srcLen := len(src)
	length := srcLen
	if maxLen > 0 && srcLen > maxLen {
		length = maxLen
	}
	requestBodyLogBytes := make([]byte, length)
	copy(requestBodyLogBytes, src)
	if length < srcLen {
		requestBodyLogBytes = append(requestBodyLogBytes, []byte(" ......")...)
	}
	return requestBodyLogBytes
}

// HumaOptions configures SetHumaContext.
type HumaOptions struct {
	headerMap      map[string]string
	customFieldMap map[string]func(ctx huma.Context) string
}

// HumaOption customizes SetHumaContext.
type HumaOption func(*HumaOptions)

// WithHumaHeaderField copies request headers into the request context.
func WithHumaHeaderField(headerMap map[string]string) HumaOption {
	return func(options *HumaOptions) {
		options.headerMap = headerMap
	}
}

// WithHumaCustomField adds a computed value to the request context.
func WithHumaCustomField(customFieldMap map[string]func(ctx huma.Context) string) HumaOption {
	return func(options *HumaOptions) {
		options.customFieldMap = customFieldMap
	}
}

// HumaCORS is the Huma equivalent of the project's Gin CORS middleware.
func HumaCORS() func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		origin := ctx.Header("Origin")
		if origin != "" {
			ctx.SetHeader("Access-Control-Allow-Origin", origin)
		}
		ctx.SetHeader("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
		ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Requested-With,Accept,Origin")
		ctx.SetHeader("Access-Control-Allow-Credentials", "true")
		ctx.SetHeader("Access-Control-Max-Age", "86400")

		if ctx.Method() == http.MethodOptions {
			ctx.SetStatus(http.StatusNoContent)
			return
		}

		next(ctx)
	}
}

// SetHumaContext is the Huma equivalent of potato/middleware.SetContext.
func SetHumaContext(opts ...HumaOption) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		nextContext := ctx

		setValue := func(key, value string) {
			nextContext = huma.WithValue(nextContext, key, value)
		}

		options := &HumaOptions{}
		for _, opt := range opts {
			if opt != nil {
				opt(options)
			}
		}

		for headerKey, contextKey := range options.headerMap {
			if headerKey == "" || contextKey == "" {
				continue
			}
			setValue(contextKey, ctx.Header(headerKey))
		}

		for contextKey, handler := range options.customFieldMap {
			if contextKey == "" || handler == nil {
				continue
			}
			setValue(contextKey, handler(nextContext))
		}

		traceID := ctx.Header(consts.HeaderRequestID)
		if traceID == "" {
			traceID = strings.ReplaceAll(uuid.New().String(), consts.SignDash, consts.EmptyString)
		}
		logID := strings.ReplaceAll(uuid.New().String(), consts.SignDash, consts.EmptyString)
		userID := ctx.Header(consts.HeaderUserID)
		authorization := ctx.Header(consts.HeaderAuthorization)
		clientIP := humaClientIP(ctx)

		setValue(consts.ContextToken, authorization)
		setValue(consts.ContextUserID, userID)
		setValue(consts.ContextTraceID, traceID)
		setValue(consts.ContextLOGID, logID)
		setValue(consts.ContextClientAddr, clientIP)

		next(nextContext)
	}
}

// HumaRequestMiddleware is the Huma equivalent of
// potato/middleware.RequestMiddleware.
//
// This middleware is intended for the humagin adapter because it captures the
// request and response bodies through the underlying Gin context.
func HumaRequestMiddleware() func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		ginContext := humagin.Unwrap(ctx)
		requestContext := ctx.Context()
		traceID := utils.StringValue(requestContext.Value(consts.ContextTraceID))
		if traceID != "" {
			ctx.SetHeader(consts.HeaderRequestID, traceID)
		}

		var requestBodyBytes []byte
		if ginContext.Request.Body != nil {
			requestBodyBytes, _ = io.ReadAll(ginContext.Request.Body)
		}
		ginContext.Request.Body = io.NopCloser(bytes.NewBuffer(requestBodyBytes))

		bodyLog := &humaBodyLog{
			body:           bytes.NewBufferString(""),
			ResponseWriter: ginContext.Writer,
		}
		originalWriter := ginContext.Writer
		ginContext.Writer = bodyLog
		defer func() {
			ginContext.Writer = originalWriter
		}()

		start := time.Now()
		logger := humaLogger()
		logger.Info(requestContext, "\t[GIN] request",
			log.String("proto", ctx.Version().Proto),
			log.String("client_ip", ginContext.ClientIP()),
			log.Int64("content_length", ginContext.Request.ContentLength),
			log.String("agent", ctx.Header("User-Agent")),
			log.String("request_body", string(logBytes(requestBodyBytes, requestBodyMaxLen))),
			log.String("method", ctx.Method()),
			log.String("path", ctx.URL().Path))

		next(ctx)

		statusCode := ginContext.Writer.Status()
		errorMessage := ginContext.Errors.ByType(gin.ErrorTypePrivate).String()
		responseBody := bodyLog.body.Bytes()
		if statusCode <= 0 {
			statusCode = http.StatusOK
		}

		logger.Info(ctx.Context(), "\t[GIN] response",
			log.Int("status_code", statusCode),
			log.String("error_message", errorMessage),
			log.String("response_body", string(logBytes(responseBody, requestBodyMaxLen))),
			log.String("path", ctx.URL().Path),
			log.String("cost", fmt.Sprintf("%dms", time.Since(start).Milliseconds())))
	}
}

// HumaRecoveryLog is the Huma equivalent of potato/middleware.RecoveryLog.
// It is intended for the humagin adapter.
func HumaRecoveryLog(stack bool) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		defer func() {
			panicValue := recover()
			if panicValue == nil {
				return
			}

			ginContext := humagin.Unwrap(ctx)
			requestContext := ctx.Context()
			requestDump := humaRequestDump(ctx, ginContext)

			if isBrokenPipe(panicValue) {
				logger := humaLogger().WithField(
					log.Any("error", panicValue),
					log.String("request", string(requestDump)),
				)
				logger.Error(requestContext, ctx.URL().Path)

				if err, ok := panicValue.(error); ok {
					_ = ginContext.Error(err)
				} else {
					_ = ginContext.Error(fmt.Errorf("%v", panicValue))
				}
				ginContext.Abort()
				return
			}

			if stack {
				logger := humaLogger().WithField(
					log.Any("error", panicValue),
					log.String("request", string(requestDump)),
				)
				logger.Error(requestContext, "[Recovery from panic] -- stack")
				logger.Error(requestContext, string(debug.Stack()))
			} else {
				logger := humaLogger().WithField(log.Any("error", panicValue))
				logger.Error(requestContext, "[Recovery from panic] -- request")
				logger.Error(requestContext, string(requestDump))
			}

			ginContext.AbortWithStatus(http.StatusInternalServerError)
		}()

		next(ctx)
	}
}

func humaClientIP(ctx huma.Context) string {
	remoteAddr := ctx.RemoteAddr()
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddr)
}

func humaRequestDump(ctx huma.Context, ginContext *gin.Context) []byte {
	if ginContext != nil && ginContext.Request != nil {
		requestDump, _ := httputil.DumpRequest(ginContext.Request, false)
		return requestDump
	}
	requestURL := ctx.URL()
	return []byte(fmt.Sprintf("%s %s", ctx.Method(), requestURL.RequestURI()))
}

func isBrokenPipe(panicValue any) bool {
	netError, ok := panicValue.(*net.OpError)
	if !ok {
		return false
	}
	syscallError, ok := netError.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	message := strings.ToLower(syscallError.Error())
	return strings.Contains(message, "broken pipe") ||
		strings.Contains(message, "connection reset by peer")
}

func humaLogger() *log.LoggerWrapper {
	if resource.Logger != nil {
		return resource.Logger
	}
	return log.GetLogger()
}
