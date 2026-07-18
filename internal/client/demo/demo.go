// Package demo 演示调用 conf/servicer/demo.yaml 注册的外部服务。
// bootstrap.initServicer 启动时将 servicer 配置注册进 potato httpx，
// 业务侧通过 httpx.GetServiceClient(服务名) 获取带重试/超时配置的客户端。
package demo

import (
	"context"
	"fmt"

	"github.com/jasonlabz/potato/httpx"
)

// serviceName 对应 conf/servicer/demo.yaml 的 Name 字段。
const serviceName = "demo"

// Ping 调用 demo 服务的健康检查接口，演示 GET 与结果反序列化。
func Ping(ctx context.Context) (map[string]any, error) {
	cli, err := httpx.GetServiceClient(serviceName)
	if err != nil {
		return nil, fmt.Errorf("get %s client: %w", serviceName, err)
	}

	var result map[string]any
	if err := cli.Get(ctx, "/health-check", &result); err != nil {
		return nil, fmt.Errorf("call %s /health-check: %w", serviceName, err)
	}
	return result, nil
}
