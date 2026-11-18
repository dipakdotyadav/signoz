package implproject

import (
	"context"
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

func (m *module) memberSubject(ctx context.Context, orgID valuer.UUID, member *types.ProjectMember) (string, error) {
	if member == nil || member.PrincipalID == "" {
		return "", errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "project member principalId is required")
	}
	id, err := valuer.NewUUID(member.PrincipalID)
	if err != nil { return "", err }
	switch member.PrincipalType {
	case "user":
		if _, err := m.users.GetUserByOrgIDAndID(ctx, orgID, id); err != nil { return "", err }
		return authtypes.MustNewSubject(coretypes.NewResourceUser(), id.String(), orgID, nil), nil
	case "serviceaccount":
		if _, err := m.serviceAccounts.Get(ctx, orgID, id); err != nil { return "", err }
		return authtypes.MustNewSubject(coretypes.NewResourceServiceAccount(), id.String(), orgID, nil), nil
	default:
		return "", errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "principalType must be user or serviceaccount")
	}
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
	existing := make([]*openfgav1.TupleKey, 0, 7)
	for _, verb := range []coretypes.Verb{coretypes.VerbRead, coretypes.VerbUpdate, coretypes.VerbCreate, coretypes.VerbDelete, coretypes.VerbAttach, coretypes.VerbDetach, coretypes.VerbList} {
		tuples, err := m.authz.ReadTuples(ctx, &openfgav1.ReadRequestTupleKey{User: subject, Relation: verb.StringValue(), Object: object})
		if err != nil { return err }
		existing = append(existing, tuples...)
	}
	additions, deletions := authtypes.DiffTuples(existing, desired)
	return m.authz.Write(ctx, additions, deletions)
}

func (m *module) RemoveMember(ctx context.Context, orgID, projectID valuer.UUID, member *types.ProjectMember) error {
	if _, err := m.store.Get(ctx, orgID, projectID); err != nil { return err }
	subject, err := m.memberSubject(ctx, orgID, member)
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
			parts := strings.Split(tuple.GetUser(), "/")
			if len(parts) == 0 { continue }
			identity := parts[len(parts)-1]
			principalType := strings.SplitN(tuple.GetUser(), ":", 2)[0]
			if principalType != "user" && principalType != "serviceaccount" { continue }
			key := principalType + ":" + identity
			if roles[key] == nil { roles[key] = map[string]bool{} }
			roles[key][verb.StringValue()] = true
		}
	}
	members := make([]*types.ProjectMember, 0, len(roles))
	for key, permissions := range roles {
		parts := strings.SplitN(key, ":", 2)
		role := "viewer"
		if permissions["create"] || permissions["delete"] || permissions["attach"] || permissions["detach"] || permissions["list"] { role = "admin" } else if permissions["update"] { role = "editor" }
		members = append(members, &types.ProjectMember{PrincipalType: parts[0], PrincipalID: parts[1], Role: role})
	}
	return members, nil
}
