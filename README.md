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

### 2、swagger使用
```shell
## swagger 依赖
go get "github.com/swaggo/files"
go get "github.com/swaggo/gin-swagger"


## swagger 命令行工具
go install github.com/swaggo/swag/cmd/swag@v1.8.12

###注释文档 main函数
// @title 这里写标题
// @version 这里写版本号
// @description 这里写描述信息
// @termsOfService http://swagger.io/terms/

// @contact.name 这里写联系人信息
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host 这里写接口服务的host
// @BasePath 这里写base path（eg：/api/v1）
func main() {}

### 接口层 controller
// @Summary 升级版帖子列表接口
// @Description 可按社区按时间或分数排序查询帖子列表接口
// @Tags 帖子相关接口
// @Accept application/json
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param object query models.ParamArtList(请求参数结构体) false "查询参数"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseArtList
// @Router /接口路由 [请求类型]
func GetArt(c *gin.Context) {}

### 结构体 struct
// 文章列表接口数据信息
type _ResponseArticle struct {
	Code    int               `json:"code"` // 业务状态码
	Message string            `json:"message"` // 提示信息
	Data    *[]model.Article  `json:"data"` // 数据
}

### 生成文档，执行：
swag init
}
```
