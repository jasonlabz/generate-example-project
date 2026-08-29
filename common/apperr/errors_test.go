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
