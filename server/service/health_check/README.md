# Health Check Service

Service 表达健康检查这个应用用例，不感知 Huma、Gin 或 HTTP DTO。

`interface.go` 声明 Controller 需要的 `HealthCheckService`；`types.go` 保存用例结果；`service.go` 只调用直接下游 `HealthCheckManager` 并补充 Service 语义的错误上下文。

生产对象由 Wire 层连接。单元测试通过 `mocks/server/manager/health_check` 中的 `MockHealthCheckManager` 替换 Manager：

```shell
bash script/go-mockgen.sh
go test ./server/service/health_check
```
