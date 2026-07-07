-- name: ListStalePayments :many
SELECT id, from_user_id, to_user_id, amount_baisa, status
FROM payments
WHERE group_id = @group_id::uuid
  AND status IN ('pending', 'proof_submitted')
  AND updated_at < now() - make_interval(hours => @hours::int)
ORDER BY created_at, id;

-- name: UpsertNudge :one
INSERT INTO notifications (payment_id, recipient_user_id, nudge_type)
VALUES (@payment_id::uuid, @recipient_user_id::uuid, @nudge_type)
ON CONFLICT (payment_id, recipient_user_id, nudge_type)
DO UPDATE SET last_sent_at = now()
WHERE notifications.last_sent_at < now() - make_interval(hours => @hours::int)
RETURNING id;
