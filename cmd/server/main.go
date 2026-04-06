package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	"golang.org/x/time/rate"

	bicepdeployer "github.com/user/bicep-deployer"
	"github.com/user/bicep-deployer/internal/config"
	"github.com/user/bicep-deployer/internal/handler"
	"github.com/user/bicep-deployer/internal/middleware"
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

	cachedStore := handler.NewCachedStore(blobClient, 2*time.Minute)

	mux := http.NewServeMux()

	// Health check for Container Apps / Kubernetes probes
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// API routes
	mux.Handle("/api/templates", handler.HandleListTemplates(cachedStore))
	mux.Handle("/api/templates/", handler.HandleGetTemplate(cachedStore))
	mux.Handle("/api/subscriptions", handler.HandleListSubscriptions())
	mux.Handle("/api/resource-groups", handler.HandleListResourceGroups())
	mux.Handle("/api/deploy", handler.HandleDeploy(cachedStore))
	mux.Handle("/api/deploy/status", handler.HandleDeployStatus())

	// Serve SPA — inject Azure config into index.html
	subFS, err := fs.Sub(bicepdeployer.WebFS, "web")
	if err != nil {
		log.Fatalf("embed web FS error: %v", err)
	}
	mux.Handle("/", spaHandler(subFS, cfg))

	// Apply middleware: security headers + rate limiting (20 req/s burst 40 per IP)
	wrapped := middleware.Chain(mux,
		middleware.SecurityHeaders,
		middleware.RateLimiter(rate.Limit(20), 40),
	)

	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      wrapped,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("bicep-deployer listening on http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down gracefully…")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	log.Println("server stopped")
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
		"AppTitle": cfg.AppTitle,
		"AppIcon":  cfg.AppIcon,
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
