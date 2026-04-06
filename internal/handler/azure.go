package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const armBaseURL = "https://management.azure.com"

// HandleListSubscriptions serves GET /api/subscriptions
// It proxies the call to ARM using the user's Bearer token.
func HandleListSubscriptions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		url := armBaseURL + "/subscriptions?api-version=2022-12-01"
		body, status, err := armGet(r.Context(), url, token)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
}

// HandleListResourceGroupsserves GET /api/resource-groups?subscriptionId=...
func HandleListResourceGroups() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		subID := r.URL.Query().Get("subscriptionId")
		if subID == "" {
			writeError(w, http.StatusBadRequest, "subscriptionId query parameter required")
			return
		}

		url := fmt.Sprintf("%s/subscriptions/%s/resourcegroups?api-version=2022-09-01", armBaseURL, subID)
		body, status, err := armGet(r.Context(), url, token)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
}

// armGetperforms a GET request to the Azure ARM API using the provided token.
func armGet(ctx context.Context, url, token string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build ARM request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("ARM request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read ARM response: %w", err)
	}

	return body, resp.StatusCode, nil
}
