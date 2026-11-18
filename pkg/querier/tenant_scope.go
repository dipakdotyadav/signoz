package querier

import (
	"context"
	"fmt"
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// applyTenantScope adds server-derived organization and project predicates to
// every structured telemetry query. The collector's tenant-context processor
// writes these attributes from authenticated ingestion identity, not client
// supplied resource attributes. Unsupported query languages fail closed.
func applyTenantScope(ctx context.Context, orgID valuer.UUID, req *qbtypes.QueryRangeRequest) error {
	if req == nil {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "query request is required")
	}

	scope, ok := ctxtypes.TenantScopeFromContext(ctx)
	if !ok {
		return nil
	}

	scopeOrgID, err := valuer.NewUUID(scope.OrgID)
	if err != nil {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid organization in authenticated tenant scope")
	}
	projectID, err := valuer.NewUUID(scope.ProjectID)
	if err != nil {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid project in authenticated tenant scope")
	}
	if scopeOrgID.String() != orgID.String() {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "authenticated tenant organization does not match query organization")
	}

	// UUID parsing above makes these literals canonical and prevents expression
	// injection. Both predicates are mandatory; project IDs are org-owned.
	tenantFilter := fmt.Sprintf(
		"resource.signoz.org.id = '%s' AND resource.signoz.project.id = '%s'",
		scopeOrgID.String(), projectID.String(),
	)

	for i := range req.CompositeQuery.Queries {
		env := &req.CompositeQuery.Queries[i]
		switch env.Type {
		case qbtypes.QueryTypeBuilder:
			switch spec := env.Spec.(type) {
			case qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]:
				spec.Filter = appendTenantFilter(spec.Filter, tenantFilter)
				env.Spec = spec
			case qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]:
				spec.Filter = appendTenantFilter(spec.Filter, tenantFilter)
				env.Spec = spec
			case qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]:
				spec.Filter = appendTenantFilter(spec.Filter, tenantFilter)
				env.Spec = spec
			default:
				return errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported builder query spec %T for project-scoped query", env.Spec)
			}
		case qbtypes.QueryTypeBuilderAI:
			spec, ok := env.Spec.(qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation])
			if !ok {
				return errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported AI builder query spec %T for project-scoped query", env.Spec)
			}
			spec.Filter = appendTenantFilter(spec.Filter, tenantFilter)
			env.Spec = spec
		case qbtypes.QueryTypeFormula:
			// Formulas operate on results from the already-scoped builder queries.
		default:
			return errors.NewInvalidInputf(
				errors.CodeInvalidInput,
				"query type %q is not yet supported for project-scoped execution; use a structured builder query",
				env.Type.StringValue(),
			)
		}
	}

	return nil
}

func appendTenantFilter(existing *qbtypes.Filter, tenantFilter string) *qbtypes.Filter {
	if existing == nil || strings.TrimSpace(existing.Expression) == "" {
		return &qbtypes.Filter{Expression: tenantFilter}
	}
	return &qbtypes.Filter{
		Expression: fmt.Sprintf("(%s) AND (%s)", existing.Expression, tenantFilter),
	}
}
