package querier

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/require"
)

const (
	testOrgID     = "11111111-1111-4111-8111-111111111111"
	testProjectID = "22222222-2222-4222-8222-222222222222"
	testOtherOrg  = "33333333-3333-4333-8333-333333333333"
)

func tenantTestContext() (context.Context, valuer.UUID) {
	orgID := valuer.MustNewUUID(testOrgID)
	projectID := valuer.MustNewUUID(testProjectID)
	return ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: orgID.String(), ProjectID: projectID.String(),
	}), orgID
}

func requireTenantPredicates(t *testing.T, filter *qbtypes.Filter) {
	t.Helper()
	require.NotNil(t, filter)
	require.Contains(t, filter.Expression, "resource.signoz.org.id = '"+testOrgID+"'")
	require.Contains(t, filter.Expression, "resource.signoz.project.id = '"+testProjectID+"'")
}

func TestApplyTenantScopeAddsMandatoryPredicates(t *testing.T) {
	ctx, orgID := tenantTestContext()
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
		"(severity_text = 'ERROR') AND (resource.signoz.org.id = '"+testOrgID+"' AND resource.signoz.project.id = '"+testProjectID+"')",
		spec.Filter.Expression,
	)
}

func TestApplyTenantScopeScopesAllBuilderSignals(t *testing.T) {
	ctx, orgID := tenantTestContext()
	req := &qbtypes.QueryRangeRequest{
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
			{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{Name: "traces", Signal: telemetrytypes.SignalTraces},
			},
			{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{Name: "logs", Signal: telemetrytypes.SignalLogs},
			},
			{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{Name: "metrics", Signal: telemetrytypes.SignalMetrics},
			},
			{
				Type: qbtypes.QueryTypeBuilderAI,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{Name: "ai", Signal: telemetrytypes.SignalTraces},
			},
		}},
	}

	require.NoError(t, applyTenantScope(ctx, orgID, req))

	trace := req.CompositeQuery.Queries[0].Spec.(qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation])
	logs := req.CompositeQuery.Queries[1].Spec.(qbtypes.QueryBuilderQuery[qbtypes.LogAggregation])
	metrics := req.CompositeQuery.Queries[2].Spec.(qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation])
	ai := req.CompositeQuery.Queries[3].Spec.(qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation])

	requireTenantPredicates(t, trace.Filter)
	requireTenantPredicates(t, logs.Filter)
	requireTenantPredicates(t, metrics.Filter)
	requireTenantPredicates(t, ai.Filter)
}

func TestApplyTenantScopeRejectsProjectScopedAuditQueries(t *testing.T) {
	ctx, orgID := tenantTestContext()
	req := &qbtypes.QueryRangeRequest{
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "audit", Signal: telemetrytypes.SignalLogs, Source: telemetrytypes.SourceAudit,
			},
		}}},
	}
	require.ErrorContains(t, applyTenantScope(ctx, orgID, req), "project-scoped audit queries are not yet supported")
}

func TestApplyTenantScopeRejectsUnscopedQueryLanguages(t *testing.T) {
	ctx, orgID := tenantTestContext()
	tests := []struct {
		name string
		env  qbtypes.QueryEnvelope
	}{
		{
			name: "raw clickhouse sql",
			env: qbtypes.QueryEnvelope{
				Type: qbtypes.QueryTypeClickHouseSQL,
				Spec: qbtypes.ClickHouseQuery{Name: "raw", Query: "SELECT 1"},
			},
		},
		{
			name: "promql",
			env: qbtypes.QueryEnvelope{
				Type: qbtypes.QueryTypePromQL,
				Spec: qbtypes.PromQuery{Name: "prom", Query: "up"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &qbtypes.QueryRangeRequest{
				CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{tt.env}},
			}
			require.Error(t, applyTenantScope(ctx, orgID, req))
		})
	}
}

func TestApplyTenantScopeRejectsOrganizationMismatch(t *testing.T) {
	orgID := valuer.MustNewUUID(testOrgID)
	otherOrgID := valuer.MustNewUUID(testOtherOrg)
	projectID := valuer.MustNewUUID(testProjectID)
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: otherOrgID.String(), ProjectID: projectID.String(),
	})

	require.Error(t, applyTenantScope(ctx, orgID, &qbtypes.QueryRangeRequest{}))
}

func TestApplyTenantScopeRejectsInvalidProjectID(t *testing.T) {
	orgID := valuer.MustNewUUID(testOrgID)
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: orgID.String(), ProjectID: "not-a-uuid",
	})

	require.Error(t, applyTenantScope(ctx, orgID, &qbtypes.QueryRangeRequest{}))
}

func TestApplyTenantScopeLeavesOrganizationWideRequestsUnchanged(t *testing.T) {
	orgID := valuer.MustNewUUID(testOrgID)
	req := &qbtypes.QueryRangeRequest{
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "A", Filter: &qbtypes.Filter{Expression: "severity_text = 'ERROR'"},
			},
		}}},
	}

	require.NoError(t, applyTenantScope(context.Background(), orgID, req))
	spec := req.CompositeQuery.Queries[0].Spec.(qbtypes.QueryBuilderQuery[qbtypes.LogAggregation])
	require.Equal(t, "severity_text = 'ERROR'", spec.Filter.Expression)
}

func TestApplyTenantScopeRejectsNilRequest(t *testing.T) {
	ctx, orgID := tenantTestContext()
	require.Error(t, applyTenantScope(ctx, orgID, nil))
}

func TestApplyTenantScopeSupportsTraceOperatorWithScopedOperands(t *testing.T) {
	ctx, orgID := tenantTestContext()
	traceQuery := func(name string) qbtypes.QueryEnvelope {
		return qbtypes.QueryEnvelope{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Name: name, Signal: telemetrytypes.SignalTraces,
			},
		}
	}
	req := &qbtypes.QueryRangeRequest{
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
			{Type: qbtypes.QueryTypeTraceOperator, Spec: qbtypes.QueryBuilderTraceOperator{Name: "T", Expression: "A => B"}},
			traceQuery("A"),
			traceQuery("B"),
		}},
	}

	require.NoError(t, applyTenantScope(ctx, orgID, req))
	for _, env := range req.CompositeQuery.Queries[1:] {
		spec := env.Spec.(qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation])
		requireTenantPredicates(t, spec.Filter)
	}
}

func TestApplyTenantScopeRejectsTraceOperatorWithUnscopedOperand(t *testing.T) {
	ctx, orgID := tenantTestContext()
	req := &qbtypes.QueryRangeRequest{
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
			{Type: qbtypes.QueryTypeTraceOperator, Spec: qbtypes.QueryBuilderTraceOperator{Name: "T", Expression: "A => Missing"}},
			{Type: qbtypes.QueryTypeBuilder, Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{Name: "A", Signal: telemetrytypes.SignalTraces}},
		}},
	}

	require.Error(t, applyTenantScope(ctx, orgID, req))
}
