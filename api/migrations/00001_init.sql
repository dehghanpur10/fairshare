-- +goose Up
CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text NOT NULL,
    name          text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

CREATE TABLE groups (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    currency   char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    created_by uuid NOT NULL REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE group_members (
    group_id  uuid NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    user_id   uuid NOT NULL REFERENCES users (id),
    role      text NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member')),
    joined_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, user_id)
);
CREATE INDEX group_members_user_id_idx ON group_members (user_id);

-- The composite foreign keys below reference group_members rather than users,
-- so the database itself refuses an expense paid by, or split with, someone
-- outside the group.
CREATE TABLE expenses (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    uuid NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    paid_by     uuid NOT NULL,
    description text NOT NULL CHECK (length(description) BETWEEN 1 AND 200),
    amount      bigint NOT NULL CHECK (amount > 0),
    spent_on    date NOT NULL,
    created_by  uuid NOT NULL REFERENCES users (id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (group_id, paid_by) REFERENCES group_members (group_id, user_id)
);
CREATE INDEX expenses_group_page_idx ON expenses (group_id, created_at DESC, id DESC);

CREATE TABLE expense_shares (
    expense_id uuid NOT NULL REFERENCES expenses (id) ON DELETE CASCADE,
    group_id   uuid NOT NULL,
    user_id    uuid NOT NULL,
    amount     bigint NOT NULL CHECK (amount >= 0),
    PRIMARY KEY (expense_id, user_id),
    FOREIGN KEY (group_id, user_id) REFERENCES group_members (group_id, user_id)
);
CREATE INDEX expense_shares_group_user_idx ON expense_shares (group_id, user_id);

CREATE TABLE settlements (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id   uuid NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    from_user  uuid NOT NULL,
    to_user    uuid NOT NULL,
    amount     bigint NOT NULL CHECK (amount > 0),
    created_by uuid NOT NULL REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (from_user <> to_user),
    FOREIGN KEY (group_id, from_user) REFERENCES group_members (group_id, user_id),
    FOREIGN KEY (group_id, to_user) REFERENCES group_members (group_id, user_id)
);
CREATE INDEX settlements_group_created_idx ON settlements (group_id, created_at DESC);

-- +goose Down
DROP TABLE settlements;
DROP TABLE expense_shares;
DROP TABLE expenses;
DROP TABLE group_members;
DROP TABLE groups;
DROP TABLE users;
