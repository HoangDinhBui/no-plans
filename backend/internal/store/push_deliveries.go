package store

import (
	"context"
	"time"
)

// ExpandPushEvent records eligible devices and completes the outbox lease atomically.
// Only event kinds with defined notification recipients enter the delivery queue.
func (s *Store) ExpandPushEvent(ctx context.Context, event OutboxEvent) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		WITH completed AS (
			UPDATE outbox SET processed_at = now(), lease_until = NULL
			WHERE id = $1 AND attempts = $2 AND lease_until > now()
			  AND processed_at IS NULL AND dead_letter_at IS NULL
			RETURNING id, actor_id, plan_id, kind, payload
		), queued AS (
			INSERT INTO push_deliveries(event_id, recipient_id, device_id)
			SELECT e.id, d.user_id, d.id
			FROM completed e
			JOIN users actor ON actor.id = e.actor_id AND actor.deleted_at IS NULL
			JOIN devices d ON d.user_id <> e.actor_id
			JOIN users recipient ON recipient.id = d.user_id AND recipient.deleted_at IS NULL
			WHERE (
				(e.kind IN ('friend.requested', 'friend.accepted', 'group.member_added')
				 AND (e.payload->'recipient_ids') ? d.user_id::text)
				OR
				(e.kind IN ('plan.invited', 'plan.created') AND (e.payload->'recipient_ids') ? d.user_id::text AND EXISTS (SELECT 1 FROM plan_members m WHERE m.plan_id = e.plan_id AND m.user_id = d.user_id AND m.response = 'invited'))
				OR (e.kind IN ('plan.updated', 'plan.confirm', 'plan.cancel')
				 AND EXISTS (SELECT 1 FROM plan_members m WHERE m.plan_id = e.plan_id AND m.user_id = d.user_id))
			)
			AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = e.actor_id AND b.blocked_id = d.user_id)
				   OR (b.blocker_id = d.user_id AND b.blocked_id = e.actor_id)
			)
			ON CONFLICT DO NOTHING
		)
		SELECT id FROM completed`, event.ID, event.Attempt)
	return err == nil && tag.RowsAffected() == 1, err
}

type PushDelivery struct {
	DeviceUpdatedAt                                               *time.Time
	EventID, RecipientID, DeviceID, Token, Kind, PlanID, TicketID string
	Attempt                                                       int
}

// ClaimPushDelivery leases one due delivery and returns its current phase.
func (s *Store) ClaimPushDelivery(ctx context.Context) (PushDelivery, error) {
	var delivery PushDelivery
	err := s.db.QueryRow(ctx, `
WITH candidate AS (
 SELECT p.event_id, p.recipient_id, p.device_id
 FROM push_deliveries p
 WHERE p.completed_at IS NULL AND p.available_at <= now()
 AND (p.lease_until IS NULL OR p.lease_until <= now())
 ORDER BY p.available_at, p.event_id, p.recipient_id, p.device_id
 LIMIT 1 FOR UPDATE SKIP LOCKED
)
UPDATE push_deliveries p
SET attempts = p.attempts + 1, lease_until = now() + interval '1 minute'
FROM candidate c, devices d, outbox o
WHERE (p.event_id, p.recipient_id, p.device_id) = (c.event_id, c.recipient_id, c.device_id)
AND d.id = p.device_id AND o.id = p.event_id
RETURNING p.event_id::text, p.recipient_id::text, p.device_id::text,
 d.push_token, o.kind, coalesce(o.plan_id::text, ''), coalesce(p.ticket_id, ''), p.attempts, CASE WHEN p.ticket_id IS NULL THEN d.updated_at ELSE p.sent_device_updated_at END
`).Scan(&delivery.EventID, &delivery.RecipientID, &delivery.DeviceID,
		&delivery.Token, &delivery.Kind, &delivery.PlanID, &delivery.TicketID, &delivery.Attempt, &delivery.DeviceUpdatedAt)
	return delivery, err
}

// ResolvePushDelivery applies a result only while the caller owns the lease.
// A new ticket schedules receipt polling. A nil delay completes the delivery.
func (s *Store) ResolvePushDelivery(ctx context.Context, delivery PushDelivery, ticket, failure string, delay *time.Duration) (bool, error) {
	var delayMillis *int64
	if delay != nil {
		millis := delay.Milliseconds()
		delayMillis = &millis
	}
	tag, err := s.db.Exec(ctx, `
WITH resolved AS (
UPDATE push_deliveries
SET lease_until = NULL,
 available_at = coalesce(now() + $7::bigint * interval '1 millisecond', available_at),
 ticket_id = coalesce(nullif($5, ''), ticket_id),
 sent_device_updated_at = CASE WHEN $5 <> '' THEN $8::timestamptz ELSE sent_device_updated_at END,
 sent_at = CASE WHEN $5 <> '' THEN now() ELSE sent_at END,
 completed_at = CASE WHEN $7::bigint IS NULL THEN now() END,
 failure_code = nullif($6, '')
WHERE event_id = $1 AND recipient_id = $2 AND device_id = $3
AND attempts = $4 AND lease_until > now() AND completed_at IS NULL
RETURNING device_id, recipient_id
), revoked AS (
 DELETE FROM devices d USING resolved r
 WHERE $6 = 'DeviceNotRegistered' AND $7::bigint IS NULL
 AND d.id = r.device_id AND d.user_id = r.recipient_id AND d.updated_at = $8::timestamptz
)
SELECT device_id FROM resolved
`, delivery.EventID, delivery.RecipientID, delivery.DeviceID, delivery.Attempt, ticket, failure, delayMillis, delivery.DeviceUpdatedAt)
	return err == nil && tag.RowsAffected() == 1, err
}
