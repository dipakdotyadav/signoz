package clickhouseReader

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/stretchr/testify/require"
)

func TestTenantTagMetadataScope(t *testing.T) {
	t.Run("organization-wide request remains unscoped", func(t *testing.T) {
		filter, args, scoped, err := tenantTagMetadataScope(context.Background(), 1)
		require.NoError(t, err)
		require.False(t, scoped)
		require.Empty(t, filter)
		require.Empty(t, args)
	})

	t.Run("authenticated project request binds both identifiers", func(t *testing.T) {
		ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
			OrgID: "11111111-1111-4111-8111-111111111111",
			ProjectID: "22222222-2222-4222-8222-222222222222",
		})
		filter, args, scoped, err := tenantTagMetadataScope(ctx, 3)
		require.NoError(t, err)
		require.True(t, scoped)
		require.Equal(t, "org_id=$3 AND project_id=$4", filter)
		require.Equal(t, []any{
			"11111111-1111-4111-8111-111111111111",
			"22222222-2222-4222-8222-222222222222",
		}, args)
	})


	t.Run("incomplete authenticated scope fails closed", func(t *testing.T) {
		ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
			OrgID: "11111111-1111-4111-8111-111111111111",
		})
		_, _, scoped, err := tenantTagMetadataScope(ctx, 1)
		require.True(t, scoped)
		require.Error(t, err)
	})

	t.Run("invalid authenticated identity fails closed", func(t *testing.T) {
		ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
			OrgID: "not-an-org",
			ProjectID: "22222222-2222-4222-8222-222222222222",
		})
		_, _, scoped, err := tenantTagMetadataScope(ctx, 1)
		require.True(t, scoped)
		require.Error(t, err)
	})
}
