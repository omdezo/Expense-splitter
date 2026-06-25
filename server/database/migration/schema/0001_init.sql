-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TYPE verification_status AS ENUM ('registered', 'pending_verification', 'verified', 'rejected');
CREATE TYPE group_status        AS ENUM ('open', 'closed', 'settled');
CREATE TYPE membership_role     AS ENUM ('group_admin', 'member');
CREATE TYPE membership_status   AS ENUM ('requested', 'approved', 'rejected');
CREATE TYPE expense_category    AS ENUM ('lodging', 'fuel', 'food', 'transport', 'other');
CREATE TYPE split_type          AS ENUM ('equal', 'subset', 'weighted');
CREATE TYPE payment_status      AS ENUM ('pending', 'proof_submitted', 'creditor_confirmed', 'disputed', 'settled');
CREATE TYPE proof_type          AS ENUM ('image', 'text');
CREATE TYPE nudge_type          AS ENUM ('debtor', 'creditor');

CREATE TABLE users (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email               citext NOT NULL UNIQUE,
    is_global_admin     boolean NOT NULL DEFAULT false,
    verification_status verification_status NOT NULL DEFAULT 'registered',
    verified_by         uuid REFERENCES users (id),
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX one_global_admin ON users (is_global_admin) WHERE is_global_admin;

CREATE TABLE groups (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  text NOT NULL,
    start_date            date NOT NULL,
    end_date              date NOT NULL,
    status                group_status NOT NULL DEFAULT 'open',
    invite_token          uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    status_token          uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    expected_member_count int,
    created_by            uuid NOT NULL REFERENCES users (id),
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date)
);

CREATE TABLE memberships (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id   uuid NOT NULL REFERENCES groups (id),
    user_id    uuid NOT NULL REFERENCES users (id),
    role       membership_role   NOT NULL DEFAULT 'member',
    status     membership_status NOT NULL DEFAULT 'requested',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (group_id, user_id),
    UNIQUE (group_id, id)
);
CREATE INDEX ON memberships (user_id);

CREATE TABLE expenses (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id     uuid NOT NULL REFERENCES groups (id),
    paid_by      uuid NOT NULL,
    amount_baisa bigint NOT NULL CHECK (amount_baisa > 0),
    category     expense_category NOT NULL,
    description  text NOT NULL,
    occurred_on  date NOT NULL,
    split_type   split_type NOT NULL DEFAULT 'equal',
    deleted_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (group_id, paid_by) REFERENCES memberships (group_id, id)
);
CREATE INDEX ON expenses (group_id);
CREATE INDEX ON expenses (group_id, paid_by);

CREATE TABLE expense_shares (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_id uuid NOT NULL REFERENCES expenses (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id),
    weight     int NOT NULL DEFAULT 1 CHECK (weight > 0),
    UNIQUE (expense_id, user_id)
);
CREATE INDEX ON expense_shares (user_id);

CREATE TABLE settlement_runs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    uuid NOT NULL UNIQUE REFERENCES groups (id),
    computed_by uuid NOT NULL REFERENCES users (id),
    snapshot    jsonb NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE payments (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    settlement_run_id uuid NOT NULL REFERENCES settlement_runs (id),
    group_id          uuid NOT NULL REFERENCES groups (id),
    from_user_id      uuid NOT NULL,
    to_user_id        uuid NOT NULL,
    amount_baisa      bigint NOT NULL CHECK (amount_baisa > 0),
    status            payment_status NOT NULL DEFAULT 'pending',
    version           int NOT NULL DEFAULT 0,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CHECK (from_user_id <> to_user_id),
    FOREIGN KEY (group_id, from_user_id) REFERENCES memberships (group_id, user_id),
    FOREIGN KEY (group_id, to_user_id)   REFERENCES memberships (group_id, user_id)
);
CREATE INDEX ON payments (settlement_run_id);
CREATE INDEX ON payments (group_id);
CREATE INDEX ON payments (from_user_id);
CREATE INDEX ON payments (to_user_id);

CREATE TABLE proofs (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id uuid NOT NULL REFERENCES payments (id) ON DELETE CASCADE,
    proof_type proof_type NOT NULL,
    sha256     char(64),
    byte_size  bigint,
    is_current boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX one_current_proof_per_payment ON proofs (payment_id) WHERE is_current;

CREATE TABLE notifications (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id        uuid NOT NULL REFERENCES payments (id) ON DELETE CASCADE,
    recipient_user_id uuid NOT NULL REFERENCES users (id),
    nudge_type        nudge_type NOT NULL,
    last_sent_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (payment_id, recipient_user_id, nudge_type)
);
CREATE INDEX ON notifications (recipient_user_id);

CREATE TABLE audit_log (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    group_id      uuid REFERENCES groups (id),
    actor_user_id uuid REFERENCES users (id),
    action        text NOT NULL,
    before        jsonb,
    after         jsonb,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON audit_log (group_id);
CREATE INDEX ON audit_log (actor_user_id);

-- +goose Down
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS proofs;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS settlement_runs;
DROP TABLE IF EXISTS expense_shares;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS nudge_type;
DROP TYPE IF EXISTS proof_type;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS split_type;
DROP TYPE IF EXISTS expense_category;
DROP TYPE IF EXISTS membership_status;
DROP TYPE IF EXISTS membership_role;
DROP TYPE IF EXISTS group_status;
DROP TYPE IF EXISTS verification_status;
