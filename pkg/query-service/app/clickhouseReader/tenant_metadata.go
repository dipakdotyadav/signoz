package clickhouseReader

import (
	"context"
	"strconv"

	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// tenantTagMetadataScope returns a mandatory ClickHouse predicate and bound
// arguments for authenticated project-scoped metadata reads. Organization-wide
// requests deliberately retain the legacy metadata table path.
func tenantTagMetadataScope(ctx context.Context, firstArg int) (string, []any, bool, error) {
	scope, ok := ctxtypes.TenantScopeFromContext(ctx)
	if !ok {
		return "", nil, false, nil
	}

	orgID, err := valuer.NewUUID(scope.OrgID)
	if err != nil {
		return "", nil, true, err
	}
	projectID, err := valuer.NewUUID(scope.ProjectID)
	if err != nil {
		return "", nil, true, err
	}

	return "org_id=$" + strconv.Itoa(firstArg) + " AND project_id=$" + strconv.Itoa(firstArg+1),
		[]any{orgID.String(), projectID.String()}, true, nil
}

