# Wire Modules

Wire 是应用的组合根：只在这里构造具体实现并连接模块依赖。Router 从对应 Wire 模块取得 Controller，不直接构造 Service、Manager 或基础设施。

每个业务模块使用一个目录，例如 `server/wire/health_check/wire.go`。Wire 可以导入各层的具体构造器，但不定义业务接口、不保存业务状态、不承载 HTTP DTO 转换或领域规则。

模块 Wire 返回 `server/module.Module`，由 `server/wire/modules.go` 集中登记。Router 只遍历登记表，因此新增模块只需新增模块 Wire 并在登记表加入一项；根路由、API 路由和 v1 路由的控制流无需改动。Router 先用 `huma.NewGroup` 创建 `/{service}/api/v1` 层级，模块拿到对应的 `huma.API` 后可继续创建资源子 Group，并通过 `Group.UseMiddleware` 注入模块中间件。

Health Check 的装配方向为：

```text
Wire -> Controller -> Service -> Manager -> DAO/Probe
```

Router 集成测试覆盖该对象图；层级单元测试则分别通过生成 Mock 替换其直接下游。

```shell
bash script/go-mockgen.sh -o server/mocks
go test ./server/router ./server/controller/health_check ./server/service/health_check ./server/manager/health_check
```
