CREATE TABLE push_deliveries (
 event_id uuid NOT NULL REFERENCES outbox(id) ON DELETE CASCADE,
 recipient_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
 available_at timestamptz NOT NULL DEFAULT now(),
 lease_until timestamptz,
 attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
 ticket_id text,
 sent_at timestamptz,
 completed_at timestamptz,
 failure_code text,
 PRIMARY KEY (event_id, recipient_id, device_id),
 CHECK ((ticket_id IS NULL) = (sent_at IS NULL))
);
CREATE INDEX push_deliveries_pending ON push_deliveries (available_at)
 WHERE completed_at IS NULL;
REVOKE ALL ON push_deliveries FROM PUBLIC;
