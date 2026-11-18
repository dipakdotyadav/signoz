package implproject

import (
	"context"
	"strings"
	"unicode"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/modules/project"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type module struct { store types.ProjectStore }

func NewModule(store types.ProjectStore) project.Module { return &module{store: store} }

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
