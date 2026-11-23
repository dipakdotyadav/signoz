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

func TestTenantScopeContextDistinguishesMissingAndIncompleteScope(t *testing.T) {
	_, ok := TenantScopeFromContext(context.Background())
	require.False(t, ok)

	ctx := NewContextWithTenantScope(context.Background(), TenantScope{OrgID: "org-1"})
	scope, ok := TenantScopeFromContext(ctx)
	require.True(t, ok, "a present but incomplete scope must remain visible to fail-closed consumers")
	require.False(t, scope.Valid())
}
