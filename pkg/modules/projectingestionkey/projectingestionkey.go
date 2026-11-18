package projectingestionkey

import (
	"context"
	"net/http"
	"time"

	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type Store interface {
	Create(context.Context, *types.ProjectIngestionKey) error
	Get(context.Context, valuer.UUID, valuer.UUID, valuer.UUID) (*types.ProjectIngestionKey, error)
	GetActiveByKeyID(context.Context, string) (*types.ProjectIngestionKey, error)
	List(context.Context, valuer.UUID, valuer.UUID) ([]*types.ProjectIngestionKey, error)
	Revoke(context.Context, valuer.UUID, valuer.UUID, valuer.UUID) error
	UpdateLastUsedAt(context.Context, string, time.Time) error
}

type Module interface {
	Create(context.Context, valuer.UUID, valuer.UUID, string, *time.Time) (*types.ProjectIngestionKey, string, error)
	Get(context.Context, valuer.UUID, valuer.UUID, valuer.UUID) (*types.ProjectIngestionKey, error)
	List(context.Context, valuer.UUID, valuer.UUID) ([]*types.ProjectIngestionKey, error)
	Revoke(context.Context, valuer.UUID, valuer.UUID, valuer.UUID) error
	Resolve(context.Context, string) (*types.ProjectIngestionKey, error)
}

type Handler interface {
	Create(http.ResponseWriter, *http.Request)
	List(http.ResponseWriter, *http.Request)
	Get(http.ResponseWriter, *http.Request)
	Revoke(http.ResponseWriter, *http.Request)
}
