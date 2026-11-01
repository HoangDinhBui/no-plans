package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var ErrRateLimited = errors.New("rate_limited")

func (s *Store) CheckAPIRate(ctx context.Context, actor string) error {
	var count int
	err := s.db.QueryRow(ctx, `INSERT INTO api_rate_buckets(actor_id,bucket,requests) VALUES($1,date_trunc('minute',clock_timestamp()),1) ON CONFLICT(actor_id,bucket) DO UPDATE SET requests=api_rate_buckets.requests+1 WHERE api_rate_buckets.requests<300 RETURNING requests`, actor).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRateLimited
	}
	return err
}

// Lock order: entity row, actor invitation lock, then sorted relationship locks.
func lockInvitations(ctx context.Context, tx pgx.Tx, actor string) error {
	_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "invites:"+actor)
	return err
}

// recordInvitations is called with the actor invitation lock held, only for new
// invitations. Quota changes roll back with the domain mutation and outbox.
func recordInvitations(ctx context.Context, tx pgx.Tx, actor, group string, recipients []string) error {
	if len(recipients) == 0 {
		return nil
	}
	var used int
	if err := tx.QueryRow(ctx, "SELECT COALESCE(sum(recipient_count),0) FROM invitation_batches WHERE actor_id=$1 AND created_at>clock_timestamp()-interval '1 hour'", actor).Scan(&used); err != nil {
		return err
	}
	if used+len(recipients) > 200 {
		return ErrRateLimited
	}
	if group != "" {
		var batches int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM invitation_batches WHERE actor_id=$1 AND group_id=$2 AND created_at>clock_timestamp()-interval '1 hour'", actor, group).Scan(&batches); err != nil {
			return err
		}
		if batches >= 20 {
			return ErrRateLimited
		}
	}
	for _, recipient := range recipients {
		var count int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM invitation_batches b JOIN invitation_recipients r ON r.batch_id=b.id WHERE b.actor_id=$1 AND r.recipient_id=$2 AND b.created_at>clock_timestamp()-interval '1 day'", actor, recipient).Scan(&count); err != nil {
			return err
		}
		if count >= 10 {
			return ErrRateLimited
		}
	}
	var batch string
	if err := tx.QueryRow(ctx, "INSERT INTO invitation_batches(actor_id,group_id,recipient_count) VALUES($1,NULLIF($2,'')::uuid,$3) RETURNING id::text", actor, group, len(recipients)).Scan(&batch); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if _, err := tx.Exec(ctx, "INSERT INTO invitation_recipients(batch_id,recipient_id) VALUES($1,$2)", batch, recipient); err != nil {
			return err
		}
	}
	return nil
}
