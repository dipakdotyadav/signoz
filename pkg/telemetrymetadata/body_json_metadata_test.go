package telemetrymetadata

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/SigNoz/signoz-otel-collector/constants"
	"github.com/SigNoz/signoz/pkg/querybuilder"
	"github.com/SigNoz/signoz/pkg/telemetryschema/logstelemetryschema"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/promotetypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/stretchr/testify/require"
)

func TestBuildListLogsJSONIndexesQuery(t *testing.T) {
	testCases := []struct {
		name         string
		cluster      string
		filters      []string
		expectedSQL  string
		expectedArgs []any
	}{
		{
			name:    "No filters",
			cluster: "test-cluster",
			filters: nil,
			expectedSQL: "SELECT name, type_full, expr, granularity FROM clusterAllReplicas('test-cluster', system.data_skipping_indices) " +
				"WHERE database = ? AND table = ? AND (LOWER(expr) LIKE LOWER(?) OR LOWER(expr) LIKE LOWER(?))",
			expectedArgs: []any{
				logstelemetryschema.DBName,
				logstelemetryschema.LogsV2LocalTableName,
				fmt.Sprintf("%%%s%%", querybuilder.FormatValueForContains(constants.BodyV2ColumnPrefix)),
				fmt.Sprintf("%%%s%%", querybuilder.FormatValueForContains(constants.BodyPromotedColumnPrefix)),
			},
		},
		{
			name:    "With filters",
			cluster: "test-cluster",
			filters: []string{"foo", "bar"},
			expectedSQL: "SELECT name, type_full, expr, granularity FROM clusterAllReplicas('test-cluster', system.data_skipping_indices) " +
				"WHERE database = ? AND table = ? AND (LOWER(expr) LIKE LOWER(?) OR LOWER(expr) LIKE LOWER(?)) AND (LOWER(replaceAll(expr, '`', '')) LIKE LOWER(?) OR LOWER(replaceAll(expr, '`', '')) LIKE LOWER(?))",
			expectedArgs: []any{
				logstelemetryschema.DBName,
				logstelemetryschema.LogsV2LocalTableName,
				fmt.Sprintf("%%%s%%", querybuilder.FormatValueForContains(constants.BodyV2ColumnPrefix)),
				fmt.Sprintf("%%%s%%", querybuilder.FormatValueForContains(constants.BodyPromotedColumnPrefix)),
				fmt.Sprintf("%%%s%%", querybuilder.FormatValueForContains("foo")),
				fmt.Sprintf("%%%s%%", querybuilder.FormatValueForContains("bar")),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query, args := buildListJSONIndexesQuery(tc.cluster, logsBodyIndexLookup, tc.filters...)

			require.Equal(t, tc.expectedSQL, query)
			require.Equal(t, tc.expectedArgs, args)
		})
	}
}

func TestUnfoldJSONSubColumnIndexExpr(t *testing.T) {
	testCases := []struct {
		name       string
		expr       string
		wantColumn string
		wantType   string
		wantErr    bool
	}{
		{
			name:       "LogsFoldedLower",
			expr:       "lower(assumeNotNull(dynamicElement(body_v2.user.name, 'String')))",
			wantColumn: "body_v2.user.name",
			wantType:   "String",
		},
		{
			name:       "LogsAssumeNotNullOnly",
			expr:       "assumeNotNull(dynamicElement(body_v2.request.duration, 'Float64'))",
			wantColumn: "body_v2.request.duration",
			wantType:   "Float64",
		},
		{
			name:       "TracesCastDottedKey",
			expr:       "CAST(attributes.`http.method`, 'String')",
			wantColumn: "attributes.`http.method`",
			wantType:   "String",
		},
		{
			name:       "TracesCastPlainKey",
			expr:       "CAST(attributes_promoted.status, 'Int64')",
			wantColumn: "attributes_promoted.status",
			wantType:   "Int64",
		},
		{
			name:       "TracesNumber",
			expr:       "toFloat64OrNull(CAST(attributes.`http.status_code`, 'String'))",
			wantColumn: "attributes.`http.status_code`",
			wantType:   "Float64",
		},
		{
			name:       "TracesNumber_BacktickedParentheses",
			expr:       "toFloat64OrNull(CAST(attributes.`latency(ms)`, 'String'))",
			wantColumn: "attributes.`latency(ms)`",
			wantType:   "Float64",
		},
		{
			name:    "TracesNumber_NotAStringCast",
			expr:    "toFloat64OrNull(CAST(attributes.a, 'Int64'))",
			wantErr: true,
		},
		{
			name:    "UnrecognizedForm",
			expr:    "toString(attributes.status)",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			columnExpr, columnType, err := unfoldJSONSubColumnIndexExpr(tc.expr)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantColumn, columnExpr)
			require.Equal(t, tc.wantType, columnType)
		})
	}
}

// ClickHouse stores a generated col.`path`::Type trace index expression as
// CAST(col.`path`, 'Type'), dropping quoting an identifier does not need; the
// listing must parse that stored form back into the original path.
func TestUnfoldJSONSubColumnIndexExprTracesRoundTrip(t *testing.T) {
	testCases := []struct {
		name         string
		column       string
		path         string
		jsonDataType string
		wantExpr     string
		storedExpr   string
	}{
		{
			name:         "DottedKey_BackticksPreservedInStorage",
			column:       "attributes",
			path:         "http.method",
			jsonDataType: "String",
			wantExpr:     "attributes.`http.method`::String",
			storedExpr:   "CAST(attributes.`http.method`, 'String')",
		},
		{
			name:         "PlainKey_UnquotedInStorage",
			column:       "attributes_promoted",
			path:         "status",
			jsonDataType: "String",
			wantExpr:     "attributes_promoted.`status`::String",
			storedExpr:   "CAST(attributes_promoted.status, 'String')",
		},
		{
			name:         "DottedNumber_BackticksPreservedInStorage",
			column:       "attributes",
			path:         "http.status_code",
			jsonDataType: "Float64",
			wantExpr:     "toFloat64OrNull(attributes.`http.status_code`::String)",
			storedExpr:   "toFloat64OrNull(CAST(attributes.`http.status_code`, 'String'))",
		},
		{
			name:         "PlainNumber_UnquotedInStorage",
			column:       "attributes_promoted",
			path:         "latency",
			jsonDataType: "Float64",
			wantExpr:     "toFloat64OrNull(attributes_promoted.`latency`::String)",
			storedExpr:   "toFloat64OrNull(CAST(attributes_promoted.latency, 'String'))",
		},
	}

	target := promotetypes.NewTracesAttributesTarget()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantExpr, target.IndexExpression(tc.column, tc.path, tc.jsonDataType))

			columnExpr, columnType, err := unfoldJSONSubColumnIndexExpr(tc.storedExpr)
			require.NoError(t, err)
			require.Equal(t, tc.column+"."+tc.path, strings.ReplaceAll(columnExpr, "`", ""))
			require.Equal(t, tc.jsonDataType, columnType)
		})
	}
}


func TestTenantPromotionScope(t *testing.T) {
	_, scoped, err := tenantPromotionScope(context.Background())
	require.NoError(t, err)
	require.False(t, scoped, "organization-wide legacy calls have no tenant scope")

	validCtx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: "11111111-1111-4111-8111-111111111111",
		ProjectID: "22222222-2222-4222-8222-222222222222",
	})
	scope, scoped, err := tenantPromotionScope(validCtx)
	require.NoError(t, err)
	require.True(t, scoped)
	require.Equal(t, "22222222-2222-4222-8222-222222222222", scope.ProjectID)

	invalidCtx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{OrgID: "org-only"})
	_, scoped, err = tenantPromotionScope(invalidCtx)
	require.True(t, scoped)
	require.Error(t, err, "an incomplete scope must fail closed")
}


func TestBuildGetPromotedPathsQueryTenantIsolation(t *testing.T) {
	entry := promotetypes.NewTracesAttributesTarget().Entry
	scopeA := &ctxtypes.TenantScope{
		OrgID: "11111111-1111-4111-8111-111111111111",
		ProjectID: "22222222-2222-4222-8222-222222222222",
	}
	scopeB := &ctxtypes.TenantScope{
		OrgID: "11111111-1111-4111-8111-111111111111",
		ProjectID: "33333333-3333-4333-8333-333333333333",
	}

	queryA, argsA := buildGetPromotedPathsQuery(entry, scopeA, "http.method", "http.status"+telemetrytypes.ArraySep+"0")
	_, argsB := buildGetPromotedPathsQuery(entry, scopeB, "http.method", "http.status"+telemetrytypes.ArraySep+"0")
	legacyQuery, legacyArgs := buildGetPromotedPathsQuery(entry, nil, "http.method")

	require.Contains(t, queryA, TenantPromotedPathsTableName)
	require.Contains(t, queryA, "org_id")
	require.Contains(t, queryA, "project_id")
	require.NotContains(t, queryA, PromotedPathsTableName+" WHERE signal")
	require.NotEqual(t, argsA, argsB, "project A and project B must produce different tenant query arguments")
	require.Contains(t, argsA, scopeA.OrgID)
	require.Contains(t, argsA, scopeA.ProjectID)
	require.NotContains(t, argsA, scopeB.ProjectID)
	require.Contains(t, argsA, "http.status")
	require.NotContains(t, argsA, "http.status"+telemetrytypes.ArraySep+"0")

	require.Contains(t, legacyQuery, PromotedPathsTableName)
	require.NotContains(t, legacyQuery, TenantPromotedPathsTableName)
	require.NotContains(t, legacyQuery, "project_id")
	require.NotContains(t, legacyArgs, scopeA.ProjectID)
}
