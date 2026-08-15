# server 目录约定

`server` 保持当前轻量分层，不做额外目录搬迁。

## 目录职责

```text
server/
  controller/                 # HTTP Controller，只处理入参、调用 service、返回响应
  module/                     # Router/Wire 共用的模块注册契约
  router/                     # 根路由与中间件分层装配
  wire/                       # 组合根；按模块组装 Controller/Service/Manager/DAO
  service/
    {module}/
      interface.go            # Service 接口
      service.go              # Service 实现
      body/
        request.go            # 请求 DTO
        response.go           # 响应 DTO
  manager/{module}/            # 业务编排与 DAO 接口依赖
dal/db/dao/                    # DAO 接口与数据库适配实现
```

## 新增模块

以 `user` 模块为例：

```text
server/controller/user.go
server/service/user.go
server/service/user/user_impl.go
server/service/user/body/request.go
server/service/user/body/response.go
```

Controller 不写业务逻辑，Service 不依赖 Gin。Router 只遍历 `server/wire.Modules()`；新增模块在自己的 Wire 中返回 `server/module.Module`，再在注册表增加一项。中间件按 Engine、API、模块路由三个作用域注入，认证等受保护逻辑不要放在公开根路由。
