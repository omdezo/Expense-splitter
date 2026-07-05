-- name: CreateSettlementRun :one
INSERT INTO settlement_runs (group_id, computed_by, snapshot)
VALUES (@group_id::uuid, @computed_by::uuid, @snapshot)
RETURNING id;

-- name: CreatePayment :exec
INSERT INTO payments (settlement_run_id, group_id, from_user_id, to_user_id, amount_baisa)
VALUES (@settlement_run_id::uuid, @group_id::uuid, @from_user_id::uuid, @to_user_id::uuid, @amount_baisa);
