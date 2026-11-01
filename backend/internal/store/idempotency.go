package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/jackc/pgx/v5"
)

var ErrIdempotencyConflict = errors.New("idempotency_conflict")

// idempotent commits the entity, audit/outbox and replay payload in one transaction.
// A per-actor/route/key advisory lock also serializes the absent-row case.
func (s *Store) idempotent(ctx context.Context, actor, route, key string, input any, create func(pgx.Tx) (any, error), output any) error {
	if key == "" || len(key) > 128 {
		return plans.ErrInvalid
	}
	for _, c := range key {
		if c < 33 || c > 126 {
			return plans.ErrInvalid
		}
	}
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", actor+"|"+route+"|"+key); err != nil {
		return err
	}
	var previousHash, body []byte
	err = tx.QueryRow(ctx, "SELECT request_hash,response FROM idempotency_keys WHERE actor_id=$1 AND route=$2 AND key=$3 AND expires_at>clock_timestamp()", actor, route, key).Scan(&previousHash, &body)
	if err == nil {
		if !bytes.Equal(previousHash, hash[:]) {
			return ErrIdempotencyConflict
		}
		return json.Unmarshal(body, output)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	result, err := create(tx)
	if err != nil {
		return err
	}
	body, err = json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO idempotency_keys(actor_id,route,key,request_hash,status_code,response,expires_at) VALUES($1,$2,$3,$4,201,$5,clock_timestamp()+interval '24 hours') ON CONFLICT(actor_id,route,key) DO UPDATE SET request_hash=excluded.request_hash,status_code=excluded.status_code,response=excluded.response,created_at=clock_timestamp(),expires_at=excluded.expires_at`, actor, route, key, hash[:], body)
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return json.Unmarshal(body, output)
}

func audit(ctx context.Context, tx pgx.Tx, actor, kind, entity string) error {
	_, err := tx.Exec(ctx, "INSERT INTO audit_events(actor_id,kind,entity_id) VALUES($1,$2,$3)", actor, kind, entity)
	return err
}

func socialEvent(ctx context.Context, tx pgx.Tx, actor, kind, entity string, recipients []string) error {
	if err := audit(ctx, tx, actor, kind, entity); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{"entity_id": entity, "recipient_ids": recipients})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO outbox(actor_id,kind,payload) VALUES($1,$2,$3)", actor, kind, payload)
	return err
}
