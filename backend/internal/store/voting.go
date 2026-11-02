package store

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/jackc/pgx/v5"
)

type OptionInput struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	PlaceID string `json:"place_id,omitempty"`
}
type PlaceOption struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	PlanID  string `json:"plan_id"`
	PlaceID string `json:"place_id,omitempty"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

var ErrVenueRequired = errors.New("venue_required")

type OptionView struct {
	PlaceOption
	VoteCount int  `json:"vote_count"`
	MyVote    bool `json:"my_vote"`
}

func votingOpen(p *plans.Plan) bool { return p.Status == "inviting" && p.StartsAt.After(time.Now()) }
func optionLocation(name, address string) string {
	if address == "" {
		return name
	}
	return name + " · " + address
}

func (s *Store) AddOption(ctx context.Context, planID, actor, key string, in OptionInput) (PlaceOption, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Address = strings.TrimSpace(in.Address)
	in.PlaceID = strings.ToLower(in.PlaceID)
	if in.PlaceID != "" && (!ids.Valid(in.PlaceID) || in.Name != "" || in.Address != "") {
		return PlaceOption{}, plans.ErrInvalid
	}
	var result PlaceOption
	err := s.idempotent(ctx, actor, "option-create:"+planID, key, in, func(tx pgx.Tx) (any, error) {
		p, err := load(ctx, tx, planID, actor, true)
		if err != nil {
			return nil, err
		}
		if p.OwnerID != actor {
			return nil, plans.ErrForbidden
		}
		if !votingOpen(p) {
			return nil, plans.ErrClosed
		}
		name, address, kind := in.Name, in.Address, "place"
		if in.PlaceID != "" {
			if err = tx.QueryRow(ctx, "SELECT name,address,kind FROM places WHERE id=$1 AND active", in.PlaceID).Scan(&name, &address, &kind); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, plans.ErrInvalid
				}
				return nil, err
			}
		}
		if name == "" || utf8.RuneCountInString(name) > 120 || utf8.RuneCountInString(optionLocation(name, address)) > 500 {
			return nil, plans.ErrInvalid
		}
		var count int
		if err = tx.QueryRow(ctx, "SELECT count(*) FROM place_options WHERE plan_id=$1", planID).Scan(&count); err != nil {
			return nil, err
		}
		if count >= 20 {
			return nil, plans.ErrInvalid
		}
		if !votingOpen(p) {
			return nil, plans.ErrClosed
		}
		option := PlaceOption{ID: ids.New(), PlanID: planID, PlaceID: in.PlaceID, Name: name, Address: address, Kind: kind}
		if _, err = tx.Exec(ctx, "INSERT INTO place_options(id,plan_id,created_by,place_id,name,address,kind) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7)", option.ID, planID, actor, in.PlaceID, name, address, kind); err != nil {
			return nil, err
		}
		p.Version++
		if _, err = tx.Exec(ctx, "UPDATE plans SET version=$2 WHERE id=$1", planID, p.Version); err != nil {
			return nil, err
		}
		if err = event(ctx, tx, planID, actor, "plan.option_added", p.Version); err != nil {
			return nil, err
		}
		return option, nil
	}, &result)
	return result, err
}

func (s *Store) CastVote(ctx context.Context, planID, actor, optionID string) (bool, error) {
	if !ids.Valid(optionID) {
		return false, plans.ErrInvalid
	}
	optionID = strings.ToLower(optionID)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	p, err := load(ctx, tx, planID, actor, true)
	if err != nil {
		return false, err
	}
	if !votingOpen(p) {
		return false, plans.ErrClosed
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM place_options WHERE plan_id=$1 AND id=$2)", planID, optionID).Scan(&exists); err != nil {
		return false, err
	}
	if !exists {
		return false, plans.ErrInvalid
	}
	if !votingOpen(p) {
		return false, plans.ErrClosed
	}
	tag, err := tx.Exec(ctx, `INSERT INTO votes(plan_id,user_id,option_id) VALUES($1,$2,$3) ON CONFLICT(plan_id,user_id) DO UPDATE SET option_id=excluded.option_id,updated_at=clock_timestamp() WHERE votes.option_id<>excluded.option_id`, planID, actor, optionID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	p.Version++
	if _, err = tx.Exec(ctx, "UPDATE plans SET version=$2 WHERE id=$1", planID, p.Version); err != nil {
		return false, err
	}
	if err = event(ctx, tx, planID, actor, "plan.voted", p.Version); err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) SelectOption(ctx context.Context, planID, actor string, version int64, optionID string) error {
	if !ids.Valid(optionID) {
		return plans.ErrInvalid
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	p, err := load(ctx, tx, planID, actor, true)
	if err != nil {
		return err
	}
	if p.OwnerID != actor {
		return plans.ErrForbidden
	}
	if !votingOpen(p) {
		return plans.ErrClosed
	}
	if p.Version != version {
		return plans.ErrConflict
	}
	var name, address, kind string
	if err = tx.QueryRow(ctx, "SELECT name,address,kind FROM place_options WHERE plan_id=$1 AND id=$2", planID, optionID).Scan(&name, &address, &kind); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return plans.ErrInvalid
		}
		return err
	}
	if kind == "idea" {
		return ErrVenueRequired
	}
	if !votingOpen(p) {
		return plans.ErrClosed
	}
	if _, err = tx.Exec(ctx, "UPDATE plans SET selected_option_id=$2,location=$3,version=version+1 WHERE id=$1", planID, optionID, optionLocation(name, address)); err != nil {
		return err
	}
	if err = event(ctx, tx, planID, actor, "plan.selection", p.Version+1); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListOptions(ctx context.Context, planID, actor, cursor string, limit int) (Page[OptionView], error) {
	page := Page[OptionView]{Items: []OptionView{}}
	if !validPage(cursor, limit) {
		return page, plans.ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	if _, err = load(ctx, tx, planID, actor, false); err != nil {
		return page, err
	}
	rows, err := tx.Query(ctx, `SELECT o.id::text,o.plan_id::text,COALESCE(o.place_id::text,''),o.name,o.address,o.kind,(SELECT count(*) FROM votes v WHERE v.plan_id=o.plan_id AND v.option_id=o.id),EXISTS(SELECT 1 FROM votes v WHERE v.plan_id=o.plan_id AND v.option_id=o.id AND v.user_id=$2) FROM place_options o WHERE o.plan_id=$1 AND ($3='' OR o.id>NULLIF($3,'')::uuid) ORDER BY o.id LIMIT $4`, planID, actor, strings.ToLower(cursor), limit+1)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var option OptionView
		if err = rows.Scan(&option.ID, &option.PlanID, &option.PlaceID, &option.Name, &option.Address, &option.Kind, &option.VoteCount, &option.MyVote); err != nil {
			return page, err
		}
		page.Items = append(page.Items, option)
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
