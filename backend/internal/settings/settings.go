// Package settings serves per-user timer settings and soundscape preferences.
// All routes are user-scoped via auth.UserID and mounted behind auth.RequireAuth.
package settings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"focus/backend/internal/auth"
	"focus/backend/internal/httpx"
)

// Service holds the pool shared by the settings handlers.
type Service struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Register attaches the settings routes onto an (auth-protected) router.
func (s *Service) Register(r chi.Router) {
	r.Get("/settings", s.getSettings)
	r.Put("/settings", s.putSettings)
	r.Get("/soundscapes", s.getSoundscapes)
	r.Put("/soundscapes", s.putSoundscapes)
}

const settingsCols = `focus_min, short_min, long_min, long_interval,
	auto_breaks, auto_focus, chime, theme, master_mute`

func defaultSettings() Settings {
	return Settings{FocusMin: 25, ShortMin: 5, LongMin: 15, LongInterval: 4, Chime: true, Theme: "dark"}
}

// --- settings handlers ---

func (s *Service) getSettings(w http.ResponseWriter, r *http.Request) {
	set, err := s.loadSettings(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	httpx.JSON(w, http.StatusOK, set)
}

func (s *Service) putSettings(w http.ResponseWriter, r *http.Request) {
	var u settingsUpdate
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := validateSettings(u); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	var set Settings
	err := s.pool.QueryRow(r.Context(),
		`update user_settings set
			focus_min     = coalesce($2, focus_min),
			short_min     = coalesce($3, short_min),
			long_min      = coalesce($4, long_min),
			long_interval = coalesce($5, long_interval),
			auto_breaks   = coalesce($6, auto_breaks),
			auto_focus    = coalesce($7, auto_focus),
			chime         = coalesce($8, chime),
			theme         = coalesce($9, theme),
			master_mute   = coalesce($10, master_mute)
		 where user_id = $1
		 returning `+settingsCols,
		auth.UserID(r.Context()),
		u.FocusMin, u.ShortMin, u.LongMin, u.LongInterval,
		u.AutoBreaks, u.AutoFocus, u.Chime, u.Theme, u.MasterMute,
	).Scan(&set.FocusMin, &set.ShortMin, &set.LongMin, &set.LongInterval,
		&set.AutoBreaks, &set.AutoFocus, &set.Chime, &set.Theme, &set.MasterMute)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not update settings")
		return
	}
	httpx.JSON(w, http.StatusOK, set)
}

func (s *Service) loadSettings(ctx context.Context, userID int64) (Settings, error) {
	var set Settings
	err := s.pool.QueryRow(ctx,
		`select `+settingsCols+` from user_settings where user_id = $1`, userID,
	).Scan(&set.FocusMin, &set.ShortMin, &set.LongMin, &set.LongInterval,
		&set.AutoBreaks, &set.AutoFocus, &set.Chime, &set.Theme, &set.MasterMute)
	if errors.Is(err, pgx.ErrNoRows) {
		// Defensive: signup seeds this row, but self-heal if it's missing.
		if _, err = s.pool.Exec(ctx, `insert into user_settings (user_id) values ($1)`, userID); err != nil {
			return Settings{}, err
		}
		return defaultSettings(), nil
	}
	return set, err
}

func validateSettings(u settingsUpdate) string {
	for _, d := range []struct {
		name string
		val  *int
	}{{"focusMin", u.FocusMin}, {"shortMin", u.ShortMin}, {"longMin", u.LongMin}} {
		if d.val != nil && (*d.val < 1 || *d.val > 90) {
			return d.name + " must be between 1 and 90"
		}
	}
	if u.LongInterval != nil && (*u.LongInterval < 1 || *u.LongInterval > 12) {
		return "longInterval must be between 1 and 12"
	}
	if u.Theme != nil && *u.Theme != "dark" && *u.Theme != "light" {
		return "theme must be 'dark' or 'light'"
	}
	return ""
}

// --- soundscape handlers ---

func (s *Service) getSoundscapes(w http.ResponseWriter, r *http.Request) {
	prefs, err := s.loadSoundscapes(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	httpx.JSON(w, http.StatusOK, prefs)
}

func (s *Service) putSoundscapes(w http.ResponseWriter, r *http.Request) {
	var updates []soundscapeUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	for _, u := range updates {
		if !knownSound(u.Key) {
			httpx.Error(w, http.StatusBadRequest, "unknown sound: "+u.Key)
			return
		}
		if u.Volume != nil && (*u.Volume < 0 || *u.Volume > 1) {
			httpx.Error(w, http.StatusBadRequest, "volume must be between 0 and 1")
			return
		}
	}

	userID := auth.UserID(r.Context())
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck // no-op after commit
	for _, u := range updates {
		if _, err = tx.Exec(r.Context(),
			`insert into soundscape_prefs (user_id, sound_key, enabled, volume)
			 values ($1, $2, coalesce($3, false), coalesce($4, 0.5))
			 on conflict (user_id, sound_key) do update
			 set enabled = coalesce($3, soundscape_prefs.enabled),
			     volume  = coalesce($4, soundscape_prefs.volume)`,
			userID, u.Key, u.Enabled, u.Volume,
		); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "could not update soundscapes")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not update soundscapes")
		return
	}

	prefs, err := s.loadSoundscapes(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	httpx.JSON(w, http.StatusOK, prefs)
}

// loadSoundscapes returns all five canonical sounds, merging stored rows over
// the defaults (enabled:false, volume:0.5) for any the user hasn't set.
func (s *Service) loadSoundscapes(ctx context.Context, userID int64) ([]SoundscapePref, error) {
	rows, err := s.pool.Query(ctx,
		`select sound_key, enabled, volume from soundscape_prefs where user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stored := make(map[string]SoundscapePref)
	for rows.Next() {
		var p SoundscapePref
		if err := rows.Scan(&p.Key, &p.Enabled, &p.Volume); err != nil {
			return nil, err
		}
		stored[p.Key] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	prefs := make([]SoundscapePref, 0, len(soundKeys))
	for _, key := range soundKeys {
		if p, ok := stored[key]; ok {
			prefs = append(prefs, p)
		} else {
			prefs = append(prefs, SoundscapePref{Key: key, Enabled: false, Volume: 0.5})
		}
	}
	return prefs, nil
}
