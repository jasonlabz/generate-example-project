package user

import (
	"strconv"
	"strings"

	"github.com/jasonlabz/generate-example-project/server/service/user"
)

// toUserVO 把业务模型转换为对外响应模型。
func toUserVO(item user.User) userVO {
	return userVO{ID: item.ID, Name: item.Name, Email: item.Email}
}

// toUserVOs 批量转换为响应模型；nil 输入返回空切片，保证 data 不输出 null。
func toUserVOs(items []user.User) []userVO {
	list := make([]userVO, 0, len(items))
	for _, item := range items {
		list = append(list, toUserVO(item))
	}
	return list
}

// csvHeader 是导出文件的表头，顺序与 toCSV 的数据列保持一致。
const csvHeader = "id,name,email"

// toCSV 把用户列表序列化为 CSV 内容。
// 这里用最简单的引号转义即可满足示例；真实项目应改用 encoding/csv。
func toCSV(items []userVO) []byte {
	var builder strings.Builder
	builder.WriteString(csvHeader)
	builder.WriteString("\n")
	for _, item := range items {
		builder.WriteString(strconv.FormatInt(item.ID, 10))
		builder.WriteString(",")
		builder.WriteString(quoteCSV(item.Name))
		builder.WriteString(",")
		builder.WriteString(quoteCSV(item.Email))
		builder.WriteString("\n")
	}
	return []byte(builder.String())
}

// quoteCSV 对含逗号、引号或换行的字段加引号并转义内部引号。
func quoteCSV(value string) string {
	if !strings.ContainsAny(value, ",\"\n") {
		return value
	}
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
