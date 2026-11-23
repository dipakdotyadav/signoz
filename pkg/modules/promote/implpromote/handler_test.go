package implpromote

import (
	"context"
	"net/http"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/stretchr/testify/require"
)

func TestValidatePromotionTenantScope(t *testing.T) {
	tests := []struct {
		name      string
		claims    authtypes.Claims
		scope     *ctxtypes.TenantScope
		wantError bool
	}{
		{name: "legacy organization-less request"},
		{name: "organization claim without project scope", claims: authtypes.Claims{OrgID: "org-a"}, wantError: true},
		{name: "matching authenticated project scope", claims: authtypes.Claims{OrgID: "org-a"}, scope: &ctxtypes.TenantScope{OrgID: "org-a", ProjectID: "project-a"}},
		{name: "organization mismatch", claims: authtypes.Claims{OrgID: "org-a"}, scope: &ctxtypes.TenantScope{OrgID: "org-b", ProjectID: "project-a"}, wantError: true},
		{name: "incomplete project scope", claims: authtypes.Claims{OrgID: "org-a"}, scope: &ctxtypes.TenantScope{OrgID: "org-a"}, wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := authtypes.NewContextWithClaims(context.Background(), tc.claims)
			if tc.scope != nil {
				ctx = ctxtypes.NewContextWithTenantScope(ctx, *tc.scope)
			}
			req := (&http.Request{}).WithContext(ctx)
			err := validatePromotionTenantScope(req)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
