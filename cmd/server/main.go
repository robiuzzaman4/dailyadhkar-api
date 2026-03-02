package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/robiuzzaman4/daily-durood-api/internal/application/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.New(ctx)
	if err != nil {
		panic(err)
	}

	go func() {
		<-ctx.Done()
		if err := app.Shutdown(context.Background()); err != nil {
			app.Logger.Error("graceful shutdown failed", "error", err)
		}
	}()

	err = app.Server.Start()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.Logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
