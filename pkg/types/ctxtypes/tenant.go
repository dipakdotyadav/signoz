package ctxtypes

import "context"

type tenantScopeKey struct{}

// TenantScope is the authenticated data boundary for a request. IDs are
// canonical UUID strings resolved by server-side authentication/authorization;
// callers must never populate this from telemetry resource attributes.
type TenantScope struct {
	OrgID     string
	ProjectID string
}

func (s TenantScope) Valid() bool {
	return s.OrgID != "" && s.ProjectID != ""
}

// NewContextWithTenantScope attaches an authenticated tenant scope.
func NewContextWithTenantScope(ctx context.Context, scope TenantScope) context.Context {
	return context.WithValue(ctx, tenantScopeKey{}, scope)
}

// TenantScopeFromContext returns the authenticated tenant scope, if present.
func TenantScopeFromContext(ctx context.Context) (TenantScope, bool) {
	scope, ok := ctx.Value(tenantScopeKey{}).(TenantScope)
	if !ok || !scope.Valid() {
		return TenantScope{}, false
	}
	return scope, true
}
