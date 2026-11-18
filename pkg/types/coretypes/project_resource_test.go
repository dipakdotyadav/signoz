package coretypes

import "testing"

func TestProjectResourceIsRegistered(t *testing.T) {
	resource, err := NewResourceFromTypeAndKind(TypeMetaResource, KindProject)
	if err != nil {
		t.Fatalf("expected project resource to be registered: %v", err)
	}

	if got, want := resource.Scope(VerbRead), "project:read"; got != want {
		t.Fatalf("unexpected project read scope: got %q, want %q", got, want)
	}

	for _, verb := range []Verb{VerbCreate, VerbList, VerbRead, VerbUpdate, VerbDelete, VerbAttach, VerbDetach} {
		if err := ErrIfVerbNotValidForResource(verb, ResourceRef{Type: TypeMetaResource, Kind: KindProject}); err != nil {
			t.Errorf("expected verb %q to be allowed for project: %v", verb.StringValue(), err)
		}
	}
}
