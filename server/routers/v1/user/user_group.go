package user

import (
	"github.com/gin-gonic/gin"

	"github.com/jasonlabz/generate-example-project/server/controller"
)

func RegisterUserGroup(userGroup *gin.RouterGroup) {
	userGroup.POST("/register", controller.RegisterUser)
	userGroup.PUT("/info", controller.UpdateUserInfo)
	userGroup.GET("/info/:user_id", controller.GetUserInfo)
	userGroup.PUT("/log_in_out/:user_id", controller.UserLogInOrLogout)
	userGroup.DELETE("/info/:user_id", controller.DeleteUser)
}
