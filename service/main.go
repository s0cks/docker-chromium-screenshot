package main

import (
	"net/http"
	"os"
	"time"

	"charm.land/log/v2"
	"github.com/s0cks/docker-chromium-screenshot/service/internal/meta"
	"github.com/s0cks/docker-chromium-screenshot/service/internal/server"
)

// Set at build time: go build -ldflags "-X main.version=... -X main.commit=... -X main.branch=... -X main.buildTime=..."
var (
	version   = "dev"
	commit    = "unknown"
	branch    = "unknown"
	buildTime = "unknown"
)

func main() {
	meta.Version = env("APP_VERSION", version)
	meta.Commit = env("GIT_COMMIT", commit)
	meta.Branch = env("GIT_BRANCH", branch)
	meta.BuildTime = env("BUILD_TIME", buildTime)

	cfg := server.Config{
		Addr:     env("ADDR", ":8080"),
		Dist:     env("DIST_DIR", "./dist"),
		Cards:    env("CARDS_DIR", "./cards"),
		Redirect: os.Getenv("INDEX_REDIRECT"),
	}
	srv := server.New(cfg)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           logRequests(srv),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Info("listening....",
		"version", meta.Version,
		"commit", meta.Commit,
		"address", cfg.Addr,
	)
	httpSrv.ListenAndServe()
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
