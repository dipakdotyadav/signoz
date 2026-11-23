package clickhouseReader

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestLogAttributeKeysAreIsolatedAcrossProjectsAndOrganizations exercises the
// same SQL builder used by GetLogAttributeKeys against a real ClickHouse server.
// Set SIGNOZ_CLICKHOUSE_TEST_ADDR and SIGNOZ_CLICKHOUSE_TEST_DATABASE to enable it.
func TestLogAttributeKeysAreIsolatedAcrossProjectsAndOrganizations(t *testing.T) {
	addr := os.Getenv("SIGNOZ_CLICKHOUSE_TEST_ADDR")
	database := os.Getenv("SIGNOZ_CLICKHOUSE_TEST_DATABASE")
	if addr == "" || database == "" {
		t.Skip("set SIGNOZ_CLICKHOUSE_TEST_ADDR and SIGNOZ_CLICKHOUSE_TEST_DATABASE to run ClickHouse integration tests")
	}

	ctx := context.Background()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: os.Getenv("SIGNOZ_CLICKHOUSE_TEST_USERNAME"),
			Password: os.Getenv("SIGNOZ_CLICKHOUSE_TEST_PASSWORD"),
		},
	})
	require.NoError(t, err)
	defer conn.Close()

	table := "tenant_metadata_it_" + uuid.NewString()
	qualifiedTable := fmt.Sprintf("%s.%s", database, table)
	require.NoError(t, conn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE %s (
			org_id String,
			project_id String,
			tag_key String,
			tag_type String,
			tag_data_type String
		) ENGINE = Memory
	`, qualifiedTable)))
	defer func() {
		_ = conn.Exec(context.Background(), "DROP TABLE IF EXISTS "+qualifiedTable)
	}()

	orgA := "11111111-1111-4111-8111-111111111111"
	orgB := "44444444-4444-4444-8444-444444444444"
	projectA := "22222222-2222-4222-8222-222222222222"
	projectB := "33333333-3333-4333-8333-333333333333"
	projectC := "55555555-5555-4555-8555-555555555555"

	insert := fmt.Sprintf(`
		INSERT INTO %s (org_id, project_id, tag_key, tag_type, tag_data_type) VALUES
		('%s', '%s', 'service.name', 'resource', 'string'),
		('%s', '%s', 'http.route', 'tag', 'string'),
		('%s', '%s', 'db.system', 'resource', 'string')
	`, qualifiedTable, orgA, projectA, orgA, projectB, orgB, projectC)
	require.NoError(t, conn.Exec(ctx, insert))

	cases := []struct {
		name      string
		orgID     string
		projectID string
		want      []string
	}{
		{name: "project A", orgID: orgA, projectID: projectA, want: []string{"service.name"}},
		{name: "project B in same organization", orgID: orgA, projectID: projectB, want: []string{"http.route"}},
		{name: "project C in another organization", orgID: orgB, projectID: projectC, want: []string{"db.system"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requestCtx := ctxtypes.NewContextWithTenantScope(ctx, ctxtypes.TenantScope{
				OrgID: tc.orgID, ProjectID: tc.projectID,
			})
			scopeFilter, scopeArgs, scoped, scopeErr := tenantTagMetadataScope(requestCtx, 1)
			require.NoError(t, scopeErr)
			require.True(t, scoped)

			query, args := buildLogTagAttributeKeysQuery(
				database, table, scoped, scopeFilter, scopeArgs, "", "", 100,
			)
			rows, queryErr := conn.Query(requestCtx, query, args...)
			require.NoError(t, queryErr)
			defer rows.Close()

			got := make([]string, 0)
			for rows.Next() {
				var key, tagType, dataType string
				require.NoError(t, rows.Scan(&key, &tagType, &dataType))
				got = append(got, key)
			}
			require.NoError(t, rows.Err())
			sort.Strings(got)
			require.Equal(t, tc.want, got)
		})
	}
}
