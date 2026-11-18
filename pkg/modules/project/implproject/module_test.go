package implproject

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/coretypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

func TestNormalizeSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: " Production API ", want: "production-api"},
		{input: "payments__service", want: "payments-service"},
		{input: "A---B", want: "a-b"},
		{input: "  ", want: ""},
		{input: "v2.0/API", want: "v20api"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeSlug(tt.input); got != tt.want {
				t.Fatalf("normalizeSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestProjectMemberRelations(t *testing.T) {
	tests := []struct {
		role string
		want []coretypes.Verb
	}{
		{role: "viewer", want: []coretypes.Verb{coretypes.VerbRead}},
		{role: "editor", want: []coretypes.Verb{coretypes.VerbRead, coretypes.VerbUpdate}},
		{role: "admin", want: []coretypes.Verb{coretypes.VerbRead, coretypes.VerbUpdate, coretypes.VerbCreate, coretypes.VerbDelete, coretypes.VerbAttach, coretypes.VerbDetach, coretypes.VerbList}},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got, err := projectMemberRelations(tt.role)
			if err != nil { t.Fatalf("unexpected error: %v", err) }
			if len(got) != len(tt.want) { t.Fatalf("got %d relations, want %d", len(got), len(tt.want)) }
			for i := range got {
				if got[i] != tt.want[i] { t.Fatalf("relation[%d] = %q, want %q", i, got[i].StringValue(), tt.want[i].StringValue()) }
			}
		})
	}
}

func TestProjectMemberRelationsRejectsUnknownRole(t *testing.T) {
	if _, err := projectMemberRelations("owner"); err == nil {
		t.Fatal("expected unknown project role to be rejected")
	}
}

func TestProjectMemberSubjectIsOrganizationScoped(t *testing.T) {
	orgID := valuer.MustNewUUID("018f0f2a-7b5c-7a00-8000-000000000001")
	member := &types.ProjectMember{PrincipalType: "user", PrincipalID: "018f0f2a-7b5c-7a00-8000-000000000002", Role: "viewer"}
	subject, _, err := projectMemberSubject(orgID, member)
	if err != nil { t.Fatalf("projectMemberSubject() error = %v", err) }
	want := "user:organization/" + orgID.String() + "/user/" + member.PrincipalID
	if subject != want { t.Fatalf("subject = %q, want %q", subject, want) }
}

func TestProjectMemberIdentityRejectsForeignOrganization(t *testing.T) {
	orgID := valuer.MustNewUUID("018f0f2a-7b5c-7a00-8000-000000000001")
	foreignOrgID := valuer.MustNewUUID("018f0f2a-7b5c-7a00-8000-000000000003")
	subject := coretypes.NewResourceUser().Prefix(foreignOrgID) + "/018f0f2a-7b5c-7a00-8000-000000000002"
	if _, _, ok := projectMemberIdentity(orgID, subject); ok { t.Fatal("expected foreign-organization subject to be rejected") }
}

func TestProjectMemberIdentityParsesValidSubjects(t *testing.T) {
	orgID := valuer.MustNewUUID("018f0f2a-7b5c-7a00-8000-000000000001")
	id := "018f0f2a-7b5c-7a00-8000-000000000002"
	for _, tc := range []struct { name string; resource coretypes.Resource; wantType string }{
		{name: "user", resource: coretypes.NewResourceUser(), wantType: "user"},
		{name: "service account", resource: coretypes.NewResourceServiceAccount(), wantType: "serviceaccount"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			subject := tc.resource.Prefix(orgID) + "/" + id
			gotType, gotID, ok := projectMemberIdentity(orgID, subject)
			if !ok || gotType != tc.wantType || gotID != id { t.Fatalf("projectMemberIdentity() = (%q, %q, %v)", gotType, gotID, ok) }
		})
	}
}

func TestProjectMemberRoleRequiresExactPermissionSet(t *testing.T) {
	if _, err := projectMemberRole(map[string]bool{"read": true, "update": true, "delete": true}); err == nil {
		t.Fatal("expected mixed permission set to be rejected")
	}
	if role, err := projectMemberRole(map[string]bool{"read": true, "update": true}); err != nil || role != "editor" {
		t.Fatalf("projectMemberRole() = (%q, %v), want (editor, nil)", role, err)
	}
}
