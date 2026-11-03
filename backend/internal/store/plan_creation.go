package store

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/jackc/pgx/v5"
)

type CreatePlanInput struct {
	Title    string    `json:"title"`
	Activity string    `json:"activity"`
	StartsAt time.Time `json:"starts_at"`
	Capacity int       `json:"capacity"`
	Location string    `json:"location"`
	GroupID  string    `json:"group_id,omitempty"`
}

func (s *Store) CreatePlanFor(ctx context.Context, actor, key string, in CreatePlanInput) (*plans.Plan, error) {
	if in.GroupID != "" && !ids.Valid(in.GroupID) {
		return nil, plans.ErrInvalid
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Location = strings.TrimSpace(in.Location)
	in.GroupID = strings.ToLower(in.GroupID)
	if utf8.RuneCountInString(in.Location) > 500 {
		return nil, plans.ErrInvalid
	}
	var result plans.Plan
	err := s.idempotent(ctx, actor, "plan-create", key, in, func(tx pgx.Tx) (any, error) {
		invited := []string{}
		if in.GroupID != "" {
			if _, err := lockedGroupOwner(ctx, tx, in.GroupID); err != nil {
				return nil, err
			}
			var member bool
			if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id=$1 AND user_id=$2)", in.GroupID, actor).Scan(&member); err != nil {
				return nil, err
			}
			if !member {
				return nil, plans.ErrForbidden
			}
			rows, err := tx.Query(ctx, "SELECT m.user_id::text FROM group_members m JOIN users u ON u.id=m.user_id WHERE m.group_id=$1 AND u.deleted_at IS NULL ORDER BY m.user_id", in.GroupID)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				var id string
				if err = rows.Scan(&id); err != nil {
					rows.Close()
					return nil, err
				}
				invited = append(invited, id)
			}
			rows.Close()
			if err = rows.Err(); err != nil {
				return nil, err
			}
		}
		p, err := plans.New(ids.New(), actor, in.Title, in.Activity, in.StartsAt, in.Capacity, time.Now())
		if err != nil {
			return nil, err
		}
		p.GroupID = in.GroupID
		p.Location = in.Location
		if err = lockInvitations(ctx, tx, actor); err != nil {
			return nil, err
		}
		recipients := []string{}
		for _, id := range invited {
			if id == actor {
				continue
			}
			if err = lockPair(ctx, tx, actor, id); err != nil {
				return nil, err
			}
			if err = canConnect(ctx, tx, actor, id); err == plans.ErrForbidden {
				continue
			} else if err != nil {
				return nil, err
			}
			p.Members[id] = "invited"
			recipients = append(recipients, id)
		}
		// The locks above may have waited across the requested start time.
		if !p.StartsAt.After(time.Now()) {
			return nil, plans.ErrInvalid
		}
		if err = recordInvitations(ctx, tx, actor, in.GroupID, recipients); err != nil {
			return nil, err
		}
		if err = insertPlan(ctx, tx, p); err != nil {
			return nil, err
		}
		return p, nil
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Store) InvitePlanMember(ctx context.Context, id, actor, target string) error {
	if !ids.Valid(target) {
		return plans.ErrInvalid
	}
	target = strings.ToLower(target)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	p, err := load(ctx, tx, id, actor, true)
	if err != nil {
		return err
	}
	if p.OwnerID != actor {
		return plans.ErrForbidden
	}
	if (p.Status != "inviting" && p.Status != "confirmed") || !p.StartsAt.After(time.Now()) {
		return plans.ErrClosed
	}
	if _, exists := p.Members[target]; exists {
		return nil
	}
	if err = lockInvitations(ctx, tx, actor); err != nil {
		return err
	}
	if err = lockPair(ctx, tx, actor, target); err != nil {
		return err
	}
	if err = canConnect(ctx, tx, actor, target); err != nil {
		return err
	}
	low, high := pair(actor, target)
	var accepted bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM friendships WHERE low_id=$1 AND high_id=$2 AND status='accepted')", low, high).Scan(&accepted); err != nil {
		return err
	}
	if !accepted {
		return plans.ErrForbidden
	}
	if !p.StartsAt.After(time.Now()) {
		return plans.ErrClosed
	}
	if err = recordInvitations(ctx, tx, actor, p.GroupID, []string{target}); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO plan_members(plan_id,user_id,response) VALUES($1,$2,'invited')", id, target); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM plan_member_revocations WHERE plan_id=$1 AND user_id=$2", id, target); err != nil {
		return err
	}
	p.Version++
	if _, err = tx.Exec(ctx, "UPDATE plans SET version=$2 WHERE id=$1", id, p.Version); err != nil {
		return err
	}
	if err = event(ctx, tx, id, actor, "plan.invited", p.Version, target); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) RevokePlanMember(ctx context.Context, id, actor, target string) error {
	if !ids.Valid(target) {
		return plans.ErrInvalid
	}
	target = strings.ToLower(target)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	p, err := load(ctx, tx, id, actor, true)
	if err != nil {
		return err
	}
	if p.OwnerID != actor {
		return plans.ErrForbidden
	}
	if target == actor {
		return plans.ErrConflict
	}
	if p.Status != "inviting" && p.Status != "confirmed" {
		return plans.ErrClosed
	}
	tag, err := tx.Exec(ctx, "DELETE FROM plan_members WHERE plan_id=$1 AND user_id=$2", id, target)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	if _, err = tx.Exec(ctx, "INSERT INTO plan_member_revocations(plan_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING", id, target); err != nil {
		return err
	}
	p.Version++
	if _, err = tx.Exec(ctx, "UPDATE plans SET version=$2 WHERE id=$1", id, p.Version); err != nil {
		return err
	}
	if err = event(ctx, tx, id, actor, "plan.invitation_revoked", p.Version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
