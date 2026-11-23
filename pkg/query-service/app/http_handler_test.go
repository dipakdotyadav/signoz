package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/query-service/model"
	v3 "github.com/SigNoz/signoz/pkg/query-service/model/v3"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
)

func TestPrepareQuery(t *testing.T) {
	type testCase struct {
		name        string
		postData    *model.DashboardVars
		query       string
		expectedErr bool
		errMsg      string
	}

	testCases := []testCase{
		{
			name: "test empty query",
			postData: &model.DashboardVars{
				Query: "",
			},
			expectedErr: true,
			errMsg:      "query is required",
		},
		{
			name: "test query with no variables",
			postData: &model.DashboardVars{
				Query: "select * from table",
			},
			expectedErr: false,
			query:       "select * from table",
		},
		{
			name: "test query with variables",
			postData: &model.DashboardVars{
				Query: "select * from table where id = {{.id}}",
				Variables: map[string]interface{}{
					"id": "1",
				},
			},
			query:       "select * from table where id = '1'",
			expectedErr: false,
		},
		// While this does seem like it should throw error, it shouldn't because empty value is a valid value
		// for certain scenarios and should be treated as such.
		{
			name: "test query with variables and no value",
			postData: &model.DashboardVars{
				Query: "select * from table where id = {{.id}}",
				Variables: map[string]interface{}{
					"id": "",
				},
			},
			query:       "select * from table where id = ''",
			expectedErr: false,
		},
		{
			name: "query contains alter table",
			postData: &model.DashboardVars{
				Query: "ALTER TABLE signoz_table DELETE where true",
			},
			expectedErr: true,
			errMsg:      "operation alter table is not allowed",
		},
		{
			name: "query text produces template exec error",
			postData: &model.DashboardVars{
				Query: "SELECT durationNano from signoz_traces.signoz_index_v2 WHERE id = {{if .X}}1{{else}}2{{else}}3{{end}}",
			},
			expectedErr: true,
			errMsg:      "expected end; found {{else}}",
		},
		{
			name: "variables contain array",
			postData: &model.DashboardVars{
				Query: "SELECT operation FROM table WHERE serviceName IN {{.serviceName}}",
				Variables: map[string]interface{}{
					"serviceName": []string{"frontend", "route"},
				},
			},
			query: "SELECT operation FROM table WHERE serviceName IN ['frontend','route']",
		},
		{
			name: "query uses mixed types of variables",
			postData: &model.DashboardVars{
				Query: "SELECT body FROM table Where id = {{.id}} AND elapsed_time >= {{.elapsed_time}} AND serviceName IN {{.serviceName}}",
				Variables: map[string]interface{}{
					"serviceName":  []string{"frontend", "route"},
					"id":           "id",
					"elapsed_time": 1.24,
				},
			},
			query: "SELECT body FROM table Where id = 'id' AND elapsed_time >= 1.240000 AND serviceName IN ['frontend','route']",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var b bytes.Buffer
			err := json.NewEncoder(&b).Encode(tc.postData)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v2/variables/query", &b)
			query, err := prepareQuery(req)

			if tc.expectedErr {
				if err == nil {
					t.Errorf("expected error, but got nil")
				}
				if !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("expected error message contain: %s, but got: %s", tc.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, but got: %s", err.Error())
				}
				if query != tc.query {
					t.Errorf("expected query: %s, but got: %s", tc.query, query)
				}
			}
		})
	}
}

func TestRejectUnsupportedProjectScopedAutocomplete(t *testing.T) {
	ctx := ctxtypes.NewContextWithTenantScope(context.Background(), ctxtypes.TenantScope{
		OrgID: "11111111-1111-4111-8111-111111111111",
		ProjectID: "22222222-2222-4222-8222-222222222222",
	})
	t.Run("logs are supported", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v3/autocomplete/attribute_keys", nil).WithContext(ctx)
		recorder := httptest.NewRecorder()
		require.False(t, rejectUnsupportedProjectScopedAutocomplete(recorder, req, v3.DataSourceLogs))
	})
	t.Run("metrics are rejected until tenant metadata exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v3/autocomplete/attribute_keys", nil).WithContext(ctx)
		recorder := httptest.NewRecorder()
		require.True(t, rejectUnsupportedProjectScopedAutocomplete(recorder, req, v3.DataSourceMetrics))
	})
	t.Run("organization-wide requests retain existing behavior", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v3/autocomplete/attribute_keys", nil)
		recorder := httptest.NewRecorder()
		require.False(t, rejectUnsupportedProjectScopedAutocomplete(recorder, req, v3.DataSourceMetrics))
	})
}
