package service

import (
	"context"

	"github.com/jasonlabz/generate-example-project/server/service/user/body"
)

type UserService interface {
	GetUserInfo(ctx context.Context, userID int64) (user *body.UserInfoDto, err error)
	GetUserList(ctx context.Context, params *body.UserListDto) (user []*body.UserInfoDto, err error)
	UpdateUserInfo(ctx context.Context, updateInfo *body.UserUpdateFieldDto) (user *body.UserInfoDto, err error)
	DeleteUser(ctx context.Context, userID int64) (err error)
	RegisterUser(ctx context.Context, userParams *body.UserRegisterDto) (user *body.UserResDto, err error)
	UserLoginOrLogout(ctx context.Context, userID int64) (user *body.UserInfoDto, err error)
}
