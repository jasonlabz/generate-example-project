# server 目录约定

`server` 保持当前轻量分层。

## 目录职责

```text
server/
  controller/                 # HTTP Controller，只处理入参、调用 service、返回响应
  router/                     # 根路由与中间件分层装配
  wire/                       # 组合根；按模块组装 Controller/Service/Manager/DAO
  service/
    {module}/
      interface.go            # Service 接口
      service.go              # Service 实现
      types.go                # Service 结果类型
  manager/{module}/           # 业务编排与 DAO/Probe 接口依赖
dal/db/dao/                   # DAO 接口与数据库适配实现（gentol 生成）
```

## 新增模块

以 `user` 模块为例，按 `health_check` 的结构复制：

```text
server/controller/user/       # register.go / controller.go / types.go / convertor.go
server/service/user/          # interface.go / service.go / types.go
server/manager/user/          # interface.go / manager.go / 具体依赖实现
server/wire/user/wire.go      # 对象图装配
```

Controller 不写业务逻辑，Service 不依赖 Gin。新增模块在 `server/wire/user/wire.go` 中装配对象图，并在 `server/router/router.go` 的 `registerRootAPI` 或 `registerV1GroupAPI` 中注册路由。中间件在 Engine 作用域统一注入；认证、授权等受保护逻辑放在受保护路由组内，不要放在公开根路由。
