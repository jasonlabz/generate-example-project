package health_check_test

import (
	"context"
	"errors"
	"testing"

	manager_mocks "github.com/jasonlabz/generate-example-project/mocks/server/manager/health_check"
	"github.com/jasonlabz/generate-example-project/server/service/health_check"
	"go.uber.org/mock/gomock"
)

func TestService_Check(t *testing.T) {
	tests := []struct {
		name       string
		managerErr error
	}{
		{name: "success"},
		{name: "wraps manager error", managerErr: errors.New("probe health: unavailable")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			manager := manager_mocks.NewMockManager(ctrl)
			manager.EXPECT().Check(gomock.Any()).Return(test.managerErr)

			result, err := health_check.NewService(manager).Check(context.Background())

			if test.managerErr == nil {
				if err != nil {
					t.Fatalf("Check() error = %v, want nil", err)
				}
				if result.Status != "success" {
					t.Fatalf("Check() status = %q, want %q", result.Status, "success")
				}
				return
			}
			if !errors.Is(err, test.managerErr) {
				t.Fatalf("Check() error = %v, want wrapped %v", err, test.managerErr)
			}
			if result.Status != "" {
				t.Fatalf("Check() result = %#v, want zero value on error", result)
			}
		})
	}
}

func TestService_CheckReadiness(t *testing.T) {
	ctrl := gomock.NewController(t)
	manager := manager_mocks.NewMockManager(ctrl)
	manager.EXPECT().CheckReadiness(gomock.Any()).Return(nil)

	result, err := health_check.NewService(manager).CheckReadiness(context.Background())

	if err != nil {
		t.Fatalf("CheckReadiness() error = %v, want nil", err)
	}
	if result.Status != "ready" {
		t.Fatalf("CheckReadiness() status = %q, want %q", result.Status, "ready")
	}
}
