package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/coretypes"
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
