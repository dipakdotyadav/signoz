package implprojectingestionkey

import (
	"context"
	"database/sql"
	"time"

	"github.com/SigNoz/signoz/pkg/modules/projectingestionkey"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type store struct { sqlstore sqlstore.SQLStore }

func NewStore(sqlstore sqlstore.SQLStore) projectingestionkey.Store {
	return &store{sqlstore: sqlstore}
}

func (s *store) Create(ctx context.Context, key *types.ProjectIngestionKey) error {
	// Validate the organization/project pair against the canonical project row.
	// This prevents credentials from being bound to a project in another org.
	project := new(types.Project)
	err := s.sqlstore.BunDBCtx(ctx).NewSelect().Model(project).
		Where("project.id = ?", key.ProjectID).
		Where("project.org_id = ?", key.OrgID).
		Where("project.status = ?", types.ProjectStatusActive).Scan(ctx)
	if err != nil {
		return s.sqlstore.WrapNotFoundErrf(err, types.ErrProjectNotFound, "active project %s does not exist in organization %s", key.ProjectID, key.OrgID)
	}
	_, err = s.sqlstore.BunDBCtx(ctx).NewInsert().Model(key).Exec(ctx)
	return err
}

func (s *store) Get(ctx context.Context, orgID, projectID, id valuer.UUID) (*types.ProjectIngestionKey, error) {
	key := new(types.ProjectIngestionKey)
	err := s.sqlstore.BunDBCtx(ctx).NewSelect().Model(key).
		Where("project_ingestion_key.org_id = ?", orgID).
		Where("project_ingestion_key.project_id = ?", projectID).
		Where("project_ingestion_key.id = ?", id).Scan(ctx)
	if err != nil { return nil, s.sqlstore.WrapNotFoundErrf(err, types.ErrProjectIngestionKeyNotFound, "project ingestion key not found") }
	return key, nil
}

func (s *store) GetActiveByKeyID(ctx context.Context, keyID string) (*types.ProjectIngestionKey, error) {
	key := new(types.ProjectIngestionKey)
	err := s.sqlstore.BunDBCtx(ctx).NewSelect().Model(key).
		Join("JOIN projects AS project").
		JoinOn("project.id = project_ingestion_key.project_id").
		JoinOn("project.org_id = project_ingestion_key.org_id").
		Where("project_ingestion_key.key_id = ?", keyID).
		Where("project.status = ?", types.ProjectStatusActive).
		Where("project_ingestion_key.revoked_at IS NULL").Scan(ctx)
	if err != nil { return nil, s.sqlstore.WrapNotFoundErrf(err, types.ErrProjectIngestionKeyNotFound, "active project ingestion key not found") }
	return key, nil
}

func (s *store) List(ctx context.Context, orgID, projectID valuer.UUID) ([]*types.ProjectIngestionKey, error) {
	keys := make([]*types.ProjectIngestionKey, 0)
	err := s.sqlstore.BunDBCtx(ctx).NewSelect().Model(&keys).
		Where("project_ingestion_key.org_id = ?", orgID).
		Where("project_ingestion_key.project_id = ?", projectID).
		OrderExpr("project_ingestion_key.created_at DESC").Scan(ctx)
	if err != nil { return nil, err }
	return keys, nil
}

func (s *store) Revoke(ctx context.Context, orgID, projectID, id valuer.UUID) error {
	now := time.Now().UTC()
	result, err := s.sqlstore.BunDBCtx(ctx).NewUpdate().Model((*types.ProjectIngestionKey)(nil)).
		Set("revoked_at = ?", now).Set("updated_at = ?", now).
		Where("org_id = ?", orgID).Where("project_id = ?", projectID).
		Where("id = ?", id).Where("revoked_at IS NULL").Exec(ctx)
	if err != nil { return err }
	if count, _ := result.RowsAffected(); count == 0 {
		return s.sqlstore.WrapNotFoundErrf(sql.ErrNoRows, types.ErrProjectIngestionKeyNotFound, "active project ingestion key not found")
	}
	return nil
}

func (s *store) UpdateLastUsedAt(ctx context.Context, keyID string, at time.Time) error {
	_, err := s.sqlstore.BunDBCtx(ctx).NewUpdate().Model((*types.ProjectIngestionKey)(nil)).
		Set("last_used_at = ?", at).Where("key_id = ?", keyID).Where("revoked_at IS NULL").Exec(ctx)
	return err
}
