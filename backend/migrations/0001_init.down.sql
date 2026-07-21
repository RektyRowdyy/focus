-- Reverse 0001_init: drop in reverse-dependency order (cascade clears indexes/constraints).
drop table if exists focus_sessions;
drop table if exists soundscape_prefs;
drop table if exists user_settings;
drop table if exists sessions_auth;
drop table if exists users;
