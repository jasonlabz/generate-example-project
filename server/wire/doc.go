// Package wire 是应用的组合根：集中构造具体实现并连接各模块依赖。
//
// Router 只从本包取得 Controller，不直接构造 Service、Manager 或基础设施。
// 本包可以导入各层的具体构造器，但不定义业务接口、不保存业务状态、
// 不承载 HTTP DTO 转换或领域规则。
//
// 装配方向：
//
//	wire -> controller -> service -> manager -> DAO/外部系统
//
// 文件按业务模块拆分（wire/<module>.go），但所有文件同属 wire 包，不按模块建子目录——
// 目录层级只用于区分职责，业务边界由包名表达。拆文件而非拆目录还有一个实际收益：
// 每个模块的文件只需导入自己那三层的包，import 别名不会互相干扰。
//
// 由于 controller / service / manager 的包名与业务域同名（例如都叫 user），import 时统一用
// <module><layer> 形式的别名（usercontroller / userservice / usermanager），避免阅读歧义。
package wire
