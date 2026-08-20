# Health Check Controller

该目录是一个可复制的 Huma Controller 模块示例：

```text
router -> wire/health_check (composition root) -> controller -> service -> manager -> probe
```

`Controller` 是具体类型，不为路由注册额外定义接口。它持有 `service.Service`，其 `Register(api)` 可以注册同一业务域的多个 HTTP 操作：本例同时维护 `/health-check` 和 `/readiness-check`。只有调用方确实需要替换多种 Controller 实现时，才抽取一个窄接口。

## 文件职责

| 文件 | 职责 |
| --- | --- |
| `register.go` | `Controller.Register` 注册本域所有 Huma operation 和 OpenAPI 元数据。 |
| `health_check_controller.go` | 保存注入的 `service.Service`，调用用例，并适配统一错误响应。 |
| `types.go` | 定义 HTTP 输出 DTO。 |
| `convertor.go` | 将 `service.Result` 转为 HTTP DTO，不让业务类型带 HTTP 语义。 |

生产对象只在 `server/wire/health_check` 组装；Controller 不自行构造 Service、Manager 或 Probe。Controller 单元测试直接注入 `mocks/server/service/health_check` 的 `MockService`。

```shell
bash script/go-mockgen.sh
go test ./server/controller/health_check ./server/service/health_check ./server/manager/health_check
go test ./server/router
```
