package user

// User 是用户业务模型，由 service 层定义并向 controller 暴露。
// manager 的存储模型 Record 与它一一对应，转换集中在 convertor 中完成，
// 两层的字段变化因此不会相互泄漏。
type User struct {
	ID    int64
	Name  string
	Email string
}
