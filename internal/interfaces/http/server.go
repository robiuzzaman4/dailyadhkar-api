package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/robiuzzaman4/dailyadhkar-api/internal/domain/user"
	"github.com/robiuzzaman4/dailyadhkar-api/internal/infrastructure/auth/clerk"
	"github.com/robiuzzaman4/dailyadhkar-api/internal/infrastructure/config"
	"github.com/robiuzzaman4/dailyadhkar-api/internal/interfaces/http/middleware"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func NewServer(cfg *config.Config, logger *slog.Logger, db *pgxpool.Pool, users user.Repository) (*Server, error) {
	mux := http.NewServeMux()

	if err := registerRoutes(mux, cfg, logger, db, users); err != nil {
		return nil, err
	}

	return &Server{
		httpServer: &http.Server{
			Addr: fmt.Sprintf(":%s", cfg.ServerPort),
			Handler: middleware.RequireRequestID(
				middleware.LogRequests(
					logger,
					middleware.CORS(middleware.CORSConfig{
						AllowedOrigins:   cfg.CORSAllowedOrigins,
						AllowedMethods:   cfg.CORSAllowedMethods,
						AllowedHeaders:   cfg.CORSAllowedHeaders,
						AllowCredentials: cfg.CORSAllowCredentials,
					}, mux),
				),
			),
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger: logger,
	}, nil
}

func (s *Server) Start() error {
	s.logger.Info("http server started", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

type clerkTokenVerifierAdapter struct {
	verifier *clerk.TokenVerifier
}

func (a clerkTokenVerifierAdapter) Verify(ctx context.Context, token string) (string, error) {
	claims, err := a.verifier.Verify(ctx, token)
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", errors.New("missing token subject")
	}
	return claims.Subject, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}
