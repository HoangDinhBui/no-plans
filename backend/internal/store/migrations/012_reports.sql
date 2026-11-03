CREATE TABLE reports (
 id uuid PRIMARY KEY,
 reporter_id uuid REFERENCES users(id) ON DELETE SET NULL,
 target_id uuid REFERENCES users(id) ON DELETE SET NULL,
 plan_id uuid REFERENCES plans(id) ON DELETE SET NULL,
 reason text NOT NULL CHECK (reason IN ('harassment', 'spam', 'unsafe', 'other')),
 created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX reports_reporter_created ON reports (reporter_id, created_at DESC);
REVOKE ALL ON reports FROM PUBLIC;
