# Wire（组合根）

Wire 是应用的组合根：只在这里构造具体实现并连接模块依赖。Router 从 Wire 取得 Controller，
不直接构造 Service、Manager 或基础设施。

Wire 可以导入各层的具体构造器，但不定义业务接口、不保存业务状态、不承载 HTTP DTO 转换或领域规则。

装配方向：

```text
wire -> controller -> service -> manager -> DAO/外部系统
```

## 文件组织：按模块拆文件，不拆目录

```text
server/wire/
├── doc.go            # 包说明：装配方向与约定
├── health_check.go   # NewHealthCheckController
├── user.go           # NewUserController
└── README.md
```

目录层级只用来区分**职责**（controller / service / manager / wire），业务边界由包名表达，
因此不在 wire 下再按模块复制一层目录。拆文件而非拆目录还有一个实际收益：每个模块的文件
只需导入自己那三层的包，import 别名互不干扰。

## 约定

- 每个业务模块一个文件，导出一个 `New<Module>Controller`：
  `NewHealthCheckController`、`NewUserController`。
- 函数体内按「依赖在前、使用者在后」的顺序自下而上组装，最后返回 Controller。
- 新增模块时新建 `wire/<module>.go`，并在 `server/router/router.go` 注册。

```go
// server/wire/user.go
package wire

import (
	usercontroller "github.com/jasonlabz/generate-example-project/server/controller/user"
	usermanager    "github.com/jasonlabz/generate-example-project/server/manager/user"
	userservice    "github.com/jasonlabz/generate-example-project/server/service/user"
)

func NewUserController() *usercontroller.Controller {
	userManager := usermanager.NewManager()
	userService := userservice.NewService(userManager)

	return usercontroller.NewController(userService)
}
```

由于 controller / service / manager 三层的包名与业务域同名（都叫 `user`），import 时统一用
`<module><layer>` 形式的别名（`usercontroller`、`userservice`、`usermanager`），避免阅读歧义。
按模块分文件后，一个文件里只会出现同一个模块的三个别名，读起来是连贯的。

## 注册路由

在 `server/router/router.go` 的 `registerRootAPI` 或 `registerV1GroupAPI` 中调用构造器：

```go
func registerV1GroupAPI(api huma.API) {
	wire.NewUserController().Register(api)
}
```

Router 集成测试覆盖该对象图；层级单元测试则分别通过生成 Mock 替换其直接下游。

```shell
bash script/go-mockgen.sh
go test ./server/router ./server/controller/health_check ./server/service/health_check ./server/manager/health_check
```
