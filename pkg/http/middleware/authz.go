package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/SigNoz/signoz/pkg/authz"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/modules/organization"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/coretypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/gorilla/mux"
)

const (
	authzDeniedMessage string = "::AUTHZ-DENIED::"
)

type selectorCallbackWithoutClaimsFn func(*http.Request, []*types.Organization) ([]coretypes.Selector, valuer.UUID, error)

type AuthZ struct {
	logger       *slog.Logger
	orgGetter    organization.Getter
	authzService authz.AuthZ
}

func NewAuthZ(logger *slog.Logger, orgGetter organization.Getter, authzService authz.AuthZ) *AuthZ {
	if logger == nil {
		panic("cannot build authz middleware, logger is empty")
	}

	return &AuthZ{logger: logger, orgGetter: orgGetter, authzService: authzService}
}

func (middleware *AuthZ) ViewAccess(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		claims, err := authtypes.ClaimsFromContext(ctx)
		if err != nil {
			render.Error(rw, err)
			return
		}

		selectors := []coretypes.Selector{
			coretypes.TypeRole.MustSelector(authtypes.SigNozAdminRoleName),
			coretypes.TypeRole.MustSelector(authtypes.SigNozEditorRoleName),
			coretypes.TypeRole.MustSelector(authtypes.SigNozViewerRoleName),
		}

		err = middleware.authzService.CheckWithTupleCreation(
			ctx,
			claims,
			valuer.MustNewUUID(claims.OrgID),
			authtypes.Relation{Verb: coretypes.VerbAssignee},
			coretypes.NewResourceRole(),
			selectors,
			selectors,
		)
		if err != nil {
			middleware.logger.WarnContext(ctx, authzDeniedMessage, slog.Any("claims", claims))
			if errors.Asc(err, authtypes.ErrCodeAuthZForbidden) {
				render.Error(rw, errors.New(errors.TypeForbidden, authtypes.ErrCodeAuthZForbidden, "only viewers/editors/admins can access this resource"))
				return
			}

			render.Error(rw, err)
			return
		}

		next(rw, req)
	})
}

func (middleware *AuthZ) EditAccess(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		claims, err := authtypes.ClaimsFromContext(ctx)
		if err != nil {
			render.Error(rw, err)
			return
		}

		selectors := []coretypes.Selector{
			coretypes.TypeRole.MustSelector(authtypes.SigNozAdminRoleName),
			coretypes.TypeRole.MustSelector(authtypes.SigNozEditorRoleName),
		}

		err = middleware.authzService.CheckWithTupleCreation(
			ctx,
			claims,
			valuer.MustNewUUID(claims.OrgID),
			authtypes.Relation{Verb: coretypes.VerbAssignee},
			coretypes.NewResourceRole(),
			selectors,
			selectors,
		)
		if err != nil {
			middleware.logger.WarnContext(ctx, authzDeniedMessage, slog.Any("claims", claims))
			if errors.Asc(err, authtypes.ErrCodeAuthZForbidden) {
				render.Error(rw, errors.New(errors.TypeForbidden, authtypes.ErrCodeAuthZForbidden, "only editors/admins can access this resource"))
				return
			}

			render.Error(rw, err)
			return
		}

		next(rw, req)
	})
}

func (middleware *AuthZ) AdminAccess(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		claims, err := authtypes.ClaimsFromContext(ctx)
		if err != nil {
			render.Error(rw, err)
			return
		}

		selectors := []coretypes.Selector{
			coretypes.TypeRole.MustSelector(authtypes.SigNozAdminRoleName),
		}

		err = middleware.authzService.CheckWithTupleCreation(
			ctx,
			claims,
			valuer.MustNewUUID(claims.OrgID),
			authtypes.Relation{Verb: coretypes.VerbAssignee},
			coretypes.NewResourceRole(),
			selectors,
			selectors,
		)
		if err != nil {
			middleware.logger.WarnContext(ctx, authzDeniedMessage, slog.Any("claims", claims))
			if errors.Asc(err, authtypes.ErrCodeAuthZForbidden) {
				render.Error(rw, errors.New(errors.TypeForbidden, authtypes.ErrCodeAuthZForbidden, "only admins can access this resource"))
				return
			}

			render.Error(rw, err)
			return
		}

		next(rw, req)
	})
}

func (middleware *AuthZ) OpenAccess(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		next(rw, req)
	})
}

// CheckResources authorizes every resolved resource for the route. roles are the
// allowed role names (the OSS role-gate); the resource selectors drive the EE check.
// ProjectAccess authorizes access to a project using direct project relationships and organization role inheritance.
// authorizeProjectAccess checks a direct project grant first, then falls back to
// the same verb on the owning organization. Only an authorization denial may
// trigger the fallback; service errors fail closed.
func authorizeProjectAccess(orgID, projectID valuer.UUID, verb coretypes.Verb, check func(coretypes.Resource, coretypes.Selector, coretypes.Verb) error) error {
	projectResource := coretypes.NewResourceMetaResource(coretypes.KindProject)
	projectSelector, err := projectResource.Type().Selector(projectID.String())
	if err != nil {
		return err
	}

	err = check(projectResource, projectSelector, verb)
	if err == nil || !errors.Asc(err, authtypes.ErrCodeAuthZForbidden) {
		return err
	}

	orgResource := coretypes.NewResourceOrganization()
	orgSelector, err := orgResource.Type().Selector(orgID.String())
	if err != nil {
		return err
	}
	// Organization resources expose read/update, not project lifecycle verbs.
	// Project attach/detach are delegated to organization update for org admins.
	organizationVerb := verb
	if verb == coretypes.VerbAttach || verb == coretypes.VerbDetach {
		organizationVerb = coretypes.VerbUpdate
	}
	return check(orgResource, orgSelector, organizationVerb)
}

func (middleware *AuthZ) ProjectAccess(next http.HandlerFunc, verb coretypes.Verb) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		claims, err := authtypes.ClaimsFromContext(ctx)
		if err != nil {
			render.Error(rw, err)
			return
		}

		orgID, err := valuer.NewUUID(claims.OrgID)
		if err != nil {
			render.Error(rw, err)
			return
		}

		projectID, err := valuer.NewUUID(mux.Vars(req)["id"])
		if err != nil {
			render.Error(rw, err)
			return
		}

		err = authorizeProjectAccess(orgID, projectID, verb, func(resource coretypes.Resource, selector coretypes.Selector, checkVerb coretypes.Verb) error {
			return middleware.authzService.CheckResource(ctx, claims, orgID, authtypes.Relation{Verb: checkVerb}, resource, selector)
		})
		if err != nil {
			render.Error(rw, err)
			return
		}

		// The scope is created only after the project permission check succeeds.
		// Never accept tenant identifiers from request headers or telemetry data.
		scopedCtx := ctxtypes.NewContextWithTenantScope(ctx, ctxtypes.TenantScope{
			OrgID: orgID.String(), ProjectID: projectID.String(),
		})
		next(rw, req.WithContext(scopedCtx))
	})
}

// ProjectHeaderAccess resolves an explicitly selected project for APIs whose
// routes do not contain a {id} path parameter (for example query-range APIs).
// The header is only a selector: the project is trusted only after the same
// project/org authorization check used by ProjectAccess succeeds.
// ProjectHeaderAccessIfPresent applies project authorization when a project
// selector is supplied. Requests without the selector retain the existing
// organization-wide behavior for backward compatibility with organization keys.
func (middleware *AuthZ) ProjectHeaderAccessIfPresent(next http.HandlerFunc, verb coretypes.Verb) http.HandlerFunc {
	projectAccess := middleware.ProjectHeaderAccess(next, verb)
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.TrimSpace(req.Header.Get("X-SigNoz-Project-ID")) == "" {
			next(rw, req)
			return
		}
		projectAccess.ServeHTTP(rw, req)
	})
}

func (middleware *AuthZ) ProjectHeaderAccess(next http.HandlerFunc, verb coretypes.Verb) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		claims, err := authtypes.ClaimsFromContext(req.Context())
		if err != nil {
			render.Error(rw, err)
			return
		}

		orgID, err := valuer.NewUUID(claims.OrgID)
		if err != nil {
			render.Error(rw, err)
			return
		}

		projectID, err := valuer.NewUUID(req.Header.Get("X-SigNoz-Project-ID"))
		if err != nil {
			render.Error(rw, errors.NewInvalidInputf(errors.CodeInvalidInput, "a valid X-SigNoz-Project-ID is required"))
			return
		}

		err = authorizeProjectAccess(orgID, projectID, verb, func(resource coretypes.Resource, selector coretypes.Selector, checkVerb coretypes.Verb) error {
			return middleware.authzService.CheckResource(req.Context(), claims, orgID, authtypes.Relation{Verb: checkVerb}, resource, selector)
		})
		if err != nil {
			render.Error(rw, err)
			return
		}

		ctx := ctxtypes.NewContextWithTenantScope(req.Context(), ctxtypes.TenantScope{
			OrgID: orgID.String(), ProjectID: projectID.String(),
		})
		next(rw, req.WithContext(ctx))
	})
}

func (middleware *AuthZ) CheckResources(next http.HandlerFunc, roles ...string) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		claims, err := authtypes.ClaimsFromContext(ctx)
		if err != nil {
			render.Error(rw, err)
			return
		}

		resolved, err := coretypes.ResolvedResourcesFromContext(ctx)
		if err != nil {
			render.Error(rw, err)
			return
		}

		orgID := valuer.MustNewUUID(claims.OrgID)

		roleSelectors := make([]coretypes.Selector, len(roles))
		for idx, role := range roles {
			roleSelectors[idx] = coretypes.TypeRole.MustSelector(role)
		}

		for _, resource := range resolved {
			if err := resource.Err(); err != nil {
				render.Error(rw, err)
				return
			}

			if err := middleware.checkResource(ctx, claims, orgID, resource.Verb(), resource.SourceResource(), resource.SourceIDs(), resource.SourceSelector(), roleSelectors); err != nil {
				render.Error(rw, err)
				return
			}

			target, ok := resource.(coretypes.ResolvedResourceWithTargetResource)
			if ok && !target.IsParentChild() {
				if err := middleware.checkResource(ctx, claims, orgID, target.Verb(), target.TargetResource(), target.TargetIDs(), target.TargetSelector(), roleSelectors); err != nil {
					render.Error(rw, err)
					return
				}
			}
		}

		next(rw, req)
	})
}

func (middleware *AuthZ) checkResource(
	ctx context.Context,
	claims authtypes.Claims,
	orgID valuer.UUID,
	verb coretypes.Verb,
	resource coretypes.Resource,
	ids []string,
	selector coretypes.SelectorFunc,
	roleSelectors []coretypes.Selector,
) error {
	if selector == nil {
		return errors.New(errors.TypeInternal, errors.CodeInternal, "resolved resource is missing a selector")
	}

	for _, id := range ids {
		selectors, err := selector(ctx, resource, id, orgID)
		if err != nil {
			return err
		}

		err = middleware.authzService.CheckWithTupleCreation(
			ctx,
			claims,
			orgID,
			authtypes.Relation{Verb: verb},
			resource,
			selectors,
			roleSelectors,
		)
		if err == nil {
			continue
		}

		if !errors.Asc(err, authtypes.ErrCodeAuthZForbidden) {
			return err
		}

		middleware.logger.WarnContext(ctx, authzDeniedMessage, slog.Any("claims", claims))
		principal := fmt.Sprintf("%s/%s", claims.Principal.StringValue(), claims.IdentityID())
		if id != "" {
			return errors.Newf(
				errors.TypeForbidden,
				authtypes.ErrCodeAuthZForbidden,
				"%s is not authorized to perform %s on resource %q",
				principal,
				resource.Scope(verb),
				id,
			)
		}

		return errors.Newf(
			errors.TypeForbidden,
			authtypes.ErrCodeAuthZForbidden,
			"%s is not authorized to perform %s",
			principal,
			resource.Scope(verb),
		)
	}

	return nil
}

func (middleware *AuthZ) CheckWithoutClaims(next http.HandlerFunc, relation authtypes.Relation, typeable coretypes.Resource, cb selectorCallbackWithoutClaimsFn, roles []string) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		orgs, err := middleware.orgGetter.ListByOwnedKeyRange(ctx)
		if err != nil {
			render.Error(rw, err)
			return
		}

		selectors, orgId, err := cb(req, orgs)
		if err != nil {
			render.Error(rw, err)
			return
		}

		roleSelectors := []coretypes.Selector{}
		for _, role := range roles {
			roleSelectors = append(roleSelectors, coretypes.TypeRole.MustSelector(role))
		}

		err = middleware.authzService.CheckWithTupleCreationWithoutClaims(ctx, orgId, relation, typeable, selectors, roleSelectors)
		if err != nil {
			render.Error(rw, err)
			return
		}

		next(rw, req)
	})
}
