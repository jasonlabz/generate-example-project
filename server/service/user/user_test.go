package user

import (
	"context"
	"fmt"
	"testing"

	"github.com/jasonlabz/potato/pointer"

	"github.com/jasonlabz/generate-example-project/server/service/user/body"
)

func Test_UserService(t *testing.T) {
	info, err := GetService().UpdateUserInfo(context.Background(), &body.UserUpdateFieldDto{
		UserID:   3,
		UserName: pointer.String("hello"),
		Phone:    pointer.String("1999999999999"),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(info)
}
