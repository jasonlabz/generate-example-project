package health_check_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jasonlabz/generate-example-project/server/mocks"
	"github.com/jasonlabz/generate-example-project/server/service/health_check"
	"go.uber.org/mock/gomock"
)

func TestService_Check(t *testing.T) {
	tests := []struct {
		name       string
		managerErr error
	}{
		{
			name: "success",
		},
		{
			name:       "wraps manager error",
			managerErr: errors.New("probe health: unavailable"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			manager := mocks.NewMockHealthCheckManager(ctrl)
			manager.EXPECT().Check(gomock.Any()).Return(test.managerErr)

			err := health_check.NewService(manager).Check(context.Background())

			if test.managerErr == nil {
				if err != nil {
					t.Fatalf("Check() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, test.managerErr) {
				t.Fatalf("Check() error = %v, want wrapped %v", err, test.managerErr)
			}
		})
	}
}
