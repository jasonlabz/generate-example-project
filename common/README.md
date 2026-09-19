# common 通用层

跨层复用的框架无关能力。**common 不依赖 server 层**，可被任意层引用。

## 目录结构

```text
common/
├── humax/      huma 响应、分页、文件流与错误封装
├── consts/     常量
└── helper/     辅助函数（上下文取值等）
```

## 代码风格

- 包名小写单数；导出标识符 GoDoc 注释（中文说明业务意图）。
- 工具函数不访问组件实例；无响应式/副作用。
- 新增强通用能力优先放这里，业务专用能力放 `server/` 各层。

## 关键约定

- **响应信封**：所有 HTTP 响应统一 `humax.Envelope[T]`（成功 `humax.New`、错误 `humax.NewError`）。
- **huma 错误**：业务失败优先使用 `apperr` 目录中登记的错误码（`apperr.NotFound.WithMessage(...)`），
  由 `humax.FromError` 映射为登记的 HTTP 状态 + 业务 code；协议层面的失败用
  `humax.BusinessError(version, code, message)`。两者都返回统一的 `Envelope`，不退回 RFC7807。
  未知内部错误才映射为安全的 HTTP 500。
- 详见 [humax/README.md](humax/README.md)。
