package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Store struct{ db *pgxpool.Pool }

func Open(ctx context.Context, url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}
	cfg.MaxConns = 10
	cfg.MinConns = 0
	cfg.MaxConnLifetime = time.Hour
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("database pool initialization failed")
	}
	return &Store{db: db}, nil
}
func (s *Store) Close()                         { s.db.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.db.Ping(ctx) }

// Migrate serializes migration runners and commits the schema and version together.
// Run using a migration role; the production API should use a restricted role.
func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(63201742)"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())"); err != nil {
		return err
	}
	names, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		var exists bool
		if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)", name).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		data, err := migrationFiles.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, "INSERT INTO schema_migrations(name) VALUES($1)", name); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) CreatePlan(ctx context.Context, p *plans.Plan) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = insertPlan(ctx, tx, p); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func insertPlan(ctx context.Context, tx pgx.Tx, p *plans.Plan) error {
	_, err := tx.Exec(ctx, `INSERT INTO plans(id,owner_id,title,activity,starts_at,capacity,status,location,version,group_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::uuid)`, p.ID, p.OwnerID, p.Title, p.Activity, p.StartsAt, p.Capacity, p.Status, p.Location, p.Version, p.GroupID)
	if err != nil {
		return err
	}
	for id, response := range p.Members {
		if _, err = tx.Exec(ctx, "INSERT INTO plan_members(plan_id,user_id,response) VALUES($1,$2,$3)", p.ID, id, response); err != nil {
			return err
		}
	}
	if err = event(ctx, tx, p.ID, p.OwnerID, "plan.created", p.Version); err != nil {
		return err
	}
	return nil
}

func (s *Store) GetPlan(ctx context.Context, id, actor string) (*plans.Plan, error) {
	// One repeatable-read snapshot prevents a torn response across plan + members.
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	p, err := load(ctx, tx, id, actor, false)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func load(ctx context.Context, tx pgx.Tx, id, actor string, lock bool) (*plans.Plan, error) {
	query := `SELECT id::text,owner_id::text,title,activity,starts_at,capacity,status,location,version,COALESCE(group_id::text,'') FROM plans WHERE id=$1 AND EXISTS(SELECT 1 FROM plan_members WHERE plan_id=plans.id AND user_id=$2)`
	if lock {
		query += " FOR UPDATE"
	}
	p := &plans.Plan{Members: map[string]string{}}
	err := tx.QueryRow(ctx, query, id, actor).Scan(&p.ID, &p.OwnerID, &p.Title, &p.Activity, &p.StartsAt, &p.Capacity, &p.Status, &p.Location, &p.Version, &p.GroupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, plans.ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, "SELECT user_id::text,response FROM plan_members WHERE plan_id=$1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var user, response string
		if err = rows.Scan(&user, &response); err != nil {
			return nil, err
		}
		p.Members[user] = response
	}
	return p, rows.Err()
}

func (s *Store) Respond(ctx context.Context, id, actor, response string) (bool, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	p, err := load(ctx, tx, id, actor, true)
	if err != nil {
		return false, err
	}
	// Capture time only after load acquired the database row lock.
	now := time.Now()
	changed, err := p.Respond(actor, response, now)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	if _, err = tx.Exec(ctx, "UPDATE plan_members SET response=$3,updated_at=$4 WHERE plan_id=$1 AND user_id=$2", id, actor, response, now); err != nil {
		return false, err
	}
	if _, err = tx.Exec(ctx, "UPDATE plans SET version=$2 WHERE id=$1", id, p.Version); err != nil {
		return false, err
	}
	if err = event(ctx, tx, id, actor, "plan.rsvp", p.Version); err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func event(ctx context.Context, tx pgx.Tx, id, actor, kind string, version int64) error {
	if err := audit(ctx, tx, actor, kind, id); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO outbox(plan_id,actor_id,kind,payload) VALUES($1::uuid,$2::uuid,$3,jsonb_build_object('plan_id',($1::uuid)::text,'version',$4::bigint))`, id, actor, kind, version)
	return err
}
