# response — 统一响应信封

框架无关的 API 响应信封，HTTP 适配层（huma/gin）共用。

## 结构

```json
{
  "code": 0,               // 业务码（0 成功；非 0 见具体业务定义）
  "message": "",           // 错误消息（成功时省略）
  "err_trace": "",         // 错误追踪（成功时省略）
  "version": "v1",         // 接口版本
  "current_time": "2026-08-16 12:00:00",  // 响应时间
  "data": {}               // 业务数据
}
```

## 使用

```go
// 成功（controller handle）
return &healthCheckOutput{Body: response.New(apiVersion, data)}, nil

// 错误（humax 封装，勿直接使用）
response.NewError(version, []any{}, 0, message, trace)
```

## 约定

- 成功响应用 `response.New`，错误响应统一走 `common/humax.Error`（信封 + 状态码）。
- 业务 code 语义：`0` 成功，非 0 由业务模块定义，错误时 `message` 必填。
