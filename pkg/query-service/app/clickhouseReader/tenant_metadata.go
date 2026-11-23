package clickhouseReader

import (
	"context"

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

	return "org_id=$" + itoa(firstArg) + " AND project_id=$" + itoa(firstArg+1),
		[]any{orgID.String(), projectID.String()}, true, nil
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[i:])
}
