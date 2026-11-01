CREATE TABLE friendships (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 low_id uuid NOT NULL REFERENCES users(id),
 high_id uuid NOT NULL REFERENCES users(id),
 requester_id uuid NOT NULL REFERENCES users(id),
 status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted')),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(low_id,high_id),
 CHECK(low_id<high_id),
 CHECK(requester_id IN (low_id,high_id))
);
CREATE TABLE blocks (
 blocker_id uuid NOT NULL REFERENCES users(id),
 blocked_id uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(blocker_id,blocked_id),
 CHECK(blocker_id<>blocked_id)
);
CREATE TABLE audit_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 actor_id uuid NOT NULL REFERENCES users(id),
 kind text NOT NULL,
 entity_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX group_members_user ON group_members(user_id,group_id);
CREATE INDEX friendships_high ON friendships(high_id,id);
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
