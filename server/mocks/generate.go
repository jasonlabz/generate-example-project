// Package mocks contains generated test doubles for server layer interfaces.
package mocks

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=mock_manager.go -package=mocks github.com/jasonlabz/generate-example-project/server/manager HealthCheckManager,HealthProbe
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=mock_service.go -package=mocks github.com/jasonlabz/generate-example-project/server/service HealthCheckService
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=mock_controller.go -package=mocks github.com/jasonlabz/generate-example-project/server/controller HealthCheckController
