package querybuilder_test

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/querybuilder"
	"github.com/SigNoz/signoz/pkg/telemetryschema/logstelemetryschema"
	"github.com/SigNoz/signoz/pkg/telemetryschema/metricstelemetryschema"
	"github.com/SigNoz/signoz/pkg/telemetryschema/tracestelemetryschema"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/huandu/go-sqlbuilder"
	"github.com/stretchr/testify/require"
)

func TestProjectTenantPredicatesCompileForEverySignalStorage(t *testing.T) {
	const orgID = "11111111-1111-4111-8111-111111111111"
	const projectID = "22222222-2222-4222-8222-222222222222"
	expression := "resource.signoz.org.id = '" + orgID + "' AND resource.signoz.project.id = '" + projectID + "'"

	tests := []struct {
		name    string
		signal  telemetrytypes.Signal
		storage qbtypes.Storage
		check   []string
	}{
		{
			name: "traces",
			signal: telemetrytypes.SignalTraces,
			storage: tracestelemetryschema.NewStorage(nil),
			check: []string{"resource.`signoz.org.id`", "resource.`signoz.project.id`", "resources_string"},
		},
		{
			name: "logs",
			signal: telemetrytypes.SignalLogs,
			storage: logstelemetryschema.NewStorage(),
			check: []string{"resource.`signoz.org.id`", "resource.`signoz.project.id`", "resources_string"},
		},
		{
			name: "metrics",
			signal: telemetrytypes.SignalMetrics,
			storage: metricstelemetryschema.NewStorage(),
			check: []string{"JSONExtractString(labels, 'signoz.org.id')", "JSONExtractString(labels, 'signoz.project.id')"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := sqlbuilder.NewSelectBuilder()
			prepared, err := querybuilder.PrepareWhereClause(expression, querybuilder.FilterExprVisitorOpts{
				Context: context.Background(),
				Query: qbtypes.QueryInfo{Signal: tt.signal},
				Storage: tt.storage,
				Builder: builder,
			})
			require.NoError(t, err)
			require.NotNil(t, prepared.WhereClause)
			// PrepareWhereClause returns a sqlbuilder expression containing internal
			// placeholders. Compile it before asserting on the generated SQL.
			sql, args := builder.Args.CompileWithFlavor(prepared.Expr, builder.Flavor())
			for _, fragment := range tt.check {
				require.Contains(t, sql, fragment)
			}
			require.Contains(t, sql, " AND ", "both tenant predicates must be joined with AND: %s", sql)
			// Resource reads can reference the same tenant key through both the
			// JSON and map-backed storage expressions. The builder therefore
			// emits each tenant value twice; no other values should be bound.
			require.ElementsMatch(t, []any{orgID, orgID, projectID, projectID}, args)
			require.Len(t, args, 4)
		})
	}
}
