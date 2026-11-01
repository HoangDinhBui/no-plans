package main

import (
	"context"
	"github.com/HoangDinhBui/no-plans/backend/internal/store"
	"log/slog"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}
func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	s, err := store.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer s.Close()
	if err = s.Migrate(ctx); err != nil {
		return err
	}
	slog.Info("database migrations applied")
	return nil
}
