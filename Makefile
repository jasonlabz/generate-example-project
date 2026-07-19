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
	golangci-lint run 2>/dev/null || $$(go env GOPATH)/bin/golangci-lint run

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
