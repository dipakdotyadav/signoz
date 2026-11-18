package implproject

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/types/coretypes"
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
