# manager 技术能力层

Manager 封装某一业务域所需的外部系统与基础设施协调能力（DAO、Redis、外部 API 等），不感知 Huma、Gin 或 HTTP DTO，也不承载业务判断。

## 目录结构（每个业务域一个目录）

```text
server/manager/<module>/
├── interface.go                    默认 Manager 与下游能力接口
├── <module>_manager_impl.go        默认 Manager 实现
└── <dependency>.go                 具体下游适配或客户端
```

## 命名与边界

- 域名由包路径表达：默认使用 `Manager`、`managerImpl`、`NewManager(dependency) Manager`。
- 一个 Manager 可以提供共享技术依赖的一组操作；不同 Service 方法不自动意味着需要拆 Manager。
- 只有新增协作者具有独立依赖、事务或生命周期时，才新增同域的命名 Manager。
- Manager 持有最小化的 DAO/Client 接口；事务在 Manager 的用例边界内协调，DAO 只负责持久化与数据库错误。
- Manager 测试替换其直接下游 mock；Service 测试只替换 Manager mock。

生产对象由 `server/wire/<module>` 连接为 `dependency -> manager -> service -> controller`。
