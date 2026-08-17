// Package health_check 是「健康检查」业务模块的 HTTP 控制器层。
//
// 控制器职责：把 huma 请求与业务层（service）对接——定义路由
// （huma.Operation）、请求/响应类型（types.go）、并做 DTO 与业务模型的转换
// （convertor.go）。控制器不包含业务逻辑，业务逻辑在 service 层。
//
// huma 路由注册约定（swag 注解 → huma 字段对照）：
//
//	@Summary         → Operation.Summary
//	@Description     → Operation.Description
//	@Tags            → Operation.Tags
//	@ID              → Operation.OperationID
//	@Accept/@Produce → Operation 的 ContentTypes（默认 application/json）
//	@Param           → 请求结构体字段的 huma tag（path/query/header/body）
//	@Success/@Failure→ 返回类型 + error（huma.StatusError 控制状态码）
//	@Router          → Operation.Method + Operation.Path
//
// 请求/响应结构体字段用 huma tag 声明位置与校验规则，huma 会自动生成
// OpenAPI 参数与 schema，无需手写 swagger 注解。
package health_check

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// HealthCheckController 定义健康检查模块的路由注册入口。
// 业务模块控制器统一实现该形态：NewXxxController(...) 构造 + Register(api) 注册。
type HealthCheckController interface {
	Register(api huma.API)
}

// Register 将 GET /health-check 注册到 api（通常传入 huma 组，如 serverAPI）。
//
// 接口契约：
//   - 200：返回 humax.Envelope[[]string]，data 为当前健康状态（通常是 success）；
//   - 500：返回 humax.Error，包含 code、message、err_trace、version 和空 data；
//   - 无请求参数、无鉴权要求，适用于负载均衡器和容器探针检查服务存活状态。
//
// huma 会根据 handler 的输入输出类型生成 OpenAPI schema；Operation 中的
// Summary、Description、Tags、OperationID 和 Errors 对应传统 Swagger 注解
// @Summary、@Description、@Tags、@ID 和 @Failure。
func (c *controller) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		// —— 路由与 OpenAPI 基础信息（对应 swag 的 @Router/@ID/@Summary/@Tags）——
		OperationID: "health-check",                                                                     // 全局唯一操作标识（对应 @ID）
		Method:      http.MethodGet,                                                                     // HTTP 方法（对应 @Router 前半部分）
		Path:        "/health-check",                                                                    // 路径，路径参数用 {name} 占位（对应 @Router）
		Summary:     "健康检查",                                                                             // 接口名（对应 @Summary，显示在文档列表）
		Description: "用于负载均衡器、容器探针和运维监控检查服务是否存活。成功时返回 code=0、data=[\"success\"]；依赖检查失败时返回 500 及统一错误信封。", // 接口说明（对应 @Description，显示在文档详情）
		Tags:        []string{"系统", "健康检查"},                                                             // 分组标签（对应 @Tags，文档按此分组展示）

		// —— 响应与错误声明 ——
		// 成功状态码：handler 返回 nil error 时使用的默认状态码。
		DefaultStatus: http.StatusOK,
		// 错误状态码声明：列出本接口可能返回的错误码，
		// 会显示在 OpenAPI 文档的 responses 中（对应 @Failure 声明）。
		Errors: []int{
			http.StatusInternalServerError, // 500：服务内部错误（humax.InternalServerError）
		},
		Responses: map[string]*huma.Response{
			"200": {Description: "服务正常，data 包含当前健康状态。"},
		},

		// —— 可选扩展（按需启用）——
		// Deprecated: true,  // 标记接口废弃，文档中显示警示（对应 @Deprecated）
		// 鉴权声明：需先在 OpenAPI.Components.SecuritySchemes 配置安全方案：
		// Security: []map[string][]string{{"Bearer": {}}},
		// 按操作挂载 huma 中间件（一般优先用组中间件，见 huma.Group.UseMiddleware）：
		// Middlewares: huma.Middlewares{...},
		// 请求体大小限制（默认继承 humaConfig.MaxBodyBytes）：
		// MaxBodyBytes: 1 << 20,
	}, c.handle)
}
