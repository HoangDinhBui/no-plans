package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/catalogue"
	"github.com/HoangDinhBui/no-plans/backend/internal/store"
)

func main() {
	file := flag.String("file", "", "Optional JSON catalogue; defaults to bundled curated catalogue")
	flag.Parse()
	if err := run(*file); err != nil {
		slog.Error("catalogue import failed", "error", err)
		os.Exit(1)
	}
}
func run(file string) error {
	places, err := catalogue.Initial()
	if err != nil {
		return err
	}
	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
		if err != nil {
			return err
		}
		if len(data) > 2<<20 {
			return fmt.Errorf("catalogue exceeds 2MiB")
		}
		if err = json.Unmarshal(data, &places); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	s, err := store.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer s.Close()
	if err = s.UpsertPlaces(ctx, places); err != nil {
		return err
	}
	slog.Info("catalogue imported", "entries", len(places))
	return nil
}
