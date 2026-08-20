# Health Check Service

`Service` 表达健康检查业务域的应用用例，不感知 Huma、Gin 或 HTTP DTO。`Check` 和 `CheckReadiness` 虽对应不同路径，但共享同一业务域、依赖图和生命周期，因此由一个 Service 接口承载。

`NewService(manager.Manager)` 的名称不重复：域名已由包路径 `service/health_check` 表达。生产对象由 Wire 层连接；单元测试通过 `mocks/server/manager/health_check` 的 `MockManager` 替换直接下游。

```shell
bash script/go-mockgen.sh
go test ./server/service/health_check
```
