# Health Check Manager

`Manager` 负责协调健康检查域的基础设施依赖，不感知 Huma、Gin 或 HTTP DTO。`Check` 与 `CheckReadiness` 共享同一 Probe 依赖，因此归属一个 Manager 接口和一个实现。

`NewManager(probe)` 的域名由包路径 `manager/health_check` 表达。生产对象只在 Wire 层连接；单元测试通过 `MockHealthProbe` 替换直接下游。

```shell
bash script/go-mockgen.sh
go test ./server/manager/health_check
```
