package signozapiserver

import (
	"net/http"

	"github.com/SigNoz/signoz/pkg/http/handler"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/coretypes"
	"github.com/gorilla/mux"
)

func (provider *provider) addProjectRoutes(router *mux.Router) error {
	if err := router.Handle("/api/v2/projects", handler.New(
		provider.authzMiddleware.CheckResources(provider.projectHandler.Create, authtypes.SigNozAdminRoleName),
		handler.OpenAPIDef{
			ID: "CreateProject", Tags: []string{"projects"}, Summary: "Create project",
			Description: "Creates a project in the authenticated organization",
			Request: new(types.Project), RequestContentType: "application/json",
			Response: new(types.Project), ResponseContentType: "application/json",
			SuccessStatusCode: http.StatusCreated, ErrorStatusCodes: []int{http.StatusBadRequest, http.StatusConflict},
			SecuritySchemes: newScopedSecuritySchemes([]string{coretypes.ResourceOrganization.Scope(coretypes.VerbUpdate)}),
		},
		handler.WithResourceDefs(handler.BasicResourceDef{
			Resource: coretypes.ResourceOrganization, Verb: coretypes.VerbUpdate,
			Category: coretypes.ActionCategoryConfigurationChange, ID: provider.orgIDExtractor(), Selector: coretypes.IDSelector,
		}),
	)).Methods(http.MethodPost).GetError(); err != nil { return err }

	if err := router.Handle("/api/v2/projects", handler.New(
		provider.authzMiddleware.CheckResources(provider.projectHandler.List, authtypes.SigNozAdminRoleName),
		handler.OpenAPIDef{
			ID: "ListProjects", Tags: []string{"projects"}, Summary: "List projects",
			Description: "Lists projects belonging to the authenticated organization",
			Response: make([]*types.Project, 0), ResponseContentType: "application/json",
			SuccessStatusCode: http.StatusOK, SecuritySchemes: newScopedSecuritySchemes([]string{coretypes.ResourceOrganization.Scope(coretypes.VerbRead)}),
		},
		handler.WithResourceDefs(handler.BasicResourceDef{
			Resource: coretypes.ResourceOrganization, Verb: coretypes.VerbRead,
			Category: coretypes.ActionCategoryDataAccess, ID: provider.orgIDExtractor(), Selector: coretypes.IDSelector,
		}),
	)).Methods(http.MethodGet).GetError(); err != nil { return err }

	if err := router.Handle("/api/v2/projects/{id}", handler.New(
		provider.authzMiddleware.CheckResources(provider.projectHandler.Get, authtypes.SigNozAdminRoleName),
		handler.OpenAPIDef{
			ID: "GetProject", Tags: []string{"projects"}, Summary: "Get project",
			Description: "Gets a project within the authenticated organization",
			Response: new(types.Project), ResponseContentType: "application/json",
			SuccessStatusCode: http.StatusOK, ErrorStatusCodes: []int{http.StatusNotFound},
			SecuritySchemes: newScopedSecuritySchemes([]string{coretypes.ResourceOrganization.Scope(coretypes.VerbRead)}),
		},
		handler.WithResourceDefs(handler.BasicResourceDef{
			Resource: coretypes.ResourceOrganization, Verb: coretypes.VerbRead,
			Category: coretypes.ActionCategoryDataAccess, ID: provider.orgIDExtractor(), Selector: coretypes.IDSelector,
		}),
	)).Methods(http.MethodGet).GetError(); err != nil { return err }

	if err := router.Handle("/api/v2/projects/{id}", handler.New(
		provider.authzMiddleware.CheckResources(provider.projectHandler.Update, authtypes.SigNozAdminRoleName),
		handler.OpenAPIDef{
			ID: "UpdateProject", Tags: []string{"projects"}, Summary: "Update project",
			Description: "Updates a project within the authenticated organization",
			Request: new(types.Project), RequestContentType: "application/json",
			Response: new(types.Project), ResponseContentType: "application/json",
			SuccessStatusCode: http.StatusOK, ErrorStatusCodes: []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
			SecuritySchemes: newScopedSecuritySchemes([]string{coretypes.ResourceOrganization.Scope(coretypes.VerbUpdate)}),
		},
		handler.WithResourceDefs(handler.BasicResourceDef{
			Resource: coretypes.ResourceOrganization, Verb: coretypes.VerbUpdate,
			Category: coretypes.ActionCategoryConfigurationChange, ID: provider.orgIDExtractor(), Selector: coretypes.IDSelector,
		}),
	)).Methods(http.MethodPut).GetError(); err != nil { return err }

	if err := router.Handle("/api/v2/projects/{id}", handler.New(
		provider.authzMiddleware.CheckResources(provider.projectHandler.Delete, authtypes.SigNozAdminRoleName),
		handler.OpenAPIDef{
			ID: "ArchiveProject", Tags: []string{"projects"}, Summary: "Archive project",
			Description: "Archives a project within the authenticated organization",
			SuccessStatusCode: http.StatusNoContent, ErrorStatusCodes: []int{http.StatusNotFound},
			SecuritySchemes: newScopedSecuritySchemes([]string{coretypes.ResourceOrganization.Scope(coretypes.VerbUpdate)}),
		},
		handler.WithResourceDefs(handler.BasicResourceDef{
			Resource: coretypes.ResourceOrganization, Verb: coretypes.VerbUpdate,
			Category: coretypes.ActionCategoryConfigurationChange, ID: provider.orgIDExtractor(), Selector: coretypes.IDSelector,
		}),
	)).Methods(http.MethodDelete).GetError(); err != nil { return err }


	if err := router.Handle("/api/v2/projects/{id}/members", handler.New(
		provider.authzMiddleware.CheckResources(provider.projectHandler.ListMembers, authtypes.SigNozAdminRoleName),
		handler.OpenAPIDef{
			ID: "ListProjectMembers", Tags: []string{"projects"}, Summary: "List project members",
			Description: "Lists users and service accounts assigned to a project",
			Response: make([]*types.ProjectMember, 0), ResponseContentType: "application/json",
			SuccessStatusCode: http.StatusOK,
			SecuritySchemes: newScopedSecuritySchemes([]string{coretypes.ResourceOrganization.Scope(coretypes.VerbRead)}),
		},
		handler.WithResourceDefs(handler.BasicResourceDef{
			Resource: coretypes.ResourceOrganization, Verb: coretypes.VerbRead,
			Category: coretypes.ActionCategoryDataAccess, ID: provider.orgIDExtractor(), Selector: coretypes.IDSelector,
		}),
	)).Methods(http.MethodGet).GetError(); err != nil { return err }

	if err := router.Handle("/api/v2/projects/{id}/members", handler.New(
		provider.authzMiddleware.CheckResources(provider.projectHandler.AddMember, authtypes.SigNozAdminRoleName),
		handler.OpenAPIDef{
			ID: "AddProjectMember", Tags: []string{"projects"}, Summary: "Add project member",
			Description: "Assigns a user or service account a project role",
			Request: new(types.ProjectMember), RequestContentType: "application/json",
			SuccessStatusCode: http.StatusNoContent,
			SecuritySchemes: newScopedSecuritySchemes([]string{coretypes.ResourceOrganization.Scope(coretypes.VerbUpdate)}),
		},
		handler.WithResourceDefs(handler.BasicResourceDef{
			Resource: coretypes.ResourceOrganization, Verb: coretypes.VerbUpdate,
			Category: coretypes.ActionCategoryConfigurationChange, ID: provider.orgIDExtractor(), Selector: coretypes.IDSelector,
		}),
	)).Methods(http.MethodPost).GetError(); err != nil { return err }

	if err := router.Handle("/api/v2/projects/{id}/members", handler.New(
		provider.authzMiddleware.CheckResources(provider.projectHandler.RemoveMember, authtypes.SigNozAdminRoleName),
		handler.OpenAPIDef{
			ID: "RemoveProjectMember", Tags: []string{"projects"}, Summary: "Remove project member",
			Description: "Removes a user or service account from a project",
			Request: new(types.ProjectMember), RequestContentType: "application/json",
			SuccessStatusCode: http.StatusNoContent,
			SecuritySchemes: newScopedSecuritySchemes([]string{coretypes.ResourceOrganization.Scope(coretypes.VerbUpdate)}),
		},
		handler.WithResourceDefs(handler.BasicResourceDef{
			Resource: coretypes.ResourceOrganization, Verb: coretypes.VerbUpdate,
			Category: coretypes.ActionCategoryConfigurationChange, ID: provider.orgIDExtractor(), Selector: coretypes.IDSelector,
		}),
	)).Methods(http.MethodDelete).GetError(); err != nil { return err }

	return nil
}
