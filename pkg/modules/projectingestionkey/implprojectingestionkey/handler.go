package implprojectingestionkey

import (
	"net/http"
	"time"

	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/modules/projectingestionkey"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/gorilla/mux"
)

type handler struct { module projectingestionkey.Module }

func NewHandler(module projectingestionkey.Module) projectingestionkey.Handler {
	return &handler{module: module}
}

type createRequest struct {
	Name string `json:"name"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type createResponse struct {
	*types.ProjectIngestionKey
	Secret string `json:"secret"`
}

func requestIdentity(r *http.Request) (valuer.UUID, valuer.UUID, error) {
	claims, err := authtypes.ClaimsFromContext(r.Context())
	if err != nil { return valuer.UUID{}, valuer.UUID{}, err }
	orgID, err := valuer.NewUUID(claims.OrgID)
	if err != nil { return valuer.UUID{}, valuer.UUID{}, err }
	projectID, err := valuer.NewUUID(mux.Vars(r)["id"])
	if err != nil { return valuer.UUID{}, valuer.UUID{}, err }
	return orgID, projectID, nil
}

func (h *handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, projectID, err := requestIdentity(r)
	if err != nil { render.Error(w, err); return }
	req := new(createRequest)
	if err := binding.JSON.BindBody(r.Body, req); err != nil { render.Error(w, err); return }
	key, secret, err := h.module.Create(r.Context(), orgID, projectID, req.Name, req.ExpiresAt)
	if err != nil { render.Error(w, err); return }
	render.Success(w, http.StatusCreated, createResponse{ProjectIngestionKey: key, Secret: secret})
}

func (h *handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, projectID, err := requestIdentity(r)
	if err != nil { render.Error(w, err); return }
	keys, err := h.module.List(r.Context(), orgID, projectID)
	if err != nil { render.Error(w, err); return }
	render.Success(w, http.StatusOK, keys)
}

func (h *handler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, projectID, err := requestIdentity(r)
	if err != nil { render.Error(w, err); return }
	id, err := valuer.NewUUID(mux.Vars(r)["credentialId"])
	if err != nil { render.Error(w, err); return }
	key, err := h.module.Get(r.Context(), orgID, projectID, id)
	if err != nil { render.Error(w, err); return }
	render.Success(w, http.StatusOK, key)
}

func (h *handler) Revoke(w http.ResponseWriter, r *http.Request) {
	orgID, projectID, err := requestIdentity(r)
	if err != nil { render.Error(w, err); return }
	id, err := valuer.NewUUID(mux.Vars(r)["credentialId"])
	if err != nil { render.Error(w, err); return }
	if err := h.module.Revoke(r.Context(), orgID, projectID, id); err != nil {
		render.Error(w, err)
		return
	}
	render.Success(w, http.StatusNoContent, nil)
}
