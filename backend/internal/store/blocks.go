package store

import (
	"context"
	"strings"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func (s *Store) BlockUser(ctx context.Context, actor, target string) error {
	if !ids.Valid(target) || strings.EqualFold(actor, target) {
		return plans.ErrInvalid
	}
	target = strings.ToLower(target)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockPair(ctx, tx, actor, target); err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND deleted_at IS NULL)", target).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return plans.ErrForbidden
	}
	tag, err := tx.Exec(ctx, "INSERT INTO blocks(blocker_id,blocked_id) VALUES($1,$2) ON CONFLICT DO NOTHING", actor, target)
	if err != nil {
		return err
	}
	low, high := pair(actor, target)
	if _, err = tx.Exec(ctx, "DELETE FROM friendships WHERE low_id=$1 AND high_id=$2", low, high); err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		if err = audit(ctx, tx, actor, "user.blocked", target); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) UnblockUser(ctx context.Context, actor, target string) error {
	if !ids.Valid(target) || strings.EqualFold(actor, target) {
		return plans.ErrInvalid
	}
	target = strings.ToLower(target)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockPair(ctx, tx, actor, target); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, "DELETE FROM blocks WHERE blocker_id=$1 AND blocked_id=$2", actor, target)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		if err = audit(ctx, tx, actor, "user.unblocked", target); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ListBlocked(ctx context.Context, actor, cursor string, limit int) (Page[Profile], error) {
	page := Page[Profile]{Items: []Profile{}}
	if !validPage(cursor, limit) {
		return page, plans.ErrInvalid
	}
	rows, err := s.db.Query(ctx, `SELECT u.id::text,u.display_name,u.avatar_url FROM blocks b JOIN users u ON u.id=b.blocked_id WHERE b.blocker_id=$1 AND ($2='' OR b.blocked_id>NULLIF($2,'')::uuid) ORDER BY b.blocked_id LIMIT $3`, actor, strings.ToLower(cursor), limit+1)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var p Profile
		if err = rows.Scan(&p.ID, &p.DisplayName, &p.AvatarURL); err != nil {
			return page, err
		}
		page.Items = append(page.Items, p)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.NextCursor = page.Items[limit-1].ID
	}
	return page, nil
}
