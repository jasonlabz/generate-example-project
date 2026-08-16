# common 通用层

跨层复用的框架无关能力。**common 不依赖 server 层**，可被任意层引用。

## 目录结构

```text
common/
├── response/   统一响应信封 Envelope（code/message/data/version/current_time）
├── humax/      huma 响应封装：Output[T] 成功响应、Error 带状态码错误（StatusError）
├── ginx/       gin 工具（分页等，仅非 huma 场景使用）
├── consts/     常量
└── helper/     辅助函数（上下文取值等）
```

## 代码风格

- 包名小写单数；导出标识符 GoDoc 注释（中文说明业务意图）。
- 工具函数不访问组件实例；无响应式/副作用。
- 新增强通用能力优先放这里，业务专用能力放 `server/` 各层。

## 关键约定

- **响应信封**：所有 HTTP 响应统一 `response.Envelope[T]`（成功 `response.New`、错误 `response.NewError`）。
- **huma 错误**：handler 错误必须携带状态码（`humax.Error` 实现 `huma.StatusError`），
  新状态码构造函数仿照 `humax.InternalServerError` 增加。
- 详见 [response/README.md](response/README.md) 与 [humax/README.md](humax/README.md)。
