package store

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/jackc/pgx/v5"
)

type Group struct {
	ID      string    `json:"id"`
	OwnerID string    `json:"owner_id"`
	Name    string    `json:"name"`
	Members []Profile `json:"members"`
}

func (s *Store) CreateGroup(ctx context.Context, actor, name, key string) (Group, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return Group{}, plans.ErrInvalid
	}
	var result Group
	err := s.idempotent(ctx, actor, "group-create", key, name, func(tx pgx.Tx) (any, error) {
		id := ids.New()
		if _, err := tx.Exec(ctx, "INSERT INTO groups(id,owner_id,name) VALUES($1,$2,$3)", id, actor, name); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, "INSERT INTO group_members(group_id,user_id) VALUES($1,$2)", id, actor); err != nil {
			return nil, err
		}
		if err := audit(ctx, tx, actor, "group.created", id); err != nil {
			return nil, err
		}
		return readGroup(ctx, tx, id, actor)
	}, &result)
	return result, err
}

func readGroup(ctx context.Context, tx pgx.Tx, id, actor string) (Group, error) {
	g := Group{Members: []Profile{}}
	err := tx.QueryRow(ctx, `SELECT id::text,owner_id::text,name FROM groups WHERE id=$1 AND EXISTS(SELECT 1 FROM group_members WHERE group_id=groups.id AND user_id=$2)`, id, actor).Scan(&g.ID, &g.OwnerID, &g.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return g, plans.ErrForbidden
	}
	if err != nil {
		return g, err
	}
	rows, err := tx.Query(ctx, "SELECT u.id::text,u.display_name,u.avatar_url FROM group_members m JOIN users u ON u.id=m.user_id WHERE m.group_id=$1 AND u.deleted_at IS NULL ORDER BY u.id", id)
	if err != nil {
		return g, err
	}
	defer rows.Close()
	for rows.Next() {
		var p Profile
		if err = rows.Scan(&p.ID, &p.DisplayName, &p.AvatarURL); err != nil {
			return g, err
		}
		g.Members = append(g.Members, p)
	}
	return g, rows.Err()
}

func (s *Store) GetGroup(ctx context.Context, id, actor string) (Group, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Group{}, err
	}
	defer tx.Rollback(ctx)
	g, err := readGroup(ctx, tx, id, actor)
	if err != nil {
		return g, err
	}
	return g, tx.Commit(ctx)
}

func lockedGroupOwner(ctx context.Context, tx pgx.Tx, id string) (string, error) {
	var owner string
	err := tx.QueryRow(ctx, "SELECT owner_id::text FROM groups WHERE id=$1 FOR UPDATE", id).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", plans.ErrForbidden
	}
	return owner, err
}

func (s *Store) AddGroupMember(ctx context.Context, id, actor, target string) error {
	if !ids.Valid(target) {
		return plans.ErrInvalid
	}
	target = strings.ToLower(target)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	owner, err := lockedGroupOwner(ctx, tx, id)
	if err != nil {
		return err
	}
	if actor != owner {
		return plans.ErrForbidden
	}
	if target == owner {
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
	tag, err := tx.Exec(ctx, "INSERT INTO group_members(group_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING", id, target)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		if err = recordInvitations(ctx, tx, actor, id, []string{target}); err != nil {
			return err
		}
		if err = socialEvent(ctx, tx, actor, "group.member_added", id, []string{target}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) RemoveGroupMember(ctx context.Context, id, actor, target string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	owner, err := lockedGroupOwner(ctx, tx, id)
	if err != nil {
		return err
	}
	if actor != owner && actor != target {
		return plans.ErrForbidden
	}
	if target == owner {
		return plans.ErrConflict
	}
	tag, err := tx.Exec(ctx, "DELETE FROM group_members WHERE group_id=$1 AND user_id=$2", id, target)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		if err = audit(ctx, tx, actor, "group.member_removed", id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) TransferGroup(ctx context.Context, id, actor, target string) error {
	if !ids.Valid(target) {
		return plans.ErrInvalid
	}
	target = strings.ToLower(target)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	owner, err := lockedGroupOwner(ctx, tx, id)
	if err != nil {
		return err
	}
	if actor != owner {
		return plans.ErrForbidden
	}
	var member bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM group_members m JOIN users u ON u.id=m.user_id WHERE m.group_id=$1 AND m.user_id=$2 AND u.deleted_at IS NULL)", id, target).Scan(&member); err != nil {
		return err
	}
	if !member {
		return plans.ErrForbidden
	}
	if _, err = tx.Exec(ctx, "UPDATE groups SET owner_id=$2 WHERE id=$1", id, target); err != nil {
		return err
	}
	if err = audit(ctx, tx, actor, "group.owner_changed", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
