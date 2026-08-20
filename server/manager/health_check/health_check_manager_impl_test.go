package health_check_test

import (
	"context"
	"errors"
	"testing"

	manager_mocks "github.com/jasonlabz/generate-example-project/mocks/server/manager/health_check"
	"github.com/jasonlabz/generate-example-project/server/manager/health_check"
	"go.uber.org/mock/gomock"
)

func TestManager_Check(t *testing.T) {
	tests := []struct {
		name     string
		probeErr error
		wantErr  error
	}{
		{name: "success"},
		{name: "wraps probe error", probeErr: errors.New("probe unavailable"), wantErr: errors.New("probe unavailable")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			probe := manager_mocks.NewMockHealthProbe(ctrl)
			probe.EXPECT().Probe(gomock.Any()).Return(test.probeErr)

			err := health_check.NewManager(probe).Check(context.Background())

			if test.wantErr == nil {
				if err != nil {
					t.Fatalf("Check() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, test.probeErr) {
				t.Fatalf("Check() error = %v, want wrapped %v", err, test.probeErr)
			}
		})
	}
}

func TestManager_CheckReadiness(t *testing.T) {
	ctrl := gomock.NewController(t)
	probe := manager_mocks.NewMockHealthProbe(ctrl)
	probe.EXPECT().Probe(gomock.Any()).Return(nil)

	err := health_check.NewManager(probe).CheckReadiness(context.Background())

	if err != nil {
		t.Fatalf("CheckReadiness() error = %v, want nil", err)
	}
}

func TestLocalProbe_Probe_ReturnsNil(t *testing.T) {
	err := health_check.NewLocalProbe().Probe(context.Background())
	if err != nil {
		t.Fatalf("Probe() error = %v, want nil", err)
	}
}
