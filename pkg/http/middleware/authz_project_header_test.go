package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/coretypes"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/stretchr/testify/require"
)

func TestProjectHeaderAccessIfPresentPreservesOrganizationWideRequests(t *testing.T) {
	middleware := &AuthZ{}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})

	handler := middleware.ProjectHeaderAccessIfPresent(next, coretypes.VerbRead)
	req := httptest.NewRequest(http.MethodPost, "/api/v5/query_range", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.True(t, called)
	require.Equal(t, http.StatusOK, recorder.Code)
}


func TestCheckResourcesForUnscopedRequestsUsesProjectScopeAuthorization(t *testing.T) {
	middleware := &AuthZ{}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})
	handler := middleware.CheckResourcesForUnscopedRequests(next, "viewer")
	req := httptest.NewRequest(http.MethodPost, "/api/v5/query_range", nil)
	req = req.WithContext(ctxtypes.NewContextWithTenantScope(req.Context(), ctxtypes.TenantScope{
		OrgID: "11111111-1111-4111-8111-111111111111",
		ProjectID: "22222222-2222-4222-8222-222222222222",
	}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.True(t, called, "a previously authorized project-scoped request must reach the query handler")
	require.Equal(t, http.StatusOK, recorder.Code)
}
