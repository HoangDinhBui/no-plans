package store

import "context"

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
				(e.kind IN ('plan.updated', 'plan.confirm', 'plan.cancel')
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
