package clickhouseReader

import (
	"context"
	"testing"
	"strings"

	"github.com/SigNoz/signoz/pkg/query-service/model/v3"
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


func TestBuildLogTagAttributeKeysQuery(t *testing.T) {
	t.Run("organization-wide query binds user filters", func(t *testing.T) {
		query, args := buildLogTagAttributeKeysQuery(
			"signoz_logs", "distributed_tag_attributes_v2",
			false, "", nil, "resource", "service", 25,
		)
		require.Contains(t, query, "signoz_logs.distributed_tag_attributes_v2")
		require.Contains(t, query, "tag_type=$1")
		require.Contains(t, query, "tag_key ILIKE $2")
		require.Contains(t, query, "limit $3")
		require.Equal(t, []any{"resource", "%service%", 25}, args)
		require.NotContains(t, query, "resource")
		require.NotContains(t, query, "service")
	})

	t.Run("project query binds tenant and user filters in order", func(t *testing.T) {
		scopeArgs := []any{
			"11111111-1111-4111-8111-111111111111",
			"22222222-2222-4222-8222-222222222222",
		}
		query, args := buildLogTagAttributeKeysQuery(
			"signoz_logs", "distributed_tag_attributes_v3",
			true, "org_id=$1 AND project_id=$2", scopeArgs, "resource", "service", 25,
		)
		require.Contains(t, query, "signoz_logs.distributed_tag_attributes_v3")
		require.Contains(t, query, "org_id=$1 AND project_id=$2")
		require.Contains(t, query, "tag_type=$3")
		require.Contains(t, query, "tag_key ILIKE $4")
		require.Contains(t, query, "limit $5")
		require.Equal(t, []any{
			scopeArgs[0], scopeArgs[1], "resource", "%service%", 25,
		}, args)
		require.NotContains(t, query, "11111111")
		require.NotContains(t, query, "22222222")
		require.NotContains(t, query, "resource")
		require.NotContains(t, query, "service")
	})

	t.Run("search text cannot alter query structure", func(t *testing.T) {
		query, args := buildLogTagAttributeKeysQuery(
			"signoz_logs", "distributed_tag_attributes_v3",
			true, "org_id=$1 AND project_id=$2",
			[]any{"org", "project"}, "tag' OR 1=1 --", "%' OR 1=1 --", 10,
		)
		require.NotContains(t, query, "1=1")
		require.Equal(t, "tag' OR 1=1 --", args[2])
		require.Equal(t, "%%' OR 1=1 --%", args[3])
	})
}

func TestBuildLogAttributeValuesQueryScopesEveryUnionArm(t *testing.T) {
	attributes := []v3.AttributeKey{
		{Key: "service.name", DataType: v3.AttributeKeyDataTypeString},
		{Key: "http.route", DataType: v3.AttributeKeyDataTypeString},
	}

	t.Run("project scope is repeated for every attribute lookup", func(t *testing.T) {
		orgID := "11111111-1111-4111-8111-111111111111"
		projectID := "22222222-2222-4222-8222-222222222222"
		filter := "org_id=$1 AND project_id=$2"
		query, args := buildLogAttributeValuesQuery(
			"signoz_logs", "distributed_tag_attributes_v3",
			true, filter, []any{orgID, projectID}, attributes, 5,
		)

		require.Equal(t, 2, strings.Count(query, filter))
		require.Contains(t, query, "tag_key = $3")
		require.Contains(t, query, "tag_key = $4")
		require.Contains(t, query, "distributed_tag_attributes_v3")
		require.Equal(t, []any{orgID, projectID, "service.name", "http.route"}, args)
		require.NotContains(t, query, orgID)
		require.NotContains(t, query, projectID)
	})

	t.Run("legacy query remains unscoped", func(t *testing.T) {
		query, args := buildLogAttributeValuesQuery(
			"signoz_logs", "distributed_tag_attributes_v2",
			false, "", nil, attributes, 5,
		)

		require.NotContains(t, query, "org_id")
		require.NotContains(t, query, "project_id")
		require.Contains(t, query, "tag_key = $1")
		require.Contains(t, query, "tag_key = $2")
		require.Equal(t, []any{"service.name", "http.route"}, args)
	})
}
