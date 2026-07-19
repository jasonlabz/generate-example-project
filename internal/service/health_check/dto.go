// Package health_check —— DTO 定义（业务模块替换或删除）
package health_check

// HealthStatus 健康检查响应
type HealthStatus struct {
	Status string `json:"status"`
}
