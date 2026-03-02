package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/robiuzzaman4/daily-durood-api/internal/infrastructure/config"
	"github.com/robiuzzaman4/daily-durood-api/internal/infrastructure/database"
	httpserver "github.com/robiuzzaman4/daily-durood-api/internal/interfaces/http"
	"github.com/robiuzzaman4/daily-durood-api/internal/shared/logger"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	Config *config.Config
	Logger *slog.Logger
	DB     *pgxpool.Pool
	Server *httpserver.Server
}

func New(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	appLogger := logger.New(cfg.AppEnv)

	db, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	server := httpserver.NewServer(cfg.ServerPort, appLogger, db)

	return &App{
		Config: cfg,
		Logger: appLogger,
		DB:     db,
		Server: server,
	}, nil
}

func (a *App) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	if err := a.Server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	a.DB.Close()
	a.Logger.Info("database connection closed")

	return nil
}
