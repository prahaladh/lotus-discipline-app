-- PostgreSQL schema for Lotus Discipline app

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS habits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    goal_minutes    INT  NOT NULL CHECK (goal_minutes > 0),
    -- unit for the goal value, e.g. 'minutes', 'pages', 'reps'
    unit            TEXT NOT NULL DEFAULT 'minutes'
);

-- Join table: which habits each user chose
CREATE TABLE IF NOT EXISTS user_habits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    habit_id        UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    UNIQUE (user_id, habit_id)
);

-- Program tracking per user
CREATE TABLE IF NOT EXISTS user_programs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    start_date      DATE NOT NULL DEFAULT CURRENT_DATE
);

-- Daily check-ins and completions
CREATE TABLE IF NOT EXISTS habit_completions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    habit_id        UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    completed_on    DATE NOT NULL DEFAULT CURRENT_DATE,
    minutes         INT  NOT NULL CHECK (minutes >= 0),
    UNIQUE (user_id, habit_id, completed_on)
);

-- Backwards‑compatible migration helpers for existing databases.
ALTER TABLE habits
    ADD COLUMN IF NOT EXISTS unit TEXT NOT NULL DEFAULT 'minutes';

