package implprojectingestionkey

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/modules/projectingestionkey"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type module struct { store projectingestionkey.Store }

func NewModule(store projectingestionkey.Store) projectingestionkey.Module { return &module{store: store} }

func (m *module) Create(ctx context.Context, orgID, projectID valuer.UUID, name string, expiresAt *time.Time) (*types.ProjectIngestionKey, string, error) {
	key, secret, err := types.NewProjectIngestionKey(orgID, projectID, name, expiresAt)
	if err != nil { return nil, "", err }
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return nil, "", errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "expiry must be in the future")
	}
	if err := m.store.Create(ctx, key); err != nil { return nil, "", err }
	return key, secret, nil
}

func (m *module) Get(ctx context.Context, orgID, projectID, id valuer.UUID) (*types.ProjectIngestionKey, error) {
	return m.store.Get(ctx, orgID, projectID, id)
}

func (m *module) List(ctx context.Context, orgID, projectID valuer.UUID) ([]*types.ProjectIngestionKey, error) {
	return m.store.List(ctx, orgID, projectID)
}

func (m *module) Revoke(ctx context.Context, orgID, projectID, id valuer.UUID) error {
	return m.store.Revoke(ctx, orgID, projectID, id)
}

func (m *module) Resolve(ctx context.Context, secret string) (*types.ProjectIngestionKey, error) {
	keyID, err := types.ParseProjectIngestionKey(secret)
	if err != nil { return nil, err }
	key, err := m.store.GetActiveByKeyID(ctx, keyID)
	if err != nil { return nil, err }
	if !key.VerifySecret(secret) {
		return nil, errors.New(errors.TypeUnauthenticated, types.ErrProjectIngestionKeyInvalid, "invalid project ingestion credential")
	}
	if err := key.Validate(time.Now()); err != nil { return nil, err }
	if err := m.store.UpdateLastUsedAt(ctx, keyID, time.Now().UTC()); err != nil { return nil, err }
	return key, nil
}
