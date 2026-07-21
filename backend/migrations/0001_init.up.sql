-- Initial schema: identity, per-user settings, soundscape prefs, completed-session log.
-- All rows are user-scoped; every FK cascades on user delete.

create table users (
    id            bigint generated always as identity primary key,
    email         text        not null unique, -- app stores lowercased
    password_hash text        not null,        -- bcrypt
    created_at    timestamptz not null default now()
);

-- httpOnly cookie sessions; token is a random 256-bit string.
create table sessions_auth (
    token      text        primary key,
    user_id    bigint      not null references users(id) on delete cascade,
    expires_at timestamptz not null,
    created_at timestamptz not null default now()
);
create index sessions_auth_user_id_idx on sessions_auth (user_id);

-- One row per user; defaults match the prototype (25 / 5 / 15 / 4).
create table user_settings (
    user_id       bigint  primary key references users(id) on delete cascade,
    focus_min     int     not null default 25  check (focus_min     > 0),
    short_min     int     not null default 5   check (short_min     > 0),
    long_min      int     not null default 15  check (long_min      > 0),
    long_interval int     not null default 4   check (long_interval > 0),
    auto_breaks   boolean not null default false,
    auto_focus    boolean not null default false,
    chime         boolean not null default true,
    theme         text    not null default 'dark',
    master_mute   boolean not null default false
);

-- Per-sound state; master mute lives on user_settings.
create table soundscape_prefs (
    user_id   bigint  not null references users(id) on delete cascade,
    sound_key text    not null,
    enabled   boolean not null default false,
    volume    real    not null default 0.5 check (volume between 0 and 1),
    primary key (user_id, sound_key)
);

-- Completed focus sessions; feeds the SQL insights (T-06).
create table focus_sessions (
    id           bigint      generated always as identity primary key,
    user_id      bigint      not null references users(id) on delete cascade,
    started_at   timestamptz not null,
    duration_min int         not null,
    type         text        not null check (type in ('focus', 'short', 'long')),
    label        text
);
create index focus_sessions_user_started_idx on focus_sessions (user_id, started_at);
