# Generated Mocks

这个目录只保存 GoMock 生成结果，不手工添加或修改 Go 文件。

在 Git Bash 中从项目根执行：

```shell
bash script/go-mockgen.sh -o server/mocks
```

在 PowerShell 中，可显式调用 Git Bash：

```powershell
& 'C:\Program Files\Git\bin\bash.exe' script/go-mockgen.sh -o server/mocks
```

常规源码按项目相对路径镜像，例如 `server/service/health_check/interface.go` 生成到 `server/mocks/server/service/health_check/mock_interface.go`。只有导出 interface 会生成外部 Mock；私有 interface 可能引用同包私有类型。`internal/` 下的源码必须生成到其 `internal` 父目录的 `.mocks/`，以满足 Go 的 internal 导入可见性规则。

生成规则的回归测试：

```shell
go test ./script -run TestGoMockgen_GeneratesMirroredMocksAndSkipsOutput
```
