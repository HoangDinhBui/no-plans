package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}
type GroupSummary struct {
	ID          string `json:"id"`
	OwnerID     string `json:"owner_id"`
	Name        string `json:"name"`
	MemberCount int    `json:"member_count"`
}
type FriendshipView struct {
	Friendship
	Peer Profile `json:"peer"`
}

func validPage(cursor string, limit int) bool {
	return limit >= 1 && limit <= 100 && (cursor == "" || ids.Valid(cursor))
}

func (s *Store) ListGroups(ctx context.Context, actor, cursor string, limit int) (Page[GroupSummary], error) {
	page := Page[GroupSummary]{Items: []GroupSummary{}}
	if !validPage(cursor, limit) {
		return page, plans.ErrInvalid
	}
	rows, err := s.db.Query(ctx, `SELECT g.id::text,g.owner_id::text,g.name,(SELECT count(*) FROM group_members m WHERE m.group_id=g.id) FROM groups g JOIN group_members mine ON mine.group_id=g.id AND mine.user_id=$1 WHERE ($2='' OR g.id>NULLIF($2,'')::uuid) ORDER BY g.id LIMIT $3`, actor, strings.ToLower(cursor), limit+1)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var g GroupSummary
		if err = rows.Scan(&g.ID, &g.OwnerID, &g.Name, &g.MemberCount); err != nil {
			return page, err
		}
		page.Items = append(page.Items, g)
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

func (s *Store) ListFriendships(ctx context.Context, actor, cursor string, limit int) (Page[FriendshipView], error) {
	page := Page[FriendshipView]{Items: []FriendshipView{}}
	if !validPage(cursor, limit) {
		return page, plans.ErrInvalid
	}
	rows, err := s.db.Query(ctx, `SELECT f.id::text,f.low_id::text,f.high_id::text,f.requester_id::text,f.status,u.id::text,u.display_name,u.avatar_url FROM friendships f JOIN users u ON u.id=CASE WHEN f.low_id=$1 THEN f.high_id ELSE f.low_id END WHERE (f.low_id=$1 OR f.high_id=$1) AND ($2='' OR f.id>NULLIF($2,'')::uuid) AND u.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM blocks b WHERE (b.blocker_id=$1 AND b.blocked_id=u.id) OR (b.blocker_id=u.id AND b.blocked_id=$1)) ORDER BY f.id LIMIT $3`, actor, strings.ToLower(cursor), limit+1)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var f FriendshipView
		if err = rows.Scan(&f.ID, &f.LowID, &f.HighID, &f.RequesterID, &f.Status, &f.Peer.ID, &f.Peer.DisplayName, &f.Peer.AvatarURL); err != nil {
			return page, err
		}
		page.Items = append(page.Items, f)
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

type PlanSummary struct {
	ID           string    `json:"id"`
	OwnerID      string    `json:"owner_id"`
	Title        string    `json:"title"`
	Activity     string    `json:"activity"`
	StartsAt     time.Time `json:"starts_at"`
	Capacity     int       `json:"capacity"`
	Status       string    `json:"status"`
	Location     string    `json:"location"`
	Version      int64     `json:"version"`
	MyResponse   string    `json:"my_response"`
	GoingCount   int       `json:"going_count"`
	PendingCount int       `json:"pending_count"`
}
type planCursor struct {
	At     time.Time `json:"at"`
	ID     string    `json:"id"`
	Status string    `json:"status"`
}

func (s *Store) ListPlans(ctx context.Context, actor, status, cursor string, limit int) (Page[PlanSummary], error) {
	page := Page[PlanSummary]{Items: []PlanSummary{}}
	if limit < 1 || limit > 100 {
		return page, plans.ErrInvalid
	}
	filter := ""
	direction := "ASC"
	compare := ">"
	switch status {
	case "upcoming":
		filter = "p.status IN ('inviting','confirmed') AND p.starts_at>now()"
	case "invited":
		filter = "mine.response='invited' AND p.status IN ('inviting','confirmed') AND p.starts_at>now()"
	case "past":
		filter = "(p.status IN ('cancelled','completed') OR p.starts_at<=now())"
		direction = "DESC"
		compare = "<"
	default:
		return page, plans.ErrInvalid
	}
	var c planCursor
	if cursor != "" {
		if len(cursor) > 512 {
			return page, plans.ErrInvalid
		}
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || json.Unmarshal(data, &c) != nil || !ids.Valid(c.ID) || c.At.IsZero() || c.Status != status {
			return page, plans.ErrInvalid
		}
	}
	query := `SELECT p.id::text,p.owner_id::text,p.title,p.activity,p.starts_at,p.capacity,p.status,p.location,p.version,mine.response,(SELECT count(*) FROM plan_members m WHERE m.plan_id=p.id AND m.response='going'),(SELECT count(*) FROM plan_members m WHERE m.plan_id=p.id AND m.response='invited') FROM plans p JOIN plan_members mine ON mine.plan_id=p.id AND mine.user_id=$1 WHERE ` + filter
	args := []any{actor}
	if cursor != "" {
		query += " AND (p.starts_at,p.id) " + compare + " ($2,$3::uuid)"
		args = append(args, c.At, c.ID)
	}
	query += " ORDER BY p.starts_at " + direction + ",p.id " + direction
	if cursor == "" {
		query += " LIMIT $2"
	} else {
		query += " LIMIT $4"
	}
	args = append(args, limit+1)
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var p PlanSummary
		if err = rows.Scan(&p.ID, &p.OwnerID, &p.Title, &p.Activity, &p.StartsAt, &p.Capacity, &p.Status, &p.Location, &p.Version, &p.MyResponse, &p.GoingCount, &p.PendingCount); err != nil {
			return page, err
		}
		page.Items = append(page.Items, p)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[limit-1]
		data, _ := json.Marshal(planCursor{At: last.StartsAt, ID: last.ID, Status: status})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return page, nil
}
