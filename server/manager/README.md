# Manager 与 DAO 边界

Manager 是业务编排层，DAO 是持久化适配层。DAO 不应放在 Router 或 Service 中直接构造；推荐沿用 `dagine-dashboard/dal/db/dao` 的“接口 + impl”分离，但把数据库实例改为显式依赖：

```text
dal/db/dao/{module}_dao.go              # DAO 接口（可由 gentol 生成基础方法）
dal/db/dao/impl/{module}_dao_impl.go    # GORM/SQL 实现与事务适配
server/manager/{module}/interface.go    # Manager 接口
server/manager/{module}/manager.go       # 只注入该用例需要的最小 DAO 接口
server/wire/{module}/wire.go             # dao -> manager -> service -> controller
```

Manager 的结构体只保存 DAO 接口，不暴露 `*gorm.DB`，也不调用全局单例。事务由 Manager 的用例边界开启，DAO 从显式 `context.Context` 或事务句柄读取当前连接；跨多个 DAO 的写操作必须在同一个 Manager 事务中完成。DAO 只负责查询、持久化和数据库错误，不承载权限、状态机或 HTTP DTO 转换。

```go
type UserManager struct {
	users UserDAO
}

func NewUserManager(users UserDAO) UserManager {
	return UserManager{users: users}
}
```

Wire 中再接入具体实现：

```go
usersDAO := userdaoimpl.New(db)
usersManager := usermanager.NewUserManager(usersDAO)
usersService := userservice.New(usersManager)
return userwire.NewModule(usersService)
```

这样 Service 测试只替换 Manager Mock，Manager 测试只替换 DAO Mock，DAO 测试使用临时数据库或 SQL 存根；新增表或更换 ORM 不需要修改 Router 和 Controller。
