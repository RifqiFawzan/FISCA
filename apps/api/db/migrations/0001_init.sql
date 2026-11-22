-- +goose Up

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- ---------------------------------------------------------------------------
-- users
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id            UUID PRIMARY KEY,
    email         CITEXT NOT NULL UNIQUE,
    display_name  TEXT NOT NULL CHECK (btrim(display_name) <> ''),
    timezone      TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    locale        TEXT NOT NULL DEFAULT 'id-ID',
    base_currency CHAR(3) NOT NULL DEFAULT 'IDR',
    onboarded_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- accounts
-- ---------------------------------------------------------------------------
CREATE TABLE accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name            TEXT NOT NULL CHECK (btrim(name) <> ''),
    type            TEXT NOT NULL CHECK (type IN ('cash', 'bank', 'ewallet', 'other')),
    opening_balance BIGINT NOT NULL DEFAULT 0,
    is_default      BOOLEAN NOT NULL DEFAULT false,
    archived_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX accounts_user_name_unique
    ON accounts (user_id, lower(name))
    WHERE archived_at IS NULL;

CREATE UNIQUE INDEX accounts_one_default_per_user
    ON accounts (user_id)
    WHERE is_default AND archived_at IS NULL;

CREATE TRIGGER accounts_set_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- categories
-- ---------------------------------------------------------------------------
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        TEXT NOT NULL CHECK (btrim(name) <> ''),
    kind        TEXT NOT NULL CHECK (kind IN ('income', 'expense')),
    icon        TEXT,
    color       TEXT,
    is_default  BOOLEAN NOT NULL DEFAULT false,
    archived_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX categories_user_name_kind_unique
    ON categories (user_id, lower(name), kind)
    WHERE archived_at IS NULL;

CREATE TRIGGER categories_set_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- transactions
-- ---------------------------------------------------------------------------
CREATE TABLE transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type            TEXT NOT NULL CHECK (type IN ('income', 'expense', 'transfer')),
    amount          BIGINT NOT NULL CHECK (amount > 0),
    currency        CHAR(3) NOT NULL DEFAULT 'IDR' CHECK (currency = 'IDR'),
    description     TEXT NOT NULL CHECK (btrim(description) <> ''),
    category_id     UUID REFERENCES categories (id) ON DELETE RESTRICT,
    account_id      UUID REFERENCES accounts (id) ON DELETE RESTRICT,
    from_account_id UUID REFERENCES accounts (id) ON DELETE RESTRICT,
    to_account_id   UUID REFERENCES accounts (id) ON DELETE RESTRICT,
    payment_method  TEXT,
    merchant_name   TEXT,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    notes           TEXT,
    source          TEXT NOT NULL DEFAULT 'manual'
                        CHECK (source IN ('manual', 'voice', 'chat')),
    status          TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'voided')),
    voided_at       TIMESTAMPTZ,
    void_reason     TEXT,
    refund_of_id    UUID REFERENCES transactions (id) ON DELETE SET NULL,
    version         INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transactions_type_shape CHECK (
        (
            type = 'transfer'
            AND from_account_id IS NOT NULL
            AND to_account_id IS NOT NULL
            AND from_account_id <> to_account_id
            AND account_id IS NULL
            AND category_id IS NULL
        )
        OR
        (
            type IN ('income', 'expense')
            AND account_id IS NOT NULL
            AND from_account_id IS NULL
            AND to_account_id IS NULL
        )
    ),
    CONSTRAINT transactions_void_shape CHECK (
        (status = 'active' AND voided_at IS NULL)
        OR (status = 'voided' AND voided_at IS NOT NULL)
    )
);

CREATE INDEX transactions_user_occurred_idx
    ON transactions (user_id, occurred_at DESC);
CREATE INDEX transactions_user_status_occurred_idx
    ON transactions (user_id, status, occurred_at DESC);
CREATE INDEX transactions_user_category_occurred_idx
    ON transactions (user_id, category_id, occurred_at DESC);
CREATE INDEX transactions_user_account_occurred_idx
    ON transactions (user_id, account_id, occurred_at DESC);
CREATE INDEX transactions_user_from_account_idx
    ON transactions (user_id, from_account_id, occurred_at DESC)
    WHERE type = 'transfer';
CREATE INDEX transactions_user_to_account_idx
    ON transactions (user_id, to_account_id, occurred_at DESC)
    WHERE type = 'transfer';
CREATE INDEX transactions_description_trgm_idx
    ON transactions USING GIN (description gin_trgm_ops);
CREATE INDEX transactions_merchant_trgm_idx
    ON transactions USING GIN (merchant_name gin_trgm_ops);

CREATE TRIGGER transactions_set_updated_at
    BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- audit_logs
-- ---------------------------------------------------------------------------
CREATE TABLE audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    actor       TEXT NOT NULL CHECK (actor IN ('user', 'ai', 'system')),
    entity_type TEXT NOT NULL,
    entity_id   UUID,
    action      TEXT NOT NULL,
    before      JSONB,
    after       JSONB,
    request_id  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_logs_user_created_idx
    ON audit_logs (user_id, created_at DESC);
CREATE INDEX audit_logs_entity_idx
    ON audit_logs (entity_type, entity_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- idempotency_keys
-- ---------------------------------------------------------------------------
CREATE TABLE idempotency_keys (
    user_id       UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    key           TEXT NOT NULL,
    endpoint      TEXT NOT NULL,
    request_hash  TEXT NOT NULL,
    status_code   INTEGER NOT NULL,
    response_body JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, key, endpoint)
);

CREATE INDEX idempotency_keys_created_idx ON idempotency_keys (created_at);

-- ---------------------------------------------------------------------------
-- budgets
-- ---------------------------------------------------------------------------
CREATE TABLE budgets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    period_start  DATE NOT NULL,
    period_end    DATE NOT NULL,
    overall_limit BIGINT NOT NULL CHECK (overall_limit > 0),
    currency      CHAR(3) NOT NULL DEFAULT 'IDR' CHECK (currency = 'IDR'),
    status        TEXT NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active', 'archived')),
    version       INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT budgets_period_valid CHECK (period_end > period_start)
);

CREATE UNIQUE INDEX budgets_user_period_unique
    ON budgets (user_id, period_start);

CREATE TRIGGER budgets_set_updated_at
    BEFORE UPDATE ON budgets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE budget_categories (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_id    UUID NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    category_id  UUID NOT NULL REFERENCES categories (id) ON DELETE RESTRICT,
    limit_amount BIGINT NOT NULL CHECK (limit_amount >= 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT budget_categories_unique UNIQUE (budget_id, category_id)
);

CREATE INDEX budget_categories_category_idx ON budget_categories (category_id);

CREATE TRIGGER budget_categories_set_updated_at
    BEFORE UPDATE ON budget_categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- budget_proposals
-- ---------------------------------------------------------------------------
CREATE TABLE budget_proposals (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    budget_id    UUID NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    base_version INTEGER NOT NULL CHECK (base_version > 0),
    status       TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'approved', 'rejected', 'expired', 'stale')),
    changes      JSONB NOT NULL,
    rationale    TEXT,
    impact       JSONB,
    expires_at   TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '24 hours',
    decided_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX budget_proposals_user_status_idx
    ON budget_proposals (user_id, status, created_at DESC);
CREATE INDEX budget_proposals_budget_idx
    ON budget_proposals (budget_id, created_at DESC);

CREATE TRIGGER budget_proposals_set_updated_at
    BEFORE UPDATE ON budget_proposals
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- assistant conversations
-- ---------------------------------------------------------------------------
CREATE TABLE assistant_conversations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX assistant_conversations_user_idx
    ON assistant_conversations (user_id, last_message_at DESC);

CREATE TABLE assistant_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES assistant_conversations (id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'tool')),
    content         TEXT,
    tool_calls      JSONB,
    card            JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX assistant_messages_conversation_idx
    ON assistant_messages (conversation_id, created_at);

-- ---------------------------------------------------------------------------
-- drafts (menunggu klarifikasi; tidak pernah masuk agregasi)
-- ---------------------------------------------------------------------------
CREATE TABLE drafts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES assistant_conversations (id) ON DELETE SET NULL,
    payload         JSONB NOT NULL,
    missing_fields  TEXT[] NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'resolved', 'expired')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '24 hours'
);

CREATE INDEX drafts_user_status_idx ON drafts (user_id, status, created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS drafts;
DROP TABLE IF EXISTS assistant_messages;
DROP TABLE IF EXISTS assistant_conversations;
DROP TABLE IF EXISTS budget_proposals;
DROP TABLE IF EXISTS budget_categories;
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS set_updated_at();
