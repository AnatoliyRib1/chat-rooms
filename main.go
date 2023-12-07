package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/cloudmachinery/chat-rooms/tree/AnatoliyRib1/main/internal/config"
	"github.com/cloudmachinery/chat-rooms/tree/AnatoliyRib1/main/internal/server"
	"golang.org/x/exp/slog"
)

const gracefulTimeout = 10 * time.Second

func main() {
	cfg, err := config.NewConfig()
	failOnError(err, "parse config")

	srv, err := server.New(cfg)
	failOnError(err, "create server")

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig

		ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
		defer cancel()

		if err = srv.Stop(ctx); err != nil {
			slog.Error("server shutdown", "error", err)
		}
	}()

	if err = srv.Start(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		slog.Error(msg, "error", err)
		os.Exit(1)
	}
}
