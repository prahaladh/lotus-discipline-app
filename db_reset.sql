-- WARNING: This script will delete all existing data in your tables.

-- Drop tables in reverse order of creation to avoid foreign key errors.
DROP TABLE IF EXISTS habit_completions;
DROP TABLE IF EXISTS user_programs;
DROP TABLE IF EXISTS user_habits;
DROP TABLE IF EXISTS habits;
DROP TABLE IF EXISTS users;


-- Create the users table
-- This table stores user login information.
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create the habits table
-- This table stores the definitions of all possible habits.
CREATE TABLE habits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    goal_minutes    INT  NOT NULL CHECK (goal_minutes > 0),
    unit            TEXT NOT NULL DEFAULT 'minutes'
);

-- Create the user_habits join table
-- This table links users to the habits they have chosen.
CREATE TABLE user_habits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    habit_id        UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    UNIQUE (user_id, habit_id)
);

-- Create the user_programs table
-- This table tracks the start date of a user's program.
CREATE TABLE user_programs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    start_date      DATE NOT NULL DEFAULT CURRENT_DATE
);

-- Create the habit_completions table
-- This table records when a user completes a habit for a specific day.
CREATE TABLE habit_completions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    habit_id        UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    completed_on    DATE NOT NULL DEFAULT CURRENT_DATE,
    minutes         INT  NOT NULL CHECK (minutes >= 0),
    UNIQUE (user_id, habit_id, completed_on)
);

-- After running this script, your database will be reset.
-- You will need to register a new user in the application.
