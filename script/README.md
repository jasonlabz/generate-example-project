# 脚本说明

## gentol.sh | gentol.ps1

统一处理数据库代码生成和 DDL 执行：

| 模式 | Bash | PowerShell |
|------|------|------------|
| 生成 DAO/Model | `bash script/gentol.sh` | `./script/gentol.ps1` |
| 执行 DDL | `bash script/gentol.sh ddl <sql文件>` | `./script/gentol.ps1 ddl <sql文件>` |

数据库连接信息通过脚本外的环境变量设置。若 `conf/db/<DB_CONF>` 存在，则只读取该 TOML 文件；否则才读取 `conf/application.yaml`。`DB_CONF` 默认值为 `db.toml`，且环境变量始终优先于被选中的配置文件。

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `GENTOL_CMD` | gentol 命令或可执行文件路径 | `gentol` |
| `DB_TYPE` | 数据库类型：`mysql`、`postgres`、`sqlserver`、`oracle`、`sqlite`、`dm` | 无 |
| `DSN` | 完整数据库连接串；设置后无需分项连接变量 | 无 |
| `DB_HOST` | 数据库地址 | 无 |
| `DB_PORT` | 数据库端口 | 无 |
| `DB_USER` | 数据库用户 | 无 |
| `DB_PASS` | 数据库密码 | 无 |
| `DB_NAME` | 数据库名；SQLite 时填写数据库文件路径 | 无 |
| `DB_SCHEMA` | 数据库 Schema | 无 |
| `DB_CONF` | TOML 数据库配置文件名，位于 `conf/db/` | `db.toml` |
| `TABLES` | 需要生成的表，多个表使用逗号分隔 | 全部表 |
| `MODEL_DIR` | Model 输出目录 | `internal/dal/db/model` |
| `DAO_DIR` | DAO 输出目录 | `internal/dal/db/dao` |
| `ONLY_MODEL` | 是否只生成 Model | `false` |
| `USE_SQL_NULLABLE` | 是否使用 `sql.Null*` 类型 | `false` |
| `RUN_GOFMT` | 是否执行 gofmt | `true` |
| `GEN_HOOK` | 是否生成 Hook | `true` |

### Bash 示例

```shell
export DB_TYPE=postgres
export DB_HOST=127.0.0.1
export DB_PORT=5432
export DB_USER=postgres
export DB_PASS='your-password'
export DB_NAME=example
export DB_SCHEMA=public
export TABLES=user,order

# 生成 DAO/Model
bash script/gentol.sh

# 执行 DDL
bash script/gentol.sh ddl conf/migrations/20240701_001_example_add_column.sql
```

### TOML 配置示例

设置 `DB_CONF=dev.toml` 后，脚本会读取 `conf/db/dev.toml`。值后的空白加 `#` 注释会自动忽略：

```toml
DBDriver = "postgres" # 数据库类型
Host = "127.0.0.1"    # 地址
Port = 5432             # 端口
Username = "postgres"
Password = "your-password"
DBName = "example"
SchemaName = "public"
```

也可以只设置 `DB_TYPE` 和 `DSN`：

```shell
export DB_TYPE=postgres
export DSN='user=postgres password=your-password host=127.0.0.1 port=5432 dbname=example sslmode=disable'
bash script/gentol.sh
```

### PowerShell 示例

```powershell
$env:DB_TYPE = "postgres"
$env:DB_HOST = "127.0.0.1"
$env:DB_PORT = "5432"
$env:DB_USER = "postgres"
$env:DB_PASS = "your-password"
$env:DB_NAME = "example"
$env:DB_SCHEMA = "public"
$env:TABLES = "user,order"

# 生成 DAO/Model
./script/gentol.ps1

# 执行 DDL
./script/gentol.ps1 ddl conf/migrations/20240701_001_example_add_column.sql
```

## proto.sh | proto.ps1

生成 `api/proto` 下所有 `.proto` 文件的 Go 代码（`*.pb.go` / `*_grpc.pb.go`）。生成物与 `.proto` 文件同目录（`paths=source_relative`），并提交入库。

| 操作 | Bash | PowerShell |
|------|------|------------|
| 生成 proto 代码 | `bash script/proto.sh` | `./script/proto.ps1` |

前置条件：需已安装 `protoc`（macOS: `brew install protobuf`；Windows: `choco install protoc`）。`protoc-gen-go` / `protoc-gen-go-grpc` 插件缺失时，设置 `AUTO_INSTALL_TOOLS=true` 可自动 `go install`。

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `AUTO_INSTALL_TOOLS` | 插件缺失时是否自动安装 | `false` |
| `PROTOC_GEN_GO_VERSION` | protoc-gen-go 版本 | 跟随 go.mod 的 `google.golang.org/protobuf` 版本 |
| `PROTOC_GEN_GO_GRPC_VERSION` | protoc-gen-go-grpc 版本 | `v1.6.2` |

## swag.sh | swag.ps1

解析注释并生成 Swagger 文档。
