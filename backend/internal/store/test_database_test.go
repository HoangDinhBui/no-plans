package store

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/jackc/pgx/v5"
)

// Each test owns a schema, including its failure-injection triggers and quota
// history. Never reset shared public tables or rely on previous test cleanup.
func openTestStore(t *testing.T, ctx context.Context, dsn string) (*Store, error) {
	t.Helper()
	root, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	schema := "store_test_" + strings.ReplaceAll(ids.New(), "-", "")
	if _, err = root.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		root.Close(ctx)
		return nil, err
	}
	t.Cleanup(func() {
		_, _ = root.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		_ = root.Close(context.Background())
	})
	parsed, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()
	s, err := Open(ctx, parsed.String())
	if err != nil {
		return nil, err
	}
	t.Cleanup(s.Close)
	return s, nil
}
