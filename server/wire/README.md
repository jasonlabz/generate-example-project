# Wire Modules

Wire 是应用的组合根：只在这里构造具体实现并连接模块依赖。Router 从对应 Wire 模块取得 Controller，不直接构造 Service、Manager 或基础设施。

每个业务模块使用一个目录，例如 `server/wire/health_check/wire.go`。Wire 可以导入各层的具体构造器，但不定义业务接口、不保存业务状态、不承载 HTTP DTO 转换或领域规则。

Health Check 的装配方向为：

```text
Wire -> Controller -> Service -> Manager -> Probe
```

Router 集成测试覆盖该对象图；层级单元测试则分别通过生成 Mock 替换其直接下游。

```shell
bash script/go-mockgen.sh -o server/mocks
go test ./server/router ./server/controller/health_check ./server/service/health_check ./server/manager/health_check
```
