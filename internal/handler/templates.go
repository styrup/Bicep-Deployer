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

// TemplateGroup groups templates by folder prefix.
type TemplateGroup struct {
	Name      string   `json:"name"`
	Templates []string `json:"templates"`
}

type templateListResponse struct {
	Groups []TemplateGroup `json:"groups"`
}

type templateDetailResponse struct {
	Name       string            `json:"name"`
	Parameters []bicep.Parameter `json:"parameters"`
}

// HandleListTemplates serves GET /api/templates
// Groups templates by virtual directory (e.g. "networking/vnet.bicep" → group "networking")
func HandleListTemplates(store TemplateStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		names, err := store.ListTemplates(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list templates: "+err.Error())
			return
		}

		groups := groupTemplates(names)
		writeJSON(w, http.StatusOK, templateListResponse{Groups: groups})
	}
}

// groupTemplates organizes flat blob paths into groups by their directory prefix.
// Files without a prefix go into a "General" group.
func groupTemplates(names []string) []TemplateGroup {
	groupMap := make(map[string][]string)
	var order []string

	for _, name := range names {
		group := "General"
		if idx := strings.LastIndex(name, "/"); idx != -1 {
			group = name[:idx]
		}
		if _, exists := groupMap[group]; !exists {
			order = append(order, group)
		}
		groupMap[group] = append(groupMap[group], name)
	}

	groups := make([]TemplateGroup, 0, len(order))
	for _, g := range order {
		groups = append(groups, TemplateGroup{Name: g, Templates: groupMap[g]})
	}
	return groups
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
