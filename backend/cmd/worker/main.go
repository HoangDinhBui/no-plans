package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/push"
	"github.com/HoangDinhBui/no-plans/backend/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := store.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("could not open worker database")
	}
	defer db.Close()

	expo := push.NewExpo(os.Getenv("EXPO_ACCESS_TOKEN"))
	for ctx.Err() == nil {
		worked, err := push.ProcessNext(ctx, db, expo)
		if err != nil && ctx.Err() == nil {
			log.Print("push worker iteration failed")
		}
		if worked && err == nil {
			continue
		}
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
		}
	}
}
