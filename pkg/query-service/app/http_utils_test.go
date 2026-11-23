package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/stretchr/testify/require"
)

func TestWithCacheControlForTenantScopedRequests(t *testing.T) {
	handler := withCacheControl(time.Minute, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	t.Run("tenant scoped response is never cached", func(t *testing.T) {
		ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
			OrgID: "11111111-1111-4111-8111-111111111111",
			ProjectID: "22222222-2222-4222-8222-222222222222",
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	})

	t.Run("organization-wide response retains configured max age", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		require.Equal(t, "max-age=60", recorder.Header().Get("Cache-Control"))
	})
}
