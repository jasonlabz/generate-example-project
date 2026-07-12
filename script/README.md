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
| `MODEL_DIR` | Model 输出目录 | `dal/db/model` |
| `DAO_DIR` | DAO 输出目录 | `dal/db/dao` |
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

## generate_idl.sh | generate_idl.ps1

解析 IDL 文件并生成 RPC 代码。

## swag.sh | swag.ps1

解析注释并生成 Swagger 文档。
