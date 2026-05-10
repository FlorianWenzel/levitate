package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/florianwenzel/levitate/backend/internal/api"
	"github.com/florianwenzel/levitate/backend/internal/auth"
	"github.com/florianwenzel/levitate/backend/internal/config"
	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fatal("config", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	deps := api.Deps{
		Logger:           logger,
		CORSOrigins:      cfg.CORSOrigins,
		AllowTestReset:   cfg.AllowTestReset,
		OIDCIssuerPublic: cfg.OIDCIssuer,
		OIDCClientID:     cfg.OIDCAudience,
	}
	if cfg.AllowTestReset {
		logger.Warn("test reset endpoint enabled — never run with this in production")
	}

	if cfg.DBURL != "" {
		if err := db.MigrateUp(cfg.DBURL); err != nil {
			fatal("migrate", err)
		}
		logger.Info("migrations applied")

		pool, err := pgxpool.New(context.Background(), cfg.DBURL)
		if err != nil {
			fatal("db pool", err)
		}
		defer pool.Close()
		deps.Pool = pool
	} else {
		logger.Warn("DB URL not configured; /api routes disabled")
	}

	if cfg.OIDCIssuer != "" {
		// In container deployments the IdP often boots a few seconds after we
		// do; retry discovery instead of giving up so /api/* always comes up.
		var v *auth.Verifier
		var lastErr error
		for attempt := 1; attempt <= 60; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			v, lastErr = auth.NewVerifier(ctx, auth.Options{
				Issuer:       cfg.OIDCIssuer,
				DiscoveryURL: cfg.OIDCDiscoveryURL,
				Audience:     cfg.OIDCAudience,
				RoleClaim:    cfg.OIDCRoleClaim,
			})
			cancel()
			if lastErr == nil {
				break
			}
			logger.Warn("oidc discovery not ready; retrying", "attempt", attempt, "err", lastErr)
			time.Sleep(2 * time.Second)
		}
		if v == nil {
			fatal("oidc init failed after retries", lastErr)
		}
		deps.Verifier = v
		logger.Info("oidc verifier ready", "issuer", cfg.OIDCIssuer)
	} else {
		logger.Warn("OIDC issuer not configured; /api routes disabled")
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewRouter(deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
}

func fatal(msg string, err error) {
	slog.Error(msg, "err", err)
	os.Exit(1)
}
