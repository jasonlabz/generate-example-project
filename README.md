# 工具介绍

> 服务启用规则：`application.server.http.enable` 默认 `true`；`application.server.grpc.enable`、`application.server.static.enable` 默认 `false`，需显式配置为 `true` 才会启动。
### 1、gentol 使用

项目通过统一脚本完成 DAO/Model 生成和 DDL 执行。若 `conf/db/<DB_CONF>`（默认 `db.toml`）存在，则只读取该 TOML 文件；否则才读取 `conf/application.yaml`。环境变量始终优先于被选中的配置文件，配置值后的空白加 `#` 注释会被忽略。

```shell
## 安装 gentol
go install github.com/jasonlabz/gentol@master

## 设置数据库环境变量
export DB_TYPE=postgres
export DB_HOST=127.0.0.1
export DB_PORT=5432
export DB_USER=postgres
export DB_PASS='your-password'
export DB_NAME=example
export DB_SCHEMA=public

## 生成 DAO/Model
bash script/gentol.sh

## 执行 DDL
bash script/gentol.sh ddl conf/migrations/20240701_001_example_add_column.sql
```

PowerShell 使用 `./script/gentol.ps1` 和 `./script/gentol.ps1 ddl <sql文件>`。完整环境变量和参数说明见 [script/README.md](script/README.md)。

### 2、API 文档

项目使用 Huma 生成 OpenAPI 3.0 文档，并使用 Knife4go 提供文档 UI。无需安装或运行额外的文档生成命令。

调试模式下，文档 UI 位于 `/{service}/doc.html`，生成的 OpenAPI 3.0 文档位于 `/{service}/v3/api-docs`。
