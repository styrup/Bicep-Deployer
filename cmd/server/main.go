package main

import (
	"io/fs"
	"log"
	"net/http"
	"strings"
	"text/template"

	bicepdeployer "github.com/user/bicep-deployer"
	"github.com/user/bicep-deployer/internal/config"
	"github.com/user/bicep-deployer/internal/handler"
	"github.com/user/bicep-deployer/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	blobClient, err := storage.New(cfg.StorageAccountName, cfg.StorageContainerName, cfg.StorageConnectionString)
	if err != nil {
		log.Fatalf("storage client error: %v", err)
	}

	mux := http.NewServeMux()

	// API routes
	mux.Handle("/api/templates", handler.HandleListTemplates(blobClient))
	mux.Handle("/api/templates/", handler.HandleGetTemplate(blobClient))
	mux.Handle("/api/subscriptions", handler.HandleListSubscriptions())
	mux.Handle("/api/resource-groups", handler.HandleListResourceGroups())
	mux.Handle("/api/deploy", handler.HandleDeploy(blobClient))
	mux.Handle("/api/deploy/status", handler.HandleDeployStatus())

	// Serve SPA — inject Azure config into index.html
	subFS, err := fs.Sub(bicepdeployer.WebFS, "web")
	if err != nil {
		log.Fatalf("embed web FS error: %v", err)
	}
	mux.Handle("/", spaHandler(subFS, cfg))

	addr := ":" + cfg.Port
	log.Printf("bicep-deployer listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// spaHandler serves the embedded web/ directory.
// index.html is rendered as a Go template so Azure config can be injected.
func spaHandler(fsys fs.FS, cfg *config.Config) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for the root, injecting config
		if r.URL.Path == "/" || r.URL.Path == "/index.html" || !fileExists(fsys, strings.TrimPrefix(r.URL.Path, "/")) {
			serveIndex(w, r, fsys, cfg)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, fsys fs.FS, cfg *config.Config) {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("index").Parse(string(data))
	if err != nil {
		http.Error(w, "template parse error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, map[string]string{
		"TenantID": cfg.AzureTenantID,
		"ClientID": cfg.AzureClientID,
	})
}

func fileExists(fsys fs.FS, path string) bool {
	if path == "" {
		return false
	}
	f, err := fsys.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}
