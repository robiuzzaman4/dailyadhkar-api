package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/robiuzzaman4/dailyadhkar-api/internal/domain/user"
	"github.com/robiuzzaman4/dailyadhkar-api/internal/infrastructure/config"
	"github.com/robiuzzaman4/dailyadhkar-api/internal/interfaces/http/middleware"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(cfg *config.Config, db *pgxpool.Pool, users user.Repository) (*Server, error) {
	mux := http.NewServeMux()

	if err := registerRoutes(mux, db, users); err != nil {
		return nil, err
	}

	return &Server{
		httpServer: &http.Server{
			Addr: fmt.Sprintf(":%s", cfg.ServerPort),
			Handler: middleware.RequireRequestID(
				middleware.LogRequests(
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
	}, nil
}

func (s *Server) Start() error {
	slog.Default().Info("http server started", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}
