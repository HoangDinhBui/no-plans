package store

import (
	"context"
	"strings"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/jackc/pgx/v5"
)

type PlanParticipant struct {
	Profile
	Response string `json:"response"`
}

func (s *Store) ListPlanMembers(ctx context.Context, id, actor, cursor string, limit int) (Page[PlanParticipant], error) {
	page := Page[PlanParticipant]{Items: []PlanParticipant{}}
	if !validPage(cursor, limit) {
		return page, plans.ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	var member bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM plan_members WHERE plan_id=$1 AND user_id=$2)", id, actor).Scan(&member); err != nil {
		return page, err
	}
	if !member {
		return page, plans.ErrForbidden
	}
	rows, err := tx.Query(ctx, `SELECT u.id::text,CASE WHEN u.deleted_at IS NULL THEN u.display_name ELSE 'Tài khoản đã xóa' END,CASE WHEN u.deleted_at IS NULL THEN u.avatar_url ELSE '' END,m.response FROM plan_members m JOIN users u ON u.id=m.user_id WHERE m.plan_id=$1 AND ($2='' OR m.user_id>NULLIF($2,'')::uuid) ORDER BY m.user_id LIMIT $3`, id, strings.ToLower(cursor), limit+1)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var p PlanParticipant
		if err = rows.Scan(&p.ID, &p.DisplayName, &p.AvatarURL, &p.Response); err != nil {
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
	return page, tx.Commit(ctx)
}
