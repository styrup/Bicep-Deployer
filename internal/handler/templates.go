package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/user/bicep-deployer/internal/bicep"
)

// TemplateStore is the interface the templates handler requires from storage.
type TemplateStore interface {
	ListTemplates(ctx context.Context) ([]string, error)
	DownloadTemplate(ctx context.Context, name string) (string, error)
}

type templateListResponse struct {
	Templates []string `json:"templates"`
}

type templateDetailResponse struct {
	Name       string            `json:"name"`
	Parameters []bicep.Parameter `json:"parameters"`
}

// HandleListTemplates serves GET /api/templates
func HandleListTemplates(store TemplateStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		names, err := store.ListTemplates(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list templates: "+err.Error())
			return
		}
		if names == nil {
			names = []string{}
		}
		writeJSON(w, http.StatusOK, templateListResponse{Templates: names})
	}
}

// HandleGetTemplate serves GET /api/templates/{name}
// It downloads the template and parses its parameters.
func HandleGetTemplate(store TemplateStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract {name} from path: /api/templates/<name>
		name := strings.TrimPrefix(r.URL.Path, "/api/templates/")
		if name == "" {
			writeError(w, http.StatusBadRequest, "template name is required")
			return
		}

		content, err := store.DownloadTemplate(r.Context(), name)
		if err != nil {
			writeError(w, http.StatusNotFound, "template not found: "+err.Error())
			return
		}

		params := bicep.ParseParameters(content)
		if params == nil {
			params = []bicep.Parameter{}
		}

		writeJSON(w, http.StatusOK, templateDetailResponse{
			Name:       name,
			Parameters: params,
		})
	}
}
