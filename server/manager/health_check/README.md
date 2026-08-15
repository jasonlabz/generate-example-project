# Health Check Manager

Manager 负责协调健康检查的基础设施依赖，不感知 Huma、Gin 或 HTTP DTO。

`interface.go` 定义 `HealthCheckManager` 和直接下游 `HealthProbe`；`manager.go` 只编排 Probe 并补充 Manager 语义的错误上下文；`local_probe.go` 是示例程序的无 I/O 实现。

生产对象只在 Wire 层连接。单元测试通过 `mocks/server/manager/health_check` 中的 `MockHealthProbe` 替换直接下游：

```shell
bash script/go-mockgen.sh
go test ./server/manager/health_check
```
