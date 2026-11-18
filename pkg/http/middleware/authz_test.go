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
		verb coretypes.Verb
	}{
		{name: "direct project grant", wantCalls: []string{"metaresource:project:read"}, verb: coretypes.VerbRead},
		{name: "organization role fallback", projectErr: forbidden, wantCalls: []string{"metaresource:project:read", "organization:organization:read"}, verb: coretypes.VerbRead},
		{name: "backend failure does not fall back", projectErr: backendFailure, wantCalls: []string{"metaresource:project:read"}, wantErr: true, verb: coretypes.VerbRead},
		{name: "organization denial", projectErr: forbidden, orgErr: forbidden, wantCalls: []string{"metaresource:project:read", "organization:organization:read"}, wantErr: true, verb: coretypes.VerbRead},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := make([]string, 0, 2)
			err := authorizeProjectAccess(orgID, projectID, tt.verb, func(resource coretypes.Resource, selector coretypes.Selector, verb coretypes.Verb) error {
				calls = append(calls, resource.Type().StringValue()+":"+resource.Kind().String()+":"+verb.StringValue())
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


func TestAuthorizeProjectMembershipFallsBackToOrganizationUpdate(t *testing.T) {
	orgID := valuer.MustNewUUID("018f0f2a-7b5c-7a00-8000-000000000001")
	projectID := valuer.MustNewUUID("018f0f2a-7b5c-7a00-8000-000000000002")
	forbidden := errors.New(errors.TypeForbidden, authtypes.ErrCodeAuthZForbidden, "denied")
	for _, verb := range []coretypes.Verb{coretypes.VerbAttach, coretypes.VerbDetach} {
		var organizationVerb coretypes.Verb
		err := authorizeProjectAccess(orgID, projectID, verb, func(resource coretypes.Resource, selector coretypes.Selector, checkedVerb coretypes.Verb) error {
			if resource.Type().Equals(coretypes.TypeOrganization) {
				organizationVerb = checkedVerb
				return nil
			}
			return forbidden
		})
		if err != nil { t.Fatalf("authorizeProjectAccess(%s) error = %v", verb.StringValue(), err) }
		if organizationVerb != coretypes.VerbUpdate {
			t.Fatalf("organization fallback verb = %s, want update", organizationVerb.StringValue())
		}
	}
}
