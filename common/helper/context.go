package helper

import (
	"context"

	"github.com/jasonlabz/potato/consts"
)

// stringValue returns the string stored under key in ctx, or "" when the value
// is missing or not a string. It avoids a runtime panic on a missing context key.
func stringValue(ctx context.Context, key string) string {
	v, ok := ctx.Value(key).(string)
	if !ok {
		return ""
	}
	return v
}

func GetClientIP(ctx context.Context) string {
	return stringValue(ctx, consts.ContextClientAddr)
}

func GetUserID(ctx context.Context) string {
	return stringValue(ctx, consts.ContextUserID)
}

func GetToken(ctx context.Context) string {
	return stringValue(ctx, consts.ContextToken)
}
