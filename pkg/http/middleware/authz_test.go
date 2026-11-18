package middleware

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/coretypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

func TestAuthorizeProjectAccess(t *testing.T) {
	orgID := valuer.MustNewUUID("018f0f2a-7b5c-7a00-8000-000000000001")
	projectID := valuer.MustNewUUID("018f0f2a-7b5c-7a00-8000-000000000002")

	forbidden := errors.New(errors.TypeForbidden, authtypes.ErrCodeAuthZForbidden, "denied")
	backendFailure := errors.New(errors.TypeInternal, errors.CodeInternal, "OpenFGA unavailable")

	tests := []struct {
		name string
		projectErr error
		orgErr error
		wantCalls []string
		wantErr bool
	}{
		{name: "direct project grant", wantCalls: []string{"metaresource:project"}},
		{name: "organization role fallback", projectErr: forbidden, wantCalls: []string{"metaresource:project", "organization:organization"}},
		{name: "backend failure does not fall back", projectErr: backendFailure, wantCalls: []string{"metaresource:project"}, wantErr: true},
		{name: "organization denial", projectErr: forbidden, orgErr: forbidden, wantCalls: []string{"metaresource:project", "organization:organization"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := make([]string, 0, 2)
			err := authorizeProjectAccess(orgID, projectID, coretypes.VerbRead, func(resource coretypes.Resource, selector coretypes.Selector, verb coretypes.Verb) error {
				calls = append(calls, resource.Type().StringValue()+":"+resource.Kind().String())
				if resource.Type().Equals(coretypes.TypeMetaResource) {
					return tt.projectErr
				}
				return tt.orgErr
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("authorizeProjectAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(calls) != len(tt.wantCalls) {
				t.Fatalf("authorization calls = %v, want %v", calls, tt.wantCalls)
			}
			for i := range calls {
				if calls[i] != tt.wantCalls[i] {
					t.Fatalf("authorization calls = %v, want %v", calls, tt.wantCalls)
				}
			}
		})
	}
}

