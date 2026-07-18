package health_check

import (
	"context"
	"sync"

	"github.com/jasonlabz/generate-example-project/internal/service"
)

var (
	svc  *Service
	once sync.Once
)

func GetService() service.HealthCheckService {
	once.Do(func() { svc = &Service{} })
	return svc
}

type Service struct {
}

func (s Service) DoCheck(_ context.Context) string {
	return "success"
}
