package implproject

import (
	"net/http"
	"time"

	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/modules/project"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/gorilla/mux"
)

type handler struct { module project.Module }

func NewHandler(module project.Module) project.Handler { return &handler{module: module} }

type projectRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Description string `json:"description"`
}

func orgID(r *http.Request) (valuer.UUID, error) {
	claims, err := authtypes.ClaimsFromContext(r.Context())
	if err != nil { return valuer.UUID{}, err }
	return valuer.NewUUID(claims.OrgID)
}

func (h *handler) Create(w http.ResponseWriter, r *http.Request) {
	org, err := orgID(r); if err != nil { render.Error(w, err); return }
	req := new(projectRequest)
	if err := binding.JSON.BindBody(r.Body, req); err != nil { render.Error(w, err); return }
	model := types.NewProject(org, req.Name, req.Slug, req.Description)
	if err := h.module.Create(r.Context(), model); err != nil { render.Error(w, err); return }
	render.Success(w, http.StatusCreated, model)
}

func (h *handler) List(w http.ResponseWriter, r *http.Request) {
	org, err := orgID(r); if err != nil { render.Error(w, err); return }
	items, err := h.module.List(r.Context(), org)
	if err != nil { render.Error(w, err); return }
	render.Success(w, http.StatusOK, items)
}

func (h *handler) Get(w http.ResponseWriter, r *http.Request) {
	org, err := orgID(r); if err != nil { render.Error(w, err); return }
	id, err := valuer.NewUUID(mux.Vars(r)["id"]); if err != nil { render.Error(w, err); return }
	item, err := h.module.Get(r.Context(), org, id)
	if err != nil { render.Error(w, err); return }
	render.Success(w, http.StatusOK, item)
}

func (h *handler) Update(w http.ResponseWriter, r *http.Request) {
	org, err := orgID(r); if err != nil { render.Error(w, err); return }
	id, err := valuer.NewUUID(mux.Vars(r)["id"]); if err != nil { render.Error(w, err); return }
	req := new(projectRequest)
	if err := binding.JSON.BindBody(r.Body, req); err != nil { render.Error(w, err); return }
	item, err := h.module.Get(r.Context(), org, id)
	if err != nil { render.Error(w, err); return }
	item.Name, item.Slug, item.Description = req.Name, req.Slug, req.Description
	if status := r.URL.Query().Get("status"); status != "" { item.Status = valuer.NewString(status) }
	item.UpdatedAt = time.Now()
	if err := h.module.Update(r.Context(), org, item); err != nil { render.Error(w, err); return }
	render.Success(w, http.StatusOK, item)
}

func (h *handler) Delete(w http.ResponseWriter, r *http.Request) {
	org, err := orgID(r); if err != nil { render.Error(w, err); return }
	id, err := valuer.NewUUID(mux.Vars(r)["id"]); if err != nil { render.Error(w, err); return }
	if err := h.module.Delete(r.Context(), org, id); err != nil { render.Error(w, err); return }
	render.Success(w, http.StatusNoContent, nil)
}

