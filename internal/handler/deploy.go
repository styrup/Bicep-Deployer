package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// DeployRequest is the JSON body for POST /api/deploy.
type DeployRequest struct {
	TemplateName      string            `json:"templateName"`
	Scope             string            `json:"scope"` // "resourceGroup" or "subscription"
	SubscriptionID    string            `json:"subscriptionId"`
	ResourceGroupName string            `json:"resourceGroupName"` // only for scope=resourceGroup
	DeploymentName    string            `json:"deploymentName"`
	Parameters        map[string]json.RawMessage `json:"parameters"`
}

// HandleDeploy serves POST /api/deploy
func HandleDeploy(store TemplateStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		token := extractBearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		var req DeployRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if err := validateDeployRequest(req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Download Bicep template from Blob Storage
		bicepContent, err := store.DownloadTemplate(r.Context(), req.TemplateName)
		if err != nil {
			writeError(w, http.StatusNotFound, "template not found: "+err.Error())
			return
		}

		// Compile Bicep → ARM JSON using bicep CLI.
		// Passes the store so that referenced modules can be downloaded too.
		armJSON, err := compileBicep(r.Context(), store, req.TemplateName, bicepContent)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "bicep compilation failed: "+err.Error())
			return
		}

		// Build ARM deployment payload
		payload, err := buildDeploymentPayload(armJSON, req.Parameters)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "build deployment payload: "+err.Error())
			return
		}

		// Determine ARM deployment URL
		deployURL := buildDeployURL(req)

		// Send PUT request to ARM
		result, status, err := armPut(r.Context(), deployURL, token, payload)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "ARM deployment failed: "+err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(result)
	}
}

// moduleRefRe matches Bicep module declarations that reference local files.
// Examples:
//
//	module budget './modules/budget.bicep' = {
//	module budget './modules/budget.bicep' = if (cond) {
var moduleRefRe = regexp.MustCompile(`(?m)^\s*module\s+\S+\s+'([^']+\.bicep)'\s*=`)

// compileBicep creates a temp directory, writes the main template (preserving
// its original filename), recursively downloads any locally-referenced modules,
// and invokes `bicep build --stdout`. The temp directory is removed afterwards.
func compileBicep(ctx context.Context, store TemplateStore, templateName, bicepContent string) (json.RawMessage, error) {
	tmpDir, err := os.MkdirTemp("", "bicep-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write the main template using its original blob path so that relative
	// module references resolve correctly.
	mainPath := filepath.Join(tmpDir, filepath.FromSlash(templateName))
	if err := writeFile(mainPath, bicepContent); err != nil {
		return nil, fmt.Errorf("write main template: %w", err)
	}

	// Recursively download referenced modules.
	if err := downloadModules(ctx, store, tmpDir, templateName, bicepContent); err != nil {
		return nil, fmt.Errorf("download modules: %w", err)
	}

	cmd := exec.CommandContext(ctx, "bicep", "build", mainPath, "--stdout")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("bicep build failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bicep build: %w — ensure 'bicep' CLI is installed and in PATH", err)
	}

	var raw json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("bicep output is not valid JSON: %w", err)
	}

	return raw, nil
}

// downloadModules parses module references from bicepContent, downloads each
// file from blob storage, writes it into tmpDir, and recurses for any modules
// those files reference. visited tracks already-processed blob paths.
func downloadModules(ctx context.Context, store TemplateStore, tmpDir, templateName, bicepContent string) error {
	visited := map[string]bool{templateName: true}
	return downloadModulesRec(ctx, store, tmpDir, templateName, bicepContent, visited)
}

func downloadModulesRec(ctx context.Context, store TemplateStore, tmpDir, templateName, bicepContent string, visited map[string]bool) error {
	templateDir := path.Dir(templateName) // blob paths use forward slashes

	for _, ref := range parseModuleRefs(bicepContent) {
		// Resolve the module's blob path relative to the referencing template.
		blobPath := path.Clean(path.Join(templateDir, ref))

		if visited[blobPath] {
			continue
		}
		visited[blobPath] = true

		content, err := store.DownloadTemplate(ctx, blobPath)
		if err != nil {
			return fmt.Errorf("download module %q: %w", blobPath, err)
		}

		filePath := filepath.Join(tmpDir, filepath.FromSlash(blobPath))
		if err := writeFile(filePath, content); err != nil {
			return fmt.Errorf("write module %q: %w", blobPath, err)
		}

		// Recurse to handle nested module references.
		if err := downloadModulesRec(ctx, store, tmpDir, blobPath, content, visited); err != nil {
			return err
		}
	}

	return nil
}

// parseModuleRefs extracts relative file paths from Bicep module declarations.
func parseModuleRefs(content string) []string {
	matches := moduleRefRe.FindAllStringSubmatch(content, -1)
	refs := make([]string, 0, len(matches))
	for _, m := range matches {
		ref := m[1]
		// Skip registry or template-spec references (e.g. "br:", "ts:").
		if strings.Contains(ref, ":") {
			continue
		}
		refs = append(refs, ref)
	}
	return refs
}

// writeFile creates parent directories and writes content to filePath.
func writeFile(filePath, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(content), 0o644)
}

// buildDeploymentPayload constructs the ARM deployment request body.
// Empty parameter values are omitted so ARM uses the template's defaults.
// Values are sent as-is (string, int, bool, object, array) from the JSON body.
func buildDeploymentPayload(template json.RawMessage, params map[string]json.RawMessage) ([]byte, error) {
	armParams := make(map[string]any, len(params))
	for k, v := range params {
		// Skip empty strings
		var str string
		if json.Unmarshal(v, &str) == nil && str == "" {
			continue
		}
		armParams[k] = map[string]any{"value": json.RawMessage(v)}
	}

	payload := map[string]any{
		"properties": map[string]any{
			"mode":       "Incremental",
			"template":   template,
			"parameters": armParams,
		},
	}

	return json.Marshal(payload)
}

// buildDeployURL returns the ARM REST API URL for the deployment.
func buildDeployURL(req DeployRequest) string {
	switch req.Scope {
	case "subscription":
		return fmt.Sprintf(
			"%s/subscriptions/%s/providers/Microsoft.Resources/deployments/%s?api-version=2022-09-01",
			armBaseURL, req.SubscriptionID, req.DeploymentName,
		)
	default: // resourceGroup
		return fmt.Sprintf(
			"%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s?api-version=2022-09-01",
			armBaseURL, req.SubscriptionID, req.ResourceGroupName, req.DeploymentName,
		)
	}
}

// armPut performs a PUT to the Azure ARM API with the provided token and body.
func armPut(ctx context.Context, url, token string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("build ARM PUT request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("ARM PUT request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read ARM response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

func validateDeployRequest(req DeployRequest) error {
	if req.TemplateName == "" {
		return fmt.Errorf("templateName is required")
	}
	if req.SubscriptionID == "" {
		return fmt.Errorf("subscriptionId is required")
	}
	if req.DeploymentName == "" {
		return fmt.Errorf("deploymentName is required")
	}
	if req.Scope == "resourceGroup" && req.ResourceGroupName == "" {
		return fmt.Errorf("resourceGroupName is required for scope=resourceGroup")
	}
	return nil
}

// HandleDeployStatus serves GET /api/deploy/status?url=<ARM deployment URL>
// It proxies the deployment status check using the user's Bearer token.
func HandleDeployStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		statusURL := r.URL.Query().Get("url")
		if statusURL == "" {
			writeError(w, http.StatusBadRequest, "url query parameter required")
			return
		}

		// Only allow ARM management URLs
		if !strings.HasPrefix(statusURL, armBaseURL) {
			writeError(w, http.StatusBadRequest, "url must be an ARM management URL")
			return
		}

		body, status, err := armGet(r.Context(), statusURL, token)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(body)
	}
}
