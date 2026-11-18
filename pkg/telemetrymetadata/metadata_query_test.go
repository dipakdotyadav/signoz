package telemetrymetadata

import (
	"context"
	"regexp"
	"testing"

	"github.com/huandu/go-sqlbuilder"
	cmock "github.com/SigNoz/clickhouse-go-mock"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/flagger/flaggertest"
	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrystore/telemetrystoretest"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type regexMatcher struct {
}

func (m *regexMatcher) Match(expectedSQL, actualSQL string) error {
	re, err := regexp.Compile(expectedSQL)
	if err != nil {
		return err
	}
	if !re.MatchString(actualSQL) {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "expected query to contain %s, got %s", expectedSQL, actualSQL)
	}
	return nil
}

func TestGetFirstSeenFromMetricMetadata(t *testing.T) {
	mockTelemetryStore := telemetrystoretest.New(telemetrystore.Config{}, &regexMatcher{})
	mock := mockTelemetryStore.Mock()

	metadata := NewTelemetryMetaStore(
		instrumentationtest.New().ToProviderSettings(),
		mockTelemetryStore,
		flaggertest.New(t),
	)

	lookupKeys := []telemetrytypes.MetricMetadataLookupKey{
		{
			MetricName:     "metric1",
			AttributeName:  "attr1",
			AttributeValue: "val1",
		},
		{
			MetricName:     "metric2",
			AttributeName:  "attr2",
			AttributeValue: "val2",
		},
	}

	// ClickHouse tuple syntax is (x, y, z)
	// the structure should lead to:
	// SELECT ... WHERE (metric_name, attr_name, attr_string_value) IN ((?, ?, ?), (?, ?, ?)) ...

	expectedQuery := `SELECT metric_name, attr_name, attr_string_value, min\\(first_reported_unix_milli\\) AS first_seen FROM signoz_metrics.distributed_metadata WHERE \\(metric_name, attr_name, attr_string_value\\) IN \\(\\(\\?, \\?, \\?\\), \\(\\?, \\?, \\?\\)\\) GROUP BY metric_name, attr_name, attr_string_value ORDER BY first_seen`

	// Note: regexMatcher uses regexp.MatchString, so we escape parens and ?

	mock.ExpectQuery(expectedQuery).
		WithArgs("metric1", "attr1", "val1", "metric2", "attr2", "val2").
		WillReturnRows(cmock.NewRows([]cmock.ColumnType{
			{Name: "metric_name", Type: "String"},
			{Name: "attr_name", Type: "String"},
			{Name: "attr_string_value", Type: "String"},
			{Name: "first_seen", Type: "UInt64"},
		}, [][]any{
			{"metric1", "attr1", "val1", uint64(1000)},
			{"metric2", "attr2", "val2", uint64(2000)},
		}))

	result, err := metadata.GetFirstSeenFromMetricMetadata(context.Background(), lookupKeys)
	require.NoError(t, err)

	assert.Equal(t, int64(1000), result[lookupKeys[0]])
	assert.Equal(t, int64(2000), result[lookupKeys[1]])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}


	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/flagger/flaggertest"
	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrystore/telemetrystoretest"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type regexMatcher struct {
}

func (m *regexMatcher) Match(expectedSQL, actualSQL string) error {
	re, err := regexp.Compile(expectedSQL)
	if err != nil {
		return err
	}
	if !re.MatchString(actualSQL) {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "expected query to contain %s, got %s", expectedSQL, actualSQL)
	}
	return nil
}

func TestGetFirstSeenFromMetricMetadata(t *testing.T) {
	mockTelemetryStore := telemetrystoretest.New(telemetrystore.Config{}, &regexMatcher{})
	mock := mockTelemetryStore.Mock()

	metadata := NewTelemetryMetaStore(
		instrumentationtest.New().ToProviderSettings(),
		mockTelemetryStore,
		flaggertest.New(t),
	)

	lookupKeys := []telemetrytypes.MetricMetadataLookupKey{
		{
			MetricName:     "metric1",
			AttributeName:  "attr1",
			AttributeValue: "val1",
		},
		{
			MetricName:     "metric2",
			AttributeName:  "attr2",
			AttributeValue: "val2",
		},
	}

	// ClickHouse tuple syntax is (x, y, z)
	// the structure should lead to:
	// SELECT ... WHERE (metric_name, attr_name, attr_string_value) IN ((?, ?, ?), (?, ?, ?)) ...

	expectedQuery := `SELECT metric_name, attr_name, attr_string_value, min\(first_reported_unix_milli\) AS first_seen FROM signoz_metrics.distributed_metadata WHERE \(metric_name, attr_name, attr_string_value\) IN \(\(\?, \?, \?\), \(\?, \?, \?\)\) GROUP BY metric_name, attr_name, attr_string_value ORDER BY first_seen`

	// Note: regexMatcher uses regexp.MatchString, so we escape parens and ?

	mock.ExpectQuery(expectedQuery).
		WithArgs("metric1", "attr1", "val1", "metric2", "attr2", "val2").
		WillReturnRows(cmock.NewRows([]cmock.ColumnType{
			{Name: "metric_name", Type: "String"},
			{Name: "attr_name", Type: "String"},
			{Name: "attr_string_value", Type: "String"},
			{Name: "first_seen", Type: "UInt64"},
		}, [][]any{
			{"metric1", "attr1", "val1", uint64(1000)},
			{"metric2", "attr2", "val2", uint64(2000)},
		}))

	result, err := metadata.GetFirstSeenFromMetricMetadata(context.Background(), lookupKeys)
	require.NoError(t, err)

	assert.Equal(t, int64(1000), result[lookupKeys[0]])
	assert.Equal(t, int64(2000), result[lookupKeys[1]])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}


func TestAddMetricMetadataTenantScope(t *testing.T) {
	orgID := valuer.MustNewUUID("11111111-1111-4111-8111-111111111111")
	projectID := "22222222-2222-4222-8222-222222222222"
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: orgID.String(), ProjectID: projectID,
	})
	sb := sqlbuilder.Select("*").From("signoz_metrics.distributed_metadata")
	require.NoError(t, addMetricMetadataTenantScope(ctx, orgID, sb, "resource_attributes"))
	query, args := sb.BuildWithFlavor(sqlbuilder.ClickHouse)
	require.Contains(t, query, "resource_attributes['signoz.org.id']")
	require.Contains(t, query, "resource_attributes['signoz.project.id']")
	require.Contains(t, args, orgID.String())
	require.Contains(t, args, projectID)
}

func TestAddMetricMetadataTenantScopeRejectsOrganizationMismatch(t *testing.T) {
	orgID := valuer.MustNewUUID("11111111-1111-4111-8111-111111111111")
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: "99999999-9999-4999-8999-999999999999",
		ProjectID: "22222222-2222-4222-8222-222222222222",
	})
	sb := sqlbuilder.Select("*").From("signoz_metrics.distributed_metadata")
	require.Error(t, addMetricMetadataTenantScope(ctx, orgID, sb, "resource_attributes"))
}

func TestAddMetricMetadataTenantScopeUsesOrganizationBoundaryWithoutProject(t *testing.T) {
	orgID := valuer.MustNewUUID("11111111-1111-4111-8111-111111111111")
	sb := sqlbuilder.Select("*").From("signoz_metrics.distributed_metadata")
	require.NoError(t, addMetricMetadataTenantScope(context.Background(), orgID, sb, "resource_attributes"))
	query, args := sb.BuildWithFlavor(sqlbuilder.ClickHouse)
	require.Contains(t, query, "resource_attributes['signoz.org.id']")
	require.NotContains(t, query, "resource_attributes['signoz.project.id']")
	require.Contains(t, args, orgID.String())
}


func TestAddMetricMetadataProjectScope(t *testing.T) {
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: "11111111-1111-4111-8111-111111111111",
		ProjectID: "22222222-2222-4222-8222-222222222222",
	})
	sb := sqlbuilder.Select("*").From("signoz_metrics.distributed_metadata")
	require.NoError(t, addMetricMetadataProjectScope(ctx, sb, "resource_attributes"))
	query, args := sb.BuildWithFlavor(sqlbuilder.ClickHouse)
	require.Contains(t, query, "resource_attributes['signoz.org.id']")
	require.Contains(t, query, "resource_attributes['signoz.project.id']")
	require.Contains(t, args, "11111111-1111-4111-8111-111111111111")
	require.Contains(t, args, "22222222-2222-4222-8222-222222222222")
}
