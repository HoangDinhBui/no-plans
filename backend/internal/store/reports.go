package store

import (
	"context"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func (s *Store) CreateReport(ctx context.Context, actor, target, planID, reason string) (string, error) {
	if !ids.Valid(target) || (planID != "" && !ids.Valid(planID)) {
		return "", plans.ErrInvalid
	}
	switch reason {
	case "harassment", "spam", "unsafe", "other":
	default:
		return "", plans.ErrInvalid
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "reports:"+actor); err != nil {
		return "", err
	}
	var limited bool
	if err = tx.QueryRow(ctx, "SELECT count(*) >= 10 FROM reports WHERE reporter_id=$1 AND created_at > now()-interval '1 day'", actor).Scan(&limited); err != nil {
		return "", err
	}
	if limited {
		return "", ErrRateLimited
	}
	id := ids.New()
	tag, err := tx.Exec(ctx, `
 INSERT INTO reports(id, reporter_id, target_id, plan_id, reason)
 SELECT $1, a.id, t.id, NULLIF($4,'')::uuid, $5
 FROM users a JOIN users t ON t.id=$3 AND t.id<>a.id
 WHERE a.id=$2 AND a.deleted_at IS NULL AND t.deleted_at IS NULL
 AND ($4='' OR (
  SELECT count(*)=2 FROM plan_members m
  WHERE m.plan_id=NULLIF($4,'')::uuid AND m.user_id IN (a.id,t.id)
 ))
 `, id, actor, target, planID, reason)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", plans.ErrForbidden
	}
	return id, tx.Commit(ctx)
}
