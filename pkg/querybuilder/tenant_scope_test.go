package querybuilder

import (
	"context"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/telemetryschema/logstelemetryschema"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/huandu/go-sqlbuilder"
	"github.com/stretchr/testify/require"
)

const (
	testTenantOrgID = "11111111-1111-4111-8111-111111111111"
	testTenantProjectID = "22222222-2222-4222-8222-222222222222"
)

func TestTenantScopedFilterAppendsAuthenticatedPredicates(t *testing.T) {
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: testTenantOrgID, ProjectID: testTenantProjectID,
	})
	got := tenantScopedFilter(ctx, "severity_text = 'error' OR severity_text = 'fatal'")
	require.Contains(t, got, "(severity_text = 'error' OR severity_text = 'fatal')")
	require.Contains(t, got, "resource.signoz.org.id")
	require.Contains(t, got, "resource.signoz.project.id")
	require.Contains(t, got, testTenantOrgID)
	require.Contains(t, got, testTenantProjectID)
	require.Contains(t, got, ") AND (")
}

func TestTenantScopedFilterAddsPredicateWhenClientFilterIsEmpty(t *testing.T) {
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: testTenantOrgID, ProjectID: testTenantProjectID,
	})
	got := tenantScopedFilter(ctx, "  ")
	require.NotEmpty(t, got)
	require.Contains(t, got, "resource.signoz.org.id")
	require.Contains(t, got, "resource.signoz.project.id")
}

func TestTenantScopedFilterLeavesUnscopedQueryUnchanged(t *testing.T) {
	query := "severity_text = 'error'"
	require.Equal(t, query, tenantScopedFilter(context.Background(), query))
}

func TestPrepareWhereClauseEnforcesAuthenticatedTenantScope(t *testing.T) {
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: testTenantOrgID, ProjectID: testTenantProjectID,
	})
	builder := sqlbuilder.NewSelectBuilder()
	prepared, err := PrepareWhereClause("severity_text = 'error' OR severity_text = 'fatal'", FilterExprVisitorOpts{
		Context: ctx,
		Query: qbtypes.QueryInfo{Signal: telemetrytypes.SignalLogs},
		Storage: logstelemetryschema.NewStorage(),
		Builder: builder,
	})
	require.NoError(t, err)
	require.NotNil(t, prepared.WhereClause)
	require.True(t, strings.Contains(prepared.Expr, " AND "), "tenant predicates must be ANDed with the complete user expression: %s", prepared.Expr)
	require.Contains(t, prepared.Expr, "resource.")
	require.Contains(t, prepared.Expr, "signoz.org.id")
	require.Contains(t, prepared.Expr, "signoz.project.id")
	require.Contains(t, builder.Args, testTenantOrgID)
	require.Contains(t, builder.Args, testTenantProjectID)
}
