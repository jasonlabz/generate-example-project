package wire

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func TestModuleMountV1_UsesHumaGroup(t *testing.T) {
	_, api := humatest.New(t)
	group := huma.NewGroup(api, "/api/v1")

	Module{
		RegisterV1: func(routeAPI huma.API) {
			huma.Register(routeAPI, huma.Operation{
				OperationID: "probe",
				Method:      http.MethodGet,
				Path:        "/probe",
			}, func(context.Context, *struct{}) (*struct{}, error) {
				return nil, nil
			})
		},
	}.MountV1(group)

	if response := api.Get("/api/v1/probe"); response.Result().StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Result().StatusCode, http.StatusNoContent)
	}
	if api.OpenAPI().Paths["/api/v1/probe"] == nil {
		t.Fatal("Huma group did not add the operation to OpenAPI")
	}
}
