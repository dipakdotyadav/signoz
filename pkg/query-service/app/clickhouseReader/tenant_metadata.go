package clickhouseReader

import (
	"context"
	"fmt"
	"strings"
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


func buildLogTagAttributeKeysQuery(
	logsDB, table string,
	scoped bool,
	scopeFilter string,
	scopeArgs []any,
	tagType, searchText string,
	limit int,
) (string, []any) {
	where := []string{"tag_type != 'logfield'"}
	args := make([]any, 0, len(scopeArgs)+3)
	if scoped {
		where = append(where, scopeFilter)
		args = append(args, scopeArgs...)
	}
	if tagType != "" {
		args = append(args, tagType)
		where = append(where, fmt.Sprintf("tag_type=$%d", len(args)))
	}
	if searchText != "" {
		args = append(args, "%"+searchText+"%")
		where = append(where, fmt.Sprintf("tag_key ILIKE $%d", len(args)))
	}
	args = append(args, limit)

	query := fmt.Sprintf(
		"select distinct tag_key, tag_type, tag_data_type from %s.%s where %s limit $%d",
		logsDB, table, strings.Join(where, " AND "), len(args),
	)
	return query, args
}
