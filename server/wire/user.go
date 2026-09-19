package wire

import (
	usercontroller "github.com/jasonlabz/generate-example-project/server/controller/user"
	usermanager "github.com/jasonlabz/generate-example-project/server/manager/user"
	userservice "github.com/jasonlabz/generate-example-project/server/service/user"
)

// NewUserController 组装 user 模块的生产依赖图。
//
// 依赖自下而上连接：Manager -> Service -> Controller。
// 接入真实数据库时，只需把 usermanager.NewManager() 换成持有 DAO 的构造器，上层不变。
func NewUserController() *usercontroller.Controller {
	userManager := usermanager.NewManager()
	userService := userservice.NewService(userManager)

	return usercontroller.NewController(userService)
}
