package implpromote

import (
	"net/http"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/modules/promote"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/promotetypes"
)

type handler struct {
	module promote.Module
}

func NewHandler(module promote.Module) promote.Handler {
	return &handler{module: module}
}

func validatePromotionTenantScope(r *http.Request) error {
	claims, err := authtypes.ClaimsFromContext(r.Context())
	if err != nil {
		return errors.NewUnauthenticatedf(errors.CodeUnauthenticated, "unauthenticated")
	}
	scope, ok := ctxtypes.TenantScopeFromContext(r.Context())
	if claims.OrgID != "" {
		if !ok || !scope.Valid() {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "authenticated project scope is required for promotion operations")
		}
		if scope.OrgID != claims.OrgID {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "promotion tenant scope does not match authenticated organization")
		}
	}
	return nil
}

func (h *handler) PromotePaths(w http.ResponseWriter, r *http.Request) {
	if err := validatePromotionTenantScope(r); err != nil {
		render.Error(w, err)
		return
	}

	var req []*promotetypes.PromotePath
	if err := binding.JSON.BindBody(r.Body, &req); err != nil {
		render.Error(w, err)
		return
	}
	if err := promotetypes.ValidatePromotePaths(req); err != nil {
		render.Error(w, err)
		return
	}

	err = h.module.PromotePaths(r.Context(), req...)
	if err != nil {
		render.Error(w, err)
		return
	}

	render.Success(w, http.StatusCreated, nil)
}

func (h *handler) ListPromotedPaths(w http.ResponseWriter, r *http.Request) {
	if err := validatePromotionTenantScope(r); err != nil {
		render.Error(w, err)
		return
	}

	var filters promotetypes.ListPromotedPathsFilters
	if err := binding.Query.BindQuery(r.URL.Query(), &filters); err != nil {
		render.Error(w, err)
		return
	}
	if err := filters.Validate(); err != nil {
		render.Error(w, err)
		return
	}

	paths, err := h.module.ListPromotedPaths(r.Context(), filters)
	if err != nil {
		render.Error(w, err)
		return
	}

	render.Success(w, http.StatusOK, paths)
}

func (h *handler) IndexMaterializedPaths(w http.ResponseWriter, r *http.Request) {
	var req promotetypes.PostableIndexMaterializedPaths
	if err := binding.JSON.BindBody(r.Body, &req); err != nil {
		render.Error(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		render.Error(w, err)
		return
	}

	paths, err := h.module.IndexMaterializedPaths(r.Context(), req)
	if err != nil {
		render.Error(w, err)
		return
	}

	render.Success(w, http.StatusOK, paths)
}
