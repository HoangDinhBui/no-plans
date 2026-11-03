package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/auth"
	"github.com/HoangDinhBui/no-plans/backend/internal/push"
	"github.com/HoangDinhBui/no-plans/backend/internal/store"
	"github.com/jackc/pgx/v5"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := store.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("could not open worker database")
	}
	defer db.Close()

	var admin *auth.IdentityAdmin
	projectURL, key := os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if projectURL != "" || key != "" {
		admin, err = auth.NewIdentityAdmin(projectURL, key)
		if err != nil {
			log.Fatal(err)
		}
	}
	expo := push.NewExpo(os.Getenv("EXPO_ACCESS_TOKEN"))
	for ctx.Err() == nil {
		if admin != nil {
			if err := processIdentityDeletion(ctx, db, admin); err != nil && ctx.Err() == nil {
				log.Printf("identity deletion worker: %v", err)
			}
		}
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

func processIdentityDeletion(ctx context.Context, db *store.Store, admin *auth.IdentityAdmin) error {
	job, err := db.ClaimIdentityDeletion(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return errors.New("could not claim identity deletion")
	}
	if err := admin.DeleteUser(ctx, job.UserID); err != nil {
		return err
	}
	if err := db.CompleteIdentityDeletion(ctx, job); err != nil {
		return errors.New("could not complete identity deletion")
	}
	return nil
}
