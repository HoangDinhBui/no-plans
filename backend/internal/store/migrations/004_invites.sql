CREATE TABLE invite_links (
 id uuid PRIMARY KEY,
 plan_id uuid NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
 created_by uuid NOT NULL REFERENCES users(id),
 token_hash bytea NOT NULL UNIQUE CHECK(octet_length(token_hash)=32),
 expires_at timestamptz NOT NULL,
 max_uses integer NOT NULL CHECK(max_uses BETWEEN 1 AND 1000),
 use_count integer NOT NULL DEFAULT 0 CHECK(use_count>=0 AND use_count<=max_uses),
 revoked_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX invite_links_plan ON invite_links(plan_id,id);
CREATE TABLE plan_member_revocations (
 plan_id uuid NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id),
 revoked_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(plan_id,user_id)
);
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
