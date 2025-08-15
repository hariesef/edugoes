package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	ltiHandler "github.com/quipper/poc/lti/be/internal/controller/http/lti"
	rosterSqlite "github.com/quipper/poc/lti/be/internal/repositories/roster/sqlite"
	scoresSqliteRepo "github.com/quipper/poc/lti/be/internal/repositories/scores/sqlite"
	sqliteRepo "github.com/quipper/poc/lti/be/internal/repositories/lti/sqlite"
	vsqliteRepo "github.com/quipper/poc/lti/be/internal/repositories/validation"
	"github.com/quipper/poc/lti/be/pkg/common/keys"
	"github.com/quipper/poc/lti/be/pkg/common/logger"
)

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "debug"
	}
	logger.Initialize(level)
	logger.Info("starting server")

	// Initialize signing keys early so that if we generate a dev key,
	// the PEM export instructions are printed immediately at startup.
	if err := keys.Init(); err != nil {
		logger.Error("init keys: %v", err)
		os.Exit(1)
	}

	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		dbPath = "./lti.db"
	}
	repo, err := sqliteRepo.NewSQLiteRepo(dbPath)
	if err != nil {
		logger.Error("init repo: %v", err)
		os.Exit(1)
	}

	vdbPath := os.Getenv("VALIDATION_SQLITE_PATH")
	if vdbPath == "" {
		vdbPath = "./validation.db"
	}
	vrepo, err := vsqliteRepo.NewSQLiteRepo(vdbPath)
	if err != nil {
		logger.Error("init validation repo: %v", err)
		os.Exit(1)
	}

	// Scores repository (can share same DB file or separate one)
	sdbPath := os.Getenv("SCORES_SQLITE_PATH")
	if sdbPath == "" {
		sdbPath = "./scores.db"
	}
	scoresRepo, err := scoresSqliteRepo.NewSQLiteRepo(sdbPath)
	if err != nil {
		logger.Error("init scores repo: %v", err)
		os.Exit(1)
	}

	// Roster repository (NRPS sandbox storage)
	rdbPath := os.Getenv("ROSTER_SQLITE_PATH")
	if rdbPath == "" {
		rdbPath = "./roster.db"
	}
	rosterRepo, err := rosterSqlite.NewSQLiteRepo(rdbPath)
	if err != nil {
		logger.Error("init roster repo: %v", err)
		os.Exit(1)
	}

	h := ltiHandler.NewHandler(repo, scoresRepo, vrepo, rosterRepo)
	router := chi.NewRouter()
	const maxBodySize = 2_100_000
	router.Use(middleware.RequestSize(maxBodySize))
	router.Use(middleware.Recoverer)

	// Prefer chi-based router for API
	router.Mount("/", h.Router())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	server := &http.Server{Addr: addr, Handler: withCORS(router)}

	go func() {
		logger.Info("listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown: %v", err)
	}
	if repo != nil {
		repo.Disconnect()
	}
	if vrepo != nil {
		vrepo.Disconnect()
	}
	if scoresRepo != nil {
		scoresRepo.Disconnect()
	}
	logger.Info("server stopped")
}
