package store

import (
	"context"
	"errors"
	"strings"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/jackc/pgx/v5"
)

type Friendship struct {
	ID          string `json:"id"`
	LowID       string `json:"low_id"`
	HighID      string `json:"high_id"`
	RequesterID string `json:"requester_id"`
	Status      string `json:"status"`
}

func pair(a, b string) (string, string) {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	if a > b {
		return b, a
	}
	return a, b
}
func lockPair(ctx context.Context, tx pgx.Tx, a, b string) error {
	low, high := pair(a, b)
	_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "pair:"+low+":"+high)
	return err
}

func canConnect(ctx context.Context, tx pgx.Tx, a, b string) error {
	var ok bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$2 AND deleted_at IS NULL) AND NOT EXISTS(SELECT 1 FROM blocks WHERE (blocker_id=$1 AND blocked_id=$2) OR (blocker_id=$2 AND blocked_id=$1))`, a, b).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return plans.ErrForbidden
	}
	return nil
}

func (s *Store) RequestFriend(ctx context.Context, actor, target, key string) (Friendship, error) {
	if !ids.Valid(target) || !ids.Valid(actor) || strings.EqualFold(actor, target) {
		return Friendship{}, plans.ErrInvalid
	}
	target = strings.ToLower(target)
	low, high := pair(actor, target)
	var result Friendship
	err := s.idempotent(ctx, actor, "friend-request", key, target, func(tx pgx.Tx) (any, error) {
		if err := lockInvitations(ctx, tx, actor); err != nil {
			return nil, err
		}
		if err := lockPair(ctx, tx, actor, target); err != nil {
			return nil, err
		}
		if err := canConnect(ctx, tx, actor, target); err != nil {
			return nil, err
		}
		var f Friendship
		err := tx.QueryRow(ctx, "SELECT id::text,low_id::text,high_id::text,requester_id::text,status FROM friendships WHERE low_id=$1 AND high_id=$2", low, high).Scan(&f.ID, &f.LowID, &f.HighID, &f.RequesterID, &f.Status)
		if err == nil {
			if f.Status == "pending" && f.RequesterID != actor {
				return nil, plans.ErrConflict
			}
			return f, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		f = Friendship{ID: ids.New(), LowID: low, HighID: high, RequesterID: actor, Status: "pending"}
		if _, err = tx.Exec(ctx, "INSERT INTO friendships(id,low_id,high_id,requester_id) VALUES($1,$2,$3,$4)", f.ID, low, high, actor); err != nil {
			return nil, err
		}
		if err = recordInvitations(ctx, tx, actor, "", []string{target}); err != nil {
			return nil, err
		}
		if err = socialEvent(ctx, tx, actor, "friend.requested", f.ID, []string{target}); err != nil {
			return nil, err
		}
		return f, nil
	}, &result)
	return result, err
}

func (s *Store) AcceptFriend(ctx context.Context, actor, id string) (Friendship, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Friendship{}, err
	}
	defer tx.Rollback(ctx)
	var f Friendship
	err = tx.QueryRow(ctx, "SELECT id::text,low_id::text,high_id::text,requester_id::text,status FROM friendships WHERE id=$1 AND (low_id=$2 OR high_id=$2)", id, actor).Scan(&f.ID, &f.LowID, &f.HighID, &f.RequesterID, &f.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, plans.ErrForbidden
	}
	if err != nil {
		return f, err
	}
	if f.RequesterID == actor {
		return f, plans.ErrForbidden
	}
	if err = lockPair(ctx, tx, f.LowID, f.HighID); err != nil {
		return f, err
	}
	target := f.LowID
	if target == actor {
		target = f.HighID
	}
	if err = canConnect(ctx, tx, actor, target); err != nil {
		return f, err
	}
	// Re-read after the pair lock so block/delete/accept cannot race this mutation.
	if err = tx.QueryRow(ctx, "SELECT status FROM friendships WHERE id=$1 FOR UPDATE", id).Scan(&f.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return f, plans.ErrForbidden
		}
		return f, err
	}
	if f.Status == "accepted" {
		return f, nil
	}
	if _, err = tx.Exec(ctx, "UPDATE friendships SET status='accepted' WHERE id=$1", id); err != nil {
		return f, err
	}
	if err = socialEvent(ctx, tx, actor, "friend.accepted", id, []string{f.RequesterID}); err != nil {
		return f, err
	}
	f.Status = "accepted"
	if err = tx.Commit(ctx); err != nil {
		return Friendship{}, err
	}
	return f, nil
}
