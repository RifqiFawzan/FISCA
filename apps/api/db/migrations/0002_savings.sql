-- +goose Up

-- ---------------------------------------------------------------------------
-- savings goals (PRD §10.4, ISSUE-20)
--
-- A goal is a *target*, not a wallet. It points at an existing account so the
-- money stays counted exactly once in the ledger: progress is always derived
-- from that account's balance and no second balance column exists (PRD O-17,
-- "hindari double counting").
-- ---------------------------------------------------------------------------

CREATE TABLE savings_goals (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name          TEXT NOT NULL CHECK (btrim(name) <> ''),
    target_amount BIGINT NOT NULL CHECK (target_amount > 0),
    deadline      DATE NOT NULL,
    account_id    UUID REFERENCES accounts (id) ON DELETE SET NULL,
    version       INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    archived_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX savings_goals_user_deadline_idx
    ON savings_goals (user_id, deadline)
    WHERE archived_at IS NULL;

CREATE TRIGGER savings_goals_set_updated_at
    BEFORE UPDATE ON savings_goals
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down

DROP TABLE IF EXISTS savings_goals;
