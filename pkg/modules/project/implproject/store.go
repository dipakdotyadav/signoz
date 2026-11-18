package implproject

import (
	"context"
	"database/sql"
	"time"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type store struct { sqlstore sqlstore.SQLStore }

func NewStore(sqlstore sqlstore.SQLStore) types.ProjectStore {
	return &store{sqlstore: sqlstore}
}

func (s *store) Create(ctx context.Context, project *types.Project) error {
	_, err := s.sqlstore.BunDBCtx(ctx).NewInsert().Model(project).Exec(ctx)
	if err != nil {
		return s.sqlstore.WrapAlreadyExistsErrf(err, types.ErrProjectAlreadyExists, "project slug %q already exists in organization", project.Slug)
	}
	return nil
}

func (s *store) Get(ctx context.Context, orgID, id valuer.UUID) (*types.Project, error) {
	project := new(types.Project)
	err := s.sqlstore.BunDBCtx(ctx).NewSelect().Model(project).
		Where("project.org_id = ?", orgID).Where("project.id = ?", id).Scan(ctx)
	if err != nil {
		return nil, s.sqlstore.WrapNotFoundErrf(err, types.ErrProjectNotFound, "project %s does not exist in organization %s", id, orgID)
	}
	return project, nil
}

func (s *store) List(ctx context.Context, orgID valuer.UUID) ([]*types.Project, error) {
	projects := make([]*types.Project, 0)
	err := s.sqlstore.BunDBCtx(ctx).NewSelect().Model(&projects).
		Where("project.org_id = ?", orgID).OrderExpr("project.created_at ASC").Scan(ctx)
	if err != nil { return nil, err }
	return projects, nil
}

func (s *store) Update(ctx context.Context, orgID valuer.UUID, project *types.Project) error {
	result, err := s.sqlstore.BunDBCtx(ctx).NewUpdate().Model(project).
		Column("name", "slug", "description", "status", "updated_at").
		Where("project.id = ?", project.ID).Where("project.org_id = ?", orgID).Exec(ctx)
	if err != nil {
		return s.sqlstore.WrapAlreadyExistsErrf(err, types.ErrProjectAlreadyExists, "project slug %q already exists in organization", project.Slug)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return s.sqlstore.WrapNotFoundErrf(sql.ErrNoRows, types.ErrProjectNotFound, "project %s does not exist in organization %s", project.ID, orgID)
	}
	return nil
}

func (s *store) Delete(ctx context.Context, orgID, id valuer.UUID) error {
	project := new(types.Project)
	result, err := s.sqlstore.BunDBCtx(ctx).NewUpdate().Model(project).
		Set("status = ?", types.ProjectStatusArchived).
		Set("updated_at = ?", time.Now()).
		Where("org_id = ?", orgID).Where("id = ?", id).Exec(ctx)
	if err != nil { return err }
	if affected, _ := result.RowsAffected(); affected == 0 {
		return s.sqlstore.WrapNotFoundErrf(sql.ErrNoRows, types.ErrProjectNotFound, "project %s does not exist in organization %s", id, orgID)
	}
	return nil
}
