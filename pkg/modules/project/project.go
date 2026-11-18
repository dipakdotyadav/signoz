package project

import (
	"context"
	"net/http"

	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type Module interface {
	Create(context.Context, *types.Project) error
	Get(context.Context, valuer.UUID, valuer.UUID) (*types.Project, error)
	List(context.Context, valuer.UUID) ([]*types.Project, error)
	Update(context.Context, valuer.UUID, *types.Project) error
	Delete(context.Context, valuer.UUID, valuer.UUID) error
}

type Handler interface {
	Create(http.ResponseWriter, *http.Request)
	Get(http.ResponseWriter, *http.Request)
	List(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
	Delete(http.ResponseWriter, *http.Request)
}
