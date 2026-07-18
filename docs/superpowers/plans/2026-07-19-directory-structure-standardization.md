# 目录结构标准化重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 generate-example-project 重构为社区标准布局（cmd/ + internal/ + pkg/）的纯后端 Gin 模板，dao 层可注入测试、启动逻辑单份收编、附带完整工程化配套。

**Architecture:** 目录平移到 internal/ 与 pkg/，启动逻辑从两份重复的 main.go 收编进 internal/server；service 层保留 GetService() 单例，dao 层用 NewXxxDao(db) 构造 + SetGormDB 装载实现可测试性；gRPC 走 api/proto 定义 + internal/rpc 实现。

**Tech Stack:** Go 1.25 / Gin / potato（configx、gormx、log、httpx）/ gorm + jasonlabz/sqlite（测试内存库）/ grpc + protoc / golangci-lint v2 / GitHub Actions

**Spec:** `docs/superpowers/specs/2026-07-19-directory-structure-design.md`

**约定：** 所有命令在项目根 `/Users/lucas/workspace/code/golang/generate-example-project` 执行。当前分支 `gin`。每个 Task 结束时 `go build ./...` 必须通过。

---

### Task 1: 目录平移 + import 路径重写

**Files:**
- Move: `bootstrap/` → `internal/bootstrap/`、`global/resource/` → `internal/resource/`、`server/routers/` → `internal/router/`、`server/controller/` → `internal/controller/`、`server/service/` → `internal/service/`、`server/middleware/` → `internal/middleware/`、`common/ginx|consts|helper/` → `pkg/ginx|consts|helper/`
- Delete: `server/README.md`（内容 Task 10 并入根 README）
- Modify: 全部 `.go` 文件的 import 路径；`internal/router/router.go` 包名

- [ ] **Step 1: git mv 目录平移**

```bash
mkdir -p internal pkg
git mv bootstrap internal/bootstrap
git mv global/resource internal/resource
git mv server/routers internal/router
git mv server/controller internal/controller
git mv server/service internal/service
git mv server/middleware internal/middleware
git mv common/ginx pkg/ginx
git mv common/consts pkg/consts
git mv common/helper pkg/helper
git rm server/README.md
rmdir global common server 2>/dev/null || true
```

- [ ] **Step 2: 全局重写 import 路径**

注意顺序：`server/service` 的前缀替换同时正确覆盖 `server/service/health_check`。

```bash
find . -name '*.go' -not -path './.git/*' -print0 | xargs -0 perl -pi -e '
  s{github.com/jasonlabz/generate-example-project/bootstrap}{github.com/jasonlabz/generate-example-project/internal/bootstrap}g;
  s{github.com/jasonlabz/generate-example-project/global/resource}{github.com/jasonlabz/generate-example-project/internal/resource}g;
  s{github.com/jasonlabz/generate-example-project/server/routers}{github.com/jasonlabz/generate-example-project/internal/router}g;
  s{github.com/jasonlabz/generate-example-project/server/controller}{github.com/jasonlabz/generate-example-project/internal/controller}g;
  s{github.com/jasonlabz/generate-example-project/server/service}{github.com/jasonlabz/generate-example-project/internal/service}g;
  s{github.com/jasonlabz/generate-example-project/server/middleware}{github.com/jasonlabz/generate-example-project/internal/middleware}g;
  s{github.com/jasonlabz/generate-example-project/common/}{github.com/jasonlabz/generate-example-project/pkg/}g;
'
```

- [ ] **Step 3: routers 包更名为 router**

```bash
perl -pi -e 's/^package routers$/package router/' internal/router/router.go
# 调用方限根 main.go 与 cmd/example-server/main.go（Task 2 会删，此处先保证编译）
perl -pi -e 's/\brouters\.InitApiRouter\b/router.InitApiRouter/g' main.go cmd/example-server/main.go
```

- [ ] **Step 4: 编译验证**

Run: `go build ./... && go vet ./...`
Expected: 无输出（成功）。若报未使用 import / 找不到包，检查 Step 2 的替换是否遗漏该文件。

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: 目录平移至 internal/ 与 pkg/，社区标准布局"
```

---

### Task 2: internal/server 启动包 + 瘦 cmd/server + 删除重复入口

**Files:**
- Create: `internal/server/server.go`、`internal/server/http.go`、`internal/server/grpc.go`、`internal/server/pprof.go`、`internal/rpc/register.go`、`cmd/server/main.go`
- Delete: `main.go`（根）、`cmd/example-server/`
- Modify: `internal/router/router.go`（删 webroot 静态路由）、`Makefile`（build 路径，最小改动）、`conf/application.yaml`（pprof 端口冲突修复）

- [ ] **Step 1: 创建 internal/rpc/register.go（gRPC 注册点，Task 6 填充实现）**

```go
package rpc

import "google.golang.org/grpc"

// Register 注册所有 gRPC 服务实现，新增服务在此挂载。
func Register(srv *grpc.Server) {
}
```

- [ ] **Step 2: 创建 internal/server/server.go**

```go
package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/jasonlabz/generate-example-project/internal/bootstrap"
)

const shutdownTimeout = 5 * time.Second

// Run 按配置启动 HTTP/gRPC/pprof 服务，阻塞至 ctx 取消或收到退出信号，统一优雅退出。
func Run(ctx context.Context) {
	cfg := bootstrap.GetConfig()

	httpSrv := startHTTPServer(cfg)
	pprofSrv := startPProfServer(cfg)
	grpcSrv, grpcLis := startGRPCServer(cfg)

	if httpSrv == nil && pprofSrv == nil && grpcSrv == nil {
		log.Println("no service enabled, exiting")
		return
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-ctx.Done():
	case <-quit:
	}
	log.Println("Shutdown Server ...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	shutdownHTTP(shutdownCtx, "pprof server", pprofSrv)
	shutdownHTTP(shutdownCtx, "http server", httpSrv)
	shutdownGRPC(shutdownCtx, grpcSrv, grpcLis)
	log.Println("Server exiting")
}

func shutdownHTTP(ctx context.Context, name string, srv *http.Server) {
	if srv == nil {
		return
	}
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("%s shutdown failed: %v", name, err)
	}
}

func shutdownGRPC(ctx context.Context, srv *grpc.Server, lis net.Listener) {
	if srv == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		srv.Stop()
	}
	if lis != nil {
		_ = lis.Close()
	}
}
```

- [ ] **Step 3: 创建 internal/server/http.go**

```go
package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jasonlabz/potato/ginmetrics"

	"github.com/jasonlabz/generate-example-project/internal/bootstrap"
	"github.com/jasonlabz/generate-example-project/internal/router"
)

// buildEngine 装配 gin engine：运行模式、metrics 中间件、业务路由。
func buildEngine(c *bootstrap.Config) *gin.Engine {
	mode := gin.ReleaseMode
	if c.IsDebugMode() {
		mode = gin.DebugMode
	}
	gin.SetMode(mode)

	engine := router.InitApiRouter()

	prometheusConf := c.GetPrometheusConfig()
	if prometheusConf.Enable {
		m := ginmetrics.GetMonitor()
		m.SetMetricPath(prometheusConf.Path)
		m.SetSlowTime(10)
		m.SetDuration([]float64{0.1, 0.3, 1.2, 5, 10})
		m.Use(engine)
	}
	return engine
}

func startHTTPServer(c *bootstrap.Config) *http.Server {
	if !c.IsHTTPEnable() {
		return nil
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", c.GetHTTPPort()),
		Handler:      buildEngine(c),
		ReadTimeout:  c.GetHTTPReadTimeout(),
		WriteTimeout: c.GetHTTPWriteTimeout(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server listen: %v", err)
		}
	}()
	return srv
}
```

- [ ] **Step 4: 创建 internal/server/grpc.go**

```go
package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/jasonlabz/generate-example-project/internal/bootstrap"
	"github.com/jasonlabz/generate-example-project/internal/rpc"
)

func startGRPCServer(c *bootstrap.Config) (*grpc.Server, net.Listener) {
	if !c.IsGRPCEnable() {
		return nil, nil
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.GetGRPCPort()))
	if err != nil {
		log.Fatalf("grpc listen failed: %v", err)
	}

	grpcCfg := c.GetServerConfig().GRPC
	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     300 * time.Second,
			MaxConnectionAge:      1800 * time.Second,
			MaxConnectionAgeGrace: 30 * time.Second,
			Time:                  30 * time.Second,
			Timeout:               10 * time.Second,
		}),
		grpc.MaxConcurrentStreams(grpcCfg.MaxConcurrentStreams),
	)

	rpc.Register(srv)

	go func() {
		if serveErr := srv.Serve(lis); serveErr != nil && !errors.Is(serveErr, net.ErrClosed) {
			log.Printf("grpc server failed: %v", serveErr)
		}
	}()
	return srv, lis
}
```

- [ ] **Step 5: 创建 internal/server/pprof.go**

pprof 只保留独立端口方式（`net/http/pprof` 的 init 注册在 DefaultServeMux），不再往 gin 路由上挂。

```go
package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/jasonlabz/generate-example-project/internal/bootstrap"
)

func startPProfServer(c *bootstrap.Config) *http.Server {
	pprofConf := c.GetPProfConfig()
	if !pprofConf.Enable {
		return nil
	}

	srv := &http.Server{Addr: fmt.Sprintf(":%d", pprofConf.Port), Handler: nil}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("pprof server failed: %v", err)
		}
	}()
	return srv
}
```

- [ ] **Step 6: 创建 cmd/server/main.go（瘦入口 + swagger 注释）**

```go
package main

import (
	"context"

	"github.com/jasonlabz/generate-example-project/internal/bootstrap"
	"github.com/jasonlabz/generate-example-project/internal/server"
)

// @title			generate-example-project
// @version		1.0
// @description	基于 Gin 的标准项目模板
// @contact.name	your name
// @contact.email	mail_name@qq.com
// @BasePath		/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bootstrap.MustInit(ctx)
	server.Run(ctx)
}
```

- [ ] **Step 7: 删除重复入口，修改 router 与配置**

```bash
git rm main.go
git rm -r cmd/example-server
```

`internal/router/router.go` 删除 webroot 静态路由两行：

```go
	// 删除以下两行：
	staticRouter := router.Group("/server")
	staticRouter.Static("/", "webroot")
```

`conf/application.yaml` 修复 pprof 端口与 http 冲突（8080 → 6060）：

```yaml
    pprof:
      enable: false  # Enable PProf tool
      port: 6060
      enabled_endpoints: ["goroutine", "heap"]  # 指定启用的端点
```

Makefile 的 build 目标改为指向 cmd/server（根 main.go 已删，原命令会失败）：

```makefile
build: prepare
	go build -o $(WORKDIR)/bin/$(TARGETNAME) ./cmd/server
```

- [ ] **Step 8: 编译 + 冒烟验证**

Run: `go build ./... && go vet ./...`
Expected: 成功。

Run 冒烟（datasource/redis 默认 disable，MustInit 可空跑）：

```bash
go run ./cmd/server &
SERVER_PID=$!
sleep 3
curl -s http://127.0.0.1:8080/health-check
kill $SERVER_PID
```

Expected: 返回 JSON，包含 `"data":["success"]`。

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "refactor: 启动逻辑收编 internal/server，cmd/server 瘦入口，删除重复 main"
```

---

### Task 3: 移除前端与静态文件服务器痕迹

**Files:**
- Delete: `web/`
- Modify: `Makefile`（整体重写）、`Dockerfile`（整体重写）、`.gitignore`、`.dockerignore`

- [ ] **Step 1: 删除 web 目录**

```bash
git rm -r web
```

- [ ] **Step 2: 重写 Makefile**

完整替换为（去掉 frontend/copy-frontend/webroot/go.env，新增 build-all 与 lint）：

```makefile
# 工作目录变量（CURDIR 是 make 内置变量，跨平台兼容）
WORKDIR := $(CURDIR)
OUTDIR  := $(WORKDIR)/output

# 目标二进制名称
TARGETNAME = generate-example-project
ifeq ($(OS),Windows_NT)
  TARGETNAME = generate-example-project.exe
  OUTDIR_WIN := $(subst /,\,$(OUTDIR))
endif

GOPKGS := $(shell go list ./...)

all: clean test package

prepare:
	go env
	go mod download || go mod download -x

compile: build
build: prepare
	go build -o $(WORKDIR)/bin/$(TARGETNAME) ./cmd/server

# 编译全部入口（server/migrate/worker）
build-all: prepare
	go build -o $(WORKDIR)/bin/$(TARGETNAME) ./cmd/server
	go build -o $(WORKDIR)/bin/migrate ./cmd/migrate
	go build -o $(WORKDIR)/bin/worker ./cmd/worker

test: prepare
	go test -race -timeout=300s -v -cover $(GOPKGS) -coverprofile=coverage.out

lint:
	golangci-lint run

ifeq ($(OS),Windows_NT)
package: build
	-if exist $(OUTDIR_WIN) rmdir /s /q $(OUTDIR_WIN)
	mkdir $(OUTDIR_WIN)
	xcopy /E /I /Y /Q bin $(OUTDIR_WIN)\bin
	xcopy /E /I /Y /Q conf $(OUTDIR_WIN)\conf
else
package: build
	rm -rf $(OUTDIR)
	mkdir -p $(OUTDIR)
	cp -a bin $(OUTDIR)/bin
	cp -a conf $(OUTDIR)/conf
	tree $(OUTDIR) || ls -R $(OUTDIR)
endif

docker:
	docker build -t generate-example-project:latest .

ifeq ($(OS),Windows_NT)
clean:
	-if exist $(OUTDIR_WIN) rmdir /s /q $(OUTDIR_WIN)
	-if exist bin rmdir /s /q bin
else
clean:
	rm -rf $(OUTDIR) bin
endif

.PHONY: all prepare compile test package clean build build-all lint docker
```

- [ ] **Step 3: 重写 Dockerfile**

完整替换为（去掉前端 stage 与 webroot、过时的 DAGINE_* 注释）：

```dockerfile
# ============================
# Stage 1: 构建后端
# ============================
FROM golang:1.26-alpine3.23 AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o bin/generate-example-project ./cmd/server

# ============================
# Stage 2: 运行镜像
# ============================
FROM debian:bullseye-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/bin/generate-example-project ./bin/
COPY --from=builder /app/conf ./conf/

# HTTP 服务端口
EXPOSE 8080
# gRPC 服务端口（application.server.grpc.enable=true 时生效）
EXPOSE 8082

ENTRYPOINT ["./bin/generate-example-project"]
```

- [ ] **Step 4: 清理 .gitignore 与 .dockerignore**

`.gitignore` 删除 `web/dist/`、`web/node_modules/`、`webroot/`、`node_modules/` 四行，保留：

```
bin/
coverage.out
log/
output/
```

`.dockerignore` 检查并删除 web/webroot 相关条目（`grep -n 'web\|webroot' .dockerignore` 逐条确认）。

- [ ] **Step 5: 验证**

Run: `make clean build && ls bin/`
Expected: `bin/generate-example-project` 生成成功。

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor: 移除前端骨架与静态文件服务，Makefile/Dockerfile 纯后端化"
```

---

### Task 4: dal 层——装载模式 + User 示例（TDD）

**Files:**
- Create: `internal/dal/db/model/user.go`、`internal/dal/db/dao/base.go`、`internal/dal/db/dao/user.go`
- Test: `internal/dal/db/dao/user_test.go`
- Modify: `internal/bootstrap/bootstrap.go`（initDB 装载 dao）

- [ ] **Step 1: 先写 model（测试的依赖，无逻辑）**

`internal/dal/db/model/user.go`：

```go
package model

import "time"

// User 示例模型：演示 gentol 生成物的落点与风格，业务项目可删除。
type User struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;size:64;not null" json:"name"`
	Email     string    `gorm:"column:email;size:128" json:"email"`
	Status    int       `gorm:"column:status;default:0" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string { return "example_user" }
```

- [ ] **Step 2: 写失败测试**

`internal/dal/db/dao/user_test.go`：

```go
package dao

import (
	"context"
	"testing"

	"github.com/jasonlabz/sqlite"
	"gorm.io/gorm"

	"github.com/jasonlabz/generate-example-project/internal/dal/db/model"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestUserDaoCRUD(t *testing.T) {
	ctx := context.Background()
	d := NewUserDao(newTestDB(t))

	u := &model.User{Name: "alice", Email: "alice@example.com"}
	if err := d.Insert(ctx, u); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("expect auto increment id")
	}

	got, err := d.SelectByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got.Name != "alice" {
		t.Fatalf("expect name alice, got %s", got.Name)
	}

	if err := d.UpdateName(ctx, u.ID, "bob"); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err = d.SelectByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("select after update: %v", err)
	}
	if got.Name != "bob" {
		t.Fatalf("expect name bob, got %s", got.Name)
	}

	if err := d.DeleteByID(ctx, u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err = d.SelectByID(ctx, u.ID); err == nil {
		t.Fatal("expect not found after delete")
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/dal/... -v`
Expected: 编译错误 `undefined: NewUserDao`（dao 包还不存在实现）。

- [ ] **Step 4: 实现 base.go 与 user.go**

`internal/dal/db/dao/base.go`：

```go
package dao

import "gorm.io/gorm"

var defaultDB *gorm.DB

// SetGormDB 装载默认数据库连接，bootstrap 初始化 DB 后调用一次。
func SetGormDB(db *gorm.DB) { defaultDB = db }

// DefaultDB 返回已装载的默认连接。
func DefaultDB() *gorm.DB { return defaultDB }
```

`internal/dal/db/dao/user.go`：

```go
package dao

import (
	"context"

	"gorm.io/gorm"

	"github.com/jasonlabz/generate-example-project/internal/dal/db/model"
)

// UserDao 示例 DAO：演示构造装载模式，业务项目可删除。
type UserDao struct {
	db *gorm.DB
}

// NewUserDao 构造指定连接的 UserDao，单测中传入内存库。
func NewUserDao(db *gorm.DB) *UserDao { return &UserDao{db: db} }

// GetUserDao 返回使用默认装载连接的 UserDao，业务代码使用。
func GetUserDao() *UserDao { return NewUserDao(defaultDB) }

func (d *UserDao) Insert(ctx context.Context, user *model.User) error {
	return d.db.WithContext(ctx).Create(user).Error
}

func (d *UserDao) SelectByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	if err := d.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *UserDao) UpdateName(ctx context.Context, id int64, name string) error {
	return d.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).Update("name", name).Error
}

func (d *UserDao) DeleteByID(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&model.User{}, id).Error
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go mod tidy && go test ./internal/dal/... -v`
Expected: `PASS`（go mod tidy 将 `github.com/jasonlabz/sqlite` 提升为直接依赖）。

- [ ] **Step 6: bootstrap 装载 dao**

`internal/bootstrap/bootstrap.go` 的 `initDB` 函数，将结尾：

```go
	_, err = gormx.InitConfig(gormConfig)
	if err != nil {
		panic(err)
	}
	// dao.SetGormDB(db)
```

改为：

```go
	db, err := gormx.InitConfig(gormConfig)
	if err != nil {
		panic(err)
	}
	dao.SetGormDB(db)
```

并在 import 块加入：

```go
	"github.com/jasonlabz/generate-example-project/internal/dal/db/dao"
```

- [ ] **Step 7: 编译验证 + Commit**

Run: `go build ./... && go test ./internal/dal/... -cover`
Expected: 编译通过，测试 PASS。

```bash
git add -A
git commit -m "feat: dal 层装载模式与 User 示例 dao/model，bootstrap 完成 dao 装载"
```

---

### Task 5: service/controller 整理 + 单测

**Files:**
- Move: `internal/service/health_check/health_check_impl.go` → `internal/service/health_check/impl.go`
- Delete: `internal/service/health_check/helper.go`（空占位文件）
- Modify: `internal/service/health_check/impl.go`（修单例竞态）
- Test: `internal/service/health_check/impl_test.go`、`internal/controller/health_check_test.go`

- [ ] **Step 1: 文件整理**

```bash
git mv internal/service/health_check/health_check_impl.go internal/service/health_check/impl.go
git rm internal/service/health_check/helper.go
```

（`dto.go` 保留——它是模块 DTO 的约定落点。）

- [ ] **Step 2: 修复 GetService 竞态**

`internal/service/health_check/impl.go` 中：

```go
// 修改前（if svc != nil 提前返回绕过 once.Do，并发首调可能拿到未初始化完成的实例）：
var svc *Service
var once sync.Once

func GetService() service.HealthCheckService {
	if svc != nil {
		return svc
	}
	once.Do(func() {
		svc = &Service{}
	})

	return svc
}
```

```go
// 修改后：
var (
	svc  *Service
	once sync.Once
)

func GetService() service.HealthCheckService {
	once.Do(func() { svc = &Service{} })
	return svc
}
```

- [ ] **Step 3: 写 service 测试**

`internal/service/health_check/impl_test.go`：

```go
package health_check

import (
	"context"
	"testing"
)

func TestDoCheck(t *testing.T) {
	if got := GetService().DoCheck(context.Background()); got != "success" {
		t.Fatalf("expect success, got %s", got)
	}
}
```

Run: `go test ./internal/service/... -v`
Expected: PASS

- [ ] **Step 4: 写 controller HTTP 测试**

`internal/controller/health_check_test.go`：

```go
package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/health-check", HealthCheck)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health-check", nil)
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expect 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "success") {
		t.Fatalf("expect body contains success, got %s", body)
	}
}
```

Run: `go test ./internal/controller/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: health_check 单例竞态修复与文件整理，补 service/controller 单测"
```

---

### Task 6: gRPC proto 示例 + rpc 实现（TDD）

**Files:**
- Create: `api/proto/hello/v1/hello.proto`、`script/proto.sh`、`script/proto.ps1`、`internal/rpc/hello.go`
- Modify: `internal/rpc/register.go`
- Test: `internal/rpc/hello_test.go`
- Generated: `api/proto/hello/v1/hello.pb.go`、`api/proto/hello/v1/hello_grpc.pb.go`（提交入库，使用者无需装 protoc 也能编译）

- [ ] **Step 1: 写 proto 定义**

`api/proto/hello/v1/hello.proto`：

```proto
syntax = "proto3";

package hello.v1;

option go_package = "github.com/jasonlabz/generate-example-project/api/proto/hello/v1;hellov1";

// HelloService 示例服务：演示 proto 定义 → 生成 → internal/rpc 实现的完整链路。
service HelloService {
  rpc SayHello(SayHelloRequest) returns (SayHelloResponse);
}

message SayHelloRequest {
  string name = 1;
}

message SayHelloResponse {
  string greeting = 1;
}
```

- [ ] **Step 2: 写 script/proto.sh**

风格对齐 script/swag.sh（AUTO_INSTALL_TOOLS 模式）：

```bash
#!/bin/bash

set -euo pipefail

AUTO_INSTALL_TOOLS="${AUTO_INSTALL_TOOLS:-false}"
PROTOC_GEN_GO_VERSION="${PROTOC_GEN_GO_VERSION:-latest}"
PROTOC_GEN_GO_GRPC_VERSION="${PROTOC_GEN_GO_GRPC_VERSION:-latest}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

log() { echo "[$(date '+%H:%M:%S')] $1"; }
log_info() { log "INFO: $1"; }
log_error() { log "ERROR: $1"; exit 1; }

check_protoc() {
    command -v protoc &>/dev/null || \
        log_error "protoc 不存在。macOS: brew install protobuf；Debian/Ubuntu: apt install -y protobuf-compiler"
}

check_plugin() {
    local cmd="$1" pkg="$2" ver="$3"
    if command -v "$cmd" &>/dev/null; then
        return
    fi
    [[ "$AUTO_INSTALL_TOOLS" == "true" ]] || \
        log_error "$cmd 不存在。请先安装，或设置 AUTO_INSTALL_TOOLS=true"
    log_info "Installing $pkg@$ver..."
    go install "$pkg@$ver"
    command -v "$cmd" &>/dev/null || {
        local go_bin
        go_bin="$(go env GOPATH)/bin/$cmd"
        [[ -x "$go_bin" ]] || log_error "$cmd 安装完成但当前 PATH 和 GOPATH/bin 中仍不可用"
        export PATH="$PATH:$(go env GOPATH)/bin"
    }
}

main() {
    check_protoc
    check_plugin protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go "$PROTOC_GEN_GO_VERSION"
    check_plugin protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc "$PROTOC_GEN_GO_GRPC_VERSION"

    cd "$PROJECT_ROOT"
    local protos
    protos=$(find api/proto -name '*.proto')
    [[ -n "$protos" ]] || log_error "api/proto 下没有 .proto 文件"

    log_info "Generating protobuf code..."
    # shellcheck disable=SC2086
    protoc --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
        $protos
    log_info "Generation completed"
}

main "$@"
```

`script/proto.ps1`（PowerShell 等价版）：

```powershell
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot

function Log-Info($msg) { Write-Host "[$(Get-Date -Format 'HH:mm:ss')] INFO: $msg" }
function Log-Error($msg) { Write-Host "[$(Get-Date -Format 'HH:mm:ss')] ERROR: $msg"; exit 1 }

if (-not (Get-Command protoc -ErrorAction SilentlyContinue)) {
    Log-Error "protoc 不存在。Windows: choco install protoc 或从 github.com/protocolbuffers/protobuf/releases 下载"
}
foreach ($tool in @(
    @{ Cmd = "protoc-gen-go"; Pkg = "google.golang.org/protobuf/cmd/protoc-gen-go" },
    @{ Cmd = "protoc-gen-go-grpc"; Pkg = "google.golang.org/grpc/cmd/protoc-gen-go-grpc" }
)) {
    if (-not (Get-Command $tool.Cmd -ErrorAction SilentlyContinue)) {
        if ($env:AUTO_INSTALL_TOOLS -ne "true") {
            Log-Error "$($tool.Cmd) 不存在。请先安装，或设置 AUTO_INSTALL_TOOLS=true"
        }
        Log-Info "Installing $($tool.Pkg)@latest..."
        go install "$($tool.Pkg)@latest"
    }
}

Set-Location $ProjectRoot
$protos = Get-ChildItem -Path "api/proto" -Filter "*.proto" -Recurse | ForEach-Object { $_.FullName }
if (-not $protos) { Log-Error "api/proto 下没有 .proto 文件" }

Log-Info "Generating protobuf code..."
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative @protos
if ($LASTEXITCODE -ne 0) { Log-Error "protoc 生成失败" }
Log-Info "Generation completed"
```

- [ ] **Step 3: 生成 pb 代码**

Run: `command -v protoc || brew install protobuf`（本机 macOS；若已安装则跳过）
Run: `AUTO_INSTALL_TOOLS=true bash script/proto.sh`
Expected: 生成 `api/proto/hello/v1/hello.pb.go` 与 `api/proto/hello/v1/hello_grpc.pb.go`。

- [ ] **Step 4: 写失败测试（bufconn 全链路）**

`internal/rpc/hello_test.go`：

```go
package rpc

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	hellov1 "github.com/jasonlabz/generate-example-project/api/proto/hello/v1"
)

func TestSayHello(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	Register(srv)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	resp, err := hellov1.NewHelloServiceClient(conn).SayHello(context.Background(),
		&hellov1.SayHelloRequest{Name: "gopher"})
	if err != nil {
		t.Fatalf("say hello: %v", err)
	}
	if resp.GetGreeting() != "hello, gopher" {
		t.Fatalf("unexpected greeting: %s", resp.GetGreeting())
	}
}
```

Run: `go test ./internal/rpc/... -v`
Expected: FAIL——`Register` 为空未注册服务，报 `Unimplemented`（或 unknown service）。

- [ ] **Step 5: 实现 hello.go 并注册**

`internal/rpc/hello.go`：

```go
package rpc

import (
	"context"
	"fmt"

	hellov1 "github.com/jasonlabz/generate-example-project/api/proto/hello/v1"
)

// HelloServer HelloService 示例实现：rpc 层是薄适配层，业务逻辑应调用 internal/service。
type HelloServer struct {
	hellov1.UnimplementedHelloServiceServer
}

func NewHelloServer() *HelloServer { return &HelloServer{} }

func (s *HelloServer) SayHello(_ context.Context, req *hellov1.SayHelloRequest) (*hellov1.SayHelloResponse, error) {
	return &hellov1.SayHelloResponse{Greeting: fmt.Sprintf("hello, %s", req.GetName())}, nil
}
```

`internal/rpc/register.go` 改为：

```go
package rpc

import (
	"google.golang.org/grpc"

	hellov1 "github.com/jasonlabz/generate-example-project/api/proto/hello/v1"
)

// Register 注册所有 gRPC 服务实现，新增服务在此挂载。
func Register(srv *grpc.Server) {
	hellov1.RegisterHelloServiceServer(srv, NewHelloServer())
}
```

- [ ] **Step 6: 测试通过 + tidy**

Run: `go mod tidy && go test ./internal/rpc/... -v`
Expected: PASS（`google.golang.org/protobuf` 变为直接依赖）。

- [ ] **Step 7: Commit**

```bash
chmod +x script/proto.sh
git add -A
git commit -m "feat: gRPC hello 示例（proto 定义、生成脚本、rpc 实现与 bufconn 测试）"
```

---

### Task 7: internal/client 外部服务调用示例

**Files:**
- Create: `internal/client/demo/demo.go`

- [ ] **Step 1: 创建 demo client**

`internal/client/demo/demo.go`：

```go
// Package demo 演示调用 conf/servicer/demo.yaml 注册的外部服务。
// bootstrap.initServicer 启动时将 servicer 配置注册进 potato httpx，
// 业务侧通过 httpx.GetServiceClient(服务名) 获取带重试/超时配置的客户端。
package demo

import (
	"context"
	"fmt"

	"github.com/jasonlabz/potato/httpx"
)

// serviceName 对应 conf/servicer/demo.yaml 的 Name 字段。
const serviceName = "demo"

// Ping 调用 demo 服务的健康检查接口，演示 GET 与结果反序列化。
func Ping(ctx context.Context) (map[string]any, error) {
	cli, err := httpx.GetServiceClient(serviceName)
	if err != nil {
		return nil, fmt.Errorf("get %s client: %w", serviceName, err)
	}

	var result map[string]any
	if err := cli.Get(ctx, "/health-check", &result); err != nil {
		return nil, fmt.Errorf("call %s /health-check: %w", serviceName, err)
	}
	return result, nil
}
```

- [ ] **Step 2: 编译验证 + Commit**

Run: `go build ./...`
Expected: 成功（demo 依赖外部服务，不写单测，仅保证编译与用法示范）。

```bash
git add -A
git commit -m "feat: internal/client demo 示例，演示 servicer 配置的客户端用法"
```

---

### Task 8: 模板改名脚本

**Files:**
- Create: `script/rename.sh`、`script/rename.ps1`

- [ ] **Step 1: 写 rename.sh**

注意：先替换完整 module path，再替换裸服务名（module 包含服务名子串，顺序不能反）。perl 兼容 macOS/Linux（BSD sed 的 `-i` 语义差异因此绕开）。

```bash
#!/bin/bash
# 模板改名脚本：将模板 module path 与服务名替换为新项目的值。
# 用法: bash script/rename.sh <new-service-name> <new-module-path>
# 示例: bash script/rename.sh my-service github.com/you/my-service

set -euo pipefail

OLD_MODULE="github.com/jasonlabz/generate-example-project"
OLD_NAME="generate-example-project"

NEW_NAME="${1:-}"
NEW_MODULE="${2:-}"

if [[ -z "$NEW_NAME" || -z "$NEW_MODULE" ]]; then
    echo "用法: bash script/rename.sh <new-service-name> <new-module-path>"
    echo "示例: bash script/rename.sh my-service github.com/you/my-service"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

find . -type f \
    \( -name '*.go' -o -name '*.mod' -o -name '*.yaml' -o -name '*.yml' \
       -o -name '*.sh' -o -name '*.ps1' -o -name '*.md' -o -name '*.proto' \
       -o -name 'Makefile' -o -name 'Dockerfile' \) \
    -not -path './.git/*' -not -path './bin/*' -not -path './output/*' -print0 |
    xargs -0 perl -pi -e "s{\Q$OLD_MODULE\E}{$NEW_MODULE}g; s{\Q$OLD_NAME\E}{$NEW_NAME}g"

echo "module 已替换为 $NEW_MODULE，服务名已替换为 $NEW_NAME"
echo "后续: go mod tidy && make build"
```

- [ ] **Step 2: 写 rename.ps1**

```powershell
# 模板改名脚本（PowerShell）
# 用法: ./script/rename.ps1 <new-service-name> <new-module-path>
param(
    [Parameter(Mandatory = $true)][string]$NewName,
    [Parameter(Mandatory = $true)][string]$NewModule
)

$ErrorActionPreference = "Stop"
$OldModule = "github.com/jasonlabz/generate-example-project"
$OldName = "generate-example-project"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

$patterns = @('*.go', '*.mod', '*.yaml', '*.yml', '*.sh', '*.ps1', '*.md', '*.proto', 'Makefile', 'Dockerfile')
$files = Get-ChildItem -Recurse -File -Include $patterns |
    Where-Object { $_.FullName -notmatch '\\\.git\\|\\bin\\|\\output\\' }

foreach ($f in $files) {
    $content = Get-Content -Raw -Path $f.FullName
    $updated = $content.Replace($OldModule, $NewModule).Replace($OldName, $NewName)
    if ($updated -ne $content) {
        Set-Content -Path $f.FullName -Value $updated -NoNewline
    }
}

Write-Host "module 已替换为 $NewModule，服务名已替换为 $NewName"
Write-Host "后续: go mod tidy; make build"
```

- [ ] **Step 3: 冒烟验证（在临时副本执行，不污染仓库）**

```bash
rm -rf /tmp/rename-smoke
rsync -a --exclude .git --exclude bin --exclude output ./ /tmp/rename-smoke/
(cd /tmp/rename-smoke && bash script/rename.sh demo-app github.com/acme/demo-app && go build ./...)
grep -r "generate-example-project" /tmp/rename-smoke --include='*.go' -l | wc -l
```

Expected: `go build ./...` 成功；grep 计数为 `0`。

- [ ] **Step 4: Commit**

```bash
chmod +x script/rename.sh
git add script/rename.sh script/rename.ps1
git commit -m "feat: 模板改名脚本（module path 与服务名一键替换）"
```

---

### Task 9: golangci-lint + CI + 工具脚本默认值

**Files:**
- Create: `.golangci.yml`、`.github/workflows/ci.yml`
- Modify: `script/gentol.sh`、`script/gentol.ps1`、`script/swag.sh`、`script/swag.ps1`、存量代码 lint 修复

- [ ] **Step 1: 写 .golangci.yml（v2 格式）**

```yaml
version: "2"

run:
  timeout: 5m

linters:
  enable:
    - revive
  exclusions:
    generated: lax
    paths:
      - docs/swagger
      - api/proto

formatters:
  enable:
    - gofmt
    - goimports
  settings:
    goimports:
      local-prefixes:
        - github.com/jasonlabz/generate-example-project
  exclusions:
    paths:
      - docs/swagger
      - api/proto
```

（v2 默认启用 errcheck / govet / ineffassign / staticcheck / unused，额外开 revive。）

- [ ] **Step 2: 运行 lint 并修复存量问题**

Run: `command -v golangci-lint || brew install golangci-lint`
Run: `golangci-lint run`

已知需修复项（以实际输出为准，全部修代码、不加 nolint）：

1. `pkg/ginx/response.go`：`handleFileDownloadFromPath` / `handleFileDownloadFromReader` / `handleFileDownloadFromContent` 函数末尾的冗余 `return`（staticcheck S1023）——删除
2. `internal/middleware/logger.go`：`BodyLog.Write` 中 `bl.body.Write(b)` 未检查返回值（errcheck）——改为 `_, _ = bl.body.Write(b)`（bytes.Buffer 的 Write 恒不出错）
3. 其余 findings 逐条修复至 `golangci-lint run` 零输出

Expected 最终: `golangci-lint run` 退出码 0、无输出。

- [ ] **Step 3: 写 CI workflow**

`.github/workflows/ci.yml`：

```yaml
name: ci

on:
  push:
  pull_request:

jobs:
  build-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go build ./...
      - run: go test -race -cover ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: golangci/golangci-lint-action@v8
```

- [ ] **Step 4: 更新工具脚本默认值**

`script/gentol.sh` 第 19-20 行：

```bash
MODEL_DIR="${MODEL_DIR:-internal/dal/db/model}"
DAO_DIR="${DAO_DIR:-internal/dal/db/dao}"
```

`script/gentol.ps1`：同样将 `dal/db/model`、`dal/db/dao` 默认值改为 `internal/dal/db/model`、`internal/dal/db/dao`（grep 定位：`grep -n 'dal/db' script/gentol.ps1`）。

`script/swag.sh` 第 13-14 行：

```bash
SWAG_OUTPUT_DIR="${SWAG_OUTPUT_DIR:-docs/swagger}"
SWAG_MAIN_FILE="${SWAG_MAIN_FILE:-cmd/server/main.go}"
```

`script/swag.ps1`：同样把输出目录默认值改为 `docs/swagger`、入口默认值改为 `cmd/server/main.go`（grep 定位：`grep -n 'main.go\|docs' script/swag.ps1`）。

- [ ] **Step 5: 验证 + Commit**

Run: `go build ./... && go test ./... && golangci-lint run && make lint`
Expected: 全部通过、无输出。

```bash
git add -A
git commit -m "chore: golangci-lint 基线与 CI，修复存量 lint 问题，工具脚本默认值对齐新结构"
```

---

### Task 10: README 重写 + .claude/rules/backend.md + 最终验收

**Files:**
- Create: `.claude/rules/backend.md`
- Modify: `README.md`（整体重写）

- [ ] **Step 1: 写 .claude/rules/backend.md**

```markdown
# 后端分层规范

> Gin 项目模板的目录职责与依赖规则，写代码前先对照此文件。

## 目录职责

| 目录 | 职责 |
|------|------|
| `cmd/<name>/` | 二进制入口，只做 bootstrap + 启动，不写业务 |
| `internal/bootstrap/` | 配置加载、资源初始化编排（含 dao 装载） |
| `internal/server/` | HTTP/gRPC/pprof 启动与优雅退出 |
| `internal/router/` | 路由分组与中间件装配 |
| `internal/controller/` | HTTP 入参校验与响应组装 |
| `internal/rpc/` | gRPC 服务实现（proto 接口的薄适配层） |
| `internal/service/` | 业务逻辑；`{module}.go` 接口 + `{module}/impl.go` 实现 + `{module}/dto.go` |
| `internal/dal/db/` | 数据访问层，gentol 生成 dao/model 的落点 |
| `internal/client/` | 外部服务调用，配合 `conf/servicer/*.yaml` |
| `internal/middleware/` | 项目自有 gin 中间件 |
| `internal/resource/` | 全局客户端单例（logger/RMQ/Redis/ES） |
| `pkg/` | 可复用库（ginx/consts/helper），不依赖 internal |
| `api/proto/` | proto 定义与生成物 |

## 依赖规则（强制）

1. controller/rpc 只做入参校验和响应组装，不写业务逻辑、不碰 dao
2. service 不 import gin（入参用 dto，保持可脱离 HTTP 测试）
3. dao 不 import service（防循环依赖）
4. pkg 不 import internal（保证 pkg 可被抽走复用）

## 模式约定

- service：接口定义在 `internal/service/{module}.go`，实现用 `GetService()` + `sync.Once` 单例（`once.Do` 内初始化，不写 `if svc != nil` 提前返回）
- dao：`NewXxxDao(db)` 构造 + `GetXxxDao()` 取默认装载连接；单测传内存 SQLite（见 `internal/dal/db/dao/user_test.go`）
- 新增 HTTP 接口：controller 方法 + `internal/router` 注册 + service 接口/实现 + swagger 注释
- 新增 gRPC 接口：`api/proto` 定义 → `bash script/proto.sh` 生成 → `internal/rpc` 实现 → `rpc.Register` 挂载
- 响应统一走 `pkg/ginx`（`JsonResult` / `PaginationResult` / `FileResult`）
```

- [ ] **Step 2: 重写 README.md**

````markdown
# generate-example-project

基于 Gin 的标准项目模板：社区标准布局（`cmd/` + `internal/` + `pkg/`）、多入口、gRPC 示例、DAO 生成、完整工程化配套。

## 快速开始

```shell
## 1. 复制模板并改名（替换 module path 与服务名）
bash script/rename.sh my-service github.com/you/my-service

## 2. 修改配置
vi conf/application.yaml     # 服务端口、数据库、Redis 等，默认只开 HTTP :8080

## 3. 构建与运行
make build && ./bin/my-service
curl http://127.0.0.1:8080/health-check
```

## 目录结构

```
├── cmd/                  # 二进制入口：server（主服务）/ migrate / worker / tools
├── internal/             # 私有代码（编译器保证外部无法 import）
│   ├── bootstrap/        # 配置加载与资源初始化编排
│   ├── server/           # HTTP/gRPC/pprof 启动与优雅退出
│   ├── router/           # 路由注册
│   ├── controller/       # HTTP 入参适配层
│   ├── rpc/              # gRPC 服务实现
│   ├── service/          # 业务层（接口 + GetService() 单例实现 + dto）
│   ├── dal/db/           # 数据访问层（gentol 生成 dao/model 落点）
│   ├── client/           # 外部服务调用（配合 conf/servicer）
│   ├── middleware/       # gin 中间件
│   └── resource/         # 全局客户端单例
├── pkg/                  # 可复用库：ginx（响应封装）/ consts / helper
├── api/proto/            # proto 定义与生成物
├── conf/                 # application.yaml、migrations、seed、servicer
├── docs/swagger/         # swag 生成产物
└── script/               # gentol / swag / proto / rename 脚本
```

## 分层规则（强制）

1. controller/rpc 只做入参校验和响应组装，不写业务逻辑、不碰 dao
2. service 不 import gin
3. dao 不 import service
4. pkg 不 import internal

新增模块套路（以 `user` 为例）：

```
internal/service/user.go            # 接口
internal/service/user/impl.go       # 实现（GetService() 单例）
internal/service/user/dto.go        # 请求/响应 DTO
internal/controller/user.go         # HTTP 适配
internal/router/router.go           # 注册路由
```

## 常用命令

| 命令 | 说明 |
|------|------|
| `make build` | 编译主服务到 `bin/` |
| `make build-all` | 编译全部入口（server/migrate/worker） |
| `make test` | 单元测试（-race + 覆盖率） |
| `make lint` | golangci-lint 检查 |
| `make package` | 打包 bin + conf 到 `output/` |
| `make docker` | 构建 Docker 镜像 |

## 工具脚本

### gentol：DAO/Model 生成与 DDL 执行

若 `conf/db/<DB_CONF>`（默认 `db.toml`）存在则读取该文件，否则读 `conf/application.yaml`；环境变量始终优先。生成物落在 `internal/dal/db/{dao,model}`。

```shell
go install github.com/jasonlabz/gentol@master

export DB_TYPE=postgres DB_HOST=127.0.0.1 DB_PORT=5432 \
       DB_USER=postgres DB_PASS='your-password' DB_NAME=example DB_SCHEMA=public

bash script/gentol.sh                                          # 生成 DAO/Model
bash script/gentol.sh ddl conf/migrations/20240701_001_example_add_column.sql   # 执行 DDL
```

PowerShell 使用 `./script/gentol.ps1`。完整参数见 [script/README.md](script/README.md)。

### swag：接口文档生成

```shell
go install github.com/swaggo/swag/cmd/swag@v1.8.12
bash script/swag.sh        # 入口 cmd/server/main.go，输出 docs/swagger
```

debug 模式下访问 `http://ip:port/<服务名>/doc.html`（knife4go 美化版 swagger-ui）。

### proto：gRPC 代码生成

```shell
brew install protobuf      # 或 apt install protobuf-compiler
AUTO_INSTALL_TOOLS=true bash script/proto.sh
```

生成物与 `.proto` 同目录（`api/proto/<svc>/v1/`），已提交入库，不装 protoc 也能编译项目。

## 服务启用规则

`application.server.http.enable` 默认 `true`；`application.server.grpc.enable` 默认 `false`，需显式配置为 `true` 才会启动；pprof 由 `application.monitor.pprof` 控制，独立端口。

## 测试写法示例

| 层 | 示例文件 | 方式 |
|----|---------|------|
| dao | `internal/dal/db/dao/user_test.go` | `NewUserDao(内存 SQLite)` |
| service | `internal/service/health_check/impl_test.go` | 直接调用 |
| controller | `internal/controller/health_check_test.go` | httptest |
| rpc | `internal/rpc/hello_test.go` | bufconn |
````

- [ ] **Step 3: 最终验收**

```bash
make clean && make build && make test && make lint
```

Expected: 全绿。

```bash
# rename 冒烟（验收标准 2）
rm -rf /tmp/rename-smoke
rsync -a --exclude .git --exclude bin --exclude output ./ /tmp/rename-smoke/
(cd /tmp/rename-smoke && bash script/rename.sh demo-app github.com/acme/demo-app && go build ./... && go test ./...)
```

Expected: build/test 通过。

```bash
# gRPC 冒烟（验收标准 3，hello 已有 bufconn 测试，此处验证真端口）
# -0 slurp 模式跨行匹配，只命中 grpc 块的 enable（http/static 块不受影响）
perl -0pi -e 's/(grpc:\n\s+enable: )false/${1}true/' conf/application.yaml
grep -A1 'grpc:' conf/application.yaml   # 确认 enable: true
go run ./cmd/server &
SERVER_PID=$!
sleep 3
grpcurl -plaintext -d '{"name":"gopher"}' 127.0.0.1:8082 hello.v1.HelloService/SayHello || echo "grpcurl 未安装则跳过（bufconn 测试已覆盖）"
kill $SERVER_PID
git checkout conf/application.yaml
```

Expected: 有 grpcurl 时返回 `{"greeting": "hello, gopher"}`；无 grpcurl 时跳过（Task 6 的 bufconn 测试已机械化覆盖此验收项）。

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "docs: README 重写与 .claude/rules 后端分层规范"
```

---

## 验收清单（对照 spec 第 9 节）

- [ ] `make build` / `make test` / `make lint` 全绿
- [ ] rename 冒烟：改名后 `go build ./...` 通过、无 `generate-example-project` 残留
- [ ] HTTP `/health-check` 正常响应；gRPC hello 经 bufconn 测试验证（真端口 grpcurl 可选）
