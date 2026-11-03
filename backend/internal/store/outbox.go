package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type OutboxEvent struct {
	ID, PlanID, ActorID, Kind string
	Payload                   json.RawMessage
	Attempt                   int
}

// ClaimOutbox leases ready events. The incremented attempt fences acknowledgments
// from workers whose lease expired before they finished.
func (s *Store) ClaimOutbox(ctx context.Context, limit, maxAttempts int, lease time.Duration) ([]OutboxEvent, error) {
	if limit < 1 || limit > 100 || maxAttempts < 1 || lease < time.Second {
		return nil, errors.New("invalid outbox claim configuration")
	}
	rows, err := s.db.Query(ctx, `
		WITH candidates AS (
			SELECT id FROM outbox
			WHERE processed_at IS NULL AND dead_letter_at IS NULL
			  AND available_at <= now() AND (lease_until IS NULL OR lease_until <= now())
			ORDER BY available_at, id LIMIT $1 FOR UPDATE SKIP LOCKED
		), claimed AS (
			UPDATE outbox o
			SET dead_letter_at = CASE WHEN attempts >= $2 THEN now() END,
			    lease_until = CASE WHEN attempts < $2 THEN now() + $3::bigint * interval '1 millisecond' END,
			    attempts = CASE WHEN attempts < $2 THEN attempts + 1 ELSE attempts END
			FROM candidates c WHERE o.id = c.id
			RETURNING o.id::text, COALESCE(o.plan_id::text,''), o.actor_id::text, o.kind, o.payload, o.attempts, o.dead_letter_at
		)
		SELECT id, coalesce, actor_id, kind, payload, attempts FROM claimed WHERE dead_letter_at IS NULL`, limit, maxAttempts, lease.Milliseconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []OutboxEvent
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(&event.ID, &event.PlanID, &event.ActorID, &event.Kind, &event.Payload, &event.Attempt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// ResolveOutbox completes a live lease or releases it for a delayed retry.
func (s *Store) ResolveOutbox(ctx context.Context, event OutboxEvent, retryAfter *time.Duration) (bool, error) {
	var retryMillis *int64
	if retryAfter != nil {
		if *retryAfter < 0 {
			return false, errors.New("negative outbox retry delay")
		}
		value := retryAfter.Milliseconds()
		retryMillis = &value
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE outbox
		SET processed_at = CASE WHEN $3::bigint IS NULL THEN now() END,
		    available_at = COALESCE(now() + $3::bigint * interval '1 millisecond', available_at),
		    lease_until = NULL
		WHERE id = $1 AND attempts = $2 AND lease_until > now()
		  AND processed_at IS NULL AND dead_letter_at IS NULL`, event.ID, event.Attempt, retryMillis)
	return err == nil && tag.RowsAffected() == 1, err
}
