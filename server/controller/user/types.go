package user

// getUserInput 路径参数：用 path tag 声明位置，huma 自动生成 OpenAPI 参数与校验。
type getUserInput struct {
	ID int64 `path:"id" minimum:"1" example:"1" doc:"用户 ID"`
}

// deleteUserInput 删除接口的路径参数。
type deleteUserInput struct {
	ID int64 `path:"id" minimum:"1" example:"1" doc:"用户 ID"`
}

// listUsersInput 查询参数。
//
// 约定：筛选条件不传表示"不限定"，因此不加 required；只有接口无法执行时才标记必填。
// 分页统一默认第一页、每页 200 条，并把页大小上限压到 200，避免一次拉全表。
type listUsersInput struct {
	Keyword  string `query:"keyword" doc:"按姓名或邮箱筛选；不传不筛选"`
	Page     int64  `query:"page" default:"1" minimum:"1" doc:"页码，从 1 开始"`
	PageSize int64  `query:"page_size" default:"200" minimum:"1" maximum:"200" doc:"每页条数，最大 200"`
}

// createUserBody 创建接口的请求体。
type createUserBody struct {
	Name  string `json:"name" required:"true" minLength:"1" maxLength:"64" doc:"用户名，全局唯一"`
	Email string `json:"email" required:"true" format:"email" doc:"邮箱"`
}

// createUserInput 请求体参数：字段固定命名为 Body，huma 按此识别请求体。
type createUserInput struct {
	Body createUserBody
}

// exportUsersInput 导出接口的查询参数。
type exportUsersInput struct {
	Format string `query:"format" enum:"csv,xlsx" default:"csv" doc:"导出格式"`
}

// userVO 用户响应模型，只暴露对外契约允许的字段。
type userVO struct {
	ID    int64  `json:"id" doc:"用户 ID"`
	Name  string `json:"name" doc:"用户名"`
	Email string `json:"email" doc:"邮箱"`
}
