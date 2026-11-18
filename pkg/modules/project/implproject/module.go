package implproject

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/authz"
	"github.com/SigNoz/signoz/pkg/modules/user"
	"github.com/SigNoz/signoz/pkg/modules/serviceaccount"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/coretypes"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/SigNoz/signoz/pkg/modules/project"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type module struct { store types.ProjectStore; authz authz.AuthZ; users user.Getter; serviceAccounts serviceaccount.Module }

func NewModule(store types.ProjectStore, authzService authz.AuthZ, users user.Getter, serviceAccounts serviceaccount.Module) project.Module { return &module{store: store, authz: authzService, users: users, serviceAccounts: serviceAccounts} }

func (m *module) Create(ctx context.Context, input *types.Project) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = normalizeSlug(input.Slug)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" || input.Slug == "" {
		return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "project name and slug are required")
	}
	if len(input.Name) > 120 || len(input.Slug) > 80 {
		return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "project name or slug exceeds its maximum length")
	}
	if input.OrgID.IsZero() {
		return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "organization id is required")
	}
	return m.store.Create(ctx, input)
}

func (m *module) Get(ctx context.Context, orgID, id valuer.UUID) (*types.Project, error) {
	return m.store.Get(ctx, orgID, id)
}

func (m *module) List(ctx context.Context, orgID valuer.UUID) ([]*types.Project, error) {
	return m.store.List(ctx, orgID)
}

func (m *module) Update(ctx context.Context, orgID valuer.UUID, input *types.Project) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = normalizeSlug(input.Slug)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" || input.Slug == "" {
		return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "project name and slug are required")
	}
	if input.Status != types.ProjectStatusActive && input.Status != types.ProjectStatusArchived {
		return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "project status must be active or archived")
	}
	input.OrgID = orgID
	return m.store.Update(ctx, orgID, input)
}

func (m *module) Delete(ctx context.Context, orgID, id valuer.UUID) error {
	if _, err := m.store.Get(ctx, orgID, id); err != nil { return err }
	return m.store.Delete(ctx, orgID, id)
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r); lastDash = false
		case r == '-' || r == '_' || unicode.IsSpace(r):
			if b.Len() > 0 && !lastDash { b.WriteByte('-'); lastDash = true }
		}
	}
	return strings.Trim(b.String(), "-")
}

func projectMemberRelations(role string) ([]coretypes.Verb, error) {
	switch role {
	case "viewer":
		return []coretypes.Verb{coretypes.VerbRead}, nil
	case "editor":
		return []coretypes.Verb{coretypes.VerbRead, coretypes.VerbUpdate}, nil
	case "admin":
		return []coretypes.Verb{coretypes.VerbRead, coretypes.VerbUpdate, coretypes.VerbCreate, coretypes.VerbDelete, coretypes.VerbAttach, coretypes.VerbDetach, coretypes.VerbList}, nil
	default:
		return nil, errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "project role must be viewer, editor or admin")
	}
}

// projectMemberSubject validates the identity shape and scopes the subject to
// the supplied organization. It deliberately does not look up the principal;
// callers decide whether the principal must still exist.
func projectMemberSubject(orgID valuer.UUID, member *types.ProjectMember) (string, valuer.UUID, error) {
	if member == nil || strings.TrimSpace(member.PrincipalID) == "" {
		return "", valuer.UUID{}, errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "project member principalId is required")
	}

	id, err := valuer.NewUUID(strings.TrimSpace(member.PrincipalID))
	if err != nil {
		return "", valuer.UUID{}, err
	}

	var resource coretypes.Resource
	switch member.PrincipalType {
	case "user":
		resource = coretypes.NewResourceUser()
	case "serviceaccount":
		resource = coretypes.NewResourceServiceAccount()
	default:
		return "", valuer.UUID{}, errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "principalType must be user or serviceaccount")
	}

	subject, err := authtypes.NewSubject(resource, id.String(), orgID, nil)
	if err != nil {
		return "", valuer.UUID{}, err
	}
	return subject, id, nil
}

func (m *module) memberSubject(ctx context.Context, orgID valuer.UUID, member *types.ProjectMember) (string, error) {
	subject, id, err := projectMemberSubject(orgID, member)
	if err != nil {
		return "", err
	}

	switch member.PrincipalType {
	case "user":
		if _, err := m.users.GetUserByOrgIDAndID(ctx, orgID, id); err != nil {
			return "", err
		}
	case "serviceaccount":
		if _, err := m.serviceAccounts.Get(ctx, orgID, id); err != nil {
			return "", err
		}
	}
	return subject, nil
}

// projectMemberIdentity accepts only subjects belonging to this organization.
// This prevents malformed or foreign-organization tuples from leaking into
// the project-members API response.
func projectMemberIdentity(orgID valuer.UUID, subject string) (string, string, bool) {
	resources := []struct {
		principalType string
		resource      coretypes.Resource
	}{
		{principalType: "user", resource: coretypes.NewResourceUser()},
		{principalType: "serviceaccount", resource: coretypes.NewResourceServiceAccount()},
	}

	for _, candidate := range resources {
		prefix := candidate.resource.Prefix(orgID) + "/"
		if !strings.HasPrefix(subject, prefix) {
			continue
		}
		id := strings.TrimPrefix(subject, prefix)
		if strings.Contains(id, "/") {
			return "", "", false
		}
		parsed, err := valuer.NewUUID(id)
		if err != nil {
			return "", "", false
		}
		return candidate.principalType, parsed.String(), true
	}
	return "", "", false
}

func projectMemberRole(permissions map[string]bool) (string, error) {
	roles := map[string][]coretypes.Verb{
		"viewer": {coretypes.VerbRead},
		"editor": {coretypes.VerbRead, coretypes.VerbUpdate},
		"admin":  {coretypes.VerbRead, coretypes.VerbUpdate, coretypes.VerbCreate, coretypes.VerbDelete, coretypes.VerbAttach, coretypes.VerbDetach, coretypes.VerbList},
	}

	for role, verbs := range roles {
		if len(permissions) != len(verbs) {
			continue
		}
		matches := true
		for _, verb := range verbs {
			if !permissions[verb.StringValue()] {
				matches = false
				break
			}
		}
		if matches {
			return role, nil
		}
	}

	return "", errors.New(errors.TypeInternal, errors.CodeInternal, "project membership contains an unsupported permission combination")
}

func (m *module) AddMember(ctx context.Context, orgID, projectID valuer.UUID, member *types.ProjectMember) error {
	if _, err := m.store.Get(ctx, orgID, projectID); err != nil { return err }
	subject, err := m.memberSubject(ctx, orgID, member)
	if err != nil { return err }
	verbs, err := projectMemberRelations(member.Role)
	if err != nil { return err }
	resource := coretypes.NewResourceMetaResource(coretypes.KindProject)
	object := resource.Object(orgID, projectID.String())
	desired := make([]*openfgav1.TupleKey, 0, len(verbs))
	for _, verb := range verbs {
		desired = append(desired, &openfgav1.TupleKey{User: subject, Relation: verb.StringValue(), Object: object})
	}
	// Construct the complete permission set and reconcile it in one OpenFGA
	// write. This avoids a read/modify/write race and guarantees that changing
	// editor -> viewer removes every permission not granted to a viewer.
	allVerbs := []coretypes.Verb{coretypes.VerbRead, coretypes.VerbUpdate, coretypes.VerbCreate, coretypes.VerbDelete, coretypes.VerbAttach, coretypes.VerbDetach, coretypes.VerbList}
	all := make([]*openfgav1.TupleKey, 0, len(allVerbs))
	for _, verb := range allVerbs {
		all = append(all, &openfgav1.TupleKey{User: subject, Relation: verb.StringValue(), Object: object})
	}
	additions, deletions := authtypes.DiffTuples(all, desired)
	return m.authz.Write(ctx, additions, deletions)
}

func (m *module) RemoveMember(ctx context.Context, orgID, projectID valuer.UUID, member *types.ProjectMember) error {
	if _, err := m.store.Get(ctx, orgID, projectID); err != nil { return err }
	subject, _, err := projectMemberSubject(orgID, member)
	if err != nil { return err }
	resource := coretypes.NewResourceMetaResource(coretypes.KindProject)
	selector := coretypes.TypeMetaResource.MustSelector(projectID.String())
	deletions := make([]*openfgav1.TupleKey, 0, 7)
	for _, verb := range []coretypes.Verb{coretypes.VerbRead, coretypes.VerbUpdate, coretypes.VerbCreate, coretypes.VerbDelete, coretypes.VerbAttach, coretypes.VerbDetach, coretypes.VerbList} {
		deletions = append(deletions, &openfgav1.TupleKey{User: subject, Relation: verb.StringValue(), Object: resource.Object(orgID, selector.String())})
	}
	return m.authz.Write(ctx, nil, deletions)
}

func (m *module) ListMembers(ctx context.Context, orgID, projectID valuer.UUID) ([]*types.ProjectMember, error) {
	if _, err := m.store.Get(ctx, orgID, projectID); err != nil { return nil, err }
	resource := coretypes.NewResourceMetaResource(coretypes.KindProject)
	object := resource.Object(orgID, projectID.String())
	roles := map[string]map[string]bool{}
	for _, verb := range []coretypes.Verb{coretypes.VerbRead, coretypes.VerbUpdate, coretypes.VerbCreate, coretypes.VerbDelete, coretypes.VerbAttach, coretypes.VerbDetach, coretypes.VerbList} {
		tuples, err := m.authz.ReadTuples(ctx, &openfgav1.ReadRequestTupleKey{Object: object, Relation: verb.StringValue()})
		if err != nil { return nil, err }
		for _, tuple := range tuples {
			principalType, identity, ok := projectMemberIdentity(orgID, tuple.GetUser())
			if !ok {
				continue
			}
			key := principalType + ":" + identity
			if roles[key] == nil {
				roles[key] = map[string]bool{}
			}
			roles[key][verb.StringValue()] = true
		}
	}

	members := make([]*types.ProjectMember, 0, len(roles))
	for key, permissions := range roles {
		parts := strings.SplitN(key, ":", 2)
		role, err := projectMemberRole(permissions)
		if err != nil {
			return nil, err
		}
		members = append(members, &types.ProjectMember{PrincipalType: parts[0], PrincipalID: parts[1], Role: role})
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].PrincipalType != members[j].PrincipalType {
			return members[i].PrincipalType < members[j].PrincipalType
		}
		return members[i].PrincipalID < members[j].PrincipalID
	})
	return members, nil
}
