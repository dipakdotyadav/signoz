package ctxtypes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTenantScopeContextRoundTrip(t *testing.T) {
	ctx := NewContextWithTenantScope(context.Background(), TenantScope{
		OrgID: "org-1", ProjectID: "project-1",
	})

	scope, ok := TenantScopeFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, TenantScope{OrgID: "org-1", ProjectID: "project-1"}, scope)
}

func TestTenantScopeContextRejectsMissingOrIncompleteScope(t *testing.T) {
	_, ok := TenantScopeFromContext(context.Background())
	require.False(t, ok)

	ctx := NewContextWithTenantScope(context.Background(), TenantScope{OrgID: "org-1"})
	_, ok = TenantScopeFromContext(ctx)
	require.False(t, ok)
}
