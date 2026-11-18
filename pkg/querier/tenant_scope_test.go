package querier

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/require"
)

func TestApplyTenantScopeAddsMandatoryPredicates(t *testing.T) {
	orgID := valuer.MustNewUUID("11111111-1111-4111-8111-111111111111")
	projectID := valuer.MustNewUUID("22222222-2222-4222-8222-222222222222")
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: orgID.String(), ProjectID: projectID.String(),
	})
	req := &qbtypes.QueryRangeRequest{
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "A",
				Filter: &qbtypes.Filter{Expression: "severity_text = 'ERROR'"},
			},
		}}},
	}

	require.NoError(t, applyTenantScope(ctx, orgID, req))
	spec := req.CompositeQuery.Queries[0].Spec.(qbtypes.QueryBuilderQuery[qbtypes.LogAggregation])
	require.Equal(t,
		"(severity_text = 'ERROR') AND (resource.signoz.org.id = '11111111-1111-4111-8111-111111111111' AND resource.signoz.project.id = '22222222-2222-4222-8222-222222222222')",
		spec.Filter.Expression,
	)
}

func TestApplyTenantScopeRejectsUnscopedQueryLanguages(t *testing.T) {
	orgID := valuer.MustNewUUID("11111111-1111-4111-8111-111111111111")
	projectID := valuer.MustNewUUID("22222222-2222-4222-8222-222222222222")
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: orgID.String(), ProjectID: projectID.String(),
	})
	req := &qbtypes.QueryRangeRequest{
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeClickHouseSQL,
			Spec: qbtypes.ClickHouseQuery{Name: "raw", Query: "SELECT 1"},
		}}},
	}

	require.Error(t, applyTenantScope(ctx, orgID, req))
}

func TestApplyTenantScopeRejectsOrganizationMismatch(t *testing.T) {
	orgID := valuer.MustNewUUID("11111111-1111-4111-8111-111111111111")
	otherOrgID := valuer.MustNewUUID("33333333-3333-4333-8333-333333333333")
	projectID := valuer.MustNewUUID("22222222-2222-4222-8222-222222222222")
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: otherOrgID.String(), ProjectID: projectID.String(),
	})
	req := &qbtypes.QueryRangeRequest{}

	require.Error(t, applyTenantScope(ctx, orgID, req))
}
