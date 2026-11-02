package store

import (
	"context"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"time"
)

func (s *Store) EditPlan(ctx context.Context, id, actor string, version int64, title string, startsAt time.Time, capacity int, location string) error {
	return s.changePlan(ctx, id, actor, "plan.updated", func(p *plans.Plan) error {
		return p.Edit(actor, version, title, startsAt, capacity, location, time.Now())
	})
}

func (s *Store) TransitionPlan(ctx context.Context, id, actor, action string) error {
	return s.changePlan(ctx, id, actor, "plan."+action, func(p *plans.Plan) error { return p.Transition(actor, action, time.Now()) })
}

func (s *Store) changePlan(ctx context.Context, id, actor, kind string, change func(*plans.Plan) error) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	p, err := load(ctx, tx, id, actor, true)
	if err != nil {
		return err
	}
	if err = change(p); err != nil {
		return err
	}
	if kind == "plan.confirm" && p.SelectedOptionID != "" {
		var optionKind string
		if err = tx.QueryRow(ctx, "SELECT kind FROM place_options WHERE plan_id=$1 AND id=$2", id, p.SelectedOptionID).Scan(&optionKind); err != nil {
			return err
		}
		if optionKind == "idea" {
			return ErrVenueRequired
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE plans SET title=$2,starts_at=$3,capacity=$4,location=$5,status=$6,version=$7,selected_option_id=NULLIF($8,'')::uuid WHERE id=$1`, id, p.Title, p.StartsAt, p.Capacity, p.Location, p.Status, p.Version, p.SelectedOptionID); err != nil {
		return err
	}
	if err = event(ctx, tx, id, actor, kind, p.Version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
