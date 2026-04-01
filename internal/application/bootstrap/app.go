package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/robiuzzaman4/dailyadhkar-api/internal/application/reminder"
	"github.com/robiuzzaman4/dailyadhkar-api/internal/infrastructure/config"
	"github.com/robiuzzaman4/dailyadhkar-api/internal/infrastructure/database"
	"github.com/robiuzzaman4/dailyadhkar-api/internal/infrastructure/email/unosend"
	postgresrepo "github.com/robiuzzaman4/dailyadhkar-api/internal/infrastructure/repository/postgres"
	httpserver "github.com/robiuzzaman4/dailyadhkar-api/internal/rest/http"
	"github.com/robiuzzaman4/dailyadhkar-api/internal/rest/http/middleware"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	Config    *config.Config
	Logger    *slog.Logger
	DB        *pgxpool.Pool
	Server    *httpserver.Server
	Scheduler *reminder.Scheduler
}

func New(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	appLogger := middleware.NewLogger(cfg.AppEnv)
	slog.SetDefault(appLogger)

	db, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	userRepository := postgresrepo.NewUserRepository(db)
	emailClient := unosend.NewClient(cfg.UnosendAPIKey, cfg.UnosendBaseURL)
	emailService := reminder.NewEmailService(emailClient, cfg.DefaultEmailSender, cfg.CompanyName, cfg.FrontendBaseURL)
	dispatcher := reminder.NewDispatcher(userRepository, emailService, cfg.EmailSendLimit, appLogger)
	scheduler, err := reminder.NewScheduler(appLogger, cfg.EmailSendTime, dispatcher)
	if err != nil {
		return nil, fmt.Errorf("initialize reminder scheduler: %w", err)
	}

	server, err := httpserver.NewServer(cfg, db, userRepository)
	if err != nil {
		return nil, fmt.Errorf("initialize http server: %w", err)
	}
	scheduler.Start()

	return &App{
		Config:    cfg,
		Logger:    appLogger,
		DB:        db,
		Server:    server,
		Scheduler: scheduler,
	}, nil
}

func (a *App) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	if err := a.Scheduler.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown scheduler: %w", err)
	}
	if err := a.Server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	a.DB.Close()
	a.Logger.Info("database connection closed")

	return nil
}
