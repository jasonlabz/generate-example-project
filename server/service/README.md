# service 服务层

业务用例编排层：在一个业务域包内组合 Manager 能力，**不直接访问 DAO/外部系统**。同一业务规则、依赖图与生命周期中的多个用例方法，应放在同一个 Service；不同 HTTP path 本身不是拆分 Service 的理由。

## 目录结构（每个业务域一个目录）

```text
server/service/<module>/
├── interface.go                    默认 Service 接口（供 mockgen 生成 mock）
├── <module>_service_impl.go        默认 Service 实现（依赖 manager 接口）
└── types.go                        业务结果类型
```

## 命名与依赖

- 域名由包路径表达：默认使用 `Service`、`serviceImpl`、`NewService(manager) Service`。
- Controller 依赖 Service 接口，测试通过 `mocks/` 下 mockgen 生成的 `MockService` 注入替身。
- 只有新增协作者具有独立依赖、事务边界或生命周期时，才新增同域的命名 Service，例如 `ExportService` / `NewExportService`。不因第二个方法或路由重命名默认 `NewService`。
- Service 方法返回业务类型（非 DTO）；DTO 转换交给 Controller 的 `convertor.go`。

## 接口 mock

在 `server/service/<module>/interface.go` 定义接口后：

```shell
bash script/go-mockgen.sh
```
