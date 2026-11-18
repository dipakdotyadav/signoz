package types

import (
	"time"

	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/uptrace/bun"
)

var (
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
