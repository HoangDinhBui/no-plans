package store

import "context"

// QueuePlanReminders records one reminder per confirmed plan version.
func (s *Store) QueuePlanReminders(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
 INSERT INTO outbox(plan_id, actor_id, kind, payload)
 SELECT id, owner_id, 'plan.reminder',
        jsonb_build_object('version', version, 'starts_at', starts_at)
 FROM plans
 WHERE status = 'confirmed' AND starts_at > now()
   AND starts_at <= now() + interval '1 hour'
 ON CONFLICT (plan_id, (payload->>'version')) WHERE kind = 'plan.reminder' DO NOTHING
 `)
	return err
}

// CurrentPlanReminder checks the schedule and RSVP before a device send.
func (s *Store) CurrentPlanReminder(ctx context.Context, delivery PushDelivery) (bool, error) {
	var current bool
	err := s.db.QueryRow(ctx, `
 SELECT EXISTS (
  SELECT 1 FROM outbox e
  JOIN plans p ON p.id = e.plan_id
  JOIN plan_members m ON m.plan_id = p.id AND m.user_id = $2
  WHERE e.id = $1 AND p.status = 'confirmed' AND p.starts_at > now()
    AND p.version::text = e.payload->>'version'
    AND p.starts_at = (e.payload->>'starts_at')::timestamptz
    AND m.response = 'going'
 )`, delivery.EventID, delivery.RecipientID).Scan(&current)
	return current, err
}
