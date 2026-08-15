# Health Check Controller

这个目录是一个可复制的 Huma Controller 模块示例。它把 HTTP 注册、请求编排、协议类型和业务到协议的转换放在同一个业务域内，同时保持依赖单向：

```text
router -> wire/health_check (composition root) -> controller -> service -> manager -> probe
```

Router 只注册 HTTP 路由；`server/wire/health_check` 创建并连接 Probe、Manager、Service 和 Controller。Controller 不自行构造下层依赖，也不接触 Manager 或 Probe。

## 文件职责

| 文件 | 职责 |
| --- | --- |
| `register.go` | 声明 Controller 契约，并注册 Huma operation、路径和 OpenAPI 元数据。 |
| `controller.go` | 保存注入的 Service，调用用例，并将错误交给现有 `humax` 错误适配器。 |
| `types.go` | 定义 HTTP 输出 DTO 和本模块的 API 版本常量。 |
| `convertor.go` | 将 `service/health_check.HealthCheckResult` 转换为 HTTP DTO；不让业务类型带 HTTP 语义。 |

## 接口与测试

`HealthCheckService` 由 `server/service/health_check` 拥有，因为它表达应用用例的语义；Controller 通过 `NewHealthCheckController` 接收该接口。这既使依赖关系清晰，也能让 Controller 测试直接注入 `mocks/server/service/health_check` 的 `MockHealthCheckService`。Service 测试注入 `MockHealthCheckManager`，Manager 测试注入 `MockHealthProbe`，每层只替换自己的直接下层。

运行相关测试和更新 Mock：

```shell
bash script/go-mockgen.sh
go test ./server/controller/health_check ./server/service/health_check ./server/manager/health_check
go test ./server/router
```

`router.go` 中关闭 Huma 默认文档路径和 schema link hooks 的设置属于全局兼容边界：Knife4go 提供文档页面，业务响应不能被额外注入 `$schema` 字段或 `Link` header。
