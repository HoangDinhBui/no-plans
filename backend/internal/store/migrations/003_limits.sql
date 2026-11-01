CREATE TABLE invitation_batches (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 actor_id uuid NOT NULL REFERENCES users(id),
 group_id uuid REFERENCES groups(id) ON DELETE SET NULL,
 recipient_count integer NOT NULL CHECK(recipient_count>0),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX invitation_actor_time ON invitation_batches(actor_id,created_at);
CREATE INDEX invitation_group_time ON invitation_batches(actor_id,group_id,created_at);
CREATE TABLE invitation_recipients (
 batch_id uuid NOT NULL REFERENCES invitation_batches(id) ON DELETE CASCADE,
 recipient_id uuid NOT NULL REFERENCES users(id),
 PRIMARY KEY(batch_id,recipient_id)
);
CREATE INDEX invitation_recipient ON invitation_recipients(recipient_id,batch_id);
CREATE TABLE api_rate_buckets (
 actor_id uuid NOT NULL REFERENCES users(id),
 bucket timestamptz NOT NULL,
 requests integer NOT NULL CHECK(requests>0),
 PRIMARY KEY(actor_id,bucket)
);
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
