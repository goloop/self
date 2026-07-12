-- Recipe 004: a notes table. tags is a text[] to show that pgc maps a Postgres
-- array to a Go []string.
CREATE TABLE notes (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title      text NOT NULL,
    body       text NOT NULL DEFAULT '',
    tags       text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);
