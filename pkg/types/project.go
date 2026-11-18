package types

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/uptrace/bun"
)

var (
	ErrProjectAlreadyExists = errors.MustNewCode("project_already_exists")
	ErrProjectNotFound = errors.MustNewCode("project_not_found")

	ProjectStatusActive   = valuer.NewString("active")
	ProjectStatusArchived = valuer.NewString("archived")
)

type Project struct {
	bun.BaseModel `bun:"table:projects,alias:project"`

	Identifiable
	OrgID       valuer.UUID   `bun:"org_id,type:text,notnull" json:"orgId"`
	Name        string        `bun:"name,type:text,notnull" json:"name"`
	Slug        string        `bun:"slug,type:text,notnull" json:"slug"`
	Description string        `bun:"description,type:text,nullzero" json:"description,omitempty"`
	Status      valuer.String `bun:"status,type:text,notnull" json:"status"`
	TimeAuditable
}

func NewProject(orgID valuer.UUID, name, slug, description string) *Project {
	now := time.Now()
	return &Project{
		Identifiable: Identifiable{ID: valuer.GenerateUUID()},
		OrgID:        orgID,
		Name:         name,
		Slug:         slug,
		Description:  description,
		Status:       ProjectStatusActive,
		TimeAuditable: TimeAuditable{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}


type ProjectStore interface {
	Create(context.Context, *Project) error
	Get(context.Context, valuer.UUID, valuer.UUID) (*Project, error)
	List(context.Context, valuer.UUID) ([]*Project, error)
	Update(context.Context, valuer.UUID, *Project) error
	Delete(context.Context, valuer.UUID, valuer.UUID) error
}

type ProjectMember struct {
	PrincipalType string `json:"principalType"`
	PrincipalID   string `json:"principalId"`
	Role          string `json:"role"`
}

