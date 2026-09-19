# 构建脚本：仅面向 Linux 部署与类 Unix 构建环境（Linux / macOS / WSL / Git Bash）。
# 不维护 Windows cmd.exe 分支——跨平台差异交给 Docker 与 CI 处理。
# 本模板只包含后端，不含前端构建步骤。

# 工作目录变量（CURDIR 是 make 内置变量）
WORKDIR := $(CURDIR)
OUTDIR  := $(WORKDIR)/output

# 目标二进制名称
TARGETNAME = generate-example-project

# Go 包列表
GOPKGS := $(shell go list ./...)

# 设置编译时所需要的 Go 环境
export GOENV = $(WORKDIR)/go.env

# 执行编译，可使用命令 make 或 make all 执行
all: clean test package

# prepare 阶段，下载 Go 依赖
prepare:
	go env
	go mod download || go mod download -x

# compile 阶段，执行编译命令
compile: build
build: prepare
	go build -o $(WORKDIR)/bin/$(TARGETNAME)

# test 阶段，进行单元测试
test: prepare
	go test -race -timeout=300s -v -cover $(GOPKGS) -coverprofile=coverage.out

# package 阶段，对编译产出进行打包：
# 二进制 + 配置（服务按 ./conf 相对路径读取配置）+ 可选目录
package: build
	rm -rf $(OUTDIR)
	mkdir -p $(OUTDIR)
	cp -a bin $(OUTDIR)/bin
	cp -a conf $(OUTDIR)/conf
	if [ -d "data" ]; then cp -r data $(OUTDIR)/data; fi
	if [ -d "docs" ]; then cp -r docs $(OUTDIR)/docs; fi
	ls -R $(OUTDIR)

# clean 阶段，清除过程中的输出
clean:
	rm -rf $(OUTDIR) bin

# clean middle 阶段，只清除中间产物，保留 output
clean-middle:
	rm -rf bin

# avoid filename conflict and speed up build
.PHONY: all prepare compile test package clean clean-middle build
