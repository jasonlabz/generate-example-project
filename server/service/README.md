# service 服务层

业务用例编排层：一个用例对应一个 service 方法，通过调用 manager 组合能力完成业务逻辑。**不直接访问 DAO/外部系统**。

## 目录结构（每个业务模块一个目录）

```text
server/service/<module>/
├── interface.go   服务接口（供 mockgen 生成 mock；控制器依赖此接口）
├── service.go     服务实现（依赖 manager 接口，构造函数注入）
└── types.go       业务结果类型（如 HealthCheckResult）
```

## 代码风格

- 接口 + 实现分离：`XxxService` 接口 + `service` 私有实现 + `NewXxxService(manager) XxxService`。
- 依赖 manager **接口**（不是实现），测试时注入 mock（`mocks/` 下 mockgen 生成）。
- service 方法返回业务类型（非 DTO），转换交给 controller 的 convertor。
- 错误向上返回原始 error，由 controller 转换为带状态码的 humax 错误。

## 接口 mock

在 `server/service/<module>/interface.go` 定义接口后：

```shell
bash script/go-mockgen.sh
```

生成到 `mocks/`，测试示例见 `health_check_impl_test.go`。
