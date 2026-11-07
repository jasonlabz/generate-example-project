# 构建脚本：仅面向 Linux 部署与类 Unix 构建环境（Linux / macOS / WSL / Git Bash）。

# 工作目录变量（CURDIR 是 make 内置变量）
WORKDIR := $(CURDIR)
OUTDIR  := $(WORKDIR)/output

# 目标二进制名称
TARGETNAME = generate-example-project

# Go 包列表
GOPKGS  := $$(go list ./...| grep -vE "vendor|bootstrap|db/dao|db/model|dao/model|dao/queries|interfaces|mocks")

# 设置编译时所需要的 Go 环境
export GOENV = $(WORKDIR)/go.env

# 执行编译，可使用命令 make 或 make all 执行
all: clean test frontend package

# prepare 阶段，下载 Go 依赖
prepare:
	go env
	go mod download || go mod download -x

# compile 阶段，执行编译命令
compile: build
build: prepare build-cmd
	go build -o $(WORKDIR)/bin/$(TARGETNAME)

# build-cmd 阶段，将 cmd 下所有子命令编译到 bin 目录（按目录名命名二进制）
# 直接使用 go list 输出的 import path，避免 Windows 下 .Dir 反斜杠被 shell 吞掉
build-cmd:
	for pkg in $$(go list ./cmd/...); do \
		name=$$(basename $$pkg); \
		echo "build cmd: $$name"; \
		go build -o $(WORKDIR)/bin/$$name $$pkg || exit 1; \
	done

# test 阶段，进行单元测试
test: prepare
	go test -race -timeout=300s -v -cover $(GOPKGS) -coverprofile=coverage.out

# 前端构建（可选）：webroot 下存在 package.json 才执行，产物在 webroot/dist
frontend:
	if [ -f "webroot/dist/package.json" ]; then cd $(WEBDIR) && pnpm install && pnpm build

# package 阶段，对编译产出进行打包：
# 二进制 + 配置（服务按 ./conf 相对路径读取配置）+ 可选目录
# 前端产物优先取 webroot/dist，其次取 webroot 根目录；占位状态（仅 .gitkeep）自动跳过
package: build
	rm -rf $(OUTDIR)
	mkdir -p $(OUTDIR)
	cp -a bin $(OUTDIR)/bin
	cp -a conf $(OUTDIR)/conf
	if [ -d "webroot/dist" ]; then cp -a webroot/dist $(OUTDIR)/webroot; fi
	if [ -d "data" ]; then cp -r data $(OUTDIR)/data; fi
	if [ -d "docs" ]; then cp -r docs $(OUTDIR)/docs; fi
	tree $(OUTDIR) || ls -R $(OUTDIR)

# clean 阶段，清除过程中的输出（webroot 是产物占位目录，只清理其中的构建产物）
clean:
	rm -rf $(OUTDIR) bin

# avoid filename conflict and speed up build
.PHONY: all prepare compile test frontend package clean build build-cmd
