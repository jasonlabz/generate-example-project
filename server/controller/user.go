package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jasonlabz/potato/consts"

	base "github.com/jasonlabz/generate-example-project/common/ginx"
	"github.com/jasonlabz/generate-example-project/server/service/user"
	body "github.com/jasonlabz/generate-example-project/server/service/user/body"
)

// GetUserInfo
//
//	@Summary	查询用户详情
//	@Tags		用户相关接口
//	@Accept		json
//	@Produce	json
//	@Param		user_id	path		string									true	"用户ID"
//	@Success	200		{object}	base.Response{data=body.UserInfoDto}	"ok"
//	@Router		/v1/user/info/{user_id} [get]
func GetUserInfo(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		base.ResponseErr(c, consts.APIVersionV1, err)
	}
	userInfo, err := user.GetService().GetUserInfo(c, userID)
	base.JsonResult(c, consts.APIVersionV1, userInfo, err)
}

// RegisterUser
//
//	@Summary	用户注册
//	@Tags		用户相关接口
//	@Accept		json
//	@Produce	json
//	@Param		user_info	body		body.UserRegisterDto	true	"用户信息"
//	@Success	200			{object}	base.Response			"ok"
//	@Router		/v1/user/register [post]
func RegisterUser(c *gin.Context) {
	params := &body.UserRegisterDto{}
	err := c.BindJSON(&params)
	if err != nil {
		base.ResponseErr(c, consts.APIVersionV1, err)
		return
	}
	registerUser, err := user.GetService().RegisterUser(c, params)
	base.JsonResult(c, consts.APIVersionV1, registerUser, err)
}

// UpdateUserInfo
//
//	@Summary	用户信息编辑
//	@Tags		用户相关接口
//	@Accept		json
//	@Produce	json
//	@Param		update_info	body		body.UserUpdateFieldDto	true	"用户信息"
//	@Success	200			{object}	base.Response			"ok"
//	@Router		/v1/user/info [put]
func UpdateUserInfo(c *gin.Context) {
	params := &body.UserUpdateFieldDto{}
	err := c.BindJSON(&params)
	if err != nil {
		base.ResponseErr(c, consts.APIVersionV1, err)
		return
	}
	updateUserInfo, err := user.GetService().UpdateUserInfo(c, params)
	base.JsonResult(c, consts.APIVersionV1, updateUserInfo, err)
}

// UserLogInOrLogout
//
//	@Summary	用户登录&登出
//	@Tags		用户相关接口
//	@Accept		json
//	@Produce	json
//	@Param		user_id	path		string			true	"用户ID"
//	@Param		status	query		string			true	"0|1 登录|登出"
//	@Success	200		{object}	base.Response	"ok"
//	@Router		/v1/user/log_in_out/{user_id} [put]
func UserLogInOrLogout(c *gin.Context) {
	base.ResponseOK(c, consts.APIVersionV1, nil)
}

// DeleteUser
//
//	@Summary	用户注销删除
//	@Tags		用户相关接口
//	@Accept		json
//	@Produce	json
//	@Param		user_id	path		string			true	"用户ID"
//	@Param		status	query		string			true	"0|1 登录|登出"
//	@Success	200		{object}	base.Response	"ok"
//	@Router		/v1/user/info/{user_id} [delete]
func DeleteUser(c *gin.Context) {
	base.ResponseOK(c, consts.APIVersionV1, nil)
}
