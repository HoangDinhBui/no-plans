package store

import "testing"

func TestPurgeExpiredQuotas(t *testing.T) {
	s, ctx, actor, _, _, _ := inviteFixture(t)
	if _, err := s.db.Exec(ctx, "INSERT INTO api_rate_buckets(actor_id,bucket,requests) VALUES($1,now(),1),($1,now()-interval '2 days',1)", actor); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(ctx, "INSERT INTO invitation_batches(actor_id,recipient_count,created_at) VALUES($1,1,now()),($1,1,now()-interval '2 days')", actor); err != nil {
		t.Fatal(err)
	}
	if err := s.PurgeExpiredQuotas(ctx); err != nil {
		t.Fatal(err)
	}
	var buckets, batches int
	err := s.db.QueryRow(ctx, `SELECT
  (SELECT count(*) FROM api_rate_buckets WHERE actor_id=$1),
  (SELECT count(*) FROM invitation_batches WHERE actor_id=$1)`, actor).Scan(&buckets, &batches)
	if err != nil || buckets != 1 || batches != 1 {
		t.Fatalf("retained %d buckets and %d batches: %v", buckets, batches, err)
	}
}
