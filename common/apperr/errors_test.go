package apperr

import (
	"net/http"
	"testing"
)

func TestCatalogUsesTemplateNineDigitCodes(t *testing.T) {
	specs := []Spec{
		InvalidRequest,
		Unauthenticated,
		Forbidden,
		NotFound,
		Conflict,
		BusinessRule,
		DependencyUnavailable,
		Internal,
		RateLimited,
	}
	seen := make(map[int]struct{}, len(specs))
	for _, spec := range specs {
		code := spec.Code()
		if code < 100000000 || code > 999999999 {
			t.Fatalf("code %d is not nine digits", code)
		}
		if code/10000000 != SystemCode {
			t.Fatalf("code %d system prefix = %d, want %d", code, code/10000000, SystemCode)
		}
		if _, duplicate := seen[code]; duplicate {
			t.Fatalf("code %d is duplicated", code)
		}
		seen[code] = struct{}{}
	}
}

func TestForHTTPStatusReturnsPublicContract(t *testing.T) {
	if got := ForHTTPStatus(http.StatusUnauthorized); got != Unauthenticated {
		t.Fatalf("unauthorized spec = %#v, want %#v", got, Unauthenticated)
	}
	if got := ForHTTPStatus(http.StatusInternalServerError); got != Internal {
		t.Fatalf("internal spec = %#v, want %#v", got, Internal)
	}
}

// TestForHTTPStatus_MapsFrameworkValidationToInvalidRequest 固定框架校验错误的归因：
// huma 用 422 表示参数校验失败，必须归 InvalidRequest 且保留 422 的 HTTP 语义，
// 否则会和业务代码主动表达的 BusinessRule 撞码。
func TestForHTTPStatus_MapsFrameworkValidationToInvalidRequest(t *testing.T) {
	spec := ForHTTPStatus(http.StatusUnprocessableEntity)

	if spec.Code() != InvalidRequest.Code() {
		t.Fatalf("code = %d, want %d (InvalidRequest)", spec.Code(), InvalidRequest.Code())
	}
	if spec.HTTPStatus != http.StatusUnprocessableEntity {
		t.Fatalf("http status = %d, want %d", spec.HTTPStatus, http.StatusUnprocessableEntity)
	}
}

// TestBusinessRuleKeepsItsOwnCode 保证上一条归因不会吃掉业务规则错误码：
// 业务代码主动表达的 422 仍使用 BusinessRule。
func TestBusinessRuleKeepsItsOwnCode(t *testing.T) {
	if BusinessRule.HTTPStatus != http.StatusUnprocessableEntity {
		t.Fatalf("http status = %d, want %d", BusinessRule.HTTPStatus, http.StatusUnprocessableEntity)
	}
	if BusinessRule.Code() == InvalidRequest.Code() {
		t.Fatal("BusinessRule must not share the code of InvalidRequest")
	}
	if ForHTTPStatus(http.StatusUnprocessableEntity).Code() == BusinessRule.Code() {
		t.Fatal("framework 422 must not be attributed to BusinessRule")
	}
}
