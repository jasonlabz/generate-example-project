package user

import (
	"context"
	"sync"

	"github.com/jasonlabz/potato/errors"
	"github.com/jasonlabz/potato/log"
	"github.com/jasonlabz/potato/times"

	"github.com/jasonlabz/generate-example-project/common/consts"
	"github.com/jasonlabz/generate-example-project/common/helper"
	"github.com/jasonlabz/generate-example-project/dal/db/dao"
	"github.com/jasonlabz/generate-example-project/dal/db/dao/impl"
	"github.com/jasonlabz/generate-example-project/dal/db/model"
	"github.com/jasonlabz/generate-example-project/server/service"
	"github.com/jasonlabz/generate-example-project/server/service/user/body"
)

var svc *Service
var once sync.Once

func GetService() service.UserService {
	if svc != nil {
		return svc
	}
	once.Do(func() {
		svc = &Service{
			userDao: impl.GetUserDao(),
		}
	})

	return svc
}

type Service struct {
	userDao dao.UserDao
}

func (s *Service) GetUserInfo(ctx context.Context, userID int64) (userRes *body.UserInfoDto, err error) {
	logger := log.GetLogger()
	defer logger.Sync()
	userRes = &body.UserInfoDto{}
	user, err := s.userDao.SelectOneByPrimaryKey(ctx, userID)
	if err != nil {
		logger.WithError(err).Error(ctx, "get user error")
		err = errors.ErrDALOperation.WithErr(err)
		return
	}
	userRes.UserID = user.UserID
	userRes.UserName = user.Nickname
	userRes.Gender = user.Gender
	userRes.RegisterIP = user.RegisterIP
	userRes.RegisterTime = user.RegisterTime

	return
}

func (s *Service) GetUserList(ctx context.Context, params *body.UserListDto) (users []*body.UserInfoDto, err error) {
	logger := log.GetLogger()
	defer logger.Sync()
	users = make([]*body.UserInfoDto, 0)
	userCondition := &model.UserCondition{}
	if len(params.Nickname) != 0 {
		userCondition.NicknameIsLike(params.Nickname)
	}
	if len(params.Phone) != 0 {
		userCondition.PhoneIsLike(params.Phone)
	}
	userCondition.StatusEqualTo(consts.UserStatusNormal)
	column := userCondition.ColumnInfo()
	userList, err := s.userDao.SelectRecordByCondition(ctx, userCondition.Build(), column.UserID, column.Nickname, column.Password)
	if err != nil {
		logger.WithError(err).Error(ctx, "get user error")
		err = errors.ErrDALOperation.WithErr(err)
		return
	}
	for _, item := range userList {
		users = append(users, &body.UserInfoDto{
			UserName:     item.Nickname,
			UserID:       item.UserID,
			Phone:        item.Phone,
			Gender:       item.Gender,
			RegisterIP:   item.RegisterIP,
			RegisterTime: item.RegisterTime,
		})
	}
	return
}

func (s *Service) UpdateUserInfo(ctx context.Context, updateInfo *body.UserUpdateFieldDto) (user *body.UserInfoDto, err error) {
	logger := log.GetLogger()
	defer logger.Sync()
	updateUser := &model.User{
		UserID: updateInfo.UserID,
	}
	if updateInfo.UserName != nil {
		updateUser.Nickname = *updateInfo.UserName
	}
	if updateInfo.Phone != nil {
		updateUser.Phone = *updateInfo.Phone
	}
	if updateInfo.Password != nil {
		updateUser.Password = *updateInfo.Password
	}
	if updateInfo.Gender != nil {
		updateUser.Gender = *updateInfo.Gender
	}
	_, err = s.userDao.UpsertRecord(ctx, updateUser)
	if err != nil {
		return
	}
	return
}

func (s *Service) DeleteUser(ctx context.Context, userID int64) (err error) {
	logger := log.GetLogger()
	defer logger.Sync()

	_, err = s.userDao.DeleteByPrimaryKey(ctx, userID)

	return
}

func (s *Service) UserLoginOrLogout(ctx context.Context, userID int64) (user *body.UserInfoDto, err error) {
	// TODO implement me
	panic("implement me")
}

func (s *Service) RegisterUser(ctx context.Context, userParams *body.UserRegisterDto) (res *body.UserResDto, err error) {
	logger := log.GetLogger()
	defer logger.Sync()
	res = &body.UserResDto{}
	remoteIP := helper.GetClientIP(ctx)
	now := times.Now()
	registerUser := &model.User{
		Nickname:      userParams.Nickname,
		Avatar:        userParams.Avatar,
		Gender:        userParams.Gender,
		Password:      userParams.Password,
		Phone:         userParams.Phone,
		RegisterIP:    remoteIP,
		RegisterTime:  now,
		LastLoginIP:   remoteIP,
		LastLoginTime: now,
		CreateTime:    now,
		UpdateTime:    now,
	}
	_, err = s.userDao.Insert(ctx, registerUser)
	if err != nil {
		err = errors.ErrDALOperation.WithErr(err)
		return
	}
	res.UserName = registerUser.Nickname
	res.Gender = registerUser.Gender
	res.UserID = registerUser.UserID
	res.RegisterIP = registerUser.RegisterIP
	res.RegisterTime = registerUser.RegisterTime
	return
}
